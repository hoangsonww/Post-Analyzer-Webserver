// Command gateway is the Post Analyzer's single user-facing binary: with
// no arguments it runs the Hertz HTTP server (unchanged default, so
// existing `docker run`/compose/k8s invocations with no args keep
// working); with a subcommand it's a CLI or REPL client of that same
// REST API. See internal/cli.
package main

import "Post_Analyzer_Webserver/internal/cli"

func main() {
	cli.Execute()
}
