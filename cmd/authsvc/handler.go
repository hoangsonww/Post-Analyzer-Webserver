package main

import (
	"context"
	"time"

	"Post_Analyzer_Webserver/internal/abac"
	authgen "Post_Analyzer_Webserver/kitex_gen/auth"
	basegen "Post_Analyzer_Webserver/kitex_gen/base"
)

// AuthServiceImpl implements the Kitex-generated auth.AuthService
// interface: token issuance (Login/ValidateToken) plus the ABAC policy
// decision point (Authorize), backed by internal/abac.
type AuthServiceImpl struct {
	users    *abac.UserStore
	policies []abac.Policy
	secret   string
	ttl      time.Duration
}

func NewAuthServiceImpl(users *abac.UserStore, secret string, ttl time.Duration) *AuthServiceImpl {
	return &AuthServiceImpl{
		users:    users,
		policies: abac.DefaultPolicies(),
		secret:   secret,
		ttl:      ttl,
	}
}

func ok() *basegen.BaseResp { return &basegen.BaseResp{StatusCode: 0, StatusMessage: "OK"} }
func fail(msg string) *basegen.BaseResp {
	return &basegen.BaseResp{StatusCode: 1, StatusMessage: msg}
}

func toThriftSubject(s abac.Subject) *authgen.Subject {
	return &authgen.Subject{
		UserId:     int64(s.UserID),
		Username:   s.Username,
		Role:       s.Role,
		Attributes: s.Attributes,
	}
}

func fromThriftSubject(s *authgen.Subject) abac.Subject {
	if s == nil {
		return abac.Subject{}
	}
	return abac.Subject{
		UserID:     int(s.UserId),
		Username:   s.Username,
		Role:       s.Role,
		Attributes: s.Attributes,
	}
}

func (s *AuthServiceImpl) Login(ctx context.Context, req *authgen.LoginRequest) (*authgen.LoginResponse, error) {
	subj, ok2 := s.users.Authenticate(req.Username, req.Password)
	if !ok2 {
		return &authgen.LoginResponse{BaseResp: fail("invalid username or password")}, nil
	}

	token, expiresAt, err := abac.IssueToken(s.secret, s.ttl, subj)
	if err != nil {
		return &authgen.LoginResponse{BaseResp: fail(err.Error())}, nil
	}

	return &authgen.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt.Unix(),
		Subject:   toThriftSubject(subj),
		BaseResp:  ok(),
	}, nil
}

func (s *AuthServiceImpl) ValidateToken(ctx context.Context, req *authgen.ValidateTokenRequest) (*authgen.ValidateTokenResponse, error) {
	subj, err := abac.ParseToken(s.secret, req.Token)
	if err != nil {
		return &authgen.ValidateTokenResponse{Valid: false, BaseResp: fail(err.Error())}, nil
	}
	return &authgen.ValidateTokenResponse{Valid: true, Subject: toThriftSubject(subj), BaseResp: ok()}, nil
}

func (s *AuthServiceImpl) Authorize(ctx context.Context, req *authgen.AuthorizeRequest) (*authgen.AuthorizeResponse, error) {
	decision := abac.Evaluate(abac.Request{
		Subject:            fromThriftSubject(req.Subject),
		Resource:           req.Resource,
		Action:             req.Action,
		ResourceAttributes: req.ResourceAttributes,
		Context:            req.Context,
	}, s.policies)

	return &authgen.AuthorizeResponse{
		Allowed:       decision.Allowed,
		Reason:        decision.Reason,
		MatchedPolicy: decision.MatchedPolicy,
		BaseResp:      ok(),
	}, nil
}
