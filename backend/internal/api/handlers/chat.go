package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"agentstore/internal/agents"
	"agentstore/internal/credits"
	"agentstore/internal/db"
	"agentstore/internal/knowledge"
	"agentstore/internal/llm"
	"agentstore/internal/middleware"
	"agentstore/internal/models"
	"agentstore/internal/validation"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ChatHandler handles AI chat endpoints.
type ChatHandler struct {
	db           *db.MongoDB
	creditsSvc   *credits.Service
	llmClient    *llm.Client
	modelRouter  *llm.Router
	knowledgeSvc *knowledge.Service
}

type publicAgent struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Slug             string                 `json:"slug,omitempty"`
	Category         string                 `json:"category"`
	Description      string                 `json:"description"`
	Avatar           string                 `json:"avatar,omitempty"`
	Icon             string                 `json:"icon,omitempty"`
	Color            string                 `json:"color,omitempty"`
	Visibility       string                 `json:"visibility,omitempty"`
	WelcomeMessage   string                 `json:"welcomeMessage,omitempty"`
	SuggestedPrompts []string               `json:"suggestedPrompts,omitempty"`
	Capabilities     []string               `json:"capabilities,omitempty"`
	CreditCost       models.AgentCreditCost `json:"creditCost"`
	Examples         []string               `json:"examples,omitempty"`
}

func toPublicAgent(agent agents.Agent) publicAgent {
	return publicAgent{
		ID:          agent.ID,
		Name:        agent.Name,
		Slug:        agent.ID,
		Category:    agent.Category,
		Description: agent.Description,
		CreditCost: models.AgentCreditCost{
			TextMessageCredits:     agent.CreditCost,
			ImageGenerationCredits: 0,
			VideoGenerationCredits: 0,
		},
		WelcomeMessage:   agent.WelcomeMessage,
		SuggestedPrompts: append([]string(nil), agent.SuggestedPrompts...),
		Examples:         append([]string(nil), agent.Examples...),
		Icon:             agent.Icon,
		Color:            agent.Color,
	}
}

func dbAgentToPublicAgent(agent models.Agent) publicAgent {
	pub := agent.ToPublic()
	return publicAgent{
		ID:               pub.ID,
		Name:             pub.Name,
		Slug:             pub.Slug,
		Category:         pub.Category,
		Description:      pub.Description,
		Avatar:           pub.Avatar,
		Icon:             pub.Icon,
		Color:            pub.Color,
		Visibility:       string(pub.Visibility),
		WelcomeMessage:   pub.WelcomeMessage,
		SuggestedPrompts: append([]string(nil), pub.SuggestedPrompts...),
		Capabilities:     agentCapabilitiesToStrings(pub.Capabilities),
		CreditCost:       pub.CreditCost,
	}
}

// NewChatHandler creates a new chat handler.
func NewChatHandler(database *db.MongoDB, creditsSvc *credits.Service, llmClient *llm.Client, modelRouter *llm.Router, knowledgeSvc *knowledge.Service) *ChatHandler {
	return &ChatHandler{
		db:           database,
		creditsSvc:   creditsSvc,
		llmClient:    llmClient,
		modelRouter:  modelRouter,
		knowledgeSvc: knowledgeSvc,
	}
}

// ListAgents returns available AI experts (DB agents first, then fallback to static catalog).
func (h *ChatHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	if h.db != nil {
		if tenant, ok := middleware.GetTenantFromContext(r.Context()); ok {
			platformAgents, err := h.listPlatformAgents(r.Context(), tenant.ID)
			if err == nil && len(platformAgents) > 0 {
				respondWithJSON(w, http.StatusOK, platformAgents)
				return
			}

			cursor, err := h.db.Agents().Find(r.Context(), bson.M{"tenantId": tenant.ID, "status": models.AgentStatusPublished})
			if err == nil {
				defer cursor.Close(r.Context())
				var dbAgents []models.Agent
				if err := cursor.All(r.Context(), &dbAgents); err == nil && len(dbAgents) > 0 {
					publicAgents := make([]publicAgent, len(dbAgents))
					for i, agent := range dbAgents {
						publicAgents[i] = dbAgentToPublicAgent(agent)
					}
					respondWithJSON(w, http.StatusOK, publicAgents)
					return
				}
			}
		}
	}

	agentList := agents.GetAllAgents()
	publicAgents := make([]publicAgent, len(agentList))
	for i, agent := range agentList {
		publicAgents[i] = toPublicAgent(agent)
	}
	respondWithJSON(w, http.StatusOK, publicAgents)
}

func (h *ChatHandler) listPlatformAgents(ctx context.Context, currentTenantID primitive.ObjectID) ([]publicAgent, error) {
	cursor, err := h.db.Tenants().Find(ctx, bson.M{"isRoot": true, "isActive": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rootTenants []models.Tenant
	if err := cursor.All(ctx, &rootTenants); err != nil {
		return nil, err
	}
	if len(rootTenants) == 0 {
		return nil, nil
	}

	rootTenantIDs := make(bson.A, 0, len(rootTenants))
	for _, tenant := range rootTenants {
		if tenant.ID != currentTenantID {
			rootTenantIDs = append(rootTenantIDs, tenant.ID)
		}
	}
	if len(rootTenantIDs) == 0 {
		return nil, nil
	}

	agentCursor, err := h.db.Agents().Find(ctx, bson.M{
		"tenantId": bson.M{"$in": rootTenantIDs},
		"status":   models.AgentStatusPublished,
	}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer agentCursor.Close(ctx)

	var dbAgents []models.Agent
	if err := agentCursor.All(ctx, &dbAgents); err != nil {
		return nil, err
	}
	if len(dbAgents) == 0 {
		return nil, nil
	}

	publicAgents := make([]publicAgent, len(dbAgents))
	for i, agent := range dbAgents {
		publicAgents[i] = dbAgentToPublicAgent(agent)
	}
	return publicAgents, nil
}

// GetAgent returns a single agent by ID.
func (h *ChatHandler) GetAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agentId"]

	if tenant, ok := middleware.GetTenantFromContext(r.Context()); ok && h.db != nil {
		var dbAgent models.Agent
		err := h.db.Agents().FindOne(r.Context(), publishedTenantAgentLookupFilter(agentID, tenant.ID)).Decode(&dbAgent)
		if err == nil {
			pub := dbAgent.ToPublic()
			respondWithJSON(w, http.StatusOK, publicAgent{
				ID:               pub.ID,
				Name:             pub.Name,
				Slug:             pub.Slug,
				Category:         pub.Category,
				Description:      pub.Description,
				Avatar:           pub.Avatar,
				Icon:             pub.Icon,
				Color:            pub.Color,
				Visibility:       string(pub.Visibility),
				WelcomeMessage:   pub.WelcomeMessage,
				SuggestedPrompts: append([]string(nil), pub.SuggestedPrompts...),
				Capabilities:     agentCapabilitiesToStrings(pub.Capabilities),
				CreditCost:       pub.CreditCost,
			})
			return
		}
	}

	agent, err := agents.GetAgentByID(agentID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get agent")
		return
	}
	if agent == nil {
		respondWithError(w, http.StatusNotFound, "Agent not found")
		return
	}

	respondWithJSON(w, http.StatusOK, toPublicAgent(*agent))
}

// SendMessageRequest represents a chat message request.
type SendMessageRequest struct {
	AgentID        string   `json:"agentId"`
	ConversationID string   `json:"conversationId,omitempty"`
	Message        string   `json:"message"`
	Attachments    []string `json:"attachments,omitempty"` // base64 image data URLs for vision
}

// SendMessageResponse represents a chat message response.
type SendMessageResponse struct {
	ConversationID   string `json:"conversationId"`
	Answer           string `json:"answer"`
	CreditsCharged   int    `json:"creditsCharged"`
	RemainingCredits int    `json:"remainingCredits"`
	Model            string `json:"model"`
}

const (
	conversationTitleMaxLength           = 48
	streamProviderFailurePlaceholderText = "AI service is temporarily unavailable. Please try again."
	unsupportedChatProviderMessage       = "This model provider is not supported for chat yet. Use an OpenAI-compatible provider."
	maxChatAttachments                   = 4
)

// validImageAttachments filters attachments to valid base64 image data URLs,
// capping the count at maxChatAttachments.
func validImageAttachments(attachments []string) []string {
	if len(attachments) == 0 {
		return nil
	}
	valid := make([]string, 0, len(attachments))
	for _, a := range attachments {
		a = strings.TrimSpace(a)
		if strings.HasPrefix(a, "data:image/") && strings.Contains(a, ";base64,") {
			valid = append(valid, a)
		}
		if len(valid) >= maxChatAttachments {
			break
		}
	}
	return valid
}

// buildUserMessage returns an LLM message for the user turn, using a vision
// message when image attachments are present.
func buildUserMessage(text string, attachments []string) llm.Message {
	images := validImageAttachments(attachments)
	if len(images) > 0 {
		return llm.NewVisionMessage("user", text, images)
	}
	return llm.NewTextMessage("user", text)
}

func conversationTitleFromMessage(message string) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(message)), " ")
	if utf8.RuneCountInString(normalized) <= conversationTitleMaxLength {
		return normalized
	}

	truncatedRunes := []rune(normalized)
	return string(truncatedRunes[:conversationTitleMaxLength-3]) + "..."
}

func llmConfigUsable(config *models.LLMConfig) bool {
	return config != nil &&
		config.IsActive &&
		strings.TrimSpace(config.APIKey) != "" &&
		strings.TrimSpace(config.BaseURL) != "" &&
		strings.TrimSpace(config.Model) != ""
}

func agentCapabilitiesToStrings(capabilities []models.AgentCapability) []string {
	result := make([]string, len(capabilities))
	for i, capability := range capabilities {
		result[i] = string(capability)
	}
	return result
}

func publishedTenantAgentLookupFilter(agentID string, tenantID primitive.ObjectID) bson.M {
	filter := bson.M{
		"tenantId": tenantID,
		"status":   models.AgentStatusPublished,
	}
	if objectID, err := primitive.ObjectIDFromHex(agentID); err == nil {
		filter["$or"] = bson.A{
			bson.M{"_id": objectID},
			bson.M{"slug": agentID},
		}
		return filter
	}
	filter["slug"] = agentID
	return filter
}

type resolvedChatAgent struct {
	Agent       agents.Agent
	CanonicalID string
	Slug        string
	DBAgent     *models.Agent
}

func resolveStaticChatAgent(agent agents.Agent) resolvedChatAgent {
	return resolvedChatAgent{
		Agent:       agent,
		CanonicalID: agent.ID,
		Slug:        agent.ID,
	}
}

func resolveDBChatAgent(agent models.Agent) resolvedChatAgent {
	return resolvedChatAgent{
		Agent: agents.Agent{
			ID:           agent.ID.Hex(),
			Name:         agent.Name,
			Category:     agent.Category,
			Description:  agent.Description,
			SystemPrompt: agent.SystemPrompt,
			CreditCost:   agent.TextCreditCost(),
			Examples:     agent.SuggestedPrompts,
			Icon:         agent.Icon,
			Color:        agent.Color,
		},
		CanonicalID: agent.ID.Hex(),
		Slug:        agent.Slug,
		DBAgent:     &agent,
	}
}

func conversationMatchesAgent(conv *models.Conversation, agentID string) bool {
	return conv != nil && conv.AgentID == agentID
}

func mapCreditDeductionErrorStatus(err error) int {
	if errors.Is(err, credits.ErrInsufficientCredits) || errors.Is(err, credits.ErrBalanceChanged) {
		return http.StatusPaymentRequired
	}
	return http.StatusInternalServerError
}

// resolveChatRequestConfig resolves the LLM request config for a chat request.
// It first tries the model router for tenant-scoped agents, then falls back to legacy config.
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

func buildStreamMessageSuccessUpdate(content, usedModel string, creditCost int, usage llm.Usage) bson.M {
	return bson.M{"$set": bson.M{
		"status":           models.ChatMessageStatusCompleted,
		"content":          content,
		"creditsCharged":   creditCost,
		"model":            usedModel,
		"promptTokens":     usage.PromptTokens,
		"completionTokens": usage.CompletionTokens,
	}}
}

func buildStreamMessageDeductionFailureUpdate(content, usedModel string) bson.M {
	return bson.M{"$set": bson.M{
		"status":         models.ChatMessageStatusError,
		"content":        content,
		"creditsCharged": 0,
		"model":          usedModel,
	}}
}

func streamProviderFailureContent(content string) string {
	if strings.TrimSpace(content) == "" {
		return streamProviderFailurePlaceholderText
	}
	return content
}

// SendMessage handles sending a message to an AI expert and getting a response.
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user and tenant from context
	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	tenant, ok := middleware.GetTenantFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusBadRequest, "Tenant context required")
		return
	}

	// Parse request
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Allow image-only messages (no text required when images are attached)
	if strings.TrimSpace(req.Message) == "" && len(validImageAttachments(req.Attachments)) == 0 {
		respondWithError(w, http.StatusBadRequest, "Message is required")
		return
	}
	if req.AgentID == "" {
		respondWithError(w, http.StatusBadRequest, "Agent ID is required")
		return
	}

	// Validate agent exists
	resolvedAgent, err := h.resolvePublishedAgent(ctx, req.AgentID, tenant.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get agent")
		return
	}
	if resolvedAgent == nil {
		respondWithError(w, http.StatusNotFound, "Agent not found")
		return
	}
	agent := resolvedAgent.Agent

	// Check if user has sufficient credits before making the LLM call
	hasCredits, err := h.creditsSvc.CheckSufficientCredits(ctx, tenant.ID, agent.CreditCost)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to check credits")
		return
	}
	if !hasCredits {
		respondWithError(w, http.StatusPaymentRequired, "Insufficient credits")
		return
	}

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

	// Get or validate conversation
	var conversationID primitive.ObjectID
	var createConversation bool
	// Use a default title for image-only messages (message may be empty)
	messageForTitle := req.Message
	if strings.TrimSpace(messageForTitle) == "" {
		messageForTitle = "[Image]"
	}
	conversationTitle := conversationTitleFromMessage(messageForTitle)
	if req.ConversationID != "" {
		conversationID, err = primitive.ObjectIDFromHex(req.ConversationID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid conversation ID")
			return
		}
		var conv models.Conversation
		err = h.db.Conversations().FindOne(ctx, bson.M{
			"_id":      conversationID,
			"tenantId": tenant.ID,
			"userId":   user.ID,
		}).Decode(&conv)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "Conversation not found")
			return
		}
		if !conversationMatchesAgent(&conv, resolvedAgent.CanonicalID) {
			respondWithError(w, http.StatusBadRequest, "Conversation agent does not match request")
			return
		}
	} else {
		conversationID = primitive.NewObjectID()
		createConversation = true
	}

	messages, err := h.getMessageHistory(ctx, conversationID, tenant.ID, user.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get message history")
		return
	}
	messages = append(messages, buildUserMessage(req.Message, req.Attachments))
	messages = truncateHistory(messages, historyTokenBudget)

	// Augment the system prompt with retrieved knowledge (degrades to base prompt
	// on any failure — chat is never blocked by the knowledge layer).
	systemPrompt := h.buildSystemPrompt(ctx, *resolvedAgent, req.Message)

	// Deduct credits BEFORE calling the LLM. If anything downstream fails we
	// refund, so the user is never charged for an answer they didn't receive —
	// and an answer is never produced for free when the balance ran out between
	// the pre-check above and now.
	metadata := map[string]interface{}{
		"agentId":        resolvedAgent.CanonicalID,
		"agentSlug":      resolvedAgent.Slug,
		"conversationId": conversationID.Hex(),
	}
	remainingCredits, err := h.creditsSvc.DeductCredits(ctx, tenant.ID, user.ID, agent.CreditCost, "agent_chat", metadata)
	if err != nil {
		statusCode := mapCreditDeductionErrorStatus(err)
		if statusCode == http.StatusPaymentRequired {
			respondWithError(w, statusCode, "Insufficient credits")
			return
		}
		respondWithError(w, statusCode, "Failed to deduct credits")
		return
	}

	completion, err := h.llmClient.CompleteWithUsage(ctx, requestConfig, systemPrompt, messages)
	if err != nil {
		// Refund the charge: the work it paid for was not delivered.
		if refundErr := h.creditsSvc.RefundDeduction(ctx, tenant.ID, user.ID, agent.CreditCost, "agent_chat", metadata); refundErr != nil {
			slog.Error("Chat: refund after LLM failure failed", "tenantId", tenant.ID.Hex(), "error", refundErr)
		}
		respondWithError(w, http.StatusInternalServerError, "AI service is temporarily unavailable. Please try again.")
		return
	}
	answer := completion.Content
	usedModel := completion.Model

	now := time.Now()
	if createConversation {
		conversation := models.Conversation{
			ID:        conversationID,
			TenantID:  tenant.ID,
			UserID:    user.ID,
			AgentID:   resolvedAgent.CanonicalID,
			Title:     conversationTitle,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if _, err := h.db.Conversations().InsertOne(ctx, conversation); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create conversation")
			return
		}
	} else {
		_, err = h.db.Conversations().UpdateOne(ctx,
			bson.M{"_id": conversationID},
			bson.M{"$set": bson.M{"updatedAt": now}},
		)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to update conversation")
			return
		}
	}

	userMessage := models.ChatMessage{
		ID:             primitive.NewObjectID(),
		TenantID:       tenant.ID,
		UserID:         user.ID,
		ConversationID: conversationID,
		AgentID:        resolvedAgent.CanonicalID,
		Role:           "user",
		Content:        req.Message,
		CreditsCharged: 0,
		Model:          usedModel,
		CreatedAt:      now,
	}
	assistantMessageID := primitive.NewObjectID()
	assistantMessage := models.ChatMessage{
		ID:               assistantMessageID,
		TenantID:         tenant.ID,
		UserID:           user.ID,
		ConversationID:   conversationID,
		AgentID:          resolvedAgent.CanonicalID,
		Role:             "assistant",
		Content:          answer,
		Status:           models.ChatMessageStatusCompleted,
		CreditsCharged:   agent.CreditCost,
		Model:            usedModel,
		PromptTokens:     completion.Usage.PromptTokens,
		CompletionTokens: completion.Usage.CompletionTokens,
		CreatedAt:        now,
	}
	if _, err := h.db.ChatMessages().InsertMany(ctx, []interface{}{userMessage, assistantMessage}); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save messages")
		return
	}

	response := SendMessageResponse{
		ConversationID:   conversationID.Hex(),
		Answer:           answer,
		CreditsCharged:   agent.CreditCost,
		RemainingCredits: remainingCredits,
		Model:            usedModel,
	}

	respondWithJSON(w, http.StatusOK, response)
}

// getMessageHistory retrieves all messages for a conversation, excluding the system prompt.
func (h *ChatHandler) getMessageHistory(ctx context.Context, conversationID, tenantID, userID primitive.ObjectID) ([]llm.Message, error) {
	cursor, err := h.db.ChatMessages().Find(ctx, bson.M{
		"conversationId": conversationID,
		"tenantId":       tenantID,
		"userId":         userID,
	}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var chatMessages []models.ChatMessage
	if err := cursor.All(ctx, &chatMessages); err != nil {
		return nil, err
	}

	var messages []llm.Message
	for _, msg := range chatMessages {
		messages = append(messages, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return messages, nil
}

// historyTokenBudget bounds the estimated tokens of prior conversation turns
// sent to the LLM. Keeping recent turns within a budget prevents long
// conversations from inflating cost without bound (and from exceeding context
// limits once a knowledge block is also prepended to the system prompt).
const historyTokenBudget = 12000

// minRetainedMessages is the floor of most-recent messages always kept, so a
// single very large turn never starves the model of immediate context.
const minRetainedMessages = 2

// estimateMessageTokens approximates the token cost of a message using the same
// heuristic as the llm package (ASCII/4 + non-ASCII runes).
func estimateMessageTokens(m llm.Message) int {
	s, _ := m.Content.(string)
	ascii, nonASCII := 0, 0
	for _, r := range s {
		if r < 128 {
			ascii++
		} else {
			nonASCII++
		}
	}
	return ascii/4 + nonASCII
}

// truncateHistory keeps the most recent messages whose estimated token total
// fits within budget, always retaining at least minRetainedMessages. The
// returned slice preserves chronological order. The just-appended current user
// message is part of msgs and is always retained (it falls inside the floor).
func truncateHistory(msgs []llm.Message, budget int) []llm.Message {
	if len(msgs) <= minRetainedMessages {
		return msgs
	}
	total := 0
	keepFrom := len(msgs)
	for i := len(msgs) - 1; i >= 0; i-- {
		total += estimateMessageTokens(msgs[i])
		kept := len(msgs) - i
		if total > budget && kept > minRetainedMessages {
			break
		}
		keepFrom = i
	}
	return msgs[keepFrom:]
}

// buildSystemPrompt augments an agent's base system prompt with retrieved
// knowledge for the given query. Retrieval failures degrade gracefully to the
// base prompt — chat must never be blocked by the knowledge layer. Only DB
// agents (resolvedAgent.DBAgent != nil) have a knowledge base; the owner tenant
// is the agent's tenant (a root tenant for platform agents).
func (h *ChatHandler) buildSystemPrompt(ctx context.Context, resolvedAgent resolvedChatAgent, query string) string {
	base := resolvedAgent.Agent.SystemPrompt
	if h.knowledgeSvc == nil || resolvedAgent.DBAgent == nil {
		return base
	}

	ownerTenant, err := h.loadAgentOwnerTenant(ctx, resolvedAgent.DBAgent.TenantID)
	if err != nil {
		slog.Warn("Chat: knowledge owner tenant load failed", "agentId", resolvedAgent.CanonicalID, "error", err)
		return base
	}

	chunks, err := h.knowledgeSvc.Retrieve(ctx, ownerTenant, resolvedAgent.CanonicalID, query)
	if err != nil {
		// Degrade silently: an unconfigured embedding model or provider hiccup
		// must not break chat.
		slog.Warn("Chat: knowledge retrieval failed", "agentId", resolvedAgent.CanonicalID, "error", err)
		return base
	}
	block := knowledge.BuildKnowledgeBlock(chunks)
	if block == "" {
		return base
	}
	return base + block
}

// loadAgentOwnerTenant fetches the tenant that owns an agent (for resolving the
// embedding model and scoping knowledge retrieval).
func (h *ChatHandler) loadAgentOwnerTenant(ctx context.Context, tenantID primitive.ObjectID) (models.Tenant, error) {
	var tenant models.Tenant
	err := h.db.Tenants().FindOne(ctx, bson.M{"_id": tenantID}).Decode(&tenant)
	return tenant, err
}

// ListConversations returns all conversations for the current user/tenant.
func (h *ChatHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	tenant, ok := middleware.GetTenantFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusBadRequest, "Tenant context required")
		return
	}

	// Get optional agentId filter
	agentID := r.URL.Query().Get("agentId")

	// Build query
	query := bson.M{
		"tenantId": tenant.ID,
		"userId":   user.ID,
	}
	if agentID != "" {
		query["agentId"] = agentID
	}

	cursor, err := h.db.Conversations().Find(ctx, query, options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}}))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to list conversations")
		return
	}
	defer cursor.Close(ctx)

	var conversations []models.Conversation
	if err := cursor.All(ctx, &conversations); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to decode conversations")
		return
	}

	respondWithJSON(w, http.StatusOK, conversations)
}

// ListMessages returns all messages in a conversation.
func (h *ChatHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	tenant, ok := middleware.GetTenantFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusBadRequest, "Tenant context required")
		return
	}

	vars := mux.Vars(r)
	conversationIDStr := vars["conversationId"]

	conversationID, err := primitive.ObjectIDFromHex(conversationIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid conversation ID")
		return
	}

	// Verify the conversation belongs to this user and tenant
	var conv models.Conversation
	err = h.db.Conversations().FindOne(ctx, bson.M{
		"_id":      conversationID,
		"tenantId": tenant.ID,
		"userId":   user.ID,
	}).Decode(&conv)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Conversation not found")
		return
	}

	// Get messages for this conversation
	cursor, err := h.db.ChatMessages().Find(ctx, bson.M{
		"conversationId": conversationID,
		"tenantId":       tenant.ID,
		"userId":         user.ID,
	}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to list messages")
		return
	}
	defer cursor.Close(ctx)

	var messages []models.ChatMessage
	if err := cursor.All(ctx, &messages); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to decode messages")
		return
	}

	respondWithJSON(w, http.StatusOK, messages)
}

// getPublishedAgent retrieves a published agent from the database.
// It first checks the static catalog, then falls back to database lookup.
func (h *ChatHandler) getPublishedAgent(ctx context.Context, agentID string, tenantID primitive.ObjectID) (*agents.Agent, error) {
	resolved, err := h.resolvePublishedAgent(ctx, agentID, tenantID)
	if err != nil || resolved == nil {
		return nil, err
	}
	return &resolved.Agent, nil
}

func (h *ChatHandler) resolvePublishedAgent(ctx context.Context, agentID string, tenantID primitive.ObjectID) (*resolvedChatAgent, error) {
	agent, err := agents.GetAgentByID(agentID)
	if err != nil {
		return nil, err
	}
	if agent != nil {
		resolved := resolveStaticChatAgent(*agent)
		return &resolved, nil
	}

	if h.db == nil {
		return nil, nil
	}

	var dbAgent models.Agent
	err = h.db.Agents().FindOne(ctx, publishedTenantAgentLookupFilter(agentID, tenantID)).Decode(&dbAgent)
	if err == nil {
		resolved := resolveDBChatAgent(dbAgent)
		return &resolved, nil
	}

	var rootTenants []models.Tenant
	cursor, rootErr := h.db.Tenants().Find(ctx, bson.M{"isRoot": true, "isActive": true})
	if rootErr != nil {
		return nil, rootErr
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &rootTenants); err != nil {
		return nil, err
	}

	rootTenantIDs := make(bson.A, 0, len(rootTenants))
	for _, tenant := range rootTenants {
		rootTenantIDs = append(rootTenantIDs, tenant.ID)
	}
	if len(rootTenantIDs) == 0 {
		return nil, err
	}

	filter := publishedTenantAgentLookupFilter(agentID, tenantID)
	filter["tenantId"] = bson.M{"$in": rootTenantIDs}
	err = h.db.Agents().FindOne(ctx, filter).Decode(&dbAgent)
	if err != nil {
		return nil, err
	}
	resolved := resolveDBChatAgent(dbAgent)
	return &resolved, nil
}

// writeSSE writes a Server-Sent Event to the response.
func (h *ChatHandler) writeSSE(w http.ResponseWriter, event string, data interface{}) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal SSE data: %w", err)
	}

	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonData)
	if err != nil {
		return fmt.Errorf("failed to write SSE: %w", err)
	}

	flusher.Flush()
	return nil
}

// StreamMessage handles streaming chat messages via SSE.
func (h *ChatHandler) StreamMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Remove the server-level WriteTimeout for this SSE handler. The default
	// WriteTimeout (15 s) kills long-running streams mid-response. Streaming
	// responses can take minutes; we rely on the request context (client
	// disconnect) and the LLM's own timeout instead.
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{}) // zero value = no deadline

	// Get user and tenant from context
	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	tenant, ok := middleware.GetTenantFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusBadRequest, "Tenant context required")
		return
	}

	// Parse request
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Allow image-only messages (no text required when images are attached)
	if strings.TrimSpace(req.Message) == "" && len(validImageAttachments(req.Attachments)) == 0 {
		respondWithError(w, http.StatusBadRequest, "Message is required")
		return
	}
	if req.AgentID == "" {
		respondWithError(w, http.StatusBadRequest, "Agent ID is required")
		return
	}

	// Validate agent exists
	resolvedAgent, err := h.resolvePublishedAgent(ctx, req.AgentID, tenant.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get agent")
		return
	}
	if resolvedAgent == nil {
		respondWithError(w, http.StatusNotFound, "Agent not found")
		return
	}
	agent := resolvedAgent.Agent

	// Check if user has sufficient credits before making the LLM call
	hasCredits, err := h.creditsSvc.CheckSufficientCredits(ctx, tenant.ID, agent.CreditCost)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to check credits")
		return
	}
	if !hasCredits {
		respondWithError(w, http.StatusPaymentRequired, "Insufficient credits")
		return
	}

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

	// Get or validate conversation
	var conversationID primitive.ObjectID
	var createConversation bool
	// Use a default title for image-only messages (message may be empty)
	messageForTitle := req.Message
	if strings.TrimSpace(messageForTitle) == "" {
		messageForTitle = "[Image]"
	}
	conversationTitle := conversationTitleFromMessage(messageForTitle)
	if req.ConversationID != "" {
		conversationID, err = primitive.ObjectIDFromHex(req.ConversationID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid conversation ID")
			return
		}
		var conv models.Conversation
		err = h.db.Conversations().FindOne(ctx, bson.M{
			"_id":      conversationID,
			"tenantId": tenant.ID,
			"userId":   user.ID,
		}).Decode(&conv)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "Conversation not found")
			return
		}
		if !conversationMatchesAgent(&conv, resolvedAgent.CanonicalID) {
			respondWithError(w, http.StatusBadRequest, "Conversation agent does not match request")
			return
		}
	} else {
		conversationID = primitive.NewObjectID()
		createConversation = true
	}

	messages, err := h.getMessageHistory(ctx, conversationID, tenant.ID, user.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get message history")
		return
	}
	messages = append(messages, buildUserMessage(req.Message, req.Attachments))
	messages = truncateHistory(messages, historyTokenBudget)

	// Augment the system prompt with retrieved knowledge (degrades gracefully).
	systemPrompt := h.buildSystemPrompt(ctx, *resolvedAgent, req.Message)

	now := time.Now()

	// Create or update conversation
	if createConversation {
		conversation := models.Conversation{
			ID:        conversationID,
			TenantID:  tenant.ID,
			UserID:    user.ID,
			AgentID:   resolvedAgent.CanonicalID,
			Title:     conversationTitle,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if _, err := h.db.Conversations().InsertOne(ctx, conversation); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create conversation")
			return
		}
	} else {
		_, err = h.db.Conversations().UpdateOne(ctx,
			bson.M{"_id": conversationID},
			bson.M{"$set": bson.M{"updatedAt": now}},
		)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to update conversation")
			return
		}
	}

	// Save user message
	userMessage := models.ChatMessage{
		ID:              primitive.NewObjectID(),
		TenantID:        tenant.ID,
		UserID:          user.ID,
		ConversationID:  conversationID,
		AgentID:         resolvedAgent.CanonicalID,
		Role:            "user",
		Content:         req.Message,
		AttachmentCount: len(validImageAttachments(req.Attachments)),
		CreditsCharged: 0,
		Model:          requestConfig.Model,
		CreatedAt:      now,
	}
	if _, err := h.db.ChatMessages().InsertOne(ctx, userMessage); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save user message")
		return
	}

	// Stream the assistant reply (insert placeholder, deduct, stream, persist, SSE).
	h.streamAssistantReply(ctx, w, *tenant, *user, conversationID, *resolvedAgent, requestConfig, messages, systemPrompt)
}

// streamAssistantReply runs the shared streaming-generation tail used by both
// StreamMessage and StreamRegenerate: it inserts the assistant placeholder,
// deducts credits up front, streams the LLM response over SSE, persists the
// final message (completed/interrupted/error), and refunds on failure/cancel.
// The caller is responsible for having already created/updated the conversation
// and (for first sends) saved the user message; `messages` is the full prepared
// history (already truncated) and `systemPrompt` already carries any knowledge.
func (h *ChatHandler) streamAssistantReply(
	ctx context.Context,
	w http.ResponseWriter,
	tenant models.Tenant,
	user models.User,
	conversationID primitive.ObjectID,
	resolvedAgent resolvedChatAgent,
	requestConfig llm.RequestConfig,
	messages []llm.Message,
	systemPrompt string,
) {
	agent := resolvedAgent.Agent
	now := time.Now()

	// Create assistant message placeholder (status: generating)
	assistantMessageID := primitive.NewObjectID()
	assistantMessage := models.ChatMessage{
		ID:             assistantMessageID,
		TenantID:       tenant.ID,
		UserID:         user.ID,
		ConversationID: conversationID,
		AgentID:        resolvedAgent.CanonicalID,
		Role:           "assistant",
		Content:        "",
		Status:         models.ChatMessageStatusGenerating,
		CreditsCharged: 0,
		Model:          requestConfig.Model,
		CreatedAt:      now,
	}
	if _, err := h.db.ChatMessages().InsertOne(ctx, assistantMessage); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create assistant message")
		return
	}

	// Send message_start event
	_ = h.writeSSE(w, "message_start", map[string]string{
		"conversationId": conversationID.Hex(),
		"messageId":      assistantMessageID.Hex(),
	})

	// Track full response content
	var fullContent strings.Builder

	// Callback for each delta
	onDelta := func(content string) error {
		fullContent.WriteString(content)
		return h.writeSSE(w, "delta", map[string]string{
			"text": content,
		})
	}

	// Deduct credits BEFORE streaming. The pre-check earlier is advisory only;
	// deducting up front closes the race where a concurrent request drains the
	// balance mid-stream and the user receives a free answer. On any failure
	// below (cancel / provider error) we refund.
	metadata := map[string]interface{}{
		"agentId":        resolvedAgent.CanonicalID,
		"agentSlug":      resolvedAgent.Slug,
		"conversationId": conversationID.Hex(),
		"model":          requestConfig.Model,
	}
	remainingCredits, err := h.creditsSvc.DeductCredits(ctx, tenant.ID, user.ID, agent.CreditCost, "agent_chat", metadata)
	if err != nil {
		dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dbCancel()
		_, _ = h.db.ChatMessages().UpdateOne(dbCtx,
			bson.M{"_id": assistantMessageID},
			buildStreamMessageDeductionFailureUpdate("", requestConfig.Model),
		)
		statusMsg := "Failed to process payment. Please contact support."
		if errors.Is(err, credits.ErrInsufficientCredits) {
			statusMsg = "Insufficient credits."
		}
		_ = h.writeSSE(w, "error", map[string]string{"message": statusMsg})
		return
	}

	// refundStream reverses the charge taken above when the answer is not delivered.
	refundStream := func(dbCtx context.Context) {
		if refundErr := h.creditsSvc.RefundDeduction(dbCtx, tenant.ID, user.ID, agent.CreditCost, "agent_chat", metadata); refundErr != nil {
			slog.Error("Chat: stream refund failed", "tenantId", tenant.ID.Hex(), "error", refundErr)
		}
	}

	// Call the streaming LLM synchronously on this goroutine.
	// All writes to w (delta events) happen here — no concurrent writer.
	completion, err := h.llmClient.CompleteStreamWithUsage(ctx, requestConfig, systemPrompt, messages, onDelta)
	usedModel := completion.Model
	if usedModel == "" {
		usedModel = requestConfig.Model
	}

	// Use a fresh background context for all DB writes that happen after the
	// request context may have been cancelled (client disconnected).
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()

	// Check if request was cancelled (client disconnected)
	if ctx.Err() == context.Canceled {
		refundStream(dbCtx)
		// Update message status to interrupted using dbCtx (req ctx already done)
		_, _ = h.db.ChatMessages().UpdateOne(dbCtx,
			bson.M{"_id": assistantMessageID},
			bson.M{"$set": bson.M{
				"status":  models.ChatMessageStatusInterrupted,
				"content": fullContent.String(),
				"model":   usedModel,
			}},
		)

		// Send error event (credits refunded)
		_ = h.writeSSE(w, "error", map[string]string{
			"message": "Request was cancelled. No credits were charged.",
		})
		return
	}

	// Handle provider errors
	if err != nil && err != io.EOF {
		refundStream(dbCtx)
		// Update message status to error
		_, _ = h.db.ChatMessages().UpdateOne(dbCtx,
			bson.M{"_id": assistantMessageID},
			bson.M{"$set": bson.M{
				"status":         models.ChatMessageStatusError,
				"content":        streamProviderFailureContent(fullContent.String()),
				"creditsCharged": 0,
				"model":          usedModel,
			}},
		)

		// Send error event (credits refunded)
		_ = h.writeSSE(w, "error", map[string]string{
			"message": streamProviderFailurePlaceholderText,
		})
		return
	}

	// Stream completed successfully. Credits were already deducted up front;
	// mark the assistant message completed.
	_, _ = h.db.ChatMessages().UpdateOne(dbCtx,
		bson.M{"_id": assistantMessageID},
		buildStreamMessageSuccessUpdate(fullContent.String(), usedModel, agent.CreditCost, completion.Usage),
	)

	// Send message_done event with credits info
	_ = h.writeSSE(w, "message_done", map[string]interface{}{
		"conversationId":   conversationID.Hex(),
		"messageId":        assistantMessageID.Hex(),
		"creditsCharged":   agent.CreditCost,
		"remainingCredits": remainingCredits,
		"model":            usedModel,
	})
}

// GenerateImage handles image generation requests (coming soon).
func (h *ChatHandler) GenerateImage(w http.ResponseWriter, r *http.Request) {
	respondWithError(w, http.StatusNotImplemented, "Image generation is coming soon and no credits were charged.")
}

// GenerateVideo handles video generation requests (coming soon).
func (h *ChatHandler) GenerateVideo(w http.ResponseWriter, r *http.Request) {
	respondWithError(w, http.StatusNotImplemented, "Video generation is coming soon and no credits were charged.")
}

// DeleteConversation deletes a conversation and cascades to its messages,
// feedback, and annotations. Scoped to the requesting tenant+user.
func (h *ChatHandler) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	tenant, ok := middleware.GetTenantFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusBadRequest, "Tenant context required")
		return
	}

	conversationID, err := primitive.ObjectIDFromHex(mux.Vars(r)["conversationId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid conversation ID")
		return
	}

	// Verify ownership before deleting anything.
	result, err := h.db.Conversations().DeleteOne(ctx, bson.M{
		"_id":      conversationID,
		"tenantId": tenant.ID,
		"userId":   user.ID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete conversation")
		return
	}
	if result.DeletedCount == 0 {
		respondWithError(w, http.StatusNotFound, "Conversation not found")
		return
	}

	// Cascade: messages, feedback, and annotations for this conversation. All
	// three collections carry conversationId; scope by tenant for safety.
	if _, err := h.db.ChatMessages().DeleteMany(ctx, bson.M{"conversationId": conversationID, "tenantId": tenant.ID}); err != nil {
		slog.Error("Chat: failed to delete conversation messages", "conversationId", conversationID.Hex(), "error", err)
	}
	if _, err := h.db.MessageFeedback().DeleteMany(ctx, bson.M{"conversationId": conversationID, "tenantId": tenant.ID}); err != nil {
		slog.Error("Chat: failed to delete conversation feedback", "conversationId", conversationID.Hex(), "error", err)
	}
	if _, err := h.db.Annotations().DeleteMany(ctx, bson.M{"conversationId": conversationID, "tenantId": tenant.ID}); err != nil {
		slog.Error("Chat: failed to delete conversation annotations", "conversationId", conversationID.Hex(), "error", err)
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// RenameConversation updates a conversation's title (tenant+user scoped).
func (h *ChatHandler) RenameConversation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	tenant, ok := middleware.GetTenantFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusBadRequest, "Tenant context required")
		return
	}

	conversationID, err := primitive.ObjectIDFromHex(mux.Vars(r)["conversationId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid conversation ID")
		return
	}

	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	title := strings.TrimSpace(req.Title)

	// Validate against the same rules as the model (required, 1..200).
	candidate := models.Conversation{
		ID:        conversationID,
		TenantID:  tenant.ID,
		UserID:    user.ID,
		AgentID:   "placeholder", // not persisted; satisfies struct validation only
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := validation.Validate(&candidate); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.db.Conversations().UpdateOne(ctx,
		bson.M{"_id": conversationID, "tenantId": tenant.ID, "userId": user.ID},
		bson.M{"$set": bson.M{"title": title, "updatedAt": time.Now()}},
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to rename conversation")
		return
	}
	if result.MatchedCount == 0 {
		respondWithError(w, http.StatusNotFound, "Conversation not found")
		return
	}

	var updated models.Conversation
	if err := h.db.Conversations().FindOne(ctx, bson.M{"_id": conversationID}).Decode(&updated); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to load conversation")
		return
	}
	respondWithJSON(w, http.StatusOK, updated)
}

// RegenerateRequest is the body for StreamRegenerate.
type RegenerateRequest struct {
	ConversationID string `json:"conversationId"`
	MessageID      string `json:"messageId"`
}

// precedingUserMessage returns the latest user message strictly before the given
// assistant message in the same conversation (the prompt that produced it).
func (h *ChatHandler) precedingUserMessage(ctx context.Context, conversationID, tenantID, userID primitive.ObjectID, before time.Time) (*models.ChatMessage, error) {
	var msg models.ChatMessage
	err := h.db.ChatMessages().FindOne(ctx,
		bson.M{
			"conversationId": conversationID,
			"tenantId":       tenantID,
			"userId":         userID,
			"role":           "user",
			"createdAt":      bson.M{"$lt": before},
		},
		options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
	).Decode(&msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// StreamRegenerate regenerates an assistant reply in place: it removes the target
// assistant message and streams a fresh reply for the same preceding user turn,
// reusing the shared streaming tail (credits, knowledge injection, persistence).
func (h *ChatHandler) StreamRegenerate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// SSE: drop the server write deadline (see StreamMessage).
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	tenant, ok := middleware.GetTenantFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusBadRequest, "Tenant context required")
		return
	}

	var req RegenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	conversationID, err := primitive.ObjectIDFromHex(req.ConversationID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid conversation ID")
		return
	}
	targetID, err := primitive.ObjectIDFromHex(req.MessageID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	// Verify the conversation belongs to this user+tenant.
	var conv models.Conversation
	if err := h.db.Conversations().FindOne(ctx, bson.M{
		"_id":      conversationID,
		"tenantId": tenant.ID,
		"userId":   user.ID,
	}).Decode(&conv); err != nil {
		respondWithError(w, http.StatusNotFound, "Conversation not found")
		return
	}

	// Load the target assistant message.
	var target models.ChatMessage
	if err := h.db.ChatMessages().FindOne(ctx, bson.M{
		"_id":            targetID,
		"conversationId": conversationID,
		"tenantId":       tenant.ID,
		"userId":         user.ID,
	}).Decode(&target); err != nil {
		respondWithError(w, http.StatusNotFound, "Message not found")
		return
	}
	if target.Role != "assistant" {
		respondWithError(w, http.StatusBadRequest, "Only assistant messages can be regenerated")
		return
	}

	// Find the user turn that produced it.
	userMsg, err := h.precedingUserMessage(ctx, conversationID, tenant.ID, user.ID, target.CreatedAt)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "No preceding user message to regenerate from")
		return
	}

	// Resolve the agent + request config (same as StreamMessage).
	resolvedAgent, err := h.resolvePublishedAgent(ctx, conv.AgentID, tenant.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get agent")
		return
	}
	if resolvedAgent == nil {
		respondWithError(w, http.StatusNotFound, "Agent not found")
		return
	}
	agent := resolvedAgent.Agent

	hasCredits, err := h.creditsSvc.CheckSufficientCredits(ctx, tenant.ID, agent.CreditCost)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to check credits")
		return
	}
	if !hasCredits {
		respondWithError(w, http.StatusPaymentRequired, "Insufficient credits")
		return
	}

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

	// Remove the old assistant reply so it is replaced (not appended to).
	if _, err := h.db.ChatMessages().DeleteOne(ctx, bson.M{"_id": targetID, "tenantId": tenant.ID}); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to clear previous reply")
		return
	}

	// Build history (now excluding the deleted reply) + knowledge-augmented prompt.
	messages, err := h.getMessageHistory(ctx, conversationID, tenant.ID, user.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get message history")
		return
	}
	messages = truncateHistory(messages, historyTokenBudget)
	systemPrompt := h.buildSystemPrompt(ctx, *resolvedAgent, userMsg.Content)

	// Touch the conversation's updatedAt so it bubbles up the list.
	_, _ = h.db.Conversations().UpdateOne(ctx,
		bson.M{"_id": conversationID},
		bson.M{"$set": bson.M{"updatedAt": time.Now()}},
	)

	h.streamAssistantReply(ctx, w, *tenant, *user, conversationID, *resolvedAgent, requestConfig, messages, systemPrompt)
}
