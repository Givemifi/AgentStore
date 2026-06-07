# AgentStore P0 Productization Closeout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the highest-impact P0 productization gaps for ordinary-user launch readiness: mobile chat recovery, honest provider execution, real admin launch readiness, and matching README positioning.

**Architecture:** Implement the closeout as focused changes around existing boundaries. Backend provider support is enforced in `internal/llm/router.go` and surfaced by chat handlers. Launch readiness is a new admin read endpoint that computes deterministic checklist items from existing collections and health state. Frontend chat keeps the current desktop layout while adding a mobile drawer and assistant-message recovery UI.

**Tech Stack:** Go 1.25, Gorilla Mux, MongoDB Go driver, React 19, TypeScript, TanStack Query, Vitest, Testing Library, Tailwind utility classes.

---

## Scope and Commit Rules

This plan implements the approved fast P0 closeout spec:

`docs/superpowers/specs/2026-06-07-agentstore-p0-productization-closeout-design.md`

Do not implement marketplace expansion, Agent detail pages, true Anthropic/Gemini execution, image/video execution, broad component refactors, or full Playwright hardening in this plan.

The repository instructions say not to commit unless the user explicitly asks. The commit steps below are checkpoint commands prepared for an authorized execution session. If the user has not explicitly authorized commits in that session, skip each commit step and report the files changed instead.

## File Structure Map

### Backend provider guard

- Modify `backend/internal/llm/router.go`
  - Add a sentinel unsupported-provider error.
  - Reject non-OpenAI-compatible providers inside model resolution.
  - Preserve legacy LLM fallback only when no explicit model config is resolved.
- Modify `backend/internal/llm/router_test.go`
  - Add integration tests for unsupported provider rejection and no fallback on explicit unsupported bindings.

### Backend chat error surfacing

- Modify `backend/internal/api/handlers/chat.go`
  - Add a small request-config resolver used by both non-streaming and streaming chat.
  - Return a friendly unsupported-provider message before creating conversations/messages or calling the LLM.
- Modify `backend/internal/api/handlers/chat_test.go`
  - Add chat tests proving unsupported providers do not create chat records and do not deduct credits.

### Backend launch readiness

- Create `backend/internal/api/handlers/admin_launch_readiness.go`
  - Own the launch readiness response types and readiness computation.
  - Keep this logic out of the already-large `admin.go` file.
- Modify `backend/cmd/server/main.go`
  - Wire `GET /api/admin/launch-readiness` into read-only admin routes.
- Modify `backend/internal/api/handlers/testhelpers_test.go`
  - Wire the test server route so integration tests can call it.
- Modify `backend/internal/api/handlers/admin_test.go`
  - Add tests for empty/unconfigured and ready-enough checklist states.

### Frontend launch readiness

- Modify `frontend/src/types/index.ts`
  - Add `LaunchReadinessStatus`, `LaunchReadinessItem`, `LaunchReadinessSummary`, and `LaunchReadinessResponse`.
- Modify `frontend/src/api/client.ts`
  - Add `adminApi.getLaunchReadiness()`.
- Modify `frontend/src/pages/admin/components/LaunchChecklist.tsx`
  - Use API item field `actionPath` instead of locally invented `to`.
  - Keep rendering responsibility isolated to the component.
- Modify `frontend/src/pages/admin/components/LaunchChecklist.test.tsx`
  - Update fixture items and href assertions.
- Modify `frontend/src/pages/admin/DashboardPage.tsx`
  - Fetch readiness from the backend and remove localStorage status logic.
  - Show a visible warning if readiness fails.
- Modify `frontend/src/pages/admin/DashboardPage.test.tsx`
  - Mock `getLaunchReadiness` and assert API-driven checklist behavior.

### Frontend chat closeout

- Modify `frontend/src/pages/app/ChatPage.tsx`
  - Extract conversation sidebar content into a local helper component.
  - Add mobile drawer state and controls.
  - Render assistant error/interrupted badges and retry actions.
  - Store interrupted temporary messages with `status: 'interrupted'` and no textual suffix.
- Modify `frontend/src/pages/app/ChatPage.test.tsx`
  - Update interruption expectations.
  - Add drawer and retry tests.

### Documentation

- Modify `README.md`
  - Replace the opening boilerplate-first positioning with marketplace + SaaS foundation positioning.

---

## Task 1: Add Backend Provider Execution Guard in the LLM Router

**Files:**
- Modify: `backend/internal/llm/router.go`
- Test: `backend/internal/llm/router_test.go`

- [ ] **Step 1: Add failing router tests for unsupported providers**

Append these tests to `backend/internal/llm/router_test.go`:

```go
func TestRouterResolveTextModel_RejectsUnsupportedAgentProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if testDB == nil {
		t.Skip("skipping: no test database connection")
	}
	cleanupCollections(t)

	ctx := context.Background()
	tenantID := primitive.NewObjectID()
	providerID := primitive.NewObjectID()
	modelID := primitive.NewObjectID()

	_, err := testDB.ModelProviders().InsertOne(ctx, models.ModelProvider{
		ID:           providerID,
		TenantID:     tenantID,
		Name:         "Anthropic Provider",
		ProviderType: models.ProviderTypeAnthropic,
		BaseURL:      "https://api.anthropic.com",
		APIKey:       "sk-ant-test-key",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = testDB.ModelConfigs().InsertOne(ctx, models.ModelConfig{
		ID:          modelID,
		TenantID:    tenantID,
		ProviderID:  providerID,
		Name:        "Claude Text",
		DisplayName: "Claude Text",
		Modality:    models.ModelModalityText,
		ModelID:     "claude-test-model",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = testDB.LLMConfigs().InsertOne(ctx, models.LLMConfig{
		Key:       models.DefaultLLMConfigKey,
		APIKey:    "legacy-key",
		BaseURL:   "https://legacy.example.com/v1",
		Model:     "legacy-model",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	tenant := models.Tenant{ID: tenantID}
	agent := models.Agent{ModelConfig: models.AgentModelConfig{TextModelID: &modelID}}

	_, err = NewRouter(testDB).ResolveTextModel(ctx, tenant, agent)
	if !errors.Is(err, ErrProviderTypeUnsupported) {
		t.Fatalf("expected ErrProviderTypeUnsupported, got %v", err)
	}
}

func TestRouterResolveTextModel_RejectsUnsupportedTenantDefaultProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if testDB == nil {
		t.Skip("skipping: no test database connection")
	}
	cleanupCollections(t)

	ctx := context.Background()
	tenantID := primitive.NewObjectID()
	providerID := primitive.NewObjectID()
	defaultModelID := primitive.NewObjectID()

	_, err := testDB.ModelProviders().InsertOne(ctx, models.ModelProvider{
		ID:           providerID,
		TenantID:     tenantID,
		Name:         "Gemini Provider",
		ProviderType: models.ProviderTypeGemini,
		BaseURL:      "https://generativelanguage.googleapis.com",
		APIKey:       "gemini-test-key",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = testDB.ModelConfigs().InsertOne(ctx, models.ModelConfig{
		ID:          defaultModelID,
		TenantID:    tenantID,
		ProviderID:  providerID,
		Name:        "Gemini Text",
		DisplayName: "Gemini Text",
		Modality:    models.ModelModalityText,
		ModelID:     "gemini-test-model",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	tenant := models.Tenant{ID: tenantID, DefaultTextModelConfigID: &defaultModelID}
	agent := models.Agent{ModelConfig: models.AgentModelConfig{}}

	_, err = NewRouter(testDB).ResolveTextModel(ctx, tenant, agent)
	if !errors.Is(err, ErrProviderTypeUnsupported) {
		t.Fatalf("expected ErrProviderTypeUnsupported, got %v", err)
	}
}
```

Also update the import block in `router_test.go` to include `errors`:

```go
import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
	// existing imports remain
)
```

- [ ] **Step 2: Run router tests and verify the new tests fail**

Run:

```bash
cd backend && go test ./internal/llm/... -run 'TestRouterResolveTextModel_(RejectsUnsupportedAgentProvider|RejectsUnsupportedTenantDefaultProvider)' -count=1
```

Expected before implementation: FAIL because `ErrProviderTypeUnsupported` is undefined or unsupported providers resolve as normal OpenAI-compatible configs.

- [ ] **Step 3: Add the unsupported-provider error and guard**

In `backend/internal/llm/router.go`, add this sentinel error near `ErrModelNotConfigured`:

```go
// ErrProviderTypeUnsupported is returned when a configured provider type is
// present in data but is not implemented by the OpenAI-compatible execution path.
var ErrProviderTypeUnsupported = errors.New("model provider type is not supported")
```

Then update the end of `resolveModelConfig` after loading the provider and before returning `RequestConfig`:

```go
	if provider.ProviderType != models.ProviderTypeOpenAICompatible {
		return RequestConfig{}, fmt.Errorf("%w: %s", ErrProviderTypeUnsupported, provider.ProviderType)
	}

	return RequestConfig{
		APIKey:  provider.APIKey,
		BaseURL: normalizeBaseURL(provider.BaseURL),
		Model:   model.ModelID,
	}, nil
```

The final `resolveModelConfig` return block should contain the provider type check before constructing the request config.

- [ ] **Step 4: Run router tests and verify they pass**

Run:

```bash
cd backend && go test ./internal/llm/... -run 'TestRouterResolveTextModel' -count=1
```

Expected: PASS, or SKIP only if `MONGODB_URI` is not configured for integration tests.

- [ ] **Step 5: Checkpoint commit if commits are authorized**

If and only if the user authorized commits in this execution session, run:

```bash
git add backend/internal/llm/router.go backend/internal/llm/router_test.go
git commit -m "fix: reject unsupported chat model providers

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Surface Unsupported Provider Errors in Chat Without Charging Credits

**Files:**
- Modify: `backend/internal/api/handlers/chat.go`
- Test: `backend/internal/api/handlers/chat_test.go`

- [ ] **Step 1: Add failing chat handler test for unsupported provider**

Append this test to `backend/internal/api/handlers/chat_test.go`:

```go
func TestStreamMessage_UnsupportedProviderDoesNotCreateMessagesOrChargeCredits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	ctx := context.Background()

	_, err := env.DB.Tenants().UpdateOne(ctx,
		bson.M{"_id": tenant.ID},
		bson.M{"$set": bson.M{"subscriptionCredits": int64(50), "purchasedCredits": int64(0)}},
	)
	if err != nil {
		t.Fatalf("seed tenant credits: %v", err)
	}

	providerID := primitive.NewObjectID()
	modelID := primitive.NewObjectID()
	now := time.Now()
	_, err = env.DB.ModelProviders().InsertOne(ctx, models.ModelProvider{
		ID:           providerID,
		TenantID:     tenant.ID,
		Name:         "Anthropic Provider",
		ProviderType: models.ProviderTypeAnthropic,
		BaseURL:      "https://api.anthropic.com",
		APIKey:       "sk-ant-test-key",
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	_, err = env.DB.ModelConfigs().InsertOne(ctx, models.ModelConfig{
		ID:          modelID,
		TenantID:    tenant.ID,
		ProviderID:  providerID,
		Name:        "Claude Text",
		DisplayName: "Claude Text",
		Modality:    models.ModelModalityText,
		ModelID:     "claude-test-model",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed model config: %v", err)
	}

	agent := models.Agent{
		ID:           primitive.NewObjectID(),
		TenantID:     tenant.ID,
		Name:         "Unsupported Provider Agent",
		Slug:         "unsupported-provider-agent",
		Category:     "Support",
		Description:  "Uses a provider that is not executable yet.",
		Status:       models.AgentStatusPublished,
		Visibility:   models.AgentVisibilityPublic,
		SystemPrompt: "Help users.",
		Capabilities: []models.AgentCapability{models.AgentCapabilityTextChat},
		CreditCost:   models.AgentCreditCost{TextMessageCredits: 3},
		ModelConfig:  models.AgentModelConfig{TextModelID: &modelID},
		CreatedBy:    user.ID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if _, err := env.DB.Agents().InsertOne(ctx, agent); err != nil {
		t.Fatalf("seed agent: %v", err)
	}

	body, err := json.Marshal(SendMessageRequest{
		AgentID: agent.Slug,
		Message: "Will this charge credits?",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := env.tenantRequest(t, http.MethodPost, "/api/chat/stream", strings.NewReader(string(body)), user, tenant.ID.Hex())
	rr := httptest.NewRecorder()
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB))

	handler.StreamMessage(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusInternalServerError, rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "This model provider is not supported for chat yet") {
		t.Fatalf("expected friendly unsupported provider message, got %s", rr.Body.String())
	}

	messages, err := env.DB.ChatMessages().CountDocuments(ctx, bson.M{"tenantId": tenant.ID, "userId": user.ID})
	if err != nil {
		t.Fatalf("count chat messages: %v", err)
	}
	if messages != 0 {
		t.Fatalf("expected no chat messages to be created, got %d", messages)
	}

	var updatedTenant models.Tenant
	if err := env.DB.Tenants().FindOne(ctx, bson.M{"_id": tenant.ID}).Decode(&updatedTenant); err != nil {
		t.Fatalf("load tenant: %v", err)
	}
	if updatedTenant.SubscriptionCredits != 50 || updatedTenant.PurchasedCredits != 0 {
		t.Fatalf("expected credits unchanged, got subscription=%d purchased=%d", updatedTenant.SubscriptionCredits, updatedTenant.PurchasedCredits)
	}
}
```

- [ ] **Step 2: Run the new chat test and verify it fails**

Run:

```bash
cd backend && go test ./internal/api/handlers/... -run TestStreamMessage_UnsupportedProviderDoesNotCreateMessagesOrChargeCredits -count=1
```

Expected before implementation: FAIL because chat falls back to legacy config or returns the generic model-not-configured message.

- [ ] **Step 3: Add a shared friendly error message and request config resolver**

In `backend/internal/api/handlers/chat.go`, add this constant near the existing chat constants or helper functions:

```go
const unsupportedChatProviderMessage = "This model provider is not supported for chat yet. Use an OpenAI-compatible provider."
```

Add this helper function below `mapCreditDeductionErrorStatus`:

```go
func (h *ChatHandler) resolveChatRequestConfig(ctx context.Context, tenant models.Tenant, resolvedAgent resolvedChatAgent) (llm.RequestConfig, error) {
	if h.modelRouter != nil && resolvedAgent.DBAgent != nil && resolvedAgent.DBAgent.TenantID == tenant.ID {
		cfg, err := h.modelRouter.ResolveTextModel(ctx, tenant, *resolvedAgent.DBAgent)
		if err == nil {
			return cfg, nil
		}
		if errors.Is(err, llm.ErrProviderTypeUnsupported) {
			return llm.RequestConfig{}, err
		}
		if !errors.Is(err, llm.ErrModelNotConfigured) {
			return llm.RequestConfig{}, err
		}
	}

	h.llmClient.ReloadConfig()
	requestConfig := h.llmClient.SnapshotConfig()
	if !llmConfigUsable(&models.LLMConfig{
		APIKey:   requestConfig.APIKey,
		BaseURL:  requestConfig.BaseURL,
		Model:    requestConfig.Model,
		IsActive: true,
	}) {
		return llm.RequestConfig{}, llm.ErrModelNotConfigured
	}
	return requestConfig, nil
}
```

`chat.go` already imports `errors`, `llm`, and `models`, so this helper should not require new imports.

- [ ] **Step 4: Use the helper in non-streaming `SendMessage`**

Replace the current legacy-only config block in `SendMessage` that starts with:

```go
	config := GetLLMConfig(ctx, h.db)
	if !llmConfigUsable(config) {
		respondWithError(w, http.StatusInternalServerError, "AI model is not configured. Please contact an administrator.")
		return
	}

	h.llmClient.ReloadConfig()
	requestConfig := h.llmClient.SnapshotConfig()
	if !llmConfigUsable(&models.LLMConfig{
		APIKey:   requestConfig.APIKey,
		BaseURL:  requestConfig.BaseURL,
		Model:    requestConfig.Model,
		IsActive: true,
	}) {
		respondWithError(w, http.StatusInternalServerError, "AI model is not configured. Please contact an administrator.")
		return
	}
```

with:

```go
	requestConfig, err := h.resolveChatRequestConfig(ctx, *tenant, *resolvedAgent)
	if err != nil {
		if errors.Is(err, llm.ErrProviderTypeUnsupported) {
			respondWithError(w, http.StatusInternalServerError, unsupportedChatProviderMessage)
			return
		}
		if errors.Is(err, llm.ErrModelNotConfigured) {
			respondWithError(w, http.StatusInternalServerError, "AI model is not configured. Please contact an administrator.")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to resolve AI model configuration")
		return
	}
```

- [ ] **Step 5: Use the helper in streaming `StreamMessage`**

Replace the block in `StreamMessage` that manually calls `h.modelRouter.ResolveTextModel`, falls back on `requestConfig.APIKey == ""`, and checks `llmConfigUsable` with:

```go
	requestConfig, err := h.resolveChatRequestConfig(ctx, *tenant, *resolvedAgent)
	if err != nil {
		if errors.Is(err, llm.ErrProviderTypeUnsupported) {
			respondWithError(w, http.StatusInternalServerError, unsupportedChatProviderMessage)
			return
		}
		if errors.Is(err, llm.ErrModelNotConfigured) {
			respondWithError(w, http.StatusInternalServerError, "AI model is not configured. Please contact an administrator.")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to resolve AI model configuration")
		return
	}
```

This replacement must happen before conversation creation and before user/assistant messages are inserted.

- [ ] **Step 6: Run the chat tests and verify they pass**

Run:

```bash
cd backend && go test ./internal/api/handlers/... -run 'Test(StreamMessage_UnsupportedProviderDoesNotCreateMessagesOrChargeCredits|StreamMessage_PersistsGeneratingPlaceholderThenCompletes|SendMessage_ConfigFailureDoesNotCreateConversationOrMessages)' -count=1
```

Expected: PASS, or SKIP only if `MONGODB_URI` is not configured for integration tests.

- [ ] **Step 7: Checkpoint commit if commits are authorized**

If and only if commits are authorized, run:

```bash
git add backend/internal/api/handlers/chat.go backend/internal/api/handlers/chat_test.go
git commit -m "fix: surface unsupported provider chat errors

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Add Backend Admin Launch Readiness API

**Files:**
- Create: `backend/internal/api/handlers/admin_launch_readiness.go`
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/api/handlers/testhelpers_test.go`
- Test: `backend/internal/api/handlers/admin_test.go`

- [ ] **Step 1: Write failing admin readiness integration tests**

Append these tests to `backend/internal/api/handlers/admin_test.go`:

```go
func TestIntegration_AdminLaunchReadiness_UnconfiguredState(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	req := env.adminRequest(t, "GET", "/api/admin/launch-readiness", nil, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}

	var got LaunchReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Summary.Total != 7 {
		t.Fatalf("expected 7 readiness items, got %d", got.Summary.Total)
	}
	assertReadinessStatus(t, got, "model", LaunchReadinessWarning)
	assertReadinessStatus(t, got, "agent", LaunchReadinessPending)
	assertReadinessStatus(t, got, "test-chat", LaunchReadinessPending)
}

func TestIntegration_AdminLaunchReadiness_ReadySignals(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)
	ctx := context.Background()
	now := time.Now()

	_, err := env.DB.BrandingConfig().InsertOne(ctx, models.BrandingConfig{
		ID:            primitive.NewObjectID(),
		AppName:       "AgentStore Launch",
		LogoMode:      "text",
		DashboardHTML: "<p>Launch dashboard copy</p>",
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		t.Fatalf("seed branding: %v", err)
	}

	providerID := primitive.NewObjectID()
	modelID := primitive.NewObjectID()
	_, err = env.DB.ModelProviders().InsertOne(ctx, models.ModelProvider{
		ID:           providerID,
		TenantID:     tenant.ID,
		Name:         "OpenAI Compatible",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk-test-key",
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	_, err = env.DB.ModelConfigs().InsertOne(ctx, models.ModelConfig{
		ID:          modelID,
		TenantID:    tenant.ID,
		ProviderID:  providerID,
		Name:        "Launch Text Model",
		DisplayName: "Launch Text Model",
		Modality:    models.ModelModalityText,
		ModelID:     "launch-model",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed model: %v", err)
	}

	_, err = env.DB.Agents().InsertOne(ctx, models.Agent{
		ID:           primitive.NewObjectID(),
		TenantID:     tenant.ID,
		Name:         "Launch Agent",
		Slug:         "launch-agent",
		Category:     "Support",
		Description:  "Launch-ready support agent.",
		Status:       models.AgentStatusPublished,
		Visibility:   models.AgentVisibilityPublic,
		SystemPrompt: "Help users launch.",
		Capabilities: []models.AgentCapability{models.AgentCapabilityTextChat},
		CreditCost:   models.AgentCreditCost{TextMessageCredits: 1},
		CreatedBy:    admin.ID,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("seed agent: %v", err)
	}

	_, err = env.DB.CreditBundles().InsertOne(ctx, models.CreditBundle{
		ID:         primitive.NewObjectID(),
		Name:       "Launch Credits",
		Credits:    100,
		PriceCents: 1000,
		IsActive:   true,
		SortOrder:  1,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err != nil {
		t.Fatalf("seed bundle: %v", err)
	}

	conversationID := primitive.NewObjectID()
	_, err = env.DB.ChatMessages().InsertOne(ctx, models.ChatMessage{
		ID:             primitive.NewObjectID(),
		TenantID:       tenant.ID,
		UserID:         admin.ID,
		ConversationID: conversationID,
		AgentID:        "launch-agent",
		Role:           "assistant",
		Content:        "Launch smoke test passed.",
		Status:         models.ChatMessageStatusCompleted,
		CreditsCharged: 1,
		Model:          "launch-model",
		CreatedAt:      now,
	})
	if err != nil {
		t.Fatalf("seed chat message: %v", err)
	}

	req := env.adminRequest(t, "GET", "/api/admin/launch-readiness", nil, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}

	var got LaunchReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assertReadinessStatus(t, got, "brand", LaunchReadinessComplete)
	assertReadinessStatus(t, got, "model", LaunchReadinessComplete)
	assertReadinessStatus(t, got, "agent", LaunchReadinessComplete)
	assertReadinessStatus(t, got, "credits", LaunchReadinessComplete)
	assertReadinessStatus(t, got, "test-chat", LaunchReadinessComplete)
}

func assertReadinessStatus(t *testing.T, response LaunchReadinessResponse, id string, want LaunchReadinessStatus) {
	t.Helper()
	for _, item := range response.Items {
		if item.ID == id {
			if item.Status != want {
				t.Fatalf("expected readiness item %s status %s, got %s", id, want, item.Status)
			}
			return
		}
	}
	t.Fatalf("readiness item %s not found in %#v", id, response.Items)
}
```

These tests compile only after the response types and route are added.

- [ ] **Step 2: Run the readiness tests and verify they fail**

Run:

```bash
cd backend && go test ./internal/api/handlers/... -run 'TestIntegration_AdminLaunchReadiness' -count=1
```

Expected before implementation: FAIL because `LaunchReadinessResponse` and the route do not exist.

- [ ] **Step 3: Create the readiness handler file**

Create `backend/internal/api/handlers/admin_launch_readiness.go` with this content:

```go
package handlers

import (
	"net/http"
	"strings"

	"agentstore/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type LaunchReadinessStatus string

const (
	LaunchReadinessComplete LaunchReadinessStatus = "complete"
	LaunchReadinessWarning  LaunchReadinessStatus = "warning"
	LaunchReadinessPending  LaunchReadinessStatus = "pending"
)

type LaunchReadinessItem struct {
	ID          string                `json:"id"`
	Label       string                `json:"label"`
	Status      LaunchReadinessStatus `json:"status"`
	Description string                `json:"description"`
	ActionPath  string                `json:"actionPath"`
}

type LaunchReadinessSummary struct {
	Complete int `json:"complete"`
	Warning  int `json:"warning"`
	Pending  int `json:"pending"`
	Total    int `json:"total"`
}

type LaunchReadinessResponse struct {
	Items   []LaunchReadinessItem  `json:"items"`
	Summary LaunchReadinessSummary `json:"summary"`
}

func (h *AdminHandler) GetLaunchReadiness(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middlewareTenantFromRequest(r)
	if !ok {
		respondWithError(w, http.StatusBadRequest, "Tenant context required")
		return
	}

	items := []LaunchReadinessItem{
		h.brandReadiness(r),
		h.modelReadiness(r),
		h.agentReadiness(r, tenant),
		h.creditReadiness(r),
		h.integrationReadiness("stripe", "Stripe webhook healthy", "Stripe integration is configured according to the health check.", "Configure Stripe keys and webhook handling before selling credits.", "/admin/health#integrations"),
		h.integrationReadiness("resend", "Email provider ready", "Email integration is configured according to the health check.", "Configure Resend before relying on invites, verification, and password resets.", "/admin/health#integrations"),
		h.testChatReadiness(r, tenant),
	}

	respondWithJSON(w, http.StatusOK, LaunchReadinessResponse{Items: items, Summary: summarizeLaunchReadiness(items)})
}

func middlewareTenantFromRequest(r *http.Request) (*models.Tenant, bool) {
	return middleware.GetTenantFromContext(r.Context())
}

func summarizeLaunchReadiness(items []LaunchReadinessItem) LaunchReadinessSummary {
	summary := LaunchReadinessSummary{Total: len(items)}
	for _, item := range items {
		switch item.Status {
		case LaunchReadinessComplete:
			summary.Complete++
		case LaunchReadinessWarning:
			summary.Warning++
		default:
			summary.Pending++
		}
	}
	return summary
}

func (h *AdminHandler) brandReadiness(r *http.Request) LaunchReadinessItem {
	item := LaunchReadinessItem{ID: "brand", Label: "Brand configured", Status: LaunchReadinessPending, Description: "Review app name, logo, landing copy, dashboard copy, and auth page text before inviting users.", ActionPath: "/admin/branding"}

	var cfg models.BrandingConfig
	err := h.db.BrandingConfig().FindOne(r.Context(), bson.M{}).Decode(&cfg)
	if err == nil && brandingCustomized(cfg) {
		item.Status = LaunchReadinessComplete
		return item
	}
	if err != nil && err != mongo.ErrNoDocuments {
		item.Status = LaunchReadinessWarning
		item.Description = "Branding status could not be checked. Open branding settings and verify the public copy before launch."
		return item
	}

	count, err := h.db.BrandingAssets().CountDocuments(r.Context(), bson.M{"key": bson.M{"$in": []string{"logo", "favicon"}}})
	if err == nil && count > 0 {
		item.Status = LaunchReadinessComplete
	}
	return item
}

func brandingCustomized(cfg models.BrandingConfig) bool {
	if strings.TrimSpace(cfg.AppName) != "" && strings.TrimSpace(cfg.AppName) != "AgentStore" {
		return true
	}
	fields := []string{cfg.Tagline, cfg.LandingTitle, cfg.LandingMeta, cfg.LandingHTML, cfg.DashboardHTML, cfg.LoginHeading, cfg.LoginSubtext, cfg.SignupHeading, cfg.SignupSubtext, cfg.CustomCSS, cfg.HeadHTML, cfg.OgImageURL}
	for _, field := range fields {
		if strings.TrimSpace(field) != "" {
			return true
		}
	}
	return false
}

func (h *AdminHandler) modelReadiness(r *http.Request) LaunchReadinessItem {
	item := LaunchReadinessItem{ID: "model", Label: "Model provider connected", Status: LaunchReadinessWarning, Description: "Connect an OpenAI-compatible provider and set a default text model for chat.", ActionPath: "/settings/models"}
	if h.hasOpenAICompatibleTextModel(r) || h.hasActiveLegacyLLMConfig(r) {
		item.Status = LaunchReadinessComplete
		item.Description = "A chat-capable model configuration is available."
	}
	return item
}

func (h *AdminHandler) hasOpenAICompatibleTextModel(r *http.Request) bool {
	cursor, err := h.db.ModelConfigs().Find(r.Context(), bson.M{"modality": models.ModelModalityText, "enabled": true})
	if err != nil {
		return false
	}
	defer cursor.Close(r.Context())

	for cursor.Next(r.Context()) {
		var model models.ModelConfig
		if err := cursor.Decode(&model); err != nil {
			continue
		}
		count, err := h.db.ModelProviders().CountDocuments(r.Context(), bson.M{"_id": model.ProviderID, "tenantId": model.TenantID, "enabled": true, "providerType": models.ProviderTypeOpenAICompatible})
		if err == nil && count > 0 {
			return true
		}
	}
	return false
}

func (h *AdminHandler) hasActiveLegacyLLMConfig(r *http.Request) bool {
	count, err := h.db.LLMConfigs().CountDocuments(r.Context(), bson.M{"key": models.DefaultLLMConfigKey, "isActive": true, "apiKey": bson.M{"$ne": ""}, "baseURL": bson.M{"$ne": ""}, "model": bson.M{"$ne": ""}})
	return err == nil && count > 0
}

func (h *AdminHandler) agentReadiness(r *http.Request, tenant *models.Tenant) LaunchReadinessItem {
	item := LaunchReadinessItem{ID: "agent", Label: "First Agent published", Status: LaunchReadinessPending, Description: "Create and publish at least one text-chat Agent so users have something to try.", ActionPath: "/settings/agents"}
	count, err := h.db.Agents().CountDocuments(r.Context(), bson.M{"tenantId": tenant.ID, "status": models.AgentStatusPublished, "capabilities": models.AgentCapabilityTextChat})
	if err == nil && count > 0 {
		item.Status = LaunchReadinessComplete
		item.Description = "At least one text-chat Agent is published."
	}
	return item
}

func (h *AdminHandler) creditReadiness(r *http.Request) LaunchReadinessItem {
	item := LaunchReadinessItem{ID: "credits", Label: "Credit bundle or plan active", Status: LaunchReadinessWarning, Description: "Enable at least one plan or credit bundle before launch so users can continue after credits run out.", ActionPath: "/admin/plans"}
	planCount, planErr := h.db.Plans().CountDocuments(r.Context(), bson.M{"isArchived": bson.M{"$ne": true}})
	bundleCount, bundleErr := h.db.CreditBundles().CountDocuments(r.Context(), bson.M{"isActive": true})
	if (planErr == nil && planCount > 0) || (bundleErr == nil && bundleCount > 0) {
		item.Status = LaunchReadinessComplete
		item.Description = "A plan or active credit bundle is available."
	}
	return item
}

func (h *AdminHandler) integrationReadiness(name, label, healthyDescription, unhealthyDescription, actionPath string) LaunchReadinessItem {
	item := LaunchReadinessItem{ID: name, Label: label, Status: LaunchReadinessWarning, Description: unhealthyDescription, ActionPath: actionPath}
	if h.health == nil {
		return item
	}
	for _, result := range h.health.GetIntegrationStatus() {
		if result.Name == name && result.Status == models.IntegrationHealthy {
			item.Status = LaunchReadinessComplete
			item.Description = healthyDescription
			return item
		}
	}
	return item
}

func (h *AdminHandler) testChatReadiness(r *http.Request, tenant *models.Tenant) LaunchReadinessItem {
	item := LaunchReadinessItem{ID: "test-chat", Label: "Test chat passed", Status: LaunchReadinessPending, Description: "Run a real smoke chat with credits before announcing the product.", ActionPath: "/dashboard"}
	count, err := h.db.ChatMessages().CountDocuments(r.Context(), bson.M{"tenantId": tenant.ID, "role": "assistant", "status": models.ChatMessageStatusCompleted, "creditsCharged": bson.M{"$gt": 0}})
	if err == nil && count > 0 {
		item.Status = LaunchReadinessComplete
		item.Description = "A completed chat with credits charged has been recorded."
	}
	return item
}
```

Then add `agentstore/internal/middleware` to the import block because `middleware.GetTenantFromContext` is used:

```go
import (
	"net/http"
	"strings"

	"agentstore/internal/middleware"
	"agentstore/internal/models"
	// existing imports remain
)
```

- [ ] **Step 4: Wire the route in the production server**

In `backend/cmd/server/main.go`, add this read-only admin route near `/admin/dashboard`:

```go
		adminAPI.HandleFunc("/launch-readiness", adminHandler.GetLaunchReadiness).Methods("GET")
```

Place it after:

```go
		adminAPI.HandleFunc("/dashboard", adminHandler.GetDashboard).Methods("GET")
```

- [ ] **Step 5: Wire the route in the test server**

In `backend/internal/api/handlers/testhelpers_test.go`, add this route near the admin dashboard route:

```go
		adminAPI.HandleFunc("/launch-readiness", adminHandler.GetLaunchReadiness).Methods("GET")
```

Place it after:

```go
		adminAPI.HandleFunc("/dashboard", adminHandler.GetDashboard).Methods("GET")
```

- [ ] **Step 6: Run readiness tests and verify they pass**

Run:

```bash
cd backend && go test ./internal/api/handlers/... -run 'TestIntegration_AdminLaunchReadiness' -count=1
```

Expected: PASS, or SKIP only if `MONGODB_URI` is not configured.

- [ ] **Step 7: Checkpoint commit if commits are authorized**

If and only if commits are authorized, run:

```bash
git add backend/internal/api/handlers/admin_launch_readiness.go backend/cmd/server/main.go backend/internal/api/handlers/testhelpers_test.go backend/internal/api/handlers/admin_test.go
git commit -m "feat: add admin launch readiness endpoint

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Connect Admin Launch Checklist to the Readiness API

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/client.ts`
- Modify: `frontend/src/pages/admin/components/LaunchChecklist.tsx`
- Test: `frontend/src/pages/admin/components/LaunchChecklist.test.tsx`
- Modify: `frontend/src/pages/admin/DashboardPage.tsx`
- Test: `frontend/src/pages/admin/DashboardPage.test.tsx`

- [ ] **Step 1: Add frontend readiness types**

In `frontend/src/types/index.ts`, add these types near the admin/system types:

```ts
export type LaunchReadinessStatus = 'complete' | 'warning' | 'pending';

export interface LaunchReadinessItem {
  id: string;
  label: string;
  status: LaunchReadinessStatus;
  description: string;
  actionPath: string;
}

export interface LaunchReadinessSummary {
  complete: number;
  warning: number;
  pending: number;
  total: number;
}

export interface LaunchReadinessResponse {
  items: LaunchReadinessItem[];
  summary: LaunchReadinessSummary;
}
```

- [ ] **Step 2: Add the API client method**

In `frontend/src/api/client.ts`, update the type import line to include `LaunchReadinessResponse`:

```ts
import type { /* existing types */, LaunchReadinessResponse } from '../types';
```

Because the import line is already long, preserve the existing imported names and add `LaunchReadinessResponse` before `ModelProvider`.

Then add this method to `adminApi` after `getDashboard`:

```ts
  getLaunchReadiness: () =>
    api.get<LaunchReadinessResponse>('/admin/launch-readiness').then(r => r.data),
```

- [ ] **Step 3: Update LaunchChecklist to use API field names**

In `frontend/src/pages/admin/components/LaunchChecklist.tsx`, replace the local status/item type definitions with imports from types:

```ts
import type { LaunchReadinessItem, LaunchReadinessStatus } from '../../../types';
```

Keep exporting the item type for existing test imports:

```ts
export type { LaunchReadinessItem };
```

Update the link target from `item.to` to `item.actionPath`:

```tsx
<Link key={item.id} to={item.actionPath} className="group rounded-2xl border border-dark-800 bg-dark-950/50 p-4 transition-colors hover:border-primary-500/30 hover:bg-dark-900">
```

The `LaunchChecklistProps` remains:

```ts
interface LaunchChecklistProps {
  items: LaunchReadinessItem[];
}
```

- [ ] **Step 4: Update LaunchChecklist tests**

In `frontend/src/pages/admin/components/LaunchChecklist.test.tsx`, replace every fixture property named `to` with `actionPath`.

The fixture should start like this:

```ts
const allSevenItems: LaunchReadinessItem[] = [
  { id: 'brand', label: 'Brand configured', status: 'complete', description: 'Review app name and logo.', actionPath: '/admin/branding' },
  { id: 'model', label: 'Model provider connected', status: 'warning', description: 'Connect a provider.', actionPath: '/settings/models' },
  { id: 'agent', label: 'First Agent published', status: 'pending', description: 'Create and publish at least one Agent.', actionPath: '/settings/agents' },
  { id: 'credits', label: 'Credit bundle or plan active', status: 'complete', description: 'Review plans and credit bundles.', actionPath: '/admin/plans' },
  { id: 'stripe', label: 'Stripe webhook healthy', status: 'complete', description: 'Stripe integration is configured.', actionPath: '/admin/health#integrations' },
  { id: 'resend', label: 'Email provider ready', status: 'warning', description: 'Configure Resend before relying on emails.', actionPath: '/admin/health#integrations' },
  { id: 'test-chat', label: 'Test chat passed', status: 'pending', description: 'Run a real smoke chat.', actionPath: '/dashboard' },
];
```

Update the text assertion for the credits item from `Stripe ready for credit sales` to `Credit bundle or plan active`.

- [ ] **Step 5: Update Admin Dashboard tests to mock readiness API**

In `frontend/src/pages/admin/DashboardPage.test.tsx`, add `getLaunchReadiness` to `apiMocks`:

```ts
const apiMocks = vi.hoisted(() => ({
  getDashboard: vi.fn(),
  getHealthIntegrations: vi.fn(),
  getFinancialMetrics: vi.fn(),
  getLaunchReadiness: vi.fn(),
}));
```

Add it to the `adminApi` mock:

```ts
getLaunchReadiness: apiMocks.getLaunchReadiness,
```

In `beforeEach`, add this default:

```ts
apiMocks.getLaunchReadiness.mockResolvedValue({
  items: [
    { id: 'brand', label: 'Brand configured', status: 'pending', description: 'Review app name and logo.', actionPath: '/admin/branding' },
    { id: 'model', label: 'Model provider connected', status: 'warning', description: 'Connect a provider.', actionPath: '/settings/models' },
    { id: 'agent', label: 'First Agent published', status: 'pending', description: 'Create and publish at least one Agent.', actionPath: '/settings/agents' },
    { id: 'credits', label: 'Credit bundle or plan active', status: 'warning', description: 'Enable a plan or credit bundle.', actionPath: '/admin/plans' },
    { id: 'stripe', label: 'Stripe webhook healthy', status: 'warning', description: 'Configure Stripe webhook handling.', actionPath: '/admin/health#integrations' },
    { id: 'resend', label: 'Email provider ready', status: 'warning', description: 'Configure Resend.', actionPath: '/admin/health#integrations' },
    { id: 'test-chat', label: 'Test chat passed', status: 'pending', description: 'Run a real smoke chat.', actionPath: '/dashboard' },
  ],
  summary: { complete: 0, warning: 4, pending: 3, total: 7 },
});
```

Update dashboard tests that refer to `Stripe ready for credit sales` so they refer to `Credit bundle or plan active`.

Add this new test:

```ts
it('renders launch checklist from readiness API instead of local storage', async () => {
  localStorage.setItem('launch-checklist-brand', 'complete');
  apiMocks.getLaunchReadiness.mockResolvedValue({
    items: [
      { id: 'brand', label: 'Brand configured', status: 'pending', description: 'Backend says brand is still pending.', actionPath: '/admin/branding' },
    ],
    summary: { complete: 0, warning: 0, pending: 1, total: 1 },
  });

  renderDashboard();

  expect(await screen.findByText('Backend says brand is still pending.')).toBeInTheDocument();
  const brandLink = screen.getByRole('link', { name: /Brand configured/i });
  expect(within(brandLink).getByText('Review')).toBeInTheDocument();
});
```

Add this error-state test:

```ts
it('shows a visible warning when launch readiness cannot be loaded', async () => {
  apiMocks.getLaunchReadiness.mockRejectedValue(new Error('readiness unavailable'));

  renderDashboard();

  expect(await screen.findByText('Launch readiness unavailable')).toBeInTheDocument();
  expect(screen.getByText('Refresh the page or check admin API health before using the checklist for launch decisions.')).toBeInTheDocument();
});
```

- [ ] **Step 6: Run dashboard tests and verify they fail**

Run:

```bash
cd frontend && npm test -- --run src/pages/admin/DashboardPage.test.tsx src/pages/admin/components/LaunchChecklist.test.tsx
```

Expected before implementation: FAIL because `getLaunchReadiness` is unused and `LaunchChecklist` still expects `to`.

- [ ] **Step 7: Update Admin Dashboard implementation**

In `frontend/src/pages/admin/DashboardPage.tsx`, remove `getManualItemStatus` entirely.

Add a separate readiness query after the dashboard/charts queries:

```ts
const { data: launchReadiness, isLoading: readinessLoading, error: readinessError } = useQuery({
  queryKey: ['admin', 'launch-readiness'],
  queryFn: adminApi.getLaunchReadiness,
});
```

Delete the inline `launchChecklist` array construction.

Replace:

```tsx
<LaunchChecklist items={launchChecklist} />
```

with:

```tsx
{readinessLoading ? (
  <section className="mb-8 rounded-3xl border border-dark-800 bg-dark-900/60 p-6 text-sm text-dark-400">
    Loading launch readiness…
  </section>
) : readinessError ? (
  <section className="mb-8 rounded-3xl border border-yellow-500/20 bg-yellow-500/10 p-6 text-yellow-100">
    <h2 className="text-xl font-bold text-white">Launch readiness unavailable</h2>
    <p className="mt-2 text-sm text-yellow-100/90">
      Refresh the page or check admin API health before using the checklist for launch decisions.
    </p>
  </section>
) : launchReadiness ? (
  <LaunchChecklist items={launchReadiness.items} />
) : null}
```

Leave the existing integration warning banner in place because it provides a compact list of unconfigured integrations.

- [ ] **Step 8: Run frontend admin tests and verify they pass**

Run:

```bash
cd frontend && npm test -- --run src/pages/admin/DashboardPage.test.tsx src/pages/admin/components/LaunchChecklist.test.tsx
```

Expected: PASS.

- [ ] **Step 9: Checkpoint commit if commits are authorized**

If and only if commits are authorized, run:

```bash
git add frontend/src/types/index.ts frontend/src/api/client.ts frontend/src/pages/admin/components/LaunchChecklist.tsx frontend/src/pages/admin/components/LaunchChecklist.test.tsx frontend/src/pages/admin/DashboardPage.tsx frontend/src/pages/admin/DashboardPage.test.tsx
git commit -m "feat: drive launch checklist from readiness API

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Add Mobile Conversation Drawer to Chat

**Files:**
- Modify: `frontend/src/pages/app/ChatPage.tsx`
- Test: `frontend/src/pages/app/ChatPage.test.tsx`

- [ ] **Step 1: Add failing mobile drawer test**

Append this test to `frontend/src/pages/app/ChatPage.test.tsx`:

```ts
it('opens and closes the mobile conversations drawer', async () => {
  const user = userEvent.setup();
  renderChatPage('/chat/agent-1?conversationId=conv-1');

  expect(await screen.findByText('How should we launch?')).toBeInTheDocument();

  await user.click(screen.getByRole('button', { name: 'Open conversations' }));

  const drawer = screen.getByRole('dialog', { name: 'Conversations' });
  expect(drawer).toBeInTheDocument();
  expect(within(drawer).getByText('Launch strategy')).toBeInTheDocument();

  await user.click(within(drawer).getByRole('button', { name: 'Close conversations' }));

  await waitFor(() => {
    expect(screen.queryByRole('dialog', { name: 'Conversations' })).not.toBeInTheDocument();
  });
});
```

Update the import line at the top of the test file to include `within`:

```ts
import { render, screen, waitFor, within } from '@testing-library/react';
```

- [ ] **Step 2: Run the drawer test and verify it fails**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/ChatPage.test.tsx -t 'opens and closes the mobile conversations drawer'
```

Expected before implementation: FAIL because there is no `Open conversations` button or dialog.

- [ ] **Step 3: Add mobile drawer state and icons**

In `frontend/src/pages/app/ChatPage.tsx`, update the lucide import to include `Menu` and `X`:

```ts
import { AlertCircle, ArrowLeft, Globe2, Headphones, Landmark, Loader2, Menu, MessageSquare, PenLine, Plus, Scale, Send, Square, TowerControl, X, Zap } from 'lucide-react';
```

Add this state near the existing React state variables:

```ts
const [isConversationDrawerOpen, setIsConversationDrawerOpen] = useState(false);
```

Update `startNewChat` and `openConversation` so both close the drawer:

```ts
const startNewChat = () => {
  setIsConversationDrawerOpen(false);
  // existing body remains
};

const openConversation = (nextConversationId: string) => {
  setIsConversationDrawerOpen(false);
  // existing body remains
};
```

- [ ] **Step 4: Extract sidebar content into a local helper component**

Above `export default function ChatPage()`, add this helper component:

```tsx
function ConversationSidebarContent({
  agent,
  conversations,
  conversationsLoading,
  conversationsError,
  conversationId,
  startNewChat,
  openConversation,
}: {
  agent: { name: string; category: string };
  conversations: Conversation[] | undefined;
  conversationsLoading: boolean;
  conversationsError: unknown;
  conversationId: string | null;
  startNewChat: () => void;
  openConversation: (nextConversationId: string) => void;
}) {
  return (
    <>
      <div className="border-b border-dark-800 p-4">
        <button
          onClick={startNewChat}
          className="flex w-full items-center justify-center gap-2 rounded-2xl bg-primary-500 px-4 py-3 text-sm font-medium text-white transition-colors hover:bg-primary-600"
        >
          <Plus className="h-4 w-4" />
          New chat
        </button>
        <div className="mt-4 rounded-2xl border border-dark-800 bg-dark-950/60 p-4">
          <p className="text-xs font-medium uppercase tracking-[0.16em] text-dark-500">Current Agent</p>
          <p className="mt-2 text-sm font-semibold text-white">{agent.name}</p>
          <p className="mt-1 text-sm text-dark-400">{agent.category}</p>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-white">Conversations</h2>
          {conversations && conversations.length > 0 && (
            <span className="text-xs text-dark-500">{conversations.length}</span>
          )}
        </div>

        {conversationsLoading ? (
          <ConversationListSkeleton />
        ) : conversationsError ? (
          <div className="rounded-2xl border border-red-500/20 bg-red-500/10 p-4 text-sm text-red-200">
            {getErrorMessage(conversationsError)}
          </div>
        ) : conversations && conversations.length > 0 ? (
          <div className="space-y-3">
            {conversations.map((conversation) => (
              <ConversationItem
                key={conversation.id}
                conversation={conversation}
                isActive={conversation.id === conversationId}
                onClick={() => openConversation(conversation.id)}
              />
            ))}
          </div>
        ) : (
          <div className="rounded-2xl border border-dashed border-dark-800 bg-dark-950/40 p-5 text-sm text-dark-400">
            No conversations yet. Start a new chat to create your first thread.
          </div>
        )}
      </div>
    </>
  );
}
```

Then replace the current desktop `<aside>` inner content with this component.

- [ ] **Step 5: Render desktop sidebar and mobile drawer separately**

Change the main return container so the desktop sidebar uses `hidden lg:flex`:

```tsx
<aside className="hidden w-full flex-col rounded-3xl border border-dark-800 bg-dark-900/60 lg:flex lg:w-80 lg:min-w-80">
  <ConversationSidebarContent
    agent={agent}
    conversations={conversations}
    conversationsLoading={conversationsLoading}
    conversationsError={conversationsError}
    conversationId={conversationId}
    startNewChat={startNewChat}
    openConversation={openConversation}
  />
</aside>
```

Add this mobile drawer immediately after the desktop aside:

```tsx
{isConversationDrawerOpen && (
  <div className="fixed inset-0 z-50 lg:hidden" role="dialog" aria-modal="true" aria-label="Conversations">
    <button
      type="button"
      className="absolute inset-0 bg-black/60"
      aria-label="Close conversations overlay"
      onClick={() => setIsConversationDrawerOpen(false)}
    />
    <aside className="relative z-10 flex h-full w-[min(22rem,86vw)] flex-col border-r border-dark-800 bg-dark-950 shadow-2xl">
      <div className="flex items-center justify-between border-b border-dark-800 px-4 py-3">
        <h2 className="text-sm font-semibold text-white">Conversations</h2>
        <button
          type="button"
          onClick={() => setIsConversationDrawerOpen(false)}
          className="rounded-xl bg-dark-800 p-2 text-dark-300 transition-colors hover:bg-dark-700 hover:text-white"
          aria-label="Close conversations"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
      <ConversationSidebarContent
        agent={agent}
        conversations={conversations}
        conversationsLoading={conversationsLoading}
        conversationsError={conversationsError}
        conversationId={conversationId}
        startNewChat={startNewChat}
        openConversation={openConversation}
      />
    </aside>
  </div>
)}
```

- [ ] **Step 6: Add the mobile open button in the chat header**

In the chat header near the existing back button, add this button before the back button:

```tsx
<button
  onClick={() => setIsConversationDrawerOpen(true)}
  className="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-dark-800 text-dark-300 transition-colors hover:bg-dark-700 hover:text-white lg:hidden"
  aria-label="Open conversations"
>
  <Menu className="h-5 w-5" />
</button>
```

Keep the existing `Back to Agents` button after it.

- [ ] **Step 7: Run the drawer test and verify it passes**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/ChatPage.test.tsx -t 'opens and closes the mobile conversations drawer'
```

Expected: PASS.

- [ ] **Step 8: Checkpoint commit if commits are authorized**

If and only if commits are authorized, run:

```bash
git add frontend/src/pages/app/ChatPage.tsx frontend/src/pages/app/ChatPage.test.tsx
git commit -m "feat: add mobile chat conversation drawer

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Add Assistant Interrupted/Failed Retry UI

**Files:**
- Modify: `frontend/src/pages/app/ChatPage.tsx`
- Test: `frontend/src/pages/app/ChatPage.test.tsx`

- [ ] **Step 1: Update the stop-stream test to expect badge instead of suffix**

In `frontend/src/pages/app/ChatPage.test.tsx`, update the test named `treats AbortError from stopping a stream as an intentional cancellation without showing an error`.

Replace:

```ts
expect(await screen.findByText('Partial answer [interrupted]')).toBeInTheDocument();
```

with:

```ts
expect(await screen.findByText('Partial answer')).toBeInTheDocument();
expect(screen.getByText('Interrupted · no charge')).toBeInTheDocument();
expect(screen.getByRole('button', { name: 'Retry message' })).toBeInTheDocument();
```

- [ ] **Step 2: Add failed-message retry test**

Append this test to `frontend/src/pages/app/ChatPage.test.tsx`:

```ts
it('retries the previous user message from an interrupted assistant response', async () => {
  const user = userEvent.setup();
  apiMocks.messages.mockResolvedValue([
    mockMessages[0],
    {
      ...mockMessages[1],
      id: 'msg-interrupted',
      content: 'Partial answer',
      creditsCharged: 0,
      status: 'interrupted',
    },
  ]);

  renderChatPage('/chat/agent-1?conversationId=conv-1');

  expect(await screen.findByText('Interrupted · no charge')).toBeInTheDocument();
  await user.click(screen.getByRole('button', { name: 'Retry message' }));

  await waitFor(() => {
    expect(apiMocks.stream).toHaveBeenCalledWith(
      {
        agentId: 'agent-1',
        conversationId: 'conv-1',
        message: 'How should we launch?',
      },
      expect.any(Function),
      expect.any(AbortSignal)
    );
  });
});
```

- [ ] **Step 3: Run retry tests and verify they fail**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/ChatPage.test.tsx -t 'Interrupted|retries the previous user message'
```

Expected before implementation: FAIL because interrupted suffix is still rendered and retry UI does not exist.

- [ ] **Step 4: Add assistant status helpers**

In `frontend/src/pages/app/ChatPage.tsx`, add these helpers above `export default function ChatPage()`:

```tsx
function getAssistantStatus(message: ChatMessage): { label: string; className: string } | null {
  if (message.role !== 'assistant') return null;
  if (message.status === 'interrupted') {
    return {
      label: 'Interrupted · no charge',
      className: 'border-amber-500/30 bg-amber-500/10 text-amber-200',
    };
  }
  if (message.status === 'error') {
    return {
      label: 'Failed · no charge',
      className: 'border-red-500/25 bg-red-500/10 text-red-200',
    };
  }
  return null;
}

function findPreviousUserMessage(messages: ChatMessage[], index: number): ChatMessage | null {
  for (let i = index - 1; i >= 0; i -= 1) {
    if (messages[i].role === 'user') {
      return messages[i];
    }
  }
  return null;
}
```

- [ ] **Step 5: Add retry handler**

Inside `ChatPage`, after `handleExampleClick`, add:

```ts
const handleRetryAssistantMessage = (messageIndex: number) => {
  const previousUserMessage = findPreviousUserMessage(displayedMessages, messageIndex);
  if (!previousUserMessage || isStreaming || sendMessageMutation.isPending || hasInsufficientCredits) {
    if (previousUserMessage) {
      setDraft(previousUserMessage.content);
      focusComposer();
    }
    return;
  }

  setComposerNotice(null);
  void sendStreamMessage(previousUserMessage.content, conversationId, resolvedAgentId);
};
```

- [ ] **Step 6: Store client-side interrupted messages with status instead of suffix**

In the expected stream abort branch, replace:

```ts
content: `${streamingContentRef.current} [interrupted]`,
creditsCharged: 0,
createdAt: new Date().toISOString(),
```

with:

```ts
content: streamingContentRef.current,
status: 'interrupted',
creditsCharged: 0,
createdAt: new Date().toISOString(),
```

- [ ] **Step 7: Render status badge and retry action in assistant bubbles**

In the message render loop, change:

```tsx
{displayedMessages.map((message) => (
```

into:

```tsx
{displayedMessages.map((message, index) => {
  const assistantStatus = getAssistantStatus(message);
  return (
```

Then close the callback with `); })}` instead of `))}`.

Inside the message bubble after the credits line, add:

```tsx
{assistantStatus && (
  <div className="mt-3 flex flex-wrap items-center gap-2 border-t border-white/6 pt-3">
    <span className={`rounded-full border px-2.5 py-1 text-xs font-semibold ${assistantStatus.className}`}>
      {assistantStatus.label}
    </span>
    <button
      type="button"
      onClick={() => handleRetryAssistantMessage(index)}
      className="rounded-lg border border-dark-700 px-2.5 py-1 text-xs font-medium text-dark-200 transition-colors hover:border-primary-400/40 hover:text-white"
      aria-label="Retry message"
    >
      Retry
    </button>
  </div>
)}
```

Only assistant messages with `status` error/interrupted will show this UI.

- [ ] **Step 8: Run chat tests and verify they pass**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/ChatPage.test.tsx
```

Expected: PASS.

- [ ] **Step 9: Checkpoint commit if commits are authorized**

If and only if commits are authorized, run:

```bash
git add frontend/src/pages/app/ChatPage.tsx frontend/src/pages/app/ChatPage.test.tsx
git commit -m "feat: add chat retry for interrupted responses

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Update README Product Positioning

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Replace the opening tagline and first paragraph**

In `README.md`, replace lines 7-11 with:

```md
**A marketplace for practical AI agents, built on an open-source SaaS foundation.**

AgentStore helps operators launch a curated AI Agent marketplace where users can discover useful Agents, chat with them, understand credit usage, buy credits, and continue their work without needing to understand the underlying platform. The product experience is backed by a production-ready SaaS foundation: multi-tenant account management, authentication, role-based access control, white-label branding, Stripe billing, API keys, outgoing webhooks, admin tooling, system health monitoring, credit-based usage tracking, and product analytics with telemetry.

For builders, AgentStore is still a fork-ready SaaS/AI platform foundation built through conversation with [Claude Code](https://claude.ai/claude-code). It lets you start from a working marketplace and keep customizing the product through agentic engineering instead of rebuilding the same SaaS plumbing from scratch.
```

Keep the project badge block and `Project Page` link unchanged.

- [ ] **Step 2: Check README wording manually**

Review the first 25 lines of `README.md` and confirm the first impression is Agent marketplace + foundation, not pure boilerplate.

- [ ] **Step 3: Checkpoint commit if commits are authorized**

If and only if commits are authorized, run:

```bash
git add README.md
git commit -m "docs: reposition AgentStore as marketplace foundation

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: Final Verification Pass

**Files:**
- No planned source changes unless verification exposes a defect.

- [ ] **Step 1: Run backend targeted tests**

Run:

```bash
cd backend && go test ./internal/llm/... ./internal/api/handlers/...
```

Expected: PASS, or integration tests SKIP only when test MongoDB is unavailable. If any test fails, fix the failure in the smallest relevant file and rerun this exact command.

- [ ] **Step 2: Run backend build**

Run:

```bash
cd backend && go build ./...
```

Expected: exit code 0. If it fails, fix compile errors and rerun the exact command.

- [ ] **Step 3: Run frontend typecheck**

Run:

```bash
cd frontend && npx tsc --noEmit
```

Expected: exit code 0. If it fails, fix type errors and rerun the exact command.

- [ ] **Step 4: Run changed frontend tests**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/ChatPage.test.tsx src/pages/admin/DashboardPage.test.tsx src/pages/admin/components/LaunchChecklist.test.tsx
```

Expected: PASS. If it fails, fix the relevant changed files and rerun this exact command.

- [ ] **Step 5: Optional validation tests if model/schema changes were made**

Only run this if implementation changed model structs, validation tags, or MongoDB schema definitions:

```bash
cd backend && go test ./internal/validation/...
```

Expected: PASS.

- [ ] **Step 6: Report verification evidence**

In the final implementation report, include:

- Exact backend test command and result.
- Exact backend build command and result.
- Exact frontend typecheck command and result.
- Exact frontend test command and result.
- Any skipped checks and the concrete reason.

- [ ] **Step 7: Final commit if commits are authorized**

If and only if commits are authorized and previous checkpoint commits were skipped, run:

```bash
git add backend/internal/llm/router.go backend/internal/llm/router_test.go backend/internal/api/handlers/chat.go backend/internal/api/handlers/chat_test.go backend/internal/api/handlers/admin_launch_readiness.go backend/cmd/server/main.go backend/internal/api/handlers/testhelpers_test.go backend/internal/api/handlers/admin_test.go frontend/src/types/index.ts frontend/src/api/client.ts frontend/src/pages/admin/components/LaunchChecklist.tsx frontend/src/pages/admin/components/LaunchChecklist.test.tsx frontend/src/pages/admin/DashboardPage.tsx frontend/src/pages/admin/DashboardPage.test.tsx frontend/src/pages/app/ChatPage.tsx frontend/src/pages/app/ChatPage.test.tsx README.md
git commit -m "feat: complete P0 productization closeout

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Self-Review Checklist

Spec coverage:

- Chat mobile drawer is covered by Task 5.
- Interrupted/failed assistant retry is covered by Task 6.
- Provider execution guard is covered by Tasks 1 and 2.
- Admin Launch Checklist readiness API is covered by Tasks 3 and 4.
- README positioning is covered by Task 7.
- Targeted verification is covered by Task 8.

Type consistency:

- Backend readiness status constants are `LaunchReadinessComplete`, `LaunchReadinessWarning`, and `LaunchReadinessPending`.
- Backend readiness item field is `ActionPath` with JSON `actionPath`.
- Frontend readiness item field is `actionPath`.
- `LaunchChecklist` consumes `LaunchReadinessItem[]` and links to `item.actionPath`.
- Chat message status values match existing backend and frontend types: `generating`, `completed`, `error`, `interrupted`.

Execution notes:

- The readiness handler code intentionally lives in a new file because `admin.go` is already large.
- The final implementation must not claim tests pass unless the verification commands in Task 8 were run and read fresh.
