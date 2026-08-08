package triton

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClassifySentiment_Success(t *testing.T) {
	var gotText string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/models/post_sentiment/infer" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req inferRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		gotText = req.Inputs[0].Data[0].(string)

		// negative=0.1, neutral=0.2, positive=0.7 -> argmax is "positive"
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(inferResponse{
			Outputs: []struct {
				Name     string        `json:"name"`
				Datatype string        `json:"datatype"`
				Shape    []int         `json:"shape"`
				Data     []interface{} `json:"data"`
			}{
				{Name: "probabilities", Data: []interface{}{0.1, 0.2, 0.7}},
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL)
	result, err := c.ClassifySentiment(context.Background(), "This Is GREAT")
	if err != nil {
		t.Fatalf("ClassifySentiment failed: %v", err)
	}
	if gotText != "this is great" {
		t.Errorf("expected the request to lowercase the text, got %q", gotText)
	}
	if result.Label != "positive" {
		t.Errorf("expected label=positive (argmax of [0.1,0.2,0.7]), got %s", result.Label)
	}
	if result.Probabilities["positive"] != 0.7 {
		t.Errorf("expected probabilities[positive]=0.7, got %+v", result.Probabilities)
	}
}

func TestClassifySentiment_ArgmaxNegative(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(inferResponse{
			Outputs: []struct {
				Name     string        `json:"name"`
				Datatype string        `json:"datatype"`
				Shape    []int         `json:"shape"`
				Data     []interface{} `json:"data"`
			}{
				{Name: "probabilities", Data: []interface{}{0.8, 0.1, 0.1}},
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL)
	result, err := c.ClassifySentiment(context.Background(), "terrible")
	if err != nil {
		t.Fatalf("ClassifySentiment failed: %v", err)
	}
	if result.Label != "negative" {
		t.Errorf("expected label=negative (argmax of [0.8,0.1,0.1]), got %s", result.Label)
	}
}

func TestClassifySentiment_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	if _, err := c.ClassifySentiment(context.Background(), "x"); err == nil {
		t.Error("expected an error for a non-200 Triton response")
	}
}

func TestClassifySentiment_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	if _, err := c.ClassifySentiment(context.Background(), "x"); err == nil {
		t.Error("expected an error for a malformed JSON response")
	}
}

func TestClassifySentiment_MissingProbabilitiesOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(inferResponse{Outputs: []struct {
			Name     string        `json:"name"`
			Datatype string        `json:"datatype"`
			Shape    []int         `json:"shape"`
			Data     []interface{} `json:"data"`
		}{
			{Name: "some_other_output", Data: []interface{}{1.0}},
		}})
	}))
	defer server.Close()

	c := NewClient(server.URL)
	if _, err := c.ClassifySentiment(context.Background(), "x"); err == nil {
		t.Error("expected an error when the response has no probabilities output")
	}
}

func TestClassifySentiment_ConnectionError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1")
	if _, err := c.ClassifySentiment(context.Background(), "x"); err == nil {
		t.Error("expected an error when the server is unreachable")
	}
}

func TestReady_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/models/post_sentiment/ready" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	if err := c.Ready(context.Background()); err != nil {
		t.Errorf("expected Ready to succeed, got %v", err)
	}
}

func TestReady_NotReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	if err := c.Ready(context.Background()); err == nil {
		t.Error("expected an error when the model isn't ready")
	}
}
