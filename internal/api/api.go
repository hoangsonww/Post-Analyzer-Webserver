package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"Post_Analyzer_Webserver/config"
	"Post_Analyzer_Webserver/internal/errors"
	"Post_Analyzer_Webserver/internal/export"
	"Post_Analyzer_Webserver/internal/gen/eventpb"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/messaging/rabbitmq"
	"Post_Analyzer_Webserver/internal/middleware"
	"Post_Analyzer_Webserver/internal/ml/triton"
	"Post_Analyzer_Webserver/internal/models"
	"Post_Analyzer_Webserver/internal/objectstore"
	"Post_Analyzer_Webserver/internal/rpcclient"

	"github.com/google/uuid"
)

// API handles REST API endpoints. It talks to the post-analysis and auth
// RPC services (postsvc, authsvc) via the rpcclient package rather than
// embedding business logic in-process.
type API struct {
	postService rpcclient.PostClient
	authService rpcclient.AuthClient
	rabbitmq    *rabbitmq.Client   // optional: nil disables ReanalyzePosts
	objects     *objectstore.Store // optional: nil disables export persistence/listing
	triton      *triton.Client     // optional: nil disables ClassifySentiment
	cfg         *config.Config     // for GET /api/v1/admin/status feature-flag reporting
	startTime   time.Time
}

// NewAPI creates a new API handler. rmq/objects/tritonClient may be nil
// when the corresponding broker/store/model isn't enabled; the affected
// endpoints then return 503 instead of failing gateway startup.
func NewAPI(postService rpcclient.PostClient, authService rpcclient.AuthClient, rmq *rabbitmq.Client, objects *objectstore.Store, tritonClient *triton.Client, cfg *config.Config) *API {
	return &API{
		postService: postService,
		authService: authService,
		rabbitmq:    rmq,
		triton:      tritonClient,
		objects:     objects,
		cfg:         cfg,
		startTime:   time.Now(),
	}
}

// Login handles POST /api/v1/auth/login, exchanging credentials for a JWT
// issued by authsvc.
func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.respondError(w, r, errors.NewValidationError("invalid request body"))
		return
	}

	token, subj, err := a.authService.Login(ctx, req.Username, req.Password)
	if err != nil {
		a.respondError(w, r, errors.ErrUnauthorized)
		return
	}

	a.respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"token":    token,
			"username": subj.Username,
			"role":     subj.Role,
		},
		"meta": &models.ResponseMeta{
			RequestID: getRequestID(ctx),
			Timestamp: time.Now(),
		},
	})
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

	filename := "posts_export_" + time.Now().Format("20060102_150405")
	contentType := "application/json"
	ext := ".json"
	if format == models.ExportFormatCSV {
		contentType = "text/csv"
		ext = ".csv"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename="+filename+ext)

	// Export posts (fetched via RPC, formatted here in the gateway)
	posts, _, err := a.postService.GetAll(ctx, filter, nil)
	if err != nil {
		logger.ErrorContext(ctx, "export failed", "error", err)
		a.respondError(w, r, err)
		return
	}

	var buf bytes.Buffer
	if err := export.Write(&buf, format, posts); err != nil {
		logger.ErrorContext(ctx, "export failed", "error", err)
		a.respondError(w, r, err)
		return
	}

	if a.objects != nil {
		key := "exports/" + filename + ext
		if err := a.objects.Put(ctx, key, bytes.NewReader(buf.Bytes()), int64(buf.Len()), contentType); err != nil {
			logger.WarnContext(ctx, "failed to persist export to object store", "key", key, "error", err)
		} else {
			logger.InfoContext(ctx, "export persisted to object store", "key", key, "size", buf.Len())
		}
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		logger.ErrorContext(ctx, "failed to write export response", "error", err)
	}

	logger.InfoContext(ctx, "posts exported", "format", format)
}

// ReanalyzePosts handles POST /api/v1/posts/reanalyze: instead of running
// the analysis inline, it enqueues a ReanalysisJob on RabbitMQ and returns
// immediately — cmd/reanalysis-worker does the actual work asynchronously.
func (a *API) ReanalyzePosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if a.rabbitmq == nil {
		a.respondError(w, r, errors.ErrServiceUnavailable)
		return
	}

	requestedBy := "anonymous"
	if subj, ok := middleware.SubjectFromContext(ctx); ok {
		requestedBy = subj.Username
	}

	job := &eventpb.ReanalysisJob{
		JobId:           uuid.NewString(),
		RequestedBy:     requestedBy,
		RequestedAtUnix: time.Now().Unix(),
		FullCorpus:      true,
	}

	if err := a.rabbitmq.Publish(ctx, rabbitmq.ReanalysisQueue, job); err != nil {
		logger.ErrorContext(ctx, "failed to enqueue reanalysis job", "error", err)
		a.respondError(w, r, errors.NewInternalError(err))
		return
	}

	logger.InfoContext(ctx, "reanalysis job enqueued", "job_id", job.JobId, "requested_by", requestedBy)

	a.respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"data": map[string]interface{}{
			"jobId":  job.JobId,
			"status": "queued",
		},
		"meta": &models.ResponseMeta{
			RequestID: getRequestID(ctx),
			Timestamp: time.Now(),
		},
	})
}

// ListExports handles GET /api/v1/exports: lists previously generated
// exports persisted to the MinIO object store.
func (a *API) ListExports(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if a.objects == nil {
		a.respondError(w, r, errors.ErrServiceUnavailable)
		return
	}

	objs, err := a.objects.List(ctx, "exports/")
	if err != nil {
		a.respondError(w, r, errors.NewInternalError(err))
		return
	}

	a.respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": objs,
		"meta": &models.ResponseMeta{
			RequestID: getRequestID(ctx),
			Timestamp: time.Now(),
		},
	})
}

// GetExport handles GET /api/v1/exports/{key}: streams a previously
// generated export back from the MinIO object store.
func (a *API) GetExport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if a.objects == nil {
		a.respondError(w, r, errors.ErrServiceUnavailable)
		return
	}

	key := strings.TrimPrefix(r.URL.Path, "/api/v1/exports/")
	if key == "" {
		a.respondError(w, r, errors.NewValidationError("export key is required"))
		return
	}

	obj, err := a.objects.Get(ctx, "exports/"+key)
	if err != nil {
		a.respondError(w, r, errors.NewNotFound("Export"))
		return
	}
	defer obj.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+key)
	if _, err := io.Copy(w, obj); err != nil {
		logger.ErrorContext(ctx, "failed to stream export", "key", key, "error", err)
	}
}

// AdminStatus handles GET /api/v1/admin/status: feature-flag / integration
// visibility for operators. Gated to the "admin" ABAC resource (only the
// admin role matches any policy for it — see internal/abac.DefaultPolicies),
// so this is enforced by the same policy engine as everything else, not a
// special-cased check in this handler.
func (a *API) AdminStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	subject, _ := middleware.SubjectFromContext(ctx)

	integrations := map[string]bool{
		"redis":    a.cfg.Redis.Enabled,
		"kafka":    a.cfg.Messaging.KafkaEnabled,
		"rabbitmq": a.cfg.Messaging.RabbitMQEnabled,
		"rocketmq": a.cfg.Messaging.RocketMQEnabled,
		"minio":    a.cfg.ObjectStore.Enabled,
		"triton":   a.cfg.ML.Enabled,
		"rpc_mux":  a.cfg.RPC.MuxTransport,
	}

	a.respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"environment":  a.cfg.Server.Environment,
			"uptime":       time.Since(a.startTime).String(),
			"integrations": integrations,
			"rpc": map[string]string{
				"postsvc": a.cfg.RPC.PostServiceAddr,
				"authsvc": a.cfg.RPC.AuthServiceAddr,
			},
			"requestedBy": subject.Username,
		},
		"meta": &models.ResponseMeta{
			RequestID: getRequestID(ctx),
			Timestamp: time.Now(),
		},
	})
}

// ClassifySentiment handles POST /api/v1/ml/sentiment: on-demand sentiment
// classification of arbitrary text via the Triton-served post_sentiment
// model. This is the same enrichment postsvc runs automatically on post
// creation (when Triton + Kafka are both enabled), exposed directly here
// so it can be tried without creating a post first.
func (a *API) ClassifySentiment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if a.triton == nil {
		a.respondError(w, r, errors.ErrServiceUnavailable)
		return
	}

	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Text) == "" {
		a.respondError(w, r, errors.NewValidationError("text is required"))
		return
	}

	result, err := a.triton.ClassifySentiment(ctx, req.Text)
	if err != nil {
		logger.ErrorContext(ctx, "triton classification failed", "error", err)
		a.respondError(w, r, errors.NewInternalError(err))
		return
	}

	a.respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"label":         result.Label,
			"probabilities": result.Probabilities,
		},
		"meta": &models.ResponseMeta{
			RequestID: getRequestID(ctx),
			Timestamp: time.Now(),
		},
	})
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
