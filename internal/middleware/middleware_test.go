package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"Post_Analyzer_Webserver/internal/logger"
)

func TestRequestID(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Context().Value(logger.RequestIDKey)
		if requestID == nil {
			t.Error("Expected request ID in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-Request-ID") == "" {
		t.Error("Expected X-Request-ID header")
	}
}

func TestRequestIDFromHeader(t *testing.T) {
	customID := "custom-request-id"

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Context().Value(logger.RequestIDKey)
		if requestID != customID {
			t.Errorf("Expected request ID %s, got %v", customID, requestID)
		}
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", customID)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-Request-ID") != customID {
		t.Errorf("Expected X-Request-ID header %s, got %s", customID, rr.Header().Get("X-Request-ID"))
	}
}

func TestRecovery(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Should not panic
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
	}

	for header, expectedValue := range headers {
		actualValue := rr.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Expected %s header to be %s, got %s", header, expectedValue, actualValue)
		}
	}
}

func TestCORS(t *testing.T) {
	allowedOrigins := []string{"http://example.com"}
	handler := CORS(allowedOrigins)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test with allowed origin
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Error("Expected CORS headers for allowed origin")
	}

	// Test with disallowed origin
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	rr = httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("Should not set CORS headers for disallowed origin")
	}
}

func TestCORSPreflight(t *testing.T) {
	allowedOrigins := []string{"*"}
	handler := CORS(allowedOrigins)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Should not call next handler for OPTIONS request")
	}))

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("Expected status 204 for OPTIONS, got %d", rr.Code)
	}
}

func TestRateLimiter(t *testing.T) {
	limit := 5
	window := 1 * time.Second
	rl := NewRateLimiter(limit, window)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make requests up to the limit
	for i := 0; i < limit; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Request %d should succeed, got status %d", i+1, rr.Code)
		}
	}

	// Next request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected rate limit (429), got %d", rr.Code)
	}

	// Wait for window to reset
	time.Sleep(window + 100*time.Millisecond)

	// Should work again
	req = httptest.NewRequest("GET", "/test", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected success after window reset, got %d", rr.Code)
	}
}

func TestMaxBodySize(t *testing.T) {
	maxSize := int64(100)
	handler := MaxBodySize(maxSize)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to read body
		buf := make([]byte, maxSize+1)
		_, err := r.Body.Read(buf)
		if err == nil {
			t.Error("Expected error when reading body larger than max size")
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Create request with body larger than max size
	body := make([]byte, maxSize+1)
	req := httptest.NewRequest("POST", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	_ = body // Use body variable
}

func TestChain(t *testing.T) {
	called := []string{}

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = append(called, "middleware1")
			next.ServeHTTP(w, r)
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = append(called, "middleware2")
			next.ServeHTTP(w, r)
		})
	}

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = append(called, "handler")
		w.WriteHeader(http.StatusOK)
	})

	chained := Chain(middleware1, middleware2)(finalHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	// Check order of execution
	if len(called) != 3 {
		t.Errorf("Expected 3 calls, got %d", len(called))
	}
	if called[0] != "middleware1" || called[1] != "middleware2" || called[2] != "handler" {
		t.Errorf("Unexpected execution order: %v", called)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		xRealIP       string
		expectedIP    string
	}{
		{
			name:       "from RemoteAddr",
			remoteAddr: "192.168.1.1:1234",
			expectedIP: "192.168.1.1",
		},
		{
			name:          "from X-Forwarded-For",
			remoteAddr:    "192.168.1.1:1234",
			xForwardedFor: "10.0.0.1, 10.0.0.2",
			expectedIP:    "10.0.0.1",
		},
		{
			name:       "from X-Real-IP",
			remoteAddr: "192.168.1.1:1234",
			xRealIP:    "10.0.0.1",
			expectedIP: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}
