package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// KnowledgeSourceType describes where a knowledge document originated.
type KnowledgeSourceType string

const (
	// KnowledgeSourceDocument is an uploaded file whose text was extracted in the browser.
	KnowledgeSourceDocument KnowledgeSourceType = "document"
	// KnowledgeSourceText is free-form text pasted by an admin.
	KnowledgeSourceText KnowledgeSourceType = "text"
	// KnowledgeSourceQA is a question/answer pair entered by an admin.
	KnowledgeSourceQA KnowledgeSourceType = "qa"
	// KnowledgeSourceAnnotation is content promoted from an annotation's ideal answer.
	KnowledgeSourceAnnotation KnowledgeSourceType = "annotation"
)

// KnowledgeStatus is the ingestion lifecycle state of a knowledge document.
type KnowledgeStatus string

const (
	KnowledgeStatusProcessing KnowledgeStatus = "processing"
	KnowledgeStatusReady      KnowledgeStatus = "ready"
	KnowledgeStatusError      KnowledgeStatus = "error"
)

func ValidKnowledgeSourceType(s KnowledgeSourceType) bool {
	return s == KnowledgeSourceDocument || s == KnowledgeSourceText || s == KnowledgeSourceQA || s == KnowledgeSourceAnnotation
}

func ValidKnowledgeStatus(s KnowledgeStatus) bool {
	return s == KnowledgeStatusProcessing || s == KnowledgeStatusReady || s == KnowledgeStatusError
}

// KnowledgeDocument is the metadata record for a single knowledge source bound
// to an agent. The extracted text is split into KnowledgeChunk records that
// carry the embedding vectors used for retrieval.
//
// AgentID is stored as a string to match the conversation/chat-message model,
// which uses either a DB agent's ObjectID hex or a static catalog slug.
type KnowledgeDocument struct {
	ID           primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	TenantID     primitive.ObjectID  `json:"tenantId" bson:"tenantId" validate:"required"`
	AgentID      string              `json:"agentId" bson:"agentId" validate:"required,min=1,max=100"`
	Name         string              `json:"name" bson:"name" validate:"required,min=1,max=200"`
	SourceType   KnowledgeSourceType `json:"sourceType" bson:"sourceType" validate:"required,valid_knowledge_source_type"`
	Status       KnowledgeStatus     `json:"status" bson:"status" validate:"required,valid_knowledge_status"`
	ErrorMessage string              `json:"errorMessage,omitempty" bson:"errorMessage,omitempty" validate:"max=500"`
	ChunkCount   int                 `json:"chunkCount" bson:"chunkCount" validate:"gte=0"`
	CharCount    int                 `json:"charCount" bson:"charCount" validate:"gte=0"`
	CreatedBy    primitive.ObjectID  `json:"createdBy" bson:"createdBy" validate:"required"`
	CreatedAt    time.Time           `json:"createdAt" bson:"createdAt" validate:"required"`
	UpdatedAt    time.Time           `json:"updatedAt" bson:"updatedAt" validate:"required"`
}
