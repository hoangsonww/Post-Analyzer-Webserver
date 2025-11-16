package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"Post_Analyzer_Webserver/internal/logger"
)

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	Up          func(*sql.DB) error
	Down        func(*sql.DB) error
}

// Migrator handles database migrations
type Migrator struct {
	db         *sql.DB
	migrations []Migration
}

// NewMigrator creates a new migrator
func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{
		db:         db,
		migrations: getMigrations(),
	}
}

// Migrate runs all pending migrations
func (m *Migrator) Migrate(ctx context.Context) error {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	currentVersion, err := m.getCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	logger.Info("starting migrations", "current_version", currentVersion)

	// Run pending migrations
	for _, migration := range m.migrations {
		if migration.Version <= currentVersion {
			continue
		}

		logger.Info("running migration", "version", migration.Version, "description", migration.Description)

		// Begin transaction
		tx, err := m.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Run migration
		if err := migration.Up(m.db); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d failed: %w", migration.Version, err)
		}

		// Update version
		if err := m.updateVersion(tx, migration.Version, migration.Description); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update version: %w", err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration: %w", err)
		}

		logger.Info("migration completed", "version", migration.Version)
	}

	logger.Info("all migrations completed")
	return nil
}

// createMigrationsTable creates the migrations tracking table
func (m *Migrator) createMigrationsTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		id SERIAL PRIMARY KEY,
		version INTEGER NOT NULL UNIQUE,
		description TEXT NOT NULL,
		applied_at TIMESTAMP NOT NULL DEFAULT NOW()
	)`

	_, err := m.db.Exec(query)
	return err
}

// getCurrentVersion gets the current migration version
func (m *Migrator) getCurrentVersion() (int, error) {
	var version int
	err := m.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

// updateVersion records a completed migration
func (m *Migrator) updateVersion(tx *sql.Tx, version int, description string) error {
	query := "INSERT INTO schema_migrations (version, description, applied_at) VALUES ($1, $2, $3)"
	_, err := tx.Exec(query, version, description, time.Now())
	return err
}

// getMigrations returns all migrations in order
func getMigrations() []Migration {
	return []Migration{
		{
			Version:     1,
			Description: "Create posts table",
			Up: func(db *sql.DB) error {
				query := `
				CREATE TABLE IF NOT EXISTS posts (
					id SERIAL PRIMARY KEY,
					user_id INTEGER NOT NULL,
					title VARCHAR(500) NOT NULL,
					body TEXT NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMP NOT NULL DEFAULT NOW()
				);

				CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(user_id);
				CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);
				`
				_, err := db.Exec(query)
				return err
			},
			Down: func(db *sql.DB) error {
				_, err := db.Exec("DROP TABLE IF EXISTS posts")
				return err
			},
		},
		{
			Version:     2,
			Description: "Create audit_logs table",
			Up: func(db *sql.DB) error {
				query := `
				CREATE TABLE IF NOT EXISTS audit_logs (
					id SERIAL PRIMARY KEY,
					user_id INTEGER NOT NULL,
					action VARCHAR(50) NOT NULL,
					resource VARCHAR(50) NOT NULL,
					resource_id INTEGER NOT NULL,
					changes TEXT,
					ip_address VARCHAR(45),
					user_agent TEXT,
					created_at TIMESTAMP NOT NULL DEFAULT NOW()
				);

				CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
				CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource, resource_id);
				CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
				`
				_, err := db.Exec(query)
				return err
			},
			Down: func(db *sql.DB) error {
				_, err := db.Exec("DROP TABLE IF EXISTS audit_logs")
				return err
			},
		},
		{
			Version:     3,
			Description: "Add full-text search indexes",
			Up: func(db *sql.DB) error {
				query := `
				CREATE INDEX IF NOT EXISTS idx_posts_title_trgm ON posts USING gin(title gin_trgm_ops);
				CREATE INDEX IF NOT EXISTS idx_posts_body_trgm ON posts USING gin(body gin_trgm_ops);
				`
				_, err := db.Exec(query)
				if err != nil {
					// If pg_trgm extension doesn't exist, skip this migration
					logger.Warn("failed to create full-text search indexes, pg_trgm extension may not be enabled")
					return nil
				}
				return err
			},
			Down: func(db *sql.DB) error {
				query := `
				DROP INDEX IF EXISTS idx_posts_title_trgm;
				DROP INDEX IF EXISTS idx_posts_body_trgm;
				`
				_, err := db.Exec(query)
				return err
			},
		},
	}
}
