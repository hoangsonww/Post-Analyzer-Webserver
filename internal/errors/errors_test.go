package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	e := &AppError{Message: "not found"}
	if got := e.Error(); got != "not found" {
		t.Errorf("Error() = %q, want %q", got, "not found")
	}

	wrapped := &AppError{Message: "db failed", Internal: errors.New("connection refused")}
	if got := wrapped.Error(); got != "db failed: connection refused" {
		t.Errorf("Error() = %q", got)
	}
}

func TestAppError_WithField(t *testing.T) {
	e := New("VALIDATION_FAILED", "bad input", http.StatusBadRequest)
	_ = e.WithField("title", "is required").WithField("body", "is too long")

	if e.Fields["title"] != "is required" || e.Fields["body"] != "is too long" {
		t.Errorf("unexpected fields: %+v", e.Fields)
	}
}

func TestWithField_InitializesNilMap(t *testing.T) {
	e := &AppError{}
	if e.Fields != nil {
		t.Fatal("expected a fresh AppError to start with a nil Fields map")
	}
	_ = e.WithField("x", "y")
	if e.Fields["x"] != "y" {
		t.Error("expected WithField to lazily initialize the map")
	}
}

func TestPredefinedErrors_HaveDistinctCodesAndStatuses(t *testing.T) {
	predefined := []*AppError{
		ErrNotFound, ErrInvalidInput, ErrUnauthorized, ErrForbidden,
		ErrConflict, ErrInternal, ErrDatabaseError, ErrValidationFailed,
		ErrRateLimitExceeded, ErrServiceUnavailable,
	}
	seen := make(map[string]bool)
	for _, e := range predefined {
		if e.Code == "" {
			t.Errorf("predefined error has an empty Code: %+v", e)
		}
		if e.StatusCode == 0 {
			t.Errorf("predefined error has a zero StatusCode: %+v", e)
		}
		if seen[e.Code] {
			t.Errorf("duplicate error code: %s", e.Code)
		}
		seen[e.Code] = true
	}
}

func TestNew(t *testing.T) {
	e := New("CUSTOM", "custom message", http.StatusTeapot)
	if e.Code != "CUSTOM" || e.Message != "custom message" || e.StatusCode != http.StatusTeapot {
		t.Errorf("unexpected AppError: %+v", e)
	}
}

func TestWrap_AppError(t *testing.T) {
	original := &AppError{Code: "NOT_FOUND", Message: "orig", StatusCode: http.StatusNotFound, Fields: map[string]string{"a": "b"}}
	wrapped := Wrap(original, "new message")

	if wrapped.Code != "NOT_FOUND" {
		t.Errorf("expected Wrap to preserve the original Code, got %s", wrapped.Code)
	}
	if wrapped.StatusCode != http.StatusNotFound {
		t.Errorf("expected Wrap to preserve the original StatusCode, got %d", wrapped.StatusCode)
	}
	if wrapped.Message != "new message" {
		t.Errorf("expected Wrap to use the new message, got %q", wrapped.Message)
	}
	if wrapped.Fields["a"] != "b" {
		t.Error("expected Wrap to preserve the original Fields")
	}
}

func TestWrap_PlainError(t *testing.T) {
	plain := errors.New("boom")
	wrapped := Wrap(plain, "context message")

	if wrapped.Code != "INTERNAL_ERROR" {
		t.Errorf("expected a plain error to wrap as INTERNAL_ERROR, got %s", wrapped.Code)
	}
	if wrapped.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", wrapped.StatusCode)
	}
	if wrapped.Internal != plain {
		t.Error("expected the original error to be preserved as Internal")
	}
}

func TestNewNotFound(t *testing.T) {
	e := NewNotFound("post")
	if e.Message != "post not found" {
		t.Errorf("unexpected message: %s", e.Message)
	}
	if e.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", e.StatusCode)
	}
}

func TestNewValidationError(t *testing.T) {
	e := NewValidationError("title is required")
	if e.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", e.StatusCode)
	}
	if e.Fields == nil {
		t.Error("expected NewValidationError to initialize a non-nil Fields map")
	}
}

func TestNewInternalError(t *testing.T) {
	cause := errors.New("disk full")
	e := NewInternalError(cause)
	if e.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", e.StatusCode)
	}
	if e.Internal != cause {
		t.Error("expected the original cause to be preserved")
	}
}
