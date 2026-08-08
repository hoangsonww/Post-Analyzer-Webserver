// Command authsvc is the Kitex RPC microservice owning authentication
// (JWT issuance) and ABAC authorization decisions for the whole system.
// It is the single policy decision point every other service defers to.
package main

import (
	"fmt"
	"os"

	"Post_Analyzer_Webserver/config"
	"Post_Analyzer_Webserver/internal/abac"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/metrics"
	auth "Post_Analyzer_Webserver/kitex_gen/auth/authservice"

	"github.com/cloudwego/kitex/pkg/utils"
	"github.com/cloudwego/kitex/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	if err := logger.Init(&cfg.Logging); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9012"
	}
	metrics.Serve("authsvc", metricsPort)

	users := abac.NewUserStore()
	users.AddUser(cfg.Auth.AdminUsername, cfg.Auth.AdminPassword, abac.Subject{UserID: 1, Role: "admin"})

	handler := NewAuthServiceImpl(users, cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)

	addr := utils.NewNetAddr("tcp", cfg.RPC.AuthServiceAddr)
	opts := []server.Option{server.WithServiceAddr(addr)}
	if cfg.RPC.MuxTransport {
		opts = append(opts, server.WithMuxTransport())
	}

	logger.Info("starting authsvc RPC server",
		"addr", cfg.RPC.AuthServiceAddr,
		"transport", "TTHeader",
		"mux", cfg.RPC.MuxTransport,
		"demo_accounts", "editor/editor123, viewer/viewer123",
	)

	svr := auth.NewServer(handler, opts...)
	if err := svr.Run(); err != nil {
		logger.Error("authsvc server stopped with error", "error", err)
		os.Exit(1)
	}
}
