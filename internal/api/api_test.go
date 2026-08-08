package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Post_Analyzer_Webserver/config"
	"Post_Analyzer_Webserver/internal/abac"
	apperrors "Post_Analyzer_Webserver/internal/errors"
	"Post_Analyzer_Webserver/internal/middleware"
	"Post_Analyzer_Webserver/internal/models"
)

// fakePostClient implements rpcclient.PostClient without any RPC, so the
// API handlers can be tested in isolation from postsvc.
type fakePostClient struct {
	posts  map[int]models.Post
	nextID int

	getAllErr  error
	getByIDErr error
	createErr  error
	updateErr  error
	deleteErr  error
	bulkErr    error
	analyzeErr error

	bulkResponse  *models.BulkCreateResponse
	analyzeResult *models.AnalyticsResult
}

func newFakePostClient() *fakePostClient {
	return &fakePostClient{posts: make(map[int]models.Post), nextID: 1}
}

func (f *fakePostClient) GetAll(ctx context.Context, filter *models.PostFilter, pagination *models.PaginationParams) ([]models.Post, *models.PaginationMeta, error) {
	if f.getAllErr != nil {
		return nil, nil, f.getAllErr
	}
	out := make([]models.Post, 0, len(f.posts))
	for _, p := range f.posts {
		if filter != nil && filter.UserID != nil && p.UserID != *filter.UserID {
			continue
		}
		out = append(out, p)
	}
	meta := &models.PaginationMeta{TotalItems: len(out)}
	if pagination != nil {
		meta.Page = pagination.Page
		meta.PageSize = pagination.PageSize
	}
	return out, meta, nil
}

func (f *fakePostClient) GetByID(ctx context.Context, id int) (*models.Post, error) {
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}
	p, ok := f.posts[id]
	if !ok {
		return nil, apperrors.NewNotFound("Post")
	}
	return &p, nil
}

func (f *fakePostClient) Create(ctx context.Context, req *models.CreatePostRequest) (*models.Post, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	p := models.Post{ID: f.nextID, UserID: req.UserID, Title: req.Title, Body: req.Body}
	f.posts[p.ID] = p
	f.nextID++
	return &p, nil
}

func (f *fakePostClient) Update(ctx context.Context, id int, req *models.UpdatePostRequest) (*models.Post, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	p, ok := f.posts[id]
	if !ok {
		return nil, apperrors.NewNotFound("Post")
	}
	if req.Title != "" {
		p.Title = req.Title
	}
	if req.Body != "" {
		p.Body = req.Body
	}
	f.posts[id] = p
	return &p, nil
}

func (f *fakePostClient) Delete(ctx context.Context, id int) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if _, ok := f.posts[id]; !ok {
		return apperrors.NewNotFound("Post")
	}
	delete(f.posts, id)
	return nil
}

func (f *fakePostClient) BulkCreate(ctx context.Context, req *models.BulkCreateRequest) (*models.BulkCreateResponse, error) {
	if f.bulkErr != nil {
		return nil, f.bulkErr
	}
	if f.bulkResponse != nil {
		return f.bulkResponse, nil
	}
	resp := &models.BulkCreateResponse{}
	for _, p := range req.Posts {
		created, _ := f.Create(ctx, &p)
		resp.Created++
		resp.PostIDs = append(resp.PostIDs, created.ID)
	}
	return resp, nil
}

func (f *fakePostClient) AnalyzeCharacterFrequency(ctx context.Context) (*models.AnalyticsResult, error) {
	if f.analyzeErr != nil {
		return nil, f.analyzeErr
	}
	if f.analyzeResult != nil {
		return f.analyzeResult, nil
	}
	return &models.AnalyticsResult{TotalPosts: len(f.posts)}, nil
}

// fakeAuthClient implements rpcclient.AuthClient without any RPC.
type fakeAuthClient struct {
	loginErr error
	token    string
	subject  abac.Subject
}

func (f *fakeAuthClient) Login(ctx context.Context, username, password string) (string, abac.Subject, error) {
	if f.loginErr != nil {
		return "", abac.Subject{}, f.loginErr
	}
	return f.token, f.subject, nil
}

func (f *fakeAuthClient) ValidateToken(ctx context.Context, token string) (abac.Subject, error) {
	return f.subject, nil
}

func (f *fakeAuthClient) Authorize(ctx context.Context, req abac.Request) (abac.Decision, error) {
	return abac.Decision{Allowed: true}, nil
}

func newTestAPI(post *fakePostClient, auth *fakeAuthClient) *API {
	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "development"},
		RPC:    config.RPCConfig{PostServiceAddr: "postsvc:9001", AuthServiceAddr: "authsvc:9002"},
	}
	return NewAPI(post, auth, nil, nil, nil, cfg)
}

func doJSONRequest(t *testing.T, handler http.HandlerFunc, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

func decodeBody(t *testing.T, rr *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rr.Body.Bytes(), v); err != nil {
		t.Fatalf("response is not valid JSON: %v (%s)", err, rr.Body.String())
	}
}

func TestLogin_Success(t *testing.T) {
	auth := &fakeAuthClient{token: "jwt-token", subject: abac.Subject{Username: "admin", Role: "admin"}}
	a := newTestAPI(newFakePostClient(), auth)

	rr := doJSONRequest(t, a.Login, http.MethodPost, "/api/v1/auth/login", map[string]string{"username": "admin", "password": "admin123"})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var body struct {
		Data struct {
			Token string `json:"token"`
			Role  string `json:"role"`
		} `json:"data"`
	}
	decodeBody(t, rr, &body)
	if body.Data.Token != "jwt-token" || body.Data.Role != "admin" {
		t.Errorf("unexpected response: %+v", body.Data)
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	auth := &fakeAuthClient{loginErr: apperrors.ErrUnauthorized}
	a := newTestAPI(newFakePostClient(), auth)

	rr := doJSONRequest(t, a.Login, http.MethodPost, "/api/v1/auth/login", map[string]string{"username": "admin", "password": "wrong"})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestLogin_MalformedBody(t *testing.T) {
	a := newTestAPI(newFakePostClient(), &fakeAuthClient{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("not json"))
	rr := httptest.NewRecorder()
	a.Login(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for malformed JSON, got %d", rr.Code)
	}
}

func TestCreatePost_Success(t *testing.T) {
	a := newTestAPI(newFakePostClient(), &fakeAuthClient{})
	rr := doJSONRequest(t, a.CreatePost, http.MethodPost, "/api/v1/posts", models.CreatePostRequest{UserID: 1, Title: "Hello", Body: "World"})
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var body struct {
		Data models.Post `json:"data"`
	}
	decodeBody(t, rr, &body)
	if body.Data.Title != "Hello" || body.Data.ID == 0 {
		t.Errorf("unexpected post: %+v", body.Data)
	}
}

func TestCreatePost_RPCError(t *testing.T) {
	post := newFakePostClient()
	post.createErr = apperrors.NewInternalError(nil)
	a := newTestAPI(post, &fakeAuthClient{})
	rr := doJSONRequest(t, a.CreatePost, http.MethodPost, "/api/v1/posts", models.CreatePostRequest{UserID: 1, Title: "X", Body: "Y"})
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestGetPost_Success(t *testing.T) {
	post := newFakePostClient()
	post.posts[1] = models.Post{ID: 1, Title: "Existing"}
	a := newTestAPI(post, &fakeAuthClient{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/1", nil)
	rr := httptest.NewRecorder()
	a.GetPost(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetPost_NotFound(t *testing.T) {
	a := newTestAPI(newFakePostClient(), &fakeAuthClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/999", nil)
	rr := httptest.NewRecorder()
	a.GetPost(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestGetPost_InvalidID(t *testing.T) {
	a := newTestAPI(newFakePostClient(), &fakeAuthClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/not-a-number", nil)
	rr := httptest.NewRecorder()
	a.GetPost(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for a non-numeric ID, got %d", rr.Code)
	}
}

func TestUpdatePost_Success(t *testing.T) {
	post := newFakePostClient()
	post.posts[1] = models.Post{ID: 1, Title: "Original"}
	a := newTestAPI(post, &fakeAuthClient{})

	body, _ := json.Marshal(models.UpdatePostRequest{Title: "Updated"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/posts/1", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	a.UpdatePost(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDeletePost_Success(t *testing.T) {
	post := newFakePostClient()
	post.posts[1] = models.Post{ID: 1}
	a := newTestAPI(post, &fakeAuthClient{})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/posts/1", nil)
	rr := httptest.NewRecorder()
	a.DeletePost(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if _, ok := post.posts[1]; ok {
		t.Error("expected the post to be removed from the fake store")
	}
}

func TestListPosts_WithFilterAndPagination(t *testing.T) {
	post := newFakePostClient()
	post.posts[1] = models.Post{ID: 1, UserID: 1}
	post.posts[2] = models.Post{ID: 2, UserID: 2}
	a := newTestAPI(post, &fakeAuthClient{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?userId=1&page=1&pageSize=10", nil)
	rr := httptest.NewRecorder()
	a.ListPosts(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var body models.PaginatedResponse
	decodeBody(t, rr, &body)
	if body.Pagination.TotalItems != 1 {
		t.Errorf("expected the userId filter to narrow to 1 post, got %+v", body)
	}
}

func TestListPosts_RPCError(t *testing.T) {
	post := newFakePostClient()
	post.getAllErr = apperrors.NewInternalError(nil)
	a := newTestAPI(post, &fakeAuthClient{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil)
	rr := httptest.NewRecorder()
	a.ListPosts(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestBulkCreatePosts_EmptyRejected(t *testing.T) {
	a := newTestAPI(newFakePostClient(), &fakeAuthClient{})
	rr := doJSONRequest(t, a.BulkCreatePosts, http.MethodPost, "/api/v1/posts/bulk", models.BulkCreateRequest{Posts: nil})
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for an empty posts array, got %d", rr.Code)
	}
}

func TestBulkCreatePosts_TooManyRejected(t *testing.T) {
	posts := make([]models.CreatePostRequest, 1001)
	for i := range posts {
		posts[i] = models.CreatePostRequest{UserID: 1, Title: "T", Body: "B"}
	}
	a := newTestAPI(newFakePostClient(), &fakeAuthClient{})
	rr := doJSONRequest(t, a.BulkCreatePosts, http.MethodPost, "/api/v1/posts/bulk", models.BulkCreateRequest{Posts: posts})
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for >1000 posts, got %d", rr.Code)
	}
}

func TestBulkCreatePosts_Success(t *testing.T) {
	a := newTestAPI(newFakePostClient(), &fakeAuthClient{})
	rr := doJSONRequest(t, a.BulkCreatePosts, http.MethodPost, "/api/v1/posts/bulk", models.BulkCreateRequest{
		Posts: []models.CreatePostRequest{{UserID: 1, Title: "A", Body: "B"}, {UserID: 1, Title: "C", Body: "D"}},
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestBulkCreatePosts_PartialFailureReturns207(t *testing.T) {
	post := newFakePostClient()
	post.bulkResponse = &models.BulkCreateResponse{Created: 1, Failed: 1, Errors: []string{"boom"}}
	a := newTestAPI(post, &fakeAuthClient{})
	rr := doJSONRequest(t, a.BulkCreatePosts, http.MethodPost, "/api/v1/posts/bulk", models.BulkCreateRequest{
		Posts: []models.CreatePostRequest{{UserID: 1, Title: "A", Body: "B"}},
	})
	if rr.Code != http.StatusMultiStatus {
		t.Errorf("expected 207 when some posts fail, got %d", rr.Code)
	}
}

func TestExportPosts_JSON(t *testing.T) {
	post := newFakePostClient()
	post.posts[1] = models.Post{ID: 1, Title: "Exportable", Body: "Body"}
	a := newTestAPI(post, &fakeAuthClient{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/export?format=json", nil)
	rr := httptest.NewRecorder()
	a.ExportPosts(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json content type, got %s", ct)
	}
	var posts []models.Post
	if err := json.Unmarshal(rr.Body.Bytes(), &posts); err != nil {
		t.Fatalf("export body isn't valid JSON: %v", err)
	}
}

func TestExportPosts_CSV(t *testing.T) {
	post := newFakePostClient()
	post.posts[1] = models.Post{ID: 1, Title: "Exportable", Body: "Body"}
	a := newTestAPI(post, &fakeAuthClient{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/export?format=csv", nil)
	rr := httptest.NewRecorder()
	a.ExportPosts(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/csv" {
		t.Errorf("expected text/csv content type, got %s", ct)
	}
}

func TestExportPosts_InvalidFormat(t *testing.T) {
	a := newTestAPI(newFakePostClient(), &fakeAuthClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/export?format=xml", nil)
	rr := httptest.NewRecorder()
	a.ExportPosts(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for an unsupported format, got %d", rr.Code)
	}
}

func TestReanalyzePosts_ServiceUnavailableWhenRabbitMQNil(t *testing.T) {
	a := newTestAPI(newFakePostClient(), &fakeAuthClient{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/posts/reanalyze", nil)
	rr := httptest.NewRecorder()
	a.ReanalyzePosts(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when rabbitmq is not configured, got %d", rr.Code)
	}
}

func TestListExports_ServiceUnavailableWhenObjectsNil(t *testing.T) {
	a := newTestAPI(newFakePostClient(), &fakeAuthClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports", nil)
	rr := httptest.NewRecorder()
	a.ListExports(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when object store is not configured, got %d", rr.Code)
	}
}

func TestGetExport_ServiceUnavailableWhenObjectsNil(t *testing.T) {
	a := newTestAPI(newFakePostClient(), &fakeAuthClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports/somekey.json", nil)
	rr := httptest.NewRecorder()
	a.GetExport(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when object store is not configured, got %d", rr.Code)
	}
}

func TestClassifySentiment_ServiceUnavailableWhenTritonNil(t *testing.T) {
	a := newTestAPI(newFakePostClient(), &fakeAuthClient{})
	rr := doJSONRequest(t, a.ClassifySentiment, http.MethodPost, "/api/v1/ml/sentiment", map[string]string{"text": "great"})
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when triton is not configured, got %d", rr.Code)
	}
}

func TestAnalyzePosts_Success(t *testing.T) {
	post := newFakePostClient()
	post.analyzeResult = &models.AnalyticsResult{TotalPosts: 5, TotalCharacters: 100}
	a := newTestAPI(post, &fakeAuthClient{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/analytics", nil)
	rr := httptest.NewRecorder()
	a.AnalyzePosts(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var body struct {
		Data models.AnalyticsResult `json:"data"`
	}
	decodeBody(t, rr, &body)
	if body.Data.TotalPosts != 5 {
		t.Errorf("unexpected analytics result: %+v", body.Data)
	}
}

func TestAnalyzePosts_Error(t *testing.T) {
	post := newFakePostClient()
	post.analyzeErr = apperrors.NewInternalError(nil)
	a := newTestAPI(post, &fakeAuthClient{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/analytics", nil)
	rr := httptest.NewRecorder()
	a.AnalyzePosts(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestAdminStatus_ReportsConfiguredIntegrationsAndSubject(t *testing.T) {
	cfg := &config.Config{
		Server:      config.ServerConfig{Environment: "production"},
		Redis:       config.RedisConfig{Enabled: true},
		Messaging:   config.MessagingConfig{KafkaEnabled: true, RabbitMQEnabled: false, RocketMQEnabled: true},
		ObjectStore: config.ObjectStoreConfig{Enabled: true},
		ML:          config.MLConfig{Enabled: false},
		RPC:         config.RPCConfig{PostServiceAddr: "postsvc:9001", AuthServiceAddr: "authsvc:9002", MuxTransport: true},
	}
	a := NewAPI(newFakePostClient(), &fakeAuthClient{}, nil, nil, nil, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/status", nil)
	ctx := context.WithValue(req.Context(), middleware.SubjectContextKey, abac.Subject{Username: "admin", Role: "admin"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	a.AdminStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var body struct {
		Data struct {
			Environment  string          `json:"environment"`
			Integrations map[string]bool `json:"integrations"`
			RequestedBy  string          `json:"requestedBy"`
		} `json:"data"`
	}
	decodeBody(t, rr, &body)
	if body.Data.Environment != "production" {
		t.Errorf("expected environment=production, got %s", body.Data.Environment)
	}
	if !body.Data.Integrations["redis"] || !body.Data.Integrations["kafka"] || body.Data.Integrations["rabbitmq"] {
		t.Errorf("integrations don't match config: %+v", body.Data.Integrations)
	}
	if body.Data.RequestedBy != "admin" {
		t.Errorf("expected requestedBy=admin from context subject, got %s", body.Data.RequestedBy)
	}
}

func TestParsePagination_Defaults(t *testing.T) {
	a := newTestAPI(newFakePostClient(), &fakeAuthClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil)
	p := a.parsePagination(req)
	if p.Page != 1 || p.PageSize != 20 {
		t.Errorf("expected default page=1 pageSize=20, got %+v", p)
	}
}

func TestParsePagination_ClampsOutOfRangePageSize(t *testing.T) {
	a := newTestAPI(newFakePostClient(), &fakeAuthClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?pageSize=500", nil)
	p := a.parsePagination(req)
	if p.PageSize != 20 {
		t.Errorf("expected an out-of-range pageSize to fall back to default 20, got %d", p.PageSize)
	}
}

func TestParsePagination_ComputesOffset(t *testing.T) {
	a := newTestAPI(newFakePostClient(), &fakeAuthClient{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?page=3&pageSize=10", nil)
	p := a.parsePagination(req)
	if p.Offset != 20 {
		t.Errorf("expected offset=20 for page=3 pageSize=10, got %d", p.Offset)
	}
}

func TestExtractID_SkipsSpecialSegments(t *testing.T) {
	a := newTestAPI(newFakePostClient(), &fakeAuthClient{})
	cases := []struct {
		path    string
		wantErr bool
	}{
		{"/api/v1/posts/42", false},
		{"/api/v1/posts/bulk", true},   // no numeric ID follows "bulk"
		{"/api/v1/posts/export", true}, // no numeric ID follows "export"
		{"/api/v1/posts/abc", true},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		_, err := a.extractID(req)
		if tc.wantErr && err == nil {
			t.Errorf("path %s: expected an error, got none", tc.path)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("path %s: unexpected error: %v", tc.path, err)
		}
	}
}
