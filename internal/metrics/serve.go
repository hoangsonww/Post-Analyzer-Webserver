package metrics

import (
	"net/http"

	"Post_Analyzer_Webserver/internal/logger"
)

// Serve starts a dedicated /metrics HTTP server on port in the
// background. Every service binary in this repo — including the pure-RPC
// ones (postsvc, authsvc) that have no other HTTP surface — calls this so
// Prometheus has one consistent way to scrape every process.
func Serve(serviceName, port string) {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", Handler())
		logger.Info(serviceName+" metrics listening", "port", port)
		if err := http.ListenAndServe(":"+port, mux); err != nil { //nolint:gosec // internal metrics endpoint, no external exposure needed
			logger.Error(serviceName+" metrics server failed", "error", err)
		}
	}()
}
