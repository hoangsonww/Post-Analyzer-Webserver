package models

import (
	"time"
)

// Post represents a post in the system
type Post struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"userId" db:"user_id"`
	Title     string    `json:"title" db:"title"`
	Body      string    `json:"body" db:"body"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// CreatePostRequest represents a request to create a post
type CreatePostRequest struct {
	UserID int    `json:"userId"`
	Title  string `json:"title" validate:"required,min=1,max=500"`
	Body   string `json:"body" validate:"required,min=1,max=10000"`
}

// UpdatePostRequest represents a request to update a post
type UpdatePostRequest struct {
	UserID int    `json:"userId,omitempty"`
	Title  string `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Body   string `json:"body,omitempty" validate:"omitempty,min=1,max=10000"`
}

// PostFilter represents filtering options for posts
type PostFilter struct {
	UserID    *int   `json:"userId,omitempty"`
	Search    string `json:"search,omitempty"`
	SortBy    string `json:"sortBy,omitempty"`    // id, title, createdAt, updatedAt
	SortOrder string `json:"sortOrder,omitempty"` // asc, desc
}

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Offset   int `json:"-"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Data       interface{}      `json:"data"`
	Pagination PaginationMeta   `json:"pagination"`
	Meta       *ResponseMeta    `json:"meta,omitempty"`
}

// PaginationMeta contains pagination metadata
type PaginationMeta struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
	HasNext    bool `json:"hasNext"`
	HasPrev    bool `json:"hasPrev"`
}

// ResponseMeta contains additional response metadata
type ResponseMeta struct {
	RequestID string        `json:"requestId,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration,omitempty"`
}

// AnalyticsResult represents character frequency analysis results
type AnalyticsResult struct {
	TotalPosts      int                `json:"totalPosts"`
	TotalCharacters int                `json:"totalCharacters"`
	UniqueChars     int                `json:"uniqueChars"`
	CharFrequency   map[rune]int       `json:"charFrequency"`
	TopCharacters   []CharacterStat    `json:"topCharacters"`
	Statistics      *AnalyticsStats    `json:"statistics,omitempty"`
}

// CharacterStat represents statistics for a single character
type CharacterStat struct {
	Character rune    `json:"character"`
	Count     int     `json:"count"`
	Frequency float64 `json:"frequency"`
}

// AnalyticsStats represents overall analytics statistics
type AnalyticsStats struct {
	AveragePostLength float64            `json:"averagePostLength"`
	MedianPostLength  int                `json:"medianPostLength"`
	PostsPerUser      map[int]int        `json:"postsPerUser"`
	TimeDistribution  map[string]int     `json:"timeDistribution"`
}

// BulkCreateRequest represents a bulk create request
type BulkCreateRequest struct {
	Posts []CreatePostRequest `json:"posts" validate:"required,min=1,max=1000"`
}

// BulkCreateResponse represents a bulk create response
type BulkCreateResponse struct {
	Created  int      `json:"created"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors,omitempty"`
	PostIDs  []int    `json:"postIds,omitempty"`
}

// ExportFormat represents export file format
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatCSV  ExportFormat = "csv"
)

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version,omitempty"`
	Uptime    time.Duration     `json:"uptime,omitempty"`
	Checks    map[string]bool   `json:"checks,omitempty"`
}

// User represents a user in the system (for future auth)
type User struct {
	ID        int       `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	Username  string    `json:"username" db:"username"`
	Role      string    `json:"role" db:"role"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"userId" db:"user_id"`
	Action    string    `json:"action" db:"action"`
	Resource  string    `json:"resource" db:"resource"`
	ResourceID int      `json:"resourceId" db:"resource_id"`
	Changes   string    `json:"changes,omitempty" db:"changes"`
	IPAddress string    `json:"ipAddress" db:"ip_address"`
	UserAgent string    `json:"userAgent" db:"user_agent"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}
