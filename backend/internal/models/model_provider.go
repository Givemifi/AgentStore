package models

import (
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProviderType string

const (
	ProviderTypeOpenAICompatible ProviderType = "openai_compatible"
	ProviderTypeAnthropic        ProviderType = "anthropic"
	ProviderTypeGemini           ProviderType = "gemini"
)

type ModelProvider struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID     primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	Name         string             `json:"name" bson:"name" validate:"required,min=1,max=120"`
	ProviderType ProviderType       `json:"providerType" bson:"providerType" validate:"required,valid_provider_type"`
	BaseURL      string             `json:"baseUrl" bson:"baseUrl" validate:"required,min=1,max=500"`
	APIKey       string             `json:"-" bson:"apiKey" validate:"required,min=1"`
	Enabled      bool               `json:"enabled" bson:"enabled"`
	CreatedAt    time.Time          `json:"createdAt" bson:"createdAt" validate:"required"`
	UpdatedAt    time.Time          `json:"updatedAt" bson:"updatedAt" validate:"required"`
}

type PublicModelProvider struct {
	ID            string       `json:"id"`
	TenantID      string       `json:"tenantId"`
	Name          string       `json:"name"`
	ProviderType  ProviderType `json:"providerType"`
	BaseURL       string       `json:"baseUrl"`
	APIKeyPreview string       `json:"apiKeyPreview"`
	Enabled       bool         `json:"enabled"`
	CreatedAt     time.Time    `json:"createdAt"`
	UpdatedAt     time.Time    `json:"updatedAt"`
}

func (p ModelProvider) ToPublic() PublicModelProvider {
	return PublicModelProvider{
		ID:            p.ID.Hex(),
		TenantID:      p.TenantID.Hex(),
		Name:          p.Name,
		ProviderType:  p.ProviderType,
		BaseURL:       p.BaseURL,
		APIKeyPreview: MaskAPIKey(p.APIKey),
		Enabled:       p.Enabled,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

func MaskAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "***"
	}
	return key[:3] + "***" + key[len(key)-4:]
}

func ValidProviderType(s ProviderType) bool {
	return s == ProviderTypeOpenAICompatible || s == ProviderTypeAnthropic || s == ProviderTypeGemini
}