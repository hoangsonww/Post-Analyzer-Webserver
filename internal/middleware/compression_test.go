package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompression_CompressesWhenClientAcceptsGzip(t *testing.T) {
	body := strings.Repeat(`{"hello":"world"} `, 50)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(body))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	Compression(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected the wrapped handler's status to survive, got %d", rr.Code)
	}
	if rr.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected Content-Encoding: gzip, got %q", rr.Header().Get("Content-Encoding"))
	}
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected the wrapped handler's headers to survive, got %q", rr.Header().Get("Content-Type"))
	}

	// The whole point of the fix: what's on the wire must be a
	// complete, valid gzip stream (magic bytes through the trailing
	// CRC32/ISIZE footer) that decompresses to exactly the original
	// body — not truncated, not double-encoded.
	gz, err := gzip.NewReader(rr.Body)
	if err != nil {
		t.Fatalf("response body is not valid gzip: %v", err)
	}
	decompressed, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("failed to fully decompress response body (truncated stream): %v", err)
	}
	if string(decompressed) != body {
		t.Errorf("decompressed body doesn't match original:\ngot:  %q\nwant: %q", decompressed, body)
	}

	if cl := rr.Header().Get("Content-Length"); cl == "" {
		t.Error("expected an explicit Content-Length on the compressed response")
	}
}

func TestCompression_SkipsWhenClientDoesNotAcceptGzip(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("plain"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil)
	rr := httptest.NewRecorder()
	Compression(next).ServeHTTP(rr, req)

	if rr.Header().Get("Content-Encoding") == "gzip" {
		t.Error("expected no compression when the client didn't send Accept-Encoding: gzip")
	}
	if rr.Body.String() != "plain" {
		t.Errorf("expected the uncompressed body to pass through unchanged, got %q", rr.Body.String())
	}
}

func TestCompression_SkipsMetricsAndHealthEvenWithGzipAccepted(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("uncompressed"))
	})

	for _, path := range []string{"/metrics", "/health"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Accept-Encoding", "gzip")
		rr := httptest.NewRecorder()
		Compression(next).ServeHTTP(rr, req)

		if rr.Header().Get("Content-Encoding") == "gzip" {
			t.Errorf("%s: expected compression to be skipped, got Content-Encoding: gzip", path)
		}
		if rr.Body.String() != "uncompressed" {
			t.Errorf("%s: expected the body to pass through unchanged, got %q", path, rr.Body.String())
		}
	}
}

func TestCompression_DefaultsStatusToOKWhenWriteHeaderNeverCalled(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("no explicit status"))
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	Compression(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected default status 200, got %d", rr.Code)
	}
}

func TestCompression_EmptyBodyProducesValidGzipStream(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/posts/1", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	Compression(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
	gz, err := gzip.NewReader(bytes.NewReader(rr.Body.Bytes()))
	if err != nil {
		t.Fatalf("expected a valid (if empty) gzip stream even for an empty body: %v", err)
	}
	if _, err := io.ReadAll(gz); err != nil {
		t.Fatalf("failed to read empty gzip stream: %v", err)
	}
}

func TestBufferedResponseWriter_HeaderIsUsableBeforeWrite(t *testing.T) {
	b := &bufferedResponseWriter{header: make(http.Header)}
	b.Header().Set("X-Test", "1")
	if b.Header().Get("X-Test") != "1" {
		t.Error("expected Header() to return a usable, mutable header map")
	}
}
