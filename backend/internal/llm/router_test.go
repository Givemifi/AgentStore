package llm

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"agentstore/internal/db"
	"agentstore/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var testDB *db.MongoDB
var testDBCleanup func()

func TestMain(m *testing.M) {
	testDB, testDBCleanup = connectTestDB()
	code := m.Run()
	if testDBCleanup != nil {
		testDBCleanup()
	}
	os.Exit(code)
}

func connectTestDB() (*db.MongoDB, func()) {
	// Reuse the same pattern as the handlers package testutil
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		return nil, func() {}
	}

	// Import config loading logic inline to avoid circular dependency
	// with the testutil package (which imports models, which is fine,
	// but we want to keep the llm package self-contained for testing)
	database, err := db.NewMongoDB(uri, "agentstore_test")
	if err != nil {
		return nil, func() {}
	}

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		colls, _ := database.Database.ListCollectionNames(ctx, bson.M{})
		for _, name := range colls {
			database.Database.Collection(name).DeleteMany(ctx, bson.M{})
		}
		database.Close(ctx)
	}

	return database, cleanup
}

func cleanupCollections(t *testing.T) {
	t.Helper()
	if testDB == nil {
		t.Skip("skipping: no test database connection")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collections := []string{
		"tenants", "agents", "model_providers", "model_configs", "llm_configs",
	}
	for _, name := range collections {
		testDB.Database.Collection(name).DeleteMany(ctx, bson.M{})
	}
}

func TestRouterResolveTextModel_UsesAgentBindingBeforeTenantDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if testDB == nil {
		t.Skip("skipping: no test database connection")
	}
	cleanupCollections(t)

	ctx := context.Background()
	tenantID := primitive.NewObjectID()
	providerID := primitive.NewObjectID()
	agentModelID := primitive.NewObjectID()
	defaultModelID := primitive.NewObjectID()

	// Seed provider
	_, err := testDB.ModelProviders().InsertOne(ctx, models.ModelProvider{
		ID:           providerID,
		TenantID:     tenantID,
		Name:         "TestProvider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk-test-key",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seed agent-bound model config
	_, err = testDB.ModelConfigs().InsertOne(ctx, models.ModelConfig{
		ID:          agentModelID,
		TenantID:    tenantID,
		ProviderID:  providerID,
		Name:        "Agent Model",
		DisplayName: "Agent Model",
		Modality:    models.ModelModalityText,
		ModelID:     "agent-model",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seed tenant default model config
	_, err = testDB.ModelConfigs().InsertOne(ctx, models.ModelConfig{
		ID:          defaultModelID,
		TenantID:    tenantID,
		ProviderID:  providerID,
		Name:        "Default Model",
		DisplayName: "Default Model",
		Modality:    models.ModelModalityText,
		ModelID:     "default-model",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	tenant := models.Tenant{
		ID:                      tenantID,
		DefaultTextModelConfigID: &defaultModelID,
	}
	agent := models.Agent{
		ModelConfig: models.AgentModelConfig{
			TextModelID: &agentModelID,
		},
	}

	router := NewRouter(testDB)
	snapshot, err := router.ResolveTextModel(ctx, tenant, agent)
	if err != nil {
		t.Fatalf("resolve text model: %v", err)
	}
	if snapshot.Model != "agent-model" {
		t.Fatalf("expected agent model, got %q", snapshot.Model)
	}
	if snapshot.APIKey != "sk-test-key" {
		t.Fatalf("expected provider API key, got %q", snapshot.APIKey)
	}
}

func TestRouterResolveTextModel_FallsBackToTenantDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if testDB == nil {
		t.Skip("skipping: no test database connection")
	}
	cleanupCollections(t)

	ctx := context.Background()
	tenantID := primitive.NewObjectID()
	providerID := primitive.NewObjectID()
	defaultModelID := primitive.NewObjectID()

	// Seed provider
	_, err := testDB.ModelProviders().InsertOne(ctx, models.ModelProvider{
		ID:           providerID,
		TenantID:     tenantID,
		Name:         "TestProvider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk-default-key",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seed tenant default model config
	_, err = testDB.ModelConfigs().InsertOne(ctx, models.ModelConfig{
		ID:          defaultModelID,
		TenantID:    tenantID,
		ProviderID:  providerID,
		Name:        "Default Model",
		DisplayName: "Default Model",
		Modality:    models.ModelModalityText,
		ModelID:     "default-model",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	tenant := models.Tenant{
		ID:                      tenantID,
		DefaultTextModelConfigID: &defaultModelID,
	}
	agent := models.Agent{
		ModelConfig: models.AgentModelConfig{}, // no agent binding
	}

	router := NewRouter(testDB)
	snapshot, err := router.ResolveTextModel(ctx, tenant, agent)
	if err != nil {
		t.Fatalf("resolve text model: %v", err)
	}
	if snapshot.Model != "default-model" {
		t.Fatalf("expected default model, got %q", snapshot.Model)
	}
}

func TestRouterResolveTextModel_FallsBackToLegacyLLMConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if testDB == nil {
		t.Skip("skipping: no test database connection")
	}
	cleanupCollections(t)

	ctx := context.Background()
	tenantID := primitive.NewObjectID()

	// Seed legacy LLMConfig
	_, err := testDB.LLMConfigs().InsertOne(ctx, models.LLMConfig{
		Key:       models.DefaultLLMConfigKey,
		APIKey:    "legacy-key",
		BaseURL:   "https://legacy.example.com/v1",
		Model:     "legacy-model",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	tenant := models.Tenant{ID: tenantID} // no defaults
	agent := models.Agent{ModelConfig: models.AgentModelConfig{}}

	router := NewRouter(testDB)
	snapshot, err := router.ResolveTextModel(ctx, tenant, agent)
	if err != nil {
		t.Fatalf("resolve legacy text model: %v", err)
	}
	if snapshot.Model != "legacy-model" {
		t.Fatalf("expected legacy model, got %q", snapshot.Model)
	}
	if snapshot.APIKey != "legacy-key" {
		t.Fatalf("expected legacy API key, got %q", snapshot.APIKey)
	}
}

func TestRouterResolveTextModel_ErrModelNotConfigured(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if testDB == nil {
		t.Skip("skipping: no test database connection")
	}
	cleanupCollections(t)

	ctx := context.Background()
	tenantID := primitive.NewObjectID()

	tenant := models.Tenant{ID: tenantID}
	agent := models.Agent{ModelConfig: models.AgentModelConfig{}}

	router := NewRouter(testDB)
	_, err := router.ResolveTextModel(ctx, tenant, agent)
	if err != ErrModelNotConfigured {
		t.Fatalf("expected ErrModelNotConfigured, got %v", err)
	}
}

func TestRouterResolveTextModel_RejectsUnsupportedAgentProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if testDB == nil {
		t.Skip("skipping: no test database connection")
	}
	cleanupCollections(t)

	ctx := context.Background()
	tenantID := primitive.NewObjectID()
	providerID := primitive.NewObjectID()
	agentModelID := primitive.NewObjectID()

	// Seed anthropic provider (unsupported type)
	_, err := testDB.ModelProviders().InsertOne(ctx, models.ModelProvider{
		ID:           providerID,
		TenantID:     tenantID,
		Name:         "AnthropicProvider",
		ProviderType: models.ProviderTypeAnthropic,
		BaseURL:      "https://api.anthropic.com/v1",
		APIKey:       "sk-ant-key",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seed agent-bound model config pointing at the unsupported provider
	_, err = testDB.ModelConfigs().InsertOne(ctx, models.ModelConfig{
		ID:          agentModelID,
		TenantID:    tenantID,
		ProviderID:  providerID,
		Name:        "Agent Anthropic Model",
		DisplayName: "Agent Anthropic Model",
		Modality:    models.ModelModalityText,
		ModelID:     "claude-3-opus",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seed a legacy LLMConfig so fallback would succeed if we let it
	_, err = testDB.LLMConfigs().InsertOne(ctx, models.LLMConfig{
		Key:       models.DefaultLLMConfigKey,
		APIKey:    "legacy-key",
		BaseURL:   "https://legacy.example.com/v1",
		Model:     "legacy-model",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	agent := models.Agent{
		ModelConfig: models.AgentModelConfig{
			TextModelID: &agentModelID,
		},
	}
	tenant := models.Tenant{ID: tenantID}

	router := NewRouter(testDB)
	_, err = router.ResolveTextModel(ctx, tenant, agent)
	if !errors.Is(err, ErrProviderTypeUnsupported) {
		t.Fatalf("expected ErrProviderTypeUnsupported, got %v", err)
	}
}

func TestRouterResolveTextModel_RejectsUnsupportedTenantDefaultProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if testDB == nil {
		t.Skip("skipping: no test database connection")
	}
	cleanupCollections(t)

	ctx := context.Background()
	tenantID := primitive.NewObjectID()
	providerID := primitive.NewObjectID()
	defaultModelID := primitive.NewObjectID()

	// Seed gemini provider (unsupported type)
	_, err := testDB.ModelProviders().InsertOne(ctx, models.ModelProvider{
		ID:           providerID,
		TenantID:     tenantID,
		Name:         "GeminiProvider",
		ProviderType: models.ProviderTypeGemini,
		BaseURL:      "https://generativelanguage.googleapis.com/v1",
		APIKey:       "gemini-key",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seed tenant default model config pointing at the unsupported provider
	_, err = testDB.ModelConfigs().InsertOne(ctx, models.ModelConfig{
		ID:          defaultModelID,
		TenantID:    tenantID,
		ProviderID:  providerID,
		Name:        "Tenant Default Gemini Model",
		DisplayName: "Tenant Default Gemini Model",
		Modality:    models.ModelModalityText,
		ModelID:     "gemini-pro",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	agent := models.Agent{ModelConfig: models.AgentModelConfig{}} // no agent binding
	tenant := models.Tenant{
		ID:                      tenantID,
		DefaultTextModelConfigID: &defaultModelID,
	}

	router := NewRouter(testDB)
	_, err = router.ResolveTextModel(ctx, tenant, agent)
	if !errors.Is(err, ErrProviderTypeUnsupported) {
		t.Fatalf("expected ErrProviderTypeUnsupported, got %v", err)
	}
}

func TestRouterResolveTextModel_UnsupportedAgentProviderDoesNotUseSupportedTenantDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if testDB == nil {
		t.Skip("skipping: no test database connection")
	}
	cleanupCollections(t)

	ctx := context.Background()
	tenantID := primitive.NewObjectID()

	// Unsupported Anthropic provider + model config bound to agent
	unsupportedProviderID := primitive.NewObjectID()
	agentModelID := primitive.NewObjectID()

	_, err := testDB.ModelProviders().InsertOne(ctx, models.ModelProvider{
		ID:           unsupportedProviderID,
		TenantID:     tenantID,
		Name:         "AnthropicProvider",
		ProviderType: models.ProviderTypeAnthropic,
		BaseURL:      "https://api.anthropic.com/v1",
		APIKey:       "sk-ant-key",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = testDB.ModelConfigs().InsertOne(ctx, models.ModelConfig{
		ID:          agentModelID,
		TenantID:    tenantID,
		ProviderID:  unsupportedProviderID,
		Name:        "Agent Anthropic Model",
		DisplayName: "Agent Anthropic Model",
		Modality:    models.ModelModalityText,
		ModelID:     "claude-3-opus",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Supported OpenAI-compatible provider + model config set as tenant default
	supportedProviderID := primitive.NewObjectID()
	defaultModelID := primitive.NewObjectID()

	_, err = testDB.ModelProviders().InsertOne(ctx, models.ModelProvider{
		ID:           supportedProviderID,
		TenantID:     tenantID,
		Name:         "OpenAIProvider",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.openai.com/v1",
		APIKey:       "sk-openai-key",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = testDB.ModelConfigs().InsertOne(ctx, models.ModelConfig{
		ID:          defaultModelID,
		TenantID:    tenantID,
		ProviderID:  supportedProviderID,
		Name:        "Tenant Default OpenAI Model",
		DisplayName: "Tenant Default OpenAI Model",
		Modality:    models.ModelModalityText,
		ModelID:     "gpt-4",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	agent := models.Agent{
		ModelConfig: models.AgentModelConfig{
			TextModelID: &agentModelID,
		},
	}
	tenant := models.Tenant{
		ID:                      tenantID,
		DefaultTextModelConfigID: &defaultModelID,
	}

	router := NewRouter(testDB)
	_, err = router.ResolveTextModel(ctx, tenant, agent)
	// Must NOT fall through to the supported tenant default; the unsupported
	// agent-level provider is a hard error, not a missing document.
	if !errors.Is(err, ErrProviderTypeUnsupported) {
		t.Fatalf("expected ErrProviderTypeUnsupported, got %v", err)
	}
}

func TestRouterResolveImageModel_RejectsUnsupportedAgentProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if testDB == nil {
		t.Skip("skipping: no test database connection")
	}
	cleanupCollections(t)

	ctx := context.Background()
	tenantID := primitive.NewObjectID()
	providerID := primitive.NewObjectID()
	agentModelID := primitive.NewObjectID()

	// Seed Gemini provider (unsupported type)
	_, err := testDB.ModelProviders().InsertOne(ctx, models.ModelProvider{
		ID:           providerID,
		TenantID:     tenantID,
		Name:         "GeminiProvider",
		ProviderType: models.ProviderTypeGemini,
		BaseURL:      "https://generativelanguage.googleapis.com/v1",
		APIKey:       "gemini-key",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seed agent-bound image model config pointing at the unsupported provider
	_, err = testDB.ModelConfigs().InsertOne(ctx, models.ModelConfig{
		ID:          agentModelID,
		TenantID:    tenantID,
		ProviderID:  providerID,
		Name:        "Agent Gemini Image Model",
		DisplayName: "Agent Gemini Image Model",
		Modality:    models.ModelModalityImage,
		ModelID:     "gemini-image-gen",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	agent := models.Agent{
		ModelConfig: models.AgentModelConfig{
			ImageModelID: &agentModelID,
		},
	}
	tenant := models.Tenant{ID: tenantID}

	router := NewRouter(testDB)
	_, err = router.ResolveImageModel(ctx, tenant, agent)
	if !errors.Is(err, ErrProviderTypeUnsupported) {
		t.Fatalf("expected ErrProviderTypeUnsupported, got %v", err)
	}
}