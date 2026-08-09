// Package triton is a minimal client for Nvidia Triton Inference
// Server's HTTP KServe v2 protocol, used to call the post_sentiment
// model (deployments/triton/model_repository) — a small TF-IDF +
// logistic-regression classifier exported to ONNX and served by Triton
// in CPU mode. It is deliberately narrow (one model, one shape) rather
// than a general-purpose Triton SDK, since that's all this repo needs.
package triton

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const ModelName = "post_sentiment"

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

type inferRequest struct {
	Inputs  []inferTensor `json:"inputs"`
	Outputs []outputSpec  `json:"outputs"`
}

type inferTensor struct {
	Name     string        `json:"name"`
	Shape    []int         `json:"shape"`
	Datatype string        `json:"datatype"`
	Data     []interface{} `json:"data"`
}

type outputSpec struct {
	Name string `json:"name"`
}

type inferResponse struct {
	Outputs []struct {
		Name     string        `json:"name"`
		Datatype string        `json:"datatype"`
		Shape    []int         `json:"shape"`
		Data     []interface{} `json:"data"`
	} `json:"outputs"`
}

// SentimentResult is the post_sentiment model's prediction for one text.
type SentimentResult struct {
	Label         string             // "positive" | "negative" | "neutral"
	Probabilities map[string]float64 // label -> probability, sums to ~1.0
}

// classOrder mirrors sklearn's alphabetically-sorted LogisticRegression
// classes_ for this training set (see train_sentiment_model.py) — the
// "probabilities" output tensor's columns come back in this order.
var classOrder = []string{"negative", "neutral", "positive"}

// ClassifySentiment sends text to the post_sentiment model and returns
// its predicted label and per-class probabilities. Lowercasing happens
// here rather than in the ONNX graph — see train_sentiment_model.py for
// why (onnxruntime's StringNormalizer op needs a locale Triton's
// container image doesn't have).
func (c *Client) ClassifySentiment(ctx context.Context, text string) (*SentimentResult, error) {
	reqBody := inferRequest{
		Inputs: []inferTensor{
			{
				Name:     "input",
				Shape:    []int{1, 1},
				Datatype: "BYTES",
				Data:     []interface{}{strings.ToLower(text)},
			},
		},
		Outputs: []outputSpec{{Name: "probabilities"}},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal triton request: %w", err)
	}

	url := fmt.Sprintf("%s/v2/models/%s/infer", c.baseURL, ModelName)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("build triton request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("triton request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("triton returned status %d", resp.StatusCode)
	}

	var out inferResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode triton response: %w", err)
	}

	result := &SentimentResult{Probabilities: make(map[string]float64, len(classOrder))}
	found := false
	for _, o := range out.Outputs {
		if o.Name != "probabilities" {
			continue
		}
		found = true
		bestIdx, bestVal := -1, -1.0
		for i, v := range o.Data {
			if i >= len(classOrder) {
				break
			}
			f, ok := v.(float64)
			if !ok {
				continue
			}
			result.Probabilities[classOrder[i]] = f
			if f > bestVal {
				bestVal, bestIdx = f, i
			}
		}
		if bestIdx >= 0 {
			result.Label = classOrder[bestIdx]
		}
	}
	if !found || result.Label == "" {
		return nil, fmt.Errorf("triton response missing probabilities output")
	}
	return result, nil
}

// Ready checks Triton's per-model readiness endpoint.
func (c *Client) Ready(ctx context.Context) error {
	url := fmt.Sprintf("%s/v2/models/%s/ready", c.baseURL, ModelName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("model not ready: status %d", resp.StatusCode)
	}
	return nil
}
