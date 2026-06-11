package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MessageFeedback records a single user's thumbs up/down (and optional comment)
// on an assistant message. A user may have at most one feedback per message
// (enforced by a unique index on messageId+userId); rating is +1 or -1.
type MessageFeedback struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID       primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	UserID         primitive.ObjectID `json:"userId" bson:"userId" validate:"required"`
	ConversationID primitive.ObjectID `json:"conversationId" bson:"conversationId" validate:"required"`
	MessageID      primitive.ObjectID `json:"messageId" bson:"messageId" validate:"required"`
	AgentID        string             `json:"agentId" bson:"agentId" validate:"required,min=1,max=100"`
	Rating         int                `json:"rating" bson:"rating" validate:"required,oneof=1 -1"`
	Comment        string             `json:"comment,omitempty" bson:"comment,omitempty" validate:"max=2000"`
	CreatedAt      time.Time          `json:"createdAt" bson:"createdAt" validate:"required"`
	UpdatedAt      time.Time          `json:"updatedAt" bson:"updatedAt" validate:"required"`
}
