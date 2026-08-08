package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Post_Analyzer_Webserver/internal/models"
)

func newTestRouter() *Router {
	return NewRouter(newTestAPI(newFakePostClient(), &fakeAuthClient{}))
}

func TestRouter_DispatchesKnownRoutes(t *testing.T) {
	cases := []struct {
		name       string
		method     string
		path       string
		wantStatus int // exact status the underlying handler is expected to produce
	}{
		{"login", http.MethodPost, "/api/v1/auth/login", http.StatusUnprocessableEntity}, // empty body -> validation error, but proves Login was reached
		{"list posts", http.MethodGet, "/api/v1/posts", http.StatusOK},
		{"posts bulk", http.MethodPost, "/api/v1/posts/bulk", http.StatusUnprocessableEntity}, // empty body decodes to Posts:nil -> validation error, proves BulkCreatePosts was reached
		{"posts export", http.MethodGet, "/api/v1/posts/export", http.StatusOK},
		{"posts analytics", http.MethodGet, "/api/v1/posts/analytics", http.StatusOK},
		{"posts reanalyze", http.MethodPost, "/api/v1/posts/reanalyze", http.StatusServiceUnavailable}, // rabbitmq nil, proves ReanalyzePosts was reached
		{"get post by id", http.MethodGet, "/api/v1/posts/999", http.StatusNotFound},
		{"ml sentiment", http.MethodPost, "/api/v1/ml/sentiment", http.StatusServiceUnavailable}, // triton nil, proves ClassifySentiment was reached
		{"list exports", http.MethodGet, "/api/v1/exports", http.StatusServiceUnavailable},       // objects nil, proves ListExports was reached
		{"get export by key", http.MethodGet, "/api/v1/exports/somekey", http.StatusServiceUnavailable},
		{"unknown path", http.MethodGet, "/api/v1/unknown", http.StatusNotFound},
	}

	router := newTestRouter()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("%s %s: expected status %d, got %d: %s", tc.method, tc.path, tc.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestRouter_MethodNotAllowed(t *testing.T) {
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/auth/login"},
		{http.MethodDelete, "/api/v1/posts/bulk"},
		{http.MethodPost, "/api/v1/posts/export"},
		{http.MethodPost, "/api/v1/posts/analytics"},
		{http.MethodGet, "/api/v1/posts/reanalyze"},
		{http.MethodPatch, "/api/v1/posts/1"},
		{http.MethodPut, "/api/v1/posts"},
		{http.MethodGet, "/api/v1/ml/sentiment"},
		{http.MethodPost, "/api/v1/exports"},
	}

	router := newTestRouter()
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: expected 405, got %d", tc.method, tc.path, rr.Code)
		}
	}
}

func TestRouter_CreatePostViaPOST(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/posts", strings.NewReader(`{"userId":1,"title":"T","body":"B"}`))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRouter_UpdateAndDeletePostByID(t *testing.T) {
	post := newFakePostClient()
	post.posts[1] = models.Post{ID: 1, Title: "Original"}
	router := NewRouter(newTestAPI(post, &fakeAuthClient{}))

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/posts/1", strings.NewReader(`{"title":"Updated"}`))
	updateRR := httptest.NewRecorder()
	router.ServeHTTP(updateRR, updateReq)
	if updateRR.Code != http.StatusOK {
		t.Fatalf("PUT: expected 200, got %d: %s", updateRR.Code, updateRR.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/posts/1", nil)
	deleteRR := httptest.NewRecorder()
	router.ServeHTTP(deleteRR, deleteReq)
	if deleteRR.Code != http.StatusOK {
		t.Fatalf("DELETE: expected 200, got %d: %s", deleteRR.Code, deleteRR.Body.String())
	}
	if _, ok := post.posts[1]; ok {
		t.Error("expected the post to be gone after DELETE routed through the router")
	}
}

func TestRouter_APIPrefixWithoutVersionRewritesToV1(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected /api/posts to be rewritten to /api/v1/posts and dispatched, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRouter_UnversionedNonAPIPathIs404(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for a non-/api path, got %d", rr.Code)
	}
}
