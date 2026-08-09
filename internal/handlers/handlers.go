package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"Post_Analyzer_Webserver/config"
	apperrors "Post_Analyzer_Webserver/internal/errors"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/metrics"
	"Post_Analyzer_Webserver/internal/models"
	"Post_Analyzer_Webserver/internal/rpcclient"
)

// Handler holds dependencies for HTTP handlers. It talks to postsvc over
// RPC (via rpcclient.PostClient) rather than touching storage directly —
// the gateway process owns no database connection.
type Handler struct {
	posts    rpcclient.PostClient
	config   *config.Config
	template *template.Template
}

// New creates a new Handler
func New(posts rpcclient.PostClient, cfg *config.Config) (*Handler, error) {
	funcMap := template.FuncMap{
		// toJSON returns template.JS rather than string: html/template
		// contextually auto-escapes a plain string used inside a
		// <script> block by wrapping it in quotes (turning it into a
		// JS string literal, not the object/array literal we actually
		// want). template.JS marks the content as already-safe raw JS
		// so it's emitted unquoted.
		"toJSON": func(v interface{}) template.JS {
			data, _ := json.Marshal(v)
			return template.JS(data)
		},
		"fmtDate": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("Jan 2, 2006 3:04 PM")
		},
		"truncate": func(s string, n int) string {
			r := []rune(s)
			if len(r) <= n {
				return s
			}
			return string(r[:n]) + "…"
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFiles("home.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &Handler{
		posts:    posts,
		config:   cfg,
		template: tmpl,
	}, nil
}

// Template variables
type HomePageVars struct {
	Title       string
	Posts       []models.Post
	CharFreq    map[rune]int
	Error       string
	HasPosts    bool
	HasAnalysis bool
}

// Health check endpoint
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// Readiness check endpoint — verifies postsvc RPC is reachable
func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, _, err := h.posts.GetAll(ctx, nil, &models.PaginationParams{Page: 1, PageSize: 1}); err != nil {
		logger.ErrorContext(r.Context(), "readiness check failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprintf(w, `{"status":"not ready","error":"%s"}`, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `{"status":"ready","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// Home serves the home page
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	posts, _, err := h.posts.GetAll(r.Context(), nil, nil)
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to get posts", "error", err)
		h.renderTemplate(w, HomePageVars{Title: "Home", Error: "Failed to read posts"})
		return
	}

	h.renderTemplate(w, HomePageVars{Title: "Home", Posts: posts, HasPosts: len(posts) > 0})
}

// FetchPosts fetches posts from external API and stores them via postsvc
func (h *Handler) FetchPosts(w http.ResponseWriter, r *http.Request) {
	fetched, err := h.fetchPostsFromAPI(r.Context())
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to fetch posts from API", "error", err)
		h.renderTemplate(w, HomePageVars{Title: "Error", Error: "Failed to fetch posts from external API"})
		return
	}

	bulkReq := &models.BulkCreateRequest{Posts: make([]models.CreatePostRequest, len(fetched))}
	for i, p := range fetched {
		bulkReq.Posts[i] = models.CreatePostRequest{UserID: p.UserID, Title: p.Title, Body: p.Body}
	}
	if _, err := h.posts.BulkCreate(r.Context(), bulkReq); err != nil {
		logger.ErrorContext(r.Context(), "failed to store posts", "error", err)
		h.renderTemplate(w, HomePageVars{Title: "Error", Error: "Failed to store posts"})
		return
	}

	posts, _, _ := h.posts.GetAll(r.Context(), nil, nil)
	logger.InfoContext(r.Context(), "posts fetched successfully", "count", len(fetched))
	h.renderTemplate(w, HomePageVars{Title: "Posts Fetched", Posts: posts, HasPosts: true})
}

// AnalyzePosts performs character frequency analysis over post text
func (h *Handler) AnalyzePosts(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	posts, _, err := h.posts.GetAll(r.Context(), nil, nil)
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to get posts for analysis", "error", err)
		h.renderTemplate(w, HomePageVars{Title: "Error", Error: "Failed to read posts for analysis"})
		return
	}

	// Combine all post text
	var allText strings.Builder
	for _, post := range posts {
		allText.WriteString(post.Title)
		allText.WriteString(" ")
		allText.WriteString(post.Body)
		allText.WriteString(" ")
	}

	charFreq := h.countCharacters(allText.String())

	metrics.RecordAnalysisOperation(time.Since(start))
	logger.InfoContext(r.Context(), "character analysis completed", "duration_ms", time.Since(start).Milliseconds())

	h.renderTemplate(w, HomePageVars{Title: "Character Analysis", CharFreq: charFreq, HasAnalysis: true})
}

// AddPost adds a new post via postsvc
func (h *Handler) AddPost(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			logger.ErrorContext(r.Context(), "failed to parse form", "error", err)
			h.renderTemplate(w, HomePageVars{Title: "Error", Error: "Failed to parse form data"})
			return
		}

		title := h.sanitizeInput(r.FormValue("title"))
		body := h.sanitizeInput(r.FormValue("body"))

		if title == "" || body == "" {
			h.renderTemplate(w, HomePageVars{Title: "Error", Error: "Title and body are required"})
			return
		}

		created, err := h.posts.Create(r.Context(), &models.CreatePostRequest{UserID: 1, Title: title, Body: body})
		if err != nil {
			logger.ErrorContext(r.Context(), "failed to create post", "error", err)
			h.renderTemplate(w, HomePageVars{Title: "Error", Error: "Failed to create post"})
			return
		}

		posts, _, _ := h.posts.GetAll(r.Context(), nil, nil)
		logger.InfoContext(r.Context(), "post added successfully", "id", created.ID)
		h.renderTemplate(w, HomePageVars{Title: "Post Added", Posts: posts, HasPosts: true})
	} else {
		h.renderTemplate(w, HomePageVars{Title: "Add New Post"})
	}
}

// --- Web JSON API ---------------------------------------------------
//
// The routes below back the interactive parts of the built-in web UI
// (search, sort, pagination, edit, delete) with small JSON endpoints
// under /web/, separate from the hardened, ABAC-gated /api/v1 surface
// used by external clients. They're same-origin, same-trust-level as
// the pre-existing /add form post — convenience endpoints for the
// bundled UI, not a public API contract.

// webPostInput is the JSON body accepted by WebCreatePost/WebUpdatePost.
type webPostInput struct {
	UserID int    `json:"userId"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

func (h *Handler) webJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logger.Error("web: failed to encode JSON response", "error", err)
	}
}

// webErrorFrom maps an error to a JSON {"error":{"code","message"}} body
// — the same shape and the same *errors.AppError status-code convention
// internal/api's respondError uses, so a "post not found" surfaces as
// 404 with a specific message rather than a blanket 500, and every
// client (home.html, the dashboard, the CLI) can parse errors from any
// backend surface the exact same way.
func (h *Handler) webErrorFrom(w http.ResponseWriter, err error) {
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		appErr = apperrors.NewInternalError(err)
	}
	h.webJSON(w, appErr.StatusCode, map[string]interface{}{
		"error": map[string]interface{}{"code": appErr.Code, "message": appErr.Message},
	})
}

// WebListPosts serves the post grid as JSON. Search, sort, and
// pagination are all pushed down to postsvc rather than done in the
// browser, so the UI stays fast no matter how many posts exist.
func (h *Handler) WebListPosts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))

	filter := &models.PostFilter{
		Search:    strings.TrimSpace(q.Get("search")),
		SortBy:    q.Get("sortBy"),
		SortOrder: q.Get("sortOrder"),
	}
	posts, meta, err := h.posts.GetAll(r.Context(), filter, &models.PaginationParams{Page: page, PageSize: pageSize})
	if err != nil {
		logger.ErrorContext(r.Context(), "web: failed to list posts", "error", err)
		h.webErrorFrom(w, err)
		return
	}
	if posts == nil {
		posts = []models.Post{}
	}
	h.webJSON(w, http.StatusOK, map[string]interface{}{"posts": posts, "pagination": meta})
}

// WebGetPost returns a single post as JSON, used to prefill the edit modal.
func (h *Handler) WebGetPost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		h.webErrorFrom(w, apperrors.NewValidationError("invalid post id"))
		return
	}
	post, err := h.posts.GetByID(r.Context(), id)
	if err != nil {
		h.webErrorFrom(w, err)
		return
	}
	h.webJSON(w, http.StatusOK, post)
}

// WebCreatePost creates a post from a JSON body (the "New post" modal).
func (h *Handler) WebCreatePost(w http.ResponseWriter, r *http.Request) {
	var in webPostInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		h.webErrorFrom(w, apperrors.NewValidationError("invalid request body"))
		return
	}
	title := h.sanitizeInput(in.Title)
	body := h.sanitizeInput(in.Body)
	if title == "" || body == "" {
		h.webErrorFrom(w, apperrors.NewValidationError("title and body are required"))
		return
	}
	userID := in.UserID
	if userID == 0 {
		userID = 1
	}

	created, err := h.posts.Create(r.Context(), &models.CreatePostRequest{UserID: userID, Title: title, Body: body})
	if err != nil {
		logger.ErrorContext(r.Context(), "web: failed to create post", "error", err)
		h.webErrorFrom(w, err)
		return
	}
	logger.InfoContext(r.Context(), "web: post created", "id", created.ID)
	h.webJSON(w, http.StatusCreated, created)
}

// WebUpdatePost updates a post's title/body from a JSON body (edit modal).
func (h *Handler) WebUpdatePost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		h.webErrorFrom(w, apperrors.NewValidationError("invalid post id"))
		return
	}
	var in webPostInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		h.webErrorFrom(w, apperrors.NewValidationError("invalid request body"))
		return
	}
	title := h.sanitizeInput(in.Title)
	body := h.sanitizeInput(in.Body)
	if title == "" || body == "" {
		h.webErrorFrom(w, apperrors.NewValidationError("title and body are required"))
		return
	}

	updated, err := h.posts.Update(r.Context(), id, &models.UpdatePostRequest{Title: title, Body: body})
	if err != nil {
		logger.ErrorContext(r.Context(), "web: failed to update post", "error", err, "id", id)
		h.webErrorFrom(w, err)
		return
	}
	logger.InfoContext(r.Context(), "web: post updated", "id", id)
	h.webJSON(w, http.StatusOK, updated)
}

// WebDeletePost deletes a post.
func (h *Handler) WebDeletePost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		h.webErrorFrom(w, apperrors.NewValidationError("invalid post id"))
		return
	}
	if err := h.posts.Delete(r.Context(), id); err != nil {
		logger.ErrorContext(r.Context(), "web: failed to delete post", "error", err, "id", id)
		h.webErrorFrom(w, err)
		return
	}
	logger.InfoContext(r.Context(), "web: post deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

// fetchPostsFromAPI fetches posts from external API
func (h *Handler) fetchPostsFromAPI(ctx context.Context) ([]models.Post, error) {
	client := &http.Client{
		Timeout: h.config.External.HTTPTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", h.config.External.JSONPlaceholderURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var posts []models.Post
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, err
	}

	return posts, nil
}

// countCharacters counts character frequency efficiently
func (h *Handler) countCharacters(text string) map[rune]int {
	charCount := make(map[rune]int)

	// Process in chunks for better performance with large texts
	const chunkSize = 1000
	if len(text) <= chunkSize {
		// Small text, process directly
		for _, char := range text {
			charCount[char]++
		}
		return charCount
	}

	// Large text, use concurrent processing
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}

	numWorkers := 4
	chunkLen := (len(text) + numWorkers - 1) / numWorkers

	for i := 0; i < numWorkers; i++ {
		start := i * chunkLen
		end := start + chunkLen
		if end > len(text) {
			end = len(text)
		}
		if start >= len(text) {
			break
		}

		wg.Add(1)
		go func(chunk string) {
			defer wg.Done()
			localCount := make(map[rune]int)
			for _, char := range chunk {
				localCount[char]++
			}

			mu.Lock()
			for char, count := range localCount {
				charCount[char] += count
			}
			mu.Unlock()
		}(text[start:end])
	}

	wg.Wait()
	return charCount
}

// sanitizeInput sanitizes user input to prevent XSS
func (h *Handler) sanitizeInput(input string) string {
	// Remove any HTML tags
	input = html.EscapeString(input)

	// Remove any potential script tags or event handlers
	input = regexp.MustCompile(`(?i)<script.*?>.*?</script>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)on\w+\s*=`).ReplaceAllString(input, "")

	// Trim whitespace
	input = strings.TrimSpace(input)

	return input
}

// renderTemplate renders the HTML template
func (h *Handler) renderTemplate(w http.ResponseWriter, vars HomePageVars) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.template.ExecuteTemplate(w, "home.html", vars); err != nil {
		logger.Error("failed to render template", "error", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

// Close releases handler resources. The RPC client owns no closable
// resource itself (Kitex manages connections internally), so this is a
// no-op kept for symmetry with the old storage-owning Handler.
func (h *Handler) Close() error {
	return nil
}
