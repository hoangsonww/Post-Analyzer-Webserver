package cli

import (
	"os"
	"strings"
	"testing"
)

func TestPaint_DisabledReturnsPlainText(t *testing.T) {
	orig := colorEnabled
	colorEnabled = false
	defer func() { colorEnabled = orig }()

	if got := red("hello"); got != "hello" {
		t.Errorf("expected plain text when color disabled, got %q", got)
	}
}

func TestPaint_EnabledWrapsInAnsiCodes(t *testing.T) {
	orig := colorEnabled
	colorEnabled = true
	defer func() { colorEnabled = orig }()

	got := red("hello")
	if !strings.HasPrefix(got, ansiRed) || !strings.HasSuffix(got, ansiReset) {
		t.Errorf("expected %q to be wrapped in ANSI red/reset codes, got %q", "hello", got)
	}
	if !strings.Contains(got, "hello") {
		t.Errorf("expected the original text to survive, got %q", got)
	}
}

func TestColorHelpers_AllRespectColorEnabled(t *testing.T) {
	orig := colorEnabled
	colorEnabled = false
	defer func() { colorEnabled = orig }()

	for name, fn := range map[string]func(string) string{
		"bold": bold, "dim": dim, "red": red, "green": green,
		"yellow": yellow, "blue": blue, "magenta": magenta, "cyan": cyan,
	} {
		if got := fn("x"); got != "x" {
			t.Errorf("%s: expected plain text with color disabled, got %q", name, got)
		}
	}
}

func TestOkAndFail_PrefixGlyphs(t *testing.T) {
	orig := colorEnabled
	colorEnabled = false
	defer func() { colorEnabled = orig }()

	if got := ok("done"); got != "✓ done" {
		t.Errorf("expected '✓ done', got %q", got)
	}
	if got := fail("nope"); got != "✗ nope" {
		t.Errorf("expected '✗ nope', got %q", got)
	}
}

func TestIsTerminal_FalseForNonTTYFile(t *testing.T) {
	// A regular temp file is never a character device.
	f, err := os.CreateTemp(t.TempDir(), "not-a-tty")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = f.Close() }()

	if isTerminal(f) {
		t.Error("expected a regular file to not be reported as a terminal")
	}
}
