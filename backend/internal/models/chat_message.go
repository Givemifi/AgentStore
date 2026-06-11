package models

import (
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatMessageStatus string

const (
	ChatMessageStatusGenerating  ChatMessageStatus = "generating"
	ChatMessageStatusCompleted   ChatMessageStatus = "completed"
	ChatMessageStatusError       ChatMessageStatus = "error"
	ChatMessageStatusInterrupted ChatMessageStatus = "interrupted"
)

func ValidChatMessageStatus(s ChatMessageStatus) bool {
	return s == "" || s == ChatMessageStatusGenerating || s == ChatMessageStatusCompleted || s == ChatMessageStatusError || s == ChatMessageStatusInterrupted
}

func (s ChatMessageStatus) AllowsEmptyContent() bool {
	return s == ChatMessageStatusGenerating
}

func (m ChatMessage) ContentRequired() bool {
	return !m.Status.AllowsEmptyContent()
}

func (m ChatMessage) HasContent() bool {
	return strings.TrimSpace(m.Content) != ""
}

// ChatMessage represents a single message in a chat conversation.
type ChatMessage struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID        primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	UserID          primitive.ObjectID `json:"userId" bson:"userId" validate:"required"`
	ConversationID  primitive.ObjectID `json:"conversationId" bson:"conversationId" validate:"required"`
	AgentID         string             `json:"agentId" bson:"agentId" validate:"required,min=1,max=100"`
	Role            string             `json:"role" bson:"role" validate:"required,oneof=user assistant"`
	Content         string             `json:"content" bson:"content" validate:"max=2000000"`
	Status          ChatMessageStatus  `json:"status" bson:"status" validate:"omitempty,valid_chat_message_status"`
	CreditsCharged  int                `json:"creditsCharged" bson:"creditsCharged" validate:"gte=0"`
	Model           string             `json:"model" bson:"model" validate:"max=100"`
	// AttachmentCount records how many image attachments the user sent with this
	// message. The actual image data is not persisted (too large); this field
	// lets the UI show a "📷 N images" placeholder in chat history.
	AttachmentCount int                `json:"attachmentCount,omitempty" bson:"attachmentCount,omitempty" validate:"gte=0"`
	// PromptTokens / CompletionTokens record token usage for assistant messages.
	// They come from the provider's usage report when available, otherwise from a
	// character-based estimate (see llm.estimateUsage), so treat them as approximate.
	PromptTokens     int       `json:"promptTokens,omitempty" bson:"promptTokens,omitempty" validate:"gte=0"`
	CompletionTokens int       `json:"completionTokens,omitempty" bson:"completionTokens,omitempty" validate:"gte=0"`
	CreatedAt       time.Time          `json:"createdAt" bson:"createdAt" validate:"required"`
}
