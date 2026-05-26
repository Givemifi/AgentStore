package credits

import (
	"context"
	"errors"
	"fmt"
	"time"

	"lastsaas/internal/db"
	"lastsaas/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrInsufficientCredits = errors.New("insufficient credits")
	ErrBalanceChanged      = errors.New("balance changed during deduction")
)

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
// Returns an error with status 402 if insufficient credits.
func (s *Service) DeductCredits(ctx context.Context, tenantID primitive.ObjectID, userID primitive.ObjectID, quantity int, usageType string, metadata map[string]interface{}) (int, error) {
	if quantity <= 0 {
		return 0, fmt.Errorf("quantity must be positive")
	}

	var tenant models.Tenant
	if err := s.db.Tenants().FindOne(ctx, bson.M{"_id": tenantID}).Decode(&tenant); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, fmt.Errorf("tenant not found")
		}
		return 0, fmt.Errorf("failed to fetch tenant: %w", err)
	}

	totalCredits := tenant.SubscriptionCredits + tenant.PurchasedCredits
	if totalCredits < int64(quantity) {
		return 0, ErrInsufficientCredits
	}

	subscriptionToDeduct := minInt64(tenant.SubscriptionCredits, int64(quantity))
	purchasedToDeduct := int64(quantity) - subscriptionToDeduct

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
				"subscriptionCredits": subscriptionToDeduct,
				"purchasedCredits":    purchasedToDeduct,
			}},
		)
		if rollbackErr != nil {
			return 0, fmt.Errorf("failed to record usage event: %w; rollback failed: %v", err, rollbackErr)
		}
		return 0, fmt.Errorf("failed to record usage event: %w", err)
	}

	return int(totalCredits - int64(quantity)), nil
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
