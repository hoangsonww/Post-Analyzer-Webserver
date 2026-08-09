package rpcclient

import (
	"errors"
	"net/http"
	"testing"

	apperrors "Post_Analyzer_Webserver/internal/errors"
	basegen "Post_Analyzer_Webserver/kitex_gen/base"
)

func TestRpcErr_NilBaseRespAndNoTransportError(t *testing.T) {
	if err := rpcErr(nil, nil); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRpcErr_ZeroStatusCodeIsSuccess(t *testing.T) {
	if err := rpcErr(nil, &basegen.BaseResp{StatusCode: 0}); err != nil {
		t.Errorf("expected no error for StatusCode 0, got %v", err)
	}
}

func TestRpcErr_TransportErrorIsGenericInternal(t *testing.T) {
	err := rpcErr(errors.New("dial tcp: connection refused"), nil)
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("expected *errors.AppError, got %T", err)
	}
	if appErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", appErr.StatusCode)
	}
	if appErr.Message != "Internal server error" {
		t.Errorf("expected the generic message (no leaking transport details), got %q", appErr.Message)
	}
}

// TestRpcErr_BusinessErrorPreservesStatusAndMessage is the direct
// regression guard for the bug where every postsvc-side business error
// ("post not found", a validation failure, ...) was flattened into a
// generic 500 "Internal server error" for every API/dashboard/CLI
// caller — see internal/adapt.Err, which is what actually produces this
// BaseResp shape in production.
func TestRpcErr_BusinessErrorPreservesStatusAndMessage(t *testing.T) {
	resp := &basegen.BaseResp{
		StatusCode:    http.StatusNotFound,
		StatusMessage: "Post not found",
		Extra:         map[string]string{"code": "NOT_FOUND"},
	}
	err := rpcErr(nil, resp)
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("expected *errors.AppError, got %T", err)
	}
	if appErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", appErr.StatusCode)
	}
	if appErr.Message != "Post not found" {
		t.Errorf("expected the specific message to survive, got %q", appErr.Message)
	}
	if appErr.Code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %q", appErr.Code)
	}
}

func TestRpcErr_BusinessErrorWithoutExtraCodeDefaults(t *testing.T) {
	resp := &basegen.BaseResp{StatusCode: http.StatusConflict, StatusMessage: "already exists"}
	err := rpcErr(nil, resp)
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("expected *errors.AppError, got %T", err)
	}
	if appErr.Code != "ERROR" {
		t.Errorf("expected the default code, got %q", appErr.Code)
	}
	if appErr.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", appErr.StatusCode)
	}
}

func TestRpcErr_ServerErrorStatusCodeIsGenericAndSafe(t *testing.T) {
	resp := &basegen.BaseResp{StatusCode: http.StatusInternalServerError, StatusMessage: "sql: connection reset by peer"}
	err := rpcErr(nil, resp)
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("expected *errors.AppError, got %T", err)
	}
	if appErr.Message != "Internal server error" {
		t.Errorf("expected the raw postsvc message to be hidden, got %q", appErr.Message)
	}
}

func TestRpcErr_OutOfRangeStatusCodeFallsBackToGeneric(t *testing.T) {
	resp := &basegen.BaseResp{StatusCode: 999, StatusMessage: "weird"}
	err := rpcErr(nil, resp)
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("expected *errors.AppError, got %T", err)
	}
	if appErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected an out-of-range status code to fall back to 500, got %d", appErr.StatusCode)
	}
}
