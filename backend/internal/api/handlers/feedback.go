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

// FeedbackHandler handles per-message thumbs up/down feedback from users.
type FeedbackHandler struct {
	db *db.MongoDB
}

// NewFeedbackHandler creates a feedback handler.
func NewFeedbackHandler(database *db.MongoDB) *FeedbackHandler {
	return &FeedbackHandler{db: database}
}

type setFeedbackRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

// SetFeedback upserts the current user's feedback on an assistant message.
func (h *FeedbackHandler) SetFeedback(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusBadRequest, "Tenant context required")
		return
	}
	messageID, err := primitive.ObjectIDFromHex(mux.Vars(r)["messageId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	var req setFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Rating != 1 && req.Rating != -1 {
		respondWithError(w, http.StatusBadRequest, "Rating must be 1 or -1")
		return
	}

	// The message must exist, belong to this user+tenant, and be a completed
	// assistant message (you rate the AI's answer, not your own prompt).
	var msg models.ChatMessage
	err = h.db.ChatMessages().FindOne(r.Context(), bson.M{
		"_id":      messageID,
		"tenantId": tenant.ID,
		"userId":   user.ID,
	}).Decode(&msg)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			respondWithError(w, http.StatusNotFound, "Message not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to load message")
		return
	}
	if msg.Role != "assistant" {
		respondWithError(w, http.StatusBadRequest, "Only assistant messages can be rated")
		return
	}

	now := time.Now()
	comment := strings.TrimSpace(req.Comment)
	if len(comment) > 2000 {
		comment = comment[:2000]
	}

	// Validate a representative document (schema/tag parity) before upsert.
	candidate := models.MessageFeedback{
		TenantID:       tenant.ID,
		UserID:         user.ID,
		ConversationID: msg.ConversationID,
		MessageID:      messageID,
		AgentID:        msg.AgentID,
		Rating:         req.Rating,
		Comment:        comment,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := validation.Validate(&candidate); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	_, err = h.db.MessageFeedback().UpdateOne(r.Context(),
		bson.M{"messageId": messageID, "userId": user.ID},
		bson.M{
			"$set": bson.M{
				"rating":    req.Rating,
				"comment":   comment,
				"updatedAt": now,
			},
			"$setOnInsert": bson.M{
				"tenantId":       tenant.ID,
				"conversationId": msg.ConversationID,
				"agentId":        msg.AgentID,
				"createdAt":      now,
			},
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save feedback")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]any{"rating": req.Rating, "comment": comment})
}

// DeleteFeedback removes the current user's feedback on a message.
func (h *FeedbackHandler) DeleteFeedback(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	messageID, err := primitive.ObjectIDFromHex(mux.Vars(r)["messageId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}
	_, err = h.db.MessageFeedback().DeleteOne(r.Context(), bson.M{"messageId": messageID, "userId": user.ID})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete feedback")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ListConversationFeedback returns the current user's feedback for all messages
// in a conversation, as a map of messageId -> {rating, comment}.
func (h *FeedbackHandler) ListConversationFeedback(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusBadRequest, "Tenant context required")
		return
	}
	conversationID, err := primitive.ObjectIDFromHex(mux.Vars(r)["conversationId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid conversation ID")
		return
	}

	cursor, err := h.db.MessageFeedback().Find(r.Context(), bson.M{
		"conversationId": conversationID,
		"tenantId":       tenant.ID,
		"userId":         user.ID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to load feedback")
		return
	}
	defer cursor.Close(r.Context())

	var items []models.MessageFeedback
	if err := cursor.All(r.Context(), &items); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to decode feedback")
		return
	}

	out := make(map[string]map[string]any, len(items))
	for _, it := range items {
		out[it.MessageID.Hex()] = map[string]any{"rating": it.Rating, "comment": it.Comment}
	}
	respondWithJSON(w, http.StatusOK, out)
}
