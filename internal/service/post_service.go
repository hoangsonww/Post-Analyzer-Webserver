package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"Post_Analyzer_Webserver/internal/errors"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/metrics"
	"Post_Analyzer_Webserver/internal/models"
	"Post_Analyzer_Webserver/internal/storage"
)

// PostService handles business logic for posts
type PostService struct {
	storage storage.Storage
}

// NewPostService creates a new post service
func NewPostService(storage storage.Storage) *PostService {
	return &PostService{
		storage: storage,
	}
}

// GetAll retrieves all posts with optional filtering and pagination
func (s *PostService) GetAll(ctx context.Context, filter *models.PostFilter, pagination *models.PaginationParams) ([]models.Post, *models.PaginationMeta, error) {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("get_all_posts", "success", time.Since(start))
	}()

	// Get all posts from storage
	storagePosts, err := s.storage.GetAll(ctx)
	if err != nil {
		metrics.RecordDBOperation("get_all_posts", "error", time.Since(start))
		return nil, nil, errors.Wrap(err, "failed to retrieve posts")
	}

	// Convert storage posts to models
	posts := make([]models.Post, len(storagePosts))
	for i, sp := range storagePosts {
		posts[i] = models.Post{
			ID:        sp.Id,
			UserID:    sp.UserId,
			Title:     sp.Title,
			Body:      sp.Body,
			CreatedAt: sp.CreatedAt,
			UpdatedAt: sp.UpdatedAt,
		}
	}

	// Apply filtering
	posts = s.filterPosts(posts, filter)

	// Apply sorting
	posts = s.sortPosts(posts, filter)

	// Calculate pagination
	totalItems := len(posts)
	paginationMeta := s.calculatePagination(totalItems, pagination)

	// Apply pagination
	if pagination != nil {
		start := pagination.Offset
		end := start + pagination.PageSize
		if start > len(posts) {
			posts = []models.Post{}
		} else if end > len(posts) {
			posts = posts[start:]
		} else {
			posts = posts[start:end]
		}
	}

	return posts, paginationMeta, nil
}

// GetByID retrieves a post by ID
func (s *PostService) GetByID(ctx context.Context, id int) (*models.Post, error) {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("get_post_by_id", "success", time.Since(start))
	}()

	storagePost, err := s.storage.GetByID(ctx, id)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, errors.NewNotFound("Post")
		}
		metrics.RecordDBOperation("get_post_by_id", "error", time.Since(start))
		return nil, errors.Wrap(err, "failed to retrieve post")
	}

	post := &models.Post{
		ID:        storagePost.Id,
		UserID:    storagePost.UserId,
		Title:     storagePost.Title,
		Body:      storagePost.Body,
		CreatedAt: storagePost.CreatedAt,
		UpdatedAt: storagePost.UpdatedAt,
	}

	return post, nil
}

// Create creates a new post
func (s *PostService) Create(ctx context.Context, req *models.CreatePostRequest) (*models.Post, error) {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("create_post", "success", time.Since(start))
	}()

	// Validate input
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Create storage post
	storagePost := &storage.Post{
		UserId: req.UserID,
		Title:  strings.TrimSpace(req.Title),
		Body:   strings.TrimSpace(req.Body),
	}

	if err := s.storage.Create(ctx, storagePost); err != nil {
		metrics.RecordDBOperation("create_post", "error", time.Since(start))
		return nil, errors.Wrap(err, "failed to create post")
	}

	logger.InfoContext(ctx, "post created", "id", storagePost.Id)

	post := &models.Post{
		ID:        storagePost.Id,
		UserID:    storagePost.UserId,
		Title:     storagePost.Title,
		Body:      storagePost.Body,
		CreatedAt: storagePost.CreatedAt,
		UpdatedAt: storagePost.UpdatedAt,
	}

	return post, nil
}

// Update updates an existing post
func (s *PostService) Update(ctx context.Context, id int, req *models.UpdatePostRequest) (*models.Post, error) {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("update_post", "success", time.Since(start))
	}()

	// Get existing post
	existing, err := s.storage.GetByID(ctx, id)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, errors.NewNotFound("Post")
		}
		return nil, errors.Wrap(err, "failed to retrieve post")
	}

	// Update fields
	if req.UserID != 0 {
		existing.UserId = req.UserID
	}
	if req.Title != "" {
		if len(req.Title) > 500 {
			return nil, errors.NewValidationError("title too long").WithField("title", "maximum 500 characters")
		}
		existing.Title = strings.TrimSpace(req.Title)
	}
	if req.Body != "" {
		if len(req.Body) > 10000 {
			return nil, errors.NewValidationError("body too long").WithField("body", "maximum 10000 characters")
		}
		existing.Body = strings.TrimSpace(req.Body)
	}

	if err := s.storage.Update(ctx, existing); err != nil {
		metrics.RecordDBOperation("update_post", "error", time.Since(start))
		return nil, errors.Wrap(err, "failed to update post")
	}

	post := &models.Post{
		ID:        existing.Id,
		UserID:    existing.UserId,
		Title:     existing.Title,
		Body:      existing.Body,
		CreatedAt: existing.CreatedAt,
		UpdatedAt: existing.UpdatedAt,
	}

	return post, nil
}

// Delete deletes a post
func (s *PostService) Delete(ctx context.Context, id int) error {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("delete_post", "success", time.Since(start))
	}()

	if err := s.storage.Delete(ctx, id); err != nil {
		if err == storage.ErrNotFound {
			return errors.NewNotFound("Post")
		}
		metrics.RecordDBOperation("delete_post", "error", time.Since(start))
		return errors.Wrap(err, "failed to delete post")
	}

	logger.InfoContext(ctx, "post deleted", "id", id)
	return nil
}

// BulkCreate creates multiple posts
func (s *PostService) BulkCreate(ctx context.Context, req *models.BulkCreateRequest) (*models.BulkCreateResponse, error) {
	start := time.Now()
	defer func() {
		metrics.RecordDBOperation("bulk_create_posts", "success", time.Since(start))
	}()

	response := &models.BulkCreateResponse{
		PostIDs: make([]int, 0),
		Errors:  make([]string, 0),
	}

	for i, postReq := range req.Posts {
		post, err := s.Create(ctx, &postReq)
		if err != nil {
			response.Failed++
			response.Errors = append(response.Errors, fmt.Sprintf("post %d: %v", i+1, err))
			continue
		}
		response.Created++
		response.PostIDs = append(response.PostIDs, post.ID)
	}

	logger.InfoContext(ctx, "bulk create completed", "created", response.Created, "failed", response.Failed)
	return response, nil
}

// ExportPosts exports posts in the specified format
func (s *PostService) ExportPosts(ctx context.Context, writer io.Writer, format models.ExportFormat, filter *models.PostFilter) error {
	posts, _, err := s.GetAll(ctx, filter, nil)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve posts for export")
	}

	switch format {
	case models.ExportFormatJSON:
		return s.exportJSON(writer, posts)
	case models.ExportFormatCSV:
		return s.exportCSV(writer, posts)
	default:
		return errors.NewValidationError("unsupported export format")
	}
}

// AnalyzeCharacterFrequency performs character frequency analysis
func (s *PostService) AnalyzeCharacterFrequency(ctx context.Context) (*models.AnalyticsResult, error) {
	start := time.Now()
	defer func() {
		metrics.RecordAnalysisOperation(time.Since(start))
	}()

	posts, _, err := s.GetAll(ctx, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve posts for analysis")
	}

	result := &models.AnalyticsResult{
		TotalPosts:    len(posts),
		CharFrequency: make(map[rune]int),
	}

	// Analyze character frequency
	totalChars := 0
	postLengths := make([]int, len(posts))
	postsPerUser := make(map[int]int)

	for i, post := range posts {
		text := post.Title + " " + post.Body
		postLengths[i] = len(text)
		postsPerUser[post.UserID]++

		for _, char := range text {
			result.CharFrequency[char]++
			totalChars++
		}
	}

	result.TotalCharacters = totalChars
	result.UniqueChars = len(result.CharFrequency)

	// Calculate top characters
	result.TopCharacters = s.calculateTopCharacters(result.CharFrequency, totalChars)

	// Calculate statistics
	result.Statistics = &models.AnalyticsStats{
		AveragePostLength: s.calculateAverage(postLengths),
		MedianPostLength:  s.calculateMedian(postLengths),
		PostsPerUser:      postsPerUser,
		TimeDistribution:  s.calculateTimeDistribution(posts),
	}

	logger.InfoContext(ctx, "character analysis completed",
		"total_posts", result.TotalPosts,
		"total_chars", result.TotalCharacters,
		"unique_chars", result.UniqueChars,
	)

	return result, nil
}

// Helper methods

func (s *PostService) validateCreateRequest(req *models.CreatePostRequest) error {
	validationErr := errors.NewValidationError("validation failed")
	hasError := false

	if req.Title == "" {
		_ = validationErr.WithField("title", "title is required")
		hasError = true
	} else if len(req.Title) > 500 {
		_ = validationErr.WithField("title", "title too long (max 500 characters)")
		hasError = true
	}

	if req.Body == "" {
		_ = validationErr.WithField("body", "body is required")
		hasError = true
	} else if len(req.Body) > 10000 {
		_ = validationErr.WithField("body", "body too long (max 10000 characters)")
		hasError = true
	}

	if hasError {
		return validationErr
	}
	return nil
}

func (s *PostService) filterPosts(posts []models.Post, filter *models.PostFilter) []models.Post {
	if filter == nil {
		return posts
	}

	filtered := make([]models.Post, 0, len(posts))
	for _, post := range posts {
		// Filter by user ID
		if filter.UserID != nil && post.UserID != *filter.UserID {
			continue
		}

		// Filter by search term
		if filter.Search != "" {
			searchLower := strings.ToLower(filter.Search)
			if !strings.Contains(strings.ToLower(post.Title), searchLower) &&
				!strings.Contains(strings.ToLower(post.Body), searchLower) {
				continue
			}
		}

		filtered = append(filtered, post)
	}

	return filtered
}

func (s *PostService) sortPosts(posts []models.Post, filter *models.PostFilter) []models.Post {
	if filter == nil || filter.SortBy == "" {
		return posts
	}

	sortBy := filter.SortBy
	sortOrder := filter.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	sort.Slice(posts, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "id":
			less = posts[i].ID < posts[j].ID
		case "title":
			less = posts[i].Title < posts[j].Title
		case "createdAt":
			less = posts[i].CreatedAt.Before(posts[j].CreatedAt)
		case "updatedAt":
			less = posts[i].UpdatedAt.Before(posts[j].UpdatedAt)
		default:
			less = posts[i].ID < posts[j].ID
		}

		if sortOrder == "desc" {
			return !less
		}
		return less
	})

	return posts
}

func (s *PostService) calculatePagination(totalItems int, params *models.PaginationParams) *models.PaginationMeta {
	if params == nil {
		return nil
	}

	totalPages := (totalItems + params.PageSize - 1) / params.PageSize
	if totalPages == 0 {
		totalPages = 1
	}

	return &models.PaginationMeta{
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasNext:    params.Page < totalPages,
		HasPrev:    params.Page > 1,
	}
}

func (s *PostService) exportJSON(writer io.Writer, posts []models.Post) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(posts)
}

func (s *PostService) exportCSV(writer io.Writer, posts []models.Post) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write header
	if err := csvWriter.Write([]string{"ID", "UserID", "Title", "Body", "CreatedAt", "UpdatedAt"}); err != nil {
		return err
	}

	// Write data
	for _, post := range posts {
		row := []string{
			fmt.Sprintf("%d", post.ID),
			fmt.Sprintf("%d", post.UserID),
			post.Title,
			post.Body,
			post.CreatedAt.Format(time.RFC3339),
			post.UpdatedAt.Format(time.RFC3339),
		}
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func (s *PostService) calculateTopCharacters(charFreq map[rune]int, totalChars int) []models.CharacterStat {
	stats := make([]models.CharacterStat, 0, len(charFreq))

	for char, count := range charFreq {
		frequency := float64(count) / float64(totalChars) * 100
		stats = append(stats, models.CharacterStat{
			Character: char,
			Count:     count,
			Frequency: frequency,
		})
	}

	// Sort by count descending
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	// Return top 20
	if len(stats) > 20 {
		stats = stats[:20]
	}

	return stats
}

func (s *PostService) calculateAverage(values []int) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

func (s *PostService) calculateMedian(values []int) int {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]int, len(values))
	copy(sorted, values)
	sort.Ints(sorted)

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func (s *PostService) calculateTimeDistribution(posts []models.Post) map[string]int {
	distribution := make(map[string]int)

	for _, post := range posts {
		hour := post.CreatedAt.Hour()
		var period string
		if hour < 6 {
			period = "night"
		} else if hour < 12 {
			period = "morning"
		} else if hour < 18 {
			period = "afternoon"
		} else {
			period = "evening"
		}
		distribution[period]++
	}

	return distribution
}
