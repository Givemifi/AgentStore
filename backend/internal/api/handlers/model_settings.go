package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"lastsaas/internal/db"
	"lastsaas/internal/middleware"
	"lastsaas/internal/models"
	"lastsaas/internal/validation"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ModelSettingsHandler handles tenant-scoped model provider, model config,
// and default model CRUD endpoints.
type ModelSettingsHandler struct {
	db *db.MongoDB
}

// NewModelSettingsHandler creates a new model settings handler.
func NewModelSettingsHandler(database *db.MongoDB) *ModelSettingsHandler {
	return &ModelSettingsHandler{db: database}
}

// --- Provider handlers ---

// ListProviders returns all model providers for the tenant.
func (h *ModelSettingsHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	cursor, err := h.db.ModelProviders().Find(r.Context(), bson.M{"tenantId": tenant.ID},
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to load providers")
		return
	}
	defer cursor.Close(r.Context())

	var providers []models.ModelProvider
	if err := cursor.All(r.Context(), &providers); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to decode providers")
		return
	}

	public := make([]models.PublicModelProvider, 0, len(providers))
	for _, p := range providers {
		public = append(public, p.ToPublic())
	}
	respondWithJSON(w, http.StatusOK, public)
}

// CreateProvider creates a new model provider in the tenant.
func (h *ModelSettingsHandler) CreateProvider(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	var req struct {
		Name         string              `json:"name"`
		ProviderType models.ProviderType `json:"providerType"`
		BaseURL      string              `json:"baseUrl"`
		APIKey       string              `json:"apiKey"`
		Enabled      bool                `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	now := time.Now()
	provider := models.ModelProvider{
		ID:           primitive.NewObjectID(),
		TenantID:     tenant.ID,
		Name:         strings.TrimSpace(req.Name),
		ProviderType: req.ProviderType,
		BaseURL:      strings.TrimSpace(req.BaseURL),
		APIKey:       strings.TrimSpace(req.APIKey),
		Enabled:      req.Enabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := validation.Validate(&provider); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := h.db.ModelProviders().InsertOne(r.Context(), provider); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			respondWithError(w, http.StatusConflict, "provider name already exists")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to save provider")
		return
	}

	respondWithJSON(w, http.StatusCreated, provider.ToPublic())
}

// UpdateProvider updates an existing model provider (tenant-scoped).
func (h *ModelSettingsHandler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	id, err := primitive.ObjectIDFromHex(mux.Vars(r)["providerId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid provider ID")
		return
	}

	var existing models.ModelProvider
	if err := h.db.ModelProviders().FindOne(r.Context(), bson.M{"_id": id, "tenantId": tenant.ID}).Decode(&existing); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			respondWithError(w, http.StatusNotFound, "provider not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to load provider")
		return
	}

	var req struct {
		Name         string              `json:"name"`
		ProviderType models.ProviderType `json:"providerType"`
		BaseURL      string              `json:"baseUrl"`
		APIKey       string              `json:"apiKey"`
		Enabled      bool                `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing.Name = strings.TrimSpace(req.Name)
	existing.ProviderType = req.ProviderType
	existing.BaseURL = strings.TrimSpace(req.BaseURL)
	existing.APIKey = preserveProviderKey(existing, req.APIKey)
	existing.Enabled = req.Enabled
	existing.UpdatedAt = time.Now()

	if err := validation.Validate(&existing); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := h.db.ModelProviders().ReplaceOne(r.Context(), bson.M{"_id": id, "tenantId": tenant.ID}, existing); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			respondWithError(w, http.StatusConflict, "provider name already exists")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to update provider")
		return
	}

	respondWithJSON(w, http.StatusOK, existing.ToPublic())
}

// DeleteProvider deletes a tenant-scoped model provider and its associated model configs.
func (h *ModelSettingsHandler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	providerID, err := primitive.ObjectIDFromHex(mux.Vars(r)["providerId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid provider ID")
		return
	}

	var provider models.ModelProvider
	if err := h.db.ModelProviders().FindOne(r.Context(), bson.M{"_id": providerID, "tenantId": tenant.ID}).Decode(&provider); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			respondWithError(w, http.StatusNotFound, "provider not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to load provider")
		return
	}

	cursor, err := h.db.ModelConfigs().Find(r.Context(), bson.M{"tenantId": tenant.ID, "providerId": providerID}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to load provider models")
		return
	}
	defer cursor.Close(r.Context())

	var modelDocs []struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	if err := cursor.All(r.Context(), &modelDocs); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to decode provider models")
		return
	}

	modelIDs := make([]primitive.ObjectID, 0, len(modelDocs))
	for _, doc := range modelDocs {
		modelIDs = append(modelIDs, doc.ID)
	}

	if err := deleteProviderWithCleanup(
		func() error {
			_, err := h.db.ModelConfigs().DeleteMany(r.Context(), bson.M{"tenantId": tenant.ID, "providerId": providerID})
			if err != nil {
				return fmt.Errorf("failed to delete provider models: %w", err)
			}
			return nil
		},
		func() error {
			return h.clearTenantDefaultsForModelIDs(r, tenant.ID, modelIDs)
		},
		func() error {
			_, err := h.db.ModelProviders().DeleteOne(r.Context(), bson.M{"_id": provider.ID, "tenantId": tenant.ID})
			if err != nil {
				return fmt.Errorf("failed to delete provider: %w", err)
			}
			return nil
		},
	); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// TestProvider tests connectivity to a model provider.
func (h *ModelSettingsHandler) TestProvider(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	id, err := primitive.ObjectIDFromHex(mux.Vars(r)["providerId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid provider ID")
		return
	}

	var provider models.ModelProvider
	if err := h.db.ModelProviders().FindOne(r.Context(), bson.M{"_id": id, "tenantId": tenant.ID}).Decode(&provider); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			respondWithError(w, http.StatusNotFound, "provider not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to load provider")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Model config handlers ---

// ListModels returns all model configs for the tenant, with optional modality filter.
func (h *ModelSettingsHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	filter := bson.M{"tenantId": tenant.ID}
	if modality := r.URL.Query().Get("modality"); modality != "" {
		filter["modality"] = modality
	}

	cursor, err := h.db.ModelConfigs().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to load models")
		return
	}
	defer cursor.Close(r.Context())

	var configs []models.ModelConfig
	if err := cursor.All(r.Context(), &configs); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to decode models")
		return
	}

	respondWithJSON(w, http.StatusOK, configs)
}

// CreateModel creates a new model config in the tenant.
func (h *ModelSettingsHandler) CreateModel(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	var req struct {
		ProviderID    string                 `json:"providerId"`
		Name          string                 `json:"name"`
		DisplayName   string                 `json:"displayName"`
		Modality      models.ModelModality   `json:"modality"`
		ModelID       string                 `json:"modelId"`
		DefaultParams map[string]interface{} `json:"defaultParams"`
		Enabled       bool                   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	providerID, err := primitive.ObjectIDFromHex(req.ProviderID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid provider ID")
		return
	}
	if err := h.ensureProviderBelongsToTenant(r.Context(), tenant.ID, providerID); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			respondWithError(w, http.StatusNotFound, "provider not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to load provider")
		return
	}

	now := time.Now()
	// Ensure DefaultParams is not nil
	defaultParams := req.DefaultParams
	if defaultParams == nil {
		defaultParams = map[string]interface{}{}
	}

	model := models.ModelConfig{
		ID:            primitive.NewObjectID(),
		TenantID:      tenant.ID,
		ProviderID:    providerID,
		Name:          strings.TrimSpace(req.Name),
		DisplayName:   strings.TrimSpace(req.DisplayName),
		Modality:      req.Modality,
		ModelID:       strings.TrimSpace(req.ModelID),
		DefaultParams: defaultParams,
		Enabled:       req.Enabled,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := validation.Validate(&model); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := h.db.ModelConfigs().InsertOne(r.Context(), model); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to save model")
		return
	}

	respondWithJSON(w, http.StatusCreated, model)
}

// UpdateModel updates an existing model config (tenant-scoped).
func (h *ModelSettingsHandler) UpdateModel(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	id, err := primitive.ObjectIDFromHex(mux.Vars(r)["modelId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid model ID")
		return
	}

	var existing models.ModelConfig
	if err := h.db.ModelConfigs().FindOne(r.Context(), bson.M{"_id": id, "tenantId": tenant.ID}).Decode(&existing); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			respondWithError(w, http.StatusNotFound, "model not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to load model")
		return
	}

	var req struct {
		ProviderID    string                 `json:"providerId"`
		Name          string                 `json:"name"`
		DisplayName   string                 `json:"displayName"`
		Modality      models.ModelModality   `json:"modality"`
		ModelID       string                 `json:"modelId"`
		DefaultParams map[string]interface{} `json:"defaultParams"`
		Enabled       bool                   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ProviderID != "" {
		providerID, err := primitive.ObjectIDFromHex(req.ProviderID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "invalid provider ID")
			return
		}
		if err := h.ensureProviderBelongsToTenant(r.Context(), tenant.ID, providerID); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				respondWithError(w, http.StatusNotFound, "provider not found")
				return
			}
			respondWithError(w, http.StatusInternalServerError, "failed to load provider")
			return
		}
		existing.ProviderID = providerID
	}

	existing.Name = strings.TrimSpace(req.Name)
	existing.DisplayName = strings.TrimSpace(req.DisplayName)
	existing.Modality = req.Modality
	existing.ModelID = strings.TrimSpace(req.ModelID)
	existing.DefaultParams = req.DefaultParams
	existing.Enabled = req.Enabled
	existing.UpdatedAt = time.Now()

	if err := validation.Validate(&existing); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := h.db.ModelConfigs().ReplaceOne(r.Context(), bson.M{"_id": id, "tenantId": tenant.ID}, existing); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to update model")
		return
	}

	respondWithJSON(w, http.StatusOK, existing)
}

// DeleteModel deletes a single tenant-scoped model config.
func (h *ModelSettingsHandler) DeleteModel(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	modelID, err := primitive.ObjectIDFromHex(mux.Vars(r)["modelId"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid model ID")
		return
	}

	deleted, err := deleteModelWithCleanup(
		func() (int64, error) {
			result, err := h.db.ModelConfigs().DeleteOne(r.Context(), bson.M{"_id": modelID, "tenantId": tenant.ID})
			if err != nil {
				return 0, fmt.Errorf("failed to delete model: %w", err)
			}
			return result.DeletedCount, nil
		},
		func() error {
			return h.clearTenantDefaultsForModelIDs(r, tenant.ID, []primitive.ObjectID{modelID})
		},
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !deleted {
		respondWithError(w, http.StatusNotFound, "model not found")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Defaults handler ---

// UpdateDefaults updates the tenant's default model config IDs.
func (h *ModelSettingsHandler) UpdateDefaults(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	var req tenantDefaultModelConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updateDoc, err := buildTenantDefaultModelConfigUpdate(req)
	if err != nil {
		switch err {
		case errInvalidTextModelConfigID:
			respondWithError(w, http.StatusBadRequest, "invalid text model config ID")
		case errInvalidImageModelConfigID:
			respondWithError(w, http.StatusBadRequest, "invalid image model config ID")
		case errInvalidVideoModelConfigID:
			respondWithError(w, http.StatusBadRequest, "invalid video model config ID")
		default:
			respondWithError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	if _, err := h.db.Tenants().UpdateOne(r.Context(), bson.M{"_id": tenant.ID}, updateDoc); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to update defaults")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Helpers ---

func deleteModelWithCleanup(deleteModel func() (int64, error), clearDefaults func() error) (bool, error) {
	deletedCount, err := deleteModel()
	if err != nil {
		return false, err
	}
	if deletedCount == 0 {
		return false, nil
	}
	if err := clearDefaults(); err != nil {
		return false, err
	}
	return true, nil
}

func deleteProviderWithCleanup(deleteModels func() error, clearDefaults func() error, deleteProvider func() error) error {
	if err := deleteModels(); err != nil {
		return err
	}
	if err := clearDefaults(); err != nil {
		return err
	}
	if err := deleteProvider(); err != nil {
		return err
	}
	return nil
}

func clearTenantDefaultsForModelIDsUpdate(modelIDs []primitive.ObjectID) bson.M {
	if len(modelIDs) == 0 {
		return nil
	}

	fields := []string{
		"defaultTextModelConfigId",
		"defaultImageModelConfigId",
		"defaultVideoModelConfigId",
	}
	unset := bson.M{}
	orClauses := make([]bson.M, 0, len(fields))
	for _, field := range fields {
		unset[field] = ""
		orClauses = append(orClauses, bson.M{field: bson.M{"$in": modelIDs}})
	}

	return bson.M{
		"$or":    orClauses,
		"$unset": unset,
		"$set":   bson.M{"updatedAt": time.Now()},
	}
}

var (
	errInvalidTextModelConfigID  = errors.New("invalid text model config ID")
	errInvalidImageModelConfigID = errors.New("invalid image model config ID")
	errInvalidVideoModelConfigID = errors.New("invalid video model config ID")
)

type tenantDefaultModelConfigUpdateRequest struct {
	DefaultTextModelConfigID  *string `json:"defaultTextModelConfigId"`
	DefaultImageModelConfigID *string `json:"defaultImageModelConfigId"`
	DefaultVideoModelConfigID *string `json:"defaultVideoModelConfigId"`
}

func buildTenantDefaultModelConfigUpdate(req tenantDefaultModelConfigUpdateRequest) (bson.M, error) {
	setDoc := bson.M{"updatedAt": time.Now()}
	unsetDoc := bson.M{}

	if err := applyTenantDefaultModelConfigField(setDoc, unsetDoc, "defaultTextModelConfigId", req.DefaultTextModelConfigID, errInvalidTextModelConfigID); err != nil {
		return nil, err
	}
	if err := applyTenantDefaultModelConfigField(setDoc, unsetDoc, "defaultImageModelConfigId", req.DefaultImageModelConfigID, errInvalidImageModelConfigID); err != nil {
		return nil, err
	}
	if err := applyTenantDefaultModelConfigField(setDoc, unsetDoc, "defaultVideoModelConfigId", req.DefaultVideoModelConfigID, errInvalidVideoModelConfigID); err != nil {
		return nil, err
	}

	updateDoc := bson.M{"$set": setDoc}
	if len(unsetDoc) > 0 {
		updateDoc["$unset"] = unsetDoc
	}
	return updateDoc, nil
}

func applyTenantDefaultModelConfigField(setDoc, unsetDoc bson.M, field string, raw *string, invalidErr error) error {
	if raw == nil {
		return nil
	}
	if strings.TrimSpace(*raw) == "" {
		unsetDoc[field] = ""
		return nil
	}
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(*raw))
	if err != nil {
		return invalidErr
	}
	setDoc[field] = &id
	return nil
}

func modelProviderTenantFilter(tenantID, providerID primitive.ObjectID) bson.M {
	return bson.M{"_id": providerID, "tenantId": tenantID}
}

func (h *ModelSettingsHandler) ensureProviderBelongsToTenant(ctx context.Context, tenantID, providerID primitive.ObjectID) error {
	return h.db.ModelProviders().FindOne(ctx, modelProviderTenantFilter(tenantID, providerID)).Err()
}

func (h *ModelSettingsHandler) clearTenantDefaultsForModelIDs(r *http.Request, tenantID primitive.ObjectID, modelIDs []primitive.ObjectID) error {
	update := clearTenantDefaultsForModelIDsUpdate(modelIDs)
	if update == nil {
		return nil
	}

	filter := bson.M{"_id": tenantID}
	for key, value := range update {
		if key == "$or" {
			filter[key] = value
		}
	}
	updateDoc := bson.M{}
	for key, value := range update {
		if key != "$or" {
			updateDoc[key] = value
		}
	}

	if _, err := h.db.Tenants().UpdateMany(r.Context(), filter, updateDoc); err != nil {
		return fmt.Errorf("failed to clear tenant defaults: %w", err)
	}
	return nil
}

// parseOptionalObjectID parses an optional ObjectID string. Returns nil if the
// value is empty or whitespace-only.
func parseOptionalObjectID(value string) (*primitive.ObjectID, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	id, err := primitive.ObjectIDFromHex(value)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// preserveProviderKey keeps the existing API key when the incoming value is
// blank or matches the masked preview of the existing key.
func preserveProviderKey(existing models.ModelProvider, incoming string) string {
	incoming = strings.TrimSpace(incoming)
	if incoming == "" || incoming == models.MaskAPIKey(existing.APIKey) {
		return existing.APIKey
	}
	return incoming
}
