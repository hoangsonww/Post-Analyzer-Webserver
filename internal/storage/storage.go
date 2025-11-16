package storage

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when a post is not found
	ErrNotFound = errors.New("post not found")
	// ErrInvalidInput is returned when input validation fails
	ErrInvalidInput = errors.New("invalid input")
)

// Post represents a post in the system
type Post struct {
	UserId    int       `json:"userId"`
	Id        int       `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

// Storage defines the interface for post storage operations
type Storage interface {
	// GetAll retrieves all posts
	GetAll(ctx context.Context) ([]Post, error)

	// GetByID retrieves a post by ID
	GetByID(ctx context.Context, id int) (*Post, error)

	// Create creates a new post
	Create(ctx context.Context, post *Post) error

	// Update updates an existing post
	Update(ctx context.Context, post *Post) error

	// Delete deletes a post by ID
	Delete(ctx context.Context, id int) error

	// BatchCreate creates multiple posts in a batch
	BatchCreate(ctx context.Context, posts []Post) error

	// Count returns the total number of posts
	Count(ctx context.Context) (int, error)

	// Close closes the storage connection
	Close() error
}

// Validate validates a post
func (p *Post) Validate() error {
	if p.Title == "" {
		return errors.New("title is required")
	}
	if len(p.Title) > 500 {
		return errors.New("title too long (max 500 characters)")
	}
	if p.Body == "" {
		return errors.New("body is required")
	}
	if len(p.Body) > 10000 {
		return errors.New("body too long (max 10000 characters)")
	}
	return nil
}
