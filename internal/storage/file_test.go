package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStorage_CreateAndGet(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_posts.json")

	store, err := NewFileStorage(filePath)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create a post
	post := &Post{
		UserId: 1,
		Title:  "Test Post",
		Body:   "This is a test post body",
	}

	err = store.Create(ctx, post)
	if err != nil {
		t.Fatalf("Failed to create post: %v", err)
	}

	if post.Id == 0 {
		t.Error("Post ID should be assigned")
	}

	// Retrieve the post
	retrieved, err := store.GetByID(ctx, post.Id)
	if err != nil {
		t.Fatalf("Failed to get post: %v", err)
	}

	if retrieved.Title != post.Title {
		t.Errorf("Expected title %s, got %s", post.Title, retrieved.Title)
	}
	if retrieved.Body != post.Body {
		t.Errorf("Expected body %s, got %s", post.Body, retrieved.Body)
	}
}

func TestFileStorage_GetAll(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_posts.json")

	store, err := NewFileStorage(filePath)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create multiple posts
	for i := 1; i <= 3; i++ {
		post := &Post{
			UserId: i,
			Title:  "Test Post " + string(rune(i)),
			Body:   "Test body",
		}
		if err := store.Create(ctx, post); err != nil {
			t.Fatalf("Failed to create post: %v", err)
		}
	}

	// Get all posts
	posts, err := store.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to get all posts: %v", err)
	}

	if len(posts) != 3 {
		t.Errorf("Expected 3 posts, got %d", len(posts))
	}
}

func TestFileStorage_Update(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_posts.json")

	store, err := NewFileStorage(filePath)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create a post
	post := &Post{
		UserId: 1,
		Title:  "Original Title",
		Body:   "Original Body",
	}
	if err := store.Create(ctx, post); err != nil {
		t.Fatalf("Failed to create post: %v", err)
	}

	// Update the post
	post.Title = "Updated Title"
	post.Body = "Updated Body"
	if err := store.Update(ctx, post); err != nil {
		t.Fatalf("Failed to update post: %v", err)
	}

	// Retrieve and verify
	retrieved, err := store.GetByID(ctx, post.Id)
	if err != nil {
		t.Fatalf("Failed to get post: %v", err)
	}

	if retrieved.Title != "Updated Title" {
		t.Errorf("Expected updated title, got %s", retrieved.Title)
	}
	if retrieved.Body != "Updated Body" {
		t.Errorf("Expected updated body, got %s", retrieved.Body)
	}
}

func TestFileStorage_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_posts.json")

	store, err := NewFileStorage(filePath)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create a post
	post := &Post{
		UserId: 1,
		Title:  "To Be Deleted",
		Body:   "This post will be deleted",
	}
	if err := store.Create(ctx, post); err != nil {
		t.Fatalf("Failed to create post: %v", err)
	}

	// Delete the post
	if err := store.Delete(ctx, post.Id); err != nil {
		t.Fatalf("Failed to delete post: %v", err)
	}

	// Try to retrieve - should get error
	_, err = store.GetByID(ctx, post.Id)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestFileStorage_Validation(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_posts.json")

	store, err := NewFileStorage(filePath)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Test empty title
	post := &Post{
		UserId: 1,
		Title:  "",
		Body:   "Body",
	}
	err = store.Create(ctx, post)
	if err == nil {
		t.Error("Expected error for empty title")
	}

	// Test empty body
	post = &Post{
		UserId: 1,
		Title:  "Title",
		Body:   "",
	}
	err = store.Create(ctx, post)
	if err == nil {
		t.Error("Expected error for empty body")
	}
}

func TestFileStorage_Count(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_posts.json")

	store, err := NewFileStorage(filePath)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Initial count should be 0
	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to count posts: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}

	// Create posts
	for i := 0; i < 5; i++ {
		post := &Post{
			UserId: 1,
			Title:  "Test",
			Body:   "Body",
		}
		store.Create(ctx, post)
	}

	// Count should be 5
	count, err = store.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to count posts: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected count 5, got %d", count)
	}
}

func TestFileStorage_BatchCreate(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_posts.json")

	store, err := NewFileStorage(filePath)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create batch of posts
	posts := []Post{
		{Id: 1, UserId: 1, Title: "Post 1", Body: "Body 1"},
		{Id: 2, UserId: 1, Title: "Post 2", Body: "Body 2"},
		{Id: 3, UserId: 1, Title: "Post 3", Body: "Body 3"},
	}

	err = store.BatchCreate(ctx, posts)
	if err != nil {
		t.Fatalf("Failed to batch create posts: %v", err)
	}

	// Verify count
	count, _ := store.Count(ctx)
	if count != 3 {
		t.Errorf("Expected 3 posts, got %d", count)
	}

	// Verify individual posts exist
	for _, post := range posts {
		retrieved, err := store.GetByID(ctx, post.Id)
		if err != nil {
			t.Errorf("Failed to get post %d: %v", post.Id, err)
		}
		if retrieved.Title != post.Title {
			t.Errorf("Expected title %s, got %s", post.Title, retrieved.Title)
		}
	}
}

func TestFileStorage_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_posts.json")

	store, err := NewFileStorage(filePath)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create posts concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			post := &Post{
				UserId: id,
				Title:  "Concurrent Post",
				Body:   "Body",
			}
			store.Create(ctx, post)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all posts were created
	count, _ := store.Count(ctx)
	if count != 10 {
		t.Errorf("Expected 10 posts, got %d", count)
	}
}

func TestFileStorage_FileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "nonexistent.json")

	// Should create the file if it doesn't exist
	store, err := NewFileStorage(filePath)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}
	defer store.Close()

	// Verify file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Expected file to be created")
	}
}
