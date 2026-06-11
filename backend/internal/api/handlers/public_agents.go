package handlers

import (
	"encoding/json"
	"net/http"

	"agentstore/internal/agents"

	"github.com/gorilla/mux"
)

// PublicAgentsHandler serves the unauthenticated public agent catalog.
// It exposes the static catalog only — no tenant-scoped agents —
// so no system prompt, model config, or internal fields are ever exposed.
type PublicAgentsHandler struct{}

func NewPublicAgentsHandler() *PublicAgentsHandler {
	return &PublicAgentsHandler{}
}

type publicCatalogAgent struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Slug             string   `json:"slug"`
	Category         string   `json:"category"`
	Description      string   `json:"description"`
	Icon             string   `json:"icon,omitempty"`
	Color            string   `json:"color,omitempty"`
	WelcomeMessage   string   `json:"welcomeMessage,omitempty"`
	SuggestedPrompts []string `json:"suggestedPrompts,omitempty"`
	CreditCost       int      `json:"creditCost"`
}

func toPublicCatalogAgent(a agents.Agent) publicCatalogAgent {
	return publicCatalogAgent{
		ID:               a.ID,
		Name:             a.Name,
		Slug:             a.ID,
		Category:         a.Category,
		Description:      a.Description,
		Icon:             a.Icon,
		Color:            a.Color,
		WelcomeMessage:   a.WelcomeMessage,
		SuggestedPrompts: append([]string(nil), a.SuggestedPrompts...),
		CreditCost:       a.CreditCost,
	}
}

// ListPublicAgents returns the public catalog agent list (no auth required).
func (h *PublicAgentsHandler) ListPublicAgents(w http.ResponseWriter, r *http.Request) {
	all := agents.GetAllAgents()
	result := make([]publicCatalogAgent, len(all))
	for i, a := range all {
		result[i] = toPublicCatalogAgent(a)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300") // 5 min cache
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"agents": result}); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// GetPublicAgent returns a single catalog agent by slug (no auth required).
func (h *PublicAgentsHandler) GetPublicAgent(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	if slug == "" {
		http.Error(w, `{"error":"slug required"}`, http.StatusBadRequest)
		return
	}

	agent, err := agents.GetAgentByID(slug)
	if err != nil || agent == nil {
		http.Error(w, `{"error":"agent not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	if err := json.NewEncoder(w).Encode(toPublicCatalogAgent(*agent)); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
