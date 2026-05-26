package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lastsaas/internal/middleware"
	"lastsaas/internal/models"
	"lastsaas/internal/testutil"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// agentTestEnv sets up users and tenants for agent handler tests.
type agentTestEnv struct {
	*testEnv
	owner        *models.User
	tenant       *models.Tenant
	admin        *models.User
	regularUser  *models.User
	otherOwner   *models.User
	otherTenant  *models.Tenant
}

func setupAgentTestEnv(t *testing.T) *agentTestEnv {
	t.Helper()
	env := setupTestServer(t)
	testutil.MarkSystemInitialized(t, env.DB)

	// Primary tenant with owner, admin, and regular user
	owner := testutil.CreateTestUser(t, env.DB, "agent-owner@test.com", "Test1234!@#$", "Agent Owner")
	tenant := testutil.CreateTestTenant(t, env.DB, "Agent Tenant", owner.ID, false)

	admin := testutil.CreateTestUser(t, env.DB, "agent-admin@test.com", "Test1234!@#$", "Agent Admin")
	testutil.CreateTestMembership(t, env.DB, admin.ID, tenant.ID, models.RoleAdmin)

	regularUser := testutil.CreateTestUser(t, env.DB, "agent-user@test.com", "Test1234!@#$", "Agent User")
	testutil.CreateTestMembership(t, env.DB, regularUser.ID, tenant.ID, models.RoleUser)

	// Second tenant for isolation tests
	otherOwner := testutil.CreateTestUser(t, env.DB, "other-owner@test.com", "Test1234!@#$", "Other Owner")
	otherTenant := testutil.CreateTestTenant(t, env.DB, "Other Tenant", otherOwner.ID, false)

	return &agentTestEnv{
		testEnv:     env,
		owner:       owner,
		tenant:      tenant,
		admin:       admin,
		regularUser: regularUser,
		otherOwner:  otherOwner,
		otherTenant: otherTenant,
	}
}

// seedAgent inserts a test agent into the database and returns it.
func seedAgent(t *testing.T, env *agentTestEnv, slug string) *models.Agent {
	t.Helper()
	now := time.Now()
	agent := models.Agent{
		ID:               primitive.NewObjectID(),
		TenantID:         env.tenant.ID,
		Name:             "Test Agent " + slug,
		Slug:             slug,
		Category:         "general",
		Description:      "A test agent",
		Status:           models.AgentStatusPublished,
		Visibility:       models.AgentVisibilityPublic,
		SystemPrompt:     "You are a helpful assistant.",
		WelcomeMessage:   "Hello!",
		SuggestedPrompts: []string{"Help me"},
		Capabilities:     []models.AgentCapability{models.AgentCapabilityTextChat},
		CreditCost:       models.AgentCreditCost{TextMessageCredits: 1},
		CreatedBy:        env.owner.ID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	_, err := env.DB.Agents().InsertOne(context.Background(), agent)
	if err != nil {
		t.Fatalf("seed agent: %v", err)
	}
	return &agent
}

func TestListAgents_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ae := setupAgentTestEnv(t)
	defer ae.Cleanup()

	// Seed agent in primary tenant
	seedAgent(t, ae, "tenant-a-agent")

	// Request as other tenant — should see empty list
	req := ae.tenantRequest(t, http.MethodGet, "/api/tenant/agents", nil, ae.otherOwner, ae.otherTenant.ID.Hex())
	resp, err := ae.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var agents []models.Agent
	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(agents) != 0 {
		t.Fatalf("expected empty list for other tenant, got %d agents", len(agents))
	}

	// Request as primary tenant — should see the agent
	req2 := ae.tenantRequest(t, http.MethodGet, "/api/tenant/agents", nil, ae.owner, ae.tenant.ID.Hex())
	resp2, err := ae.Client.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}

	var agents2 []models.Agent
	if err := json.NewDecoder(resp2.Body).Decode(&agents2); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(agents2) != 1 {
		t.Fatalf("expected 1 agent for primary tenant, got %d", len(agents2))
	}
}

func TestListAgents_StatusFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ae := setupAgentTestEnv(t)
	defer ae.Cleanup()

	// Seed a published agent
	seedAgent(t, ae, "published-agent")

	// Seed a draft agent
	now := time.Now()
	draftAgent := models.Agent{
		ID:               primitive.NewObjectID(),
		TenantID:         ae.tenant.ID,
		Name:             "Draft Agent",
		Slug:             "draft-agent",
		Category:         "general",
		Description:      "A draft agent",
		Status:           models.AgentStatusDraft,
		Visibility:       models.AgentVisibilityPrivate,
		SystemPrompt:     "You are a draft assistant.",
		Capabilities:     []models.AgentCapability{models.AgentCapabilityTextChat},
		CreditCost:       models.AgentCreditCost{TextMessageCredits: 1},
		CreatedBy:        ae.owner.ID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	_, err := ae.DB.Agents().InsertOne(context.Background(), draftAgent)
	if err != nil {
		t.Fatalf("seed draft agent: %v", err)
	}

	// Filter by status=published
	req := ae.tenantRequest(t, http.MethodGet, "/api/tenant/agents?status=published", nil, ae.owner, ae.tenant.ID.Hex())
	resp, err := ae.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var agents []models.Agent
	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 published agent, got %d", len(agents))
	}
	if agents[0].Slug != "published-agent" {
		t.Fatalf("expected published-agent, got %s", agents[0].Slug)
	}
}

func TestCreateAgent_RequiresAdmin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ae := setupAgentTestEnv(t)
	defer ae.Cleanup()

	body := strings.NewReader(`{
		"name": "New Agent",
		"slug": "new-agent",
		"category": "general",
		"description": "A new agent",
		"status": "draft",
		"visibility": "private",
		"systemPrompt": "You are helpful.",
		"capabilities": ["text_chat"],
		"creditCost": {"textMessageCredits": 1}
	}`)

	// Regular user should get 403
	req := ae.tenantRequest(t, http.MethodPost, "/api/tenant/agents", body, ae.regularUser, ae.tenant.ID.Hex())
	resp, err := ae.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for regular user creating agent, got %d", resp.StatusCode)
	}
}

func TestCreateAgent_AdminCanCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ae := setupAgentTestEnv(t)
	defer ae.Cleanup()

	body := strings.NewReader(`{
		"name": "New Agent",
		"slug": "new-agent",
		"category": "general",
		"description": "A new agent",
		"status": "draft",
		"visibility": "private",
		"systemPrompt": "You are helpful.",
		"capabilities": ["text_chat"],
		"creditCost": {"textMessageCredits": 1}
	}`)

	req := ae.tenantRequest(t, http.MethodPost, "/api/tenant/agents", body, ae.admin, ae.tenant.ID.Hex())
	resp, err := ae.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for admin creating agent, got %d", resp.StatusCode)
	}

	var agent models.Agent
	if err := json.NewDecoder(resp.Body).Decode(&agent); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if agent.Slug != "new-agent" {
		t.Fatalf("expected slug new-agent, got %s", agent.Slug)
	}
	if agent.TenantID != ae.tenant.ID {
		t.Fatalf("expected tenant ID %s, got %s", ae.tenant.ID.Hex(), agent.TenantID.Hex())
	}
}

func TestCreateAgent_DuplicateSlug(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ae := setupAgentTestEnv(t)
	defer ae.Cleanup()

	// Seed agent with slug "test-agent"
	seedAgent(t, ae, "test-agent")

	// Try to create another agent with the same slug
	body := strings.NewReader(`{
		"name": "Duplicate Agent",
		"slug": "test-agent",
		"category": "general",
		"description": "A duplicate agent",
		"status": "draft",
		"visibility": "private",
		"systemPrompt": "You are helpful.",
		"capabilities": ["text_chat"],
		"creditCost": {"textMessageCredits": 1}
	}`)

	req := ae.tenantRequest(t, http.MethodPost, "/api/tenant/agents", body, ae.admin, ae.tenant.ID.Hex())
	resp, err := ae.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate slug, got %d", resp.StatusCode)
	}
}

func TestCreateAgent_DuplicateSlugDifferentTenant(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ae := setupAgentTestEnv(t)
	defer ae.Cleanup()

	// Seed agent with slug "shared-slug" in primary tenant
	seedAgent(t, ae, "shared-slug")

	// Create agent with same slug in other tenant — should succeed
	body := strings.NewReader(`{
		"name": "Other Agent",
		"slug": "shared-slug",
		"category": "general",
		"description": "An agent in another tenant",
		"status": "draft",
		"visibility": "private",
		"systemPrompt": "You are helpful.",
		"capabilities": ["text_chat"],
		"creditCost": {"textMessageCredits": 1}
	}`)

	req := ae.tenantRequest(t, http.MethodPost, "/api/tenant/agents", body, ae.otherOwner, ae.otherTenant.ID.Hex())
	resp, err := ae.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for same slug in different tenant, got %d", resp.StatusCode)
	}
}

func TestGetAgent_NotFoundInOtherTenant(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ae := setupAgentTestEnv(t)
	defer ae.Cleanup()

	agent := seedAgent(t, ae, "isolated-agent")

	// Request as other tenant — should get 404
	req := ae.tenantRequest(t, http.MethodGet, "/api/tenant/agents/"+agent.ID.Hex(), nil, ae.otherOwner, ae.otherTenant.ID.Hex())
	resp, err := ae.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for agent in other tenant, got %d", resp.StatusCode)
	}

	// Request as primary tenant — should get 200
	req2 := ae.tenantRequest(t, http.MethodGet, "/api/tenant/agents/"+agent.ID.Hex(), nil, ae.owner, ae.tenant.ID.Hex())
	resp2, err := ae.Client.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for agent in own tenant, got %d", resp2.StatusCode)
	}
}

func TestUpdateAgent_RequiresAdmin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ae := setupAgentTestEnv(t)
	defer ae.Cleanup()

	agent := seedAgent(t, ae, "update-agent")

	body := strings.NewReader(`{"name": "Updated Name"}`)

	// Regular user should get 403
	req := ae.tenantRequest(t, http.MethodPut, "/api/tenant/agents/"+agent.ID.Hex(), body, ae.regularUser, ae.tenant.ID.Hex())
	resp, err := ae.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for regular user updating agent, got %d", resp.StatusCode)
	}
}

func TestUpdateAgent_AdminCanUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ae := setupAgentTestEnv(t)
	defer ae.Cleanup()

	agent := seedAgent(t, ae, "updatable-agent")

	body := strings.NewReader(`{"name": "Updated Name"}`)

	req := ae.tenantRequest(t, http.MethodPut, "/api/tenant/agents/"+agent.ID.Hex(), body, ae.admin, ae.tenant.ID.Hex())
	resp, err := ae.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for admin updating agent, got %d", resp.StatusCode)
	}

	var updated models.Agent
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Fatalf("expected name 'Updated Name', got %q", updated.Name)
	}
}

func TestUpdateAgent_CannotUpdateOtherTenantAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ae := setupAgentTestEnv(t)
	defer ae.Cleanup()

	agent := seedAgent(t, ae, "cross-tenant-update")

	body := strings.NewReader(`{"name": "Hacked Name"}`)

	// Other tenant admin tries to update primary tenant's agent
	req := ae.tenantRequest(t, http.MethodPut, "/api/tenant/agents/"+agent.ID.Hex(), body, ae.otherOwner, ae.otherTenant.ID.Hex())
	resp, err := ae.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for updating agent in other tenant, got %d", resp.StatusCode)
	}
}

func TestDeleteAgent_SoftDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ae := setupAgentTestEnv(t)
	defer ae.Cleanup()

	agent := seedAgent(t, ae, "soft-delete-agent")

	req := ae.tenantRequest(t, http.MethodDelete, "/api/tenant/agents/"+agent.ID.Hex(), nil, ae.admin, ae.tenant.ID.Hex())
	resp, err := ae.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for deleting agent, got %d", resp.StatusCode)
	}

	// Verify the document still exists with status "archived"
	var archived models.Agent
	err = ae.DB.Agents().FindOne(context.Background(), bson.M{"_id": agent.ID}).Decode(&archived)
	if err != nil {
		t.Fatalf("expected agent document to still exist, got error: %v", err)
	}
	if archived.Status != models.AgentStatusArchived {
		t.Fatalf("expected status 'archived', got %q", archived.Status)
	}
}

func TestDeleteAgent_RequiresAdmin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ae := setupAgentTestEnv(t)
	defer ae.Cleanup()

	agent := seedAgent(t, ae, "delete-rbac-agent")

	// Regular user should get 403
	req := ae.tenantRequest(t, http.MethodDelete, "/api/tenant/agents/"+agent.ID.Hex(), nil, ae.regularUser, ae.tenant.ID.Hex())
	resp, err := ae.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for regular user deleting agent, got %d", resp.StatusCode)
	}
}

func TestDeleteAgent_CannotDeleteOtherTenantAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ae := setupAgentTestEnv(t)
	defer ae.Cleanup()

	agent := seedAgent(t, ae, "cross-tenant-delete")

	// Other tenant owner tries to delete primary tenant's agent
	req := ae.tenantRequest(t, http.MethodDelete, "/api/tenant/agents/"+agent.ID.Hex(), nil, ae.otherOwner, ae.otherTenant.ID.Hex())
	resp, err := ae.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for deleting agent in other tenant, got %d", resp.StatusCode)
	}
}

func TestDeleteAgent_AlreadyArchived(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ae := setupAgentTestEnv(t)
	defer ae.Cleanup()

	// Seed an already-archived agent
	now := time.Now()
	archivedAgent := models.Agent{
		ID:               primitive.NewObjectID(),
		TenantID:         ae.tenant.ID,
		Name:             "Already Archived",
		Slug:             "already-archived",
		Category:         "general",
		Description:      "An archived agent",
		Status:           models.AgentStatusArchived,
		Visibility:       models.AgentVisibilityPrivate,
		SystemPrompt:     "You are archived.",
		Capabilities:     []models.AgentCapability{models.AgentCapabilityTextChat},
		CreditCost:       models.AgentCreditCost{TextMessageCredits: 1},
		CreatedBy:        ae.owner.ID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	_, err := ae.DB.Agents().InsertOne(context.Background(), archivedAgent)
	if err != nil {
		t.Fatalf("seed archived agent: %v", err)
	}

	// Deleting an already-archived agent should return 404
	req := ae.tenantRequest(t, http.MethodDelete, "/api/tenant/agents/"+archivedAgent.ID.Hex(), nil, ae.admin, ae.tenant.ID.Hex())
	resp, err := ae.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for deleting already-archived agent, got %d", resp.StatusCode)
	}
}

func TestCreateAgent_ValidationRejectsInvalidInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ae := setupAgentTestEnv(t)
	defer ae.Cleanup()

	// Missing required fields
	body := strings.NewReader(`{
		"slug": "missing-name",
		"category": "general",
		"description": "Missing name",
		"status": "draft",
		"visibility": "private",
		"systemPrompt": "You are helpful.",
		"capabilities": ["text_chat"],
		"creditCost": {"textMessageCredits": 1}
	}`)

	req := ae.tenantRequest(t, http.MethodPost, "/api/tenant/agents", body, ae.admin, ae.tenant.ID.Hex())
	resp, err := ae.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for validation error, got %d", resp.StatusCode)
	}
}

// Unit tests that don't require a database connection

func TestAgentHandler_ListAgents_MissingTenantContext(t *testing.T) {
	handler := NewAgentHandler(nil) // nil DB is fine — should fail before DB access
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/agents", nil)
	rr := httptest.NewRecorder()

	handler.ListAgents(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without tenant context, got %d", rr.Code)
	}
}

func TestAgentHandler_GetAgent_InvalidID(t *testing.T) {
	handler := NewAgentHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/agents/not-a-valid-id", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.TenantContextKey, &models.Tenant{ID: primitive.NewObjectID()}))
	// Use gorilla/mux to set URL vars
	req = mux.SetURLVars(req, map[string]string{"agentId": "not-a-valid-id"})
	rr := httptest.NewRecorder()

	handler.GetAgent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid agent ID, got %d", rr.Code)
	}
}

func TestAgentHandler_DeleteAgent_MissingTenantContext(t *testing.T) {
	handler := NewAgentHandler(nil)
	req := httptest.NewRequest(http.MethodDelete, "/api/tenant/agents/abc", nil)
	rr := httptest.NewRecorder()

	handler.DeleteAgent(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without tenant context, got %d", rr.Code)
	}
}
