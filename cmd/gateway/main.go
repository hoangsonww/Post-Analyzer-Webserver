package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"Post_Analyzer_Webserver/config"
	"Post_Analyzer_Webserver/internal/api"
	"Post_Analyzer_Webserver/internal/cache"
	"Post_Analyzer_Webserver/internal/handlers"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/metrics"
	"Post_Analyzer_Webserver/internal/middleware"
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

	// Dial the post-analysis RPC service (postsvc). The gateway holds no
	// database connection of its own — all storage lives behind postsvc.
	postClient, err := rpcclient.NewPostClient(cfg.RPC.PostServiceAddr, cfg.RPC.MuxTransport)
	if err != nil {
		logger.Error("failed to create postsvc RPC client", "error", err)
		os.Exit(1)
	}

	// Initialize cache
	_ = cache.NewCache(cfg) // Cache initialized for future use
	logger.Info("cache initialized", "type", "memory")

	// Initialize API handlers
	apiHandler := api.NewAPI(postClient)
	apiRouter := api.NewRouter(apiHandler)
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

	// API endpoints (v1)
	mux.Handle("/api/", apiRouter)
	mux.Handle("/api/v1/", apiRouter)

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
