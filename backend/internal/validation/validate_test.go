package validation

import (
	"strings"
	"testing"
	"time"

	"agentstore/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func validUser() models.User {
	return models.User{
		Email:       "test@example.com",
		DisplayName: "Test User",
		AuthMethods: []models.AuthMethod{models.AuthMethodPassword},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func TestValidate_ValidUser(t *testing.T) {
	u := validUser()
	if err := Validate(&u); err != nil {
		t.Errorf("expected valid user to pass: %v", err)
	}
}

func TestValidate_UserMissingEmail(t *testing.T) {
	u := validUser()
	u.Email = ""
	err := Validate(&u)
	if err == nil {
		t.Fatal("expected validation error for missing email")
	}
	if !strings.Contains(err.Error(), "Email") {
		t.Errorf("expected error to mention Email, got: %v", err)
	}
}

func TestValidate_UserInvalidEmail(t *testing.T) {
	u := validUser()
	u.Email = "not-an-email"
	if err := Validate(&u); err == nil {
		t.Fatal("expected validation error for invalid email")
	}
}

func TestValidate_UserMissingDisplayName(t *testing.T) {
	u := validUser()
	u.DisplayName = ""
	if err := Validate(&u); err == nil {
		t.Fatal("expected validation error for missing display name")
	}
}

func TestValidate_UserEmptyAuthMethods(t *testing.T) {
	u := validUser()
	u.AuthMethods = nil
	if err := Validate(&u); err == nil {
		t.Fatal("expected validation error for empty auth methods")
	}
}

func TestValidate_UserInvalidAuthMethod(t *testing.T) {
	u := validUser()
	u.AuthMethods = []models.AuthMethod{"carrier_pigeon"}
	if err := Validate(&u); err == nil {
		t.Fatal("expected validation error for invalid auth method")
	}
}

func TestValidate_UserValidThemePreference(t *testing.T) {
	for _, theme := range []string{"light", "dark", "system", ""} {
		u := validUser()
		u.ThemePreference = theme
		if err := Validate(&u); err != nil {
			t.Errorf("theme %q should be valid: %v", theme, err)
		}
	}
}

func TestValidate_UserInvalidThemePreference(t *testing.T) {
	u := validUser()
	u.ThemePreference = "neon"
	if err := Validate(&u); err == nil {
		t.Fatal("expected validation error for invalid theme preference")
	}
}

func TestValidate_ValidTenant(t *testing.T) {
	tenant := models.Tenant{
		Name:      "Acme Corp",
		Slug:      "acme-corp",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := Validate(&tenant); err != nil {
		t.Errorf("expected valid tenant to pass: %v", err)
	}
}

func TestValidate_TenantNegativeCredits(t *testing.T) {
	tenant := models.Tenant{
		Name:                "Acme Corp",
		Slug:                "acme-corp",
		SubscriptionCredits: -1,
		PurchasedCredits:    -1,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	if err := Validate(&tenant); err == nil {
		t.Fatal("expected validation error for negative tenant credits")
	}
}

func TestValidate_TenantMissingName(t *testing.T) {
	tenant := models.Tenant{Slug: "slug", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := Validate(&tenant); err == nil {
		t.Fatal("expected validation error for missing tenant name")
	}
}

func TestValidate_TenantInvalidBillingStatus(t *testing.T) {
	tenant := models.Tenant{
		Name: "Test", Slug: "test", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		BillingStatus: "bogus",
	}
	if err := Validate(&tenant); err == nil {
		t.Fatal("expected validation error for invalid billing status")
	}
}

func TestValidate_ValidMembership(t *testing.T) {
	m := models.TenantMembership{
		UserID:    primitive.NewObjectID(),
		TenantID:  primitive.NewObjectID(),
		Role:      models.RoleAdmin,
		JoinedAt:  time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := Validate(&m); err != nil {
		t.Errorf("expected valid membership to pass: %v", err)
	}
}

func TestValidate_MembershipInvalidRole(t *testing.T) {
	m := models.TenantMembership{
		UserID: primitive.NewObjectID(), TenantID: primitive.NewObjectID(),
		Role: "superadmin", JoinedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := Validate(&m); err == nil {
		t.Fatal("expected validation error for invalid role")
	}
}

func TestValidate_ValidAPIKey(t *testing.T) {
	k := models.APIKey{
		Name: "test-key", KeyHash: "hash", KeyPreview: "prev",
		Authority: models.APIKeyAuthorityAdmin,
		CreatedBy: primitive.NewObjectID(), CreatedAt: time.Now(),
	}
	if err := Validate(&k); err != nil {
		t.Errorf("expected valid API key to pass: %v", err)
	}
}

func TestValidate_APIKeyInvalidAuthority(t *testing.T) {
	k := models.APIKey{
		Name: "test-key", KeyHash: "hash", KeyPreview: "prev",
		Authority: "superuser",
		CreatedBy: primitive.NewObjectID(), CreatedAt: time.Now(),
	}
	if err := Validate(&k); err == nil {
		t.Fatal("expected validation error for invalid authority")
	}
}

func TestValidate_ValidPlan(t *testing.T) {
	p := models.Plan{
		Name: "Pro", PricingModel: models.PricingModelFlat,
		CreditResetPolicy: models.CreditResetPolicyReset,
		CreatedAt:         time.Now(), UpdatedAt: time.Now(),
	}
	if err := Validate(&p); err != nil {
		t.Errorf("expected valid plan to pass: %v", err)
	}
}

func TestValidate_PlanInvalidPricingModel(t *testing.T) {
	p := models.Plan{
		Name: "Bad", PricingModel: "usage_based",
		CreditResetPolicy: models.CreditResetPolicyReset,
		CreatedAt:         time.Now(), UpdatedAt: time.Now(),
	}
	if err := Validate(&p); err == nil {
		t.Fatal("expected validation error for invalid pricing model")
	}
}

func TestValidate_PlanNegativePrice(t *testing.T) {
	p := models.Plan{
		Name: "Bad", PricingModel: models.PricingModelFlat,
		CreditResetPolicy: models.CreditResetPolicyReset,
		MonthlyPriceCents: -100,
		CreatedAt:         time.Now(), UpdatedAt: time.Now(),
	}
	if err := Validate(&p); err == nil {
		t.Fatal("expected validation error for negative price")
	}
}

func TestValidate_ValidWebhook(t *testing.T) {
	w := models.Webhook{
		Name: "test", URL: "https://example.com/hook",
		Secret: "whsec_test", SecretPreview: "test1234",
		Events:    []models.WebhookEventType{models.WebhookEventPaymentReceived},
		CreatedBy: primitive.NewObjectID(),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := Validate(&w); err != nil {
		t.Errorf("expected valid webhook to pass: %v", err)
	}
}

func TestValidate_WebhookInvalidEvent(t *testing.T) {
	w := models.Webhook{
		Name: "test", URL: "https://example.com/hook",
		Secret: "whsec_test", SecretPreview: "test1234",
		Events:    []models.WebhookEventType{"bogus.event"},
		CreatedBy: primitive.NewObjectID(),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := Validate(&w); err == nil {
		t.Fatal("expected validation error for invalid webhook event")
	}
}

func TestValidate_ValidInvitation(t *testing.T) {
	inv := models.Invitation{
		TenantID: primitive.NewObjectID(), Email: "user@test.com",
		Role: models.RoleUser, Token: "tok123",
		Status: models.InvitationPending, InvitedBy: primitive.NewObjectID(),
		ExpiresAt: time.Now().Add(24 * time.Hour), CreatedAt: time.Now(),
	}
	if err := Validate(&inv); err != nil {
		t.Errorf("expected valid invitation to pass: %v", err)
	}
}

func TestValidate_InvitationInvalidStatus(t *testing.T) {
	inv := models.Invitation{
		TenantID: primitive.NewObjectID(), Email: "user@test.com",
		Role: models.RoleUser, Token: "tok123",
		Status: "expired", InvitedBy: primitive.NewObjectID(),
		ExpiresAt: time.Now(), CreatedAt: time.Now(),
	}
	if err := Validate(&inv); err == nil {
		t.Fatal("expected validation error for invalid invitation status")
	}
}

func TestValidate_ValidConfigVar(t *testing.T) {
	cv := models.ConfigVar{
		Name: "app.title", Type: models.ConfigTypeString,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := Validate(&cv); err != nil {
		t.Errorf("expected valid config var to pass: %v", err)
	}
}

func TestValidate_ConfigVarInvalidType(t *testing.T) {
	cv := models.ConfigVar{
		Name: "test", Type: "yaml",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := Validate(&cv); err == nil {
		t.Fatal("expected validation error for invalid config var type")
	}
}

func TestValidate_CreditBundleZeroCredits(t *testing.T) {
	cb := models.CreditBundle{
		Name: "Small", Credits: 0, PriceCents: 100,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := Validate(&cb); err == nil {
		t.Fatal("expected validation error for zero credits")
	}
}

func TestValidate_ValidFinancialTransaction(t *testing.T) {
	ft := models.FinancialTransaction{
		TenantID: primitive.NewObjectID(), UserID: primitive.NewObjectID(),
		Type: models.TransactionSubscription, Currency: "usd",
		InvoiceNumber: "INV-000001", CreatedAt: time.Now(),
	}
	if err := Validate(&ft); err != nil {
		t.Errorf("expected valid transaction to pass: %v", err)
	}
}

func TestValidate_TransactionInvalidType(t *testing.T) {
	ft := models.FinancialTransaction{
		TenantID: primitive.NewObjectID(), UserID: primitive.NewObjectID(),
		Type: "chargeback", Currency: "usd",
		InvoiceNumber: "INV-000001", CreatedAt: time.Now(),
	}
	if err := Validate(&ft); err == nil {
		t.Fatal("expected validation error for invalid transaction type")
	}
}

func TestValidate_ValidChatModels(t *testing.T) {
	conversation := models.Conversation{
		TenantID:  primitive.NewObjectID(),
		UserID:    primitive.NewObjectID(),
		AgentID:   "legal-expert",
		Title:     "Contract review",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := Validate(&conversation); err != nil {
		t.Errorf("expected valid conversation to pass: %v", err)
	}

	message := models.ChatMessage{
		TenantID:       primitive.NewObjectID(),
		UserID:         primitive.NewObjectID(),
		ConversationID: primitive.NewObjectID(),
		AgentID:        "legal-expert",
		Role:           "assistant",
		Content:        "Here is the answer.",
		CreditsCharged: 3,
		Model:          "gpt-test",
		CreatedAt:      time.Now(),
	}
	if err := Validate(&message); err != nil {
		t.Errorf("expected valid chat message to pass: %v", err)
	}
}

func TestValidate_InvalidChatModels(t *testing.T) {
	conversation := models.Conversation{
		TenantID:  primitive.NewObjectID(),
		UserID:    primitive.NewObjectID(),
		AgentID:   "",
		Title:     "",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := Validate(&conversation); err == nil {
		t.Fatal("expected validation error for invalid conversation")
	}

	message := models.ChatMessage{
		TenantID:       primitive.NewObjectID(),
		UserID:         primitive.NewObjectID(),
		ConversationID: primitive.NewObjectID(),
		AgentID:        "legal-expert",
		Role:           "system",
		Content:        "",
		CreditsCharged: -1,
		Model:          strings.Repeat("m", 101),
		CreatedAt:      time.Now(),
	}
	if err := Validate(&message); err == nil {
		t.Fatal("expected validation error for invalid chat message")
	}
}

func TestValidate_ValidLLMConfig(t *testing.T) {
	config := models.LLMConfig{
		APIKey:    "sk-test",
		BaseURL:   "https://api.example.com/v1",
		Model:     "gpt-test",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := Validate(&config); err != nil {
		t.Errorf("expected valid LLM config to pass: %v", err)
	}
}

func TestValidate_InvalidActiveLLMConfig(t *testing.T) {
	config := models.LLMConfig{
		APIKey:    "",
		BaseURL:   "",
		Model:     "",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := Validate(&config); err == nil {
		t.Fatal("expected validation error for invalid active LLM config")
	}
}

func TestValidate_ErrorFormatting(t *testing.T) {
	u := models.User{} // all required fields missing
	err := Validate(&u)
	if err == nil {
		t.Fatal("expected validation error")
	}
	msg := err.Error()
	if !strings.HasPrefix(msg, "validation failed: ") {
		t.Errorf("expected 'validation failed:' prefix, got: %s", msg)
	}
}

func validAgent() models.Agent {
	return models.Agent{
		TenantID:         primitive.NewObjectID(),
		Name:             "Growth Strategist",
		Slug:             "growth-strategist",
		Category:         "Marketing",
		Description:      "Plans launch and growth work",
		Icon:             "rocket",
		Color:            "#7C3AED",
		Status:           models.AgentStatusDraft,
		Visibility:       models.AgentVisibilityPrivate,
		SystemPrompt:     "You are a growth strategist.",
		WelcomeMessage:   "Tell me about your launch.",
		SuggestedPrompts: []string{"Draft a launch plan"},
		Capabilities:     []models.AgentCapability{models.AgentCapabilityTextChat},
		CreditCost:       models.AgentCreditCost{TextMessageCredits: 3},
		CreatedBy:        primitive.NewObjectID(),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

func TestValidate_ValidAgent(t *testing.T) {
	agent := validAgent()
	if err := Validate(&agent); err != nil {
		t.Fatalf("expected valid agent to pass: %v", err)
	}
}

func TestValidate_AgentInvalidStatus(t *testing.T) {
	agent := validAgent()
	agent.Status = "deleted"
	if err := Validate(&agent); err == nil {
		t.Fatal("expected invalid agent status to fail")
	}
}

func TestValidate_AgentRequiresTextCapability(t *testing.T) {
	agent := validAgent()
	agent.Capabilities = []models.AgentCapability{"spreadsheet_magic"}
	if err := Validate(&agent); err == nil {
		t.Fatal("expected invalid capability to fail")
	}
}

func TestValidate_ValidModelProvider(t *testing.T) {
	provider := models.ModelProvider{
		TenantID:     primitive.NewObjectID(),
		Name:         "OpenAI Compatible",
		ProviderType: models.ProviderTypeOpenAICompatible,
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk-test",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := Validate(&provider); err != nil {
		t.Fatalf("expected valid model provider to pass: %v", err)
	}
}

func TestValidate_ModelProviderInvalidType(t *testing.T) {
	provider := models.ModelProvider{
		TenantID:     primitive.NewObjectID(),
		Name:         "Bad",
		ProviderType: "unknown",
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk-test",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := Validate(&provider); err == nil {
		t.Fatal("expected invalid provider type to fail")
	}
}

func TestValidate_ValidModelConfig(t *testing.T) {
	config := models.ModelConfig{
		TenantID:      primitive.NewObjectID(),
		ProviderID:    primitive.NewObjectID(),
		Name:          "Default Text",
		DisplayName:   "Default Text Model",
		Modality:      models.ModelModalityText,
		ModelID:       "gpt-4o-mini",
		DefaultParams: map[string]interface{}{"temperature": 0.7},
		Enabled:       true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := Validate(&config); err != nil {
		t.Fatalf("expected valid model config to pass: %v", err)
	}
}

func TestValidate_ModelConfigInvalidModality(t *testing.T) {
	config := models.ModelConfig{
		TenantID:    primitive.NewObjectID(),
		ProviderID:  primitive.NewObjectID(),
		Name:        "Bad",
		DisplayName: "Bad",
		Modality:    "audio",
		ModelID:     "model",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := Validate(&config); err == nil {
		t.Fatal("expected invalid model modality to fail")
	}
}

func TestValidate_ChatMessageStatus(t *testing.T) {
	msg := models.ChatMessage{
		TenantID:       primitive.NewObjectID(),
		UserID:         primitive.NewObjectID(),
		ConversationID: primitive.NewObjectID(),
		AgentID:        "growth-strategist",
		Role:           "assistant",
		Content:        "Working on it",
		Status:         models.ChatMessageStatusGenerating,
		CreatedAt:      time.Now(),
	}
	if err := Validate(&msg); err != nil {
		t.Fatalf("expected generating chat message to pass: %v", err)
	}
	msg.Status = "vanished"
	if err := Validate(&msg); err == nil {
		t.Fatal("expected invalid chat message status to fail")
	}
}

func TestValidate_GeneratingChatMessageAllowsEmptyContent(t *testing.T) {
	msg := models.ChatMessage{
		TenantID:       primitive.NewObjectID(),
		UserID:         primitive.NewObjectID(),
		ConversationID: primitive.NewObjectID(),
		AgentID:        "growth-strategist",
		Role:           "assistant",
		Content:        "",
		Status:         models.ChatMessageStatusGenerating,
		CreatedAt:      time.Now(),
	}
	if err := Validate(&msg); err != nil {
		t.Fatalf("expected generating placeholder message to pass: %v", err)
	}
}

func validKnowledgeDocument() models.KnowledgeDocument {
	return models.KnowledgeDocument{
		TenantID:   primitive.NewObjectID(),
		AgentID:    "growth-strategist",
		Name:       "Pricing FAQ",
		SourceType: models.KnowledgeSourceText,
		Status:     models.KnowledgeStatusProcessing,
		ChunkCount: 0,
		CharCount:  120,
		CreatedBy:  primitive.NewObjectID(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func TestValidate_ValidKnowledgeDocument(t *testing.T) {
	doc := validKnowledgeDocument()
	if err := Validate(&doc); err != nil {
		t.Fatalf("expected valid knowledge document to pass: %v", err)
	}
}

func TestValidate_KnowledgeDocumentInvalidSourceType(t *testing.T) {
	doc := validKnowledgeDocument()
	doc.SourceType = "spreadsheet"
	if err := Validate(&doc); err == nil {
		t.Fatal("expected invalid knowledge source type to fail")
	}
}

func TestValidate_KnowledgeDocumentInvalidStatus(t *testing.T) {
	doc := validKnowledgeDocument()
	doc.Status = "pending"
	if err := Validate(&doc); err == nil {
		t.Fatal("expected invalid knowledge status to fail")
	}
}

func validKnowledgeChunk() models.KnowledgeChunk {
	return models.KnowledgeChunk{
		TenantID:   primitive.NewObjectID(),
		AgentID:    "growth-strategist",
		DocumentID: primitive.NewObjectID(),
		Seq:        0,
		Text:       "Our pricing starts at $10/month.",
		Embedding:  []float64{0.1, 0.2, 0.3},
		CreatedAt:  time.Now(),
	}
}

func TestValidate_ValidKnowledgeChunk(t *testing.T) {
	chunk := validKnowledgeChunk()
	if err := Validate(&chunk); err != nil {
		t.Fatalf("expected valid knowledge chunk to pass: %v", err)
	}
}

func TestValidate_KnowledgeChunkRequiresText(t *testing.T) {
	chunk := validKnowledgeChunk()
	chunk.Text = ""
	if err := Validate(&chunk); err == nil {
		t.Fatal("expected empty chunk text to fail")
	}
}

func validMessageFeedback() models.MessageFeedback {
	return models.MessageFeedback{
		TenantID:       primitive.NewObjectID(),
		UserID:         primitive.NewObjectID(),
		ConversationID: primitive.NewObjectID(),
		MessageID:      primitive.NewObjectID(),
		AgentID:        "growth-strategist",
		Rating:         1,
		Comment:        "Helpful answer",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func TestValidate_ValidMessageFeedback(t *testing.T) {
	fb := validMessageFeedback()
	if err := Validate(&fb); err != nil {
		t.Fatalf("expected valid message feedback to pass: %v", err)
	}
}

func TestValidate_MessageFeedbackInvalidRating(t *testing.T) {
	fb := validMessageFeedback()
	fb.Rating = 5
	if err := Validate(&fb); err == nil {
		t.Fatal("expected invalid rating to fail")
	}
}

func validAnnotation() models.Annotation {
	return models.Annotation{
		TenantID:       primitive.NewObjectID(),
		ConversationID: primitive.NewObjectID(),
		MessageID:      primitive.NewObjectID(),
		AgentID:        "growth-strategist",
		AnnotatorID:    primitive.NewObjectID(),
		QualityScore:   4,
		IssueTags:      []string{"wrong_fact", "incomplete"},
		IdealAnswer:    "The correct answer is X.",
		Notes:          "Reviewed",
		Status:         models.AnnotationStatusAnnotated,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func TestValidate_ValidAnnotation(t *testing.T) {
	a := validAnnotation()
	if err := Validate(&a); err != nil {
		t.Fatalf("expected valid annotation to pass: %v", err)
	}
}

func TestValidate_AnnotationInvalidIssueTag(t *testing.T) {
	a := validAnnotation()
	a.IssueTags = []string{"made_up_tag"}
	if err := Validate(&a); err == nil {
		t.Fatal("expected invalid issue tag to fail")
	}
}

func TestValidate_AnnotationScoreOutOfRange(t *testing.T) {
	a := validAnnotation()
	a.QualityScore = 6
	if err := Validate(&a); err == nil {
		t.Fatal("expected out-of-range quality score to fail")
	}
}

func TestValidate_AnnotationInvalidStatus(t *testing.T) {
	a := validAnnotation()
	a.Status = "draft"
	if err := Validate(&a); err == nil {
		t.Fatal("expected invalid annotation status to fail")
	}
}
