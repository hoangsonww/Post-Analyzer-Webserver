package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApiResponse_DecodeAndErr(t *testing.T) {
	ok := &apiResponse{status: http.StatusOK, body: []byte(`{"id":42,"title":"hi"}`)}
	var out struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}
	if err := ok.decode(&out); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if out.ID != 42 || out.Title != "hi" {
		t.Errorf("unexpected decode result: %+v", out)
	}

	empty := &apiResponse{status: http.StatusNoContent, body: nil}
	if err := empty.decode(&out); err != nil {
		t.Errorf("decode of empty body should be a no-op, got: %v", err)
	}

	withMsg := &apiResponse{status: http.StatusForbidden, body: []byte(`{"error":{"code":"forbidden","message":"nope"}}`)}
	err := withMsg.err()
	if err == nil {
		t.Fatal("expected an error")
	}
	if got := err.Error(); got != "nope (HTTP 403, forbidden)" {
		t.Errorf("unexpected error message: %q", got)
	}

	noMsg := &apiResponse{status: http.StatusInternalServerError, body: []byte(`not json`)}
	err = noMsg.err()
	if err == nil || err.Error() != "request failed with HTTP 500" {
		t.Errorf("expected generic fallback error, got: %v", err)
	}
}

func TestClient_Do_SetsAuthAndDecodesResponse(t *testing.T) {
	var sawAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["title"] != "hello" {
			t.Errorf("expected request body to round-trip, got %+v", body)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-token")
	resp, err := c.do(http.MethodPost, "/api/v1/posts", map[string]string{"title": "hello"}, nil)
	if err != nil {
		t.Fatalf("do failed: %v", err)
	}
	if resp.status != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.status)
	}
	if sawAuth != "Bearer test-token" {
		t.Errorf("expected bearer token to be set, got %q", sawAuth)
	}

	var out struct {
		ID int `json:"id"`
	}
	if err := resp.decode(&out); err != nil || out.ID != 1 {
		t.Errorf("unexpected decoded response: id=%d err=%v", out.ID, err)
	}
}

func TestClient_Do_NoTokenNoAuthHeader(t *testing.T) {
	var sawAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient(server.URL, "")
	if _, err := c.do(http.MethodGet, "/api/v1/posts", nil, nil); err != nil {
		t.Fatalf("do failed: %v", err)
	}
	if sawAuth != "" {
		t.Errorf("expected no Authorization header when token is empty, got %q", sawAuth)
	}
}
