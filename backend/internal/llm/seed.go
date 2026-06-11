// Package llm provides the OpenAI-compatible LLM client and seed helpers.
package llm

import (
	"context"
	"os"
	"strings"
	"time"

	"agentstore/internal/db"
	"agentstore/internal/models"
	"agentstore/internal/validation"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Seed creates a default LLM config document from environment variables when
// OPENAI_API_KEY, OPENAI_BASE_URL, and OPENAI_MODEL are all non-empty and no
// active default config already exists.
//
// This lets operators get a chat-ready deployment by simply setting those three
// env vars; the admin UI still takes precedence — if a config is already saved
// in the DB, this function is a no-op.
func Seed(ctx context.Context, database *db.MongoDB) error {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	baseURL := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	model := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))

	// Nothing to seed if any field is missing
	if apiKey == "" || baseURL == "" || model == "" {
		return nil
	}

	// Idempotent: skip if an active default config already exists
	var existing models.LLMConfig
	err := database.LLMConfigs().FindOne(ctx, bson.M{
		"key":      models.DefaultLLMConfigKey,
		"isActive": true,
	}).Decode(&existing)
	if err == nil {
		// Already configured — leave it alone
		return nil
	}
	if err != mongo.ErrNoDocuments {
		return err
	}

	now := time.Now()
	cfg := models.LLMConfig{
		Key:       models.DefaultLLMConfigKey,
		APIKey:    apiKey,
		BaseURL:   baseURL,
		Model:     model,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := validation.Validate(&cfg); err != nil {
		return err
	}

	_, err = database.LLMConfigs().InsertOne(ctx, cfg)
	return err
}
