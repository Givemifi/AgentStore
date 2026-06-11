package credits

import (
	"context"
	"errors"
	"fmt"
	"time"

	"agentstore/internal/db"
	"agentstore/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrInsufficientCredits = errors.New("insufficient credits")
	ErrBalanceChanged      = errors.New("balance changed during deduction")
)

// maxDeductAttempts bounds the optimistic-concurrency retry loop in DeductCredits.
// Each attempt re-reads the balance and retries the compare-and-swap when a
// concurrent deduction changed the balance underneath us. This keeps legitimate
// concurrent chats from being spuriously rejected with ErrBalanceChanged while
// still guaranteeing the balance can never go negative.
const maxDeductAttempts = 8

// Service handles credit operations for tenants.
type Service struct {
	db *db.MongoDB
}

// NewService creates a new credits service.
func NewService(database *db.MongoDB) *Service {
	return &Service{db: database}
}

// DeductCredits atomically deducts credits from a tenant.
// Returns remaining credits after deduction (subscriptionCredits + purchasedCredits).
// Returns ErrInsufficientCredits when the combined balance is too low.
//
// Concurrency: the common case (the charge fits entirely in one balance bucket)
// is a single atomic conditional `$inc` with a `$gte` guard — contention-free and
// never negative. Only the rare cross-bucket split (subscription covers part, the
// rest comes from purchased) falls back to an optimistic compare-and-swap on the
// exact balances, retried up to maxDeductAttempts times. No multi-document
// transaction is used, so this works on a standalone (non-replica-set) MongoDB.
func (s *Service) DeductCredits(ctx context.Context, tenantID primitive.ObjectID, userID primitive.ObjectID, quantity int, usageType string, metadata map[string]interface{}) (int, error) {
	if quantity <= 0 {
		return 0, fmt.Errorf("quantity must be positive")
	}

	var lastErr error
	for attempt := 0; attempt < maxDeductAttempts; attempt++ {
		remaining, err := s.tryDeduct(ctx, tenantID, userID, quantity, usageType, metadata)
		if err == nil {
			return remaining, nil
		}
		// Only retry when a concurrent balance change caused the CAS to miss.
		// Insufficient credits and hard errors are returned immediately.
		if !errors.Is(err, ErrBalanceChanged) {
			return 0, err
		}
		lastErr = err
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
	}
	return 0, lastErr
}

// tryDeduct performs a single deduction attempt.
//
// Fast path: if subscription credits alone, or purchased credits alone, cover the
// full charge, deduct with one atomic conditional update — no read, no contention.
// Slow path: when the charge must be split across both buckets, read the exact
// balances and compare-and-swap; a concurrent write makes the swap miss and yields
// ErrBalanceChanged so DeductCredits retries.
func (s *Service) tryDeduct(ctx context.Context, tenantID primitive.ObjectID, userID primitive.ObjectID, quantity int, usageType string, metadata map[string]interface{}) (int, error) {
	q := int64(quantity)

	// Fast path 1: subscription credits alone cover the charge (subscription-first).
	if ok, err := s.deductFromField(ctx, tenantID, "subscriptionCredits", q); err != nil {
		return 0, err
	} else if ok {
		return s.recordOrRollback(ctx, tenantID, userID, quantity, usageType, metadata, "subscriptionCredits", 0)
	}

	// Fast path 2: purchased credits alone cover the charge.
	if ok, err := s.deductFromField(ctx, tenantID, "purchasedCredits", q); err != nil {
		return 0, err
	} else if ok {
		return s.recordOrRollback(ctx, tenantID, userID, quantity, usageType, metadata, "purchasedCredits", 0)
	}

	// Slow path: neither bucket alone is enough — read and split via CAS.
	var tenant models.Tenant
	if err := s.db.Tenants().FindOne(ctx, bson.M{"_id": tenantID}).Decode(&tenant); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, fmt.Errorf("tenant not found")
		}
		return 0, fmt.Errorf("failed to fetch tenant: %w", err)
	}

	totalCredits := tenant.SubscriptionCredits + tenant.PurchasedCredits
	if totalCredits < q {
		return 0, ErrInsufficientCredits
	}

	subscriptionToDeduct := minInt64(tenant.SubscriptionCredits, q)
	purchasedToDeduct := q - subscriptionToDeduct

	update := bson.M{}
	if subscriptionToDeduct > 0 {
		update["subscriptionCredits"] = -subscriptionToDeduct
	}
	if purchasedToDeduct > 0 {
		update["purchasedCredits"] = -purchasedToDeduct
	}

	result, err := s.db.Tenants().UpdateOne(ctx,
		bson.M{
			"_id":                 tenantID,
			"subscriptionCredits": tenant.SubscriptionCredits,
			"purchasedCredits":    tenant.PurchasedCredits,
		},
		bson.M{"$inc": update},
	)
	if err != nil {
		return 0, fmt.Errorf("failed to deduct credits: %w", err)
	}
	if result.ModifiedCount != 1 {
		return 0, ErrBalanceChanged
	}

	if err := s.recordUsageOrRollback(ctx, tenantID, userID, quantity, usageType, metadata, subscriptionToDeduct, purchasedToDeduct); err != nil {
		return 0, err
	}
	return int(totalCredits - q), nil
}

// deductFromField atomically deducts q from a single balance field, but only when
// that field alone is >= q. Returns (true, nil) on success, (false, nil) when the
// field couldn't cover it (caller tries the next path), or an error on DB failure.
func (s *Service) deductFromField(ctx context.Context, tenantID primitive.ObjectID, field string, q int64) (bool, error) {
	result, err := s.db.Tenants().UpdateOne(ctx,
		bson.M{"_id": tenantID, field: bson.M{"$gte": q}},
		bson.M{"$inc": bson.M{field: -q}},
	)
	if err != nil {
		return false, fmt.Errorf("failed to deduct credits: %w", err)
	}
	return result.ModifiedCount == 1, nil
}

// recordOrRollback records the usage event after a single-field fast-path deduction,
// computing remaining credits with a follow-up read. fieldA is the field that was
// charged; fieldB/qB describe an optional second field (used only by the split path,
// here always empty for fast paths). On usage-event failure it restores the charge.
func (s *Service) recordOrRollback(ctx context.Context, tenantID, userID primitive.ObjectID, quantity int, usageType string, metadata map[string]interface{}, chargedField string, _ int64) (int, error) {
	var subRestore, purRestore int64
	if chargedField == "subscriptionCredits" {
		subRestore = int64(quantity)
	} else {
		purRestore = int64(quantity)
	}
	if err := s.recordUsageOrRollback(ctx, tenantID, userID, quantity, usageType, metadata, subRestore, purRestore); err != nil {
		return 0, err
	}
	remaining, err := s.GetRemainingCredits(ctx, tenantID)
	if err != nil {
		// Charge and ledger both succeeded; only the remaining-balance read failed.
		// Report success with a best-effort 0 rather than double-charging on retry.
		return 0, nil
	}
	return remaining, nil
}

// recordUsageOrRollback writes the usage ledger entry; on failure it restores the
// given amounts to each balance field so the charge is fully reversed.
func (s *Service) recordUsageOrRollback(ctx context.Context, tenantID, userID primitive.ObjectID, quantity int, usageType string, metadata map[string]interface{}, subDeducted, purDeducted int64) error {
	event := models.UsageEvent{
		ID:        primitive.NewObjectID(),
		TenantID:  tenantID,
		UserID:    userID,
		Type:      usageType,
		Quantity:  quantity,
		Metadata:  convertMetadata(metadata),
		CreatedAt: time.Now(),
	}
	if _, err := s.db.UsageEvents().InsertOne(ctx, event); err != nil {
		_, rollbackErr := s.db.Tenants().UpdateOne(ctx,
			bson.M{"_id": tenantID},
			bson.M{"$inc": bson.M{
				"subscriptionCredits": subDeducted,
				"purchasedCredits":    purDeducted,
			}},
		)
		if rollbackErr != nil {
			return fmt.Errorf("failed to record usage event: %w; rollback failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("failed to record usage event: %w", err)
	}
	return nil
}

// CheckSufficientCredits checks if a tenant has sufficient credits without deducting.
func (s *Service) CheckSufficientCredits(ctx context.Context, tenantID primitive.ObjectID, quantity int) (bool, error) {
	var tenant models.Tenant
	err := s.db.Tenants().FindOne(ctx, bson.M{"_id": tenantID}).Decode(&tenant)
	if err != nil {
		return false, fmt.Errorf("failed to fetch tenant: %w", err)
	}

	totalCredits := tenant.SubscriptionCredits + tenant.PurchasedCredits
	return totalCredits >= int64(quantity), nil
}

// GetRemainingCredits returns the total remaining credits for a tenant.
func (s *Service) GetRemainingCredits(ctx context.Context, tenantID primitive.ObjectID) (int, error) {
	var tenant models.Tenant
	err := s.db.Tenants().FindOne(ctx, bson.M{"_id": tenantID}).Decode(&tenant)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch tenant: %w", err)
	}

	return int(tenant.SubscriptionCredits + tenant.PurchasedCredits), nil
}

// GrantPurchasedCredits adds credits to a tenant's purchasedCredits balance.
// Used for referral rewards, bonuses, and admin grants. Idempotency must be ensured by the caller.
func (s *Service) GrantPurchasedCredits(ctx context.Context, tenantID primitive.ObjectID, amount int64, reason string) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	_, err := s.db.Tenants().UpdateOne(ctx,
		bson.M{"_id": tenantID},
		bson.M{"$inc": bson.M{"purchasedCredits": amount}},
	)
	if err != nil {
		return fmt.Errorf("GrantPurchasedCredits(%s, %d): %w", reason, amount, err)
	}
	return nil
}

// RefundDeduction reverses a prior DeductCredits charge for a tenant, restoring
// the credits and recording a compensating usage event. It is used when a charge
// was taken but the work it paid for could not be delivered (e.g. the LLM call
// failed or the client disconnected after credits were already deducted).
//
// Refunds always go back to purchasedCredits to avoid inflating subscription
// balances beyond a plan's monthly allowance. The compensating event keeps the
// usage ledger balanced so reconciliation nets to zero.
func (s *Service) RefundDeduction(ctx context.Context, tenantID primitive.ObjectID, userID primitive.ObjectID, quantity int, usageType string, metadata map[string]interface{}) error {
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if _, err := s.db.Tenants().UpdateOne(ctx,
		bson.M{"_id": tenantID},
		bson.M{"$inc": bson.M{"purchasedCredits": int64(quantity)}},
	); err != nil {
		return fmt.Errorf("refund deduction: %w", err)
	}

	event := models.UsageEvent{
		ID:        primitive.NewObjectID(),
		TenantID:  tenantID,
		UserID:    userID,
		Type:      usageType + "_refund",
		Quantity:  -quantity,
		Metadata:  convertMetadata(metadata),
		CreatedAt: time.Now(),
	}
	if _, err := s.db.UsageEvents().InsertOne(ctx, event); err != nil {
		// The credit was already restored; a missing ledger entry is logged by
		// the caller but must not surface as a refund failure to the user.
		return fmt.Errorf("refund recorded but usage event failed: %w", err)
	}
	return nil
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// convertMetadata converts map[string]interface{} to map[string]string for UsageEvent.
func convertMetadata(metadata map[string]interface{}) map[string]string {
	if metadata == nil {
		return nil
	}
	result := make(map[string]string)
	for k, v := range metadata {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}
