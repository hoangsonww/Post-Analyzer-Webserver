package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"Post_Analyzer_Webserver/config"
	"Post_Analyzer_Webserver/internal/api"
	"Post_Analyzer_Webserver/internal/cache"
	"Post_Analyzer_Webserver/internal/handlers"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/metrics"
	"Post_Analyzer_Webserver/internal/middleware"
	"Post_Analyzer_Webserver/internal/migrations"
	"Post_Analyzer_Webserver/internal/service"
	"Post_Analyzer_Webserver/internal/storage"

	_ "github.com/lib/pq"
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

	logger.Info("starting Post Analyzer Webserver",
		"version", version,
		"environment", cfg.Server.Environment,
		"port", cfg.Server.Port,
		"database_type", cfg.Database.Type,
	)

	// Initialize storage
	var store storage.Storage
	var db *sql.DB

	if cfg.Database.Type == "postgres" {
		pgStore, err := storage.NewPostgresStorage(&cfg.Database)
		if err != nil {
			logger.Error("failed to initialize PostgreSQL storage", "error", err)
			os.Exit(1)
		}
		store = pgStore

		// Get underlying DB for migrations
		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
			cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode,
		)
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			logger.Error("failed to open database for migrations", "error", err)
			os.Exit(1)
		}
		defer db.Close()

		// Run migrations
		logger.Info("running database migrations...")
		migrator := migrations.NewMigrator(db)
		if err := migrator.Migrate(context.Background()); err != nil {
			logger.Error("migration failed", "error", err)
			os.Exit(1)
		}

		logger.Info("using PostgreSQL storage")
	} else {
		fileStore, err := storage.NewFileStorage(cfg.Database.FilePath)
		if err != nil {
			logger.Error("failed to initialize file storage", "error", err)
			os.Exit(1)
		}
		store = fileStore
		logger.Info("using file storage", "path", cfg.Database.FilePath)
	}
	defer store.Close()

	// Initialize cache
	_ = cache.NewCache(cfg) // Cache initialized for future use
	logger.Info("cache initialized", "type", "memory")

	// Initialize service layer
	postService := service.NewPostService(store)
	logger.Info("service layer initialized")

	// Initialize API handlers
	apiHandler := api.NewAPI(postService)
	apiRouter := api.NewRouter(apiHandler)
	logger.Info("API handlers initialized")

	// Initialize web handlers
	webHandlers, err := handlers.New(store, cfg)
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

	// Create HTTP server with production settings
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("server listening",
			"address", server.Addr,
			"read_timeout", cfg.Server.ReadTimeout,
			"write_timeout", cfg.Server.WriteTimeout,
		)
		logger.Info("endpoints available",
			"web", fmt.Sprintf("http://%s:%s/", cfg.Server.Host, cfg.Server.Port),
			"api", fmt.Sprintf("http://%s:%s/api/v1/posts", cfg.Server.Host, cfg.Server.Port),
			"health", fmt.Sprintf("http://%s:%s/health", cfg.Server.Host, cfg.Server.Port),
			"metrics", fmt.Sprintf("http://%s:%s/metrics", cfg.Server.Host, cfg.Server.Port),
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server gracefully...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped gracefully")
}
