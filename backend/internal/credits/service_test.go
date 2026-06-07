package credits

import (
	"context"
	"testing"

	"agentstore/internal/models"
	"agentstore/internal/testutil"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestDeductCredits_UsesCombinedBalancesWithSubscriptionFirst(t *testing.T) {
	database, cleanup := testutil.MustConnectTestDB(t)
	defer cleanup()
	testutil.CleanupCollections(t, database)

	ctx := context.Background()
	tenantID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	_, err := database.Tenants().InsertOne(ctx, models.Tenant{
		ID:                  tenantID,
		Name:                "Credits Tenant",
		Slug:                "credits-tenant",
		IsActive:            true,
		SubscriptionCredits: 3,
		PurchasedCredits:    4,
	})
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	service := NewService(database)

	remaining, err := service.DeductCredits(ctx, tenantID, userID, 5, "agent_chat", map[string]interface{}{"source": "test"})
	if err != nil {
		t.Fatalf("deduct credits: %v", err)
	}
	if remaining != 2 {
		t.Fatalf("expected 2 remaining credits, got %d", remaining)
	}

	var tenant models.Tenant
	if err := database.Tenants().FindOne(ctx, bson.M{"_id": tenantID}).Decode(&tenant); err != nil {
		t.Fatalf("fetch tenant: %v", err)
	}
	if tenant.SubscriptionCredits != 0 {
		t.Fatalf("expected subscription credits to be spent first, got %d", tenant.SubscriptionCredits)
	}
	if tenant.PurchasedCredits != 2 {
		t.Fatalf("expected purchased credits to cover remainder, got %d", tenant.PurchasedCredits)
	}

	count, err := database.UsageEvents().CountDocuments(ctx, bson.M{"tenantId": tenantID, "userId": userID, "type": "agent_chat"})
	if err != nil {
		t.Fatalf("count usage events: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one usage event, got %d", count)
	}
}

func TestDeductCredits_RejectsWhenCombinedBalanceIsInsufficient(t *testing.T) {
	database, cleanup := testutil.MustConnectTestDB(t)
	defer cleanup()
	testutil.CleanupCollections(t, database)

	ctx := context.Background()
	tenantID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	_, err := database.Tenants().InsertOne(ctx, models.Tenant{
		ID:                  tenantID,
		Name:                "Low Credits Tenant",
		Slug:                "low-credits-tenant",
		IsActive:            true,
		SubscriptionCredits: 2,
		PurchasedCredits:    1,
	})
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	service := NewService(database)

	if _, err := service.DeductCredits(ctx, tenantID, userID, 4, "agent_chat", nil); err == nil {
		t.Fatal("expected insufficient credits error")
	}

	var tenant models.Tenant
	if err := database.Tenants().FindOne(ctx, bson.M{"_id": tenantID}).Decode(&tenant); err != nil {
		t.Fatalf("fetch tenant: %v", err)
	}
	if tenant.SubscriptionCredits != 2 || tenant.PurchasedCredits != 1 {
		t.Fatalf("expected balances to remain unchanged, got subscription=%d purchased=%d", tenant.SubscriptionCredits, tenant.PurchasedCredits)
	}
}
