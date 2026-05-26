package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient_DefaultsToEmptyConfiguration(t *testing.T) {
	client := NewClient()

	if client.APIKey != "" {
		t.Fatalf("expected no default api key, got %q", client.APIKey)
	}
	if client.BaseURL != "" {
		t.Fatalf("expected no default base URL, got %q", client.BaseURL)
	}
	if client.Model != "" {
		t.Fatalf("expected no default model, got %q", client.Model)
	}
}

func TestSnapshotConfig_NormalizesBaseURLAndCopiesValues(t *testing.T) {
	client := &Client{
		APIKey:  "test-key",
		BaseURL: "https://example.com/custom/",
		Model:   "test-model",
	}

	config := client.SnapshotConfig()

	if config.APIKey != "test-key" {
		t.Fatalf("expected api key to be copied, got %q", config.APIKey)
	}
	if config.Model != "test-model" {
		t.Fatalf("expected model to be copied, got %q", config.Model)
	}
	if config.BaseURL != "https://example.com/custom/v1" {
		t.Fatalf("expected normalized base URL, got %q", config.BaseURL)
	}
}

func TestSnapshotConfig_DoesNotDuplicateV1WithTrailingSlash(t *testing.T) {
	client := &Client{
		APIKey:  "test-key",
		BaseURL: "https://api.openai.com/v1/",
		Model:   "test-model",
	}

	config := client.SnapshotConfig()

	if config.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("expected base URL to keep a single /v1 suffix, got %q", config.BaseURL)
	}
}

func TestCompleteWithConfig_UsesProvidedRequestConfig(t *testing.T) {
	type capturedRequest struct {
		Authorization string
		Path          string
		Model         string
	}

	captured := capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.Authorization = r.Header.Get("Authorization")
		captured.Path = r.URL.Path

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		captured.Model = req.Model

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"configured answer"}}]}`))
	}))
	defer server.Close()

	client := &Client{
		APIKey:  "mutable-key",
		BaseURL: "https://should-not-be-used.example.com/v1",
		Model:   "mutable-model",
	}

	answer, usedModel, err := client.CompleteWithConfig(context.Background(), RequestConfig{
		APIKey:  "fixed-key",
		BaseURL: server.URL,
		Model:   "fixed-model",
	}, "system prompt", []Message{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("complete with config: %v", err)
	}

	if answer != "configured answer" {
		t.Fatalf("expected configured answer, got %q", answer)
	}
	if usedModel != "fixed-model" {
		t.Fatalf("expected used model to be fixed-model, got %q", usedModel)
	}
	if captured.Authorization != "Bearer fixed-key" {
		t.Fatalf("expected provided api key to be used, got %q", captured.Authorization)
	}
	if captured.Path != "/v1/chat/completions" {
		t.Fatalf("expected request to use normalized v1 path, got %q", captured.Path)
	}
	if captured.Model != "fixed-model" {
		t.Fatalf("expected request body to use fixed model, got %q", captured.Model)
	}
}
