package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ShareLink is a snapshot-based public share of a conversation.
// Once created, the captured messages never change; revoking means deleting the record.
type ShareLink struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Token     string             `json:"token" bson:"token" validate:"required,min=16,max=64"`
	TenantID  primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	UserID    primitive.ObjectID `json:"userId" bson:"userId" validate:"required"`
	AgentID   string             `json:"agentId" bson:"agentId" validate:"required,min=1,max=100"`
	AgentName string             `json:"agentName" bson:"agentName" validate:"required,min=1,max=120"`
	Title     string             `json:"title" bson:"title" validate:"max=200"`
	// Messages is a snapshot of the conversation at share time (stripped to safe fields).
	Messages  []SharedMessage    `json:"messages" bson:"messages"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt" validate:"required"`
}

// SharedMessage is the safe subset of a chat message included in a share snapshot.
type SharedMessage struct {
	Role    string `json:"role" bson:"role" validate:"required,oneof=user assistant"`
	Content string `json:"content" bson:"content" validate:"max=2000000"`
}
