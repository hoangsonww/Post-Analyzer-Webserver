package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagServerURL string
	flagToken     string
	flagNoColor   bool
)

// Execute is the gateway binary's single entry point. With no subcommand
// it behaves exactly like the pre-CLI gateway binary always did (runs the
// HTTP server) — Dockerfile/docker-compose invoke `./service` with no
// arguments, so that default must never change. Every other invocation
// (`post-analyzer posts list`, `post-analyzer repl`, ...) is a client of
// a *running* gateway's REST API, reachable via --server.
func Execute() {
	root := &cobra.Command{
		Use:     "post-analyzer",
		Short:   "Post Analyzer — server, CLI, and REPL in one binary",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			runServe()
			return nil
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if flagNoColor {
				colorEnabled = false
			}
		},
		SilenceUsage: true,
		// Cobra's default error handling prints its own plain "Error:
		// ..." to stderr before returning the error up the call stack —
		// which would print *in addition to* our own colored fail()
		// message below, once for every failed command. We're the only
		// error-printer here.
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&flagServerURL, "server", envOr("POST_ANALYZER_SERVER", "http://localhost:8080"), "gateway base URL (CLI/REPL modes)")
	root.PersistentFlags().StringVar(&flagToken, "token", "", "JWT token (defaults to the one saved by `login`, then $POST_ANALYZER_TOKEN)")
	root.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "disable colored output (also respects $NO_COLOR)")

	root.AddCommand(newServeCmd())
	root.AddCommand(newLoginCmd())
	root.AddCommand(newPostsCmd())
	root.AddCommand(newSentimentCmd())
	root.AddCommand(newReplCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, fail(err.Error()))
		os.Exit(1)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// clientFromFlags builds a Client using --server/--token, falling back to
// $POST_ANALYZER_TOKEN then the token file written by `login`.
func clientFromFlags() *Client {
	token := flagToken
	if token == "" {
		token = envOr("POST_ANALYZER_TOKEN", "")
	}
	if token == "" {
		token = LoadToken()
	}
	return NewClient(flagServerURL, token)
}
