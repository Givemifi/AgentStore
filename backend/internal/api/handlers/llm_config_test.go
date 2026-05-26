package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"lastsaas/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestUpdateLLMConfig_RejectsActiveConfigWithoutRequiredFields(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()

	handler := NewLLMConfigHandler(env.DB)
	req := httptest.NewRequest(http.MethodPut, "/api/admin/llm-config", bytes.NewReader([]byte(`{"apiKey":"","baseURL":"","model":"","isActive":true}`)))
	rr := httptest.NewRecorder()

	handler.UpdateConfig(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}

	count, err := env.DB.LLMConfigs().CountDocuments(context.Background(), bson.M{})
	if err != nil {
		t.Fatalf("count configs: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected invalid active config not to be saved, got %d configs", count)
	}
}

func TestLLMConfig_UpsertsSingletonAndPreservesExistingAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	handler := NewLLMConfigHandler(env.DB)
	ctx := context.Background()
	now := time.Now()

	_, err := env.DB.LLMConfigs().InsertOne(ctx, models.LLMConfig{
		ID:        primitive.NewObjectID(),
		Key:       models.DefaultLLMConfigKey,
		APIKey:    "old-key",
		BaseURL:   "https://old.example.com/v1",
		Model:     "old-model",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed existing config: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/admin/llm-config", bytes.NewReader([]byte(`{"apiKey":"********","baseURL":"https://new.example.com/v1","model":"new-model","isActive":true}`)))
	rr := httptest.NewRecorder()

	handler.UpdateConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	count, err := env.DB.LLMConfigs().CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatalf("count configs: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected singleton config, got %d configs", count)
	}

	config := GetLLMConfig(ctx, env.DB)
	if config == nil {
		t.Fatal("expected active config")
	}
	if config.APIKey != "old-key" {
		t.Fatalf("expected existing API key to be preserved, got %q", config.APIKey)
	}
	if config.BaseURL != "https://new.example.com/v1" || config.Model != "new-model" {
		t.Fatalf("expected config fields to update, got baseURL=%q model=%q", config.BaseURL, config.Model)
	}
}

func TestGetLLMConfig_RedactsAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	handler := NewLLMConfigHandler(env.DB)
	ctx := context.Background()
	now := time.Now()

	_, err := env.DB.LLMConfigs().InsertOne(ctx, models.LLMConfig{
		ID:        primitive.NewObjectID(),
		Key:       models.DefaultLLMConfigKey,
		APIKey:    "secret-key",
		BaseURL:   "https://api.example.com/v1",
		Model:     "model",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("seed config: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.GetConfig(rr, httptest.NewRequest(http.MethodGet, "/api/admin/llm-config", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp models.LLMConfig
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.APIKey != "" {
		t.Fatalf("expected API key to be redacted, got %q", resp.APIKey)
	}
}
