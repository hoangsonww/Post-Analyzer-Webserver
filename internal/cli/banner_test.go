package cli

import (
	"strings"
	"testing"
)

func TestPrintBanner_DoesNotPanic(t *testing.T) {
	printBanner()
}

// TestReplHelpLine_PadsBeforeColorizing guards against the class of bug
// where wrapping a string in ANSI codes and *then* applying %-Ns padding
// counts the invisible escape bytes toward the width, breaking column
// alignment. With color disabled the padding is directly observable.
func TestReplHelpLine_PadsBeforeColorizing(t *testing.T) {
	orig := colorEnabled
	colorEnabled = false
	defer func() { colorEnabled = orig }()

	got := replHelpLine("short", "description")
	if !strings.Contains(got, "description") {
		t.Fatalf("expected description to appear, got %q", got)
	}
	// "  " + "short" + padding + "description" — the gap between
	// "short" and "description" should be many spaces, not zero/one.
	idx := strings.Index(got, "short")
	descIdx := strings.Index(got, "description")
	gap := got[idx+len("short") : descIdx]
	if len(strings.TrimSpace(gap)) != 0 {
		t.Errorf("expected only whitespace between command and description, got %q", gap)
	}
	if len(gap) < 5 {
		t.Errorf("expected meaningful column padding, got gap of %d chars: %q", len(gap), gap)
	}
}

func TestReplHelpLine_NoDescriptionOmitsTrailingSpace(t *testing.T) {
	orig := colorEnabled
	colorEnabled = false
	defer func() { colorEnabled = orig }()

	got := replHelpLine("clear", "")
	if strings.Contains(got, "  \n") || strings.HasSuffix(got, " ") {
		t.Errorf("expected no dangling padding with an empty description, got %q", got)
	}
}
