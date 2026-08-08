package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"Post_Analyzer_Webserver/config"
	apperrors "Post_Analyzer_Webserver/internal/errors"
	"Post_Analyzer_Webserver/internal/models"
)

// fakePostClient is a minimal rpcclient.PostClient fake — only what the
// legacy web UI handlers actually call.
type fakePostClient struct {
	posts         []models.Post
	getAllErr     error
	createErr     error
	bulkCreateErr error
}

func (f *fakePostClient) GetAll(ctx context.Context, filter *models.PostFilter, pagination *models.PaginationParams) ([]models.Post, *models.PaginationMeta, error) {
	if f.getAllErr != nil {
		return nil, nil, f.getAllErr
	}
	return f.posts, &models.PaginationMeta{TotalItems: len(f.posts)}, nil
}
func (f *fakePostClient) GetByID(ctx context.Context, id int) (*models.Post, error) {
	return nil, apperrors.NewNotFound("Post")
}
func (f *fakePostClient) Create(ctx context.Context, req *models.CreatePostRequest) (*models.Post, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	p := models.Post{ID: len(f.posts) + 1, UserID: req.UserID, Title: req.Title, Body: req.Body}
	f.posts = append(f.posts, p)
	return &p, nil
}
func (f *fakePostClient) Update(ctx context.Context, id int, req *models.UpdatePostRequest) (*models.Post, error) {
	return nil, apperrors.NewNotFound("Post")
}
func (f *fakePostClient) Delete(ctx context.Context, id int) error { return nil }
func (f *fakePostClient) BulkCreate(ctx context.Context, req *models.BulkCreateRequest) (*models.BulkCreateResponse, error) {
	if f.bulkCreateErr != nil {
		return nil, f.bulkCreateErr
	}
	for _, p := range req.Posts {
		f.posts = append(f.posts, models.Post{ID: len(f.posts) + 1, UserID: p.UserID, Title: p.Title, Body: p.Body})
	}
	return &models.BulkCreateResponse{Created: len(req.Posts)}, nil
}
func (f *fakePostClient) AnalyzeCharacterFrequency(ctx context.Context) (*models.AnalyticsResult, error) {
	return &models.AnalyticsResult{}, nil
}

// chdirRepoRoot changes into the repo root for the duration of a test —
// New() loads home.html via a path relative to the process's working
// directory (as it does in production, where the binary runs from the
// repo/container root), so tests that exercise template rendering need
// the same cwd a real deployment has.
func chdirRepoRoot(t *testing.T) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatalf("Chdir to repo root failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

func testConfig() *config.Config {
	return &config.Config{External: config.ExternalConfig{HTTPTimeout: 5_000_000_000}}
}

func TestNew_LoadsTemplate(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if h.template == nil {
		t.Error("expected New to load a non-nil template")
	}
}

func TestHealth(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	h.Health(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"status":"healthy"`) {
		t.Errorf("unexpected body: %s", rr.Body.String())
	}
}

func TestReadiness_Ready(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/readiness", nil)
	rr := httptest.NewRecorder()
	h.Readiness(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReadiness_NotReadyWhenRPCFails(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{getAllErr: apperrors.NewInternalError(nil)}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/readiness", nil)
	rr := httptest.NewRecorder()
	h.Readiness(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when postsvc is unreachable, got %d", rr.Code)
	}
}

func TestHome_RendersPostsFromRPC(t *testing.T) {
	chdirRepoRoot(t)
	fake := &fakePostClient{posts: []models.Post{{ID: 1, Title: "Hello", Body: "World"}}}
	h, err := New(fake, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.Home(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Hello") {
		t.Error("expected the rendered page to include the post title")
	}
}

func TestAddPost_GET_ShowsForm(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/add", nil)
	rr := httptest.NewRecorder()
	h.AddPost(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAddPost_POST_CreatesPost(t *testing.T) {
	chdirRepoRoot(t)
	fake := &fakePostClient{}
	h, err := New(fake, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/add", strings.NewReader("title=New+Post&body=Some+body"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.AddPost(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if len(fake.posts) != 1 || fake.posts[0].Title != "New Post" {
		t.Errorf("expected the post to be created via the fake RPC client, got %+v", fake.posts)
	}
}

func TestAddPost_POST_MissingFieldsShowsError(t *testing.T) {
	chdirRepoRoot(t)
	fake := &fakePostClient{}
	h, err := New(fake, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/add", strings.NewReader("title=&body="))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.AddPost(rr, req)

	if len(fake.posts) != 0 {
		t.Error("expected no post to be created when title/body are empty")
	}
}

func TestAddPost_SanitizesXSSInput(t *testing.T) {
	chdirRepoRoot(t)
	fake := &fakePostClient{}
	h, err := New(fake, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/add",
		strings.NewReader("title="+`%3Cscript%3Ealert(1)%3C%2Fscript%3EHello`+"&body=safe+body"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.AddPost(rr, req)

	if len(fake.posts) != 1 {
		t.Fatalf("expected the post to be created, got %+v", fake.posts)
	}
	if strings.Contains(fake.posts[0].Title, "<script>") {
		t.Errorf("expected the script tag to be sanitized out of the stored title, got %q", fake.posts[0].Title)
	}
}

func TestAnalyzePosts_RendersCharacterFrequency(t *testing.T) {
	chdirRepoRoot(t)
	fake := &fakePostClient{posts: []models.Post{{Title: "aa", Body: "bb"}}}
	h, err := New(fake, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/analyze", nil)
	rr := httptest.NewRecorder()
	h.AnalyzePosts(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestFetchPostsFromAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"userId":1,"id":1,"title":"t","body":"b"}]`))
	}))
	defer server.Close()

	chdirRepoRoot(t)
	cfg := testConfig()
	cfg.External.JSONPlaceholderURL = server.URL
	h, err := New(&fakePostClient{}, cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	posts, err := h.fetchPostsFromAPI(context.Background())
	if err != nil {
		t.Fatalf("fetchPostsFromAPI failed: %v", err)
	}
	if len(posts) != 1 || posts[0].Title != "t" {
		t.Errorf("unexpected posts: %+v", posts)
	}
}

func TestFetchPostsFromAPI_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	chdirRepoRoot(t)
	cfg := testConfig()
	cfg.External.JSONPlaceholderURL = server.URL
	h, err := New(&fakePostClient{}, cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if _, err := h.fetchPostsFromAPI(context.Background()); err == nil {
		t.Error("expected an error for a non-200 upstream response")
	}
}

func TestFetchPosts_Handler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"userId":1,"id":1,"title":"external","body":"b"}]`))
	}))
	defer server.Close()

	chdirRepoRoot(t)
	cfg := testConfig()
	cfg.External.JSONPlaceholderURL = server.URL
	fake := &fakePostClient{}
	h, err := New(fake, cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/fetch", nil)
	rr := httptest.NewRecorder()
	h.FetchPosts(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if len(fake.posts) != 1 {
		t.Errorf("expected the fetched post to be stored via BulkCreate, got %+v", fake.posts)
	}
}

func TestCountCharacters_SmallText(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	got := h.countCharacters("aab")
	if got['a'] != 2 || got['b'] != 1 {
		t.Errorf("unexpected char counts: %+v", got)
	}
}

func TestCountCharacters_LargeTextMatchesSmallTextPath(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Repeat a known pattern past the 1000-byte concurrent-path threshold
	// and verify the concurrent chunked counting produces the exact same
	// totals as manually counting the string — the concurrent path must
	// not lose or double-count characters across chunk boundaries.
	unit := "abc"
	text := strings.Repeat(unit, 500) // 1500 bytes, well past chunkSize=1000

	got := h.countCharacters(text)
	wantA, wantB, wantC := 500, 500, 500
	if got['a'] != wantA || got['b'] != wantB || got['c'] != wantC {
		t.Errorf("concurrent counting mismatch: got a=%d b=%d c=%d, want a=%d b=%d c=%d",
			got['a'], got['b'], got['c'], wantA, wantB, wantC)
	}

	var total int
	for _, c := range got {
		total += c
	}
	if total != len(text) {
		t.Errorf("expected total character count %d to equal input length %d", total, len(text))
	}
}

func TestSanitizeInput_EscapesHTML(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	got := h.sanitizeInput(`<b>bold</b>`)
	if strings.Contains(got, "<b>") {
		t.Errorf("expected HTML tags to be escaped, got %q", got)
	}
}

func TestSanitizeInput_StripsScriptTags(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	got := h.sanitizeInput(`<script>alert('xss')</script>hello`)
	if strings.Contains(strings.ToLower(got), "<script") {
		t.Errorf("expected script tags to be stripped, got %q", got)
	}
}

func TestSanitizeInput_StripsEventHandlers(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	got := h.sanitizeInput(`<img onerror=alert(1)>`)
	if strings.Contains(got, "onerror=") {
		t.Errorf("expected event handler attributes to be stripped, got %q", got)
	}
}

func TestSanitizeInput_TrimsWhitespace(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if got := h.sanitizeInput("  hello  "); got != "hello" {
		t.Errorf("expected whitespace to be trimmed, got %q", got)
	}
}

func TestClose_IsANoOp(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if err := h.Close(); err != nil {
		t.Errorf("expected Close to be a no-op, got %v", err)
	}
}
