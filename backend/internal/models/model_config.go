package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ModelModality string

const (
	ModelModalityText      ModelModality = "text"
	ModelModalityImage     ModelModality = "image"
	ModelModalityVideo     ModelModality = "video"
	ModelModalityEmbedding ModelModality = "embedding"
)

type ModelConfig struct {
	ID            primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	TenantID      primitive.ObjectID     `json:"tenantId" bson:"tenantId" validate:"required"`
	ProviderID    primitive.ObjectID     `json:"providerId" bson:"providerId" validate:"required"`
	Name          string                 `json:"name" bson:"name" validate:"required,min=1,max=120"`
	DisplayName   string                 `json:"displayName" bson:"displayName" validate:"required,min=1,max=160"`
	Modality      ModelModality          `json:"modality" bson:"modality" validate:"required,valid_model_modality"`
	ModelID       string                 `json:"modelId" bson:"modelId" validate:"required,min=1,max=200"`
	DefaultParams map[string]interface{} `json:"defaultParams" bson:"defaultParams"`
	Enabled       bool                   `json:"enabled" bson:"enabled"`
	CreatedAt     time.Time              `json:"createdAt" bson:"createdAt" validate:"required"`
	UpdatedAt     time.Time              `json:"updatedAt" bson:"updatedAt" validate:"required"`
}

func ValidModelModality(s ModelModality) bool {
	return s == ModelModalityText || s == ModelModalityImage || s == ModelModalityVideo || s == ModelModalityEmbedding
}