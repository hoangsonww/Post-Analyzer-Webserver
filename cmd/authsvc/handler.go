package main

import (
	auth "Post_Analyzer_Webserver/kitex_gen/auth"
	"context"
)

// AuthServiceImpl implements the last service interface defined in the IDL.
type AuthServiceImpl struct{}

// Login implements the AuthServiceImpl interface.
func (s *AuthServiceImpl) Login(ctx context.Context, req *auth.LoginRequest) (resp *auth.LoginResponse, err error) {
	// TODO: Your code here...
	return
}

// ValidateToken implements the AuthServiceImpl interface.
func (s *AuthServiceImpl) ValidateToken(ctx context.Context, req *auth.ValidateTokenRequest) (resp *auth.ValidateTokenResponse, err error) {
	// TODO: Your code here...
	return
}

// Authorize implements the AuthServiceImpl interface.
func (s *AuthServiceImpl) Authorize(ctx context.Context, req *auth.AuthorizeRequest) (resp *auth.AuthorizeResponse, err error) {
	// TODO: Your code here...
	return
}
