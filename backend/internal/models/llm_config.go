package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// LLMConfig stores LLM provider configuration.
const DefaultLLMConfigKey = "default"

type LLMConfig struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Key       string             `json:"-" bson:"key,omitempty" validate:"omitempty,max=100"`
	APIKey    string             `json:"-" bson:"apiKey" validate:"required_if=IsActive true"`
	BaseURL   string             `json:"baseURL" bson:"baseURL" validate:"required_if=IsActive true"`
	Model     string             `json:"model" bson:"model" validate:"required_if=IsActive true,max=100"`
	IsActive  bool               `json:"isActive" bson:"isActive"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt" validate:"required"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt" validate:"required"`
}
