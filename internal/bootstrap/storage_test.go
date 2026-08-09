package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	"Post_Analyzer_Webserver/config"
	"Post_Analyzer_Webserver/internal/storage"
)

func TestInitStorage_FileBackend(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "posts.json")

	cfg := &config.Config{Database: config.DatabaseConfig{Type: "file", FilePath: filePath}}
	store, err := InitStorage(cfg)
	if err != nil {
		t.Fatalf("InitStorage failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	if _, ok := store.(*storage.FileStorage); !ok {
		t.Errorf("expected a *storage.FileStorage for Type=file, got %T", store)
	}
	if _, err := os.Stat(filePath); err != nil {
		t.Errorf("expected InitStorage to create the backing file, got %v", err)
	}
}

func TestInitStorage_DefaultsToFileBackendForUnknownType(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "posts.json")

	// InitStorage's branch is `if Type == "postgres" {...} else {file}` —
	// any non-"postgres" value (including an empty/unset one) takes the
	// file-storage path, which is the safe local-dev default.
	cfg := &config.Config{Database: config.DatabaseConfig{Type: "", FilePath: filePath}}
	store, err := InitStorage(cfg)
	if err != nil {
		t.Fatalf("InitStorage failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	if _, ok := store.(*storage.FileStorage); !ok {
		t.Errorf("expected a *storage.FileStorage for an unrecognized Type, got %T", store)
	}
}

func TestInitStorage_FileBackend_InvalidPathFails(t *testing.T) {
	cfg := &config.Config{Database: config.DatabaseConfig{Type: "file", FilePath: "/nonexistent-dir-xyz/posts.json"}}
	if _, err := InitStorage(cfg); err == nil {
		t.Error("expected an error when the file storage path's directory doesn't exist")
	}
}
