package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"lastsaas/internal/db"
	"lastsaas/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// StreamChatRequest represents a streaming request to the chat completions API.
type StreamChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// StreamChunk represents a single chunk in a streaming response.
type StreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// ChatRequest represents a request to the chat completions API.
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// ChatResponse represents a response from the chat completions API.
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int      `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// RequestConfig is a fixed per-request LLM configuration snapshot.
type RequestConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// Choice represents a single choice in the response.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage information.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Client is an OpenAI-compatible Chat Completions client.
type Client struct {
	APIKey  string
	BaseURL string
	Model   string
	db      *db.MongoDB
	mu      sync.RWMutex
}

// NewClient creates a new LLM client with environment variable configuration.
func NewClient() *Client {
	return &Client{}
}

// NewClientWithDB creates a client that loads config from database.
func NewClientWithDB(database *db.MongoDB) *Client {
	c := &Client{db: database}
	c.reloadConfig()
	return c
}

// reloadConfig loads configuration from database
func (c *Client) reloadConfig() {
	if c.db == nil {
		return
	}

	ctx := context.Background()
	config := models.LLMConfig{}
	err := c.db.LLMConfigs().FindOne(ctx, bson.M{"key": models.DefaultLLMConfigKey, "isActive": true}).Decode(&config)
	if err != nil {
		return
	}

	c.mu.Lock()
	c.APIKey = config.APIKey
	c.BaseURL = normalizeBaseURL(config.BaseURL)
	c.Model = config.Model
	c.mu.Unlock()
}

// ReloadConfig triggers a reload from database
func (c *Client) ReloadConfig() {
	c.reloadConfig()
}

func normalizeBaseURL(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return ""
	}
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed
	}
	return trimmed + "/v1"
}

// SnapshotConfig captures the current client configuration for a single request.
func (c *Client) SnapshotConfig() RequestConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return RequestConfig{
		APIKey:  c.APIKey,
		BaseURL: normalizeBaseURL(c.BaseURL),
		Model:   c.Model,
	}
}

// Complete sends a chat completion request and returns the assistant's response.
func (c *Client) Complete(ctx context.Context, systemPrompt string, messages []Message) (string, error) {
	answer, _, err := c.CompleteWithConfig(ctx, c.SnapshotConfig(), systemPrompt, messages)
	return answer, err
}

// CompleteWithConfig sends a chat completion request using a fixed per-request configuration snapshot.
func (c *Client) CompleteWithConfig(ctx context.Context, config RequestConfig, systemPrompt string, messages []Message) (string, string, error) {
	apiKey := strings.TrimSpace(config.APIKey)
	baseURL := normalizeBaseURL(config.BaseURL)
	model := strings.TrimSpace(config.Model)

	if apiKey == "" {
		return "", "", fmt.Errorf("OPENAI_API_KEY is not set")
	}
	if baseURL == "" {
		return "", "", fmt.Errorf("OPENAI_BASE_URL is not set")
	}
	if model == "" {
		return "", "", fmt.Errorf("OPENAI_MODEL is not set")
	}

	fullMessages := []Message{
		{Role: "system", Content: systemPrompt},
	}
	fullMessages = append(fullMessages, messages...)

	reqBody := ChatRequest{
		Model:    model,
		Messages: fullMessages,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", "", fmt.Errorf("no response choices returned")
	}

	return chatResp.Choices[0].Message.Content, model, nil
}

// CompleteStreamWithConfig sends a streaming chat completion request and calls onDelta for each chunk.
// Returns the full response content, the model used, and any error encountered.
func (c *Client) CompleteStreamWithConfig(ctx context.Context, config RequestConfig, systemPrompt string, messages []Message, onDelta func(string) error) (string, string, error) {
	apiKey := strings.TrimSpace(config.APIKey)
	baseURL := normalizeBaseURL(config.BaseURL)
	model := strings.TrimSpace(config.Model)

	if apiKey == "" {
		return "", "", fmt.Errorf("OPENAI_API_KEY is not set")
	}
	if baseURL == "" {
		return "", "", fmt.Errorf("OPENAI_BASE_URL is not set")
	}
	if model == "" {
		return "", "", fmt.Errorf("OPENAI_MODEL is not set")
	}

	fullMessages := []Message{
		{Role: "system", Content: systemPrompt},
	}
	fullMessages = append(fullMessages, messages...)

	reqBody := StreamChatRequest{
		Model:    model,
		Messages: fullMessages,
		Stream:   true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "text/event-stream")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	// Create a bufio.Scanner to read line by line
	// Use a custom split function to handle the large response bodies
	scanner := bufio.NewScanner(resp.Body)
	// Increase buffer size for large responses
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var fullContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse SSE format: "data: {...}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for [DONE] sentinel
		if data == "[DONE]" {
			break
		}

		// Parse the JSON chunk
		var chunk StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// Skip invalid JSON but continue processing
			continue
		}

		// Extract content delta
		if len(chunk.Choices) > 0 && len(chunk.Choices[0].Delta.Content) > 0 {
			content := chunk.Choices[0].Delta.Content
			fullContent.WriteString(content)

			// Call the onDelta callback
			if err := onDelta(content); err != nil {
				// Return the partial content and the error
				return fullContent.String(), model, err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("error reading stream: %w", err)
	}

	return fullContent.String(), model, nil
}
