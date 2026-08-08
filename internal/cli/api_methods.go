package cli

import "net/http"

// These mirror the gateway's JSON response shapes just enough for CLI
// display purposes — not full internal/models reuse, since the CLI only
// ever needs to print them, not operate on them as domain objects.

type Post struct {
	ID        int    `json:"id"`
	UserID    int    `json:"userId"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type dataEnvelope[T any] struct {
	Data T `json:"data"`
}

// Login exchanges credentials for a JWT via POST /api/v1/auth/login.
func (c *Client) Login(username, password string) (token, role string, err error) {
	resp, err := c.do(http.MethodPost, "/api/v1/auth/login", map[string]string{
		"username": username, "password": password,
	}, nil)
	if err != nil {
		return "", "", err
	}
	if resp.status != http.StatusOK {
		return "", "", resp.err()
	}
	var out dataEnvelope[struct {
		Token    string `json:"token"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}]
	if err := resp.decode(&out); err != nil {
		return "", "", err
	}
	return out.Data.Token, out.Data.Role, nil
}

func (c *Client) ListPosts() ([]Post, error) {
	resp, err := c.do(http.MethodGet, "/api/v1/posts?pageSize=100", nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.status != http.StatusOK {
		return nil, resp.err()
	}
	var out dataEnvelope[[]Post]
	if err := resp.decode(&out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func (c *Client) GetPost(id string) (*Post, error) {
	resp, err := c.do(http.MethodGet, "/api/v1/posts/"+id, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.status != http.StatusOK {
		return nil, resp.err()
	}
	var out dataEnvelope[Post]
	if err := resp.decode(&out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

func (c *Client) CreatePost(userID int, title, body string) (*Post, error) {
	resp, err := c.do(http.MethodPost, "/api/v1/posts", map[string]interface{}{
		"userId": userID, "title": title, "body": body,
	}, nil)
	if err != nil {
		return nil, err
	}
	if resp.status != http.StatusCreated {
		return nil, resp.err()
	}
	var out dataEnvelope[Post]
	if err := resp.decode(&out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

func (c *Client) UpdatePost(id, title, body string) (*Post, error) {
	payload := map[string]interface{}{}
	if title != "" {
		payload["title"] = title
	}
	if body != "" {
		payload["body"] = body
	}
	resp, err := c.do(http.MethodPut, "/api/v1/posts/"+id, payload, nil)
	if err != nil {
		return nil, err
	}
	if resp.status != http.StatusOK {
		return nil, resp.err()
	}
	var out dataEnvelope[Post]
	if err := resp.decode(&out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

func (c *Client) DeletePost(id string, mfaVerified bool) error {
	headers := map[string]string{}
	if mfaVerified {
		headers["X-MFA-Verified"] = "true"
	}
	resp, err := c.do(http.MethodDelete, "/api/v1/posts/"+id, nil, headers)
	if err != nil {
		return err
	}
	if resp.status != http.StatusOK {
		return resp.err()
	}
	return nil
}

func (c *Client) Analyze() (map[string]interface{}, error) {
	resp, err := c.do(http.MethodGet, "/api/v1/posts/analytics", nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.status != http.StatusOK {
		return nil, resp.err()
	}
	var out dataEnvelope[map[string]interface{}]
	if err := resp.decode(&out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func (c *Client) Reanalyze() (string, error) {
	resp, err := c.do(http.MethodPost, "/api/v1/posts/reanalyze", nil, nil)
	if err != nil {
		return "", err
	}
	if resp.status != http.StatusAccepted {
		return "", resp.err()
	}
	var out dataEnvelope[struct {
		JobID  string `json:"jobId"`
		Status string `json:"status"`
	}]
	if err := resp.decode(&out); err != nil {
		return "", err
	}
	return out.Data.JobID, nil
}

func (c *Client) Sentiment(text string) (label string, probs map[string]float64, err error) {
	resp, err := c.do(http.MethodPost, "/api/v1/ml/sentiment", map[string]string{"text": text}, nil)
	if err != nil {
		return "", nil, err
	}
	if resp.status != http.StatusOK {
		return "", nil, resp.err()
	}
	var out dataEnvelope[struct {
		Label         string             `json:"label"`
		Probabilities map[string]float64 `json:"probabilities"`
	}]
	if err := resp.decode(&out); err != nil {
		return "", nil, err
	}
	return out.Data.Label, out.Data.Probabilities, nil
}

func (c *Client) ExportPosts(format string) ([]byte, error) {
	resp, err := c.do(http.MethodGet, "/api/v1/posts/export?format="+format, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.status != http.StatusOK {
		return nil, resp.err()
	}
	return resp.body, nil
}
