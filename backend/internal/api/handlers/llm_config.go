package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"agentstore/internal/db"
	"agentstore/internal/models"
	"agentstore/internal/validation"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// LLMConfigHandler handles LLM configuration API endpoints.
type LLMConfigHandler struct {
	db *db.MongoDB
}

// NewLLMConfigHandler creates a new LLM config handler.
func NewLLMConfigHandler(database *db.MongoDB) *LLMConfigHandler {
	return &LLMConfigHandler{db: database}
}

// GetConfig returns the current LLM configuration.
func (h *LLMConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	var config models.LLMConfig
	err := h.db.LLMConfigs().FindOne(r.Context(), bson.M{"key": models.DefaultLLMConfigKey}).Decode(&config)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			respondWithJSON(w, http.StatusOK, nil)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to load config")
		return
	}

	config.APIKey = ""
	respondWithJSON(w, http.StatusOK, config)
}

// UpdateConfig updates the LLM configuration.
func (h *LLMConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		APIKey   string `json:"apiKey"`
		BaseURL  string `json:"baseURL"`
		Model    string `json:"model"`
		IsActive bool   `json:"isActive"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	now := time.Now()
	var existing models.LLMConfig
	err := h.db.LLMConfigs().FindOne(r.Context(), bson.M{"key": models.DefaultLLMConfigKey}).Decode(&existing)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		respondWithError(w, http.StatusInternalServerError, "Failed to load config")
		return
	}

	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "********" {
		apiKey = ""
	}
	if apiKey == "" {
		apiKey = existing.APIKey
	}

	config := models.LLMConfig{
		ID:        existing.ID,
		Key:       models.DefaultLLMConfigKey,
		APIKey:    apiKey,
		BaseURL:   strings.TrimSpace(req.BaseURL),
		Model:     strings.TrimSpace(req.Model),
		IsActive:  req.IsActive,
		CreatedAt: existing.CreatedAt,
		UpdatedAt: now,
	}
	if config.ID.IsZero() {
		config.ID = primitive.NewObjectID()
	}
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	if err := validation.Validate(&config); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	_, err = h.db.LLMConfigs().UpdateOne(
		r.Context(),
		bson.M{"key": models.DefaultLLMConfigKey},
		bson.M{
			"$set": bson.M{
				"apiKey":    config.APIKey,
				"baseURL":   config.BaseURL,
				"model":     config.Model,
				"isActive":  config.IsActive,
				"updatedAt": config.UpdatedAt,
			},
			"$setOnInsert": bson.M{
				"_id":       config.ID,
				"key":       models.DefaultLLMConfigKey,
				"createdAt": config.CreatedAt,
			},
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save config")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// GetLLMConfig returns the current LLM config for use by the LLM client.
func GetLLMConfig(ctx context.Context, db *db.MongoDB) *models.LLMConfig {
	var config models.LLMConfig
	err := db.LLMConfigs().FindOne(ctx, bson.M{"key": models.DefaultLLMConfigKey, "isActive": true}).Decode(&config)
	if err != nil {
		return nil
	}
	return &config
}

func (h *LLMConfigHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("", h.GetConfig).Methods("GET")
	r.HandleFunc("", h.UpdateConfig).Methods("PUT")
}