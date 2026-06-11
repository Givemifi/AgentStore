package credits

import (
	"context"
	"sync"
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

// TestDeductCredits_ConcurrentDeductionsNeverGoNegative verifies the optimistic
// compare-and-swap with retry: many concurrent deductions against a balance that
// only covers some of them must succeed exactly up to the balance, never overspend,
// and never drive the balance negative.
func TestDeductCredits_ConcurrentDeductionsNeverGoNegative(t *testing.T) {
	database, cleanup := testutil.MustConnectTestDB(t)
	defer cleanup()
	testutil.CleanupCollections(t, database)

	ctx := context.Background()
	tenantID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	const startingPurchased = 50
	const goroutines = 80 // more requests than the balance can satisfy

	_, err := database.Tenants().InsertOne(ctx, models.Tenant{
		ID:               tenantID,
		Name:             "Concurrent Tenant",
		Slug:             "concurrent-tenant",
		IsActive:         true,
		PurchasedCredits: startingPurchased,
	})
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	service := NewService(database)

	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := 0
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := service.DeductCredits(ctx, tenantID, userID, 1, "agent_chat", nil); err == nil {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if successes != startingPurchased {
		t.Fatalf("expected exactly %d successful deductions, got %d", startingPurchased, successes)
	}

	var tenant models.Tenant
	if err := database.Tenants().FindOne(ctx, bson.M{"_id": tenantID}).Decode(&tenant); err != nil {
		t.Fatalf("fetch tenant: %v", err)
	}
	if tenant.PurchasedCredits != 0 {
		t.Fatalf("expected balance to land exactly at 0, got %d", tenant.PurchasedCredits)
	}
	if tenant.SubscriptionCredits < 0 || tenant.PurchasedCredits < 0 {
		t.Fatalf("balance went negative: subscription=%d purchased=%d", tenant.SubscriptionCredits, tenant.PurchasedCredits)
	}

	count, err := database.UsageEvents().CountDocuments(ctx, bson.M{"tenantId": tenantID, "type": "agent_chat"})
	if err != nil {
		t.Fatalf("count usage events: %v", err)
	}
	if count != int64(startingPurchased) {
		t.Fatalf("expected %d usage events, got %d", startingPurchased, count)
	}
}

// TestRefundDeduction_RestoresPurchasedCreditsAndRecordsEvent verifies that a
// refund returns credits to purchasedCredits and writes a compensating ledger
// entry so usage nets to zero.
func TestRefundDeduction_RestoresPurchasedCreditsAndRecordsEvent(t *testing.T) {
	database, cleanup := testutil.MustConnectTestDB(t)
	defer cleanup()
	testutil.CleanupCollections(t, database)

	ctx := context.Background()
	tenantID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	_, err := database.Tenants().InsertOne(ctx, models.Tenant{
		ID:                  tenantID,
		Name:                "Refund Tenant",
		Slug:                "refund-tenant",
		IsActive:            true,
		SubscriptionCredits: 5,
		PurchasedCredits:    5,
	})
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	service := NewService(database)

	if _, err := service.DeductCredits(ctx, tenantID, userID, 4, "agent_chat", nil); err != nil {
		t.Fatalf("deduct credits: %v", err)
	}
	if err := service.RefundDeduction(ctx, tenantID, userID, 4, "agent_chat", nil); err != nil {
		t.Fatalf("refund deduction: %v", err)
	}

	var tenant models.Tenant
	if err := database.Tenants().FindOne(ctx, bson.M{"_id": tenantID}).Decode(&tenant); err != nil {
		t.Fatalf("fetch tenant: %v", err)
	}
	// 4 came out of subscription (subscription-first), the refund of 4 lands in purchased.
	if tenant.SubscriptionCredits != 1 {
		t.Fatalf("expected subscriptionCredits 1 after deduct, got %d", tenant.SubscriptionCredits)
	}
	if tenant.PurchasedCredits != 9 {
		t.Fatalf("expected purchasedCredits 9 after refund (5 + 4), got %d", tenant.PurchasedCredits)
	}

	// One charge event and one compensating refund event.
	charge, err := database.UsageEvents().CountDocuments(ctx, bson.M{"tenantId": tenantID, "type": "agent_chat"})
	if err != nil {
		t.Fatalf("count charge events: %v", err)
	}
	if charge != 1 {
		t.Fatalf("expected 1 charge event, got %d", charge)
	}
	refund, err := database.UsageEvents().CountDocuments(ctx, bson.M{"tenantId": tenantID, "type": "agent_chat_refund"})
	if err != nil {
		t.Fatalf("count refund events: %v", err)
	}
	if refund != 1 {
		t.Fatalf("expected 1 refund event, got %d", refund)
	}
}
