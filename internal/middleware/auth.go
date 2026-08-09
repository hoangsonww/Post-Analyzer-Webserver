package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"Post_Analyzer_Webserver/internal/abac"
	"Post_Analyzer_Webserver/internal/logger"
	"Post_Analyzer_Webserver/internal/rpcclient"
)

type ctxKey string

// SubjectContextKey is the context key handlers can use to read the
// authenticated ABAC subject after the ABAC middleware has run.
const SubjectContextKey ctxKey = "abac_subject"

// ABAC enforces authentication (JWT via authsvc) and authorization (ABAC
// decision via authsvc) on every request it wraps. resource is the fixed
// ABAC resource name for this route group (e.g. "post"); actionFor maps
// the concrete request to an ABAC action (e.g. GET -> "read").
func ABAC(authz rpcclient.AuthClient, resource string, actionFor func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				respondAuthError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}

			subj, err := authz.ValidateToken(r.Context(), token)
			if err != nil {
				logger.WarnContext(r.Context(), "token validation failed", "error", err)
				respondAuthError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			action := actionFor(r)
			decision, err := authz.Authorize(r.Context(), abac.Request{
				Subject:  subj,
				Resource: resource,
				Action:   action,
				Context: map[string]string{
					"mfa": r.Header.Get("X-MFA-Verified"),
				},
			})
			if err != nil {
				logger.ErrorContext(r.Context(), "authorization check failed", "error", err)
				respondAuthError(w, http.StatusForbidden, "authorization check failed")
				return
			}
			if !decision.Allowed {
				logger.WarnContext(r.Context(), "access denied",
					"user", subj.Username, "role", subj.Role,
					"resource", resource, "action", action, "reason", decision.Reason,
				)
				respondAuthError(w, http.StatusForbidden, decision.Reason)
				return
			}

			ctx := context.WithValue(r.Context(), SubjectContextKey, subj)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ActionByMethod is the default HTTP-method -> ABAC-action mapping used
// for the post CRUD API: reads are "read", creates/updates are "write",
// deletes are "delete".
func ActionByMethod(r *http.Request) string {
	switch r.Method {
	case http.MethodGet:
		return "read"
	case http.MethodDelete:
		return "delete"
	default:
		return "write"
	}
}

// SubjectFromContext returns the authenticated subject a downstream
// handler can use for owner checks, auditing, etc.
func SubjectFromContext(ctx context.Context) (abac.Subject, bool) {
	subj, ok := ctx.Value(SubjectContextKey).(abac.Subject)
	return subj, ok
}

func extractBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, prefix))
}

func respondAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{"message": message},
	})
}
