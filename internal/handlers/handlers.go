package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"Post_Analyzer_Webserver/config"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/metrics"
	"Post_Analyzer_Webserver/internal/storage"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	storage  storage.Storage
	config   *config.Config
	template *template.Template
}

// New creates a new Handler
func New(store storage.Storage, cfg *config.Config) (*Handler, error) {
	// Custom template functions
	funcMap := template.FuncMap{
		"toJSON": func(v interface{}) string {
			data, _ := json.Marshal(v)
			return string(data)
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFiles("home.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &Handler{
		storage:  store,
		config:   cfg,
		template: tmpl,
	}, nil
}

// Template variables
type HomePageVars struct {
	Title       string
	Posts       []storage.Post
	CharFreq    map[rune]int
	Error       string
	HasPosts    bool
	HasAnalysis bool
}

// Health check endpoint
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// Readiness check endpoint
func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check if storage is accessible
	_, err := h.storage.Count(ctx)
	if err != nil {
		logger.ErrorContext(r.Context(), "readiness check failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"not ready","error":"%s"}`, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ready","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// Home serves the home page
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	posts, err := h.storage.GetAll(r.Context())
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to get posts", "error", err)
		h.renderTemplate(w, HomePageVars{Title: "Home", Error: "Failed to read posts"})
		return
	}

	h.renderTemplate(w, HomePageVars{Title: "Home", Posts: posts, HasPosts: len(posts) > 0})
}

// FetchPosts fetches posts from external API and stores them
func (h *Handler) FetchPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := h.fetchPostsFromAPI(r.Context())
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to fetch posts from API", "error", err)
		h.renderTemplate(w, HomePageVars{Title: "Error", Error: "Failed to fetch posts from external API"})
		return
	}

	if err := h.storage.BatchCreate(r.Context(), posts); err != nil {
		logger.ErrorContext(r.Context(), "failed to store posts", "error", err)
		h.renderTemplate(w, HomePageVars{Title: "Error", Error: "Failed to store posts"})
		return
	}

	logger.InfoContext(r.Context(), "posts fetched successfully", "count", len(posts))
	h.renderTemplate(w, HomePageVars{Title: "Posts Fetched", Posts: posts, HasPosts: true})
}

// AnalyzePosts performs character frequency analysis
func (h *Handler) AnalyzePosts(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	posts, err := h.storage.GetAll(r.Context())
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

// AddPost adds a new post
func (h *Handler) AddPost(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Parse form data
		if err := r.ParseForm(); err != nil {
			logger.ErrorContext(r.Context(), "failed to parse form", "error", err)
			h.renderTemplate(w, HomePageVars{Title: "Error", Error: "Failed to parse form data"})
			return
		}

		// Sanitize and validate input
		title := h.sanitizeInput(r.FormValue("title"))
		body := h.sanitizeInput(r.FormValue("body"))

		if title == "" || body == "" {
			h.renderTemplate(w, HomePageVars{Title: "Error", Error: "Title and body are required"})
			return
		}

		post := &storage.Post{
			UserId: 1,
			Title:  title,
			Body:   body,
		}

		if err := h.storage.Create(r.Context(), post); err != nil {
			logger.ErrorContext(r.Context(), "failed to create post", "error", err)
			h.renderTemplate(w, HomePageVars{Title: "Error", Error: "Failed to create post"})
			return
		}

		// Get all posts to display
		posts, _ := h.storage.GetAll(r.Context())
		logger.InfoContext(r.Context(), "post added successfully", "id", post.Id)
		h.renderTemplate(w, HomePageVars{Title: "Post Added", Posts: posts, HasPosts: true})
	} else {
		h.renderTemplate(w, HomePageVars{Title: "Add New Post"})
	}
}

// fetchPostsFromAPI fetches posts from external API
func (h *Handler) fetchPostsFromAPI(ctx context.Context) ([]storage.Post, error) {
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var posts []storage.Post
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

// Close closes handler resources
func (h *Handler) Close() error {
	return h.storage.Close()
}
