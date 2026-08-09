package middleware

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"strconv"
	"strings"
)

// Compression gzip-compresses JSON API responses when the client asks
// for them (Accept-Encoding: gzip). It skips /metrics and /health:
// those are small, scraped/probed by tools that don't need compression.
//
// It buffers the full response before writing anything, rather than
// streaming through a gzip.Writer wrapped around the live
// http.ResponseWriter. This whole handler chain runs behind Hertz's
// net/http compatibility adaptor (see internal/cli.runServe), and the
// streaming version — write compressed bytes as the handler produces
// them, flush the gzip trailer via a deferred Close() — produced
// responses that curl and Go's own net/http client reported as
// truncated ("unexpected EOF" decompressing) for short responses in
// particular; only a browser's more tolerant network stack papered
// over it. Buffering and sending one Content-Length-declared write
// sidesteps that framing interaction entirely. The tradeoff (the full
// response sits in memory before any bytes go out) is a non-issue for
// a bounded JSON API response.
func Compression(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		rec := &bufferedResponseWriter{header: make(http.Header)}
		next.ServeHTTP(rec, r)
		writeCompressed(w, rec)
	})
}

// writeCompressed gzips rec's buffered body and writes it to w as a
// single response. If compression itself fails (never expected for an
// in-memory gzip.Writer, but checked rather than assumed), it falls
// back to sending the original uncompressed body instead of a broken one.
func writeCompressed(w http.ResponseWriter, rec *bufferedResponseWriter) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, writeErr := gz.Write(rec.body.Bytes())
	closeErr := gz.Close()

	header := w.Header()
	for k, vv := range rec.header {
		if k == "Content-Length" {
			continue
		}
		for _, v := range vv {
			header.Add(k, v)
		}
	}

	if writeErr != nil || closeErr != nil {
		header.Set("Content-Length", strconv.Itoa(rec.body.Len()))
		w.WriteHeader(rec.statusCode())
		_, _ = w.Write(rec.body.Bytes())
		return
	}

	header.Set("Content-Encoding", "gzip")
	header.Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(rec.statusCode())
	_, _ = w.Write(buf.Bytes())
}

// bufferedResponseWriter collects a handler's response in memory
// instead of writing it straight through, so Compression can gzip the
// complete body in one shot.
type bufferedResponseWriter struct {
	header http.Header
	status int
	wrote  bool
	body   bytes.Buffer
}

func (b *bufferedResponseWriter) Header() http.Header { return b.header }

func (b *bufferedResponseWriter) WriteHeader(status int) {
	if !b.wrote {
		b.status = status
		b.wrote = true
	}
}

func (b *bufferedResponseWriter) Write(p []byte) (int, error) {
	if !b.wrote {
		b.WriteHeader(http.StatusOK)
	}
	return b.body.Write(p)
}

func (b *bufferedResponseWriter) statusCode() int {
	if !b.wrote {
		return http.StatusOK
	}
	return b.status
}
