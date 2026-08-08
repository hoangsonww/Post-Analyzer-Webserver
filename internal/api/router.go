package api

import (
	"net/http"
	"strings"
)

// Router handles API routing with versioning
type Router struct {
	api *API
}

// NewRouter creates a new API router
func NewRouter(api *API) *Router {
	return &Router{api: api}
}

// ServeHTTP implements http.Handler
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract API version from path
	path := r.URL.Path

	// Handle /api/v1/* routes
	if strings.HasPrefix(path, "/api/v1/") {
		router.handleV1(w, r)
		return
	}

	// Handle /api/* routes (default to v1)
	if strings.HasPrefix(path, "/api/") {
		// Remove /api prefix and add /api/v1
		r.URL.Path = "/api/v1" + strings.TrimPrefix(path, "/api")
		router.handleV1(w, r)
		return
	}

	http.NotFound(w, r)
}

// handleV1 handles version 1 API routes
func (router *Router) handleV1(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Auth endpoints (unauthenticated: this is where a token is obtained)
	if path == "/api/v1/auth/login" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		router.api.Login(w, r)
		return
	}

	// Export storage endpoints (MinIO-backed)
	if strings.HasPrefix(path, "/api/v1/exports") {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		remaining := strings.TrimPrefix(path, "/api/v1/exports")
		if remaining == "" || remaining == "/" {
			router.api.ListExports(w, r)
		} else {
			router.api.GetExport(w, r)
		}
		return
	}

	// Posts endpoints
	if strings.HasPrefix(path, "/api/v1/posts") {
		remaining := strings.TrimPrefix(path, "/api/v1/posts")

		// /api/v1/posts/bulk
		if remaining == "/bulk" {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			router.api.BulkCreatePosts(w, r)
			return
		}

		// /api/v1/posts/export
		if remaining == "/export" {
			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			router.api.ExportPosts(w, r)
			return
		}

		// /api/v1/posts/analytics
		if remaining == "/analytics" {
			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			router.api.AnalyzePosts(w, r)
			return
		}

		// /api/v1/posts/reanalyze — enqueues an async reanalysis job
		// (RabbitMQ) instead of computing it inline.
		if remaining == "/reanalyze" {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			router.api.ReanalyzePosts(w, r)
			return
		}

		// /api/v1/posts/{id}
		if remaining != "" && remaining != "/" {
			switch r.Method {
			case http.MethodGet:
				router.api.GetPost(w, r)
			case http.MethodPut:
				router.api.UpdatePost(w, r)
			case http.MethodDelete:
				router.api.DeletePost(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		// /api/v1/posts
		switch r.Method {
		case http.MethodGet:
			router.api.ListPosts(w, r)
		case http.MethodPost:
			router.api.CreatePost(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.NotFound(w, r)
}
