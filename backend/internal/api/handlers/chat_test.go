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

	"lastsaas/internal/credits"
	"lastsaas/internal/llm"
	"lastsaas/internal/models"

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

func TestBuildSendMessageDeductionFailureUpdate_MarksAssistantErrorWithoutCharge(t *testing.T) {
	update := buildSendMessageDeductionFailureUpdate("configured answer")
	setMap := requireSingleSetMap(t, update)

	if got := setMap["status"]; got != models.ChatMessageStatusError {
		t.Fatalf("expected status %q, got %#v", models.ChatMessageStatusError, got)
	}
	if got := setMap["creditsCharged"]; got != 0 {
		t.Fatalf("expected creditsCharged 0, got %#v", got)
	}
	if got := setMap["content"]; got != "configured answer" {
		t.Fatalf("expected retained content, got %#v", got)
	}
}

func TestBuildStreamMessageSuccessUpdate_MarksCompletedAndCharges(t *testing.T) {
	update := buildStreamMessageSuccessUpdate("Hello", "test-model", 3)
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

	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB))

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
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB))

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
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB))

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
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB))

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
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB))

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
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB))

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
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB))

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
}

func TestSendMessage_DeductionRaceMarksAssistantMessageErrorWithoutCharging(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	ctx := context.Background()
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := env.DB.Tenants().UpdateOne(ctx,
			bson.M{"_id": tenant.ID},
			bson.M{"$set": bson.M{"subscriptionCredits": int64(0)}},
		)
		if err != nil {
			t.Fatalf("simulate concurrent credit drain: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"configured answer"}}]}`))
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
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB))

	// The provider above simulates another request spending the remaining credits
	// after this request has passed its preflight credit check.
	handler.SendMessage(rr, req)

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusPaymentRequired, rr.Code, rr.Body.String())
	}

	var saved []models.ChatMessage
	cursor, err := env.DB.ChatMessages().Find(ctx, bson.M{"conversationId": conversationID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
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
	if assistant.Status != models.ChatMessageStatusError {
		t.Fatalf("expected assistant status %q, got %q", models.ChatMessageStatusError, assistant.Status)
	}
	if assistant.CreditsCharged != 0 {
		t.Fatalf("expected assistant credits 0 after failed deduction, got %d", assistant.CreditsCharged)
	}
	if assistant.Content != "configured answer" {
		t.Fatalf("expected assistant content to be retained, got %q", assistant.Content)
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
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB))

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
	if assistant.Content != "Hello" {
		t.Fatalf("expected assistant content %q, got %q", "Hello", assistant.Content)
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
	handler := NewChatHandler(env.DB, credits.NewService(env.DB), llm.NewClientWithDB(env.DB), llm.NewRouter(env.DB))

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
