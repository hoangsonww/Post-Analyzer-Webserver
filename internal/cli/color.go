package cli

import "os"

// ANSI SGR codes. Kept minimal and dependency-free — this is a small,
// deliberately tasteful set of colors for CLI/REPL output, not a full
// terminal-styling library.
const (
	ansiReset   = "\033[0m"
	ansiBold    = "\033[1m"
	ansiDim     = "\033[2m"
	ansiRed     = "\033[31m"
	ansiGreen   = "\033[32m"
	ansiYellow  = "\033[33m"
	ansiBlue    = "\033[34m"
	ansiMagenta = "\033[35m"
	ansiCyan    = "\033[36m"
)

// colorEnabled follows the standard conventions: off when NO_COLOR is
// set (see https://no-color.org), off when stdout isn't a terminal
// (piped to a file, redirected in a script, captured by a test), on
// otherwise. The --no-color flag (see root.go) can also force it off.
var colorEnabled = os.Getenv("NO_COLOR") == "" && isTerminal(os.Stdout)

func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func paint(code, s string) string {
	if !colorEnabled {
		return s
	}
	return code + s + ansiReset
}

func bold(s string) string    { return paint(ansiBold, s) }
func dim(s string) string     { return paint(ansiDim, s) }
func red(s string) string     { return paint(ansiRed, s) }
func green(s string) string   { return paint(ansiGreen, s) }
func yellow(s string) string  { return paint(ansiYellow, s) }
func blue(s string) string    { return paint(ansiBlue, s) }
func magenta(s string) string { return paint(ansiMagenta, s) }
func cyan(s string) string    { return paint(ansiCyan, s) }

// ok/fail prefix a message with a colored glyph — the one visual cue
// used consistently everywhere the CLI reports an outcome.
func ok(s string) string   { return green("✓") + " " + s }
func fail(s string) string { return red("✗") + " " + s }
