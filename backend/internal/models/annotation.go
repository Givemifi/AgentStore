package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AnnotationStatus tracks whether an annotation has been promoted to knowledge.
type AnnotationStatus string

const (
	AnnotationStatusAnnotated AnnotationStatus = "annotated"
	AnnotationStatusPromoted  AnnotationStatus = "promoted"
)

func ValidAnnotationStatus(s AnnotationStatus) bool {
	return s == AnnotationStatusAnnotated || s == AnnotationStatusPromoted
}

// AnnotationIssueTags are the canonical quality-issue labels an annotator can apply.
var AnnotationIssueTags = []string{
	"wrong_fact", "off_topic", "hallucination", "format", "tone", "incomplete", "other",
}

func ValidAnnotationIssueTag(tag string) bool {
	for _, t := range AnnotationIssueTags {
		if t == tag {
			return true
		}
	}
	return false
}

// Annotation is an admin's quality review of a single assistant message. It can
// carry a corrected "ideal answer" which may later be promoted into the agent's
// knowledge base and exported as SFT training data.
type Annotation struct {
	ID                primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	TenantID          primitive.ObjectID  `json:"tenantId" bson:"tenantId" validate:"required"`
	ConversationID    primitive.ObjectID  `json:"conversationId" bson:"conversationId" validate:"required"`
	MessageID         primitive.ObjectID  `json:"messageId" bson:"messageId" validate:"required"`
	AgentID           string              `json:"agentId" bson:"agentId" validate:"required,min=1,max=100"`
	AnnotatorID       primitive.ObjectID  `json:"annotatorId" bson:"annotatorId" validate:"required"`
	QualityScore      int                 `json:"qualityScore" bson:"qualityScore" validate:"required,gte=1,lte=5"`
	IssueTags         []string            `json:"issueTags" bson:"issueTags" validate:"dive,valid_annotation_issue_tag"`
	IdealAnswer       string              `json:"idealAnswer,omitempty" bson:"idealAnswer,omitempty" validate:"max=20000"`
	Notes             string              `json:"notes,omitempty" bson:"notes,omitempty" validate:"max=2000"`
	Status            AnnotationStatus    `json:"status" bson:"status" validate:"required,valid_annotation_status"`
	PromotedDocumentID *primitive.ObjectID `json:"promotedDocumentId,omitempty" bson:"promotedDocumentId,omitempty"`
	CreatedAt         time.Time           `json:"createdAt" bson:"createdAt" validate:"required"`
	UpdatedAt         time.Time           `json:"updatedAt" bson:"updatedAt" validate:"required"`
}
