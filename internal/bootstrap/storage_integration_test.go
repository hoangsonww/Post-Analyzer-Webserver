//go:build integration

// Integration test for InitStorage's postgres branch (migrations +
// storage.NewPostgresStorage) against a real local Postgres — the
// file-backend branch is covered by the plain unit tests in
// storage_test.go. Run with:
//
//	go test -tags=integration ./internal/bootstrap/... -v
package bootstrap

import (
	"context"
	"os"
	"testing"

	"Post_Analyzer_Webserver/config"
	"Post_Analyzer_Webserver/internal/storage"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func TestInitStorage_PostgresBackend(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Type:     "postgres",
			Host:     envOr("DB_HOST", "localhost"),
			Port:     envOr("DB_PORT", "5432"),
			User:     envOr("DB_USER", "localuser"),
			Password: envOr("DB_PASSWORD", "localpass"),
			DBName:   envOr("DB_NAME", "post_analyzer_dev"),
			SSLMode:  envOr("DB_SSL_MODE", "disable"),
			MaxConns: 5,
			MinConns: 1,
		},
	}

	store, err := InitStorage(cfg)
	if err != nil {
		t.Skipf("skipping: postgres not reachable (%v) — start it with `docker compose up -d postgres`", err)
	}
	defer store.Close()

	if _, ok := store.(*storage.PostgresStorage); !ok {
		t.Errorf("expected a *storage.PostgresStorage for Type=postgres, got %T", store)
	}

	// Migrations should have run and left a usable posts table —
	// confirm with a real Count() call, not just a type assertion.
	if _, err := store.Count(context.Background()); err != nil {
		t.Errorf("expected the migrated schema to support Count(), got %v", err)
	}
}
