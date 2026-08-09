package logger

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"Post_Analyzer_Webserver/config"
)

func TestInit_AllLevelsAndFormats(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error", "unrecognized-defaults-to-info"} {
		for _, format := range []string{"json", "text"} {
			cfg := &config.LoggingConfig{Level: level, Format: format, Output: "stdout", TimeFormat: "2006-01-02T15:04:05Z07:00"}
			if err := Init(cfg); err != nil {
				t.Fatalf("Init(level=%s, format=%s) failed: %v", level, format, err)
			}
			if Get() == nil {
				t.Errorf("expected a non-nil logger after Init(level=%s, format=%s)", level, format)
			}
		}
	}
}

func TestInit_EmptyOutputDefaultsToStdout(t *testing.T) {
	cfg := &config.LoggingConfig{Level: "info", Format: "json", Output: "", TimeFormat: time.RFC3339}
	if err := Init(cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
}

func TestInit_WritesToFileOutput(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "app.log")
	cfg := &config.LoggingConfig{Level: "info", Format: "json", Output: logPath, TimeFormat: time.RFC3339}
	if err := Init(cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	Info("test message")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("expected the log file to exist: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected the log file to contain the logged message")
	}

	// Restore stdout logging so subsequent tests in this package aren't
	// silently writing into a temp file that's about to be removed.
	_ = Init(&config.LoggingConfig{Level: "info", Format: "json", Output: "stdout", TimeFormat: time.RFC3339})
}

func TestInit_InvalidFileOutputPathFails(t *testing.T) {
	cfg := &config.LoggingConfig{Level: "info", Format: "json", Output: "/nonexistent-dir-xyz/app.log", TimeFormat: time.RFC3339}
	if err := Init(cfg); err == nil {
		t.Error("expected an error when the log file's directory doesn't exist")
	}
}

func TestGet_FallsBackToSlogDefaultWhenUninitialized(t *testing.T) {
	defaultLogger = nil
	if Get() == nil {
		t.Error("expected Get() to fall back to slog.Default() when Init was never called")
	}
}

func TestWithRequestID(t *testing.T) {
	_ = Init(&config.LoggingConfig{Level: "info", Format: "json", Output: "stdout", TimeFormat: time.RFC3339})

	ctx := context.WithValue(context.Background(), RequestIDKey, "req-123")
	l := WithRequestID(ctx)
	if l == nil {
		t.Fatal("expected a non-nil logger")
	}

	// No request ID in context: should still return a usable logger.
	if WithRequestID(context.Background()) == nil {
		t.Error("expected a non-nil logger even with no request ID in context")
	}
}

func TestWithContext(t *testing.T) {
	_ = Init(&config.LoggingConfig{Level: "info", Format: "json", Output: "stdout", TimeFormat: time.RFC3339})

	ctx := context.WithValue(context.Background(), RequestIDKey, "req-123")
	ctx = context.WithValue(ctx, UserIDKey, 42)
	if WithContext(ctx) == nil {
		t.Fatal("expected a non-nil logger")
	}

	if WithContext(context.Background()) == nil {
		t.Error("expected a non-nil logger even with no context values set")
	}
}

func TestLogFunctions_DoNotPanic(t *testing.T) {
	_ = Init(&config.LoggingConfig{Level: "debug", Format: "json", Output: "stdout", TimeFormat: time.RFC3339})
	ctx := context.Background()

	Debug("debug", "k", "v")
	Info("info")
	Warn("warn")
	Error("error")
	DebugContext(ctx, "debug ctx")
	InfoContext(ctx, "info ctx")
	WarnContext(ctx, "warn ctx")
	ErrorContext(ctx, "error ctx")
}
