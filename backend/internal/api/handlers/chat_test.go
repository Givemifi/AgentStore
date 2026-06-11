package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"agentstore/internal/credits"
	"agentstore/internal/llm"
	"agentstore/internal/middleware"
	"agentstore/internal/models"
	"agentstore/internal/testutil"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestConversationTitleFromMessage_NormalizesWhitespaceAndTruncates(t *testing.T) {
	input := "  Hello\n\tthere    expert,   please help me understand why my build keeps failing after deploy.  "

	got := conversationTitleFromMessage(input)

	if strings.ContainsAny(got, "\n\t") {
		t.Fatalf("expected normalized title without tabs/newlines, got %q", got)
	}
	if strings.Contains(got, "  ") {
		t.Fatalf("expected collapsed internal whitespace, got %q", got)
	}
	if runeCount := utf8.RuneCountInString(got); runeCount < 40 || runeCount > 60 {
		t.Fatalf("expected UI-friendly title length, got %d (%q)", runeCount, got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected truncated title to end with ellipsis, got %q", got)
	}
}

func TestConversationTitleFromMessage_EmptyAfterWhitespace(t *testing.T) {
	got := conversationTitleFromMessage(" \n\t ")
	if got != "" {
		t.Fatalf("expected empty normalized title, got %q", got)
	}
}

func TestConversationTitleFromMessage_TruncatesUnicodeSafely(t *testing.T) {
	input := strings.Repeat("😀", conversationTitleMaxLength+12)

	got := conversationTitleFromMessage(input)

	if !utf8.ValidString(got) {
		t.Fatalf("expected valid UTF-8 title, got %q", got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected truncated title to end with ellipsis, got %q", got)
	}
	if runeCount := utf8.RuneCountInString(got); runeCount != conversationTitleMaxLength {
		t.Fatalf("expected truncated title to be %d runes, got %d (%q)", conversationTitleMaxLength, runeCount, got)
	}
	if got != strings.Repeat("😀", conversationTitleMaxLength-3)+"..." {
		t.Fatalf("expected rune-safe truncation, got %q", got)
	}
}

func TestBuildStreamMessageSuccessUpdate_MarksCompletedAndCharges(t *testing.T) {
	update := buildStreamMessageSuccessUpdate("Hello", "test-model", 3, llm.Usage{PromptTokens: 10, CompletionTokens: 5})
	setMap := requireSingleSetMap(t, update)

	if got := setMap["status"]; got != models.ChatMessageStatusCompleted {
		t.Fatalf("expected status %q, got %#v", models.ChatMessageStatusCompleted, got)
	}
	if got := setMap["creditsCharged"]; got != 3 {
		t.Fatalf("expected creditsCharged 3, got %#v", got)
	}
	if got := setMap["content"]; got != "Hello" {
		t.Fatalf("expected content %q, got %#v", "Hello", got)
	}
	if got := setMap["promptTokens"]; got != 10 {
		t.Fatalf("expected promptTokens 10, got %#v", got)
	}
	if got := setMap["completionTokens"]; got != 5 {
		t.Fatalf("expected completionTokens 5, got %#v", got)
	}
}

func TestBuildStreamMessageDeductionFailureUpdate_MarksAssistantErrorWithoutCharge(t *testing.T) {
	update := buildStreamMessageDeductionFailureUpdate("Hello", "test-model")
	setMap := requireSingleSetMap(t, update)

	if got := setMap["status"]; got != models.ChatMessageStatusError {
		t.Fatalf("expected status %q, got %#v", models.ChatMessageStatusError, got)
	}
	if got := setMap["creditsCharged"]; got != 0 {
		t.Fatalf("expected creditsCharged 0, got %#v", got)
	}
	if got := setMap["content"]; got != "Hello" {
		t.Fatalf("expected content %q, got %#v", "Hello", got)
	}
}

func TestStreamProviderFailureContent_UsesPlaceholderWhenNoContentGenerated(t *testing.T) {
	message := streamProviderFailureContent("")
	if strings.TrimSpace(message) == "" {
		t.Fatal("expected non-empty placeholder when no content was generated")
	}
}

func TestStreamProviderFailureContent_PreservesGeneratedContent(t *testing.T) {
	const partial = "Partial answer"
	if got := streamProviderFailureContent(partial); got != partial {
		t.Fatalf("expected generated content to be preserved, got %q", got)
	}
}

func TestMapCreditDeductionErrorStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "insufficient credits", err: credits.ErrInsufficientCredits, want: http.StatusPaymentRequired},
		{name: "balance changed", err: credits.ErrBalanceChanged, want: http.StatusPaymentRequired},
		{name: "other error", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapCreditDeductionErrorStatus(tt.err); got != tt.want {
				t.Fatalf("expected status %d, got %d", tt.want, got)
			}
		})
	}
}

func requireSingleSetMap(t *testing.T, update interface{}) bson.M {
	t.Helper()
	updateMap, ok := update.(bson.M)
	if !ok {
		t.Fatalf("expected bson.M update, got %#v", update)
	}
	setMap, ok := updateMap["$set"].(bson.M)
	if !ok {
		t.Fatalf("expected $set bson.M, got %#v", updateMap["$set"])
	}
	return setMap
}

func TestListAgents_DoesNotExposeSystemPrompt(t *testing.T) {
	rr := httptest.NewRecorder()
	handler := &ChatHandler{}

	handler.ListAgents(rr, httptest.NewRequest(http.MethodGet, "/api/chat/agents", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "systemPrompt") {
		t.Fatalf("expected systemPrompt to be redacted, got body %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "legal-expert") {
		t.Fatalf("expected public agent fields in response, got %s", rr.Body.String())
	}

	var got []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected at least one agent in response")
	}
	creditCost, ok := got[0]["creditCost"].(map[string]any)
	if !ok {
		t.Fatalf("expected creditCost object, got %#v", got[0]["creditCost"])
	}
	if creditCost["textMessageCredits"] == nil {
		t.Fatalf("expected textMessageCredits in creditCost, got %#v", creditCost)
	}
}

func TestGetAgent_DoesNotExposeSystemPrompt(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/chat/agents/legal-expert", nil)
	req = mux.SetURLVars(req, map[string]string{"agentId": "legal-expert"})
	rr := httptest.NewRecorder()
	handler := &ChatHandler{}

	handler.GetAgent(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "systemPrompt") {
		t.Fatalf("expected systemPrompt to be redacted, got body %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "legal-expert") {
		t.Fatalf("expected public agent fields in response, got %s", rr.Body.String())
	}

	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	creditCost, ok := got["creditCost"].(map[string]any)
	if !ok {
		t.Fatalf("expected creditCost object, got %#v", got["creditCost"])
	}
	if got["slug"] != "legal-expert" {
		t.Fatalf("expected slug legal-expert, got %#v", got["slug"])
	}
	if creditCost["textMessageCredits"] == nil {
		t.Fatalf("expected textMessageCredits in creditCost, got %#v", creditCost)
	}
}

func TestConversationMatchesAgent(t *testing.T) {
	tests := []struct {
		name    string
		conv    *models.Conversation
		agentID string
		want    bool
	}{
		{
			name:    "matching agent",
			conv:    &models.Conversation{AgentID: "customer-support"},
			agentID: "customer-support",
			want:    true,
		},
		{
			name:    "mismatched agent",
			conv:    &models.Conversation{AgentID: "customer-support"},
			agentID: "legal-expert",
			want:    false,
		},
		{
			name:    "nil conversation",
			conv:    nil,
			agentID: "customer-support",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := conversationMatchesAgent(tt.conv, tt.agentID); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestPublishedTenantAgentLookupFilter_UsesObjectIDAndSlugWhenIdentifierIsHex(t *testing.T) {
	tenantID := primitive.NewObjectID()
	agentObjectID := primitive.NewObjectID()

	filter := publishedTenantAgentLookupFilter(agentObjectID.Hex(), tenantID)

	if got := filter["tenantId"]; got != tenantID {
		t.Fatalf("expected tenantId %v, got %#v", tenantID, got)
	}
	if got := filter["status"]; got != models.AgentStatusPublished {
		t.Fatalf("expected published status, got %#v", got)
	}
	orConditions, ok := filter["$or"].(bson.A)
	if !ok {
		t.Fatalf("expected $or conditions, got %#v", filter["$or"])
	}
	if len(orConditions) != 2 {
		t.Fatalf("expected two lookup branches, got %d", len(orConditions))
	}
}

func TestPublishedTenantAgentLookupFilter_UsesSlugForNonHexIdentifier(t *testing.T) {
	tenantID := primitive.NewObjectID()
	agentSlug := "support-concierge"

	filter := publishedTenantAgentLookupFilter(agentSlug, tenantID)

	if got := filter["tenantId"]; got != tenantID {
		t.Fatalf("expected tenantId %v, got %#v", tenantID, got)
	}
	if got := filter["slug"]; got != agentSlug {
		t.Fatalf("expected slug %q, got %#v", agentSlug, got)
	}
	if _, hasOr := filter["$or"]; hasOr {
		t.Fatalf("expected non-hex lookup to avoid $or, got %#v", filter["$or"])
	}
}

func TestGetPublishedAgent_ResolvesTenantAgentBySlugAndObjectID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	now := time.Now()
	dbAgent := models.Agent{
		ID:           primitive.NewObjectID(),
		TenantID:     tenant.ID,
		Name:         "Support Concierge",
		Slug:         "support-concierge",
		Category:     "Support",
		Description:  "Tenant support concierge",
		Status:       models.AgentStatusPublished,
		Visibility:   models.AgentVisibilityPublic,
		SystemPrompt: "Help the tenant support team.",
		Capabilities: []models.AgentCapability{models.AgentCapabilityTextChat},
		CreditCost:   models.AgentCreditCost{TextMessageCredits: 4},
		CreatedBy:    user.ID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if _, err := env.DB.Agents().InsertOne(context.Background(), dbAgent); err != nil {
		t.Fatalf("seed tenant agent: %v", err)
	}

	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB), nil)

	agentBySlug, err := handler.getPublishedAgent(context.Background(), dbAgent.Slug, tenant.ID)
	if err != nil {
		t.Fatalf("resolve tenant agent by slug: %v", err)
	}
	if agentBySlug == nil {
		t.Fatal("expected tenant agent by slug")
	}
	if agentBySlug.ID != dbAgent.ID.Hex() {
		t.Fatalf("expected slug lookup id %s, got %s", dbAgent.ID.Hex(), agentBySlug.ID)
	}
	if agentBySlug.CreditCost != 4 {
		t.Fatalf("expected slug lookup credit cost 4, got %d", agentBySlug.CreditCost)
	}

	agentByID, err := handler.getPublishedAgent(context.Background(), dbAgent.ID.Hex(), tenant.ID)
	if err != nil {
		t.Fatalf("resolve tenant agent by object id: %v", err)
	}
	if agentByID == nil {
		t.Fatal("expected tenant agent by object id")
	}
	if agentByID.ID != dbAgent.ID.Hex() {
		t.Fatalf("expected object id lookup id %s, got %s", dbAgent.ID.Hex(), agentByID.ID)
	}
}

func TestGetAgent_ReturnsTenantPublishedAgentBySlugWithObjectCreditCost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	now := time.Now()
	dbAgent := models.Agent{
		ID:           primitive.NewObjectID(),
		TenantID:     tenant.ID,
		Name:         "Support Concierge",
		Slug:         "support-concierge",
		Category:     "Support",
		Description:  "Tenant support concierge",
		Status:       models.AgentStatusPublished,
		Visibility:   models.AgentVisibilityPublic,
		SystemPrompt: "Help the tenant support team.",
		Capabilities: []models.AgentCapability{models.AgentCapabilityTextChat},
		CreditCost:   models.AgentCreditCost{TextMessageCredits: 4, ImageGenerationCredits: 7, VideoGenerationCredits: 11},
		CreatedBy:    user.ID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if _, err := env.DB.Agents().InsertOne(context.Background(), dbAgent); err != nil {
		t.Fatalf("seed tenant agent: %v", err)
	}

	req := env.tenantRequest(t, http.MethodGet, "/api/chat/agents/"+dbAgent.Slug, nil, user, tenant.ID.Hex())
	req = mux.SetURLVars(req, map[string]string{"agentId": dbAgent.Slug})
	rr := httptest.NewRecorder()
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB), nil)

	handler.GetAgent(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "systemPrompt") {
		t.Fatalf("expected systemPrompt to be redacted, got body %s", rr.Body.String())
	}

	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got["slug"] != dbAgent.Slug {
		t.Fatalf("expected slug %q, got %#v", dbAgent.Slug, got["slug"])
	}
	creditCost, ok := got["creditCost"].(map[string]any)
	if !ok {
		t.Fatalf("expected creditCost object, got %#v", got["creditCost"])
	}
	if creditCost["textMessageCredits"] != float64(4) {
		t.Fatalf("expected textMessageCredits 4, got %#v", creditCost["textMessageCredits"])
	}
}

func TestListAgents_ReturnsTenantPublishedAgentWithObjectCreditCost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	now := time.Now()
	dbAgent := models.Agent{
		ID:           primitive.NewObjectID(),
		TenantID:     tenant.ID,
		Name:         "Support Concierge",
		Slug:         "support-concierge",
		Category:     "Support",
		Description:  "Tenant support concierge",
		Status:       models.AgentStatusPublished,
		Visibility:   models.AgentVisibilityPublic,
		SystemPrompt: "Help the tenant support team.",
		Capabilities: []models.AgentCapability{models.AgentCapabilityTextChat},
		CreditCost:   models.AgentCreditCost{TextMessageCredits: 4, ImageGenerationCredits: 7, VideoGenerationCredits: 11},
		CreatedBy:    user.ID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if _, err := env.DB.Agents().InsertOne(context.Background(), dbAgent); err != nil {
		t.Fatalf("seed tenant agent: %v", err)
	}

	req := env.tenantRequest(t, http.MethodGet, "/api/chat/agents", nil, user, tenant.ID.Hex())
	rr := httptest.NewRecorder()
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB), nil)

	handler.ListAgents(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "systemPrompt") {
		t.Fatalf("expected systemPrompt to be redacted, got body %s", rr.Body.String())
	}

	var got []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one tenant agent, got %d", len(got))
	}
	creditCost, ok := got[0]["creditCost"].(map[string]any)
	if !ok {
		t.Fatalf("expected creditCost object, got %#v", got[0]["creditCost"])
	}
	if creditCost["textMessageCredits"] != float64(4) {
		t.Fatalf("expected textMessageCredits 4, got %#v", creditCost["textMessageCredits"])
	}
}

func TestChatListAgentsIncludesRootPublishedAgentsForNormalTenants(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	rootUser, rootTenant := createAdminEnv(t, env)
	normalUser := testutil.CreateTestUser(t, env.DB, "member@test.com", "Test1234!@#$", "Member User")
	normalTenant := testutil.CreateTestTenant(t, env.DB, "Member Workspace", normalUser.ID, false)

	now := time.Now()
	platformAgent := models.Agent{
		ID:           primitive.NewObjectID(),
		TenantID:     rootTenant.ID,
		Name:         "Growth Copywriter",
		Slug:         "growth-copywriter",
		Category:     "Marketing",
		Description:  "Writes launch copy for commercial teams.",
		Status:       models.AgentStatusPublished,
		Visibility:   models.AgentVisibilityPublic,
		SystemPrompt: "Write high-converting marketing copy.",
		Capabilities: []models.AgentCapability{models.AgentCapabilityTextChat},
		CreditCost:   models.AgentCreditCost{TextMessageCredits: 2},
		CreatedBy:    rootUser.ID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if _, err := env.DB.Agents().InsertOne(context.Background(), platformAgent); err != nil {
		t.Fatalf("seed platform agent: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/chat/agents", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.TenantContextKey, normalTenant))
	rr := httptest.NewRecorder()
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB), nil)

	handler.ListAgents(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var got []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected platform agent in normal tenant marketplace")
	}
	if got[0]["id"] != platformAgent.ID.Hex() {
		t.Fatalf("expected canonical platform agent id %s, got %#v", platformAgent.ID.Hex(), got[0]["id"])
	}
	if got[0]["slug"] != platformAgent.Slug {
		t.Fatalf("expected platform agent slug %q, got %#v", platformAgent.Slug, got[0]["slug"])
	}
}

func TestStreamMessage_PersistsGeneratingPlaceholderThenCompletes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}))
	defer provider.Close()

	ctx := context.Background()
	_, err := env.DB.Tenants().UpdateOne(ctx,
		bson.M{"_id": tenant.ID},
		bson.M{"$set": bson.M{"subscriptionCredits": int64(50), "purchasedCredits": int64(0)}},
	)
	if err != nil {
		t.Fatalf("seed tenant credits: %v", err)
	}
	_, err = env.DB.LLMConfigs().InsertOne(ctx, models.LLMConfig{
		Key:       models.DefaultLLMConfigKey,
		APIKey:    "test-key",
		BaseURL:   provider.URL,
		Model:     "test-model",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("seed llm config: %v", err)
	}

	body, err := json.Marshal(SendMessageRequest{
		AgentID: "customer-support",
		Message: "Please help me debug this deploy issue.",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := env.tenantRequest(t, http.MethodPost, "/api/chat/stream", strings.NewReader(string(body)), user, tenant.ID.Hex())
	rr := httptest.NewRecorder()
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB), nil)

	handler.StreamMessage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "event: message_done") {
		t.Fatalf("expected stream completion event, got %s", rr.Body.String())
	}

	var saved []models.ChatMessage
	cursor, err := env.DB.ChatMessages().Find(ctx, bson.M{"tenantId": tenant.ID, "userId": user.ID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		t.Fatalf("find chat messages: %v", err)
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &saved); err != nil {
		t.Fatalf("decode chat messages: %v", err)
	}
	if len(saved) != 2 {
		t.Fatalf("expected 2 saved messages, got %d", len(saved))
	}
	assistant := saved[1]
	if assistant.Role != "assistant" {
		t.Fatalf("expected assistant role, got %q", assistant.Role)
	}
	if assistant.Status != models.ChatMessageStatusCompleted {
		t.Fatalf("expected completed status, got %q", assistant.Status)
	}
	if assistant.Content != "Hello" {
		t.Fatalf("expected assistant content %q, got %q", "Hello", assistant.Content)
	}
	if assistant.CreditsCharged != 3 {
		t.Fatalf("expected assistant credits 3, got %d", assistant.CreditsCharged)
	}
}

func TestStreamMessageStoresCanonicalAgentIDForSlugRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}))
	defer provider.Close()

	ctx := context.Background()
	_, err := env.DB.Tenants().UpdateOne(ctx,
		bson.M{"_id": tenant.ID},
		bson.M{"$set": bson.M{"subscriptionCredits": int64(50), "purchasedCredits": int64(0)}},
	)
	if err != nil {
		t.Fatalf("seed tenant credits: %v", err)
	}
	_, err = env.DB.LLMConfigs().InsertOne(ctx, models.LLMConfig{
		Key:       models.DefaultLLMConfigKey,
		APIKey:    "test-key",
		BaseURL:   provider.URL,
		Model:     "test-model",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("seed llm config: %v", err)
	}

	now := time.Now()
	dbAgent := models.Agent{
		ID:           primitive.NewObjectID(),
		TenantID:     tenant.ID,
		Name:         "Support Concierge",
		Slug:         "support-concierge",
		Category:     "Support",
		Description:  "Tenant support concierge",
		Status:       models.AgentStatusPublished,
		Visibility:   models.AgentVisibilityPublic,
		SystemPrompt: "Help the tenant support team.",
		Capabilities: []models.AgentCapability{models.AgentCapabilityTextChat},
		CreditCost:   models.AgentCreditCost{TextMessageCredits: 4},
		CreatedBy:    user.ID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if _, err := env.DB.Agents().InsertOne(context.Background(), dbAgent); err != nil {
		t.Fatalf("seed tenant agent: %v", err)
	}

	body, err := json.Marshal(SendMessageRequest{
		AgentID: dbAgent.Slug,
		Message: "Please help me debug this deploy issue.",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", strings.NewReader(string(body)))
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))
	req = req.WithContext(context.WithValue(req.Context(), middleware.TenantContextKey, tenant))
	rr := httptest.NewRecorder()
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB), nil)

	handler.StreamMessage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var conv models.Conversation
	if err := env.DB.Conversations().FindOne(ctx, bson.M{"tenantId": tenant.ID, "userId": user.ID}).Decode(&conv); err != nil {
		t.Fatalf("find conversation: %v", err)
	}
	if conv.AgentID != dbAgent.ID.Hex() {
		t.Fatalf("expected canonical agent id %s, got %s", dbAgent.ID.Hex(), conv.AgentID)
	}
}

func TestSendMessage_RejectsCrossAgentConversation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	ctx := context.Background()
	_, err := env.DB.Tenants().UpdateOne(ctx,
		bson.M{"_id": tenant.ID},
		bson.M{"$set": bson.M{"subscriptionCredits": int64(50)}},
	)
	if err != nil {
		t.Fatalf("seed tenant credits: %v", err)
	}

	conversationID := primitive.NewObjectID()
	_, err = env.DB.Conversations().InsertOne(ctx, models.Conversation{
		ID:        conversationID,
		TenantID:  tenant.ID,
		UserID:    user.ID,
		AgentID:   "customer-support",
		Title:     "Existing support thread",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("seed conversation: %v", err)
	}

	body, err := json.Marshal(SendMessageRequest{
		AgentID:        "legal-expert",
		ConversationID: conversationID.Hex(),
		Message:        "Can you switch this conversation to legal?",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := env.tenantRequest(t, http.MethodPost, "/api/chat/send", strings.NewReader(string(body)), user, tenant.ID.Hex())
	rr := httptest.NewRecorder()
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB), nil)

	handler.SendMessage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error != "Conversation agent does not match request" {
		t.Fatalf("expected agent mismatch error, got %q", resp.Error)
	}

	count, err := env.DB.ChatMessages().CountDocuments(ctx, bson.M{"conversationId": conversationID})
	if err != nil {
		t.Fatalf("count chat messages: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no messages saved for rejected request, got %d", count)
	}
}

func TestSendMessage_ConfigFailureDoesNotCreateConversationOrMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	ctx := context.Background()
	_, err := env.DB.Tenants().UpdateOne(ctx,
		bson.M{"_id": tenant.ID},
		bson.M{"$set": bson.M{"subscriptionCredits": int64(50)}},
	)
	if err != nil {
		t.Fatalf("seed tenant credits: %v", err)
	}

	body, err := json.Marshal(SendMessageRequest{
		AgentID: "customer-support",
		Message: "Please help me debug this deploy issue.",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := env.tenantRequest(t, http.MethodPost, "/api/chat/send", strings.NewReader(string(body)), user, tenant.ID.Hex())
	rr := httptest.NewRecorder()
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB), nil)

	handler.SendMessage(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusInternalServerError, rr.Code, rr.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error != "AI model is not configured. Please contact an administrator." {
		t.Fatalf("expected config error, got %q", resp.Error)
	}

	conversationCount, err := env.DB.Conversations().CountDocuments(ctx, bson.M{"tenantId": tenant.ID, "userId": user.ID})
	if err != nil {
		t.Fatalf("count conversations: %v", err)
	}
	if conversationCount != 0 {
		t.Fatalf("expected no conversations saved on config failure, got %d", conversationCount)
	}

	messageCount, err := env.DB.ChatMessages().CountDocuments(ctx, bson.M{"tenantId": tenant.ID, "userId": user.ID})
	if err != nil {
		t.Fatalf("count chat messages: %v", err)
	}
	if messageCount != 0 {
		t.Fatalf("expected no messages saved on config failure, got %d", messageCount)
	}
}

func TestSendMessage_LLMFailureDoesNotCreateConversationOrMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"provider failed"}`, http.StatusInternalServerError)
	}))
	defer provider.Close()

	ctx := context.Background()
	_, err := env.DB.Tenants().UpdateOne(ctx,
		bson.M{"_id": tenant.ID},
		bson.M{"$set": bson.M{"subscriptionCredits": int64(50)}},
	)
	if err != nil {
		t.Fatalf("seed tenant credits: %v", err)
	}
	_, err = env.DB.LLMConfigs().InsertOne(ctx, models.LLMConfig{
		Key:       models.DefaultLLMConfigKey,
		APIKey:    "test-key",
		BaseURL:   provider.URL,
		Model:     "test-model",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("seed llm config: %v", err)
	}

	body, err := json.Marshal(SendMessageRequest{
		AgentID: "customer-support",
		Message: "Please help me debug this deploy issue.",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := env.tenantRequest(t, http.MethodPost, "/api/chat/send", strings.NewReader(string(body)), user, tenant.ID.Hex())
	rr := httptest.NewRecorder()
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB), nil)

	handler.SendMessage(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusInternalServerError, rr.Code, rr.Body.String())
	}

	conversationCount, err := env.DB.Conversations().CountDocuments(ctx, bson.M{"tenantId": tenant.ID, "userId": user.ID})
	if err != nil {
		t.Fatalf("count conversations: %v", err)
	}
	if conversationCount != 0 {
		t.Fatalf("expected no conversations saved on LLM failure, got %d", conversationCount)
	}

	messageCount, err := env.DB.ChatMessages().CountDocuments(ctx, bson.M{"tenantId": tenant.ID, "userId": user.ID})
	if err != nil {
		t.Fatalf("count chat messages: %v", err)
	}
	if messageCount != 0 {
		t.Fatalf("expected no messages saved on LLM failure, got %d", messageCount)
	}

	// Credits are deducted before the LLM call and refunded when it fails, so the
	// combined balance must be fully restored to the seeded 50.
	var refreshed models.Tenant
	if err := env.DB.Tenants().FindOne(ctx, bson.M{"_id": tenant.ID}).Decode(&refreshed); err != nil {
		t.Fatalf("refetch tenant: %v", err)
	}
	if total := refreshed.SubscriptionCredits + refreshed.PurchasedCredits; total != 50 {
		t.Fatalf("expected credits fully refunded to 50 after LLM failure, got %d (subscription=%d purchased=%d)",
			total, refreshed.SubscriptionCredits, refreshed.PurchasedCredits)
	}
}

func TestSendMessage_InsufficientCreditsChargesBeforeLLMAndPersistsNothing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	ctx := context.Background()
	var providerCalled bool
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerCalled = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"configured answer"}}]}`))
	}))
	defer provider.Close()

	// Tenant has zero credits: deduction (which now runs BEFORE the LLM call)
	// must fail up front, so the provider is never hit and nothing is persisted.
	_, err := env.DB.Tenants().UpdateOne(ctx,
		bson.M{"_id": tenant.ID},
		bson.M{"$set": bson.M{"subscriptionCredits": int64(0), "purchasedCredits": int64(0)}},
	)
	if err != nil {
		t.Fatalf("seed tenant credits: %v", err)
	}
	_, err = env.DB.LLMConfigs().InsertOne(ctx, models.LLMConfig{
		Key:       models.DefaultLLMConfigKey,
		APIKey:    "test-key",
		BaseURL:   provider.URL,
		Model:     "test-model",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("seed llm config: %v", err)
	}

	conversationID := primitive.NewObjectID()
	_, err = env.DB.Conversations().InsertOne(ctx, models.Conversation{
		ID:        conversationID,
		TenantID:  tenant.ID,
		UserID:    user.ID,
		AgentID:   "customer-support",
		Title:     "Existing support thread",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("seed conversation: %v", err)
	}

	body, err := json.Marshal(SendMessageRequest{
		AgentID:        "customer-support",
		ConversationID: conversationID.Hex(),
		Message:        "Please help me debug this deploy issue.",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := env.tenantRequest(t, http.MethodPost, "/api/chat/send", strings.NewReader(string(body)), user, tenant.ID.Hex())
	rr := httptest.NewRecorder()
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB), nil)

	handler.SendMessage(rr, req)

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusPaymentRequired, rr.Code, rr.Body.String())
	}
	if providerCalled {
		t.Fatal("expected LLM provider NOT to be called when credits are insufficient")
	}

	count, err := env.DB.ChatMessages().CountDocuments(ctx, bson.M{"conversationId": conversationID})
	if err != nil {
		t.Fatalf("count chat messages: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no messages persisted on insufficient credits, got %d", count)
	}
}

func TestStreamMessage_DeductionFailureMarksAssistantMessageErrorAndSkipsMessageDone(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	ctx := context.Background()
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}))
	defer provider.Close()

	_, err := env.DB.Tenants().UpdateOne(ctx,
		bson.M{"_id": tenant.ID},
		bson.M{"$set": bson.M{"subscriptionCredits": int64(3), "purchasedCredits": int64(0)}},
	)
	if err != nil {
		t.Fatalf("seed tenant credits: %v", err)
	}
	_, err = env.DB.LLMConfigs().InsertOne(ctx, models.LLMConfig{
		Key:       models.DefaultLLMConfigKey,
		APIKey:    "test-key",
		BaseURL:   provider.URL,
		Model:     "test-model",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("seed llm config: %v", err)
	}

	body, err := json.Marshal(SendMessageRequest{
		AgentID: "customer-support",
		Message: "Please help me debug this deploy issue.",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := env.tenantRequest(t, http.MethodPost, "/api/chat/stream", strings.NewReader(string(body)), user, tenant.ID.Hex())
	rr := httptest.NewRecorder()
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB), nil)

	conversationCountBefore, err := env.DB.Conversations().CountDocuments(ctx, bson.M{"tenantId": tenant.ID, "userId": user.ID})
	if err != nil {
		t.Fatalf("count conversations before stream: %v", err)
	}

	_, err = env.DB.Tenants().UpdateOne(ctx,
		bson.M{"_id": tenant.ID},
		bson.M{"$set": bson.M{"subscriptionCredits": int64(0), "purchasedCredits": int64(0)}},
	)
	if err != nil {
		t.Fatalf("simulate concurrent credit drain: %v", err)
	}

	handler.StreamMessage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "event: message_done") {
		t.Fatalf("expected no completion event on deduction failure, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "event: error") {
		t.Fatalf("expected error event on deduction failure, got %s", rr.Body.String())
	}

	var saved []models.ChatMessage
	cursor, err := env.DB.ChatMessages().Find(ctx, bson.M{"tenantId": tenant.ID, "userId": user.ID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		t.Fatalf("find chat messages: %v", err)
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &saved); err != nil {
		t.Fatalf("decode chat messages: %v", err)
	}
	if len(saved) != 2 {
		t.Fatalf("expected 2 saved messages, got %d", len(saved))
	}
	assistant := saved[1]
	if assistant.Status != models.ChatMessageStatusError {
		t.Fatalf("expected assistant status %q, got %q", models.ChatMessageStatusError, assistant.Status)
	}
	if assistant.CreditsCharged != 0 {
		t.Fatalf("expected assistant credits 0 after failed deduction, got %d", assistant.CreditsCharged)
	}
	// Deduction now runs BEFORE streaming, so when it fails up front the provider
	// is never consumed and no content is produced.
	if assistant.Content != "" {
		t.Fatalf("expected empty assistant content when deduction fails before streaming, got %q", assistant.Content)
	}

	conversationCountAfter, err := env.DB.Conversations().CountDocuments(ctx, bson.M{"tenantId": tenant.ID, "userId": user.ID})
	if err != nil {
		t.Fatalf("count conversations after stream: %v", err)
	}
	if conversationCountAfter != conversationCountBefore+1 {
		t.Fatalf("expected conversation to remain saved, before=%d after=%d", conversationCountBefore, conversationCountAfter)
	}
}

func TestStreamMessage_ProviderFailureBeforeContentMarksAssistantMessageErrorWithPlaceholder(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"provider failed"}`, http.StatusInternalServerError)
	}))
	defer provider.Close()

	ctx := context.Background()
	_, err := env.DB.Tenants().UpdateOne(ctx,
		bson.M{"_id": tenant.ID},
		bson.M{"$set": bson.M{"subscriptionCredits": int64(50), "purchasedCredits": int64(0)}},
	)
	if err != nil {
		t.Fatalf("seed tenant credits: %v", err)
	}
	_, err = env.DB.LLMConfigs().InsertOne(ctx, models.LLMConfig{
		Key:       models.DefaultLLMConfigKey,
		APIKey:    "test-key",
		BaseURL:   provider.URL,
		Model:     "test-model",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("seed llm config: %v", err)
	}

	body, err := json.Marshal(SendMessageRequest{
		AgentID: "customer-support",
		Message: "Please help me debug this deploy issue.",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := env.tenantRequest(t, http.MethodPost, "/api/chat/stream", strings.NewReader(string(body)), user, tenant.ID.Hex())
	rr := httptest.NewRecorder()
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB), nil)

	handler.StreamMessage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "event: error") {
		t.Fatalf("expected error event, got %s", rr.Body.String())
	}

	var saved []models.ChatMessage
	cursor, err := env.DB.ChatMessages().Find(ctx, bson.M{"tenantId": tenant.ID, "userId": user.ID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		t.Fatalf("find chat messages: %v", err)
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &saved); err != nil {
		t.Fatalf("decode chat messages: %v", err)
	}
	if len(saved) != 2 {
		t.Fatalf("expected 2 saved messages, got %d", len(saved))
	}
	assistant := saved[1]
	if assistant.Status != models.ChatMessageStatusError {
		t.Fatalf("expected assistant status %q, got %q", models.ChatMessageStatusError, assistant.Status)
	}
	if strings.TrimSpace(assistant.Content) == "" {
		t.Fatal("expected assistant error placeholder content when generation fails before any output")
	}
	if assistant.CreditsCharged != 0 {
		t.Fatalf("expected assistant credits 0 on provider failure, got %d", assistant.CreditsCharged)
	}
}

func TestLLMConfigUsable(t *testing.T) {
	tests := []struct {
		name   string
		config *models.LLMConfig
		want   bool
	}{
		{
			name: "usable active config",
			config: &models.LLMConfig{
				APIKey:   "key",
				BaseURL:  "https://example.com",
				Model:    "gpt-test",
				IsActive: true,
			},
			want: true,
		},
		{
			name:   "missing config",
			config: nil,
			want:   false,
		},
		{
			name: "inactive config",
			config: &models.LLMConfig{
				APIKey:   "key",
				BaseURL:  "https://example.com",
				Model:    "gpt-test",
				IsActive: false,
			},
			want: false,
		},
		{
			name: "blank api key",
			config: &models.LLMConfig{
				APIKey:   "   ",
				BaseURL:  "https://example.com",
				Model:    "gpt-test",
				IsActive: true,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := llmConfigUsable(tt.config); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

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
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB), nil)

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

func TestTruncateHistory_KeepsRecentWithinBudget(t *testing.T) {
	// Each message ~25 ASCII chars => ~6 tokens. With a tiny budget, only the
	// most recent messages (above the floor) survive.
	mk := func(role string) llm.Message {
		return llm.Message{Role: role, Content: strings.Repeat("a", 400)} // ~100 tokens
	}
	msgs := []llm.Message{mk("user"), mk("assistant"), mk("user"), mk("assistant"), mk("user")}

	got := truncateHistory(msgs, 250)
	if len(got) < minRetainedMessages {
		t.Fatalf("expected at least %d retained, got %d", minRetainedMessages, len(got))
	}
	if len(got) >= len(msgs) {
		t.Fatalf("expected truncation, kept all %d messages", len(got))
	}
	// The last message must always be present (chronological tail).
	if got[len(got)-1].Content != msgs[len(msgs)-1].Content {
		t.Fatal("expected the most recent message to be retained")
	}
}

func TestTruncateHistory_ShortHistoryUntouched(t *testing.T) {
	msgs := []llm.Message{{Role: "user", Content: "hi"}}
	got := truncateHistory(msgs, 10)
	if len(got) != 1 {
		t.Fatalf("expected short history untouched, got %d", len(got))
	}
}

func TestTruncateHistory_AlwaysKeepsFloor(t *testing.T) {
	big := llm.Message{Role: "user", Content: strings.Repeat("x", 100000)}
	msgs := []llm.Message{big, big, big}
	got := truncateHistory(msgs, 1) // budget far below a single message
	if len(got) != minRetainedMessages {
		t.Fatalf("expected floor of %d, got %d", minRetainedMessages, len(got))
	}
}

func newChatHandlerForTest(env *testEnv) *ChatHandler {
	return NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB), nil)
}

func TestDeleteConversation_CascadesMessagesFeedbackAnnotations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	ctx := context.Background()

	conversationID := primitive.NewObjectID()
	messageID := primitive.NewObjectID()
	if _, err := env.DB.Conversations().InsertOne(ctx, models.Conversation{
		ID: conversationID, TenantID: tenant.ID, UserID: user.ID, AgentID: "customer-support",
		Title: "Old chat", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed conversation: %v", err)
	}
	if _, err := env.DB.ChatMessages().InsertOne(ctx, models.ChatMessage{
		ID: messageID, TenantID: tenant.ID, UserID: user.ID, ConversationID: conversationID,
		AgentID: "customer-support", Role: "assistant", Content: "hi", CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed message: %v", err)
	}
	if _, err := env.DB.MessageFeedback().InsertOne(ctx, models.MessageFeedback{
		ID: primitive.NewObjectID(), TenantID: tenant.ID, UserID: user.ID, ConversationID: conversationID,
		MessageID: messageID, AgentID: "customer-support", Rating: -1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed feedback: %v", err)
	}
	if _, err := env.DB.Annotations().InsertOne(ctx, models.Annotation{
		ID: primitive.NewObjectID(), TenantID: tenant.ID, ConversationID: conversationID, MessageID: messageID,
		AgentID: "customer-support", AnnotatorID: user.ID, QualityScore: 2, Status: models.AnnotationStatusAnnotated,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed annotation: %v", err)
	}

	req := env.tenantRequest(t, http.MethodDelete, "/api/chat/conversations/"+conversationID.Hex(), strings.NewReader(""), user, tenant.ID.Hex())
	req = mux.SetURLVars(req, map[string]string{"conversationId": conversationID.Hex()})
	rr := httptest.NewRecorder()
	newChatHandlerForTest(env).DeleteConversation(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	for name, count := range map[string]func() (int64, error){
		"conversations": func() (int64, error) { return env.DB.Conversations().CountDocuments(ctx, bson.M{"_id": conversationID}) },
		"messages":      func() (int64, error) { return env.DB.ChatMessages().CountDocuments(ctx, bson.M{"conversationId": conversationID}) },
		"feedback":      func() (int64, error) { return env.DB.MessageFeedback().CountDocuments(ctx, bson.M{"conversationId": conversationID}) },
		"annotations":   func() (int64, error) { return env.DB.Annotations().CountDocuments(ctx, bson.M{"conversationId": conversationID}) },
	} {
		n, err := count()
		if err != nil {
			t.Fatalf("count %s: %v", name, err)
		}
		if n != 0 {
			t.Fatalf("expected %s cascade-deleted, got %d remaining", name, n)
		}
	}
}

func TestDeleteConversation_OtherUsersConversationNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	ctx := context.Background()

	otherUser := testutil.CreateTestUser(t, env.DB, "other@test.com", "Test1234!@#$", "Other")
	conversationID := primitive.NewObjectID()
	if _, err := env.DB.Conversations().InsertOne(ctx, models.Conversation{
		ID: conversationID, TenantID: tenant.ID, UserID: otherUser.ID, AgentID: "customer-support",
		Title: "Not yours", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed conversation: %v", err)
	}

	req := env.tenantRequest(t, http.MethodDelete, "/api/chat/conversations/"+conversationID.Hex(), strings.NewReader(""), user, tenant.ID.Hex())
	req = mux.SetURLVars(req, map[string]string{"conversationId": conversationID.Hex()})
	rr := httptest.NewRecorder()
	newChatHandlerForTest(env).DeleteConversation(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for other user's conversation, got %d", rr.Code)
	}
	n, _ := env.DB.Conversations().CountDocuments(ctx, bson.M{"_id": conversationID})
	if n != 1 {
		t.Fatalf("expected other user's conversation to survive, got %d", n)
	}
}

func TestRenameConversation_ValidatesAndUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	ctx := context.Background()

	conversationID := primitive.NewObjectID()
	if _, err := env.DB.Conversations().InsertOne(ctx, models.Conversation{
		ID: conversationID, TenantID: tenant.ID, UserID: user.ID, AgentID: "customer-support",
		Title: "Original", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed conversation: %v", err)
	}

	doRename := func(title string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"title": title})
		req := env.tenantRequest(t, http.MethodPatch, "/api/chat/conversations/"+conversationID.Hex(), strings.NewReader(string(body)), user, tenant.ID.Hex())
		req = mux.SetURLVars(req, map[string]string{"conversationId": conversationID.Hex()})
		rr := httptest.NewRecorder()
		newChatHandlerForTest(env).RenameConversation(rr, req)
		return rr
	}

	if rr := doRename("   "); rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty title, got %d", rr.Code)
	}
	if rr := doRename(strings.Repeat("x", 201)); rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for over-long title, got %d", rr.Code)
	}
	if rr := doRename("My renamed chat"); rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid title, got %d (%s)", rr.Code, rr.Body.String())
	}

	var conv models.Conversation
	if err := env.DB.Conversations().FindOne(ctx, bson.M{"_id": conversationID}).Decode(&conv); err != nil {
		t.Fatalf("reload conversation: %v", err)
	}
	if conv.Title != "My renamed chat" {
		t.Fatalf("expected updated title, got %q", conv.Title)
	}
}

func TestStreamRegenerate_ReplacesAssistantReply(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	ctx := context.Background()

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Regenerated reply\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}))
	defer provider.Close()

	if _, err := env.DB.Tenants().UpdateOne(ctx, bson.M{"_id": tenant.ID},
		bson.M{"$set": bson.M{"subscriptionCredits": int64(50), "purchasedCredits": int64(0)}}); err != nil {
		t.Fatalf("seed credits: %v", err)
	}
	if _, err := env.DB.LLMConfigs().InsertOne(ctx, models.LLMConfig{
		Key: models.DefaultLLMConfigKey, APIKey: "test-key", BaseURL: provider.URL, Model: "test-model",
		IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed llm config: %v", err)
	}

	conversationID := primitive.NewObjectID()
	if _, err := env.DB.Conversations().InsertOne(ctx, models.Conversation{
		ID: conversationID, TenantID: tenant.ID, UserID: user.ID, AgentID: "customer-support",
		Title: "Chat", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed conversation: %v", err)
	}
	base := time.Now()
	if _, err := env.DB.ChatMessages().InsertOne(ctx, models.ChatMessage{
		ID: primitive.NewObjectID(), TenantID: tenant.ID, UserID: user.ID, ConversationID: conversationID,
		AgentID: "customer-support", Role: "user", Content: "What is your refund policy?", CreatedAt: base,
	}); err != nil {
		t.Fatalf("seed user message: %v", err)
	}
	oldAssistantID := primitive.NewObjectID()
	if _, err := env.DB.ChatMessages().InsertOne(ctx, models.ChatMessage{
		ID: oldAssistantID, TenantID: tenant.ID, UserID: user.ID, ConversationID: conversationID,
		AgentID: "customer-support", Role: "assistant", Content: "Old answer", Status: models.ChatMessageStatusCompleted,
		CreditsCharged: 3, CreatedAt: base.Add(time.Second),
	}); err != nil {
		t.Fatalf("seed old assistant message: %v", err)
	}

	body, _ := json.Marshal(RegenerateRequest{ConversationID: conversationID.Hex(), MessageID: oldAssistantID.Hex()})
	req := env.tenantRequest(t, http.MethodPost, "/api/chat/regenerate", strings.NewReader(string(body)), user, tenant.ID.Hex())
	rr := httptest.NewRecorder()
	newChatHandlerForTest(env).StreamRegenerate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "event: message_done") {
		t.Fatalf("expected stream completion, got %s", rr.Body.String())
	}

	// Old assistant message removed.
	if n, _ := env.DB.ChatMessages().CountDocuments(ctx, bson.M{"_id": oldAssistantID}); n != 0 {
		t.Fatalf("expected old assistant message deleted, got %d", n)
	}
	// Exactly one user + one (new) assistant message remain.
	var msgs []models.ChatMessage
	cursor, err := env.DB.ChatMessages().Find(ctx, bson.M{"conversationId": conversationID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		t.Fatalf("find messages: %v", err)
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &msgs); err != nil {
		t.Fatalf("decode messages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages after regenerate (no duplicate user turn), got %d", len(msgs))
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "Regenerated reply" {
		t.Fatalf("expected regenerated assistant reply, got role=%q content=%q", msgs[1].Role, msgs[1].Content)
	}
}
