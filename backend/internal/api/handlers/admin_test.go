package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"agentstore/internal/models"
	"agentstore/internal/testutil"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestIntegration_AdminDashboard(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	req := env.adminRequest(t, "GET", "/api/admin/dashboard", nil, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}
}

func TestIntegration_AdminListTenants(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	for i := 0; i < 3; i++ {
		otherUser := testutil.CreateTestUser(t, env.DB, "tenant"+string(rune('a'+i))+"@test.com", "StrongP@ss1!", "Tenant User")
		testutil.CreateTestTenant(t, env.DB, "Tenant "+string(rune('A'+i)), otherUser.ID, false)
	}

	req := env.adminRequest(t, "GET", "/api/admin/tenants", nil, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}
}

func TestIntegration_AdminGetTenant(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, rootTenant := createAdminEnv(t, env)

	req := env.adminRequest(t, "GET", "/api/admin/tenants/"+rootTenant.ID.Hex(), nil, admin, rootTenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}
}

func TestIntegration_AdminGetTenantNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	fakeID := primitive.NewObjectID().Hex()
	req := env.adminRequest(t, "GET", "/api/admin/tenants/"+fakeID, nil, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestIntegration_AdminListUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	testutil.CreateTestUser(t, env.DB, "user1@test.com", "StrongP@ss1!", "User One")
	testutil.CreateTestUser(t, env.DB, "user2@test.com", "StrongP@ss1!", "User Two")

	req := env.adminRequest(t, "GET", "/api/admin/users", nil, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}
}

func TestIntegration_AdminGetUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	user := testutil.CreateTestUser(t, env.DB, "getuser@test.com", "StrongP@ss1!", "Get User")

	req := env.adminRequest(t, "GET", "/api/admin/users/"+user.ID.Hex(), nil, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}
}

func TestIntegration_AdminGetUserNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	fakeID := primitive.NewObjectID().Hex()
	req := env.adminRequest(t, "GET", "/api/admin/users/"+fakeID, nil, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestIntegration_AdminUpdateTenantStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, rootTenant := createAdminEnv(t, env)

	otherUser := testutil.CreateTestUser(t, env.DB, "other@test.com", "StrongP@ss1!", "Other User")
	otherTenant := testutil.CreateTestTenant(t, env.DB, "Other Tenant", otherUser.ID, false)

	body := strings.NewReader(`{"isActive":false}`)
	req := env.adminRequest(t, "PATCH", "/api/admin/tenants/"+otherTenant.ID.Hex()+"/status", body, admin, rootTenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}

	var updated models.Tenant
	env.DB.Tenants().FindOne(context.Background(), bson.M{"_id": otherTenant.ID}).Decode(&updated)
	if updated.IsActive {
		t.Error("expected tenant to be deactivated")
	}
}

func TestIntegration_AdminUpdateUserStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	user := testutil.CreateTestUser(t, env.DB, "deactivate@test.com", "StrongP@ss1!", "Deactivate User")

	body := strings.NewReader(`{"isActive":false}`)
	req := env.adminRequest(t, "PATCH", "/api/admin/users/"+user.ID.Hex()+"/status", body, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}

	var updated models.User
	env.DB.Users().FindOne(context.Background(), bson.M{"_id": user.ID}).Decode(&updated)
	if updated.IsActive {
		t.Error("expected user to be deactivated")
	}
}

func TestIntegration_AdminRequiresRootTenant(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	testutil.MarkSystemInitialized(t, env.DB)

	user := testutil.CreateTestUser(t, env.DB, "nonadmin@test.com", "StrongP@ss1!", "Non Admin")
	nonRootTenant := testutil.CreateTestTenant(t, env.DB, "Non Root", user.ID, false)

	req := env.adminRequest(t, "GET", "/api/admin/dashboard", nil, user, nonRootTenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestIntegration_AdminRequiresAdminRole(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	testutil.MarkSystemInitialized(t, env.DB)

	owner := testutil.CreateTestUser(t, env.DB, "owner@test.com", "StrongP@ss1!", "Owner")
	rootTenant := testutil.CreateTestTenant(t, env.DB, "Root Tenant", owner.ID, true)

	regularUser := testutil.CreateTestUser(t, env.DB, "regular@test.com", "StrongP@ss1!", "Regular User")
	membership := models.TenantMembership{
		ID:        primitive.NewObjectID(),
		UserID:    regularUser.ID,
		TenantID:  rootTenant.ID,
		Role:      models.RoleUser,
		JoinedAt:  time.Now(),
		UpdatedAt: time.Now(),
	}
	env.DB.TenantMemberships().InsertOne(context.Background(), membership)

	req := env.adminRequest(t, "GET", "/api/admin/dashboard", nil, regularUser, rootTenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestIntegration_AdminSearchTenants(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	user := testutil.CreateTestUser(t, env.DB, "search@test.com", "StrongP@ss1!", "Search User")
	testutil.CreateTestTenant(t, env.DB, "Findable Corp", user.ID, false)

	req := env.adminRequest(t, "GET", "/api/admin/tenants?search=Findable", nil, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}
}

func TestIntegration_AdminSearchUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	testutil.CreateTestUser(t, env.DB, "findme@test.com", "StrongP@ss1!", "Findme Person")

	req := env.adminRequest(t, "GET", "/api/admin/users?search=findme", nil, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}
}

// --- Root Members tests ---

func TestIntegration_AdminListRootMembers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	// Add a second member
	user2 := testutil.CreateTestUser(t, env.DB, "member2@test.com", "StrongP@ss1!", "Member Two")
	testutil.CreateTestMembership(t, env.DB, user2.ID, tenant.ID, models.RoleUser)

	req := env.adminRequest(t, "GET", "/api/admin/members", nil, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}

	var result struct {
		Members     []json.RawMessage `json:"members"`
		Invitations []json.RawMessage `json:"invitations"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Members) < 2 {
		t.Errorf("expected at least 2 members, got %d", len(result.Members))
	}
}

func TestIntegration_AdminInviteRootMember(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	body := strings.NewReader(`{"email":"newinvite@test.com","role":"user"}`)
	req := env.adminRequest(t, "POST", "/api/admin/members/invite", body, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}

	// Verify invitation exists in DB
	count, _ := env.DB.Invitations().CountDocuments(context.Background(), bson.M{
		"tenantId": tenant.ID,
		"email":    "newinvite@test.com",
		"status":   models.InvitationPending,
	})
	if count != 1 {
		t.Errorf("expected 1 invitation in DB, got %d", count)
	}
}

func TestIntegration_AdminInviteRootMemberDuplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	// First invite
	body := strings.NewReader(`{"email":"dup@test.com","role":"user"}`)
	req := env.adminRequest(t, "POST", "/api/admin/members/invite", body, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 on first invite, got %d", resp.StatusCode)
	}

	// Duplicate invite
	body = strings.NewReader(`{"email":"dup@test.com","role":"user"}`)
	req = env.adminRequest(t, "POST", "/api/admin/members/invite", body, admin, tenant.ID.Hex())
	resp, err = env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}
}

func TestIntegration_AdminRemoveRootMember(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	// Add a user to remove
	user2 := testutil.CreateTestUser(t, env.DB, "removeme@test.com", "StrongP@ss1!", "Remove Me")
	testutil.CreateTestMembership(t, env.DB, user2.ID, tenant.ID, models.RoleUser)

	req := env.adminRequest(t, "DELETE", "/api/admin/members/"+user2.ID.Hex(), nil, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}

	// Verify membership gone
	count, _ := env.DB.TenantMemberships().CountDocuments(context.Background(), bson.M{
		"userId":   user2.ID,
		"tenantId": tenant.ID,
	})
	if count != 0 {
		t.Errorf("expected membership to be deleted, found %d", count)
	}
}

func TestIntegration_AdminRemoveRootMemberSelf(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	req := env.adminRequest(t, "DELETE", "/api/admin/members/"+admin.ID.Hex(), nil, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}
}

func TestIntegration_AdminChangeRootMemberRole(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	// Add a user to change role
	user2 := testutil.CreateTestUser(t, env.DB, "rolechange@test.com", "StrongP@ss1!", "Role Change")
	testutil.CreateTestMembership(t, env.DB, user2.ID, tenant.ID, models.RoleUser)

	body := strings.NewReader(`{"role":"admin"}`)
	req := env.adminRequest(t, "PATCH", "/api/admin/members/"+user2.ID.Hex()+"/role", body, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}

	// Verify role changed in DB
	var membership models.TenantMembership
	env.DB.TenantMemberships().FindOne(context.Background(), bson.M{
		"userId":   user2.ID,
		"tenantId": tenant.ID,
	}).Decode(&membership)
	if membership.Role != models.RoleAdmin {
		t.Errorf("expected role admin, got %s", membership.Role)
	}
}

func TestIntegration_AdminCancelRootInvitation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	invitation := testutil.CreateTestInvitation(t, env.DB, "cancel@test.com", tenant.ID, admin.ID, models.RoleUser)

	req := env.adminRequest(t, "DELETE", "/api/admin/members/invitations/"+invitation.ID.Hex(), nil, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}

	// Verify invitation deleted
	count, _ := env.DB.Invitations().CountDocuments(context.Background(), bson.M{"_id": invitation.ID})
	if count != 0 {
		t.Errorf("expected invitation to be deleted, found %d", count)
	}
}

func TestIntegration_AdminLaunchReadiness_UnconfiguredState(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)

	req := env.adminRequest(t, "GET", "/api/admin/launch-readiness", nil, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}

	var got LaunchReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Summary.Total != 7 {
		t.Fatalf("expected 7 readiness items, got %d", got.Summary.Total)
	}
	assertReadinessStatus(t, got, "model", LaunchReadinessWarning)
	assertReadinessStatus(t, got, "agent", LaunchReadinessPending)
	assertReadinessStatus(t, got, "test-chat", LaunchReadinessPending)
}

func TestIntegration_AdminLaunchReadiness_ReadySignals(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)
	ctx := context.Background()
	now := time.Now()

	_, err := env.DB.BrandingConfig().InsertOne(ctx, models.BrandingConfig{
		ID:            primitive.NewObjectID(),
		AppName:       "AgentStore Launch",
		LogoMode:      "text",
		DashboardHTML: "<p>Launch dashboard copy</p>",
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		t.Fatalf("seed branding: %v", err)
	}

	providerID := primitive.NewObjectID()
	modelID := primitive.NewObjectID()
	_, err = env.DB.ModelProviders().InsertOne(ctx, models.ModelProvider{
		ID:           providerID,
		TenantID:     tenant.ID,
		Name:         "OpenAI Compatible",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk-test-key",
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	_, err = env.DB.ModelConfigs().InsertOne(ctx, models.ModelConfig{
		ID:          modelID,
		TenantID:    tenant.ID,
		ProviderID:  providerID,
		Name:        "Launch Text Model",
		DisplayName: "Launch Text Model",
		Modality:    models.ModelModalityText,
		ModelID:     "launch-model",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed model: %v", err)
	}

	_, err = env.DB.Agents().InsertOne(ctx, models.Agent{
		ID:           primitive.NewObjectID(),
		TenantID:     tenant.ID,
		Name:         "Launch Agent",
		Slug:         "launch-agent",
		Category:     "Support",
		Description:  "Launch-ready support agent.",
		Status:       models.AgentStatusPublished,
		Visibility:   models.AgentVisibilityPublic,
		SystemPrompt: "Help users launch.",
		Capabilities: []models.AgentCapability{models.AgentCapabilityTextChat},
		CreditCost:   models.AgentCreditCost{TextMessageCredits: 1},
		CreatedBy:    admin.ID,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("seed agent: %v", err)
	}

	_, err = env.DB.CreditBundles().InsertOne(ctx, models.CreditBundle{
		ID:         primitive.NewObjectID(),
		Name:       "Launch Credits",
		Credits:    100,
		PriceCents: 1000,
		IsActive:   true,
		SortOrder:  1,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err != nil {
		t.Fatalf("seed bundle: %v", err)
	}

	conversationID := primitive.NewObjectID()
	_, err = env.DB.ChatMessages().InsertOne(ctx, models.ChatMessage{
		ID:             primitive.NewObjectID(),
		TenantID:       tenant.ID,
		UserID:         admin.ID,
		ConversationID: conversationID,
		AgentID:        "launch-agent",
		Role:           "assistant",
		Content:        "Launch smoke test passed.",
		Status:         models.ChatMessageStatusCompleted,
		CreditsCharged: 1,
		Model:          "launch-model",
		CreatedAt:      now,
	})
	if err != nil {
		t.Fatalf("seed chat message: %v", err)
	}

	req := env.adminRequest(t, "GET", "/api/admin/launch-readiness", nil, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}

	var got LaunchReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assertReadinessStatus(t, got, "brand", LaunchReadinessComplete)
	assertReadinessStatus(t, got, "model", LaunchReadinessComplete)
	assertReadinessStatus(t, got, "agent", LaunchReadinessComplete)
	assertReadinessStatus(t, got, "credits", LaunchReadinessComplete)
	assertReadinessStatus(t, got, "test-chat", LaunchReadinessComplete)
}

func TestIntegration_AdminLaunchReadiness_CrossTenantModel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := setupTestServer(t)
	defer env.Cleanup()
	admin, tenant := createAdminEnv(t, env)
	ctx := context.Background()
	now := time.Now()

	// Create a second tenant with its own model + provider.
	otherUser := testutil.CreateTestUser(t, env.DB, "other-tenant@test.com", "StrongP@ss1!", "Other Admin")
	otherTenant := testutil.CreateTestTenant(t, env.DB, "Other Tenant", otherUser.ID, false)

	otherProviderID := primitive.NewObjectID()
	_, err := env.DB.ModelProviders().InsertOne(ctx, models.ModelProvider{
		ID:           otherProviderID,
		TenantID:     otherTenant.ID,
		Name:         "Other OpenAI",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.other.com/v1",
		APIKey:       "sk-other-key",
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("seed other tenant provider: %v", err)
	}
	_, err = env.DB.ModelConfigs().InsertOne(ctx, models.ModelConfig{
		ID:          primitive.NewObjectID(),
		TenantID:    otherTenant.ID,
		ProviderID:  otherProviderID,
		Name:        "Other Text Model",
		DisplayName: "Other Text Model",
		Modality:    models.ModelModalityText,
		ModelID:     "other-model",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("seed other tenant model: %v", err)
	}

	// Current tenant has no model config — readiness should still be warning.
	req := env.adminRequest(t, "GET", "/api/admin/launch-readiness", nil, admin, tenant.ID.Hex())
	resp, err := env.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, testutil.ReadResponseBody(t, resp))
	}

	var got LaunchReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assertReadinessStatus(t, got, "model", LaunchReadinessWarning)
}

func assertReadinessStatus(t *testing.T, response LaunchReadinessResponse, id string, want LaunchReadinessStatus) {
	t.Helper()
	for _, item := range response.Items {
		if item.ID == id {
			if item.Status != want {
				t.Fatalf("expected readiness item %s status %s, got %s", id, want, item.Status)
			}
			return
		}
	}
	t.Fatalf("readiness item %s not found in %#v", id, response.Items)
}
