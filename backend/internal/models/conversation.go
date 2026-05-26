package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Conversation represents a chat conversation between a user and an AI expert.
type Conversation struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID  primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	UserID    primitive.ObjectID `json:"userId" bson:"userId" validate:"required"`
	AgentID   string             `json:"agentId" bson:"agentId" validate:"required,min=1,max=100"`
	Title     string             `json:"title" bson:"title" validate:"required,min=1,max=200"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt" validate:"required"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt" validate:"required"`
}
