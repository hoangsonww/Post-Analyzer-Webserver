//go:build integration

// Integration tests against a real PostgreSQL instance. Skipped by
// default (see the build tag) — run with:
//
//	go test -tags=integration ./internal/storage/... -v
//
// Connection is configured via the same DB_* env vars as the service
// binaries (config.Load), defaulting to this repo's documented local
// setup (localhost:5432, localuser/localpass) so `go test -tags=integration
// ./...` works out of the box against "any user having a postgresql
// running" locally, per this repo's stated goal — not just in CI.
package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"Post_Analyzer_Webserver/config"
)

func testDBConfig() *config.DatabaseConfig {
	return &config.DatabaseConfig{
		Type:     "postgres",
		Host:     envOr("DB_HOST", "localhost"),
		Port:     envOr("DB_PORT", "5432"),
		User:     envOr("DB_USER", "localuser"),
		Password: envOr("DB_PASSWORD", "localpass"),
		DBName:   envOr("DB_NAME", "post_analyzer_dev"),
		SSLMode:  envOr("DB_SSL_MODE", "disable"),
		MaxConns: 5,
		MinConns: 1,
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func newIntegrationStorage(t *testing.T) *PostgresStorage {
	t.Helper()
	store, err := NewPostgresStorage(testDBConfig())
	if err != nil {
		t.Skipf("skipping: postgres not reachable (%v) — start it with `docker compose up -d postgres` or point DB_* env vars at a running instance", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestPostgresStorage_CreateGetDelete(t *testing.T) {
	store := newIntegrationStorage(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	post := &Post{UserId: 1, Title: "integration test post", Body: "created by go test -tags=integration"}
	if err := store.Create(ctx, post); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if post.Id == 0 {
		t.Fatal("expected a generated ID after Create")
	}
	t.Cleanup(func() { _ = store.Delete(context.Background(), post.Id) })

	got, err := store.GetByID(ctx, post.Id)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Title != post.Title || got.Body != post.Body {
		t.Errorf("unexpected post: %+v", got)
	}

	if err := store.Delete(ctx, post.Id); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := store.GetByID(ctx, post.Id); err == nil {
		t.Error("expected the deleted post to be gone")
	}
}

func TestPostgresStorage_UpdateAndCount(t *testing.T) {
	store := newIntegrationStorage(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	before, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	post := &Post{UserId: 2, Title: "before update", Body: "body"}
	if err := store.Create(ctx, post); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	t.Cleanup(func() { _ = store.Delete(context.Background(), post.Id) })

	after, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if after != before+1 {
		t.Errorf("expected Count to increase by 1, got before=%d after=%d", before, after)
	}

	post.Title = "after update"
	if err := store.Update(ctx, post); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	got, err := store.GetByID(ctx, post.Id)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Title != "after update" {
		t.Errorf("expected updated title, got %q", got.Title)
	}
}

func TestPostgresStorage_BatchCreateAndGetAll(t *testing.T) {
	store := newIntegrationStorage(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	batch := []Post{
		{UserId: 3, Title: "batch 1", Body: "b1"},
		{UserId: 3, Title: "batch 2", Body: "b2"},
	}
	if err := store.BatchCreate(ctx, batch); err != nil {
		t.Fatalf("BatchCreate failed: %v", err)
	}

	all, err := store.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	found := 0
	for _, p := range all {
		if p.Title == "batch 1" || p.Title == "batch 2" {
			found++
			t.Cleanup(func(id int) func() { return func() { _ = store.Delete(context.Background(), id) } }(p.Id))
		}
	}
	if found != 2 {
		t.Errorf("expected to find both batch-created posts in GetAll, found %d", found)
	}
}
