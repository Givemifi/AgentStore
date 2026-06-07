package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
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

// AgentHandler handles tenant-scoped agent CRUD endpoints.
type AgentHandler struct {
	db *db.MongoDB
}

// NewAgentHandler creates a new agent handler.
func NewAgentHandler(database *db.MongoDB) *AgentHandler {
	return &AgentHandler{db: database}
}

// ListAgents returns all agents for the tenant, with optional status filter.
func (h *AgentHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	filter := bson.M{"tenantId": tenant.ID}
	if status := r.URL.Query().Get("status"); status != "" {
		filter["status"] = status
	}

	opts := options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}})
	cursor, err := h.db.Agents().Find(r.Context(), filter, opts)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to list agents")
		return
	}
	defer cursor.Close(r.Context())

	var agents []models.Agent
	if err := cursor.All(r.Context(), &agents); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to decode agents")
		return
	}
	if agents == nil {
		agents = []models.Agent{}
	}
	respondWithJSON(w, http.StatusOK, agents)
}

// GetAgent returns a single agent by ID (tenant-scoped).
func (h *AgentHandler) GetAgent(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	agentID, err := primitive.ObjectIDFromHex(mux.Vars(r)["agentId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid agent ID")
		return
	}

	var agent models.Agent
	err = h.db.Agents().FindOne(r.Context(), bson.M{"_id": agentID, "tenantId": tenant.ID}).Decode(&agent)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			respondWithError(w, http.StatusNotFound, "agent not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to get agent")
		return
	}
	respondWithJSON(w, http.StatusOK, agent)
}

// CreateAgent creates a new agent in the tenant.
func (h *AgentHandler) CreateAgent(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req struct {
		Name             string                  `json:"name"`
		Slug             string                  `json:"slug"`
		Category         string                  `json:"category"`
		Description      string                  `json:"description"`
		Avatar           string                  `json:"avatar"`
		Icon             string                  `json:"icon"`
		Color            string                  `json:"color"`
		Status           string                  `json:"status"`
		Visibility       string                  `json:"visibility"`
		SystemPrompt     string                  `json:"systemPrompt"`
		WelcomeMessage   string                  `json:"welcomeMessage"`
		SuggestedPrompts []string                `json:"suggestedPrompts"`
		Capabilities     []string                `json:"capabilities"`
		CreditCost       models.AgentCreditCost  `json:"creditCost"`
		ModelConfig      models.AgentModelConfig `json:"modelConfig"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	now := time.Now()
	// Default status to draft if not provided
	status := models.AgentStatus(req.Status)
	if status == "" {
		status = models.AgentStatusDraft
	}
	// Default visibility to private if not provided
	visibility := models.AgentVisibility(req.Visibility)
	if visibility == "" {
		visibility = models.AgentVisibilityPrivate
	}
	// Default capabilities if not provided
	capabilities := make([]models.AgentCapability, len(req.Capabilities))
	if len(req.Capabilities) == 0 {
		capabilities = []models.AgentCapability{models.AgentCapabilityTextChat}
	} else {
		for i, c := range req.Capabilities {
			capabilities[i] = models.AgentCapability(c)
		}
	}
	// Default credit cost if not provided
	creditCost := req.CreditCost
	if creditCost.TextMessageCredits == 0 && creditCost.ImageGenerationCredits == 0 && creditCost.VideoGenerationCredits == 0 {
		creditCost = models.AgentCreditCost{TextMessageCredits: 1}
	}

	// Ensure SuggestedPrompts is not nil (MongoDB schema requires array)
	suggestedPrompts := req.SuggestedPrompts
	if suggestedPrompts == nil {
		suggestedPrompts = []string{}
	}

	agent := models.Agent{
		TenantID:         tenant.ID,
		Name:             req.Name,
		Slug:             req.Slug,
		Category:         req.Category,
		Description:      req.Description,
		Avatar:           req.Avatar,
		Icon:             req.Icon,
		Color:            req.Color,
		Status:           status,
		Visibility:       visibility,
		SystemPrompt:     req.SystemPrompt,
		WelcomeMessage:   req.WelcomeMessage,
		SuggestedPrompts: suggestedPrompts,
		Capabilities:     capabilities,
		CreditCost:       creditCost,
		ModelConfig:      req.ModelConfig,
		CreatedBy:        user.ID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := validation.Validate(&agent); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.db.Agents().InsertOne(r.Context(), agent)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			respondWithError(w, http.StatusConflict, "agent slug already exists")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to create agent")
		return
	}

	agent.ID = result.InsertedID.(primitive.ObjectID)
	respondWithJSON(w, http.StatusCreated, agent)
}

// UpdateAgent updates an existing agent (tenant-scoped).
func (h *AgentHandler) UpdateAgent(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	agentID, err := primitive.ObjectIDFromHex(mux.Vars(r)["agentId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid agent ID")
		return
	}

	var req struct {
		Name             *string                  `json:"name"`
		Slug             *string                  `json:"slug"`
		Category         *string                  `json:"category"`
		Description      *string                  `json:"description"`
		Avatar           *string                  `json:"avatar"`
		Icon             *string                  `json:"icon"`
		Color            *string                  `json:"color"`
		Status           *string                  `json:"status"`
		Visibility       *string                  `json:"visibility"`
		SystemPrompt     *string                  `json:"systemPrompt"`
		WelcomeMessage   *string                  `json:"welcomeMessage"`
		SuggestedPrompts *[]string                `json:"suggestedPrompts"`
		Capabilities     *[]string                `json:"capabilities"`
		CreditCost       *models.AgentCreditCost  `json:"creditCost"`
		ModelConfig      *models.AgentModelConfig `json:"modelConfig"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	update := bson.M{"updatedAt": time.Now()}
	if req.Name != nil {
		update["name"] = *req.Name
	}
	if req.Slug != nil {
		update["slug"] = *req.Slug
	}
	if req.Category != nil {
		update["category"] = *req.Category
	}
	if req.Description != nil {
		update["description"] = *req.Description
	}
	if req.Avatar != nil {
		update["avatar"] = *req.Avatar
	}
	if req.Icon != nil {
		update["icon"] = *req.Icon
	}
	if req.Color != nil {
		update["color"] = *req.Color
	}
	if req.Status != nil {
		update["status"] = *req.Status
	}
	if req.Visibility != nil {
		update["visibility"] = *req.Visibility
	}
	if req.SystemPrompt != nil {
		update["systemPrompt"] = *req.SystemPrompt
	}
	if req.WelcomeMessage != nil {
		update["welcomeMessage"] = *req.WelcomeMessage
	}
	if req.SuggestedPrompts != nil {
		update["suggestedPrompts"] = *req.SuggestedPrompts
	}
	if req.Capabilities != nil {
		caps := make([]models.AgentCapability, len(*req.Capabilities))
		for i, c := range *req.Capabilities {
			caps[i] = models.AgentCapability(c)
		}
		update["capabilities"] = caps
	}
	if req.CreditCost != nil {
		update["creditCost"] = *req.CreditCost
	}
	if req.ModelConfig != nil {
		update["modelConfig"] = *req.ModelConfig
	}

	result, err := h.db.Agents().UpdateOne(
		r.Context(),
		bson.M{"_id": agentID, "tenantId": tenant.ID},
		bson.M{"$set": update},
	)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			respondWithError(w, http.StatusConflict, "agent slug already exists")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to update agent")
		return
	}
	if result.MatchedCount == 0 {
		respondWithError(w, http.StatusNotFound, "agent not found")
		return
	}

	// Return updated agent
	var updated models.Agent
	h.db.Agents().FindOne(r.Context(), bson.M{"_id": agentID}).Decode(&updated)
	respondWithJSON(w, http.StatusOK, updated)
}

// DeleteAgent soft-deletes an agent by setting status to "archived" (tenant-scoped).
func (h *AgentHandler) DeleteAgent(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	agentID, err := primitive.ObjectIDFromHex(mux.Vars(r)["agentId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid agent ID")
		return
	}

	result, err := h.db.Agents().UpdateOne(
		r.Context(),
		bson.M{"_id": agentID, "tenantId": tenant.ID, "status": bson.M{"$ne": models.AgentStatusArchived}},
		bson.M{"$set": bson.M{"status": models.AgentStatusArchived, "updatedAt": time.Now()}},
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to delete agent")
		return
	}
	if result.MatchedCount == 0 {
		respondWithError(w, http.StatusNotFound, "agent not found")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "archived"})
}

// PublishAgent sets agent status to "published" (tenant-scoped).
func (h *AgentHandler) PublishAgent(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	agentID, err := primitive.ObjectIDFromHex(mux.Vars(r)["agentId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid agent ID")
		return
	}

	result, err := h.db.Agents().UpdateOne(
		r.Context(),
		bson.M{"_id": agentID, "tenantId": tenant.ID},
		bson.M{"$set": bson.M{"status": models.AgentStatusPublished, "updatedAt": time.Now()}},
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to publish agent")
		return
	}
	if result.MatchedCount == 0 {
		respondWithError(w, http.StatusNotFound, "agent not found")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "published"})
}

// ArchiveAgent sets agent status to "archived" (tenant-scoped).
func (h *AgentHandler) ArchiveAgent(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	agentID, err := primitive.ObjectIDFromHex(mux.Vars(r)["agentId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid agent ID")
		return
	}

	result, err := h.db.Agents().UpdateOne(
		r.Context(),
		bson.M{"_id": agentID, "tenantId": tenant.ID},
		bson.M{"$set": bson.M{"status": models.AgentStatusArchived, "updatedAt": time.Now()}},
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to archive agent")
		return
	}
	if result.MatchedCount == 0 {
		respondWithError(w, http.StatusNotFound, "agent not found")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "archived"})
}