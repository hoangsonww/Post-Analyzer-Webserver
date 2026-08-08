// Package bootstrap holds startup wiring shared by every service binary
// (gateway, postsvc, authsvc) so storage/migration setup isn't duplicated
// across cmd/*/main.go.
package bootstrap

import (
	"context"
	"database/sql"
	"fmt"

	"Post_Analyzer_Webserver/config"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/migrations"
	"Post_Analyzer_Webserver/internal/storage"

	_ "github.com/lib/pq"
)

// InitStorage opens the configured storage backend (postgres or file),
// running migrations first when using postgres. The caller owns closing
// the returned Storage.
func InitStorage(cfg *config.Config) (storage.Storage, error) {
	if cfg.Database.Type == "postgres" {
		pgStore, err := storage.NewPostgresStorage(&cfg.Database)
		if err != nil {
			return nil, fmt.Errorf("init postgres storage: %w", err)
		}

		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
			cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode,
		)
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			return nil, fmt.Errorf("open database for migrations: %w", err)
		}
		defer func() { _ = db.Close() }()

		logger.Info("running database migrations...")
		migrator := migrations.NewMigrator(db)
		if err := migrator.Migrate(context.Background()); err != nil {
			return nil, fmt.Errorf("migration failed: %w", err)
		}

		logger.Info("using PostgreSQL storage")
		return pgStore, nil
	}

	fileStore, err := storage.NewFileStorage(cfg.Database.FilePath)
	if err != nil {
		return nil, fmt.Errorf("init file storage: %w", err)
	}
	logger.Info("using file storage", "path", cfg.Database.FilePath)
	return fileStore, nil
}
