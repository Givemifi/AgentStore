package handlers

import (
	"net/http"
	"strings"

	"agentstore/internal/middleware"
	"agentstore/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type LaunchReadinessStatus string

const (
	LaunchReadinessComplete LaunchReadinessStatus = "complete"
	LaunchReadinessWarning  LaunchReadinessStatus = "warning"
	LaunchReadinessPending  LaunchReadinessStatus = "pending"
)

type LaunchReadinessItem struct {
	ID          string                `json:"id"`
	Label       string                `json:"label"`
	Status      LaunchReadinessStatus `json:"status"`
	Description string                `json:"description"`
	ActionPath  string                `json:"actionPath"`
}

type LaunchReadinessSummary struct {
	Complete int `json:"complete"`
	Warning  int `json:"warning"`
	Pending  int `json:"pending"`
	Total    int `json:"total"`
}

type LaunchReadinessResponse struct {
	Items   []LaunchReadinessItem  `json:"items"`
	Summary LaunchReadinessSummary `json:"summary"`
}

func (h *AdminHandler) GetLaunchReadiness(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusBadRequest, "Tenant context required")
		return
	}

	items := []LaunchReadinessItem{
		h.brandReadiness(r),
		h.modelReadiness(r, tenant),
		h.agentReadiness(r, tenant),
		h.creditReadiness(r),
		h.integrationReadiness("stripe", "Stripe webhook healthy", "Stripe integration is configured according to the health check.", "Configure Stripe keys and webhook handling before selling credits.", "/admin/health#integrations"),
		h.integrationReadiness("resend", "Email provider ready", "Email integration is configured according to the health check.", "Configure Resend before relying on invites, verification, and password resets.", "/admin/health#integrations"),
		h.testChatReadiness(r, tenant),
	}

	respondWithJSON(w, http.StatusOK, LaunchReadinessResponse{Items: items, Summary: summarizeLaunchReadiness(items)})
}

func summarizeLaunchReadiness(items []LaunchReadinessItem) LaunchReadinessSummary {
	summary := LaunchReadinessSummary{Total: len(items)}
	for _, item := range items {
		switch item.Status {
		case LaunchReadinessComplete:
			summary.Complete++
		case LaunchReadinessWarning:
			summary.Warning++
		default:
			summary.Pending++
		}
	}
	return summary
}

func (h *AdminHandler) brandReadiness(r *http.Request) LaunchReadinessItem {
	item := LaunchReadinessItem{ID: "brand", Label: "Brand configured", Status: LaunchReadinessPending, Description: "Review app name, logo, landing copy, dashboard copy, and auth page text before inviting users.", ActionPath: "/admin/branding"}

	var cfg models.BrandingConfig
	err := h.db.BrandingConfig().FindOne(r.Context(), bson.M{}).Decode(&cfg)
	if err == nil && brandingCustomized(cfg) {
		item.Status = LaunchReadinessComplete
		return item
	}
	if err != nil && err != mongo.ErrNoDocuments {
		item.Status = LaunchReadinessWarning
		item.Description = "Branding status could not be checked. Open branding settings and verify the public copy before launch."
		return item
	}

	count, err := h.db.BrandingAssets().CountDocuments(r.Context(), bson.M{"key": bson.M{"$in": []string{"logo", "favicon"}}})
	if err != nil {
		item.Status = LaunchReadinessWarning
		item.Description = "Branding assets status could not be checked. Open branding settings and verify the public copy before launch."
		return item
	}
	if count > 0 {
		item.Status = LaunchReadinessComplete
	}
	return item
}

func brandingCustomized(cfg models.BrandingConfig) bool {
	if strings.TrimSpace(cfg.AppName) != "" && strings.TrimSpace(cfg.AppName) != "AgentStore" {
		return true
	}
	fields := []string{cfg.Tagline, cfg.LandingTitle, cfg.LandingMeta, cfg.LandingHTML, cfg.DashboardHTML, cfg.LoginHeading, cfg.LoginSubtext, cfg.SignupHeading, cfg.SignupSubtext, cfg.CustomCSS, cfg.HeadHTML, cfg.OgImageURL}
	for _, field := range fields {
		if strings.TrimSpace(field) != "" {
			return true
		}
	}
	return false
}

func (h *AdminHandler) modelReadiness(r *http.Request, tenant *models.Tenant) LaunchReadinessItem {
	item := LaunchReadinessItem{ID: "model", Label: "Model provider connected", Status: LaunchReadinessWarning, Description: "Connect an OpenAI-compatible provider and set a default text model for chat.", ActionPath: "/settings/models"}
	if h.hasOpenAICompatibleTextModel(r, tenant.ID) || h.hasActiveLegacyLLMConfig(r) {
		item.Status = LaunchReadinessComplete
		item.Description = "A chat-capable model configuration is available."
	}
	return item
}

func (h *AdminHandler) hasOpenAICompatibleTextModel(r *http.Request, tenantID primitive.ObjectID) bool {
	cursor, err := h.db.ModelConfigs().Find(r.Context(), bson.M{"tenantId": tenantID, "modality": models.ModelModalityText, "enabled": true})
	if err != nil {
		return false
	}
	defer cursor.Close(r.Context())

	for cursor.Next(r.Context()) {
		var model models.ModelConfig
		if err := cursor.Decode(&model); err != nil {
			continue
		}
		count, err := h.db.ModelProviders().CountDocuments(r.Context(), bson.M{"_id": model.ProviderID, "tenantId": tenantID, "enabled": true, "providerType": models.ProviderTypeOpenAICompatible})
		if err == nil && count > 0 {
			return true
		}
	}
	return false
}

func (h *AdminHandler) hasActiveLegacyLLMConfig(r *http.Request) bool {
	count, err := h.db.LLMConfigs().CountDocuments(r.Context(), bson.M{"key": models.DefaultLLMConfigKey, "isActive": true, "apiKey": bson.M{"$ne": ""}, "baseURL": bson.M{"$ne": ""}, "model": bson.M{"$ne": ""}})
	return err == nil && count > 0
}

func (h *AdminHandler) agentReadiness(r *http.Request, tenant *models.Tenant) LaunchReadinessItem {
	item := LaunchReadinessItem{ID: "agent", Label: "First Agent published", Status: LaunchReadinessPending, Description: "Create and publish at least one text-chat Agent so users have something to try.", ActionPath: "/settings/agents"}
	count, err := h.db.Agents().CountDocuments(r.Context(), bson.M{"tenantId": tenant.ID, "status": models.AgentStatusPublished, "capabilities": models.AgentCapabilityTextChat})
	if err == nil && count > 0 {
		item.Status = LaunchReadinessComplete
		item.Description = "At least one text-chat Agent is published."
	}
	return item
}

func (h *AdminHandler) creditReadiness(r *http.Request) LaunchReadinessItem {
	item := LaunchReadinessItem{ID: "credits", Label: "Credit bundle or plan active", Status: LaunchReadinessWarning, Description: "Enable at least one plan or credit bundle before launch so users can continue after credits run out.", ActionPath: "/admin/plans"}
	planCount, planErr := h.db.Plans().CountDocuments(r.Context(), bson.M{"isArchived": bson.M{"$ne": true}})
	bundleCount, bundleErr := h.db.CreditBundles().CountDocuments(r.Context(), bson.M{"isActive": true})
	if (planErr == nil && planCount > 0) || (bundleErr == nil && bundleCount > 0) {
		item.Status = LaunchReadinessComplete
		item.Description = "A plan or active credit bundle is available."
	}
	return item
}

func (h *AdminHandler) integrationReadiness(name, label, healthyDescription, unhealthyDescription, actionPath string) LaunchReadinessItem {
	item := LaunchReadinessItem{ID: name, Label: label, Status: LaunchReadinessWarning, Description: unhealthyDescription, ActionPath: actionPath}
	if h.health == nil {
		return item
	}
	for _, result := range h.health.GetIntegrationStatus() {
		if result.Name == name && result.Status == models.IntegrationHealthy {
			item.Status = LaunchReadinessComplete
			item.Description = healthyDescription
			return item
		}
	}
	return item
}

func (h *AdminHandler) testChatReadiness(r *http.Request, tenant *models.Tenant) LaunchReadinessItem {
	item := LaunchReadinessItem{ID: "test-chat", Label: "Test chat passed", Status: LaunchReadinessPending, Description: "Run a real smoke chat with credits before announcing the product.", ActionPath: "/dashboard"}
	count, err := h.db.ChatMessages().CountDocuments(r.Context(), bson.M{"tenantId": tenant.ID, "role": "assistant", "status": models.ChatMessageStatusCompleted, "creditsCharged": bson.M{"$gt": 0}})
	if err == nil && count > 0 {
		item.Status = LaunchReadinessComplete
		item.Description = "A completed chat with credits charged has been recorded."
	}
	return item
}