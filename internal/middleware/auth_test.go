package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"Post_Analyzer_Webserver/internal/abac"
)

// fakeAuthClient implements rpcclient.AuthClient without any RPC — lets
// the ABAC middleware be tested in isolation from authsvc.
type fakeAuthClient struct {
	validToken  string
	subject     abac.Subject
	policies    []abac.Policy
	validateErr error
}

func (f *fakeAuthClient) Login(ctx context.Context, username, password string) (string, abac.Subject, error) {
	return f.validToken, f.subject, nil
}

func (f *fakeAuthClient) ValidateToken(ctx context.Context, token string) (abac.Subject, error) {
	if f.validateErr != nil {
		return abac.Subject{}, f.validateErr
	}
	if token != f.validToken {
		return abac.Subject{}, errInvalidToken
	}
	return f.subject, nil
}

func (f *fakeAuthClient) Authorize(ctx context.Context, req abac.Request) (abac.Decision, error) {
	return abac.Evaluate(req, f.policies), nil
}

var errInvalidToken = &testError{"invalid token"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

func newTestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestABAC_MissingToken(t *testing.T) {
	client := &fakeAuthClient{validToken: "good-token", subject: abac.Subject{Role: "admin"}, policies: abac.DefaultPolicies()}
	handler := ABAC(client, "post", ActionByMethod)(newTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing token, got %d", rr.Code)
	}
}

func TestABAC_InvalidToken(t *testing.T) {
	client := &fakeAuthClient{validToken: "good-token", subject: abac.Subject{Role: "admin"}, policies: abac.DefaultPolicies()}
	handler := ABAC(client, "post", ActionByMethod)(newTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid token, got %d", rr.Code)
	}
}

func TestABAC_ValidTokenInsufficientRole(t *testing.T) {
	client := &fakeAuthClient{
		validToken: "viewer-token",
		subject:    abac.Subject{Role: "viewer", Username: "viewer"},
		policies:   abac.DefaultPolicies(),
	}
	handler := ABAC(client, "post", ActionByMethod)(newTestHandler())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/posts", nil)
	req.Header.Set("Authorization", "Bearer viewer-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for viewer write, got %d", rr.Code)
	}
}

func TestABAC_ValidTokenAllowed(t *testing.T) {
	client := &fakeAuthClient{
		validToken: "admin-token",
		subject:    abac.Subject{Role: "admin", Username: "admin"},
		policies:   abac.DefaultPolicies(),
	}

	var sawSubject abac.Subject
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawSubject, _ = SubjectFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	handler := ABAC(client, "post", ActionByMethod)(inner)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/posts", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin write, got %d", rr.Code)
	}
	if sawSubject.Username != "admin" {
		t.Errorf("expected the authenticated subject to be injected into the request context, got %+v", sawSubject)
	}
}

func TestActionByMethod(t *testing.T) {
	cases := map[string]string{
		http.MethodGet:    "read",
		http.MethodPost:   "write",
		http.MethodPut:    "write",
		http.MethodPatch:  "write",
		http.MethodDelete: "delete",
	}
	for method, want := range cases {
		req := httptest.NewRequest(method, "/", nil)
		if got := ActionByMethod(req); got != want {
			t.Errorf("ActionByMethod(%s) = %q, want %q", method, got, want)
		}
	}
}

func TestExtractBearerToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer abc123")
	if got := extractBearerToken(req); got != "abc123" {
		t.Errorf("expected abc123, got %q", got)
	}

	reqNoHeader := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := extractBearerToken(reqNoHeader); got != "" {
		t.Errorf("expected empty string when no Authorization header, got %q", got)
	}

	reqWrongScheme := httptest.NewRequest(http.MethodGet, "/", nil)
	reqWrongScheme.Header.Set("Authorization", "Basic abc123")
	if got := extractBearerToken(reqWrongScheme); got != "" {
		t.Errorf("expected empty string for non-Bearer scheme, got %q", got)
	}
}
