package rpcclient

import (
	"context"
	"fmt"
	"net/http"

	"Post_Analyzer_Webserver/internal/abac"
	"Post_Analyzer_Webserver/internal/errors"

	auth "Post_Analyzer_Webserver/kitex_gen/auth"
	authservice "Post_Analyzer_Webserver/kitex_gen/auth/authservice"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/transport"
)

// AuthClient is the gateway-facing view of authsvc: token issuance plus
// the ABAC policy decision point.
type AuthClient interface {
	Login(ctx context.Context, username, password string) (token string, subj abac.Subject, err error)
	ValidateToken(ctx context.Context, token string) (abac.Subject, error)
	Authorize(ctx context.Context, req abac.Request) (abac.Decision, error)
}

type authRPCClient struct {
	cli authservice.Client
}

func NewAuthClient(addr string, mux bool) (AuthClient, error) {
	opts := []client.Option{
		client.WithHostPorts(addr),
		client.WithTransportProtocol(transport.TTHeader),
	}
	if mux {
		opts = append(opts, client.WithMuxConnection(1))
	}
	cli, err := authservice.NewClient("authsvc", opts...)
	if err != nil {
		return nil, fmt.Errorf("dial authsvc at %s: %w", addr, err)
	}
	return &authRPCClient{cli: cli}, nil
}

func toThriftSubject(s abac.Subject) *auth.Subject {
	return &auth.Subject{UserId: int64(s.UserID), Username: s.Username, Role: s.Role, Attributes: s.Attributes}
}

func fromThriftSubject(s *auth.Subject) abac.Subject {
	if s == nil {
		return abac.Subject{}
	}
	return abac.Subject{UserID: int(s.UserId), Username: s.Username, Role: s.Role, Attributes: s.Attributes}
}

// Login returns a typed *errors.AppError (not a plain error) so the
// gateway's HTTP handler can tell "wrong username/password" (safe,
// specific, worth showing the user) apart from "authsvc is unreachable"
// (an infra problem — worth a distinct status code, but the transport
// details shouldn't leak into the response body).
func (c *authRPCClient) Login(ctx context.Context, username, password string) (string, abac.Subject, error) {
	resp, err := c.cli.Login(ctx, &auth.LoginRequest{Base: newBase(), Username: username, Password: password})
	if err != nil {
		appErr := errors.New("SERVICE_UNAVAILABLE", "authentication service unavailable", http.StatusServiceUnavailable)
		appErr.Internal = fmt.Errorf("authsvc rpc: %w", err) // logged server-side only (json:"-"), never sent to the client
		return "", abac.Subject{}, appErr
	}
	if resp.BaseResp != nil && resp.BaseResp.StatusCode != 0 {
		msg := resp.BaseResp.StatusMessage
		if msg == "" {
			msg = "invalid username or password"
		}
		return "", abac.Subject{}, errors.New("UNAUTHORIZED", msg, http.StatusUnauthorized)
	}
	return resp.Token, fromThriftSubject(resp.Subject), nil
}

func (c *authRPCClient) ValidateToken(ctx context.Context, token string) (abac.Subject, error) {
	resp, err := c.cli.ValidateToken(ctx, &auth.ValidateTokenRequest{Base: newBase(), Token: token})
	if err != nil {
		return abac.Subject{}, fmt.Errorf("authsvc rpc: %w", err)
	}
	if !resp.Valid {
		msg := "invalid token"
		if resp.BaseResp != nil && resp.BaseResp.StatusMessage != "" {
			msg = resp.BaseResp.StatusMessage
		}
		return abac.Subject{}, fmt.Errorf("%s", msg)
	}
	return fromThriftSubject(resp.Subject), nil
}

func (c *authRPCClient) Authorize(ctx context.Context, req abac.Request) (abac.Decision, error) {
	resp, err := c.cli.Authorize(ctx, &auth.AuthorizeRequest{
		Base:               newBase(),
		Subject:            toThriftSubject(req.Subject),
		Resource:           req.Resource,
		Action:             req.Action,
		ResourceAttributes: req.ResourceAttributes,
		Context:            req.Context,
	})
	if err != nil {
		return abac.Decision{}, fmt.Errorf("authsvc rpc: %w", err)
	}
	if resp.BaseResp != nil && resp.BaseResp.StatusCode != 0 {
		return abac.Decision{}, fmt.Errorf("authsvc: %s", resp.BaseResp.StatusMessage)
	}
	return abac.Decision{Allowed: resp.Allowed, Reason: resp.Reason, MatchedPolicy: resp.MatchedPolicy}, nil
}
