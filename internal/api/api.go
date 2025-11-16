package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"Post_Analyzer_Webserver/internal/errors"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/models"
	"Post_Analyzer_Webserver/internal/service"
)

// API handles REST API endpoints
type API struct {
	postService *service.PostService
}

// NewAPI creates a new API handler
func NewAPI(postService *service.PostService) *API {
	return &API{
		postService: postService,
	}
}

// ListPosts handles GET /api/v1/posts
func (a *API) ListPosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse filters
	filter := &models.PostFilter{}
	if userIDStr := r.URL.Query().Get("userId"); userIDStr != "" {
		userID, err := strconv.Atoi(userIDStr)
		if err == nil {
			filter.UserID = &userID
		}
	}
	filter.Search = r.URL.Query().Get("search")
	filter.SortBy = r.URL.Query().Get("sortBy")
	filter.SortOrder = r.URL.Query().Get("sortOrder")

	// Parse pagination
	pagination := a.parsePagination(r)

	// Get posts
	posts, paginationMeta, err := a.postService.GetAll(ctx, filter, pagination)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	// Build response
	response := &models.PaginatedResponse{
		Data:       posts,
		Pagination: *paginationMeta,
		Meta: &models.ResponseMeta{
			RequestID: getRequestID(ctx),
			Timestamp: time.Now(),
		},
	}

	a.respondJSON(w, http.StatusOK, response)
}

// GetPost handles GET /api/v1/posts/{id}
func (a *API) GetPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract ID from URL path
	id, err := a.extractID(r)
	if err != nil {
		a.respondError(w, r, errors.NewValidationError("invalid post ID"))
		return
	}

	// Get post
	post, err := a.postService.GetByID(ctx, id)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	a.respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": post,
		"meta": &models.ResponseMeta{
			RequestID: getRequestID(ctx),
			Timestamp: time.Now(),
		},
	})
}

// CreatePost handles POST /api/v1/posts
func (a *API) CreatePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req models.CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.respondError(w, r, errors.NewValidationError("invalid request body"))
		return
	}

	// Create post
	post, err := a.postService.Create(ctx, &req)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	logger.InfoContext(ctx, "post created via API", "id", post.ID)

	a.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"data": post,
		"meta": &models.ResponseMeta{
			RequestID: getRequestID(ctx),
			Timestamp: time.Now(),
		},
	})
}

// UpdatePost handles PUT /api/v1/posts/{id}
func (a *API) UpdatePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract ID
	id, err := a.extractID(r)
	if err != nil {
		a.respondError(w, r, errors.NewValidationError("invalid post ID"))
		return
	}

	// Parse request body
	var req models.UpdatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.respondError(w, r, errors.NewValidationError("invalid request body"))
		return
	}

	// Update post
	post, err := a.postService.Update(ctx, id, &req)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	logger.InfoContext(ctx, "post updated via API", "id", post.ID)

	a.respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": post,
		"meta": &models.ResponseMeta{
			RequestID: getRequestID(ctx),
			Timestamp: time.Now(),
		},
	})
}

// DeletePost handles DELETE /api/v1/posts/{id}
func (a *API) DeletePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract ID
	id, err := a.extractID(r)
	if err != nil {
		a.respondError(w, r, errors.NewValidationError("invalid post ID"))
		return
	}

	// Delete post
	if err := a.postService.Delete(ctx, id); err != nil {
		a.respondError(w, r, err)
		return
	}

	logger.InfoContext(ctx, "post deleted via API", "id", id)

	a.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Post deleted successfully",
		"meta": &models.ResponseMeta{
			RequestID: getRequestID(ctx),
			Timestamp: time.Now(),
		},
	})
}

// BulkCreatePosts handles POST /api/v1/posts/bulk
func (a *API) BulkCreatePosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req models.BulkCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.respondError(w, r, errors.NewValidationError("invalid request body"))
		return
	}

	// Validate request
	if len(req.Posts) == 0 {
		a.respondError(w, r, errors.NewValidationError("posts array cannot be empty"))
		return
	}
	if len(req.Posts) > 1000 {
		a.respondError(w, r, errors.NewValidationError("maximum 1000 posts per bulk request"))
		return
	}

	// Create posts
	response, err := a.postService.BulkCreate(ctx, &req)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	logger.InfoContext(ctx, "bulk create completed via API",
		"created", response.Created,
		"failed", response.Failed,
	)

	statusCode := http.StatusCreated
	if response.Failed > 0 {
		statusCode = http.StatusMultiStatus
	}

	a.respondJSON(w, statusCode, map[string]interface{}{
		"data": response,
		"meta": &models.ResponseMeta{
			RequestID: getRequestID(ctx),
			Timestamp: time.Now(),
		},
	})
}

// ExportPosts handles GET /api/v1/posts/export
func (a *API) ExportPosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse format
	format := models.ExportFormat(r.URL.Query().Get("format"))
	if format == "" {
		format = models.ExportFormatJSON
	}

	// Validate format
	if format != models.ExportFormatJSON && format != models.ExportFormatCSV {
		a.respondError(w, r, errors.NewValidationError("invalid export format (json or csv)"))
		return
	}

	// Parse filter
	filter := &models.PostFilter{}
	if userIDStr := r.URL.Query().Get("userId"); userIDStr != "" {
		userID, err := strconv.Atoi(userIDStr)
		if err == nil {
			filter.UserID = &userID
		}
	}
	filter.Search = r.URL.Query().Get("search")

	// Set headers
	filename := "posts_export_" + time.Now().Format("20060102_150405")
	if format == models.ExportFormatJSON {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename+".json")
	} else {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename+".csv")
	}

	// Export posts
	if err := a.postService.ExportPosts(ctx, w, format, filter); err != nil {
		logger.ErrorContext(ctx, "export failed", "error", err)
		a.respondError(w, r, err)
		return
	}

	logger.InfoContext(ctx, "posts exported", "format", format)
}

// AnalyzePosts handles GET /api/v1/posts/analytics
func (a *API) AnalyzePosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	// Perform analysis
	result, err := a.postService.AnalyzeCharacterFrequency(ctx)
	if err != nil {
		a.respondError(w, r, err)
		return
	}

	logger.InfoContext(ctx, "analysis completed via API",
		"total_posts", result.TotalPosts,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	a.respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": result,
		"meta": &models.ResponseMeta{
			RequestID: getRequestID(ctx),
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		},
	})
}

// Helper methods

func (a *API) parsePagination(r *http.Request) *models.PaginationParams {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20 // default
	}

	return &models.PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Offset:   (page - 1) * pageSize,
	}
}

func (a *API) extractID(r *http.Request) (int, error) {
	// Extract ID from path: /api/v1/posts/{id}
	path := r.URL.Path
	parts := strings.Split(path, "/")

	// Find the "posts" segment
	for i, part := range parts {
		if part == "posts" && i+1 < len(parts) {
			// The next part should be the ID
			idStr := parts[i+1]
			// Remove any query parameters
			if idx := strings.Index(idStr, "?"); idx != -1 {
				idStr = idStr[:idx]
			}
			// Skip if it's a special endpoint
			if idStr == "bulk" || idStr == "export" || idStr == "analytics" {
				continue
			}
			return strconv.Atoi(idStr)
		}
	}

	return 0, errors.NewValidationError("invalid post ID")
}

func (a *API) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error("failed to encode JSON response", "error", err)
	}
}

func (a *API) respondError(w http.ResponseWriter, r *http.Request, err error) {
	appErr, ok := err.(*errors.AppError)
	if !ok {
		appErr = errors.NewInternalError(err)
	}

	logger.ErrorContext(r.Context(), "API error",
		"code", appErr.Code,
		"message", appErr.Message,
		"status", appErr.StatusCode,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.StatusCode)

	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    appErr.Code,
			"message": appErr.Message,
		},
		"meta": &models.ResponseMeta{
			RequestID: getRequestID(r.Context()),
			Timestamp: time.Now(),
		},
	}

	if appErr.Fields != nil && len(appErr.Fields) > 0 {
		response["error"].(map[string]interface{})["fields"] = appErr.Fields
	}

	_ = json.NewEncoder(w).Encode(response)
}

func getRequestID(ctx interface{}) string {
	if reqID, ok := ctx.(interface{ Value(interface{}) interface{} }); ok {
		if val := reqID.Value(logger.RequestIDKey); val != nil {
			if id, ok := val.(string); ok {
				return id
			}
		}
	}
	return ""
}
