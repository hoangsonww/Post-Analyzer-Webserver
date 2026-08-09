package service

import (
	"context"
	"testing"
	"time"

	"Post_Analyzer_Webserver/internal/cache"
	"Post_Analyzer_Webserver/internal/models"
	"Post_Analyzer_Webserver/internal/storage"
)

// fakeStorage is an in-memory storage.Storage for exercising PostService
// business logic (filtering, sorting, pagination, analysis) without a
// real database.
type fakeStorage struct {
	posts  map[int]storage.Post
	nextID int
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{posts: make(map[int]storage.Post), nextID: 1}
}

func (f *fakeStorage) GetAll(ctx context.Context) ([]storage.Post, error) {
	out := make([]storage.Post, 0, len(f.posts))
	for _, p := range f.posts {
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeStorage) GetByID(ctx context.Context, id int) (*storage.Post, error) {
	p, ok := f.posts[id]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return &p, nil
}

func (f *fakeStorage) Create(ctx context.Context, post *storage.Post) error {
	post.Id = f.nextID
	f.nextID++
	post.CreatedAt = time.Now()
	post.UpdatedAt = post.CreatedAt
	f.posts[post.Id] = *post
	return nil
}

func (f *fakeStorage) Update(ctx context.Context, post *storage.Post) error {
	if _, ok := f.posts[post.Id]; !ok {
		return storage.ErrNotFound
	}
	post.UpdatedAt = time.Now()
	f.posts[post.Id] = *post
	return nil
}

func (f *fakeStorage) Delete(ctx context.Context, id int) error {
	if _, ok := f.posts[id]; !ok {
		return storage.ErrNotFound
	}
	delete(f.posts, id)
	return nil
}

func (f *fakeStorage) BatchCreate(ctx context.Context, posts []storage.Post) error {
	for i := range posts {
		if err := f.Create(ctx, &posts[i]); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeStorage) Count(ctx context.Context) (int, error) { return len(f.posts), nil }
func (f *fakeStorage) Close() error                           { return nil }

func newTestService() (*PostService, *fakeStorage) {
	fs := newFakeStorage()
	return NewPostService(fs, nil), fs
}

func TestCreateAndGetByID(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, err := svc.Create(ctx, &models.CreatePostRequest{UserID: 1, Title: "Hello", Body: "World"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected a non-zero generated ID")
	}

	got, err := svc.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Title != "Hello" || got.Body != "World" {
		t.Errorf("unexpected post: %+v", got)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	svc, _ := newTestService()
	if _, err := svc.GetByID(context.Background(), 9999); err == nil {
		t.Error("expected an error for a nonexistent post ID")
	}
}

func TestCreate_ValidationErrors(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	cases := []*models.CreatePostRequest{
		{UserID: 1, Title: "", Body: "has body"},
		{UserID: 1, Title: "has title", Body: ""},
	}
	for _, req := range cases {
		if _, err := svc.Create(ctx, req); err == nil {
			t.Errorf("expected validation error for %+v", req)
		}
	}
}

func TestUpdate_PartialFields(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.Create(ctx, &models.CreatePostRequest{UserID: 1, Title: "Original", Body: "Body"})

	updated, err := svc.Update(ctx, created.ID, &models.UpdatePostRequest{Title: "Changed"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Title != "Changed" {
		t.Errorf("expected title to change, got %q", updated.Title)
	}
	if updated.Body != "Body" {
		t.Errorf("expected body to be left unchanged, got %q", updated.Body)
	}
}

func TestDelete(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.Create(ctx, &models.CreatePostRequest{UserID: 1, Title: "T", Body: "B"})
	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := svc.GetByID(ctx, created.ID); err == nil {
		t.Error("expected deleted post to be gone")
	}
}

func TestGetAll_FilterSortPaginate(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	titles := []string{"Banana", "Apple", "Cherry"}
	for i, title := range titles {
		userID := 1
		if i == 2 {
			userID = 2
		}
		if _, err := svc.Create(ctx, &models.CreatePostRequest{UserID: userID, Title: title, Body: "body " + title}); err != nil {
			t.Fatalf("setup Create failed: %v", err)
		}
	}

	// Filter by userID. Note: GetAll returns a nil *PaginationMeta when
	// pagination is nil (see calculatePagination) — the HTTP API never
	// hits this since its handler always builds pagination params with
	// defaults, but callers of the Go API directly need to know.
	uid := 1
	filtered, meta, err := svc.GetAll(ctx, &models.PostFilter{UserID: &uid}, nil)
	if err != nil {
		t.Fatalf("GetAll (filter) failed: %v", err)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2 posts for userID=1, got %d", len(filtered))
	}
	if meta != nil {
		t.Errorf("expected nil pagination meta when pagination param is nil, got %+v", meta)
	}

	// Same filter, but with pagination requested this time — this is the
	// path that actually populates TotalItems.
	_, metaWithPagination, err := svc.GetAll(ctx, &models.PostFilter{UserID: &uid}, &models.PaginationParams{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("GetAll (filter+pagination) failed: %v", err)
	}
	if metaWithPagination == nil || metaWithPagination.TotalItems != 2 {
		t.Errorf("expected pagination TotalItems=2, got %+v", metaWithPagination)
	}

	// Sort by title ascending
	sorted, _, err := svc.GetAll(ctx, &models.PostFilter{SortBy: "title", SortOrder: "asc"}, nil)
	if err != nil {
		t.Fatalf("GetAll (sort) failed: %v", err)
	}
	if len(sorted) != 3 || sorted[0].Title != "Apple" || sorted[2].Title != "Cherry" {
		t.Fatalf("expected alphabetical order, got %v", titlesOf(sorted))
	}

	// Pagination: page 1 of size 2
	page, pmeta, err := svc.GetAll(ctx, nil, &models.PaginationParams{Page: 1, PageSize: 2, Offset: 0})
	if err != nil {
		t.Fatalf("GetAll (paginate) failed: %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("expected page size 2, got %d", len(page))
	}
	if !pmeta.HasNext || pmeta.HasPrev {
		t.Errorf("unexpected pagination meta on first page: %+v", pmeta)
	}
}

func titlesOf(posts []models.Post) []string {
	out := make([]string, len(posts))
	for i, p := range posts {
		out[i] = p.Title
	}
	return out
}

func TestAnalyzeCharacterFrequency(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	if _, err := svc.Create(ctx, &models.CreatePostRequest{UserID: 1, Title: "aa", Body: "bb"}); err != nil {
		t.Fatalf("setup Create failed: %v", err)
	}

	result, err := svc.AnalyzeCharacterFrequency(ctx)
	if err != nil {
		t.Fatalf("AnalyzeCharacterFrequency failed: %v", err)
	}
	if result.TotalPosts != 1 {
		t.Errorf("expected TotalPosts=1, got %d", result.TotalPosts)
	}
	if result.TotalCharacters != 5 { // "aa" + " " + "bb" — title and body are joined with a space
		t.Errorf("expected TotalCharacters=5, got %d", result.TotalCharacters)
	}
	if result.UniqueChars != 3 { // 'a', 'b', and the joining space
		t.Errorf("expected UniqueChars=3, got %d", result.UniqueChars)
	}
}

// TestCache_InvalidatedOnWrite exercises the real in-memory cache (not a
// fake) to confirm GetByID actually invalidates on Update rather than
// serving stale data — the specific bug class caching introduces.
func TestCache_InvalidatedOnWrite(t *testing.T) {
	fs := newFakeStorage()
	svc := NewPostService(fs, cache.NewMemoryCache())
	ctx := context.Background()

	created, err := svc.Create(ctx, &models.CreatePostRequest{UserID: 1, Title: "Original", Body: "Body"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Warm the cache.
	if _, err := svc.GetByID(ctx, created.ID); err != nil {
		t.Fatalf("GetByID (warm) failed: %v", err)
	}

	if _, err := svc.Update(ctx, created.ID, &models.UpdatePostRequest{Title: "Updated"}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err := svc.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID (post-update) failed: %v", err)
	}
	if got.Title != "Updated" {
		t.Errorf("expected cache to be invalidated and reflect the update, got stale title %q", got.Title)
	}
}
