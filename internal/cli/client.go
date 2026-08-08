// Package cli implements the gateway binary's non-server modes: `serve`
// (the default, moved here unchanged from the old cmd/gateway/main.go),
// one-shot CLI subcommands, and an interactive REPL. The CLI/REPL modes
// are plain HTTP clients of the gateway's own REST API — they don't dial
// postsvc/authsvc directly — so they work identically whether the server
// they're talking to is this same binary or one running elsewhere.
package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client is a thin wrapper over the gateway's REST API used by every CLI
// and REPL command.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) SetToken(token string) { c.token = token }

// apiResponse is the raw result of a call: status code plus the full
// body, read exactly once so callers can decode it as either the success
// shape or the error shape without fighting over an already-drained
// http.Response.Body.
type apiResponse struct {
	status int
	body   []byte
}

func (r *apiResponse) decode(out interface{}) error {
	if len(r.body) == 0 {
		return nil
	}
	return json.Unmarshal(r.body, out)
}

func (r *apiResponse) err() error {
	var e struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(r.body, &e)
	if e.Error.Message != "" {
		return fmt.Errorf("%s (HTTP %d, %s)", e.Error.Message, r.status, e.Error.Code)
	}
	return fmt.Errorf("request failed with HTTP %d", r.status)
}

func (c *Client) do(method, path string, body interface{}, headers map[string]string) (*apiResponse, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return &apiResponse{status: resp.StatusCode, body: data}, nil
}

// --- Token persistence --------------------------------------------------
//
// A successful `post-analyzer login` writes the JWT here so subsequent
// CLI invocations (separate processes) don't need --token every time.
// REPL sessions keep the token in memory instead and only touch this file
// on an explicit `login` command.

func tokenFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".post-analyzer", "token")
}

func SaveToken(token string) error {
	path := tokenFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(token), 0o600)
}

func LoadToken() string {
	data, err := os.ReadFile(tokenFilePath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
