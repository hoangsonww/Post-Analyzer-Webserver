package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "Post_Analyzer_Webserver/internal/errors"
	"Post_Analyzer_Webserver/internal/models"
)

func TestWebListPosts_ReturnsPostsAndPagination(t *testing.T) {
	chdirRepoRoot(t)
	fake := &fakePostClient{posts: []models.Post{{ID: 1, Title: "Hello", Body: "World"}}}
	h, err := New(fake, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/web/posts?search=hi&sortBy=title&sortOrder=asc&page=2&pageSize=5", nil)
	rr := httptest.NewRecorder()
	h.WebListPosts(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var out struct {
		Posts      []models.Post          `json:"posts"`
		Pagination *models.PaginationMeta `json:"pagination"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(out.Posts) != 1 || out.Posts[0].Title != "Hello" {
		t.Errorf("unexpected posts in response: %+v", out.Posts)
	}
	if out.Pagination == nil || out.Pagination.TotalItems != 1 {
		t.Errorf("expected pagination metadata, got %+v", out.Pagination)
	}

	if fake.lastFilter == nil || fake.lastFilter.Search != "hi" || fake.lastFilter.SortBy != "title" || fake.lastFilter.SortOrder != "asc" {
		t.Errorf("expected search/sort query params to reach postsvc, got %+v", fake.lastFilter)
	}
	if fake.lastPagination == nil || fake.lastPagination.Page != 2 || fake.lastPagination.PageSize != 5 {
		t.Errorf("expected page/pageSize query params to reach postsvc, got %+v", fake.lastPagination)
	}
}

func TestWebListPosts_EmptyResultIsEmptyArrayNotNull(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/web/posts", nil)
	rr := httptest.NewRecorder()
	h.WebListPosts(rr, req)

	if !strings.Contains(rr.Body.String(), `"posts":[]`) {
		t.Errorf(`expected "posts":[] (not null) in response, got: %s`, rr.Body.String())
	}
}

func TestWebListPosts_RPCError(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{getAllErr: apperrors.NewInternalError(nil)}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/web/posts", nil)
	rr := httptest.NewRecorder()
	h.WebListPosts(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestWebGetPost_Success(t *testing.T) {
	chdirRepoRoot(t)
	fake := &fakePostClient{getByIDResult: &models.Post{ID: 7, Title: "Found"}}
	h, err := New(fake, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/web/posts/7", nil)
	req.SetPathValue("id", "7")
	rr := httptest.NewRecorder()
	h.WebGetPost(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Found") {
		t.Errorf("expected the post in the response, got: %s", rr.Body.String())
	}
}

func TestWebGetPost_InvalidID(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/web/posts/abc", nil)
	req.SetPathValue("id", "abc")
	rr := httptest.NewRecorder()
	h.WebGetPost(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for a non-numeric id, got %d", rr.Code)
	}
}

func TestWebGetPost_NotFound(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/web/posts/404", nil)
	req.SetPathValue("id", "404")
	rr := httptest.NewRecorder()
	h.WebGetPost(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestWebCreatePost_Success(t *testing.T) {
	chdirRepoRoot(t)
	fake := &fakePostClient{}
	h, err := New(fake, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	body := strings.NewReader(`{"title":"New","body":"Body text"}`)
	req := httptest.NewRequest(http.MethodPost, "/web/posts", body)
	rr := httptest.NewRecorder()
	h.WebCreatePost(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if len(fake.posts) != 1 || fake.posts[0].Title != "New" {
		t.Errorf("expected the post to be created, got %+v", fake.posts)
	}
}

func TestWebCreatePost_DefaultsUserID(t *testing.T) {
	chdirRepoRoot(t)
	fake := &fakePostClient{}
	h, err := New(fake, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	body := strings.NewReader(`{"title":"New","body":"Body text"}`)
	req := httptest.NewRequest(http.MethodPost, "/web/posts", body)
	rr := httptest.NewRecorder()
	h.WebCreatePost(rr, req)

	if len(fake.posts) != 1 || fake.posts[0].UserID != 1 {
		t.Errorf("expected userId to default to 1, got %+v", fake.posts)
	}
}

func TestWebCreatePost_MissingFields(t *testing.T) {
	chdirRepoRoot(t)
	fake := &fakePostClient{}
	h, err := New(fake, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/web/posts", strings.NewReader(`{"title":"","body":""}`))
	rr := httptest.NewRecorder()
	h.WebCreatePost(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
	if len(fake.posts) != 0 {
		t.Error("expected no post to be created")
	}
}

func TestWebCreatePost_InvalidJSON(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/web/posts", strings.NewReader(`not json`))
	rr := httptest.NewRecorder()
	h.WebCreatePost(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for invalid JSON, got %d", rr.Code)
	}
}

func TestWebCreatePost_SanitizesXSS(t *testing.T) {
	chdirRepoRoot(t)
	fake := &fakePostClient{}
	h, err := New(fake, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/web/posts", strings.NewReader(`{"title":"<script>alert(1)</script>Hi","body":"safe"}`))
	rr := httptest.NewRecorder()
	h.WebCreatePost(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(fake.posts[0].Title, "<script>") {
		t.Errorf("expected script tag to be sanitized, got %q", fake.posts[0].Title)
	}
}

func TestWebCreatePost_RPCError(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{createErr: apperrors.NewInternalError(nil)}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/web/posts", strings.NewReader(`{"title":"a","body":"b"}`))
	rr := httptest.NewRecorder()
	h.WebCreatePost(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestWebUpdatePost_Success(t *testing.T) {
	chdirRepoRoot(t)
	fake := &fakePostClient{updateResult: &models.Post{ID: 3, Title: "Updated"}}
	h, err := New(fake, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/web/posts/3", strings.NewReader(`{"title":"Updated","body":"new body"}`))
	req.SetPathValue("id", "3")
	rr := httptest.NewRecorder()
	h.WebUpdatePost(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Updated") {
		t.Errorf("expected updated post in response, got: %s", rr.Body.String())
	}
}

func TestWebUpdatePost_InvalidID(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/web/posts/abc", strings.NewReader(`{"title":"a","body":"b"}`))
	req.SetPathValue("id", "abc")
	rr := httptest.NewRecorder()
	h.WebUpdatePost(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
}

func TestWebUpdatePost_MissingFields(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/web/posts/1", strings.NewReader(`{"title":"","body":""}`))
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()
	h.WebUpdatePost(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
}

func TestWebUpdatePost_RPCError(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{updateErr: apperrors.NewNotFound("Post")}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/web/posts/1", strings.NewReader(`{"title":"a","body":"b"}`))
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()
	h.WebUpdatePost(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestWebDeletePost_Success(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodDelete, "/web/posts/1", nil)
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()
	h.WebDeletePost(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
}

func TestWebDeletePost_InvalidID(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodDelete, "/web/posts/abc", nil)
	req.SetPathValue("id", "abc")
	rr := httptest.NewRecorder()
	h.WebDeletePost(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
}

func TestWebDeletePost_RPCError(t *testing.T) {
	chdirRepoRoot(t)
	h, err := New(&fakePostClient{deleteErr: apperrors.NewInternalError(nil)}, testConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodDelete, "/web/posts/1", nil)
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()
	h.WebDeletePost(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}
