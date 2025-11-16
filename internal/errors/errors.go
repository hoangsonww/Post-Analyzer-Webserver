package errors

import (
	"fmt"
	"net/http"
)

// AppError represents a custom application error with HTTP context
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
	Internal   error  `json:"-"`
	Fields     map[string]string `json:"fields,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Internal)
	}
	return e.Message
}

// WithField adds a field-specific error message
func (e *AppError) WithField(field, message string) *AppError {
	if e.Fields == nil {
		e.Fields = make(map[string]string)
	}
	e.Fields[field] = message
	return e
}

// Predefined error types
var (
	// ErrNotFound indicates a resource was not found
	ErrNotFound = &AppError{
		Code:       "NOT_FOUND",
		Message:    "Resource not found",
		StatusCode: http.StatusNotFound,
	}

	// ErrInvalidInput indicates invalid input data
	ErrInvalidInput = &AppError{
		Code:       "INVALID_INPUT",
		Message:    "Invalid input data",
		StatusCode: http.StatusBadRequest,
	}

	// ErrUnauthorized indicates authentication failure
	ErrUnauthorized = &AppError{
		Code:       "UNAUTHORIZED",
		Message:    "Authentication required",
		StatusCode: http.StatusUnauthorized,
	}

	// ErrForbidden indicates insufficient permissions
	ErrForbidden = &AppError{
		Code:       "FORBIDDEN",
		Message:    "Insufficient permissions",
		StatusCode: http.StatusForbidden,
	}

	// ErrConflict indicates a conflict with existing data
	ErrConflict = &AppError{
		Code:       "CONFLICT",
		Message:    "Resource conflict",
		StatusCode: http.StatusConflict,
	}

	// ErrInternal indicates an internal server error
	ErrInternal = &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    "Internal server error",
		StatusCode: http.StatusInternalServerError,
	}

	// ErrDatabaseError indicates a database operation failure
	ErrDatabaseError = &AppError{
		Code:       "DATABASE_ERROR",
		Message:    "Database operation failed",
		StatusCode: http.StatusInternalServerError,
	}

	// ErrValidationFailed indicates validation failure
	ErrValidationFailed = &AppError{
		Code:       "VALIDATION_FAILED",
		Message:    "Validation failed",
		StatusCode: http.StatusUnprocessableEntity,
	}

	// ErrRateLimitExceeded indicates rate limit exceeded
	ErrRateLimitExceeded = &AppError{
		Code:       "RATE_LIMIT_EXCEEDED",
		Message:    "Rate limit exceeded",
		StatusCode: http.StatusTooManyRequests,
	}

	// ErrServiceUnavailable indicates service is unavailable
	ErrServiceUnavailable = &AppError{
		Code:       "SERVICE_UNAVAILABLE",
		Message:    "Service temporarily unavailable",
		StatusCode: http.StatusServiceUnavailable,
	}
)

// New creates a new AppError
func New(code, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// Wrap wraps an error with additional context
func Wrap(err error, message string) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return &AppError{
			Code:       appErr.Code,
			Message:    message,
			StatusCode: appErr.StatusCode,
			Internal:   appErr.Internal,
			Fields:     appErr.Fields,
		}
	}

	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Internal:   err,
	}
}

// NewNotFound creates a not found error
func NewNotFound(resource string) *AppError {
	return &AppError{
		Code:       "NOT_FOUND",
		Message:    fmt.Sprintf("%s not found", resource),
		StatusCode: http.StatusNotFound,
	}
}

// NewValidationError creates a validation error
func NewValidationError(message string) *AppError {
	return &AppError{
		Code:       "VALIDATION_FAILED",
		Message:    message,
		StatusCode: http.StatusUnprocessableEntity,
		Fields:     make(map[string]string),
	}
}

// NewInternalError creates an internal error
func NewInternalError(err error) *AppError {
	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    "Internal server error",
		StatusCode: http.StatusInternalServerError,
		Internal:   err,
	}
}
