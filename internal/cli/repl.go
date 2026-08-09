package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newReplCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repl",
		Short: "Interactive shell: run posts/login/sentiment commands without relaunching the binary each time",
		RunE: func(cmd *cobra.Command, args []string) error {
			runRepl()
			return nil
		},
	}
}

// runRepl reuses the same cobra command tree that powers one-shot CLI
// invocations — each line typed is tokenized and dispatched through
// Execute() on a fresh copy of that tree, so "posts list", "login admin
// admin123", etc. behave identically here and on the command line. Errors
// are printed and the loop continues; only `exit`/`quit`/EOF ends it.
func runRepl() {
	printBanner()
	fmt.Println(dim("connected to ") + cyan(flagServerURL))
	fmt.Println(dim(`Type "help" for commands, "exit" to quit.`))

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(bold(cyan("post-analyzer> ")))
		if !scanner.Scan() {
			fmt.Println()
			return
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		switch line {
		case "exit", "quit":
			return
		case "help", "?":
			printReplHelp()
			continue
		case "clear":
			fmt.Print("\033[H\033[2J")
			continue
		}

		args, err := tokenize(line)
		if err != nil {
			fmt.Println(fail(err.Error()))
			continue
		}

		replRoot := buildReplCommandTree()
		replRoot.SetArgs(args)
		if err := replRoot.Execute(); err != nil {
			fmt.Println(fail(err.Error()))
		}
	}
}

// buildReplCommandTree returns a fresh root each iteration. Cobra command
// trees carry per-run state (parsed flags, etc.), so reusing one instance
// across REPL lines risks stale flag values leaking between commands —
// rebuilding is cheap and avoids that entirely.
func buildReplCommandTree() *cobra.Command {
	// SilenceErrors: true — same reasoning as root.go's Execute(): the
	// REPL loop's own fail()-colored print below is the single source
	// of error output, not cobra's built-in plain "Error: ..." print.
	root := &cobra.Command{Use: "post-analyzer", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(newLoginCmd(), newPostsCmd(), newSentimentCmd())
	return root
}

// replHelpLine pads cmdText to a fixed column width *before* colorizing
// it — padding a string that already contains ANSI escape codes would
// count those invisible bytes toward the width and break alignment.
func replHelpLine(cmdText, desc string) string {
	const col = 32
	pad := col - len(cmdText)
	if pad < 1 {
		pad = 1
	}
	styled := "  " + bold(yellow(cmdText))
	if desc == "" {
		return styled
	}
	return styled + strings.Repeat(" ", pad) + dim(desc)
}

func printReplHelp() {
	fmt.Println(bold("Commands:"))
	for _, l := range [][2]string{
		{`login <username> <password>`, "Log in (admin/admin123, editor/editor123, viewer/viewer123 by default)"},
		{`posts list`, ""},
		{`posts get <id>`, ""},
		{`posts create --title "..." --body "..." [--user-id N]`, ""},
		{`posts update <id> [--title "..."] [--body "..."]`, ""},
		{`posts delete <id> [--mfa]`, ""},
		{`posts analyze`, ""},
		{`posts reanalyze`, ""},
		{`posts export [--format json|csv] [--output file]`, ""},
		{`sentiment <text...>`, ""},
		{`clear`, "Clear the screen"},
		{`exit | quit`, "Leave the REPL"},
	} {
		fmt.Println(replHelpLine(l[0], l[1]))
	}
}

// tokenize is a minimal shell-style splitter supporting "double" and
// 'single' quoted arguments (so --title "hello world" works) without
// pulling in a shlex dependency for one small feature.
func tokenize(line string) ([]string, error) {
	var tokens []string
	var cur strings.Builder
	var quote rune
	inToken := false

	flush := func() {
		if inToken {
			tokens = append(tokens, cur.String())
			cur.Reset()
			inToken = false
		}
	}

	for _, r := range line {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote = r
			inToken = true
		case r == ' ' || r == '\t':
			flush()
		default:
			inToken = true
			cur.WriteRune(r)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	flush()
	return tokens, nil
}
