package models

import (
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AgentStatus string

const (
	AgentStatusDraft     AgentStatus = "draft"
	AgentStatusPublished AgentStatus = "published"
	AgentStatusArchived  AgentStatus = "archived"
)

type AgentVisibility string

const (
	AgentVisibilityPrivate AgentVisibility = "private"
	AgentVisibilityPublic  AgentVisibility = "public"
)

type AgentCapability string

const (
	AgentCapabilityTextChat        AgentCapability = "text_chat"
	AgentCapabilityImageGeneration AgentCapability = "image_generation"
	AgentCapabilityVideoGeneration AgentCapability = "video_generation"
)

type AgentCreditCost struct {
	TextMessageCredits     int `json:"textMessageCredits" bson:"textMessageCredits" validate:"gte=0"`
	ImageGenerationCredits int `json:"imageGenerationCredits" bson:"imageGenerationCredits" validate:"gte=0"`
	VideoGenerationCredits int `json:"videoGenerationCredits" bson:"videoGenerationCredits" validate:"gte=0"`
}

type AgentModelConfig struct {
	TextModelID  *primitive.ObjectID `json:"textModelId,omitempty" bson:"textModelId,omitempty"`
	ImageModelID *primitive.ObjectID `json:"imageModelId,omitempty" bson:"imageModelId,omitempty"`
	VideoModelID *primitive.ObjectID `json:"videoModelId,omitempty" bson:"videoModelId,omitempty"`
}

type Agent struct {
	ID               primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID         primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	Name             string             `json:"name" bson:"name" validate:"required,min=1,max=120"`
	Slug             string             `json:"slug" bson:"slug" validate:"required,min=1,max=120"`
	Category         string             `json:"category" bson:"category" validate:"required,min=1,max=80"`
	Description      string             `json:"description" bson:"description" validate:"required,min=1,max=500"`
	Avatar           string             `json:"avatar" bson:"avatar" validate:"max=500"`
	Icon             string             `json:"icon" bson:"icon" validate:"max=80"`
	Color            string             `json:"color" bson:"color" validate:"max=32"`
	Status           AgentStatus        `json:"status" bson:"status" validate:"required,valid_agent_status"`
	Visibility       AgentVisibility    `json:"visibility" bson:"visibility" validate:"required,valid_agent_visibility"`
	SystemPrompt     string             `json:"systemPrompt" bson:"systemPrompt" validate:"required,min=1,max=20000"`
	WelcomeMessage   string             `json:"welcomeMessage" bson:"welcomeMessage" validate:"max=1000"`
	SuggestedPrompts []string           `json:"suggestedPrompts" bson:"suggestedPrompts" validate:"dive,max=300"`
	Capabilities     []AgentCapability  `json:"capabilities" bson:"capabilities" validate:"required,min=1,dive,valid_agent_capability"`
	CreditCost       AgentCreditCost    `json:"creditCost" bson:"creditCost" validate:"required"`
	ModelConfig      AgentModelConfig   `json:"modelConfig" bson:"modelConfig"`
	CreatedBy        primitive.ObjectID `json:"createdBy" bson:"createdBy" validate:"required"`
	CreatedAt        time.Time          `json:"createdAt" bson:"createdAt" validate:"required"`
	UpdatedAt        time.Time          `json:"updatedAt" bson:"updatedAt" validate:"required"`
}

type PublicAgent struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Slug             string            `json:"slug"`
	Category         string            `json:"category"`
	Description      string            `json:"description"`
	Avatar           string            `json:"avatar"`
	Icon             string            `json:"icon"`
	Color            string            `json:"color"`
	Visibility       AgentVisibility   `json:"visibility"`
	WelcomeMessage   string            `json:"welcomeMessage"`
	SuggestedPrompts []string          `json:"suggestedPrompts"`
	Capabilities     []AgentCapability `json:"capabilities"`
	CreditCost       AgentCreditCost   `json:"creditCost"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}

func (a Agent) TextCreditCost() int {
	if a.CreditCost.TextMessageCredits > 0 {
		return a.CreditCost.TextMessageCredits
	}
	return 1
}

func (a Agent) ToPublic() PublicAgent {
	return PublicAgent{
		ID:               a.ID.Hex(),
		Name:             a.Name,
		Slug:             a.Slug,
		Category:         a.Category,
		Description:      a.Description,
		Avatar:           a.Avatar,
		Icon:             a.Icon,
		Color:            a.Color,
		Visibility:       a.Visibility,
		WelcomeMessage:   a.WelcomeMessage,
		SuggestedPrompts: append([]string(nil), a.SuggestedPrompts...),
		Capabilities:     append([]AgentCapability(nil), a.Capabilities...),
		CreditCost:       a.CreditCost,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        a.UpdatedAt,
	}
}

func NormalizeAgentSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if valid {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func ValidAgentStatus(s AgentStatus) bool {
	return s == AgentStatusDraft || s == AgentStatusPublished || s == AgentStatusArchived
}

func ValidAgentVisibility(s AgentVisibility) bool {
	return s == AgentVisibilityPrivate || s == AgentVisibilityPublic
}

func ValidAgentCapability(s AgentCapability) bool {
	return s == AgentCapabilityTextChat || s == AgentCapabilityImageGeneration || s == AgentCapabilityVideoGeneration
}