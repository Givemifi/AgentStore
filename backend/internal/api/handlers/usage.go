package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"agentstore/internal/credits"
	"agentstore/internal/db"
	"agentstore/internal/middleware"
	"agentstore/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

type UsageHandler struct {
	db      *db.MongoDB
	credits *credits.Service
}

func NewUsageHandler(database *db.MongoDB, creditsSvc *credits.Service) *UsageHandler {
	return &UsageHandler{db: database, credits: creditsSvc}
}

// RecordUsage records a usage event and deducts credits from the tenant.
func (h *UsageHandler) RecordUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenant, ok := middleware.GetTenantFromContext(ctx)
	if !ok {
		http.Error(w, `{"error":"Tenant context required"}`, http.StatusBadRequest)
		return
	}
	user, ok := middleware.GetUserFromContext(ctx)
	if !ok {
		http.Error(w, `{"error":"Not authenticated"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		Type     string                 `json:"type"`
		Quantity int                    `json:"quantity"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Type == "" {
		http.Error(w, `{"error":"Type is required"}`, http.StatusBadRequest)
		return
	}
	if req.Quantity <= 0 {
		http.Error(w, `{"error":"Quantity must be positive"}`, http.StatusBadRequest)
		return
	}
	if req.Quantity > 10000 {
		http.Error(w, `{"error":"Quantity exceeds maximum of 10000 per request"}`, http.StatusBadRequest)
		return
	}

	// Deduct credits and record the usage event via the shared credits service.
	// This uses an atomic compare-and-swap with bounded retry (no multi-document
	// transaction), so it works on standalone MongoDB and stays consistent with
	// the chat deduction path.
	metadata := make(map[string]interface{}, len(req.Metadata))
	for k, v := range req.Metadata {
		metadata[k] = v
	}

	remaining, err := h.credits.DeductCredits(ctx, tenant.ID, user.ID, req.Quantity, req.Type, metadata)
	if err != nil {
		if errors.Is(err, credits.ErrInsufficientCredits) {
			http.Error(w, `{"error":"Insufficient credits"}`, http.StatusPaymentRequired)
			return
		}
		http.Error(w, `{"error":"Failed to deduct credits"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"type":             req.Type,
		"quantity":         req.Quantity,
		"remainingCredits": remaining,
	})
}

// GetSummary returns usage summary for the current billing period.
func (h *UsageHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenant, ok := middleware.GetTenantFromContext(ctx)
	if !ok {
		http.Error(w, `{"error":"Tenant context required"}`, http.StatusBadRequest)
		return
	}

	// Determine period start: use current month start as default.
	periodStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)

	// If the tenant has a current period end, derive the period start from it.
	if tenant.CurrentPeriodEnd != nil && !tenant.CurrentPeriodEnd.IsZero() {
		// Billing period is typically one month, so period start = period end - 1 month.
		periodStart = tenant.CurrentPeriodEnd.AddDate(0, -1, 0)
	}

	// Aggregate usage by type for this period.
	pipeline := []bson.M{
		{"$match": bson.M{
			"tenantId":  tenant.ID,
			"createdAt": bson.M{"$gte": periodStart},
		}},
		{"$group": bson.M{
			"_id":           "$type",
			"totalQuantity": bson.M{"$sum": "$quantity"},
			"count":         bson.M{"$sum": 1},
		}},
		{"$sort": bson.M{"totalQuantity": -1}},
	}

	cursor, err := h.db.UsageEvents().Aggregate(ctx, pipeline)
	if err != nil {
		http.Error(w, `{"error":"Failed to aggregate usage"}`, http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	type usageSummaryItem struct {
		Type          string `json:"type" bson:"_id"`
		TotalQuantity int    `json:"totalQuantity" bson:"totalQuantity"`
		Count         int    `json:"count" bson:"count"`
	}

	var items []usageSummaryItem
	if err := cursor.All(ctx, &items); err != nil {
		http.Error(w, `{"error":"Failed to read usage data"}`, http.StatusInternalServerError)
		return
	}

	// Calculate total credits used this period.
	totalUsed := 0
	for _, item := range items {
		totalUsed += item.TotalQuantity
	}

	// Refresh tenant data for current credits.
	var currentTenant models.Tenant
	if err := h.db.Tenants().FindOne(ctx, bson.M{"_id": tenant.ID}).Decode(&currentTenant); err != nil {
		http.Error(w, `{"error":"Failed to fetch tenant"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"periodStart":         periodStart.Format(time.RFC3339),
		"usage":               items,
		"totalCreditsUsed":    totalUsed,
		"subscriptionCredits": currentTenant.SubscriptionCredits,
		"purchasedCredits":    currentTenant.PurchasedCredits,
	})
}
