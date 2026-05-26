package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"lastsaas/internal/agents"
	"lastsaas/internal/credits"
	"lastsaas/internal/db"
	"lastsaas/internal/llm"
	"lastsaas/internal/middleware"
	"lastsaas/internal/models"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ChatHandler handles AI chat endpoints.
type ChatHandler struct {
	db          *db.MongoDB
	creditsSvc  *credits.Service
	llmClient   *llm.Client
	modelRouter *llm.Router
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
		Examples: append([]string(nil), agent.Examples...),
		Icon:     agent.Icon,
		Color:    agent.Color,
	}
}

// NewChatHandler creates a new chat handler.
func NewChatHandler(database *db.MongoDB, creditsSvc *credits.Service, llmClient *llm.Client, modelRouter *llm.Router) *ChatHandler {
	return &ChatHandler{
		db:          database,
		creditsSvc:  creditsSvc,
		llmClient:   llmClient,
		modelRouter: modelRouter,
	}
}

// ListAgents returns available AI experts (DB agents first, then fallback to static catalog).
func (h *ChatHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	// Try to get published agents from database
	tenant, hasTenant := middleware.GetTenantFromContext(r.Context())
	var publicAgents []publicAgent

	if hasTenant {
		// Query published agents from DB
		cursor, err := h.db.Agents().Find(r.Context(), bson.M{"tenantId": tenant.ID, "status": models.AgentStatusPublished})
		if err == nil {
			var dbAgents []models.Agent
			if err := cursor.All(r.Context(), &dbAgents); err == nil && len(dbAgents) > 0 {
				publicAgents = make([]publicAgent, len(dbAgents))
				for i, agent := range dbAgents {
					pub := agent.ToPublic()
					publicAgents[i] = publicAgent{
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
				respondWithJSON(w, http.StatusOK, publicAgents)
				return
			}
		}
	}

	// Fallback to static catalog
	agentList := agents.GetAllAgents()
	publicAgents = make([]publicAgent, len(agentList))
	for i, agent := range agentList {
		publicAgents[i] = toPublicAgent(agent)
	}
	respondWithJSON(w, http.StatusOK, publicAgents)
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
	AgentID        string `json:"agentId"`
	ConversationID string `json:"conversationId,omitempty"`
	Message        string `json:"message"`
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
)

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

func conversationMatchesAgent(conv *models.Conversation, agentID string) bool {
	return conv != nil && conv.AgentID == agentID
}

func mapCreditDeductionErrorStatus(err error) int {
	if errors.Is(err, credits.ErrInsufficientCredits) || errors.Is(err, credits.ErrBalanceChanged) {
		return http.StatusPaymentRequired
	}
	return http.StatusInternalServerError
}

func buildSendMessageDeductionFailureUpdate(answer string) bson.M {
	return bson.M{"$set": bson.M{
		"status":         models.ChatMessageStatusError,
		"content":        answer,
		"creditsCharged": 0,
	}}
}

func buildStreamMessageSuccessUpdate(content, usedModel string, creditCost int) bson.M {
	return bson.M{"$set": bson.M{
		"status":         models.ChatMessageStatusCompleted,
		"content":        content,
		"creditsCharged": creditCost,
		"model":          usedModel,
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

	if strings.TrimSpace(req.Message) == "" {
		respondWithError(w, http.StatusBadRequest, "Message is required")
		return
	}
	if req.AgentID == "" {
		respondWithError(w, http.StatusBadRequest, "Agent ID is required")
		return
	}

	// Validate agent exists
	agent, err := h.getPublishedAgent(ctx, req.AgentID, tenant.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get agent")
		return
	}
	if agent == nil {
		respondWithError(w, http.StatusNotFound, "Agent not found")
		return
	}

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

	// Get or validate conversation
	var conversationID primitive.ObjectID
	var createConversation bool
	conversationTitle := conversationTitleFromMessage(req.Message)
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
		if !conversationMatchesAgent(&conv, req.AgentID) {
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
	messages = append(messages, llm.Message{Role: "user", Content: req.Message})

	answer, usedModel, err := h.llmClient.CompleteWithConfig(ctx, requestConfig, agent.SystemPrompt, messages)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "AI service is temporarily unavailable. Please try again.")
		return
	}

	now := time.Now()
	if createConversation {
		conversation := models.Conversation{
			ID:        conversationID,
			TenantID:  tenant.ID,
			UserID:    user.ID,
			AgentID:   req.AgentID,
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
		AgentID:        req.AgentID,
		Role:           "user",
		Content:        req.Message,
		CreditsCharged: 0,
		Model:          usedModel,
		CreatedAt:      now,
	}
	assistantMessageID := primitive.NewObjectID()
	assistantMessage := models.ChatMessage{
		ID:             assistantMessageID,
		TenantID:       tenant.ID,
		UserID:         user.ID,
		ConversationID: conversationID,
		AgentID:        req.AgentID,
		Role:           "assistant",
		Content:        answer,
		Status:         models.ChatMessageStatusCompleted,
		CreditsCharged: agent.CreditCost,
		Model:          usedModel,
		CreatedAt:      now,
	}
	if _, err := h.db.ChatMessages().InsertMany(ctx, []interface{}{userMessage, assistantMessage}); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save messages")
		return
	}

	metadata := map[string]interface{}{
		"agentId":        req.AgentID,
		"conversationId": conversationID.Hex(),
		"model":          usedModel,
	}
	remainingCredits, err := h.creditsSvc.DeductCredits(ctx, tenant.ID, user.ID, agent.CreditCost, "agent_chat", metadata)
	if err != nil {
		_, _ = h.db.ChatMessages().UpdateOne(ctx, bson.M{"_id": assistantMessageID}, buildSendMessageDeductionFailureUpdate(answer))
		statusCode := mapCreditDeductionErrorStatus(err)
		if statusCode == http.StatusPaymentRequired {
			respondWithError(w, statusCode, "Insufficient credits")
			return
		}
		respondWithError(w, statusCode, "Failed to deduct credits")
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
	// First, check the static catalog
	agent, err := agents.GetAgentByID(agentID)
	if err != nil {
		return nil, err
	}
	if agent != nil {
		return agent, nil
	}

	// If not in catalog, try to look up in the database as a tenant-scoped agent
	var dbAgent models.Agent
	err = h.db.Agents().FindOne(ctx, publishedTenantAgentLookupFilter(agentID, tenantID)).Decode(&dbAgent)
	if err != nil {
		return nil, err
	}

	// Convert database agent to catalog agent format
	return &agents.Agent{
		ID:           dbAgent.ID.Hex(),
		Name:         dbAgent.Name,
		Category:     dbAgent.Category,
		Description:  dbAgent.Description,
		SystemPrompt: dbAgent.SystemPrompt,
		CreditCost:   dbAgent.TextCreditCost(),
		Examples:     dbAgent.SuggestedPrompts,
		Icon:         dbAgent.Icon,
		Color:        dbAgent.Color,
	}, nil
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

	if strings.TrimSpace(req.Message) == "" {
		respondWithError(w, http.StatusBadRequest, "Message is required")
		return
	}
	if req.AgentID == "" {
		respondWithError(w, http.StatusBadRequest, "Agent ID is required")
		return
	}

	// Validate agent exists
	agent, err := h.getPublishedAgent(ctx, req.AgentID, tenant.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get agent")
		return
	}
	if agent == nil {
		respondWithError(w, http.StatusNotFound, "Agent not found")
		return
	}

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

	// Get LLM config using the model router
	var requestConfig llm.RequestConfig
	if h.modelRouter != nil {
		// Try to find database agent by string ID or ObjectID/slug
		var dbAgent models.Agent
		err = h.db.Agents().FindOne(ctx, publishedTenantAgentLookupFilter(req.AgentID, tenant.ID)).Decode(&dbAgent)
		if err == nil {
			// Use the router for database agents
			cfg, err := h.modelRouter.ResolveTextModel(ctx, *tenant, dbAgent)
			if err == nil {
				requestConfig = cfg
			}
		}
	}

	// Fallback to LLM config from database
	if requestConfig.APIKey == "" {
		h.llmClient.ReloadConfig()
		requestConfig = h.llmClient.SnapshotConfig()
	}

	if !llmConfigUsable(&models.LLMConfig{
		APIKey:   requestConfig.APIKey,
		BaseURL:  requestConfig.BaseURL,
		Model:    requestConfig.Model,
		IsActive: true,
	}) {
		respondWithError(w, http.StatusInternalServerError, "AI model is not configured. Please contact an administrator.")
		return
	}

	// Get or validate conversation
	var conversationID primitive.ObjectID
	var createConversation bool
	conversationTitle := conversationTitleFromMessage(req.Message)
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
		if !conversationMatchesAgent(&conv, req.AgentID) {
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
	messages = append(messages, llm.Message{Role: "user", Content: req.Message})

	now := time.Now()

	// Create or update conversation
	if createConversation {
		conversation := models.Conversation{
			ID:        conversationID,
			TenantID:  tenant.ID,
			UserID:    user.ID,
			AgentID:   req.AgentID,
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
		ID:             primitive.NewObjectID(),
		TenantID:       tenant.ID,
		UserID:         user.ID,
		ConversationID: conversationID,
		AgentID:        req.AgentID,
		Role:           "user",
		Content:        req.Message,
		CreditsCharged: 0,
		Model:          requestConfig.Model,
		CreatedAt:      now,
	}
	if _, err := h.db.ChatMessages().InsertOne(ctx, userMessage); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save user message")
		return
	}

	// Create assistant message placeholder (status: generating)
	assistantMessageID := primitive.NewObjectID()
	assistantMessage := models.ChatMessage{
		ID:             assistantMessageID,
		TenantID:       tenant.ID,
		UserID:         user.ID,
		ConversationID: conversationID,
		AgentID:        req.AgentID,
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
		// Send delta event
		return h.writeSSE(w, "delta", map[string]string{
			"text": content,
		})
	}

	// Call the streaming LLM
	_, usedModel, err := h.llmClient.CompleteStreamWithConfig(ctx, requestConfig, agent.SystemPrompt, messages, onDelta)

	// Check if request was cancelled (client disconnected)
	if ctx.Err() == context.Canceled {
		// Update message status to interrupted
		_, _ = h.db.ChatMessages().UpdateOne(ctx,
			bson.M{"_id": assistantMessageID},
			bson.M{"$set": bson.M{
				"status":  models.ChatMessageStatusInterrupted,
				"content": fullContent.String(),
				"model":   usedModel,
			}},
		)

		// Send error event (no credits charged)
		_ = h.writeSSE(w, "error", map[string]string{
			"message": "Request was cancelled. No credits were charged.",
		})
		return
	}

	// Handle provider errors
	if err != nil && err != io.EOF {
		// Update message status to error
		_, _ = h.db.ChatMessages().UpdateOne(ctx,
			bson.M{"_id": assistantMessageID},
			bson.M{"$set": bson.M{
				"status":         models.ChatMessageStatusError,
				"content":        streamProviderFailureContent(fullContent.String()),
				"creditsCharged": 0,
				"model":          usedModel,
			}},
		)

		// Send error event (no credits charged)
		_ = h.writeSSE(w, "error", map[string]string{
			"message": streamProviderFailurePlaceholderText,
		})
		return
	}

	// Deduct credits before marking the assistant message as completed.
	metadata := map[string]interface{}{
		"agentId":        req.AgentID,
		"conversationId": conversationID.Hex(),
		"model":          usedModel,
	}
	remainingCredits, err := h.creditsSvc.DeductCredits(ctx, tenant.ID, user.ID, agent.CreditCost, "agent_chat", metadata)
	if err != nil {
		_, _ = h.db.ChatMessages().UpdateOne(ctx,
			bson.M{"_id": assistantMessageID},
			buildStreamMessageDeductionFailureUpdate(fullContent.String(), usedModel),
		)
		_ = h.writeSSE(w, "error", map[string]string{
			"message": "Failed to process payment. Please contact support.",
		})
		return
	}

	// Update assistant message with full content and completed status
	_, _ = h.db.ChatMessages().UpdateOne(ctx,
		bson.M{"_id": assistantMessageID},
		buildStreamMessageSuccessUpdate(fullContent.String(), usedModel, agent.CreditCost),
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
