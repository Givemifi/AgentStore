package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// KnowledgeChunk is one retrievable slice of a KnowledgeDocument together with
// its embedding vector. Retrieval loads all of an agent's chunks and ranks them
// by cosine similarity against the query embedding (see internal/knowledge).
type KnowledgeChunk struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID   primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	AgentID    string             `json:"agentId" bson:"agentId" validate:"required,min=1,max=100"`
	DocumentID primitive.ObjectID `json:"documentId" bson:"documentId" validate:"required"`
	Seq        int                `json:"seq" bson:"seq" validate:"gte=0"`
	Text       string             `json:"text" bson:"text" validate:"required,min=1,max=8000"`
	Embedding  []float64          `json:"-" bson:"embedding" validate:"required,min=1"`
	CreatedAt  time.Time          `json:"createdAt" bson:"createdAt" validate:"required"`
}
