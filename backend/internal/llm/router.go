package llm

import (
	"context"
	"errors"
	"fmt"

	"lastsaas/internal/db"
	"lastsaas/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// ErrModelNotConfigured is returned when no model configuration can be resolved
// through the fallback chain (agent binding -> tenant default -> legacy LLMConfig).
var ErrModelNotConfigured = errors.New("model is not configured")

// Router resolves the correct LLM configuration for a request using a
// fallback chain: agent-level binding -> tenant default -> legacy LLMConfig.
type Router struct {
	db *db.MongoDB
}

// NewRouter creates a new model router backed by the given database.
func NewRouter(database *db.MongoDB) *Router {
	return &Router{db: database}
}

// ResolveTextModel resolves the text model configuration for the given tenant and agent.
// Fallback chain: agent.ModelConfig.TextModelID -> tenant.DefaultTextModelConfigID -> legacy LLMConfig.
func (r *Router) ResolveTextModel(ctx context.Context, tenant models.Tenant, agent models.Agent) (RequestConfig, error) {
	// 1. Agent-level binding
	if agent.ModelConfig.TextModelID != nil {
		if cfg, err := r.resolveModelConfig(ctx, tenant.ID, *agent.ModelConfig.TextModelID, models.ModelModalityText); err == nil {
			return cfg, nil
		} else if !errors.Is(err, mongo.ErrNoDocuments) {
			return RequestConfig{}, err
		}
	}

	// 2. Tenant default
	if tenant.DefaultTextModelConfigID != nil {
		if cfg, err := r.resolveModelConfig(ctx, tenant.ID, *tenant.DefaultTextModelConfigID, models.ModelModalityText); err == nil {
			return cfg, nil
		} else if !errors.Is(err, mongo.ErrNoDocuments) {
			return RequestConfig{}, err
		}
	}

	// 3. Legacy LLMConfig
	var legacy models.LLMConfig
	if err := r.db.LLMConfigs().FindOne(ctx, bson.M{"key": models.DefaultLLMConfigKey, "isActive": true}).Decode(&legacy); err == nil {
		return RequestConfig{
			APIKey:  legacy.APIKey,
			BaseURL: normalizeBaseURL(legacy.BaseURL),
			Model:   legacy.Model,
		}, nil
	}

	return RequestConfig{}, ErrModelNotConfigured
}

// ResolveImageModel resolves the image model configuration for the given tenant and agent.
// Fallback chain: agent.ModelConfig.ImageModelID -> tenant.DefaultImageModelConfigID.
func (r *Router) ResolveImageModel(ctx context.Context, tenant models.Tenant, agent models.Agent) (RequestConfig, error) {
	if agent.ModelConfig.ImageModelID != nil {
		if cfg, err := r.resolveModelConfig(ctx, tenant.ID, *agent.ModelConfig.ImageModelID, models.ModelModalityImage); err == nil {
			return cfg, nil
		} else if !errors.Is(err, mongo.ErrNoDocuments) {
			return RequestConfig{}, err
		}
	}

	if tenant.DefaultImageModelConfigID != nil {
		if cfg, err := r.resolveModelConfig(ctx, tenant.ID, *tenant.DefaultImageModelConfigID, models.ModelModalityImage); err == nil {
			return cfg, nil
		} else if !errors.Is(err, mongo.ErrNoDocuments) {
			return RequestConfig{}, err
		}
	}

	return RequestConfig{}, fmt.Errorf("image %w", ErrModelNotConfigured)
}

// ResolveVideoModel resolves the video model configuration for the given tenant and agent.
// Fallback chain: agent.ModelConfig.VideoModelID -> tenant.DefaultVideoModelConfigID.
func (r *Router) ResolveVideoModel(ctx context.Context, tenant models.Tenant, agent models.Agent) (RequestConfig, error) {
	if agent.ModelConfig.VideoModelID != nil {
		if cfg, err := r.resolveModelConfig(ctx, tenant.ID, *agent.ModelConfig.VideoModelID, models.ModelModalityVideo); err == nil {
			return cfg, nil
		} else if !errors.Is(err, mongo.ErrNoDocuments) {
			return RequestConfig{}, err
		}
	}

	if tenant.DefaultVideoModelConfigID != nil {
		if cfg, err := r.resolveModelConfig(ctx, tenant.ID, *tenant.DefaultVideoModelConfigID, models.ModelModalityVideo); err == nil {
			return cfg, nil
		} else if !errors.Is(err, mongo.ErrNoDocuments) {
			return RequestConfig{}, err
		}
	}

	return RequestConfig{}, fmt.Errorf("video %w", ErrModelNotConfigured)
}

// resolveModelConfig loads a ModelConfig and its associated ModelProvider,
// returning a RequestConfig ready for use with the LLM client.
func (r *Router) resolveModelConfig(ctx context.Context, tenantID, modelID primitive.ObjectID, modality models.ModelModality) (RequestConfig, error) {
	var model models.ModelConfig
	if err := r.db.ModelConfigs().FindOne(ctx, bson.M{
		"_id":       modelID,
		"tenantId":  tenantID,
		"modality":  modality,
		"enabled":   true,
	}).Decode(&model); err != nil {
		return RequestConfig{}, err
	}

	var provider models.ModelProvider
	if err := r.db.ModelProviders().FindOne(ctx, bson.M{
		"_id":      model.ProviderID,
		"tenantId": tenantID,
		"enabled":  true,
	}).Decode(&provider); err != nil {
		return RequestConfig{}, err
	}

	return RequestConfig{
		APIKey:  provider.APIKey,
		BaseURL: normalizeBaseURL(provider.BaseURL),
		Model:   model.ModelID,
	}, nil
}