package storage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"

	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/metrics"
)

// FileStorage implements Storage interface using JSON file
type FileStorage struct {
	filePath string
	mu       sync.RWMutex
}

// NewFileStorage creates a new file-based storage
func NewFileStorage(filePath string) (*FileStorage, error) {
	fs := &FileStorage{
		filePath: filePath,
	}

	// Initialize file if it doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := fs.writeToFile([]Post{}); err != nil {
			return nil, err
		}
	}

	return fs, nil
}

// GetAll retrieves all posts
func (fs *FileStorage) GetAll(ctx context.Context) ([]Post, error) {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("get_all", "success", time.Since(start))
	}()

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	posts, err := fs.readFromFile()
	if err != nil {
		metrics.RecordDBOperation("get_all", "error", time.Since(start))
		logger.ErrorContext(ctx, "failed to read posts from file", "error", err)
		return nil, err
	}

	metrics.RecordPostsTotal(len(posts))
	return posts, nil
}

// GetByID retrieves a post by ID
func (fs *FileStorage) GetByID(ctx context.Context, id int) (*Post, error) {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("get_by_id", "success", time.Since(start))
	}()

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	posts, err := fs.readFromFile()
	if err != nil {
		metrics.RecordDBOperation("get_by_id", "error", time.Since(start))
		return nil, err
	}

	for _, post := range posts {
		if post.Id == id {
			return &post, nil
		}
	}

	return nil, ErrNotFound
}

// Create creates a new post
func (fs *FileStorage) Create(ctx context.Context, post *Post) error {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("create", "success", time.Since(start))
	}()

	if err := post.Validate(); err != nil {
		return err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	posts, err := fs.readFromFile()
	if err != nil {
		metrics.RecordDBOperation("create", "error", time.Since(start))
		return err
	}

	// Generate new ID
	maxID := 0
	for _, p := range posts {
		if p.Id > maxID {
			maxID = p.Id
		}
	}
	post.Id = maxID + 1
	post.CreatedAt = time.Now()
	post.UpdatedAt = time.Now()

	posts = append(posts, *post)

	if err := fs.writeToFile(posts); err != nil {
		metrics.RecordDBOperation("create", "error", time.Since(start))
		return err
	}

	metrics.RecordPostAdded()
	logger.InfoContext(ctx, "post created", "id", post.Id)
	return nil
}

// Update updates an existing post
func (fs *FileStorage) Update(ctx context.Context, post *Post) error {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("update", "success", time.Since(start))
	}()

	if err := post.Validate(); err != nil {
		return err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	posts, err := fs.readFromFile()
	if err != nil {
		metrics.RecordDBOperation("update", "error", time.Since(start))
		return err
	}

	found := false
	for i, p := range posts {
		if p.Id == post.Id {
			post.UpdatedAt = time.Now()
			posts[i] = *post
			found = true
			break
		}
	}

	if !found {
		return ErrNotFound
	}

	if err := fs.writeToFile(posts); err != nil {
		metrics.RecordDBOperation("update", "error", time.Since(start))
		return err
	}

	logger.InfoContext(ctx, "post updated", "id", post.Id)
	return nil
}

// Delete deletes a post by ID
func (fs *FileStorage) Delete(ctx context.Context, id int) error {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("delete", "success", time.Since(start))
	}()

	fs.mu.Lock()
	defer fs.mu.Unlock()

	posts, err := fs.readFromFile()
	if err != nil {
		metrics.RecordDBOperation("delete", "error", time.Since(start))
		return err
	}

	found := false
	newPosts := make([]Post, 0, len(posts))
	for _, p := range posts {
		if p.Id != id {
			newPosts = append(newPosts, p)
		} else {
			found = true
		}
	}

	if !found {
		return ErrNotFound
	}

	if err := fs.writeToFile(newPosts); err != nil {
		metrics.RecordDBOperation("delete", "error", time.Since(start))
		return err
	}

	logger.InfoContext(ctx, "post deleted", "id", id)
	return nil
}

// BatchCreate creates multiple posts in a batch
func (fs *FileStorage) BatchCreate(ctx context.Context, newPosts []Post) error {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("batch_create", "success", time.Since(start))
	}()

	fs.mu.Lock()
	defer fs.mu.Unlock()

	existingPosts, err := fs.readFromFile()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		metrics.RecordDBOperation("batch_create", "error", time.Since(start))
		return err
	}

	// Mirrors Create's ID-assignment: append to (not overwrite) existing
	// posts, and assign each new post the next sequential ID. Earlier
	// version wrote newPosts alone (silently dropping existingPosts) and
	// never assigned IDs, so freshly-constructed batch posts all kept
	// Id=0.
	maxID := 0
	for _, p := range existingPosts {
		if p.Id > maxID {
			maxID = p.Id
		}
	}

	now := time.Now()
	for i := range newPosts {
		maxID++
		newPosts[i].Id = maxID
		if newPosts[i].CreatedAt.IsZero() {
			newPosts[i].CreatedAt = now
		}
		if newPosts[i].UpdatedAt.IsZero() {
			newPosts[i].UpdatedAt = now
		}
	}

	allPosts := append(existingPosts, newPosts...)
	if err := fs.writeToFile(allPosts); err != nil {
		metrics.RecordDBOperation("batch_create", "error", time.Since(start))
		return err
	}

	metrics.RecordPostsFetched(len(newPosts))
	logger.InfoContext(ctx, "batch posts created", "count", len(newPosts), "previous_count", len(existingPosts))
	return nil
}

// Count returns the total number of posts
func (fs *FileStorage) Count(ctx context.Context) (int, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	posts, err := fs.readFromFile()
	if err != nil {
		return 0, err
	}

	return len(posts), nil
}

// Close closes the storage (no-op for file storage)
func (fs *FileStorage) Close() error {
	return nil
}

// readFromFile reads posts from the JSON file
func (fs *FileStorage) readFromFile() ([]Post, error) {
	data, err := os.ReadFile(fs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Post{}, nil
		}
		return nil, err
	}

	if len(data) == 0 {
		return []Post{}, nil
	}

	var posts []Post
	if err := json.Unmarshal(data, &posts); err != nil {
		return nil, err
	}

	return posts, nil
}

// writeToFile writes posts to the JSON file
func (fs *FileStorage) writeToFile(posts []Post) error {
	data, err := json.MarshalIndent(posts, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fs.filePath, data, 0644)
}
