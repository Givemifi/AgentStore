// Package bootstrap provides shared first-run initialization logic used by both
// the CLI setup command and the HTTP setup wizard endpoint.
package bootstrap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agentstore/internal/auth"
	"agentstore/internal/db"
	"agentstore/internal/models"
	"agentstore/internal/validation"
	"agentstore/internal/version"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// ErrAlreadyInitialized is returned when the system has already been set up.
var ErrAlreadyInitialized = fmt.Errorf("system is already initialized")

// SetupInput holds the fields collected during first-run setup.
type SetupInput struct {
	OrgName     string // display name of the root organisation
	DisplayName string // owner's display name
	Email       string // owner's email (normalised to lowercase)
	Password    string // plain-text password (will be hashed)
}

// InitializeSystem creates the root tenant, owner user, owner membership, and
// SystemConfig.Initialized record in a single transactional sequence with
// rollback on partial failure.  It returns the created owner User on success.
//
// The function is idempotent in the sense that it returns ErrAlreadyInitialized
// immediately when SystemConfig already has Initialized = true, so it is safe
// to call speculatively.
func InitializeSystem(ctx context.Context, database *db.MongoDB, in SetupInput) (*models.User, error) {
	// Validate inputs
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	in.OrgName = strings.TrimSpace(in.OrgName)
	in.DisplayName = strings.TrimSpace(in.DisplayName)

	if in.OrgName == "" || in.DisplayName == "" || in.Email == "" || in.Password == "" {
		return nil, fmt.Errorf("all fields are required")
	}

	// Guard: bail out if already initialised
	var sys models.SystemConfig
	if err := database.SystemConfig().FindOne(ctx, bson.M{}).Decode(&sys); err == nil && sys.Initialized {
		return nil, ErrAlreadyInitialized
	}

	// Validate password strength
	passwordService := auth.NewPasswordService()
	if err := passwordService.ValidatePasswordStrength(in.Password); err != nil {
		return nil, fmt.Errorf("password too weak: %w", err)
	}

	passwordHash, err := passwordService.HashPassword(in.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()

	// Create root tenant
	tenant := models.Tenant{
		ID:        primitive.NewObjectID(),
		Name:      in.OrgName,
		Slug:      "root",
		IsRoot:    true,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := validation.Validate(&tenant); err != nil {
		return nil, fmt.Errorf("tenant validation failed: %w", err)
	}
	if _, err := database.Tenants().InsertOne(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to create root tenant: %w", err)
	}

	// Create owner user
	user := models.User{
		ID:            primitive.NewObjectID(),
		Email:         in.Email,
		DisplayName:   in.DisplayName,
		PasswordHash:  passwordHash,
		AuthMethods:   []models.AuthMethod{models.AuthMethodPassword},
		EmailVerified: true,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := validation.Validate(&user); err != nil {
		database.Tenants().DeleteOne(ctx, bson.M{"_id": tenant.ID}) //nolint:errcheck
		return nil, fmt.Errorf("user validation failed: %w", err)
	}
	if _, err := database.Users().InsertOne(ctx, user); err != nil {
		database.Tenants().DeleteOne(ctx, bson.M{"_id": tenant.ID}) //nolint:errcheck
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create owner membership
	membership := models.TenantMembership{
		ID:        primitive.NewObjectID(),
		UserID:    user.ID,
		TenantID:  tenant.ID,
		Role:      models.RoleOwner,
		JoinedAt:  now,
		UpdatedAt: now,
	}
	if err := validation.Validate(&membership); err != nil {
		database.Users().DeleteOne(ctx, bson.M{"_id": user.ID})         //nolint:errcheck
		database.Tenants().DeleteOne(ctx, bson.M{"_id": tenant.ID})     //nolint:errcheck
		return nil, fmt.Errorf("membership validation failed: %w", err)
	}
	if _, err := database.TenantMemberships().InsertOne(ctx, membership); err != nil {
		database.Users().DeleteOne(ctx, bson.M{"_id": user.ID})      //nolint:errcheck
		database.Tenants().DeleteOne(ctx, bson.M{"_id": tenant.ID})  //nolint:errcheck
		return nil, fmt.Errorf("failed to create membership: %w", err)
	}

	// Mark system as initialised
	sysConfig := models.SystemConfig{
		ID:            primitive.NewObjectID(),
		Initialized:   true,
		InitializedAt: &now,
		InitializedBy: &user.ID,
		Version:       version.Current,
	}
	if _, err := database.SystemConfig().InsertOne(ctx, sysConfig); err != nil {
		database.TenantMemberships().DeleteOne(ctx, bson.M{"_id": membership.ID}) //nolint:errcheck
		database.Users().DeleteOne(ctx, bson.M{"_id": user.ID})                    //nolint:errcheck
		database.Tenants().DeleteOne(ctx, bson.M{"_id": tenant.ID})                //nolint:errcheck
		return nil, fmt.Errorf("failed to mark system as initialized: %w", err)
	}

	// Send welcome in-app message (best-effort)
	welcomeMsg := models.Message{
		ID:        primitive.NewObjectID(),
		UserID:    user.ID,
		Subject:   "Welcome to AgentStore v" + version.Current,
		Body:      "Your system has been initialized. Welcome to AgentStore!",
		IsSystem:  true,
		Read:      false,
		CreatedAt: now,
	}
	database.Messages().InsertOne(ctx, welcomeMsg) //nolint:errcheck

	return &user, nil
}

// IsInitialized reports whether the system has already been set up.
func IsInitialized(ctx context.Context, database *db.MongoDB) (bool, error) {
	var sys models.SystemConfig
	err := database.SystemConfig().FindOne(ctx, bson.M{}).Decode(&sys)
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return sys.Initialized, nil
}
