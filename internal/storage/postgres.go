package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"Post_Analyzer_Webserver/config"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/metrics"

	_ "github.com/lib/pq"
)

// PostgresStorage implements Storage interface using PostgreSQL
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage creates a new PostgreSQL storage
func NewPostgresStorage(cfg *config.DatabaseConfig) (*PostgresStorage, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MinConns)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	ps := &PostgresStorage{db: db}

	// Initialize schema
	if err := ps.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return ps, nil
}

// initSchema creates the necessary database tables
func (ps *PostgresStorage) initSchema() error {
	schema := `
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

	_, err := ps.db.Exec(schema)
	return err
}

// GetAll retrieves all posts
func (ps *PostgresStorage) GetAll(ctx context.Context) ([]Post, error) {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("get_all", "success", time.Since(start))
	}()

	query := `SELECT id, user_id, title, body, created_at, updated_at FROM posts ORDER BY created_at DESC`

	rows, err := ps.db.QueryContext(ctx, query)
	if err != nil {
		metrics.RecordDBOperation("get_all", "error", time.Since(start))
		logger.ErrorContext(ctx, "failed to query posts", "error", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.Id, &post.UserId, &post.Title, &post.Body, &post.CreatedAt, &post.UpdatedAt); err != nil {
			metrics.RecordDBOperation("get_all", "error", time.Since(start))
			return nil, err
		}
		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		metrics.RecordDBOperation("get_all", "error", time.Since(start))
		return nil, err
	}

	metrics.RecordPostsTotal(len(posts))
	return posts, nil
}

// GetByID retrieves a post by ID
func (ps *PostgresStorage) GetByID(ctx context.Context, id int) (*Post, error) {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("get_by_id", "success", time.Since(start))
	}()

	query := `SELECT id, user_id, title, body, created_at, updated_at FROM posts WHERE id = $1`

	var post Post
	err := ps.db.QueryRowContext(ctx, query, id).Scan(
		&post.Id, &post.UserId, &post.Title, &post.Body, &post.CreatedAt, &post.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		metrics.RecordDBOperation("get_by_id", "error", time.Since(start))
		logger.ErrorContext(ctx, "failed to query post", "id", id, "error", err)
		return nil, err
	}

	return &post, nil
}

// Create creates a new post
func (ps *PostgresStorage) Create(ctx context.Context, post *Post) error {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("create", "success", time.Since(start))
	}()

	if err := post.Validate(); err != nil {
		return err
	}

	query := `
		INSERT INTO posts (user_id, title, body, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	err := ps.db.QueryRowContext(
		ctx, query,
		post.UserId, post.Title, post.Body, now, now,
	).Scan(&post.Id, &post.CreatedAt, &post.UpdatedAt)

	if err != nil {
		metrics.RecordDBOperation("create", "error", time.Since(start))
		logger.ErrorContext(ctx, "failed to create post", "error", err)
		return err
	}

	metrics.RecordPostAdded()
	logger.InfoContext(ctx, "post created", "id", post.Id)
	return nil
}

// Update updates an existing post
func (ps *PostgresStorage) Update(ctx context.Context, post *Post) error {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("update", "success", time.Since(start))
	}()

	if err := post.Validate(); err != nil {
		return err
	}

	query := `
		UPDATE posts
		SET user_id = $1, title = $2, body = $3, updated_at = $4
		WHERE id = $5
		RETURNING updated_at
	`

	err := ps.db.QueryRowContext(
		ctx, query,
		post.UserId, post.Title, post.Body, time.Now(), post.Id,
	).Scan(&post.UpdatedAt)

	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		metrics.RecordDBOperation("update", "error", time.Since(start))
		logger.ErrorContext(ctx, "failed to update post", "id", post.Id, "error", err)
		return err
	}

	logger.InfoContext(ctx, "post updated", "id", post.Id)
	return nil
}

// Delete deletes a post by ID
func (ps *PostgresStorage) Delete(ctx context.Context, id int) error {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("delete", "success", time.Since(start))
	}()

	query := `DELETE FROM posts WHERE id = $1`

	result, err := ps.db.ExecContext(ctx, query, id)
	if err != nil {
		metrics.RecordDBOperation("delete", "error", time.Since(start))
		logger.ErrorContext(ctx, "failed to delete post", "id", id, "error", err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	logger.InfoContext(ctx, "post deleted", "id", id)
	return nil
}

// BatchCreate creates multiple posts in a batch
func (ps *PostgresStorage) BatchCreate(ctx context.Context, posts []Post) error {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("batch_create", "success", time.Since(start))
	}()

	if len(posts) == 0 {
		return nil
	}

	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		metrics.RecordDBOperation("batch_create", "error", time.Since(start))
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// id is intentionally NOT in the insert column list: these are new
	// posts, so the id column's SERIAL/IDENTITY default must assign it,
	// same as the singular Create. An earlier version of this method
	// inserted an explicit id (always 0 for freshly-constructed Post
	// values) with "ON CONFLICT (id) DO NOTHING" — every post after the
	// first in a batch silently collided on id=0 and was dropped.
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO posts (user_id, title, body, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`)
	if err != nil {
		metrics.RecordDBOperation("batch_create", "error", time.Since(start))
		return err
	}
	defer func() { _ = stmt.Close() }()

	now := time.Now()
	for i := range posts {
		createdAt := posts[i].CreatedAt
		if createdAt.IsZero() {
			createdAt = now
		}
		updatedAt := posts[i].UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = now
		}

		err := stmt.QueryRowContext(ctx, posts[i].UserId, posts[i].Title, posts[i].Body, createdAt, updatedAt).
			Scan(&posts[i].Id, &posts[i].CreatedAt, &posts[i].UpdatedAt)
		if err != nil {
			metrics.RecordDBOperation("batch_create", "error", time.Since(start))
			logger.ErrorContext(ctx, "failed to insert post in batch", "index", i, "error", err)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		metrics.RecordDBOperation("batch_create", "error", time.Since(start))
		return err
	}

	metrics.RecordPostsFetched(len(posts))
	logger.InfoContext(ctx, "batch posts created", "count", len(posts))
	return nil
}

// Count returns the total number of posts
func (ps *PostgresStorage) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM posts`

	var count int
	err := ps.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		logger.ErrorContext(ctx, "failed to count posts", "error", err)
		return 0, err
	}

	return count, nil
}

// Close closes the database connection
func (ps *PostgresStorage) Close() error {
	return ps.db.Close()
}
