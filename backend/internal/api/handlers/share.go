package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"agentstore/internal/db"
	"agentstore/internal/middleware"
	"agentstore/internal/models"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ShareHandler struct {
	db *db.MongoDB
}

func NewShareHandler(database *db.MongoDB) *ShareHandler {
	return &ShareHandler{db: database}
}

type createShareResponse struct {
	Token     string    `json:"token"`
	ShareURL  string    `json:"shareUrl"`
	CreatedAt time.Time `json:"createdAt"`
}

type publicShareResponse struct {
	Token     string                 `json:"token"`
	AgentID   string                 `json:"agentId"`
	AgentName string                 `json:"agentName"`
	Title     string                 `json:"title"`
	Messages  []models.SharedMessage `json:"messages"`
	CreatedAt time.Time              `json:"createdAt"`
}

// CreateShare creates a public share snapshot for a conversation (requires auth).
func (h *ShareHandler) CreateShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	conversationID := mux.Vars(r)["conversationId"]
	if conversationID == "" {
		respondWithError(w, http.StatusBadRequest, "conversationId required")
		return
	}

	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tenant, ok := middleware.GetTenantFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	convObjID, err := primitive.ObjectIDFromHex(conversationID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid conversationId")
		return
	}

	// Load conversation — must belong to this user + tenant
	var conv models.Conversation
	if err := h.db.Conversations().FindOne(ctx, bson.M{
		"_id":      convObjID,
		"userId":   user.ID,
		"tenantId": tenant.ID,
	}).Decode(&conv); err != nil {
		if err == mongo.ErrNoDocuments {
			respondWithError(w, http.StatusNotFound, "conversation not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to load conversation")
		return
	}

	// Load completed messages (exclude generating/error), strip to safe fields
	cursor, err := h.db.ChatMessages().Find(ctx, bson.M{
		"conversationId": convObjID,
		"status":         bson.M{"$in": bson.A{"completed", "interrupted", ""}},
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to load messages")
		return
	}
	defer cursor.Close(ctx)

	var messages []models.SharedMessage
	for cursor.Next(ctx) {
		var msg models.ChatMessage
		if err := cursor.Decode(&msg); err != nil {
			continue
		}
		if msg.Content == "" {
			continue
		}
		messages = append(messages, models.SharedMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Resolve agent name
	agentName := conv.AgentID
	// Try DB agents first
	var dbAgent models.Agent
	if err := h.db.Agents().FindOne(ctx, bson.M{"slug": conv.AgentID, "tenantId": tenant.ID}).Decode(&dbAgent); err == nil {
		agentName = dbAgent.Name
	}

	// Generate cryptographically random token
	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)

	now := time.Now().UTC()
	share := models.ShareLink{
		Token:     token,
		TenantID:  tenant.ID,
		UserID:    user.ID,
		AgentID:   conv.AgentID,
		AgentName: agentName,
		Title:     conv.Title,
		Messages:  messages,
		CreatedAt: now,
	}

	if _, err := h.db.ShareLinks().InsertOne(ctx, share); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to create share")
		return
	}

	respondWithJSON(w, http.StatusCreated, createShareResponse{
		Token:     token,
		CreatedAt: now,
	})
}

// RevokeShare deletes a share link (requires auth, must be owner).
func (h *ShareHandler) RevokeShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := mux.Vars(r)["token"]

	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	res, err := h.db.ShareLinks().DeleteOne(ctx, bson.M{"token": token, "userId": user.ID})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to delete share")
		return
	}
	if res.DeletedCount == 0 {
		respondWithError(w, http.StatusNotFound, "share not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListMyShares returns the current user's share links.
func (h *ShareHandler) ListMyShares(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	cursor, err := h.db.ShareLinks().Find(ctx, bson.M{"userId": user.ID})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to list shares")
		return
	}
	defer cursor.Close(ctx)

	var shares []models.ShareLink
	if err := cursor.All(ctx, &shares); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to decode shares")
		return
	}
	if shares == nil {
		shares = []models.ShareLink{}
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"shares": shares})
}

// GetPublicShare returns a share link for public viewing (no auth required).
func (h *ShareHandler) GetPublicShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := mux.Vars(r)["token"]
	if token == "" {
		respondWithError(w, http.StatusBadRequest, "token required")
		return
	}

	var share models.ShareLink
	if err := h.db.ShareLinks().FindOne(ctx, bson.M{"token": token}).Decode(&share); err != nil {
		if err == mongo.ErrNoDocuments {
			respondWithError(w, http.StatusNotFound, "share not found or revoked")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to load share")
		return
	}

	messages := share.Messages
	if messages == nil {
		messages = []models.SharedMessage{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	if err := json.NewEncoder(w).Encode(publicShareResponse{
		Token:     share.Token,
		AgentID:   share.AgentID,
		AgentName: share.AgentName,
		Title:     share.Title,
		Messages:  messages,
		CreatedAt: share.CreatedAt,
	}); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
