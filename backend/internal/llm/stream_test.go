package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCompleteStreamWithConfig_BasicStreaming(t *testing.T) {
	// Create a mock server that returns SSE stream
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		// Simulate streaming response
		chunks := []string{
			`{"choices":[{"delta":{"content":"Hello"}}]}`,
			`{"choices":[{"delta":{"content":" there"}}]}`,
			`{"choices":[{"delta":{"content":"!"}}]}`,
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			w.(http.Flusher).Flush()
			time.Sleep(10 * time.Millisecond)
		}

		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	client := &Client{}

	// Parse the base URL from the test server
	baseURL := strings.TrimSuffix(server.URL, "/")

	config := RequestConfig{
		APIKey:  "test-key",
		BaseURL: baseURL,
		Model:   "gpt-4",
	}

	var receivedDeltas []string
	onDelta := func(content string) error {
		receivedDeltas = append(receivedDeltas, content)
		return nil
	}

	ctx := context.Background()
	result, model, err := client.CompleteStreamWithConfig(ctx, config, "You are a helpful assistant.", []Message{
		{Role: "user", Content: "Hi"},
	}, onDelta)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedContent := "Hello there!"
	if result != expectedContent {
		t.Errorf("expected %q, got %q", expectedContent, result)
	}

	if model != "gpt-4" {
		t.Errorf("expected model gpt-4, got %q", model)
	}

	if len(receivedDeltas) != 3 {
		t.Errorf("expected 3 deltas, got %d", len(receivedDeltas))
	}

	expectedDeltas := []string{"Hello", " there", "!"}
	for i, expected := range expectedDeltas {
		if receivedDeltas[i] != expected {
			t.Errorf("delta %d: expected %q, got %q", i, expected, receivedDeltas[i])
		}
	}
}

func TestCompleteStreamWithConfig_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	client := &Client{}
	baseURL := strings.TrimSuffix(server.URL, "/")

	config := RequestConfig{
		APIKey:  "test-key",
		BaseURL: baseURL,
		Model:   "gpt-4",
	}

	onDelta := func(content string) error {
		return nil
	}

	ctx := context.Background()
	result, _, err := client.CompleteStreamWithConfig(ctx, config, "You are helpful.", []Message{
		{Role: "user", Content: "Hi"},
	}, onDelta)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestCompleteStreamWithConfig_MissingAPIKey(t *testing.T) {
	client := &Client{}

	config := RequestConfig{
		APIKey:  "",
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4",
	}

	onDelta := func(content string) error {
		return nil
	}

	ctx := context.Background()
	_, _, err := client.CompleteStreamWithConfig(ctx, config, "You are helpful.", []Message{}, onDelta)

	if err == nil {
		t.Fatal("expected error for missing API key")
	}

	if err.Error() != "OPENAI_API_KEY is not set" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCompleteStreamWithConfig_MissingBaseURL(t *testing.T) {
	client := &Client{}

	config := RequestConfig{
		APIKey:  "test-key",
		BaseURL: "",
		Model:   "gpt-4",
	}

	onDelta := func(content string) error {
		return nil
	}

	ctx := context.Background()
	_, _, err := client.CompleteStreamWithConfig(ctx, config, "You are helpful.", []Message{}, onDelta)

	if err == nil {
		t.Fatal("expected error for missing base URL")
	}

	if err.Error() != "OPENAI_BASE_URL is not set" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCompleteStreamWithConfig_MissingModel(t *testing.T) {
	client := &Client{}

	config := RequestConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.openai.com/v1",
		Model:   "",
	}

	onDelta := func(content string) error {
		return nil
	}

	ctx := context.Background()
	_, _, err := client.CompleteStreamWithConfig(ctx, config, "You are helpful.", []Message{}, onDelta)

	if err == nil {
		t.Fatal("expected error for missing model")
	}

	if err.Error() != "OPENAI_MODEL is not set" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCompleteStreamWithConfig_ErrorStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &Client{}
	baseURL := strings.TrimSuffix(server.URL, "/")

	config := RequestConfig{
		APIKey:  "test-key",
		BaseURL: baseURL,
		Model:   "gpt-4",
	}

	onDelta := func(content string) error {
		return nil
	}

	ctx := context.Background()
	_, _, err := client.CompleteStreamWithConfig(ctx, config, "You are helpful.", []Message{
		{Role: "user", Content: "Hi"},
	}, onDelta)

	if err == nil {
		t.Fatal("expected error for server error status")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error with status 500, got: %v", err)
	}
}

func TestCompleteStreamWithConfig_DeltaCallbackError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		chunks := []string{
			`{"choices":[{"delta":{"content":"Hello"}}]}`,
			`{"choices":[{"delta":{"content":" there"}}]}`,
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			w.(http.Flusher).Flush()
		}
	}))
	defer server.Close()

	client := &Client{}
	baseURL := strings.TrimSuffix(server.URL, "/")

	config := RequestConfig{
		APIKey:  "test-key",
		BaseURL: baseURL,
		Model:   "gpt-4",
	}

	var receivedDeltas []string
	onDelta := func(content string) error {
		receivedDeltas = append(receivedDeltas, content)
		// Return error after receiving first content
		if content == "Hello" {
			return fmt.Errorf("callback error")
		}
		return nil
	}

	ctx := context.Background()
	result, _, err := client.CompleteStreamWithConfig(ctx, config, "You are helpful.", []Message{
		{Role: "user", Content: "Hi"},
	}, onDelta)

	// We expect an error from the callback
	if err == nil {
		t.Fatal("expected error from callback")
	}

	// The result should contain what was accumulated before the error
	if !strings.Contains(result, "Hello") {
		t.Errorf("expected result to contain Hello, got %q", result)
	}

	// We should have received at least one delta
	if len(receivedDeltas) < 1 {
		t.Errorf("expected at least 1 delta, got %d", len(receivedDeltas))
	}
}

func TestStreamChunk_JSONUnmarshal(t *testing.T) {
	data := `{"choices":[{"delta":{"content":"Test content"}}]}`
	var chunk StreamChunk
	err := json.Unmarshal([]byte(data), &chunk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunk.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(chunk.Choices))
	}

	if chunk.Choices[0].Delta.Content != "Test content" {
		t.Errorf("expected content 'Test content', got %q", chunk.Choices[0].Delta.Content)
	}
}

func TestStreamChatRequest_JSONMarshal(t *testing.T) {
	req := StreamChatRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello"},
		},
		Stream: true,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["model"] != "gpt-4" {
		t.Errorf("expected model gpt-4, got %v", result["model"])
	}

	if result["stream"] != true {
		t.Errorf("expected stream true, got %v", result["stream"])
	}
}