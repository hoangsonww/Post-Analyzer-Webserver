package abac

import (
	"testing"
	"time"
)

func TestIssueAndParseToken_RoundTrip(t *testing.T) {
	secret := "test-secret"
	subj := Subject{
		UserID: 42, Username: "alice", Role: "editor",
		Attributes: map[string]string{"team": "infra"},
	}

	token, expiresAt, err := IssueToken(secret, time.Hour, subj)
	if err != nil {
		t.Fatalf("IssueToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty token")
	}
	if time.Until(expiresAt) <= 0 || time.Until(expiresAt) > time.Hour {
		t.Errorf("expiresAt not within expected window: %v", expiresAt)
	}

	got, err := ParseToken(secret, token)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if got.UserID != subj.UserID || got.Username != subj.Username || got.Role != subj.Role {
		t.Errorf("round-tripped subject mismatch: got %+v, want %+v", got, subj)
	}
}

func TestParseToken_WrongSecretRejected(t *testing.T) {
	token, _, err := IssueToken("secret-a", time.Hour, Subject{Username: "bob", Role: "viewer"})
	if err != nil {
		t.Fatalf("IssueToken failed: %v", err)
	}
	if _, err := ParseToken("secret-b", token); err == nil {
		t.Error("expected parsing with the wrong secret to fail")
	}
}

func TestParseToken_ExpiredRejected(t *testing.T) {
	token, _, err := IssueToken("secret", -time.Minute, Subject{Username: "carol", Role: "admin"})
	if err != nil {
		t.Fatalf("IssueToken failed: %v", err)
	}
	if _, err := ParseToken("secret", token); err == nil {
		t.Error("expected an already-expired token to fail validation")
	}
}

func TestParseToken_GarbageRejected(t *testing.T) {
	if _, err := ParseToken("secret", "not-a-jwt"); err == nil {
		t.Error("expected a malformed token string to fail parsing")
	}
}
