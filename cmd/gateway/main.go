package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"Post_Analyzer_Webserver/config"
	"Post_Analyzer_Webserver/internal/api"
	"Post_Analyzer_Webserver/internal/handlers"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/messaging/rabbitmq"
	"Post_Analyzer_Webserver/internal/metrics"
	"Post_Analyzer_Webserver/internal/middleware"
	"Post_Analyzer_Webserver/internal/ml/triton"
	"Post_Analyzer_Webserver/internal/objectstore"
	"Post_Analyzer_Webserver/internal/rpcclient"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
)

var (
	version   = "2.0.0"
	buildTime = time.Now().Format(time.RFC3339)
	startTime = time.Now()
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init(&cfg.Logging); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("starting Post Analyzer Gateway",
		"version", version,
		"environment", cfg.Server.Environment,
		"port", cfg.Server.Port,
		"postsvc_addr", cfg.RPC.PostServiceAddr,
	)

	// Dial the post-analysis and auth RPC services. The gateway holds no
	// database connection of its own — all storage lives behind postsvc,
	// and all auth/ABAC decisions live behind authsvc.
	postClient, err := rpcclient.NewPostClient(cfg.RPC.PostServiceAddr, cfg.RPC.MuxTransport)
	if err != nil {
		logger.Error("failed to create postsvc RPC client", "error", err)
		os.Exit(1)
	}
	authClient, err := rpcclient.NewAuthClient(cfg.RPC.AuthServiceAddr, cfg.RPC.MuxTransport)
	if err != nil {
		logger.Error("failed to create authsvc RPC client", "error", err)
		os.Exit(1)
	}

	// Optional: RabbitMQ for the async reanalysis queue (POST
	// /api/v1/posts/reanalyze). nil when disabled — the endpoint then
	// responds 503 instead of failing gateway startup.
	var rmqClient *rabbitmq.Client
	if cfg.Messaging.RabbitMQEnabled {
		rmqClient, err = rabbitmq.Connect(cfg.Messaging.RabbitMQURL)
		if err != nil {
			logger.Error("failed to connect to rabbitmq", "error", err)
			os.Exit(1)
		}
		if err := rmqClient.DeclareQueue(rabbitmq.ReanalysisQueue); err != nil {
			logger.Error("failed to declare rabbitmq queue", "error", err)
			os.Exit(1)
		}
		defer rmqClient.Close()
		logger.Info("rabbitmq reanalysis queue enabled", "url", cfg.Messaging.RabbitMQURL, "queue", rabbitmq.ReanalysisQueue)
	}

	// Optional: MinIO object store for persisted post exports
	// (GET /api/v1/exports, /api/v1/exports/{key}). nil when disabled.
	var objectStore *objectstore.Store
	if cfg.ObjectStore.Enabled {
		objectStore, err = objectstore.New(context.Background(),
			cfg.ObjectStore.Endpoint, cfg.ObjectStore.AccessKey, cfg.ObjectStore.SecretKey, cfg.ObjectStore.UseSSL)
		if err != nil {
			logger.Error("failed to connect to minio", "error", err)
			os.Exit(1)
		}
		logger.Info("minio object store enabled", "endpoint", cfg.ObjectStore.Endpoint, "bucket", objectstore.ExportsBucket)
	}

	// Optional: Triton sentiment classification (POST /api/v1/ml/sentiment).
	// nil when disabled.
	var tritonClient *triton.Client
	if cfg.ML.Enabled {
		tritonClient = triton.NewClient(cfg.ML.TritonURL)
		logger.Info("triton sentiment classification enabled", "url", cfg.ML.TritonURL, "model", triton.ModelName)
	}

	// Initialize API handlers. The posts routes require a valid JWT +
	// ABAC allow decision from authsvc; /api/v1/auth/login (registered
	// separately below) is the one unauthenticated endpoint that issues
	// that JWT in the first place.
	apiHandler := api.NewAPI(postClient, authClient, rmqClient, objectStore, tritonClient)
	apiRouter := api.NewRouter(apiHandler)
	protectedAPI := middleware.ABAC(authClient, "post", middleware.ActionByMethod)(apiRouter)
	logger.Info("API handlers initialized")

	// Initialize web handlers
	webHandlers, err := handlers.New(postClient, cfg)
	if err != nil {
		logger.Error("failed to initialize web handlers", "error", err)
		os.Exit(1)
	}
	defer webHandlers.Close()

	// Setup HTTP router
	mux := http.NewServeMux()

	// Health and monitoring endpoints
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		uptime := time.Since(startTime)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","version":"%s","uptime":"%s","timestamp":"%s"}`,
			version, uptime, time.Now().Format(time.RFC3339))
	})
	mux.HandleFunc("/readiness", webHandlers.Readiness)
	mux.Handle("/metrics", metrics.Handler())

	// API endpoints (v1). Login is registered as an exact path so it wins
	// over the "/api/v1/" prefix match and stays outside the ABAC gate.
	mux.HandleFunc("/api/v1/auth/login", apiHandler.Login)
	mux.Handle("/api/", protectedAPI)
	mux.Handle("/api/v1/", protectedAPI)

	// Web interface endpoints
	mux.HandleFunc("/", webHandlers.Home)
	mux.HandleFunc("/fetch", webHandlers.FetchPosts)
	mux.HandleFunc("/analyze", webHandlers.AnalyzePosts)
	mux.HandleFunc("/add", webHandlers.AddPost)

	// Serve static assets
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	// Create rate limiter
	rateLimiter := middleware.NewRateLimiter(
		cfg.Security.RateLimitRequests,
		cfg.Security.RateLimitWindow,
	)

	// Apply middleware chain
	handler := middleware.Chain(
		middleware.RequestID,
		middleware.Logging,
		middleware.Recovery,
		middleware.SecurityHeaders,
		middleware.CORS(cfg.Security.AllowedOrigins),
		rateLimiter.Middleware,
		middleware.MaxBodySize(cfg.Security.MaxBodySize),
		middleware.Timeout(cfg.Server.WriteTimeout),
		middleware.Compression,
		metrics.Middleware,
	)(mux)

	// Serve everything through Hertz: it owns the listener, the Netpoll
	// transport, and TTHeader-free plain HTTP/1.1 on the edge. The whole
	// existing middleware-wrapped mux is mounted as Hertz's fallback route
	// via the framework's official net/http compatibility adaptor, so
	// none of the handler logic above needs to know it's running inside
	// Hertz rather than stdlib net/http.
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	h := server.New(
		server.WithHostPorts(addr),
		server.WithReadTimeout(cfg.Server.ReadTimeout),
		server.WithWriteTimeout(cfg.Server.WriteTimeout),
		server.WithMaxRequestBodySize(int(cfg.Security.MaxBodySize)),
		server.WithExitWaitTime(cfg.Server.ShutdownTimeout),
	)
	h.NoRoute(adaptor.HertzHandler(handler))

	logger.Info("server listening",
		"address", addr,
		"engine", "Hertz (Netpoll)",
		"read_timeout", cfg.Server.ReadTimeout,
		"write_timeout", cfg.Server.WriteTimeout,
	)
	logger.Info("endpoints available",
		"web", fmt.Sprintf("http://%s/", addr),
		"api", fmt.Sprintf("http://%s/api/v1/posts", addr),
		"health", fmt.Sprintf("http://%s/health", addr),
		"metrics", fmt.Sprintf("http://%s/metrics", addr),
	)

	// h.Spin() blocks, serving requests until SIGINT/SIGTERM, then drains
	// in-flight requests for up to WithExitWaitTime before returning.
	h.Spin()

	logger.Info("server stopped gracefully")
}
