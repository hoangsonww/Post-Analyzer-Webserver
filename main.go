package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"Post_Analyzer_Webserver/config"
	"Post_Analyzer_Webserver/internal/handlers"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/metrics"
	"Post_Analyzer_Webserver/internal/middleware"
	"Post_Analyzer_Webserver/internal/storage"
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
		"environment", cfg.Server.Environment,
		"port", cfg.Server.Port,
		"database_type", cfg.Database.Type,
	)

	// Initialize storage
	var store storage.Storage
	if cfg.Database.Type == "postgres" {
		store, err = storage.NewPostgresStorage(&cfg.Database)
		if err != nil {
			logger.Error("failed to initialize PostgreSQL storage", "error", err)
			os.Exit(1)
		}
		logger.Info("using PostgreSQL storage")
	} else {
		store, err = storage.NewFileStorage(cfg.Database.FilePath)
		if err != nil {
			logger.Error("failed to initialize file storage", "error", err)
			os.Exit(1)
		}
		logger.Info("using file storage", "path", cfg.Database.FilePath)
	}
	defer store.Close()

	// Initialize handlers
	h, err := handlers.New(store, cfg)
	if err != nil {
		logger.Error("failed to initialize handlers", "error", err)
		os.Exit(1)
	}
	defer h.Close()

	// Setup HTTP router
	mux := http.NewServeMux()

	// Health and monitoring endpoints
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/readiness", h.Readiness)
	mux.Handle("/metrics", metrics.Handler())

	// Application endpoints
	mux.HandleFunc("/", h.Home)
	mux.HandleFunc("/fetch", h.FetchPosts)
	mux.HandleFunc("/analyze", h.AnalyzePosts)
	mux.HandleFunc("/add", h.AddPost)

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
