package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"lastsaas/internal/middleware"
	"lastsaas/internal/models"
	"lastsaas/internal/testutil"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestModelProviderAPI_MasksAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	_, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{
		ID:           primitive.NewObjectID(),
		TenantID:     tenant.ID,
		Name:         "Provider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk-secret-1234",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	req := env.tenantRequest(t, http.MethodGet, "/api/tenant/model-providers", nil, user, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body %s", resp.StatusCode, body)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if strings.Contains(bodyStr, "sk-secret-1234") {
		t.Fatalf("leaked raw key: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "sk-***1234") {
		t.Fatalf("expected masked key preview, got %s", bodyStr)
	}
}

func TestModelProviderAPI_PreservesExistingKeyOnMaskedUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	providerID := primitive.NewObjectID()
	_, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{
		ID:           providerID,
		TenantID:     tenant.ID,
		Name:         "Provider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://old.example.com/v1",
		APIKey:       "sk-secret-1234",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	body := strings.NewReader(`{"name":"Provider","providerType":"openai_compatible","baseUrl":"https://new.example.com/v1","apiKey":"sk-***1234","enabled":true}`)
	req := env.tenantRequest(t, http.MethodPut, "/api/tenant/model-providers/"+providerID.Hex(), body, user, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body %s", resp.StatusCode, respBody)
	}

	var saved models.ModelProvider
	if err := env.DB.ModelProviders().FindOne(context.Background(), bson.M{"_id": providerID}).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.APIKey != "sk-secret-1234" {
		t.Fatalf("expected key preserved, got %q", saved.APIKey)
	}
	if saved.BaseURL != "https://new.example.com/v1" {
		t.Fatalf("expected base URL update, got %q", saved.BaseURL)
	}
}

func TestModelDefaults_UpdateTenantDefaults(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	modelID := primitive.NewObjectID()
	providerID := primitive.NewObjectID()
	_, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{
		ID:           providerID,
		TenantID:     tenant.ID,
		Name:         "Provider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = env.DB.ModelConfigs().InsertOne(context.Background(), models.ModelConfig{
		ID:          modelID,
		TenantID:    tenant.ID,
		ProviderID:  providerID,
		Name:        "Text",
		DisplayName: "Text",
		Modality:    models.ModelModalityText,
		ModelID:     "gpt-4o-mini",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	body := strings.NewReader(`{"defaultTextModelConfigId":"` + modelID.Hex() + `"}`)
	req := env.tenantRequest(t, http.MethodPost, "/api/tenant/model-defaults", body, user, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body %s", resp.StatusCode, respBody)
	}

	var saved models.Tenant
	if err := env.DB.Tenants().FindOne(context.Background(), bson.M{"_id": tenant.ID}).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.DefaultTextModelConfigID == nil || *saved.DefaultTextModelConfigID != modelID {
		t.Fatalf("expected default text model saved, got %v", saved.DefaultTextModelConfigID)
	}
}

func TestModelDefaults_UpdateTenantDefaults_EmptyStringClearsAndOmittedFieldsRemain(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	providerID := primitive.NewObjectID()
	if _, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{
		ID:           providerID,
		TenantID:     tenant.ID,
		Name:         "Provider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	textModelID := primitive.NewObjectID()
	imageModelID := primitive.NewObjectID()
	for _, model := range []models.ModelConfig{
		{
			ID:          textModelID,
			TenantID:    tenant.ID,
			ProviderID:  providerID,
			Name:        "Text",
			DisplayName: "Text",
			Modality:    models.ModelModalityText,
			ModelID:     "gpt-text",
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          imageModelID,
			TenantID:    tenant.ID,
			ProviderID:  providerID,
			Name:        "Image",
			DisplayName: "Image",
			Modality:    models.ModelModalityImage,
			ModelID:     "gpt-image",
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	} {
		if _, err := env.DB.ModelConfigs().InsertOne(context.Background(), model); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := env.DB.Tenants().UpdateOne(context.Background(), bson.M{"_id": tenant.ID}, bson.M{"$set": bson.M{
		"defaultTextModelConfigId":  textModelID,
		"defaultImageModelConfigId": imageModelID,
		"updatedAt":                 time.Now(),
	}}); err != nil {
		t.Fatal(err)
	}

	body := strings.NewReader(`{"defaultTextModelConfigId":""}`)
	req := env.tenantRequest(t, http.MethodPost, "/api/tenant/model-defaults", body, user, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body %s", resp.StatusCode, respBody)
	}

	var saved models.Tenant
	if err := env.DB.Tenants().FindOne(context.Background(), bson.M{"_id": tenant.ID}).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.DefaultTextModelConfigID != nil {
		t.Fatalf("expected empty string request to clear text default, got %v", saved.DefaultTextModelConfigID)
	}
	if saved.DefaultImageModelConfigID == nil || *saved.DefaultImageModelConfigID != imageModelID {
		t.Fatalf("expected omitted image default to remain unchanged, got %v", saved.DefaultImageModelConfigID)
	}
}

func TestModelProviderAPI_CreateProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	body := strings.NewReader(`{"name":"OpenAI","providerType":"openai_compatible","baseUrl":"https://api.openai.com/v1","apiKey":"sk-new-key-12345678","enabled":true}`)
	req := env.tenantRequest(t, http.MethodPost, "/api/tenant/model-providers", body, user, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d body %s", resp.StatusCode, respBody)
	}

	var public models.PublicModelProvider
	if err := json.NewDecoder(resp.Body).Decode(&public); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if public.APIKeyPreview != "sk-***5678" {
		t.Fatalf("expected masked key preview, got %q", public.APIKeyPreview)
	}
	if public.Name != "OpenAI" {
		t.Fatalf("expected name OpenAI, got %q", public.Name)
	}
}

func TestModelConfigAPI_CreateModel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	providerID := primitive.NewObjectID()
	_, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{
		ID:           providerID,
		TenantID:     tenant.ID,
		Name:         "Provider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	body := strings.NewReader(`{"providerId":"` + providerID.Hex() + `","name":"GPT-4o","displayName":"GPT-4o","modality":"text","modelId":"gpt-4o","enabled":true}`)
	req := env.tenantRequest(t, http.MethodPost, "/api/tenant/model-configs", body, user, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d body %s", resp.StatusCode, respBody)
	}

	var config models.ModelConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if config.ModelID != "gpt-4o" {
		t.Fatalf("expected modelId gpt-4o, got %q", config.ModelID)
	}
	if config.ProviderID != providerID {
		t.Fatalf("expected providerId %s, got %s", providerID.Hex(), config.ProviderID.Hex())
	}
}

func TestModelConfigAPI_CreateModel_RejectsMissingProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	missingProviderID := primitive.NewObjectID()
	body := strings.NewReader(`{"providerId":"` + missingProviderID.Hex() + `","name":"GPT-4o","displayName":"GPT-4o","modality":"text","modelId":"gpt-4o","enabled":true}`)
	req := env.tenantRequest(t, http.MethodPost, "/api/tenant/model-configs", body, user, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d body %s", resp.StatusCode, respBody)
	}

	var got ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if got.Error != "provider not found" {
		t.Fatalf("expected provider not found error, got %q", got.Error)
	}
}

func TestModelConfigAPI_CreateModel_RejectsCrossTenantProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	otherTenant := testutil.CreateTestTenant(t, env.DB, "Other Tenant", user.ID, false)

	otherProviderID := primitive.NewObjectID()
	if _, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{
		ID:           otherProviderID,
		TenantID:     otherTenant.ID,
		Name:         "Other Provider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk-other",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	body := strings.NewReader(`{"providerId":"` + otherProviderID.Hex() + `","name":"GPT-4o","displayName":"GPT-4o","modality":"text","modelId":"gpt-4o","enabled":true}`)
	req := env.tenantRequest(t, http.MethodPost, "/api/tenant/model-configs", body, user, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d body %s", resp.StatusCode, respBody)
	}

	var got ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if got.Error != "provider not found" {
		t.Fatalf("expected provider not found error, got %q", got.Error)
	}
}

func TestModelSettingsRoutes_RegisterDeleteEndpoints(t *testing.T) {
	serverFile, err := os.ReadFile("../../../cmd/server/main.go")
	if err != nil {
		t.Fatalf("read server routes: %v", err)
	}
	serverText := string(serverFile)
	if !strings.Contains(serverText, `modelSettingsRouter.HandleFunc("/model-providers/{providerId}", modelSettingsHandler.DeleteProvider).Methods("DELETE")`) {
		t.Fatalf("expected DELETE /model-providers route in backend/cmd/server/main.go")
	}
	if !strings.Contains(serverText, `modelSettingsRouter.HandleFunc("/model-configs/{modelId}", modelSettingsHandler.DeleteModel).Methods("DELETE")`) {
		t.Fatalf("expected DELETE /model-configs route in backend/cmd/server/main.go")
	}

	testHelpersFile, err := os.ReadFile("testhelpers_test.go")
	if err != nil {
		t.Fatalf("read test helper routes: %v", err)
	}
	testHelpersText := string(testHelpersFile)
	if !strings.Contains(testHelpersText, `modelSettingsRouter.HandleFunc("/model-providers/{providerId}", modelSettingsHandler.DeleteProvider).Methods("DELETE")`) {
		t.Fatalf("expected DELETE /model-providers route in backend/internal/api/handlers/testhelpers_test.go")
	}
	if !strings.Contains(testHelpersText, `modelSettingsRouter.HandleFunc("/model-configs/{modelId}", modelSettingsHandler.DeleteModel).Methods("DELETE")`) {
		t.Fatalf("expected DELETE /model-configs route in backend/internal/api/handlers/testhelpers_test.go")
	}
}

func TestModelSettingsHandler_DeleteProvider_InvalidID(t *testing.T) {
	handler := NewModelSettingsHandler(nil)
	req := httptest.NewRequest(http.MethodDelete, "/api/tenant/model-providers/not-a-valid-id", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.TenantContextKey, &models.Tenant{ID: primitive.NewObjectID()}))
	req = mux.SetURLVars(req, map[string]string{"providerId": "not-a-valid-id"})
	rr := httptest.NewRecorder()

	handler.DeleteProvider(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid provider ID, got %d", rr.Code)
	}
}

func TestModelSettingsHandler_DeleteModel_InvalidID(t *testing.T) {
	handler := NewModelSettingsHandler(nil)
	req := httptest.NewRequest(http.MethodDelete, "/api/tenant/model-configs/not-a-valid-id", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.TenantContextKey, &models.Tenant{ID: primitive.NewObjectID()}))
	req = mux.SetURLVars(req, map[string]string{"modelId": "not-a-valid-id"})
	rr := httptest.NewRecorder()

	handler.DeleteModel(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid model ID, got %d", rr.Code)
	}
}

func TestModelProviderAPI_DeleteProvider_CascadesTenantModels(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	otherTenant := testutil.CreateTestTenant(t, env.DB, "Other Tenant", user.ID, false)

	providerID := primitive.NewObjectID()
	otherProviderID := primitive.NewObjectID()
	for _, provider := range []models.ModelProvider{
		{
			ID:           providerID,
			TenantID:     tenant.ID,
			Name:         "Delete Me",
			ProviderType: models.ProviderTypeOpenAICompatible,
			BaseURL:      "https://api.example.com/v1",
			APIKey:       "sk-delete-me",
			Enabled:      true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			ID:           otherProviderID,
			TenantID:     tenant.ID,
			Name:         "Keep Me",
			ProviderType: models.ProviderTypeOpenAICompatible,
			BaseURL:      "https://api.example.com/v1",
			APIKey:       "sk-keep-me",
			Enabled:      true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	} {
		if _, err := env.DB.ModelProviders().InsertOne(context.Background(), provider); err != nil {
			t.Fatal(err)
		}
	}

	deleteTextModelID := primitive.NewObjectID()
	deleteImageModelID := primitive.NewObjectID()
	keepSameTenantModelID := primitive.NewObjectID()
	for _, model := range []models.ModelConfig{
		{
			ID:          deleteTextModelID,
			TenantID:    tenant.ID,
			ProviderID:  providerID,
			Name:        "Delete Text",
			DisplayName: "Delete Text",
			Modality:    models.ModelModalityText,
			ModelID:     "gpt-delete-1",
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          deleteImageModelID,
			TenantID:    tenant.ID,
			ProviderID:  providerID,
			Name:        "Delete Image",
			DisplayName: "Delete Image",
			Modality:    models.ModelModalityImage,
			ModelID:     "gpt-delete-2",
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          keepSameTenantModelID,
			TenantID:    tenant.ID,
			ProviderID:  otherProviderID,
			Name:        "Keep Same Tenant",
			DisplayName: "Keep Same Tenant",
			Modality:    models.ModelModalityVideo,
			ModelID:     "gpt-keep-1",
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          primitive.NewObjectID(),
			TenantID:    otherTenant.ID,
			ProviderID:  providerID,
			Name:        "Keep Other Tenant",
			DisplayName: "Keep Other Tenant",
			Modality:    models.ModelModalityText,
			ModelID:     "gpt-keep-2",
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	} {
		if _, err := env.DB.ModelConfigs().InsertOne(context.Background(), model); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := env.DB.Tenants().UpdateOne(context.Background(), bson.M{"_id": tenant.ID}, bson.M{"$set": bson.M{
		"defaultTextModelConfigId":  deleteTextModelID,
		"defaultImageModelConfigId": deleteImageModelID,
		"defaultVideoModelConfigId": keepSameTenantModelID,
		"updatedAt":                 time.Now(),
	}}); err != nil {
		t.Fatal(err)
	}

	req := env.tenantRequest(t, http.MethodDelete, "/api/tenant/model-providers/"+providerID.Hex(), nil, user, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body %s", resp.StatusCode, respBody)
	}

	if count, err := env.DB.ModelProviders().CountDocuments(context.Background(), bson.M{"_id": providerID, "tenantId": tenant.ID}); err != nil {
		t.Fatal(err)
	} else if count != 0 {
		t.Fatalf("expected provider deleted, found %d", count)
	}
	if count, err := env.DB.ModelConfigs().CountDocuments(context.Background(), bson.M{"tenantId": tenant.ID, "providerId": providerID}); err != nil {
		t.Fatal(err)
	} else if count != 0 {
		t.Fatalf("expected associated tenant models deleted, found %d", count)
	}
	if count, err := env.DB.ModelConfigs().CountDocuments(context.Background(), bson.M{"tenantId": tenant.ID, "providerId": otherProviderID}); err != nil {
		t.Fatal(err)
	} else if count != 1 {
		t.Fatalf("expected unrelated tenant model retained, found %d", count)
	}
	if count, err := env.DB.ModelConfigs().CountDocuments(context.Background(), bson.M{"tenantId": otherTenant.ID, "providerId": providerID}); err != nil {
		t.Fatal(err)
	} else if count != 1 {
		t.Fatalf("expected other tenant model retained, found %d", count)
	}

	var saved models.Tenant
	if err := env.DB.Tenants().FindOne(context.Background(), bson.M{"_id": tenant.ID}).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.DefaultTextModelConfigID != nil {
		t.Fatalf("expected deleted text default cleared, got %v", saved.DefaultTextModelConfigID)
	}
	if saved.DefaultImageModelConfigID != nil {
		t.Fatalf("expected deleted image default cleared, got %v", saved.DefaultImageModelConfigID)
	}
	if saved.DefaultVideoModelConfigID == nil || *saved.DefaultVideoModelConfigID != keepSameTenantModelID {
		t.Fatalf("expected unrelated video default retained, got %v", saved.DefaultVideoModelConfigID)
	}
}

func TestModelConfigAPI_DeleteModel_TenantScoped(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	otherTenant := testutil.CreateTestTenant(t, env.DB, "Second Tenant", user.ID, false)

	providerID := primitive.NewObjectID()
	if _, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{
		ID:           providerID,
		TenantID:     tenant.ID,
		Name:         "Provider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	targetModelID := primitive.NewObjectID()
	otherTenantModelID := primitive.NewObjectID()
	otherTenantProviderID := primitive.NewObjectID()
	if _, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{
		ID:           otherTenantProviderID,
		TenantID:     otherTenant.ID,
		Name:         "Other Tenant Provider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk-other",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	otherTenantImageModelID := primitive.NewObjectID()
	otherTenantVideoModelID := primitive.NewObjectID()
	for _, model := range []models.ModelConfig{
		{
			ID:          targetModelID,
			TenantID:    tenant.ID,
			ProviderID:  providerID,
			Name:        "Delete Me",
			DisplayName: "Delete Me",
			Modality:    models.ModelModalityText,
			ModelID:     "gpt-delete",
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          otherTenantModelID,
			TenantID:    otherTenant.ID,
			ProviderID:  providerID,
			Name:        "Keep Other Tenant",
			DisplayName: "Keep Other Tenant",
			Modality:    models.ModelModalityText,
			ModelID:     "gpt-keep",
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          otherTenantImageModelID,
			TenantID:    otherTenant.ID,
			ProviderID:  otherTenantProviderID,
			Name:        "Other Tenant Image",
			DisplayName: "Other Tenant Image",
			Modality:    models.ModelModalityImage,
			ModelID:     "gpt-image",
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          otherTenantVideoModelID,
			TenantID:    otherTenant.ID,
			ProviderID:  otherTenantProviderID,
			Name:        "Other Tenant Video",
			DisplayName: "Other Tenant Video",
			Modality:    models.ModelModalityVideo,
			ModelID:     "gpt-video",
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	} {
		if _, err := env.DB.ModelConfigs().InsertOne(context.Background(), model); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := env.DB.Tenants().UpdateOne(context.Background(), bson.M{"_id": tenant.ID}, bson.M{"$set": bson.M{
		"defaultTextModelConfigId": targetModelID,
		"updatedAt":                time.Now(),
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := env.DB.Tenants().UpdateOne(context.Background(), bson.M{"_id": otherTenant.ID}, bson.M{"$set": bson.M{
		"defaultTextModelConfigId":  otherTenantModelID,
		"defaultImageModelConfigId": otherTenantImageModelID,
		"defaultVideoModelConfigId": otherTenantVideoModelID,
		"updatedAt":                 time.Now(),
	}}); err != nil {
		t.Fatal(err)
	}

	req := env.tenantRequest(t, http.MethodDelete, "/api/tenant/model-configs/"+targetModelID.Hex(), nil, user, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body %s", resp.StatusCode, respBody)
	}

	if count, err := env.DB.ModelConfigs().CountDocuments(context.Background(), bson.M{"_id": targetModelID, "tenantId": tenant.ID}); err != nil {
		t.Fatal(err)
	} else if count != 0 {
		t.Fatalf("expected model deleted, found %d", count)
	}
	if count, err := env.DB.ModelConfigs().CountDocuments(context.Background(), bson.M{"_id": otherTenantModelID, "tenantId": otherTenant.ID}); err != nil {
		t.Fatal(err)
	} else if count != 1 {
		t.Fatalf("expected other tenant model retained, found %d", count)
	}

	var savedTenant models.Tenant
	if err := env.DB.Tenants().FindOne(context.Background(), bson.M{"_id": tenant.ID}).Decode(&savedTenant); err != nil {
		t.Fatal(err)
	}
	if savedTenant.DefaultTextModelConfigID != nil {
		t.Fatalf("expected deleted tenant default cleared, got %v", savedTenant.DefaultTextModelConfigID)
	}

	var savedOtherTenant models.Tenant
	if err := env.DB.Tenants().FindOne(context.Background(), bson.M{"_id": otherTenant.ID}).Decode(&savedOtherTenant); err != nil {
		t.Fatal(err)
	}
	if savedOtherTenant.DefaultTextModelConfigID == nil || *savedOtherTenant.DefaultTextModelConfigID != otherTenantModelID {
		t.Fatalf("expected other tenant text default retained, got %v", savedOtherTenant.DefaultTextModelConfigID)
	}
	if savedOtherTenant.DefaultImageModelConfigID == nil || *savedOtherTenant.DefaultImageModelConfigID != otherTenantImageModelID {
		t.Fatalf("expected other tenant image default retained, got %v", savedOtherTenant.DefaultImageModelConfigID)
	}
	if savedOtherTenant.DefaultVideoModelConfigID == nil || *savedOtherTenant.DefaultVideoModelConfigID != otherTenantVideoModelID {
		t.Fatalf("expected other tenant video default retained, got %v", savedOtherTenant.DefaultVideoModelConfigID)
	}
}

func TestModelConfigAPI_DeleteModel_NotFoundDoesNotMutateDefaults(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	otherTenant := testutil.CreateTestTenant(t, env.DB, "Second Tenant", user.ID, false)

	providerID := primitive.NewObjectID()
	if _, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{
		ID:           providerID,
		TenantID:     tenant.ID,
		Name:         "Provider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	otherTenantProviderID := primitive.NewObjectID()
	if _, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{
		ID:           otherTenantProviderID,
		TenantID:     otherTenant.ID,
		Name:         "Other Tenant Provider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk-other",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	retainedImageModelID := primitive.NewObjectID()
	otherTenantModelID := primitive.NewObjectID()
	for _, model := range []models.ModelConfig{
		{
			ID:          retainedImageModelID,
			TenantID:    tenant.ID,
			ProviderID:  providerID,
			Name:        "Keep Image",
			DisplayName: "Keep Image",
			Modality:    models.ModelModalityImage,
			ModelID:     "gpt-keep-image",
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          otherTenantModelID,
			TenantID:    otherTenant.ID,
			ProviderID:  otherTenantProviderID,
			Name:        "Other Tenant Model",
			DisplayName: "Other Tenant Model",
			Modality:    models.ModelModalityText,
			ModelID:     "gpt-other",
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	} {
		if _, err := env.DB.ModelConfigs().InsertOne(context.Background(), model); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := env.DB.Tenants().UpdateOne(context.Background(), bson.M{"_id": tenant.ID}, bson.M{"$set": bson.M{
		"defaultTextModelConfigId":  otherTenantModelID,
		"defaultImageModelConfigId": retainedImageModelID,
		"updatedAt":                 time.Now(),
	}}); err != nil {
		t.Fatal(err)
	}

	req := env.tenantRequest(t, http.MethodDelete, "/api/tenant/model-configs/"+otherTenantModelID.Hex(), nil, user, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d body %s", resp.StatusCode, respBody)
	}

	var savedTenant models.Tenant
	if err := env.DB.Tenants().FindOne(context.Background(), bson.M{"_id": tenant.ID}).Decode(&savedTenant); err != nil {
		t.Fatal(err)
	}
	if savedTenant.DefaultTextModelConfigID == nil || *savedTenant.DefaultTextModelConfigID != otherTenantModelID {
		t.Fatalf("expected cross-tenant default retained on 404 delete, got %v", savedTenant.DefaultTextModelConfigID)
	}
	if savedTenant.DefaultImageModelConfigID == nil || *savedTenant.DefaultImageModelConfigID != retainedImageModelID {
		t.Fatalf("expected unrelated same-tenant image default retained on 404 delete, got %v", savedTenant.DefaultImageModelConfigID)
	}
}

func TestModelConfigAPI_DeleteModel_ClearsOnlyMatchingTenantDefaults(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	providerID := primitive.NewObjectID()
	if _, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{
		ID:           providerID,
		TenantID:     tenant.ID,
		Name:         "Provider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	deletedModelID := primitive.NewObjectID()
	retainedImageModelID := primitive.NewObjectID()
	retainedVideoModelID := primitive.NewObjectID()
	for _, model := range []models.ModelConfig{
		{
			ID:          deletedModelID,
			TenantID:    tenant.ID,
			ProviderID:  providerID,
			Name:        "Delete Text",
			DisplayName: "Delete Text",
			Modality:    models.ModelModalityText,
			ModelID:     "gpt-delete",
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          retainedImageModelID,
			TenantID:    tenant.ID,
			ProviderID:  providerID,
			Name:        "Keep Image",
			DisplayName: "Keep Image",
			Modality:    models.ModelModalityImage,
			ModelID:     "gpt-image",
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          retainedVideoModelID,
			TenantID:    tenant.ID,
			ProviderID:  providerID,
			Name:        "Keep Video",
			DisplayName: "Keep Video",
			Modality:    models.ModelModalityVideo,
			ModelID:     "gpt-video",
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	} {
		if _, err := env.DB.ModelConfigs().InsertOne(context.Background(), model); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := env.DB.Tenants().UpdateOne(context.Background(), bson.M{"_id": tenant.ID}, bson.M{"$set": bson.M{
		"defaultTextModelConfigId":  deletedModelID,
		"defaultImageModelConfigId": retainedImageModelID,
		"defaultVideoModelConfigId": retainedVideoModelID,
		"updatedAt":                 time.Now(),
	}}); err != nil {
		t.Fatal(err)
	}

	req := env.tenantRequest(t, http.MethodDelete, "/api/tenant/model-configs/"+deletedModelID.Hex(), nil, user, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body %s", resp.StatusCode, respBody)
	}

	var savedTenant models.Tenant
	if err := env.DB.Tenants().FindOne(context.Background(), bson.M{"_id": tenant.ID}).Decode(&savedTenant); err != nil {
		t.Fatal(err)
	}
	if savedTenant.DefaultTextModelConfigID != nil {
		t.Fatalf("expected deleted text default cleared, got %v", savedTenant.DefaultTextModelConfigID)
	}
	if savedTenant.DefaultImageModelConfigID == nil || *savedTenant.DefaultImageModelConfigID != retainedImageModelID {
		t.Fatalf("expected unrelated image default retained, got %v", savedTenant.DefaultImageModelConfigID)
	}
	if savedTenant.DefaultVideoModelConfigID == nil || *savedTenant.DefaultVideoModelConfigID != retainedVideoModelID {
		t.Fatalf("expected unrelated video default retained, got %v", savedTenant.DefaultVideoModelConfigID)
	}
}

func TestBuildTenantDefaultModelConfigUpdate_OmittedFieldsRemainUnchanged(t *testing.T) {
	textModelID := primitive.NewObjectID()
	videoModelID := primitive.NewObjectID()

	update, err := buildTenantDefaultModelConfigUpdate(tenantDefaultModelConfigUpdateRequest{
		DefaultTextModelConfigID:  ptrString(textModelID.Hex()),
		DefaultImageModelConfigID: nil,
		DefaultVideoModelConfigID: ptrString(videoModelID.Hex()),
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	setDoc, ok := update["$set"].(bson.M)
	if !ok {
		t.Fatalf("expected $set bson.M, got %T", update["$set"])
	}
	if got, ok := setDoc["defaultTextModelConfigId"].(*primitive.ObjectID); !ok || got == nil || *got != textModelID {
		t.Fatalf("expected text default set to %s, got %v", textModelID.Hex(), setDoc["defaultTextModelConfigId"])
	}
	if _, exists := setDoc["defaultImageModelConfigId"]; exists {
		t.Fatalf("expected omitted image field to be left unchanged, got $set=%v", setDoc)
	}
	if got, ok := setDoc["defaultVideoModelConfigId"].(*primitive.ObjectID); !ok || got == nil || *got != videoModelID {
		t.Fatalf("expected video default set to %s, got %v", videoModelID.Hex(), setDoc["defaultVideoModelConfigId"])
	}
	if _, ok := setDoc["updatedAt"]; !ok {
		t.Fatalf("expected updatedAt in $set, got %v", setDoc)
	}

	if unsetDoc, ok := update["$unset"].(bson.M); ok && len(unsetDoc) > 0 {
		t.Fatalf("expected no fields unset, got %v", unsetDoc)
	}
}

func TestBuildTenantDefaultModelConfigUpdate_EmptyStringClearsField(t *testing.T) {
	update, err := buildTenantDefaultModelConfigUpdate(tenantDefaultModelConfigUpdateRequest{
		DefaultTextModelConfigID:  ptrString(""),
		DefaultImageModelConfigID: nil,
		DefaultVideoModelConfigID: nil,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	unsetDoc, ok := update["$unset"].(bson.M)
	if !ok {
		t.Fatalf("expected $unset bson.M, got %T", update["$unset"])
	}
	if got := unsetDoc["defaultTextModelConfigId"]; got != "" {
		t.Fatalf("expected text default unset marker, got %v", got)
	}

	setDoc, ok := update["$set"].(bson.M)
	if !ok {
		t.Fatalf("expected $set bson.M, got %T", update["$set"])
	}
	if _, exists := setDoc["defaultTextModelConfigId"]; exists {
		t.Fatalf("expected cleared field excluded from $set, got %v", setDoc)
	}
}

func TestBuildTenantDefaultModelConfigUpdate_InvalidID(t *testing.T) {
	_, err := buildTenantDefaultModelConfigUpdate(tenantDefaultModelConfigUpdateRequest{
		DefaultImageModelConfigID: ptrString("not-a-valid-id"),
	})
	if err == nil {
		t.Fatal("expected invalid image model config ID error")
	}
}

func TestModelProviderTenantFilter(t *testing.T) {
	tenantID := primitive.NewObjectID()
	providerID := primitive.NewObjectID()

	filter := modelProviderTenantFilter(tenantID, providerID)
	if got, ok := filter["_id"].(primitive.ObjectID); !ok || got != providerID {
		t.Fatalf("expected provider _id filter %s, got %v", providerID.Hex(), filter["_id"])
	}
	if got, ok := filter["tenantId"].(primitive.ObjectID); !ok || got != tenantID {
		t.Fatalf("expected tenantId filter %s, got %v", tenantID.Hex(), filter["tenantId"])
	}
}

func ptrString(value string) *string {
	return &value
}

func TestClearTenantDefaultsForModelIDsUpdate_EmptyModelIDs(t *testing.T) {
	update := clearTenantDefaultsForModelIDsUpdate(nil)
	if update != nil {
		t.Fatalf("expected nil update for no model IDs, got %v", update)
	}
}

func TestClearTenantDefaultsForModelIDsUpdate_MatchesAllDefaultFields(t *testing.T) {
	modelID := primitive.NewObjectID()
	otherID := primitive.NewObjectID()

	update := clearTenantDefaultsForModelIDsUpdate([]primitive.ObjectID{modelID, otherID})
	if update == nil {
		t.Fatal("expected update document")
	}

	unsetDoc, ok := update["$unset"].(bson.M)
	if !ok {
		t.Fatalf("expected $unset bson.M, got %T", update["$unset"])
	}
	for _, field := range []string{"defaultTextModelConfigId", "defaultImageModelConfigId", "defaultVideoModelConfigId"} {
		if value, ok := unsetDoc[field]; !ok || value != "" {
			t.Fatalf("expected %s to be unset, got %v", field, unsetDoc[field])
		}
	}

	orClauses, ok := update["$or"].([]bson.M)
	if !ok {
		t.Fatalf("expected $or clauses, got %T", update["$or"])
	}
	if len(orClauses) != 3 {
		t.Fatalf("expected 3 default-field clauses, got %d", len(orClauses))
	}

	for _, field := range []string{"defaultTextModelConfigId", "defaultImageModelConfigId", "defaultVideoModelConfigId"} {
		found := false
		for _, clause := range orClauses {
			if condition, ok := clause[field].(bson.M); ok {
				if ids, ok := condition["$in"].([]primitive.ObjectID); ok && len(ids) == 2 && ids[0] == modelID && ids[1] == otherID {
					found = true
					break
				}
			}
		}
		if !found {
			t.Fatalf("expected clause for %s with both model IDs", field)
		}
	}
}

func TestDeleteModelClearsDefaultsAfterDelete(t *testing.T) {
	events := make([]string, 0, 2)

	deleted, err := deleteModelWithCleanup(
		func() (int64, error) {
			events = append(events, "delete")
			return 1, nil
		},
		func() error {
			events = append(events, "defaults")
			return nil
		},
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !deleted {
		t.Fatal("expected deleteModelWithCleanup to report deletion")
	}
	if strings.Join(events, ",") != "delete,defaults" {
		t.Fatalf("expected delete/defaults order, got %v", events)
	}
}

func TestDeleteModelDoesNotClearDefaultsWhenDeleteMatchesNothing(t *testing.T) {
	defaultsCalled := false

	deleted, err := deleteModelWithCleanup(
		func() (int64, error) {
			return 0, nil
		},
		func() error {
			defaultsCalled = true
			return nil
		},
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if deleted {
		t.Fatal("expected deleteModelWithCleanup to report no deletion")
	}
	if defaultsCalled {
		t.Fatal("expected defaults cleanup skipped when delete matched nothing")
	}
}

func TestDeleteProviderRemovesModelsBeforeProvider(t *testing.T) {
	events := make([]string, 0, 4)
	deleteProvider := func() error {
		events = append(events, "provider")
		return nil
	}
	deleteModels := func() error {
		events = append(events, "models")
		return nil
	}
	clearDefaults := func() error {
		events = append(events, "defaults")
		return nil
	}

	err := deleteProviderWithCleanup(deleteModels, clearDefaults, deleteProvider)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if strings.Join(events, ",") != "models,defaults,provider" {
		t.Fatalf("expected models/defaults/provider order, got %v", events)
	}
}

func TestDeleteProviderDoesNotDeleteProviderWhenModelDeletionFails(t *testing.T) {
	modelErr := errors.New("delete models failed")
	providerCalled := false
	defaultsCalled := false

	err := deleteProviderWithCleanup(
		func() error { return modelErr },
		func() error {
			defaultsCalled = true
			return nil
		},
		func() error {
			providerCalled = true
			return nil
		},
	)
	if !errors.Is(err, modelErr) {
		t.Fatalf("expected model deletion error, got %v", err)
	}
	if defaultsCalled {
		t.Fatal("expected defaults cleanup not called after model deletion failure")
	}
	if providerCalled {
		t.Fatal("expected provider delete skipped after model deletion failure")
	}
}

func TestModelSettingsHandler_ListProviders_MissingTenantContext(t *testing.T) {
	handler := NewModelSettingsHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/model-providers", nil)
	rr := httptest.NewRecorder()

	handler.ListProviders(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without tenant context, got %d", rr.Code)
	}
}

func TestModelSettingsHandler_UpdateProvider_InvalidID(t *testing.T) {
	handler := NewModelSettingsHandler(nil)
	req := httptest.NewRequest(http.MethodPut, "/api/tenant/model-providers/not-a-valid-id", strings.NewReader("{}"))
	req = req.WithContext(context.WithValue(req.Context(), middleware.TenantContextKey, &models.Tenant{ID: primitive.NewObjectID()}))
	req = mux.SetURLVars(req, map[string]string{"providerId": "not-a-valid-id"})
	rr := httptest.NewRecorder()

	handler.UpdateProvider(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid provider ID, got %d", rr.Code)
	}
}

func TestModelConfigAPI_UpdateModel_RejectsMissingProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)

	providerID := primitive.NewObjectID()
	if _, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{
		ID:           providerID,
		TenantID:     tenant.ID,
		Name:         "Provider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	modelID := primitive.NewObjectID()
	if _, err := env.DB.ModelConfigs().InsertOne(context.Background(), models.ModelConfig{
		ID:          modelID,
		TenantID:    tenant.ID,
		ProviderID:  providerID,
		Name:        "GPT-4o",
		DisplayName: "GPT-4o",
		Modality:    models.ModelModalityText,
		ModelID:     "gpt-4o",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	missingProviderID := primitive.NewObjectID()
	body := strings.NewReader(`{"providerId":"` + missingProviderID.Hex() + `","name":"GPT-4o","displayName":"GPT-4o","modality":"text","modelId":"gpt-4o","enabled":true}`)
	req := env.tenantRequest(t, http.MethodPut, "/api/tenant/model-configs/"+modelID.Hex(), body, user, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d body %s", resp.StatusCode, respBody)
	}

	var got ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if got.Error != "provider not found" {
		t.Fatalf("expected provider not found error, got %q", got.Error)
	}
}

func TestModelConfigAPI_UpdateModel_RejectsCrossTenantProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	env := setupTestServer(t)
	defer env.Cleanup()
	user, tenant := createAdminEnv(t, env)
	otherTenant := testutil.CreateTestTenant(t, env.DB, "Other Tenant", user.ID, false)

	providerID := primitive.NewObjectID()
	if _, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{
		ID:           providerID,
		TenantID:     tenant.ID,
		Name:         "Provider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	modelID := primitive.NewObjectID()
	if _, err := env.DB.ModelConfigs().InsertOne(context.Background(), models.ModelConfig{
		ID:          modelID,
		TenantID:    tenant.ID,
		ProviderID:  providerID,
		Name:        "GPT-4o",
		DisplayName: "GPT-4o",
		Modality:    models.ModelModalityText,
		ModelID:     "gpt-4o",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	otherProviderID := primitive.NewObjectID()
	if _, err := env.DB.ModelProviders().InsertOne(context.Background(), models.ModelProvider{
		ID:           otherProviderID,
		TenantID:     otherTenant.ID,
		Name:         "Other Provider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk-other",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	body := strings.NewReader(`{"providerId":"` + otherProviderID.Hex() + `","name":"GPT-4o","displayName":"GPT-4o","modality":"text","modelId":"gpt-4o","enabled":true}`)
	req := env.tenantRequest(t, http.MethodPut, "/api/tenant/model-configs/"+modelID.Hex(), body, user, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d body %s", resp.StatusCode, respBody)
	}

	var got ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if got.Error != "provider not found" {
		t.Fatalf("expected provider not found error, got %q", got.Error)
	}
}

func TestPreserveProviderKey_KeepsExistingOnMaskedInput(t *testing.T) {
	existing := models.ModelProvider{APIKey: "sk-secret-1234"}

	// Masked key preview should preserve original
	result := preserveProviderKey(existing, "sk-***1234")
	if result != "sk-secret-1234" {
		t.Fatalf("expected original key preserved, got %q", result)
	}

	// Empty key should preserve original
	result = preserveProviderKey(existing, "")
	if result != "sk-secret-1234" {
		t.Fatalf("expected original key preserved on empty input, got %q", result)
	}

	// New key should replace
	result = preserveProviderKey(existing, "sk-new-key-9999")
	if result != "sk-new-key-9999" {
		t.Fatalf("expected new key, got %q", result)
	}
}

func TestParseOptionalObjectID_EmptyString(t *testing.T) {
	result, err := parseOptionalObjectID("")
	if err != nil || result != nil {
		t.Fatalf("expected nil result for empty string, got result=%v err=%v", result, err)
	}
}

func TestParseOptionalObjectID_ValidID(t *testing.T) {
	id := primitive.NewObjectID()
	result, err := parseOptionalObjectID(id.Hex())
	if err != nil || result == nil || *result != id {
		t.Fatalf("expected parsed ObjectID, got result=%v err=%v", result, err)
	}
}

func TestParseOptionalObjectID_InvalidID(t *testing.T) {
	_, err := parseOptionalObjectID("not-a-valid-id")
	if err == nil {
		t.Fatalf("expected error for invalid ObjectID string")
	}
}
