# AgentStore Mainline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the current AgentStore-derived app into AgentStore’s mainline product loop with tenant-scoped agents, tenant model settings, `/admin` routing, visible rebranding, and streaming chat.

**Architecture:** Keep the existing Go + MongoDB + React architecture. Add focused backend models/handlers/services for agents and models, then update chat to resolve tenant-published agents and stream responses while preserving the existing non-streaming endpoint. Frontend changes reuse the existing app shell, settings page, TanStack Query API client, and route structure.

**Tech Stack:** Go 1.25, Gorilla Mux, MongoDB Go driver, go-playground/validator, React 19, Vite, TypeScript, React Router, TanStack Query, Vitest, Testing Library, Tailwind.

---

## File Structure

### Backend models and validation

- Modify `backend/internal/models/agent.go` — replace the static catalog alias with the persistent Agent model and public/admin response helpers.
- Create `backend/internal/models/model_provider.go` — tenant-scoped model provider model and enum helpers.
- Create `backend/internal/models/model_config.go` — tenant-scoped model config model and enum helpers.
- Modify `backend/internal/models/tenant.go` — add optional tenant default model references.
- Modify `backend/internal/models/chat_message.go` — add message status enum and field.
- Modify `backend/internal/validation/validate.go` — register enum validators for agent status, visibility, capability, provider type, model modality, and message status.
- Modify `backend/internal/validation/validate_test.go` — add validation tests for new models and message status.

### Backend database

- Modify `backend/internal/db/schema.go` — add MongoDB JSON Schema for `agents`, `model_providers`, and `model_configs`; extend tenant/chat message schemas.
- Modify `backend/internal/db/mongodb.go` — add indexes and collection accessors.
- Create `backend/internal/db/schema_test.go` — assert required schemas and key properties are present.

### Backend APIs and services

- Create `backend/internal/api/handlers/agents.go` — public agent list/read plus tenant agent admin CRUD/status handlers.
- Create `backend/internal/api/handlers/agents_test.go` — tenant isolation, redaction, archive/publish tests.
- Create `backend/internal/api/handlers/model_settings.go` — tenant model provider/config/default handlers.
- Create `backend/internal/api/handlers/model_settings_test.go` — key masking, key preservation, tenant isolation tests.
- Create `backend/internal/llm/router.go` — model router fallback from agent binding to tenant defaults to legacy `llm_configs`.
- Create `backend/internal/llm/router_test.go` — fallback-order tests.
- Modify `backend/internal/llm/openai.go` — add streaming support for OpenAI-compatible providers.
- Create `backend/internal/llm/stream_test.go` — streaming parser/client behavior tests.
- Modify `backend/internal/api/handlers/chat.go` — resolve persistent agents, keep non-streaming endpoint, add SSE stream endpoint, add image/video coming-soon handlers.
- Modify `backend/internal/api/handlers/chat_test.go` — add streaming success/failure/interruption credit tests and update static catalog tests.
- Modify `backend/cmd/server/main.go` — register `/api/agents`, `/api/tenant/agents`, `/api/tenant/model-*`, `/api/chat/stream`, `/api/agents/{agentId}/generate-image`, and `/api/agents/{agentId}/generate-video`.

### Frontend routing, types, and APIs

- Create `frontend/src/utils/storageKeys.ts` — compatible localStorage key migration from `agentstore_*` to `agentstore_*`.
- Modify `frontend/src/api/client.ts` — use new storage helpers, add public agent APIs, tenant agent APIs, tenant model APIs, and streaming chat helper.
- Modify `frontend/src/types/index.ts` — add AgentStore agent/model/provider/message-status types.
- Modify `frontend/src/App.tsx` — replace `/last` routes with `/admin`, redirect `/last/*` to `/admin/*`, add `/chat` redirect, add settings subroutes.
- Modify `frontend/src/components/AdminLayout.tsx` and `frontend/src/components/AdminLayout.test.tsx` — update root admin links to `/admin` and replace LLM Config with system-only admin links.
- Modify `frontend/src/components/Layout.tsx` — rebrand nav, link root users to `/admin`, and write AgentStore localStorage keys.
- Modify `frontend/src/components/BrandingThemeInjector.tsx` — treat `/admin` as admin path.
- Modify `frontend/src/contexts/AuthContext.tsx`, `frontend/src/contexts/TenantContext.tsx`, `frontend/src/contexts/ThemeContext.tsx`, `frontend/src/hooks/useTelemetry.ts` — use migrated AgentStore storage keys.

### Frontend product pages

- Modify `frontend/src/pages/app/DashboardPage.tsx` and `frontend/src/pages/app/DashboardPage.test.tsx` — AgentStore marketplace-style published agent discovery.
- Modify `frontend/src/pages/app/ChatPage.tsx` and `frontend/src/pages/app/ChatPage.test.tsx` — streaming deltas, abort, retry, mobile drawer, `/chat` fallback.
- Modify `frontend/src/pages/app/SettingsPage.tsx` — route-driven tabs for profile/security/sessions/billing/agents/models.
- Create `frontend/src/pages/app/settings/AgentsTab.tsx` and `frontend/src/pages/app/settings/AgentsTab.test.tsx` — tenant-scoped agent management.
- Create `frontend/src/pages/app/settings/ModelSettingsTab.tsx` and `frontend/src/pages/app/settings/ModelSettingsTab.test.tsx` — tenant-scoped provider/model settings.
- Modify `frontend/src/pages/admin/AboutPage.tsx` — show AgentStore.
- Modify `frontend/src/pages/BootstrapPage.tsx` — AgentStore setup instructions.
- Modify admin pages with hardcoded `/last` links: `frontend/src/pages/admin/DashboardPage.tsx`, `TenantProfilePage.tsx`, `LogsPage.tsx`, `UserProfilePage.tsx`, `TenantsPage.tsx`, `UsersPage.tsx`, `BrandingPage.tsx`.

### Docs and delivery

- Modify `README.md` — AgentStore features, tech stack, quick start, env setup.
- Create or modify `.env.example` — safe AgentStore-oriented config with no real keys.
- Modify `backend/config/dev.yaml`, `backend/config/dev.example.yaml`, `backend/config/test.yaml` — safe AgentStore defaults while keeping env compatibility.
- Remove `backend/internal/agents/catalog.go` and `backend/internal/agents/catalog_test.go` only after DB-backed agent APIs are passing.

---

## Execution Rules

- Do not change auth, tenant, billing, credits, Stripe, webhooks, telemetry, or bootstrap behavior except where route wiring requires it.
- Do not return provider API keys to the frontend.
- Do not deduct credits until a model response completes successfully.
- Do not hard-delete agents; archive them.
- Do not commit unless the user has explicitly authorized commits for the implementation session. The commit commands below are checkpoints to use only after authorization.
- Before reporting completion, run backend build, frontend typecheck, focused tests, and a browser verification of dashboard/settings/chat.

---

## Task 1: Add persistent model structs and validation ~~DONE~~

**Files:**
- Modify: `backend/internal/models/agent.go`
- Create: `backend/internal/models/model_provider.go`
- Create: `backend/internal/models/model_config.go`
- Modify: `backend/internal/models/tenant.go`
- Modify: `backend/internal/models/chat_message.go`
- Modify: `backend/internal/validation/validate.go`
- Test: `backend/internal/validation/validate_test.go`

- [x] **Step 1: Write failing validation tests**

Append these tests to `backend/internal/validation/validate_test.go`:

```go
func validAgent() models.Agent {
	return models.Agent{
		TenantID:    primitive.NewObjectID(),
		Name:        "Growth Strategist",
		Slug:        "growth-strategist",
		Category:    "Marketing",
		Description: "Plans launch and growth work",
		Icon:        "rocket",
		Color:       "#7C3AED",
		Status:      models.AgentStatusDraft,
		Visibility:  models.AgentVisibilityPrivate,
		SystemPrompt: "You are a growth strategist.",
		WelcomeMessage: "Tell me about your launch.",
		SuggestedPrompts: []string{"Draft a launch plan"},
		Capabilities: []models.AgentCapability{models.AgentCapabilityTextChat},
		CreditCost: models.AgentCreditCost{TextMessageCredits: 3},
		CreatedBy: primitive.NewObjectID(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestValidate_ValidAgent(t *testing.T) {
	agent := validAgent()
	if err := Validate(&agent); err != nil {
		t.Fatalf("expected valid agent to pass: %v", err)
	}
}

func TestValidate_AgentInvalidStatus(t *testing.T) {
	agent := validAgent()
	agent.Status = "deleted"
	if err := Validate(&agent); err == nil {
		t.Fatal("expected invalid agent status to fail")
	}
}

func TestValidate_AgentRequiresTextCapability(t *testing.T) {
	agent := validAgent()
	agent.Capabilities = []models.AgentCapability{"spreadsheet_magic"}
	if err := Validate(&agent); err == nil {
		t.Fatal("expected invalid capability to fail")
	}
}

func TestValidate_ValidModelProvider(t *testing.T) {
	provider := models.ModelProvider{
		TenantID: primitive.NewObjectID(),
		Name: "OpenAI Compatible",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL: "https://api.example.com/v1",
		APIKey: "sk-test",
		Enabled: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := Validate(&provider); err != nil {
		t.Fatalf("expected valid model provider to pass: %v", err)
	}
}

func TestValidate_ModelProviderInvalidType(t *testing.T) {
	provider := models.ModelProvider{
		TenantID: primitive.NewObjectID(),
		Name: "Bad",
		ProviderType: "unknown",
		BaseURL: "https://api.example.com/v1",
		APIKey: "sk-test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := Validate(&provider); err == nil {
		t.Fatal("expected invalid provider type to fail")
	}
}

func TestValidate_ValidModelConfig(t *testing.T) {
	config := models.ModelConfig{
		TenantID: primitive.NewObjectID(),
		ProviderID: primitive.NewObjectID(),
		Name: "Default Text",
		DisplayName: "Default Text Model",
		Modality: models.ModelModalityText,
		ModelID: "gpt-4o-mini",
		DefaultParams: map[string]interface{}{"temperature": 0.7},
		Enabled: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := Validate(&config); err != nil {
		t.Fatalf("expected valid model config to pass: %v", err)
	}
}

func TestValidate_ModelConfigInvalidModality(t *testing.T) {
	config := models.ModelConfig{
		TenantID: primitive.NewObjectID(),
		ProviderID: primitive.NewObjectID(),
		Name: "Bad",
		DisplayName: "Bad",
		Modality: "audio",
		ModelID: "model",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := Validate(&config); err == nil {
		t.Fatal("expected invalid model modality to fail")
	}
}

func TestValidate_ChatMessageStatus(t *testing.T) {
	msg := models.ChatMessage{
		TenantID: primitive.NewObjectID(),
		UserID: primitive.NewObjectID(),
		ConversationID: primitive.NewObjectID(),
		AgentID: "growth-strategist",
		Role: "assistant",
		Content: "Working on it",
		Status: models.ChatMessageStatusGenerating,
		CreatedAt: time.Now(),
	}
	if err := Validate(&msg); err != nil {
		t.Fatalf("expected generating chat message to pass: %v", err)
	}
	msg.Status = "vanished"
	if err := Validate(&msg); err == nil {
		t.Fatal("expected invalid chat message status to fail")
	}
}
```

- [x] **Step 2: Run validation tests and verify failure**

Run:

```bash
cd backend && go test ./internal/validation/...
```

Expected: FAIL because `models.Agent`, `models.ModelProvider`, `models.ModelConfig`, and enum constants are not defined yet.

- [x] **Step 3: Replace agent model alias with persistent Agent model**

Replace `backend/internal/models/agent.go` with:

```go
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
	TextMessageCredits      int `json:"textMessageCredits" bson:"textMessageCredits" validate:"gte=0"`
	ImageGenerationCredits  int `json:"imageGenerationCredits" bson:"imageGenerationCredits" validate:"gte=0"`
	VideoGenerationCredits  int `json:"videoGenerationCredits" bson:"videoGenerationCredits" validate:"gte=0"`
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
		ID: a.ID.Hex(),
		Name: a.Name,
		Slug: a.Slug,
		Category: a.Category,
		Description: a.Description,
		Avatar: a.Avatar,
		Icon: a.Icon,
		Color: a.Color,
		Visibility: a.Visibility,
		WelcomeMessage: a.WelcomeMessage,
		SuggestedPrompts: append([]string(nil), a.SuggestedPrompts...),
		Capabilities: append([]AgentCapability(nil), a.Capabilities...),
		CreditCost: a.CreditCost,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
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
```

- [x] **Step 4: Add model provider/config structs**

Create `backend/internal/models/model_provider.go`:

```go
package models

import (
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProviderType string

const (
	ProviderTypeOpenAICompatible ProviderType = "openai_compatible"
	ProviderTypeAnthropic        ProviderType = "anthropic"
	ProviderTypeGemini           ProviderType = "gemini"
)

type ModelProvider struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID     primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	Name         string             `json:"name" bson:"name" validate:"required,min=1,max=120"`
	ProviderType ProviderType       `json:"providerType" bson:"providerType" validate:"required,valid_provider_type"`
	BaseURL      string             `json:"baseUrl" bson:"baseUrl" validate:"required,min=1,max=500"`
	APIKey       string             `json:"-" bson:"apiKey" validate:"required,min=1"`
	Enabled      bool               `json:"enabled" bson:"enabled"`
	CreatedAt    time.Time          `json:"createdAt" bson:"createdAt" validate:"required"`
	UpdatedAt    time.Time          `json:"updatedAt" bson:"updatedAt" validate:"required"`
}

type PublicModelProvider struct {
	ID            string       `json:"id"`
	TenantID      string       `json:"tenantId"`
	Name          string       `json:"name"`
	ProviderType  ProviderType `json:"providerType"`
	BaseURL       string       `json:"baseUrl"`
	APIKeyPreview string       `json:"apiKeyPreview"`
	Enabled       bool         `json:"enabled"`
	CreatedAt     time.Time    `json:"createdAt"`
	UpdatedAt     time.Time    `json:"updatedAt"`
}

func (p ModelProvider) ToPublic() PublicModelProvider {
	return PublicModelProvider{
		ID: p.ID.Hex(),
		TenantID: p.TenantID.Hex(),
		Name: p.Name,
		ProviderType: p.ProviderType,
		BaseURL: p.BaseURL,
		APIKeyPreview: MaskAPIKey(p.APIKey),
		Enabled: p.Enabled,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func MaskAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "***"
	}
	return key[:3] + "***" + key[len(key)-4:]
}

func ValidProviderType(s ProviderType) bool {
	return s == ProviderTypeOpenAICompatible || s == ProviderTypeAnthropic || s == ProviderTypeGemini
}
```

Create `backend/internal/models/model_config.go`:

```go
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ModelModality string

const (
	ModelModalityText  ModelModality = "text"
	ModelModalityImage ModelModality = "image"
	ModelModalityVideo ModelModality = "video"
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
	return s == ModelModalityText || s == ModelModalityImage || s == ModelModalityVideo
}
```

- [x] **Step 5: Extend tenant and chat message models**

In `backend/internal/models/tenant.go`, add these fields before `CreatedAt`:

```go
	DefaultTextModelConfigID  *primitive.ObjectID `json:"defaultTextModelConfigId,omitempty" bson:"defaultTextModelConfigId,omitempty"`
	DefaultImageModelConfigID *primitive.ObjectID `json:"defaultImageModelConfigId,omitempty" bson:"defaultImageModelConfigId,omitempty"`
	DefaultVideoModelConfigID *primitive.ObjectID `json:"defaultVideoModelConfigId,omitempty" bson:"defaultVideoModelConfigId,omitempty"`
```

In `backend/internal/models/chat_message.go`, add the enum above `ChatMessage`:

```go
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
```

Then add this field to `ChatMessage` after `Content`:

```go
	Status         ChatMessageStatus `json:"status" bson:"status" validate:"omitempty,valid_chat_message_status"`
```

- [x] **Step 6: Register validators**

Add these registrations inside `init()` in `backend/internal/validation/validate.go`:

```go
	v.RegisterValidation("valid_agent_status", func(fl validator.FieldLevel) bool {
		return models.ValidAgentStatus(models.AgentStatus(fl.Field().String()))
	})
	v.RegisterValidation("valid_agent_visibility", func(fl validator.FieldLevel) bool {
		return models.ValidAgentVisibility(models.AgentVisibility(fl.Field().String()))
	})
	v.RegisterValidation("valid_agent_capability", func(fl validator.FieldLevel) bool {
		return models.ValidAgentCapability(models.AgentCapability(fl.Field().String()))
	})
	v.RegisterValidation("valid_provider_type", func(fl validator.FieldLevel) bool {
		return models.ValidProviderType(models.ProviderType(fl.Field().String()))
	})
	v.RegisterValidation("valid_model_modality", func(fl validator.FieldLevel) bool {
		return models.ValidModelModality(models.ModelModality(fl.Field().String()))
	})
	v.RegisterValidation("valid_chat_message_status", func(fl validator.FieldLevel) bool {
		return models.ValidChatMessageStatus(models.ChatMessageStatus(fl.Field().String()))
	})
```

- [x] **Step 7: Run validation tests**

Run:

```bash
cd backend && go test ./internal/validation/...
```

Expected: PASS.

- [x] **Step 8: Commit checkpoint after authorization**

If commit authorization exists, run:

```bash
git add backend/internal/models/agent.go backend/internal/models/model_provider.go backend/internal/models/model_config.go backend/internal/models/tenant.go backend/internal/models/chat_message.go backend/internal/validation/validate.go backend/internal/validation/validate_test.go
git commit -m "Add AgentStore model validation"
```

---

## Task 2: Add MongoDB schema, indexes, and collection accessors ~~DONE~~

**Files:**
- Modify: `backend/internal/db/schema.go`
- Modify: `backend/internal/db/mongodb.go`
- Test: `backend/internal/db/schema_test.go`

- [x] **Step 1: Write schema tests**

Create `backend/internal/db/schema_test.go`:

```go
package db

import "testing"

func findSchema(collection string) (CollectionSchema, bool) {
	for _, schema := range AllSchemas() {
		if schema.Collection == collection {
			return schema, true
		}
	}
	return CollectionSchema{}, false
}

func TestAllSchemasIncludesAgentStoreCollections(t *testing.T) {
	for _, collection := range []string{"agents", "model_providers", "model_configs"} {
		if _, ok := findSchema(collection); !ok {
			t.Fatalf("expected schema for %s", collection)
		}
	}
}

func TestAgentsSchemaRequiresTenantScopedFields(t *testing.T) {
	schema, ok := findSchema("agents")
	if !ok {
		t.Fatal("agents schema missing")
	}
	jsonSchema := schema.Schema["$jsonSchema"].(map[string]interface{})
	required := jsonSchema["required"].([]interface{})
	want := map[string]bool{"tenantId": true, "name": true, "slug": true, "status": true, "visibility": true, "systemPrompt": true}
	for _, item := range required {
		delete(want, item.(string))
	}
	if len(want) != 0 {
		t.Fatalf("missing required agent fields: %#v", want)
	}
}
```

If direct type assertions fail because BSON values are `bson.M` and `bson.A`, adjust the assertions to use `bson.M` and `bson.A` imports:

```go
jsonSchema := schema.Schema["$jsonSchema"].(bson.M)
required := jsonSchema["required"].(bson.A)
```

- [x] **Step 2: Run db tests and verify failure**

Run:

```bash
cd backend && go test ./internal/db -run 'TestAllSchemasIncludesAgentStoreCollections|TestAgentsSchemaRequiresTenantScopedFields'
```

Expected: FAIL because the new collection schemas are not registered.

- [x] **Step 3: Register new schemas**

In `backend/internal/db/schema.go`, update `AllSchemas()` to include:

```go
	agentsSchema(),
	modelProvidersSchema(),
	modelConfigsSchema(),
```

Place them near `conversationsSchema()` and `llmConfigsSchema()` so chat/model data stays grouped.

- [x] **Step 4: Add schema functions**

Add these functions near the existing chat schemas in `backend/internal/db/schema.go`:

```go
func agentsSchema() CollectionSchema {
	return CollectionSchema{Collection: "agents", Schema: bson.M{"$jsonSchema": bson.M{
		"bsonType": "object",
		"required": bson.A{"tenantId", "name", "slug", "category", "description", "status", "visibility", "systemPrompt", "capabilities", "creditCost", "createdBy", "createdAt", "updatedAt"},
		"properties": bson.M{
			"tenantId": bson.M{"bsonType": "objectId"},
			"name": bson.M{"bsonType": "string", "minLength": 1, "maxLength": 120},
			"slug": bson.M{"bsonType": "string", "minLength": 1, "maxLength": 120},
			"category": bson.M{"bsonType": "string", "minLength": 1, "maxLength": 80},
			"description": bson.M{"bsonType": "string", "minLength": 1, "maxLength": 500},
			"avatar": bson.M{"bsonType": "string", "maxLength": 500},
			"icon": bson.M{"bsonType": "string", "maxLength": 80},
			"color": bson.M{"bsonType": "string", "maxLength": 32},
			"status": bson.M{"bsonType": "string", "enum": bson.A{"draft", "published", "archived"}},
			"visibility": bson.M{"bsonType": "string", "enum": bson.A{"private", "public"}},
			"systemPrompt": bson.M{"bsonType": "string", "minLength": 1},
			"welcomeMessage": bson.M{"bsonType": "string", "maxLength": 1000},
			"suggestedPrompts": bson.M{"bsonType": "array", "items": bson.M{"bsonType": "string", "maxLength": 300}},
			"capabilities": bson.M{"bsonType": "array", "minItems": 1, "items": bson.M{"bsonType": "string", "enum": bson.A{"text_chat", "image_generation", "video_generation"}}},
			"creditCost": bson.M{"bsonType": "object", "properties": bson.M{
				"textMessageCredits": bson.M{"bsonType": "int", "minimum": 0},
				"imageGenerationCredits": bson.M{"bsonType": "int", "minimum": 0},
				"videoGenerationCredits": bson.M{"bsonType": "int", "minimum": 0},
			}},
			"modelConfig": bson.M{"bsonType": "object", "properties": bson.M{
				"textModelId": bson.M{"bsonType": "objectId"},
				"imageModelId": bson.M{"bsonType": "objectId"},
				"videoModelId": bson.M{"bsonType": "objectId"},
			}},
			"createdBy": bson.M{"bsonType": "objectId"},
			"createdAt": bson.M{"bsonType": "date"},
			"updatedAt": bson.M{"bsonType": "date"},
		},
	}}}
}

func modelProvidersSchema() CollectionSchema {
	return CollectionSchema{Collection: "model_providers", Schema: bson.M{"$jsonSchema": bson.M{
		"bsonType": "object",
		"required": bson.A{"tenantId", "name", "providerType", "baseUrl", "apiKey", "enabled", "createdAt", "updatedAt"},
		"properties": bson.M{
			"tenantId": bson.M{"bsonType": "objectId"},
			"name": bson.M{"bsonType": "string", "minLength": 1, "maxLength": 120},
			"providerType": bson.M{"bsonType": "string", "enum": bson.A{"openai_compatible", "anthropic", "gemini"}},
			"baseUrl": bson.M{"bsonType": "string", "minLength": 1, "maxLength": 500},
			"apiKey": bson.M{"bsonType": "string", "minLength": 1},
			"enabled": bson.M{"bsonType": "bool"},
			"createdAt": bson.M{"bsonType": "date"},
			"updatedAt": bson.M{"bsonType": "date"},
		},
	}}}
}

func modelConfigsSchema() CollectionSchema {
	return CollectionSchema{Collection: "model_configs", Schema: bson.M{"$jsonSchema": bson.M{
		"bsonType": "object",
		"required": bson.A{"tenantId", "providerId", "name", "displayName", "modality", "modelId", "enabled", "createdAt", "updatedAt"},
		"properties": bson.M{
			"tenantId": bson.M{"bsonType": "objectId"},
			"providerId": bson.M{"bsonType": "objectId"},
			"name": bson.M{"bsonType": "string", "minLength": 1, "maxLength": 120},
			"displayName": bson.M{"bsonType": "string", "minLength": 1, "maxLength": 160},
			"modality": bson.M{"bsonType": "string", "enum": bson.A{"text", "image", "video"}},
			"modelId": bson.M{"bsonType": "string", "minLength": 1, "maxLength": 200},
			"defaultParams": bson.M{"bsonType": "object"},
			"enabled": bson.M{"bsonType": "bool"},
			"createdAt": bson.M{"bsonType": "date"},
			"updatedAt": bson.M{"bsonType": "date"},
		},
	}}}
}
```

- [x] **Step 5: Extend existing tenant and chat message schemas**

In `tenantsSchema()`, add optional properties:

```go
						"defaultTextModelConfigId": bson.M{"bsonType": "objectId"},
						"defaultImageModelConfigId": bson.M{"bsonType": "objectId"},
						"defaultVideoModelConfigId": bson.M{"bsonType": "objectId"},
```

In `chatMessagesSchema()`, add optional property:

```go
						"status": bson.M{"bsonType": "string", "enum": bson.A{"generating", "completed", "error", "interrupted", ""}},
```

- [x] **Step 6: Add indexes and accessors**

In `backend/internal/db/mongodb.go`, add index entries:

```go
		{
			"agents",
			[]mongo.IndexModel{
				{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "slug", Value: 1}}, Options: options.Index().SetUnique(true)},
				{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "status", Value: 1}}},
				{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "updatedAt", Value: -1}}},
			},
		},
		{
			"model_providers",
			[]mongo.IndexModel{
				{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "name", Value: 1}}, Options: options.Index().SetUnique(true)},
				{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "enabled", Value: 1}}},
			},
		},
		{
			"model_configs",
			[]mongo.IndexModel{
				{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "providerId", Value: 1}, {Key: "modality", Value: 1}}},
				{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "modality", Value: 1}, {Key: "enabled", Value: 1}}},
			},
		},
```

Add collection accessors at the end:

```go
func (m *MongoDB) Agents() *mongo.Collection {
	return m.Database.Collection("agents")
}

func (m *MongoDB) ModelProviders() *mongo.Collection {
	return m.Database.Collection("model_providers")
}

func (m *MongoDB) ModelConfigs() *mongo.Collection {
	return m.Database.Collection("model_configs")
}
```

- [x] **Step 7: Run db and validation tests**

Run:

```bash
cd backend && go test ./internal/db ./internal/validation/...
```

Expected: PASS.

- [x] **Step 8: Commit checkpoint after authorization**

If commit authorization exists, run:

```bash
git add backend/internal/db/schema.go backend/internal/db/mongodb.go backend/internal/db/schema_test.go
git commit -m "Add AgentStore MongoDB schemas"
```

---

## Task 3: Add tenant-scoped agent APIs and public agent APIs ~~DONE~~

**Files:**
- Create: `backend/internal/api/handlers/agents.go`
- Create: `backend/internal/api/handlers/agents_test.go`
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/api/handlers/chat.go`

- [x] **Step 1: Write agent handler tests**

Create `backend/internal/api/handlers/agents_test.go` with these tests:

```go
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agentstore/internal/middleware"
	"agentstore/internal/models"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func seedAgent(t *testing.T, env *testEnv, tenantID primitive.ObjectID, userID primitive.ObjectID, status models.AgentStatus, slug string) models.Agent {
	t.Helper()
	now := time.Now()
	agent := models.Agent{
		ID: primitive.NewObjectID(), TenantID: tenantID, Name: "Growth Strategist", Slug: slug,
		Category: "Marketing", Description: "Plans growth", Icon: "rocket", Color: "#7C3AED",
		Status: status, Visibility: models.AgentVisibilityPrivate, SystemPrompt: "Secret prompt",
		WelcomeMessage: "Welcome", SuggestedPrompts: []string{"Draft a launch plan"},
		Capabilities: []models.AgentCapability{models.AgentCapabilityTextChat},
		CreditCost: models.AgentCreditCost{TextMessageCredits: 3}, CreatedBy: userID,
		CreatedAt: now, UpdatedAt: now,
	}
	if _, err := env.DB.Agents().InsertOne(context.Background(), agent); err != nil {
		t.Fatalf("seed agent: %v", err)
	}
	return agent
}

func TestPublicListAgents_RedactsSystemPromptAndDrafts(t *testing.T) {
	if testing.Short() { t.Skip("skipping integration test in short mode") }
	env := setupTestServer(t); defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	seedAgent(t, env, tenant.ID, user.ID, models.AgentStatusPublished, "published-agent")
	seedAgent(t, env, tenant.ID, user.ID, models.AgentStatusDraft, "draft-agent")

	req := env.tenantRequest(t, http.MethodGet, "/api/agents", nil, user, tenant.ID.Hex())
	rr := httptest.NewRecorder()
	NewAgentHandler(env.DB).ListPublic(rr, req)

	if rr.Code != http.StatusOK { t.Fatalf("expected 200, got %d body %s", rr.Code, rr.Body.String()) }
	if strings.Contains(rr.Body.String(), "systemPrompt") || strings.Contains(rr.Body.String(), "Secret prompt") {
		t.Fatalf("public response leaked prompt: %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "published-agent") || strings.Contains(rr.Body.String(), "draft-agent") {
		t.Fatalf("expected only published agent, got %s", rr.Body.String())
	}
}

func TestTenantListAgents_IncludesDraftsForAdmins(t *testing.T) {
	if testing.Short() { t.Skip("skipping integration test in short mode") }
	env := setupTestServer(t); defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	seedAgent(t, env, tenant.ID, user.ID, models.AgentStatusDraft, "draft-agent")

	req := env.tenantRequest(t, http.MethodGet, "/api/tenant/agents", nil, user, tenant.ID.Hex())
	rr := httptest.NewRecorder()
	NewAgentHandler(env.DB).ListTenant(rr, req)

	if rr.Code != http.StatusOK { t.Fatalf("expected 200, got %d body %s", rr.Code, rr.Body.String()) }
	if !strings.Contains(rr.Body.String(), "systemPrompt") || !strings.Contains(rr.Body.String(), "draft-agent") {
		t.Fatalf("expected management response with draft and prompt, got %s", rr.Body.String())
	}
}

func TestCreateAgent_NormalizesSlugAndUsesTenant(t *testing.T) {
	if testing.Short() { t.Skip("skipping integration test in short mode") }
	env := setupTestServer(t); defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	body := []byte(`{"name":"Launch Coach","slug":"Launch Coach!","category":"Marketing","description":"Launch planning","icon":"rocket","color":"#7C3AED","systemPrompt":"Help launch.","welcomeMessage":"Ready","suggestedPrompts":["Plan launch"],"capabilities":["text_chat"],"creditCost":{"textMessageCredits":2,"imageGenerationCredits":0,"videoGenerationCredits":0},"visibility":"private"}`)

	req := env.tenantRequest(t, http.MethodPost, "/api/tenant/agents", bytes.NewReader(body), user, tenant.ID.Hex())
	rr := httptest.NewRecorder()
	NewAgentHandler(env.DB).Create(rr, req)

	if rr.Code != http.StatusCreated { t.Fatalf("expected 201, got %d body %s", rr.Code, rr.Body.String()) }
	var saved models.Agent
	if err := env.DB.Agents().FindOne(context.Background(), bson.M{"tenantId": tenant.ID, "slug": "launch-coach"}).Decode(&saved); err != nil {
		t.Fatalf("expected saved normalized agent: %v", err)
	}
	if saved.CreatedBy != user.ID { t.Fatalf("expected createdBy current user") }
}

func TestArchiveAgent_UpdatesStatusOnlyWithinTenant(t *testing.T) {
	if testing.Short() { t.Skip("skipping integration test in short mode") }
	env := setupTestServer(t); defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	agent := seedAgent(t, env, tenant.ID, user.ID, models.AgentStatusPublished, "published-agent")

	req := env.tenantRequest(t, http.MethodPost, "/api/tenant/agents/"+agent.ID.Hex()+"/archive", nil, user, tenant.ID.Hex())
	req = mux.SetURLVars(req, map[string]string{"agentId": agent.ID.Hex()})
	rr := httptest.NewRecorder()
	NewAgentHandler(env.DB).Archive(rr, req)

	if rr.Code != http.StatusOK { t.Fatalf("expected 200, got %d body %s", rr.Code, rr.Body.String()) }
	var saved models.Agent
	if err := env.DB.Agents().FindOne(context.Background(), bson.M{"_id": agent.ID}).Decode(&saved); err != nil { t.Fatal(err) }
	if saved.Status != models.AgentStatusArchived { t.Fatalf("expected archived, got %s", saved.Status) }
}

func TestAgentHandler_RequiresTenantContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, &models.User{ID: primitive.NewObjectID()}))
	rr := httptest.NewRecorder()
	NewAgentHandler(nil).ListPublic(rr, req)
	if rr.Code != http.StatusBadRequest { t.Fatalf("expected tenant context error, got %d", rr.Code) }
}
```

- [x] **Step 2: Run tests and verify failure**

Run:

```bash
cd backend && go test ./internal/api/handlers -run 'TestPublicListAgents|TestTenantListAgents|TestCreateAgent|TestArchiveAgent|TestAgentHandler'
```

Expected: FAIL because `NewAgentHandler` and DB accessors may not be wired yet if Task 2 has not landed.

- [x] **Step 3: Implement agent handler**

Create `backend/internal/api/handlers/agents.go` with:

```go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"agentstore/internal/db"
	"agentstore/internal/middleware"
	"agentstore/internal/models"
	"agentstore/internal/validation"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AgentHandler struct{ db *db.MongoDB }

func NewAgentHandler(database *db.MongoDB) *AgentHandler { return &AgentHandler{db: database} }

type agentRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Category string `json:"category"`
	Description string `json:"description"`
	Avatar string `json:"avatar"`
	Icon string `json:"icon"`
	Color string `json:"color"`
	Visibility models.AgentVisibility `json:"visibility"`
	SystemPrompt string `json:"systemPrompt"`
	WelcomeMessage string `json:"welcomeMessage"`
	SuggestedPrompts []string `json:"suggestedPrompts"`
	Capabilities []models.AgentCapability `json:"capabilities"`
	CreditCost models.AgentCreditCost `json:"creditCost"`
	ModelConfig models.AgentModelConfig `json:"modelConfig"`
}

func tenantAndUser(r *http.Request) (*models.Tenant, *models.User, bool) {
	ctx := r.Context()
	user, userOK := middleware.GetUserFromContext(ctx)
	tenant, tenantOK := middleware.GetTenantFromContext(ctx)
	return tenant, user, userOK && tenantOK
}

func (h *AgentHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	tenant, _, ok := tenantAndUser(r)
	if !ok { respondWithError(w, http.StatusBadRequest, "Tenant context required"); return }
	cursor, err := h.db.Agents().Find(r.Context(), bson.M{"tenantId": tenant.ID, "status": models.AgentStatusPublished}, options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}}))
	if err != nil { respondWithError(w, http.StatusInternalServerError, "Failed to load agents"); return }
	defer cursor.Close(r.Context())
	var agents []models.Agent
	if err := cursor.All(r.Context(), &agents); err != nil { respondWithError(w, http.StatusInternalServerError, "Failed to decode agents"); return }
	publicAgents := make([]models.PublicAgent, 0, len(agents))
	for _, agent := range agents { publicAgents = append(publicAgents, agent.ToPublic()) }
	respondWithJSON(w, http.StatusOK, publicAgents)
}

func (h *AgentHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	tenant, _, ok := tenantAndUser(r)
	if !ok { respondWithError(w, http.StatusBadRequest, "Tenant context required"); return }
	slug := mux.Vars(r)["slug"]
	var agent models.Agent
	err := h.db.Agents().FindOne(r.Context(), bson.M{"tenantId": tenant.ID, "slug": slug, "status": models.AgentStatusPublished}).Decode(&agent)
	if errors.Is(err, mongo.ErrNoDocuments) { respondWithError(w, http.StatusNotFound, "Agent not found"); return }
	if err != nil { respondWithError(w, http.StatusInternalServerError, "Failed to load agent"); return }
	respondWithJSON(w, http.StatusOK, agent.ToPublic())
}

func (h *AgentHandler) ListTenant(w http.ResponseWriter, r *http.Request) {
	tenant, _, ok := tenantAndUser(r)
	if !ok { respondWithError(w, http.StatusBadRequest, "Tenant context required"); return }
	cursor, err := h.db.Agents().Find(r.Context(), bson.M{"tenantId": tenant.ID, "status": bson.M{"$ne": models.AgentStatusArchived}}, options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}}))
	if err != nil { respondWithError(w, http.StatusInternalServerError, "Failed to load agents"); return }
	defer cursor.Close(r.Context())
	var agents []models.Agent
	if err := cursor.All(r.Context(), &agents); err != nil { respondWithError(w, http.StatusInternalServerError, "Failed to decode agents"); return }
	respondWithJSON(w, http.StatusOK, agents)
}

func (h *AgentHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenant, user, ok := tenantAndUser(r)
	if !ok { respondWithError(w, http.StatusBadRequest, "Tenant context required"); return }
	var req agentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { respondWithError(w, http.StatusBadRequest, "Invalid request body"); return }
	now := time.Now()
	slug := models.NormalizeAgentSlug(req.Slug)
	if slug == "" { slug = models.NormalizeAgentSlug(req.Name) }
	agent := models.Agent{ID: primitive.NewObjectID(), TenantID: tenant.ID, Name: strings.TrimSpace(req.Name), Slug: slug, Category: strings.TrimSpace(req.Category), Description: strings.TrimSpace(req.Description), Avatar: strings.TrimSpace(req.Avatar), Icon: strings.TrimSpace(req.Icon), Color: strings.TrimSpace(req.Color), Status: models.AgentStatusDraft, Visibility: req.Visibility, SystemPrompt: strings.TrimSpace(req.SystemPrompt), WelcomeMessage: strings.TrimSpace(req.WelcomeMessage), SuggestedPrompts: req.SuggestedPrompts, Capabilities: req.Capabilities, CreditCost: req.CreditCost, ModelConfig: req.ModelConfig, CreatedBy: user.ID, CreatedAt: now, UpdatedAt: now}
	if agent.Visibility == "" { agent.Visibility = models.AgentVisibilityPrivate }
	if err := validation.Validate(&agent); err != nil { respondWithError(w, http.StatusBadRequest, err.Error()); return }
	if _, err := h.db.Agents().InsertOne(r.Context(), agent); err != nil { respondWithError(w, http.StatusInternalServerError, "Failed to save agent"); return }
	respondWithJSON(w, http.StatusCreated, agent)
}

func (h *AgentHandler) GetTenant(w http.ResponseWriter, r *http.Request) { h.getTenantAgent(w, r, true) }

func (h *AgentHandler) getTenantAgent(w http.ResponseWriter, r *http.Request, includeArchived bool) {
	tenant, _, ok := tenantAndUser(r)
	if !ok { respondWithError(w, http.StatusBadRequest, "Tenant context required"); return }
	id, err := primitive.ObjectIDFromHex(mux.Vars(r)["agentId"])
	if err != nil { respondWithError(w, http.StatusBadRequest, "Invalid agent ID"); return }
	filter := bson.M{"_id": id, "tenantId": tenant.ID}
	if !includeArchived { filter["status"] = bson.M{"$ne": models.AgentStatusArchived} }
	var agent models.Agent
	err = h.db.Agents().FindOne(r.Context(), filter).Decode(&agent)
	if errors.Is(err, mongo.ErrNoDocuments) { respondWithError(w, http.StatusNotFound, "Agent not found"); return }
	if err != nil { respondWithError(w, http.StatusInternalServerError, "Failed to load agent"); return }
	respondWithJSON(w, http.StatusOK, agent)
}

func (h *AgentHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenant, _, ok := tenantAndUser(r)
	if !ok { respondWithError(w, http.StatusBadRequest, "Tenant context required"); return }
	id, err := primitive.ObjectIDFromHex(mux.Vars(r)["agentId"])
	if err != nil { respondWithError(w, http.StatusBadRequest, "Invalid agent ID"); return }
	var existing models.Agent
	if err := h.db.Agents().FindOne(r.Context(), bson.M{"_id": id, "tenantId": tenant.ID}).Decode(&existing); err != nil {
		respondWithError(w, http.StatusNotFound, "Agent not found"); return
	}
	var req agentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { respondWithError(w, http.StatusBadRequest, "Invalid request body"); return }
	existing.Name = strings.TrimSpace(req.Name)
	existing.Slug = models.NormalizeAgentSlug(req.Slug)
	if existing.Slug == "" { existing.Slug = models.NormalizeAgentSlug(existing.Name) }
	existing.Category = strings.TrimSpace(req.Category)
	existing.Description = strings.TrimSpace(req.Description)
	existing.Avatar = strings.TrimSpace(req.Avatar)
	existing.Icon = strings.TrimSpace(req.Icon)
	existing.Color = strings.TrimSpace(req.Color)
	existing.Visibility = req.Visibility
	existing.SystemPrompt = strings.TrimSpace(req.SystemPrompt)
	existing.WelcomeMessage = strings.TrimSpace(req.WelcomeMessage)
	existing.SuggestedPrompts = req.SuggestedPrompts
	existing.Capabilities = req.Capabilities
	existing.CreditCost = req.CreditCost
	existing.ModelConfig = req.ModelConfig
	existing.UpdatedAt = time.Now()
	if existing.Visibility == "" { existing.Visibility = models.AgentVisibilityPrivate }
	if err := validation.Validate(&existing); err != nil { respondWithError(w, http.StatusBadRequest, err.Error()); return }
	_, err = h.db.Agents().ReplaceOne(r.Context(), bson.M{"_id": id, "tenantId": tenant.ID}, existing)
	if err != nil { respondWithError(w, http.StatusInternalServerError, "Failed to update agent"); return }
	respondWithJSON(w, http.StatusOK, existing)
}

func (h *AgentHandler) Publish(w http.ResponseWriter, r *http.Request) { h.setStatus(w, r, models.AgentStatusPublished) }
func (h *AgentHandler) Archive(w http.ResponseWriter, r *http.Request) { h.setStatus(w, r, models.AgentStatusArchived) }

func (h *AgentHandler) setStatus(w http.ResponseWriter, r *http.Request, status models.AgentStatus) {
	tenant, _, ok := tenantAndUser(r)
	if !ok { respondWithError(w, http.StatusBadRequest, "Tenant context required"); return }
	id, err := primitive.ObjectIDFromHex(mux.Vars(r)["agentId"])
	if err != nil { respondWithError(w, http.StatusBadRequest, "Invalid agent ID"); return }
	result, err := h.db.Agents().UpdateOne(r.Context(), bson.M{"_id": id, "tenantId": tenant.ID}, bson.M{"$set": bson.M{"status": status, "updatedAt": time.Now()}})
	if err != nil { respondWithError(w, http.StatusInternalServerError, "Failed to update agent"); return }
	if result.MatchedCount == 0 { respondWithError(w, http.StatusNotFound, "Agent not found"); return }
	respondWithJSON(w, http.StatusOK, map[string]string{"status": string(status)})
}
```

- [x] **Step 4: Register routes**

In `backend/cmd/server/main.go`, initialize the handler near the chat handler:

```go
	agentHandler := handlers.NewAgentHandler(database)
```

Add public tenant-scoped agent routes after guarded auth routes and before chat:

```go
	agentsAPI := guarded.PathPrefix("/agents").Subrouter()
	agentsAPI.Use(authMiddleware.RequireAuth)
	agentsAPI.Use(tenantMiddleware.RequireTenant)
	agentsAPI.HandleFunc("", agentHandler.ListPublic).Methods("GET")
	agentsAPI.HandleFunc("/{slug}", agentHandler.GetPublic).Methods("GET")
```

Add management routes under the existing `tenantAPI` after tenant settings:

```go
	tenantAgentsRouter := tenantAPI.PathPrefix("/agents").Subrouter()
	tenantAgentsRouter.Use(middleware.RequireRole(models.RoleAdmin))
	tenantAgentsRouter.HandleFunc("", agentHandler.ListTenant).Methods("GET")
	tenantAgentsRouter.HandleFunc("", agentHandler.Create).Methods("POST")
	tenantAgentsRouter.HandleFunc("/{agentId}", agentHandler.GetTenant).Methods("GET")
	tenantAgentsRouter.HandleFunc("/{agentId}", agentHandler.Update).Methods("PUT")
	tenantAgentsRouter.HandleFunc("/{agentId}/publish", agentHandler.Publish).Methods("POST")
	tenantAgentsRouter.HandleFunc("/{agentId}/archive", agentHandler.Archive).Methods("POST")
```

- [x] **Step 5: Temporarily bridge old chat agent endpoints**

In `backend/internal/api/handlers/chat.go`, replace `ListAgents` and `GetAgent` bodies with calls that query persistent agents or move those route registrations to `agentHandler.ListPublic` and `agentHandler.GetPublic` in `server/main.go`:

```go
	chatAPI.HandleFunc("/agents", agentHandler.ListPublic).Methods("GET")
	chatAPI.HandleFunc("/agents/{slug}", agentHandler.GetPublic).Methods("GET")
```

Keep this bridge only for compatibility while frontend switches to `/api/agents`.

- [x] **Step 6: Run agent handler tests**

Run:

```bash
cd backend && go test ./internal/api/handlers -run 'TestPublicListAgents|TestTenantListAgents|TestCreateAgent|TestArchiveAgent|TestAgentHandler'
```

Expected: PASS.

- [x] **Step 7: Commit checkpoint after authorization**

If commit authorization exists, run:

```bash
git add backend/internal/api/handlers/agents.go backend/internal/api/handlers/agents_test.go backend/cmd/server/main.go backend/internal/api/handlers/chat.go
git commit -m "Add tenant-scoped agent APIs"
```

---

## Task 4: Add tenant model settings APIs and model router ~~DONE~~

**Files:**
- Create: `backend/internal/api/handlers/model_settings.go`
- Create: `backend/internal/api/handlers/model_settings_test.go`
- Create: `backend/internal/llm/router.go`
- Create: `backend/internal/llm/router_test.go`
- Modify: `backend/cmd/server/main.go`

- [x] **Step 1: Write model settings handler tests**

Create `backend/internal/api/handlers/model_settings_test.go`:

```go
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agentstore/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestModelProviderAPI_MasksAPIKey(t *testing.T) {
	if testing.Short() { t.Skip("skipping integration test in short mode") }
	env := setupTestServer(t); defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	_, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{ID: primitive.NewObjectID(), TenantID: tenant.ID, Name: "Provider", ProviderType: models.ProviderTypeOpenAICompatible, BaseURL: "https://api.example.com/v1", APIKey: "sk-secret-1234", Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	if err != nil { t.Fatal(err) }

	req := env.tenantRequest(t, http.MethodGet, "/api/tenant/model-providers", nil, user, tenant.ID.Hex())
	rr := httptest.NewRecorder()
	NewModelSettingsHandler(env.DB).ListProviders(rr, req)

	if rr.Code != http.StatusOK { t.Fatalf("expected 200, got %d body %s", rr.Code, rr.Body.String()) }
	if strings.Contains(rr.Body.String(), "sk-secret-1234") { t.Fatalf("leaked raw key: %s", rr.Body.String()) }
	if !strings.Contains(rr.Body.String(), "sk-***1234") { t.Fatalf("expected masked key preview, got %s", rr.Body.String()) }
}

func TestModelProviderAPI_PreservesExistingKeyOnMaskedUpdate(t *testing.T) {
	if testing.Short() { t.Skip("skipping integration test in short mode") }
	env := setupTestServer(t); defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	providerID := primitive.NewObjectID()
	_, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{ID: providerID, TenantID: tenant.ID, Name: "Provider", ProviderType: models.ProviderTypeOpenAICompatible, BaseURL: "https://old.example.com/v1", APIKey: "sk-secret-1234", Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	if err != nil { t.Fatal(err) }

	body := []byte(`{"name":"Provider","providerType":"openai_compatible","baseUrl":"https://new.example.com/v1","apiKey":"sk-***1234","enabled":true}`)
	req := env.tenantRequest(t, http.MethodPut, "/api/tenant/model-providers/"+providerID.Hex(), bytes.NewReader(body), user, tenant.ID.Hex())
	req = mux.SetURLVars(req, map[string]string{"providerId": providerID.Hex()})
	rr := httptest.NewRecorder()
	NewModelSettingsHandler(env.DB).UpdateProvider(rr, req)

	if rr.Code != http.StatusOK { t.Fatalf("expected 200, got %d body %s", rr.Code, rr.Body.String()) }
	var saved models.ModelProvider
	if err := env.DB.ModelProviders().FindOne(context.Background(), bson.M{"_id": providerID}).Decode(&saved); err != nil { t.Fatal(err) }
	if saved.APIKey != "sk-secret-1234" { t.Fatalf("expected key preserved, got %q", saved.APIKey) }
	if saved.BaseURL != "https://new.example.com/v1" { t.Fatalf("expected base URL update") }
}

func TestModelDefaults_UpdateTenantDefaults(t *testing.T) {
	if testing.Short() { t.Skip("skipping integration test in short mode") }
	env := setupTestServer(t); defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	modelID := primitive.NewObjectID()
	providerID := primitive.NewObjectID()
	_, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{ID: providerID, TenantID: tenant.ID, Name: "Provider", ProviderType: models.ProviderTypeOpenAICompatible, BaseURL: "https://api.example.com/v1", APIKey: "sk", Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	if err != nil { t.Fatal(err) }
	_, err = env.DB.ModelConfigs().InsertOne(context.Background(), models.ModelConfig{ID: modelID, TenantID: tenant.ID, ProviderID: providerID, Name: "Text", DisplayName: "Text", Modality: models.ModelModalityText, ModelID: "gpt-4o-mini", Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	if err != nil { t.Fatal(err) }

	body := []byte(`{"defaultTextModelConfigId":"` + modelID.Hex() + `"}`)
	req := env.tenantRequest(t, http.MethodPost, "/api/tenant/model-defaults", bytes.NewReader(body), user, tenant.ID.Hex())
	rr := httptest.NewRecorder()
	NewModelSettingsHandler(env.DB).UpdateDefaults(rr, req)

	if rr.Code != http.StatusOK { t.Fatalf("expected 200, got %d body %s", rr.Code, rr.Body.String()) }
	var saved models.Tenant
	if err := env.DB.Tenants().FindOne(context.Background(), bson.M{"_id": tenant.ID}).Decode(&saved); err != nil { t.Fatal(err) }
	if saved.DefaultTextModelConfigID == nil || *saved.DefaultTextModelConfigID != modelID { t.Fatalf("expected default text model saved") }
}
```

Add `github.com/gorilla/mux` to imports when using `mux.SetURLVars`.

- [x] **Step 2: Write router tests**

Create `backend/internal/llm/router_test.go`:

```go
package llm

import (
	"context"
	"testing"
	"time"

	"agentstore/internal/models"
	"agentstore/internal/testutil"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestRouterResolveTextModel_UsesAgentBindingBeforeTenantDefault(t *testing.T) {
	if testing.Short() { t.Skip("skipping integration test in short mode") }
	env := testutil.SetupMongo(t); defer env.Cleanup()
	ctx := context.Background()
	tenantID := primitive.NewObjectID()
	providerID := primitive.NewObjectID()
	agentModelID := primitive.NewObjectID()
	defaultModelID := primitive.NewObjectID()
	_, _ = env.DB.ModelProviders().InsertOne(ctx, models.ModelProvider{ID: providerID, TenantID: tenantID, Name: "Provider", ProviderType: models.ProviderTypeOpenAICompatible, BaseURL: "https://api.example.com/v1", APIKey: "sk-agent", Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	_, _ = env.DB.ModelConfigs().InsertOne(ctx, models.ModelConfig{ID: agentModelID, TenantID: tenantID, ProviderID: providerID, Name: "Agent", DisplayName: "Agent", Modality: models.ModelModalityText, ModelID: "agent-model", Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	_, _ = env.DB.ModelConfigs().InsertOne(ctx, models.ModelConfig{ID: defaultModelID, TenantID: tenantID, ProviderID: providerID, Name: "Default", DisplayName: "Default", Modality: models.ModelModalityText, ModelID: "default-model", Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	tenant := models.Tenant{ID: tenantID, DefaultTextModelConfigID: &defaultModelID}
	agent := models.Agent{TenantID: tenantID, ModelConfig: models.AgentModelConfig{TextModelID: &agentModelID}}

	snapshot, err := NewRouter(env.DB).ResolveTextModel(ctx, tenant, agent)
	if err != nil { t.Fatalf("resolve text model: %v", err) }
	if snapshot.Model != "agent-model" { t.Fatalf("expected agent model, got %q", snapshot.Model) }
}

func TestRouterResolveTextModel_FallsBackToLegacyLLMConfig(t *testing.T) {
	if testing.Short() { t.Skip("skipping integration test in short mode") }
	env := testutil.SetupMongo(t); defer env.Cleanup()
	ctx := context.Background()
	_, _ = env.DB.LLMConfigs().InsertOne(ctx, models.LLMConfig{Key: models.DefaultLLMConfigKey, APIKey: "legacy-key", BaseURL: "https://legacy.example.com/v1", Model: "legacy-model", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	snapshot, err := NewRouter(env.DB).ResolveTextModel(ctx, models.Tenant{ID: primitive.NewObjectID()}, models.Agent{})
	if err != nil { t.Fatalf("resolve legacy text model: %v", err) }
	if snapshot.Model != "legacy-model" || snapshot.APIKey != "legacy-key" { t.Fatalf("expected legacy config, got %#v", snapshot) }
}
```

- [x] **Step 3: Run tests and verify failure**

Run:

```bash
cd backend && go test ./internal/api/handlers -run 'TestModelProviderAPI|TestModelDefaults' && go test ./internal/llm -run 'TestRouterResolveTextModel'
```

Expected: FAIL because model settings handler and router do not exist.

- [x] **Step 4: Implement model router**

Create `backend/internal/llm/router.go`:

```go
package llm

import (
	"context"
	"errors"
	"fmt"

	"agentstore/internal/db"
	"agentstore/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var ErrModelNotConfigured = errors.New("model is not configured")

type Router struct{ db *db.MongoDB }

func NewRouter(database *db.MongoDB) *Router { return &Router{db: database} }

func (r *Router) ResolveTextModel(ctx context.Context, tenant models.Tenant, agent models.Agent) (RequestConfig, error) {
	if agent.ModelConfig.TextModelID != nil {
		if cfg, err := r.resolveModelConfig(ctx, tenant.ID, *agent.ModelConfig.TextModelID, models.ModelModalityText); err == nil { return cfg, nil } else if !errors.Is(err, mongo.ErrNoDocuments) { return RequestConfig{}, err }
	}
	if tenant.DefaultTextModelConfigID != nil {
		if cfg, err := r.resolveModelConfig(ctx, tenant.ID, *tenant.DefaultTextModelConfigID, models.ModelModalityText); err == nil { return cfg, nil } else if !errors.Is(err, mongo.ErrNoDocuments) { return RequestConfig{}, err }
	}
	var legacy models.LLMConfig
	if err := r.db.LLMConfigs().FindOne(ctx, bson.M{"key": models.DefaultLLMConfigKey, "isActive": true}).Decode(&legacy); err == nil {
		return RequestConfig{APIKey: legacy.APIKey, BaseURL: normalizeBaseURL(legacy.BaseURL), Model: legacy.Model}, nil
	}
	return RequestConfig{}, ErrModelNotConfigured
}

func (r *Router) ResolveImageModel(ctx context.Context, tenant models.Tenant, agent models.Agent) (RequestConfig, error) {
	if agent.ModelConfig.ImageModelID != nil { return r.resolveModelConfig(ctx, tenant.ID, *agent.ModelConfig.ImageModelID, models.ModelModalityImage) }
	if tenant.DefaultImageModelConfigID != nil { return r.resolveModelConfig(ctx, tenant.ID, *tenant.DefaultImageModelConfigID, models.ModelModalityImage) }
	return RequestConfig{}, fmt.Errorf("image %w", ErrModelNotConfigured)
}

func (r *Router) ResolveVideoModel(ctx context.Context, tenant models.Tenant, agent models.Agent) (RequestConfig, error) {
	if agent.ModelConfig.VideoModelID != nil { return r.resolveModelConfig(ctx, tenant.ID, *agent.ModelConfig.VideoModelID, models.ModelModalityVideo) }
	if tenant.DefaultVideoModelConfigID != nil { return r.resolveModelConfig(ctx, tenant.ID, *tenant.DefaultVideoModelConfigID, models.ModelModalityVideo) }
	return RequestConfig{}, fmt.Errorf("video %w", ErrModelNotConfigured)
}

func (r *Router) resolveModelConfig(ctx context.Context, tenantID, modelID primitive.ObjectID, modality models.ModelModality) (RequestConfig, error) {
	var model models.ModelConfig
	if err := r.db.ModelConfigs().FindOne(ctx, bson.M{"_id": modelID, "tenantId": tenantID, "modality": modality, "enabled": true}).Decode(&model); err != nil { return RequestConfig{}, err }
	var provider models.ModelProvider
	if err := r.db.ModelProviders().FindOne(ctx, bson.M{"_id": model.ProviderID, "tenantId": tenantID, "enabled": true}).Decode(&provider); err != nil { return RequestConfig{}, err }
	return RequestConfig{APIKey: provider.APIKey, BaseURL: normalizeBaseURL(provider.BaseURL), Model: model.ModelID}, nil
}
```

- [x] **Step 5: Implement model settings handler**

Create `backend/internal/api/handlers/model_settings.go` with provider list/create/update, model list/create/update, and defaults update. Use `models.MaskAPIKey` when responding. Preserve the existing API key when update receives an empty value or a value equal to the current preview.

Core helper code:

```go
func parseOptionalObjectID(value string) (*primitive.ObjectID, error) {
	if strings.TrimSpace(value) == "" { return nil, nil }
	id, err := primitive.ObjectIDFromHex(value)
	if err != nil { return nil, err }
	return &id, nil
}

func preserveProviderKey(existing models.ModelProvider, incoming string) string {
	incoming = strings.TrimSpace(incoming)
	if incoming == "" || incoming == models.MaskAPIKey(existing.APIKey) {
		return existing.APIKey
	}
	return incoming
}
```

Implement methods with these signatures:

```go
type ModelSettingsHandler struct{ db *db.MongoDB }
func NewModelSettingsHandler(database *db.MongoDB) *ModelSettingsHandler
func (h *ModelSettingsHandler) ListProviders(w http.ResponseWriter, r *http.Request)
func (h *ModelSettingsHandler) CreateProvider(w http.ResponseWriter, r *http.Request)
func (h *ModelSettingsHandler) UpdateProvider(w http.ResponseWriter, r *http.Request)
func (h *ModelSettingsHandler) TestProvider(w http.ResponseWriter, r *http.Request)
func (h *ModelSettingsHandler) ListModels(w http.ResponseWriter, r *http.Request)
func (h *ModelSettingsHandler) CreateModel(w http.ResponseWriter, r *http.Request)
func (h *ModelSettingsHandler) UpdateModel(w http.ResponseWriter, r *http.Request)
func (h *ModelSettingsHandler) UpdateDefaults(w http.ResponseWriter, r *http.Request)
```

`TestProvider` should validate that the provider exists in the current tenant and return:

```go
respondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
```

- [x] **Step 6: Register model settings routes**

In `backend/cmd/server/main.go`, initialize:

```go
	modelSettingsHandler := handlers.NewModelSettingsHandler(database)
```

Add under `tenantAPI`:

```go
	tenantModelRouter := tenantAPI.PathPrefix("").Subrouter()
	tenantModelRouter.Use(middleware.RequireRole(models.RoleAdmin))
	tenantModelRouter.HandleFunc("/model-providers", modelSettingsHandler.ListProviders).Methods("GET")
	tenantModelRouter.HandleFunc("/model-providers", modelSettingsHandler.CreateProvider).Methods("POST")
	tenantModelRouter.HandleFunc("/model-providers/{providerId}", modelSettingsHandler.UpdateProvider).Methods("PUT")
	tenantModelRouter.HandleFunc("/model-providers/{providerId}/test", modelSettingsHandler.TestProvider).Methods("POST")
	tenantModelRouter.HandleFunc("/model-configs", modelSettingsHandler.ListModels).Methods("GET")
	tenantModelRouter.HandleFunc("/model-configs", modelSettingsHandler.CreateModel).Methods("POST")
	tenantModelRouter.HandleFunc("/model-configs/{modelId}", modelSettingsHandler.UpdateModel).Methods("PUT")
	tenantModelRouter.HandleFunc("/model-defaults", modelSettingsHandler.UpdateDefaults).Methods("POST")
```

- [x] **Step 7: Run model tests**

Run:

```bash
cd backend && go test ./internal/api/handlers -run 'TestModelProviderAPI|TestModelDefaults' && go test ./internal/llm -run 'TestRouterResolveTextModel'
```

Expected: PASS.

- [x] **Step 8: Commit checkpoint after authorization**

If commit authorization exists, run:

```bash
git add backend/internal/api/handlers/model_settings.go backend/internal/api/handlers/model_settings_test.go backend/internal/llm/router.go backend/internal/llm/router_test.go backend/cmd/server/main.go
git commit -m "Add tenant model settings and router"
```

---

## Task 5: Add streaming LLM support and streaming chat endpoint ~~DONE~~

**Files:**
- Modify: `backend/internal/llm/openai.go`
- Create: `backend/internal/llm/stream_test.go`
- Modify: `backend/internal/api/handlers/chat.go`
- Modify: `backend/internal/api/handlers/chat_test.go`
- Modify: `backend/cmd/server/main.go`

- [x] **Step 1: Write streaming LLM tests**

Create `backend/internal/llm/stream_test.go`:

```go
package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompleteStreamWithConfig_StreamsDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hel\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	client := NewClient()
	var deltas []string
	full, model, err := client.CompleteStreamWithConfig(context.Background(), RequestConfig{APIKey: "sk", BaseURL: server.URL, Model: "test-model"}, "system", []Message{{Role: "user", Content: "Hi"}}, func(delta string) error {
		deltas = append(deltas, delta)
		return nil
	})
	if err != nil { t.Fatalf("stream completion: %v", err) }
	if model != "test-model" { t.Fatalf("expected model snapshot") }
	if full != "Hello" { t.Fatalf("expected full text Hello, got %q", full) }
	if len(deltas) != 2 || deltas[0] != "Hel" || deltas[1] != "lo" { t.Fatalf("unexpected deltas %#v", deltas) }
}
```

- [x] **Step 2: Write chat streaming credit tests**

Append to `backend/internal/api/handlers/chat_test.go`:

```go
func TestStreamMessage_SuccessDeductsCreditsAfterCompletion(t *testing.T) {
	if testing.Short() { t.Skip("skipping integration test in short mode") }
	env := setupTestServer(t); defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Done\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer provider.Close()
	ctx := context.Background()
	_, _ = env.DB.Tenants().UpdateOne(ctx, bson.M{"_id": tenant.ID}, bson.M{"$set": bson.M{"subscriptionCredits": int64(10)}})
	_, _ = env.DB.LLMConfigs().InsertOne(ctx, models.LLMConfig{Key: models.DefaultLLMConfigKey, APIKey: "sk", BaseURL: provider.URL, Model: "stream-model", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	seedAgent(t, env, tenant.ID, user.ID, models.AgentStatusPublished, "support-agent")

	body := []byte(`{"agentId":"support-agent","message":"Help"}`)
	req := env.tenantRequest(t, http.MethodPost, "/api/chat/stream", bytes.NewReader(body), user, tenant.ID.Hex())
	rr := httptest.NewRecorder()
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB))
	handler.StreamMessage(rr, req)

	if rr.Code != http.StatusOK { t.Fatalf("expected 200, got %d body %s", rr.Code, rr.Body.String()) }
	if !strings.Contains(rr.Body.String(), "event: delta") || !strings.Contains(rr.Body.String(), "Done") { t.Fatalf("expected delta stream, got %s", rr.Body.String()) }
	remaining, err := credits.NewService(env.DB).GetRemainingCredits(ctx, tenant.ID)
	if err != nil { t.Fatal(err) }
	if remaining != 7 { t.Fatalf("expected 3 credits charged after completion, got remaining %d", remaining) }
}

func TestStreamMessage_ProviderFailureDoesNotDeductCredits(t *testing.T) {
	if testing.Short() { t.Skip("skipping integration test in short mode") }
	env := setupTestServer(t); defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "failed", http.StatusInternalServerError) }))
	defer provider.Close()
	ctx := context.Background()
	_, _ = env.DB.Tenants().UpdateOne(ctx, bson.M{"_id": tenant.ID}, bson.M{"$set": bson.M{"subscriptionCredits": int64(10)}})
	_, _ = env.DB.LLMConfigs().InsertOne(ctx, models.LLMConfig{Key: models.DefaultLLMConfigKey, APIKey: "sk", BaseURL: provider.URL, Model: "stream-model", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	seedAgent(t, env, tenant.ID, user.ID, models.AgentStatusPublished, "support-agent")

	body := []byte(`{"agentId":"support-agent","message":"Help"}`)
	req := env.tenantRequest(t, http.MethodPost, "/api/chat/stream", bytes.NewReader(body), user, tenant.ID.Hex())
	rr := httptest.NewRecorder()
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB))
	handler.StreamMessage(rr, req)

	remaining, err := credits.NewService(env.DB).GetRemainingCredits(ctx, tenant.ID)
	if err != nil { t.Fatal(err) }
	if remaining != 10 { t.Fatalf("expected no charge on provider failure, got remaining %d", remaining) }
}
```

Add imports if missing: `fmt`, `agentstore/internal/credits`, and `agentstore/internal/llm`.

- [x] **Step 3: Run streaming tests and verify failure**

Run:

```bash
cd backend && go test ./internal/llm -run TestCompleteStreamWithConfig && go test ./internal/api/handlers -run 'TestStreamMessage_'
```

Expected: FAIL because streaming methods are not implemented and `NewChatHandler` signature has not been updated.

- [x] **Step 4: Add LLM streaming method**

In `backend/internal/llm/openai.go`, add request structs:

```go
type StreamChatRequest struct {
	Model string `json:"model"`
	Messages []Message `json:"messages"`
	Stream bool `json:"stream"`
}

type StreamChunk struct {
	Choices []struct {
		Delta struct { Content string `json:"content"` } `json:"delta"`
	} `json:"choices"`
}
```

Add method:

```go
func (c *Client) CompleteStreamWithConfig(ctx context.Context, config RequestConfig, systemPrompt string, messages []Message, onDelta func(string) error) (string, string, error) {
	apiKey := strings.TrimSpace(config.APIKey)
	baseURL := normalizeBaseURL(config.BaseURL)
	model := strings.TrimSpace(config.Model)
	if apiKey == "" { return "", "", fmt.Errorf("OPENAI_API_KEY is not set") }
	if baseURL == "" { return "", "", fmt.Errorf("OPENAI_BASE_URL is not set") }
	if model == "" { return "", "", fmt.Errorf("OPENAI_MODEL is not set") }
	fullMessages := []Message{{Role: "system", Content: systemPrompt}}
	fullMessages = append(fullMessages, messages...)
	jsonBody, err := json.Marshal(StreamChatRequest{Model: model, Messages: fullMessages, Stream: true})
	if err != nil { return "", "", fmt.Errorf("failed to marshal request: %w", err) }
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil { return "", "", fmt.Errorf("failed to create request: %w", err) }
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := (&http.Client{}).Do(req)
	if err != nil { return "", "", fmt.Errorf("failed to send request: %w", err) }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { return "", "", fmt.Errorf("API request failed with status: %d", resp.StatusCode) }
	scanner := bufio.NewScanner(resp.Body)
	var builder strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") { continue }
		if !strings.HasPrefix(line, "data:") { continue }
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" { return builder.String(), model, nil }
		var chunk StreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil { return "", "", fmt.Errorf("failed to decode stream chunk: %w", err) }
		if len(chunk.Choices) == 0 { continue }
		delta := chunk.Choices[0].Delta.Content
		if delta == "" { continue }
		builder.WriteString(delta)
		if onDelta != nil {
			if err := onDelta(delta); err != nil { return "", "", err }
		}
	}
	if err := scanner.Err(); err != nil { return "", "", err }
	return builder.String(), model, nil
}
```

Add `bufio` to imports.

- [x] **Step 5: Update ChatHandler dependencies**

Change `ChatHandler` struct in `backend/internal/api/handlers/chat.go`:

```go
type ChatHandler struct {
	db *db.MongoDB
	creditsSvc *credits.Service
	llmClient *llm.Client
	modelRouter *llm.Router
}
```

Change constructor:

```go
func NewChatHandler(database *db.MongoDB, creditsSvc *credits.Service, llmClient *llm.Client, modelRouter *llm.Router) *ChatHandler {
	return &ChatHandler{db: database, creditsSvc: creditsSvc, llmClient: llmClient, modelRouter: modelRouter}
}
```

Update all call sites to pass `llm.NewRouter(database)`.

- [x] **Step 6: Resolve persistent published agents in chat**

Add helper in `chat.go`:

```go
func (h *ChatHandler) getPublishedAgent(ctx context.Context, tenantID primitive.ObjectID, value string) (*models.Agent, error) {
	filter := bson.M{"tenantId": tenantID, "status": models.AgentStatusPublished}
	if id, err := primitive.ObjectIDFromHex(value); err == nil { filter["_id"] = id } else { filter["slug"] = value }
	var agent models.Agent
	if err := h.db.Agents().FindOne(ctx, filter).Decode(&agent); err != nil { return nil, err }
	return &agent, nil
}
```

Replace static catalog lookup in `SendMessage` with this helper. Use `agent.TextCreditCost()` and `agent.SystemPrompt`.

- [x] **Step 7: Add SSE writer helper and StreamMessage**

Add to `chat.go`:

```go
func writeSSE(w http.ResponseWriter, event string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil { return err }
	_, err = w.Write([]byte("event: " + event + "\n" + "data: " + string(data) + "\n\n"))
	if flusher, ok := w.(http.Flusher); ok { flusher.Flush() }
	return err
}
```

Implement `StreamMessage` with the same validation/conversation logic as `SendMessage`, but save the user message and assistant placeholder before calling `CompleteStreamWithConfig`. On success, update assistant content/status to completed and deduct credits. On provider error, update status to error and emit `error`. If `errors.Is(ctx.Err(), context.Canceled)`, update status to interrupted and return without charging.

Use these response payloads:

```go
map[string]string{"conversationId": conversationID.Hex(), "messageId": assistantMessageID.Hex()}
map[string]string{"text": delta}
map[string]interface{}{"conversationId": conversationID.Hex(), "messageId": assistantMessageID.Hex(), "creditsCharged": creditCost, "remainingCredits": remainingCredits, "model": usedModel}
map[string]string{"message": "AI service is temporarily unavailable. Please try again."}
```

- [x] **Step 8: Add coming-soon generation endpoints**

Add handlers to `chat.go`:

```go
func (h *ChatHandler) GenerateImage(w http.ResponseWriter, r *http.Request) {
	respondWithError(w, http.StatusNotImplemented, "Image generation is coming soon and no credits were charged.")
}

func (h *ChatHandler) GenerateVideo(w http.ResponseWriter, r *http.Request) {
	respondWithError(w, http.StatusNotImplemented, "Video generation is coming soon and no credits were charged.")
}
```

- [x] **Step 9: Register streaming and generation routes**

In `backend/cmd/server/main.go`, update chat handler construction:

```go
	chatModelRouter := llm.NewRouter(database)
	chatHandler := handlers.NewChatHandler(database, chatCreditsService, chatLLMClient, chatModelRouter)
```

Add routes:

```go
	chatAPI.HandleFunc("/stream", chatHandler.StreamMessage).Methods("POST")
	agentsAPI.HandleFunc("/{agentId}/generate-image", chatHandler.GenerateImage).Methods("POST")
	agentsAPI.HandleFunc("/{agentId}/generate-video", chatHandler.GenerateVideo).Methods("POST")
```

- [x] **Step 10: Run streaming tests**

Run:

```bash
cd backend && go test ./internal/llm -run TestCompleteStreamWithConfig && go test ./internal/api/handlers -run 'TestStreamMessage_'
```

Expected: PASS.

- [x] **Step 11: Commit checkpoint after authorization**

If commit authorization exists, run:

```bash
git add backend/internal/llm/openai.go backend/internal/llm/stream_test.go backend/internal/api/handlers/chat.go backend/internal/api/handlers/chat_test.go backend/cmd/server/main.go
git commit -m "Add streaming chat endpoint"
```

---

## Task 6: Migrate visible routing and storage from AgentStore to AgentStore ~~DONE~~

**Files:**
- Create: `frontend/src/utils/storageKeys.ts`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/AdminLayout.tsx`
- Modify: `frontend/src/components/AdminLayout.test.tsx`
- Modify: `frontend/src/components/Layout.tsx`
- Modify: `frontend/src/components/BrandingThemeInjector.tsx`
- Modify: `frontend/src/contexts/BrandingContext.tsx`
- Modify: `frontend/src/contexts/AuthContext.tsx`
- Modify: `frontend/src/contexts/TenantContext.tsx`
- Modify: `frontend/src/contexts/ThemeContext.tsx`
- Modify: `frontend/src/hooks/useTelemetry.ts`
- Modify: admin pages containing `/last` links.

- [x] **Step 1: Update AdminLayout tests first**

In `frontend/src/components/AdminLayout.test.tsx`, change test entries from `/last` to `/admin` and route path from `/last` to `/admin`:

```tsx
function renderAdminLayout(initialEntry = '/admin/llm-config') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/admin" element={<AdminLayout />}>
          <Route path="llm-config" element={<div>LLM Config Outlet</div>} />
          <Route index element={<div>Admin Outlet</div>} />
        </Route>
        <Route path="/dashboard" element={<LocationDisplay />} />
      </Routes>
    </MemoryRouter>
  );
}
```

Update calls:

```tsx
renderAdminLayout('/admin/llm-config');
renderAdminLayout('/admin');
```

- [x] **Step 2: Run AdminLayout test and verify failure**

Run:

```bash
cd frontend && npx vitest run src/components/AdminLayout.test.tsx
```

Expected: FAIL because `AdminLayout` still links to `/last`.

- [x] **Step 3: Create storage key helper**

Create `frontend/src/utils/storageKeys.ts`:

```ts
export const storageKeys = {
  accessToken: 'agentstore_access_token',
  refreshToken: 'agentstore_refresh_token',
  activeTenant: 'agentstore_active_tenant',
  theme: 'agentstore_theme',
  impersonating: 'agentstore_impersonating',
  sessionId: 'agentstore_session_id',
} as const;

const legacyKeys: Record<string, string> = {
  [storageKeys.accessToken]: 'agentstore_access_token',
  [storageKeys.refreshToken]: 'agentstore_refresh_token',
  [storageKeys.activeTenant]: 'agentstore_active_tenant',
  [storageKeys.theme]: 'agentstore_theme',
  [storageKeys.impersonating]: 'agentstore_impersonating',
  [storageKeys.sessionId]: 'agentstore_session_id',
};

export function getStoredValue(key: string) {
  const current = localStorage.getItem(key) ?? sessionStorage.getItem(key);
  if (current !== null) return current;
  const legacy = legacyKeys[key];
  if (!legacy) return null;
  return localStorage.getItem(legacy) ?? sessionStorage.getItem(legacy);
}

export function setStoredValue(key: string, value: string, storage: Storage = localStorage) {
  storage.setItem(key, value);
  const legacy = legacyKeys[key];
  if (legacy) storage.removeItem(legacy);
}

export function removeStoredValue(key: string, storage: Storage = localStorage) {
  storage.removeItem(key);
  const legacy = legacyKeys[key];
  if (legacy) storage.removeItem(legacy);
}
```

- [x] **Step 4: Update routes in App**

In `frontend/src/App.tsx`, add a redirect component:

```tsx
function LastAdminRedirect() {
  const location = useLocation();
  const nextPath = location.pathname.replace(/^\/last/, '/admin');
  return <Navigate to={`${nextPath}${location.search}`} replace />;
}
```

Change admin route block:

```tsx
<Route path="/chat" element={<Navigate to="/dashboard" replace />} />
<Route path="/admin" element={<AdminLayout />}>...</Route>
<Route path="/last/*" element={<LastAdminRedirect />} />
```

Change every admin child path under `/last` to live under `/admin` without changing child names.

- [x] **Step 5: Update admin layout links**

In `frontend/src/components/AdminLayout.tsx`, replace nav item paths:

```tsx
const navItems = [
  { path: '/admin', icon: LayoutDashboard, label: 'Dashboard' },
  { path: '/admin/messages', icon: Mail, label: 'Messages' },
  { path: '/admin/users', icon: Users, label: 'Users' },
  { path: '/admin/tenants', icon: Building2, label: 'Tenants' },
  { path: '/admin/members', icon: UserPlus, label: 'Root Members' },
  { path: '/admin/plans', icon: CreditCard, label: 'Plans' },
  { path: '/admin/financial', icon: DollarSign, label: 'Financial' },
  { path: '/admin/pm', icon: BarChart3, label: 'Product' },
  { path: '/admin/promotions', icon: Tag, label: 'Promotions' },
  { path: '/admin/announcements', icon: Megaphone, label: 'Announcements' },
  { path: '/admin/health', icon: Activity, label: 'System Health' },
  { path: '/admin/logs', icon: FileText, label: 'Logs' },
  { path: '/admin/config', icon: Settings, label: 'Configuration' },
  { path: '/admin/branding', icon: Paintbrush, label: 'Branding' },
  { path: '/admin/api', icon: Code2, label: 'API' },
  { path: '/admin/about', icon: Info, label: 'About' },
];
```

Keep root-owner LLM config hidden from primary nav if model settings move to tenant settings.

- [x] **Step 6: Update app shell links and branding defaults**

In `Layout.tsx`:

```tsx
const defaultNavItems = [
  { path: '/dashboard', icon: MessageCircle, label: 'Agents' },
  ...(showTeam ? [{ path: '/team', icon: Users, label: 'Team' }] : []),
  { path: '/plan', icon: CreditCard, label: 'Plan' },
  { path: '/settings', icon: Settings, label: 'Settings' },
];
const appName = branding.appName || 'AgentStore';
```

Change admin link:

```tsx
to="/admin"
location.pathname.startsWith('/admin')
```

In `BrandingContext.tsx`:

```ts
appName: 'AgentStore',
tagline: 'Launch, sell, and monetize AI agents',
```

In `BrandingThemeInjector.tsx`:

```ts
const isAdmin = location.pathname.startsWith('/admin');
```

- [x] **Step 7: Update localStorage consumers**

Use `storageKeys`, `getStoredValue`, `setStoredValue`, and `removeStoredValue` in:

- `AuthContext.tsx`
- `TenantContext.tsx`
- `ThemeContext.tsx`
- `useTelemetry.ts`
- `api/client.ts`
- `ImpersonationBanner.tsx`
- admin `UsersPage.tsx` impersonation code

Example replacement in `AuthContext.tsx`:

```ts
import { getStoredValue, removeStoredValue, setStoredValue, storageKeys } from '../utils/storageKeys';

const ACCESS_TOKEN_KEY = storageKeys.accessToken;
const REFRESH_TOKEN_KEY = storageKeys.refreshToken;
```

Then replace direct `localStorage.getItem`, `setItem`, and `removeItem` calls for those keys with the helper functions.

- [x] **Step 8: Replace hardcoded `/last` links in admin pages**

Run:

```bash
grep -RIn "/last" frontend/src frontend/e2e --exclude-dir=node_modules
```

Update each result to `/admin`, including:

- `frontend/src/pages/admin/DashboardPage.tsx`
- `frontend/src/pages/admin/TenantProfilePage.tsx`
- `frontend/src/pages/admin/LogsPage.tsx`
- `frontend/src/pages/admin/UserProfilePage.tsx`
- `frontend/src/pages/admin/TenantsPage.tsx`
- `frontend/src/pages/admin/UsersPage.tsx`
- `frontend/e2e/admin.spec.ts`

- [x] **Step 9: Run admin route tests**

Run:

```bash
cd frontend && npx vitest run src/components/AdminLayout.test.tsx
```

Expected: PASS.

- [x] **Step 10: Commit checkpoint after authorization**

If commit authorization exists, run:

```bash
git add frontend/src/utils/storageKeys.ts frontend/src/App.tsx frontend/src/components/AdminLayout.tsx frontend/src/components/AdminLayout.test.tsx frontend/src/components/Layout.tsx frontend/src/components/BrandingThemeInjector.tsx frontend/src/contexts frontend/src/hooks frontend/src/api/client.ts frontend/src/pages/admin frontend/e2e/admin.spec.ts
git commit -m "Migrate visible admin routing to AgentStore"
```

---

## Task 7: Update frontend API types, client, and dashboard ~~DONE~~

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/client.ts`
- Modify: `frontend/src/pages/app/DashboardPage.tsx`
- Modify: `frontend/src/pages/app/DashboardPage.test.tsx`

- [x] **Step 1: Update frontend agent tests**

In `frontend/src/pages/app/DashboardPage.test.tsx`, update the mock agent shape to use:

```ts
const mockAgent = {
  id: 'agent-1',
  name: 'Growth Strategist',
  slug: 'growth-strategist',
  category: 'Marketing',
  description: 'Plans launch and growth work',
  avatar: '',
  icon: 'rocket',
  color: '#7C3AED',
  visibility: 'private' as const,
  welcomeMessage: 'Tell me about your launch.',
  suggestedPrompts: ['Draft a launch plan', 'Review my pricing page'],
  capabilities: ['text_chat'] as const,
  creditCost: { textMessageCredits: 3, imageGenerationCredits: 0, videoGenerationCredits: 0 },
  createdAt: '2026-05-21T12:00:00.000Z',
  updatedAt: '2026-05-21T12:00:00.000Z',
};
```

Assert visible copy:

```ts
expect(await screen.findByText('Launch your AgentStore')).toBeInTheDocument();
expect(screen.getByText('3 credits/message')).toBeInTheDocument();
expect(screen.getByText('Text chat')).toBeInTheDocument();
```

- [x] **Step 2: Run dashboard test and verify failure**

Run:

```bash
cd frontend && npx vitest run src/pages/app/DashboardPage.test.tsx
```

Expected: FAIL because the component still expects `examples` and `creditCost` number.

- [x] **Step 3: Update types**

In `frontend/src/types/index.ts`, replace the Agent/chat section with:

```ts
export type AgentStatus = 'draft' | 'published' | 'archived';
export type AgentVisibility = 'private' | 'public';
export type AgentCapability = 'text_chat' | 'image_generation' | 'video_generation';
export type ModelModality = 'text' | 'image' | 'video';
export type ProviderType = 'openai_compatible' | 'anthropic' | 'gemini';

export interface AgentCreditCost {
  textMessageCredits: number;
  imageGenerationCredits: number;
  videoGenerationCredits: number;
}

export interface AgentModelConfig {
  textModelId?: string;
  imageModelId?: string;
  videoModelId?: string;
}

export interface Agent {
  id: string;
  name: string;
  slug: string;
  category: string;
  description: string;
  avatar: string;
  icon: string;
  color: string;
  status?: AgentStatus;
  visibility: AgentVisibility;
  systemPrompt?: string;
  welcomeMessage: string;
  suggestedPrompts: string[];
  capabilities: AgentCapability[];
  creditCost: AgentCreditCost;
  modelConfig?: AgentModelConfig;
  createdBy?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ModelProvider {
  id: string;
  tenantId: string;
  name: string;
  providerType: ProviderType;
  baseUrl: string;
  apiKeyPreview: string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface ModelConfig {
  id: string;
  tenantId: string;
  providerId: string;
  name: string;
  displayName: string;
  modality: ModelModality;
  modelId: string;
  defaultParams: Record<string, unknown>;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export type ChatMessageStatus = 'generating' | 'completed' | 'error' | 'interrupted';
```

Update `ChatMessage`:

```ts
status?: ChatMessageStatus;
```

Update `ChatResponse`:

```ts
model: string;
```

- [x] **Step 4: Update API client methods**

In `frontend/src/api/client.ts`, change `agentsApi`:

```ts
export const agentsApi = {
  list: () => api.get<Agent[]>('/agents').then(r => r.data),
  get: (slug: string) => api.get<Agent>(`/agents/${slug}`).then(r => r.data),
};
```

Add tenant admin APIs:

```ts
export const tenantAgentsApi = {
  list: () => api.get<Agent[]>('/tenant/agents').then(r => r.data),
  create: (data: Partial<Agent>) => api.post<Agent>('/tenant/agents', data).then(r => r.data),
  update: (id: string, data: Partial<Agent>) => api.put<Agent>(`/tenant/agents/${id}`, data).then(r => r.data),
  publish: (id: string) => api.post(`/tenant/agents/${id}/publish`).then(r => r.data),
  archive: (id: string) => api.post(`/tenant/agents/${id}/archive`).then(r => r.data),
};

export const tenantModelsApi = {
  listProviders: () => api.get<ModelProvider[]>('/tenant/model-providers').then(r => r.data),
  createProvider: (data: { name: string; providerType: ProviderType; baseUrl: string; apiKey: string; enabled: boolean }) => api.post<ModelProvider>('/tenant/model-providers', data).then(r => r.data),
  updateProvider: (id: string, data: { name: string; providerType: ProviderType; baseUrl: string; apiKey: string; enabled: boolean }) => api.put<ModelProvider>(`/tenant/model-providers/${id}`, data).then(r => r.data),
  testProvider: (id: string) => api.post(`/tenant/model-providers/${id}/test`).then(r => r.data),
  listModels: () => api.get<ModelConfig[]>('/tenant/model-configs').then(r => r.data),
  createModel: (data: Partial<ModelConfig>) => api.post<ModelConfig>('/tenant/model-configs', data).then(r => r.data),
  updateModel: (id: string, data: Partial<ModelConfig>) => api.put<ModelConfig>(`/tenant/model-configs/${id}`, data).then(r => r.data),
  updateDefaults: (data: { defaultTextModelConfigId?: string; defaultImageModelConfigId?: string; defaultVideoModelConfigId?: string }) => api.post('/tenant/model-defaults', data).then(r => r.data),
};
```

- [x] **Step 5: Update DashboardPage UI**

In `DashboardPage.tsx`:

- Use `agent.slug` for chat navigation: `navigate(`/chat/${agent.slug}`)`.
- Replace `agent.examples` with `agent.suggestedPrompts`.
- Replace `agent.creditCost` with `agent.creditCost.textMessageCredits`.
- Add capability badges:

```tsx
const capabilityLabels: Record<string, string> = {
  text_chat: 'Text chat',
  image_generation: 'Image',
  video_generation: 'Video',
};
```

Hero copy should include:

```tsx
<h1>Launch your AgentStore</h1>
<p>Choose a tenant-published agent, start a conversation, and turn specialist AI workflows into a product your team can sell and monetize.</p>
```

- [x] **Step 6: Run dashboard test**

Run:

```bash
cd frontend && npx vitest run src/pages/app/DashboardPage.test.tsx
```

Expected: PASS.

- [x] **Step 7: Commit checkpoint after authorization**

If commit authorization exists, run:

```bash
git add frontend/src/types/index.ts frontend/src/api/client.ts frontend/src/pages/app/DashboardPage.tsx frontend/src/pages/app/DashboardPage.test.tsx
git commit -m "Update AgentStore dashboard APIs"
```

---

## Task 8: Add tenant Agent and Model settings UI ~~DONE~~

**Files:**
- Modify: `frontend/src/pages/app/SettingsPage.tsx`
- Create: `frontend/src/pages/app/settings/AgentsTab.tsx`
- Create: `frontend/src/pages/app/settings/AgentsTab.test.tsx`
- Create: `frontend/src/pages/app/settings/ModelSettingsTab.tsx`
- Create: `frontend/src/pages/app/settings/ModelSettingsTab.test.tsx`
- Modify: `frontend/src/App.tsx`

- [x] **Step 1: Write AgentsTab validation test**

Create `frontend/src/pages/app/settings/AgentsTab.test.tsx`:

```tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import AgentsTab from './AgentsTab';

const apiMocks = vi.hoisted(() => ({ list: vi.fn(), create: vi.fn(), update: vi.fn(), publish: vi.fn(), archive: vi.fn() }));
vi.mock('../../../api/client', () => ({ tenantAgentsApi: apiMocks, tenantModelsApi: { listModels: vi.fn().mockResolvedValue([]) } }));

function renderAgentsTab() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={queryClient}><AgentsTab /></QueryClientProvider>);
}

describe('AgentsTab', () => {
  beforeEach(() => {
    Object.values(apiMocks).forEach((mock) => mock.mockReset());
    apiMocks.list.mockResolvedValue([]);
    apiMocks.create.mockResolvedValue({});
  });

  it('does not save without required fields', async () => {
    const user = userEvent.setup();
    renderAgentsTab();
    await screen.findByText('No agents yet');
    await user.click(screen.getByRole('button', { name: /new agent/i }));
    await user.click(screen.getByRole('button', { name: /save agent/i }));
    expect(await screen.findByText('Name is required.')).toBeInTheDocument();
    expect(screen.getByText('System prompt is required.')).toBeInTheDocument();
    expect(apiMocks.create).not.toHaveBeenCalled();
  });
});
```

- [x] **Step 2: Write ModelSettingsTab masking test**

Create `frontend/src/pages/app/settings/ModelSettingsTab.test.tsx`:

```tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ModelSettingsTab from './ModelSettingsTab';

const apiMocks = vi.hoisted(() => ({
  listProviders: vi.fn(), createProvider: vi.fn(), updateProvider: vi.fn(), testProvider: vi.fn(),
  listModels: vi.fn(), createModel: vi.fn(), updateModel: vi.fn(), updateDefaults: vi.fn(),
}));
vi.mock('../../../api/client', () => ({ tenantModelsApi: apiMocks }));

function renderModelSettingsTab() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={queryClient}><ModelSettingsTab /></QueryClientProvider>);
}

describe('ModelSettingsTab', () => {
  beforeEach(() => {
    apiMocks.listProviders.mockResolvedValue([{ id: 'provider-1', tenantId: 'tenant-1', name: 'OpenAI', providerType: 'openai_compatible', baseUrl: 'https://api.example.com/v1', apiKeyPreview: 'sk-***1234', enabled: true, createdAt: '', updatedAt: '' }]);
    apiMocks.listModels.mockResolvedValue([]);
  });

  it('shows masked provider keys without raw secrets', async () => {
    renderModelSettingsTab();
    expect(await screen.findByText('sk-***1234')).toBeInTheDocument();
    expect(screen.queryByText('sk-secret-1234')).not.toBeInTheDocument();
  });
});
```

- [x] **Step 3: Run settings tests and verify failure**

Run:

```bash
cd frontend && npx vitest run src/pages/app/settings/AgentsTab.test.tsx src/pages/app/settings/ModelSettingsTab.test.tsx
```

Expected: FAIL because the tabs do not exist.

- [x] **Step 4: Implement AgentsTab**

Create `frontend/src/pages/app/settings/AgentsTab.tsx` with:

- Query `tenantAgentsApi.list()` and `tenantModelsApi.listModels()`.
- Button `New Agent` opens editor.
- Required validation for name, category, description, systemPrompt, and at least one capability.
- Save calls `tenantAgentsApi.create` or `tenantAgentsApi.update`.
- Publish/archive buttons call matching APIs.

Use this initial form state:

```tsx
const emptyForm = {
  name: '', slug: '', category: '', description: '', avatar: '', icon: 'bot', color: '#7C3AED',
  visibility: 'private' as const, systemPrompt: '', welcomeMessage: '', suggestedPrompts: [''],
  capabilities: ['text_chat'] as AgentCapability[],
  creditCost: { textMessageCredits: 1, imageGenerationCredits: 0, videoGenerationCredits: 0 },
  modelConfig: {},
};
```

Use these error messages exactly for the test:

```tsx
Name is required.
System prompt is required.
```

- [x] **Step 5: Implement ModelSettingsTab**

Create `frontend/src/pages/app/settings/ModelSettingsTab.tsx` with:

- Query `tenantModelsApi.listProviders()` and `tenantModelsApi.listModels()`.
- Provider card shows name, provider type, base URL, masked `apiKeyPreview`, enabled state.
- Simple provider form for name/provider/baseUrl/apiKey/enabled.
- Simple model form for provider/modality/modelId/displayName/enabled.
- Default text/image/video selectors from model list.
- Test connection button calls `tenantModelsApi.testProvider(provider.id)`.

- [x] **Step 6: Update SettingsPage to expose route-driven tabs**

In `SettingsPage.tsx`, import `useLocation`, `useNavigate`, `AgentsTab`, and `ModelSettingsTab`. Include tabs:

```tsx
{ key: 'agents' as const, label: 'Agents', path: '/settings/agents' },
{ key: 'models' as const, label: 'Models', path: '/settings/models' },
```

Resolve active tab from pathname:

```tsx
const pathTab = location.pathname.endsWith('/agents') ? 'agents' : location.pathname.endsWith('/models') ? 'models' : tab;
```

Render:

```tsx
{pathTab === 'agents' && <AgentsTab />}
{pathTab === 'models' && <ModelSettingsTab />}
```

- [x] **Step 7: Add settings subroutes**

In `App.tsx`, add protected layout routes:

```tsx
<Route path="/settings/agents" element={<SettingsPage />} />
<Route path="/settings/models" element={<SettingsPage />} />
```

- [x] **Step 8: Run settings tests**

Run:

```bash
cd frontend && npx vitest run src/pages/app/settings/AgentsTab.test.tsx src/pages/app/settings/ModelSettingsTab.test.tsx
```

Expected: PASS.

- [x] **Step 9: Commit checkpoint after authorization**

If commit authorization exists, run:

```bash
git add frontend/src/pages/app/SettingsPage.tsx frontend/src/pages/app/settings/AgentsTab.tsx frontend/src/pages/app/settings/AgentsTab.test.tsx frontend/src/pages/app/settings/ModelSettingsTab.tsx frontend/src/pages/app/settings/ModelSettingsTab.test.tsx frontend/src/App.tsx
git commit -m "Add tenant AgentStore settings UI"
```

---

## Task 9: Update chat UI for streaming and mobile usability ~~DONE~~

**Files:**
- Modify: `frontend/src/api/client.ts`
- Modify: `frontend/src/pages/app/ChatPage.tsx`
- Modify: `frontend/src/pages/app/ChatPage.test.tsx`

- [x] **Step 1: Write chat streaming tests**

In `frontend/src/pages/app/ChatPage.test.tsx`, update mock agent shape to the new `Agent` type and add tests:

```tsx
it('redirects /chat without an agent to the dashboard', async () => {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={['/chat']}>
        <Routes>
          <Route path="/chat" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<div>Agent dashboard</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
  expect(await screen.findByText('Agent dashboard')).toBeInTheDocument();
});

it('shows streamed assistant deltas while sending', async () => {
  const user = userEvent.setup();
  apiMocks.conversations.mockResolvedValue([]);
  apiMocks.messages.mockResolvedValue([]);
  const encoder = new TextEncoder();
  global.fetch = vi.fn().mockResolvedValue({
    ok: true,
    body: new ReadableStream({
      start(controller) {
        controller.enqueue(encoder.encode('event: message_start\ndata: {"conversationId":"conv-stream","messageId":"msg-stream"}\n\n'));
        controller.enqueue(encoder.encode('event: delta\ndata: {"text":"Hel"}\n\n'));
        controller.enqueue(encoder.encode('event: delta\ndata: {"text":"lo"}\n\n'));
        controller.enqueue(encoder.encode('event: message_done\ndata: {"conversationId":"conv-stream","messageId":"msg-stream","creditsCharged":3,"remainingCredits":17,"model":"stream-model"}\n\n'));
        controller.close();
      },
    }),
    headers: new Headers(),
  } as Response);

  renderChatPage('/chat/growth-strategist');
  const textbox = await screen.findByRole('textbox');
  await user.type(textbox, 'Help me launch');
  await user.keyboard('{Enter}');

  expect(await screen.findByText('Hello')).toBeInTheDocument();
  expect(screen.queryByRole('button', { name: /send/i })).not.toBeDisabled();
});
```

- [x] **Step 2: Run chat tests and verify failure**

Run:

```bash
cd frontend && npx vitest run src/pages/app/ChatPage.test.tsx
```

Expected: FAIL because streaming client/UI is not implemented.

- [x] **Step 3: Add stream helper to API client**

In `frontend/src/api/client.ts`, export:

```ts
export type ChatStreamEvent =
  | { event: 'message_start'; data: { conversationId: string; messageId: string } }
  | { event: 'delta'; data: { text: string } }
  | { event: 'message_done'; data: { conversationId: string; messageId: string; creditsCharged: number; remainingCredits: number; model: string } }
  | { event: 'error'; data: { message: string } };

export async function streamChat(data: ChatRequest, onEvent: (event: ChatStreamEvent) => void, signal?: AbortSignal) {
  const response = await fetch('/api/chat/stream', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(api.defaults.headers.common.Authorization ? { Authorization: String(api.defaults.headers.common.Authorization) } : {}),
      ...(api.defaults.headers.common['X-Tenant-ID'] ? { 'X-Tenant-ID': String(api.defaults.headers.common['X-Tenant-ID']) } : {}),
    },
    body: JSON.stringify(data),
    signal,
  });
  if (!response.ok || !response.body) throw new Error('Unable to start chat stream');
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const events = buffer.split('\n\n');
    buffer = events.pop() ?? '';
    for (const raw of events) {
      const lines = raw.split('\n');
      const event = lines.find((line) => line.startsWith('event: '))?.slice(7) as ChatStreamEvent['event'] | undefined;
      const dataLine = lines.find((line) => line.startsWith('data: '));
      if (!event || !dataLine) continue;
      onEvent({ event, data: JSON.parse(dataLine.slice(6)) } as ChatStreamEvent);
    }
  }
}

export const chatApi = {
  send: (data: ChatRequest) => api.post<ChatResponse>('/chat', data).then(r => r.data),
  stream: streamChat,
  conversations: (agentId?: string) => api.get<Conversation[]>('/chat/conversations', { params: { agentId } }).then(r => r.data),
  messages: (conversationId: string) => api.get<ChatMessage[]>(`/chat/conversations/${conversationId}/messages`).then(r => r.data),
};
```

- [x] **Step 4: Update ChatPage for streaming state**

In `ChatPage.tsx`:

- Rename route param to `agentSlug` or continue `agentId` but pass it to `agentsApi.get(agentId)`.
- Use `agent.slug` in query keys where possible.
- Replace send mutation with local `isStreaming`, `streamingAssistantMessage`, `abortControllerRef`, and `handleSendMessage` that calls `chatApi.stream`.
- Append deltas to a temporary assistant message.
- Add Stop button:

```tsx
<button type="button" onClick={() => abortControllerRef.current?.abort()} disabled={!isStreaming}>Stop generating</button>
```

- Disable textarea while `isStreaming`.
- Add mobile drawer state:

```tsx
const [sidebarOpen, setSidebarOpen] = useState(false);
```

- Use classes that collapse conversations on small screens:

```tsx
<aside className={`${sidebarOpen ? 'fixed inset-0 z-40 block bg-dark-950 p-4' : 'hidden'} lg:static lg:block lg:w-80`}>
```

- Composer wrapper should include:

```tsx
className="sticky bottom-0 bg-dark-950/95 pb-[max(1rem,env(safe-area-inset-bottom))] pt-3"
```

- Suggested prompts container should include:

```tsx
className="flex gap-2 overflow-x-auto pb-2"
```

- [x] **Step 5: Run chat tests**

Run:

```bash
cd frontend && npx vitest run src/pages/app/ChatPage.test.tsx
```

Expected: PASS.

- [x] **Step 6: Commit checkpoint after authorization**

If commit authorization exists, run:

```bash
git add frontend/src/api/client.ts frontend/src/pages/app/ChatPage.tsx frontend/src/pages/app/ChatPage.test.tsx
git commit -m "Add streaming chat UI"
```

---

## Task 10: Rebrand docs, config defaults, and remove static catalog dependency ~~DONE~~

**Files:**
- Modify: `README.md`
- Create or modify: `.env.example`
- Modify: `backend/config/dev.yaml`
- Modify: `backend/config/dev.example.yaml`
- Modify: `backend/config/test.yaml`
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/configstore/seed.go`
- Modify: `backend/internal/api/handlers/branding.go`
- Modify: `backend/internal/api/handlers/auth.go`
- Modify: `backend/internal/version/check.go`
- Modify: `backend/cmd/server/main.go`
- Remove: `backend/internal/agents/catalog.go`
- Remove: `backend/internal/agents/catalog_test.go`

- [ ] **Step 1: Search remaining visible old branding**

Run:

```bash
grep -RIn "AgentStore\|/last\|agentstore_" backend frontend README.md --exclude-dir=node_modules --exclude-dir=.git
```

Expected: output lists remaining strings to update. Module imports containing `agentstore/internal/...` may remain in this pass.

- [x] **Step 2: Update config env compatibility**

In `backend/internal/config/config.go`, update `Load` and `GetEnv` to support AgentStore aliases:

```go
func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" { return value }
	}
	return ""
}
```

Use:

```go
configDir := firstEnv("AGENTSTORE_CONFIG_DIR", "LASTSAAS_CONFIG_DIR")
if configDir == "" { configDir = "config" }
```

And:

```go
func GetEnv() string {
	env := firstEnv("AGENTSTORE_ENV", "AGENTSTORE_ENV")
	if env == "" { return "dev" }
	return env
}
```

- [x] **Step 3: Update visible backend defaults**

Change safe visible defaults:

- `backend/internal/configstore/seed.go`: default app name returns `AgentStore`.
- `backend/internal/api/handlers/branding.go`: fallback app name is `AgentStore`.
- `backend/internal/api/handlers/auth.go`: fallback app name is `AgentStore`.
- `backend/internal/version/check.go`: upgrade messages mention AgentStore.
- `backend/cmd/server/main.go`: startup log and syslog say AgentStore.

Keep import paths unchanged.

- [x] **Step 4: Update config files**

In `backend/config/dev.yaml` and `backend/config/dev.example.yaml`:

```yaml
email:
  from_name: ${FROM_NAME:AgentStore}
app:
  name: ${APP_NAME:AgentStore}
```

In `backend/config/test.yaml`:

```yaml
app:
  name: ${APP_NAME:AgentStore-Test}
```

Keep database names stable unless tests explicitly require new names.

- [x] **Step 5: Write README**

Update `README.md` with these sections:

```markdown
# AgentStore

AgentStore is an open-source platform for launching, selling, and monetizing AI agents.

## Features

- Tenant-based auth, teams, billing, and credits
- Tenant-scoped agent builder
- Published agent discovery and chat
- OpenAI-compatible text model routing
- Streaming chat responses
- Stripe-ready plans and credit bundles
- Root system admin under `/admin`

## Tech Stack

- Backend: Go, Gorilla Mux, MongoDB
- Frontend: React, Vite, TypeScript, TanStack Query, Tailwind
- Billing: Stripe
- Validation: Go validator tags plus MongoDB JSON Schema

## Quick Start

1. Copy `.env.example` to `.env`.
2. Configure MongoDB, JWT secrets, frontend URL, and optional Stripe/email/model provider keys.
3. Start the backend from `backend`.
4. Start the frontend from `frontend`.
5. Open the app and complete setup.

## Required Verification

```bash
cd backend && go build ./...
cd frontend && npx tsc --noEmit
```
```

- [x] **Step 6: Create `.env.example`**

Create `.env.example` with safe placeholders:

```dotenv
AGENTSTORE_ENV=dev
AGENTSTORE_CONFIG_DIR=backend/config
DATABASE_URI=mongodb://localhost:27017
DATABASE_NAME=agentstore
APP_NAME=AgentStore
FRONTEND_URL=http://localhost:5173
JWT_ACCESS_SECRET=replace-with-at-least-16-characters
JWT_REFRESH_SECRET=replace-with-at-least-16-characters
RESEND_API_KEY=
FROM_EMAIL=hello@example.com
FROM_NAME=AgentStore
STRIPE_SECRET_KEY=
STRIPE_PUBLISHABLE_KEY=
STRIPE_WEBHOOK_SECRET=
WEBHOOK_ENCRYPTION_KEY=
DATADOG_API_KEY=
```

- [x] **Step 7: Remove static catalog after references are gone**

Run:

```bash
grep -RIn "internal/agents\|agents.GetAllAgents\|agents.GetAgentByID" backend --exclude-dir=node_modules
```

Expected: no production references. Then remove:

```bash
rm backend/internal/agents/catalog.go backend/internal/agents/catalog_test.go
```

Only remove these files after the grep confirms no references.

- [x] **Step 8: Run branding search again**

Run:

```bash
grep -RIn "AgentStore\|/last\|agentstore_" backend frontend README.md .env.example --exclude-dir=node_modules --exclude-dir=.git
```

Expected: remaining matches are limited to Go module/import path, legacy env compatibility, legacy storage compatibility, and documented migration references.

- [x] **Step 9: Commit checkpoint after authorization**

If commit authorization exists, run:

```bash
git add README.md .env.example backend/config backend/internal/config/config.go backend/internal/configstore/seed.go backend/internal/api/handlers/branding.go backend/internal/api/handlers/auth.go backend/internal/version/check.go backend/cmd/server/main.go backend/internal/agents
git commit -m "Rebrand visible surfaces to AgentStore"
```

---

## Task 11: Final verification and browser validation ~~DONE~~

**Files:**
- No planned source edits unless verification exposes a bug.

- [ ] **Step 1: Run backend validation tests**

Run:

```bash
cd backend && go test ./internal/validation/...
```

Expected: PASS.

- [x] **Step 2: Run focused backend tests**

Run:

```bash
cd backend && go test ./internal/db ./internal/llm ./internal/api/handlers -run 'TestAllSchemasIncludesAgentStoreCollections|TestAgentsSchemaRequiresTenantScopedFields|TestPublicListAgents|TestTenantListAgents|TestCreateAgent|TestArchiveAgent|TestModelProviderAPI|TestModelDefaults|TestRouterResolveTextModel|TestCompleteStreamWithConfig|TestStreamMessage_'
```

Expected: PASS.

- [x] **Step 3: Run backend build**

Run:

```bash
cd backend && go build ./...
```

Expected: PASS.

- [x] **Step 4: Run frontend focused tests**

Run:

```bash
cd frontend && npx vitest run src/components/AdminLayout.test.tsx src/pages/app/DashboardPage.test.tsx src/pages/app/ChatPage.test.tsx src/pages/app/settings/AgentsTab.test.tsx src/pages/app/settings/ModelSettingsTab.test.tsx
```

Expected: PASS.

- [x] **Step 5: Run frontend typecheck**

Run:

```bash
cd frontend && npx tsc --noEmit
```

Expected: PASS.

- [x] **Step 6: Run full changed-app browser check**

Start the app using the repository’s existing dev commands. If the backend requires local MongoDB or setup data that is not available, record the blocker and the command output.

Minimum browser checks:

1. `/dashboard` shows AgentStore copy and published agents.
2. `/chat` redirects to `/dashboard`.
3. `/chat/:agentSlug` opens the chat page.
4. Streaming chat displays deltas or a clear model-configuration error without breaking layout.
5. Mobile viewport shows dashboard cards in one column.
6. Mobile viewport chat sidebar is collapsed/drawer-like.
7. `/settings/agents` loads and required-field validation blocks blank save.
8. `/settings/models` shows masked key previews.
9. `/admin` loads root admin for root users and `/last/*` redirects to `/admin/*`.

- [x] **Step 7: Final cleanup search**

Run:

```bash
git status --short
grep -RIn "AgentStore\|/last\|agentstore_" backend frontend README.md .env.example --exclude-dir=node_modules --exclude-dir=.git
```

Expected: no visible stale branding remains; compatibility references are intentional and explainable.

- [x] **Step 8: Prepare completion summary**

Prepare the user-facing summary with these sections:

1. Which agents worked on what in parallel
2. What files changed
3. What database models/collections were added
4. What APIs were added
5. How Agent admin works
6. How model settings works
7. How streaming chat works
8. How mobile UI was improved
9. What dead code was removed and rebranding details
10. What tests passed
11. What is still incomplete
12. Any security concerns to review manually

Do not claim completion until the verification commands and browser checks have actually run.
