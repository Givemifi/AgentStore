# Payment Config Admin Page — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an admin UI page at `/admin/payment-config` that lets operators configure WeChat Pay and Alipay credentials stored in MongoDB, with hot-reload (no server restart needed).

**Architecture:** New `payment_configs` MongoDB collection stores one doc per provider (`key: "wechat_pay" | "alipay"`). A `paymentregistry.Registry` struct holds live service pointers behind a `sync.RWMutex`; on PUT the handler saves to DB then calls `registry.Reload()` which re-initialises gopay clients under a write lock. The existing `billing.go` and `payment_notify.go` handlers are refactored to access services via the registry instead of holding them directly.

**Tech Stack:** Go 1.25, gorilla/mux, MongoDB driver, github.com/go-pay/gopay v1.5.118, React 19 + TypeScript, @tanstack/react-query, sonner toasts, lucide-react icons.

---

## File Map

**New files:**
- `backend/internal/paymentregistry/registry.go` — Registry struct + Wechat/Alipay/Reload/Seed
- `backend/internal/paymentregistry/registry_test.go` — unit tests
- `backend/internal/api/handlers/payment_config.go` — GetWechat/UpdateWechat/GetAlipay/UpdateAlipay
- `backend/internal/api/handlers/payment_config_test.go` — handler tests
- `frontend/src/pages/admin/PaymentConfigPage.tsx` — admin UI

**Modified files:**
- `backend/internal/models/billing.go` — add `PaymentProviderConfig` struct
- `backend/internal/db/mongodb.go` — add `PaymentConfigs()` accessor
- `backend/internal/db/schema.go` — add `paymentConfigsSchema()` + register in `AllSchemas()`
- `backend/internal/api/handlers/billing.go` — replace wechat/alipay fields with registry
- `backend/internal/api/handlers/payment_notify.go` — same
- `backend/internal/api/handlers/testhelpers_test.go` — update `NewBillingHandler` call
- `backend/cmd/server/main.go` — init registry, seed, reload, register routes
- `frontend/src/api/client.ts` — add `getPaymentConfig` / `updatePaymentConfig`
- `frontend/src/App.tsx` — add `/admin/payment-config` route
- `frontend/src/components/AdminLayout.tsx` — add sidebar nav item

---

## Task 1: DB Model + Collection + Schema

**Files:**
- Modify: `backend/internal/models/billing.go`
- Modify: `backend/internal/db/mongodb.go`
- Modify: `backend/internal/db/schema.go`

- [ ] **Step 1.1: Add PaymentProviderConfig model to billing.go**

Open `backend/internal/models/billing.go`. Add this struct at the bottom of the file (after the existing `PaymentOrder` struct):

```go
// PaymentProviderConfig stores WeChat Pay or Alipay credentials in MongoDB.
// Key field distinguishes providers: "wechat_pay" or "alipay".
type PaymentProviderConfig struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"   json:"id,omitempty"`
	Key          string             `bson:"key"             json:"key"          validate:"required,oneof=wechat_pay alipay"`
	AppID        string             `bson:"appId"           json:"appId"`
	// WeChat-specific
	MchID        string             `bson:"mchId"           json:"mchId"`
	APIv3Key     string             `bson:"apiV3Key"        json:"apiV3Key"`
	CertSerialNo string             `bson:"certSerialNo"    json:"certSerialNo"`
	// Alipay-specific
	PublicKey    string             `bson:"publicKey"       json:"publicKey"`
	ReturnURL    string             `bson:"returnUrl"       json:"returnUrl"`
	IsSandbox    bool               `bson:"isSandbox"       json:"isSandbox"`
	// Shared
	PrivateKey   string             `bson:"privateKey"      json:"privateKey"`
	NotifyURL    string             `bson:"notifyUrl"       json:"notifyUrl"`
	Enabled      bool               `bson:"enabled"         json:"enabled"`
	CreatedAt    time.Time          `bson:"createdAt"       json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt"       json:"updatedAt"`
}
```

- [ ] **Step 1.2: Add PaymentConfigs() accessor to mongodb.go**

Open `backend/internal/db/mongodb.go`. After the `PaymentOrders()` method (around line 450), add:

```go
func (m *MongoDB) PaymentConfigs() *mongo.Collection {
	return m.client.Database(m.dbName).Collection("payment_configs")
}
```

- [ ] **Step 1.3: Add paymentConfigsSchema() to schema.go**

Open `backend/internal/db/schema.go`. In the `AllSchemas()` function body, add `paymentConfigsSchema(),` after `paymentOrdersSchema()`:

```go
// In AllSchemas(), after paymentOrdersSchema():
paymentConfigsSchema(),
```

Then add the schema function at the bottom of the file:

```go
func paymentConfigsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "payment_configs",
		Schema: bson.D{
			{Key: "$jsonSchema", Value: bson.D{
				{Key: "bsonType", Value: "object"},
				{Key: "required", Value: bson.A{"key"}},
				{Key: "properties", Value: bson.D{
					{Key: "key", Value: bson.D{
						{Key: "bsonType", Value: "string"},
						{Key: "enum", Value: bson.A{"wechat_pay", "alipay"}},
					}},
					{Key: "appId",        Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "mchId",        Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "apiV3Key",     Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "privateKey",   Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "certSerialNo", Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "publicKey",    Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "notifyUrl",    Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "returnUrl",    Value: bson.D{{Key: "bsonType", Value: "string"}}},
					{Key: "isSandbox",    Value: bson.D{{Key: "bsonType", Value: "bool"}}},
					{Key: "enabled",      Value: bson.D{{Key: "bsonType", Value: "bool"}}},
					{Key: "createdAt",    Value: bson.D{{Key: "bsonType", Value: "date"}}},
					{Key: "updatedAt",    Value: bson.D{{Key: "bsonType", Value: "date"}}},
				}},
			}},
		},
	}
}
```

- [ ] **Step 1.4: Build to verify**

```bash
cd backend && go build ./...
```

Expected: no errors.

---

## Task 2: PaymentServiceRegistry

**Files:**
- Create: `backend/internal/paymentregistry/registry.go`
- Create: `backend/internal/paymentregistry/registry_test.go`

- [ ] **Step 2.1: Write the failing test**

Create `backend/internal/paymentregistry/registry_test.go`:

```go
package paymentregistry_test

import (
	"context"
	"testing"

	"agentstore/internal/paymentregistry"
	"agentstore/internal/testutil"
)

func TestRegistry_NewIsEmpty(t *testing.T) {
	reg := paymentregistry.New(nil)
	if reg.Wechat() != nil {
		t.Fatal("expected nil wechat service on new registry")
	}
	if reg.Alipay() != nil {
		t.Fatal("expected nil alipay service on new registry")
	}
}

func TestRegistry_NilSafe(t *testing.T) {
	// nil registry must not panic
	var reg *paymentregistry.Registry
	if reg.Wechat() != nil {
		t.Fatal("nil registry Wechat() should return nil")
	}
	if reg.Alipay() != nil {
		t.Fatal("nil registry Alipay() should return nil")
	}
}

func TestRegistry_ReloadEmptyDB(t *testing.T) {
	db, cleanup := testutil.NewTestDB(t)
	defer cleanup()

	reg := paymentregistry.New(db)
	if err := reg.Reload(context.Background()); err != nil {
		t.Fatalf("Reload with empty DB should not error, got: %v", err)
	}
	if reg.Wechat() != nil {
		t.Fatal("expected nil wechat service after reload with no config")
	}
	if reg.Alipay() != nil {
		t.Fatal("expected nil alipay service after reload with no config")
	}
}

func TestRegistry_SeedNoOp_WhenEmpty(t *testing.T) {
	db, cleanup := testutil.NewTestDB(t)
	defer cleanup()

	reg := paymentregistry.New(db)
	// Seed with empty configs — should not insert any docs
	if err := reg.Seed(context.Background(),
		paymentregistry.WechatSeedConfig{},
		paymentregistry.AlipaySeedConfig{},
	); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count, _ := db.PaymentConfigs().CountDocuments(context.Background(), map[string]any{})
	if count != 0 {
		t.Fatalf("expected 0 docs, got %d", count)
	}
}
```

- [ ] **Step 2.2: Run test — expect compile failure**

```bash
cd backend && go test ./internal/paymentregistry/... 2>&1 | head -20
```

Expected: `cannot find package "agentstore/internal/paymentregistry"` — this is the signal to implement.

- [ ] **Step 2.3: Create registry.go**

Create `backend/internal/paymentregistry/registry.go`:

```go
// Package paymentregistry manages live WeChat Pay and Alipay service instances.
// Call Reload to hot-swap credentials without restarting the server.
package paymentregistry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	alipayservice "agentstore/internal/alipay"
	"agentstore/internal/config"
	"agentstore/internal/db"
	"agentstore/internal/models"
	wechatservice "agentstore/internal/wechat"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	KeyWechatPay = "wechat_pay"
	KeyAlipay    = "alipay"
)

// Registry holds the live payment service instances and reloads them from DB on demand.
type Registry struct {
	mu     sync.RWMutex
	db     *db.MongoDB
	wechat *wechatservice.Service
	alipay *alipayservice.Service
}

func New(database *db.MongoDB) *Registry {
	return &Registry{db: database}
}

// Wechat returns the current WeChat Pay service; nil means not configured.
// Safe to call on a nil Registry.
func (r *Registry) Wechat() *wechatservice.Service {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.wechat
}

// Alipay returns the current Alipay service; nil means not configured.
// Safe to call on a nil Registry.
func (r *Registry) Alipay() *alipayservice.Service {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.alipay
}

// Reload reads both provider configs from DB and rebuilds the service instances.
// If either service fails to initialise, the old instances are preserved and an
// error is returned. On success, both instances are atomically replaced.
func (r *Registry) Reload(ctx context.Context) error {
	var wechatDoc, alipayDoc models.PaymentProviderConfig

	wechatErr := r.db.PaymentConfigs().FindOne(ctx, bson.M{"key": KeyWechatPay}).Decode(&wechatDoc)
	alipayErr := r.db.PaymentConfigs().FindOne(ctx, bson.M{"key": KeyAlipay}).Decode(&alipayDoc)

	if wechatErr != nil && !errors.Is(wechatErr, mongo.ErrNoDocuments) {
		return fmt.Errorf("paymentregistry: read wechat config: %w", wechatErr)
	}
	if alipayErr != nil && !errors.Is(alipayErr, mongo.ErrNoDocuments) {
		return fmt.Errorf("paymentregistry: read alipay config: %w", alipayErr)
	}

	// Build services outside the lock (wechat cert download is a network call).
	var newWechat *wechatservice.Service
	var newAlipay *alipayservice.Service

	if wechatErr == nil && wechatDoc.Enabled {
		svc, err := wechatservice.New(config.WeChatPayConfig{
			AppID:        wechatDoc.AppID,
			MchID:        wechatDoc.MchID,
			APIv3Key:     wechatDoc.APIv3Key,
			PrivateKey:   wechatDoc.PrivateKey,
			CertSerialNo: wechatDoc.CertSerialNo,
			NotifyURL:    wechatDoc.NotifyURL,
		})
		if err != nil {
			return fmt.Errorf("paymentregistry: init wechat: %w", err)
		}
		newWechat = svc
	}

	if alipayErr == nil && alipayDoc.Enabled {
		svc, err := alipayservice.New(config.AlipayConfig{
			AppID:      alipayDoc.AppID,
			PrivateKey: alipayDoc.PrivateKey,
			PublicKey:  alipayDoc.PublicKey,
			NotifyURL:  alipayDoc.NotifyURL,
			ReturnURL:  alipayDoc.ReturnURL,
			IsSandbox:  alipayDoc.IsSandbox,
		})
		if err != nil {
			return fmt.Errorf("paymentregistry: init alipay: %w", err)
		}
		newAlipay = svc
	}

	r.mu.Lock()
	r.wechat = newWechat
	r.alipay = newAlipay
	r.mu.Unlock()
	return nil
}

// WechatSeedConfig holds env/yaml WeChat credentials for one-time migration.
type WechatSeedConfig struct {
	AppID        string
	MchID        string
	APIv3Key     string
	PrivateKey   string
	CertSerialNo string
	NotifyURL    string
}

// AlipaySeedConfig holds env/yaml Alipay credentials for one-time migration.
type AlipaySeedConfig struct {
	AppID      string
	PrivateKey string
	PublicKey  string
	NotifyURL  string
	ReturnURL  string
	IsSandbox  bool
}

// Seed writes env/yaml credentials to the DB if the collection is empty.
// This is a one-time migration; subsequent starts skip it.
func (r *Registry) Seed(ctx context.Context, wechat WechatSeedConfig, alipay AlipaySeedConfig) error {
	count, err := r.db.PaymentConfigs().CountDocuments(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("paymentregistry: seed count: %w", err)
	}
	if count > 0 {
		return nil // already populated, skip
	}

	now := time.Now()

	if wechat.MchID != "" {
		doc := models.PaymentProviderConfig{
			ID:           primitive.NewObjectID(),
			Key:          KeyWechatPay,
			AppID:        wechat.AppID,
			MchID:        wechat.MchID,
			APIv3Key:     wechat.APIv3Key,
			PrivateKey:   wechat.PrivateKey,
			CertSerialNo: wechat.CertSerialNo,
			NotifyURL:    wechat.NotifyURL,
			Enabled:      true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if _, err := r.db.PaymentConfigs().InsertOne(ctx, doc); err != nil {
			return fmt.Errorf("paymentregistry: seed wechat: %w", err)
		}
	}

	if alipay.AppID != "" {
		doc := models.PaymentProviderConfig{
			ID:        primitive.NewObjectID(),
			Key:       KeyAlipay,
			AppID:     alipay.AppID,
			PrivateKey: alipay.PrivateKey,
			PublicKey:  alipay.PublicKey,
			NotifyURL:  alipay.NotifyURL,
			ReturnURL:  alipay.ReturnURL,
			IsSandbox:  alipay.IsSandbox,
			Enabled:   true,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if _, err := r.db.PaymentConfigs().InsertOne(ctx, doc); err != nil {
			return fmt.Errorf("paymentregistry: seed alipay: %w", err)
		}
	}

	// Upsert unique index on key field (idempotent)
	_, _ = r.db.PaymentConfigs().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "key", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	return nil
}
```

- [ ] **Step 2.4: Run the tests**

```bash
cd backend && go test ./internal/paymentregistry/... -v
```

Expected: `PASS` for all 4 tests.

- [ ] **Step 2.5: Commit**

```bash
git add backend/internal/paymentregistry/ backend/internal/models/billing.go backend/internal/db/mongodb.go backend/internal/db/schema.go
git commit -m "feat: add PaymentProviderConfig model, PaymentConfigs collection, and paymentregistry"
```

---

## Task 3: PaymentConfigHandler

**Files:**
- Create: `backend/internal/api/handlers/payment_config.go`
- Create: `backend/internal/api/handlers/payment_config_test.go`

- [ ] **Step 3.1: Write the failing tests**

Create `backend/internal/api/handlers/payment_config_test.go`:

```go
package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agentstore/internal/api/handlers"
	"agentstore/internal/paymentregistry"
)

func TestPaymentConfig_GetWechat_NotConfigured(t *testing.T) {
	reg := paymentregistry.New(sharedDB)

	h := handlers.NewPaymentConfigHandler(sharedDB, reg)
	req := httptest.NewRequest(http.MethodGet, "/admin/payment-config/wechat", nil)
	w := httptest.NewRecorder()
	h.GetWechat(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["configured"] != false {
		t.Fatalf("expected configured=false, got %v", resp["configured"])
	}
}

func TestPaymentConfig_UpdateAndGetWechat(t *testing.T) {
	reg := paymentregistry.New(sharedDB)
	h := handlers.NewPaymentConfigHandler(sharedDB, reg)

	// Save config (disabled so no gopay client init attempted)
	body, _ := json.Marshal(map[string]interface{}{
		"appId":        "wx_test_app",
		"mchId":        "1234567890",
		"apiV3Key":     "my-api-v3-key-32chars-exactly-ok",
		"privateKey":   "-----BEGIN RSA PRIVATE KEY-----\nfake\n-----END RSA PRIVATE KEY-----",
		"certSerialNo": "ABCDEF123456",
		"notifyUrl":    "https://example.com/wechat/notify",
		"enabled":      false,
	})
	putReq := httptest.NewRequest(http.MethodPut, "/admin/payment-config/wechat", bytes.NewReader(body))
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	h.UpdateWechat(putW, putReq)

	if putW.Code != http.StatusOK {
		t.Fatalf("PUT expected 200, got %d: %s", putW.Code, putW.Body.String())
	}
	var putResp map[string]string
	json.NewDecoder(putW.Body).Decode(&putResp)
	if putResp["status"] != "saved" {
		t.Fatalf("expected status=saved, got %v", putResp)
	}

	// GET should now return masked sensitive fields
	getReq := httptest.NewRequest(http.MethodGet, "/admin/payment-config/wechat", nil)
	getW := httptest.NewRecorder()
	h.GetWechat(getW, getReq)

	var getResp map[string]interface{}
	json.NewDecoder(getW.Body).Decode(&getResp)

	if getResp["configured"] != true {
		t.Fatalf("expected configured=true after save, got %v", getResp["configured"])
	}
	if getResp["apiV3Key"] != "••••••" {
		t.Fatalf("expected apiV3Key masked, got %v", getResp["apiV3Key"])
	}
	if getResp["privateKey"] != "••••••" {
		t.Fatalf("expected privateKey masked, got %v", getResp["privateKey"])
	}
	// Non-sensitive field should be readable
	if getResp["mchId"] != "1234567890" {
		t.Fatalf("expected mchId=1234567890, got %v", getResp["mchId"])
	}
}

func TestPaymentConfig_GetAlipay_NotConfigured(t *testing.T) {
	reg := paymentregistry.New(sharedDB)
	h := handlers.NewPaymentConfigHandler(sharedDB, reg)

	req := httptest.NewRequest(http.MethodGet, "/admin/payment-config/alipay", nil)
	w := httptest.NewRecorder()
	h.GetAlipay(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["configured"] != false {
		t.Fatalf("expected configured=false")
	}
}

func TestPaymentConfig_SensitiveFieldPreservedOnEmptyUpdate(t *testing.T) {
	reg := paymentregistry.New(sharedDB)
	h := handlers.NewPaymentConfigHandler(sharedDB, reg)

	// First save with actual apiV3Key
	body, _ := json.Marshal(map[string]interface{}{
		"mchId":    "999",
		"apiV3Key": "real-secret-key",
		"enabled":  false,
	})
	r1 := httptest.NewRequest(http.MethodPut, "/admin/payment-config/wechat", bytes.NewReader(body))
	r1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	h.UpdateWechat(w1, r1)

	// Second save with empty apiV3Key (should preserve)
	body2, _ := json.Marshal(map[string]interface{}{
		"mchId":    "999",
		"apiV3Key": "",
		"enabled":  false,
	})
	r2 := httptest.NewRequest(http.MethodPut, "/admin/payment-config/wechat", bytes.NewReader(body2))
	r2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	h.UpdateWechat(w2, r2)

	// GET should still return masked value (original key preserved)
	r3 := httptest.NewRequest(http.MethodGet, "/admin/payment-config/wechat", nil)
	w3 := httptest.NewRecorder()
	h.GetWechat(w3, r3)

	var resp map[string]interface{}
	json.NewDecoder(w3.Body).Decode(&resp)
	if resp["apiV3Key"] != "••••••" {
		t.Fatalf("expected apiV3Key to be masked (still set), got %v", resp["apiV3Key"])
	}
}
```

- [ ] **Step 3.2: Run tests — expect compile failure**

```bash
cd backend && go test ./internal/api/handlers/... -run TestPaymentConfig 2>&1 | head -10
```

Expected: compile error — `handlers.NewPaymentConfigHandler` not defined.

- [ ] **Step 3.3: Implement payment_config.go**

Create `backend/internal/api/handlers/payment_config.go`:

```go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"agentstore/internal/db"
	"agentstore/internal/models"
	"agentstore/internal/paymentregistry"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const maskedValue = "••••••"

func maskField(v string) string {
	if v != "" {
		return maskedValue
	}
	return ""
}

func keepExisting(incoming, existing string) string {
	if incoming == "" || incoming == maskedValue {
		return existing
	}
	return incoming
}

// PaymentConfigHandler handles admin GET/PUT for WeChat Pay and Alipay credentials.
type PaymentConfigHandler struct {
	db       *db.MongoDB
	registry *paymentregistry.Registry
}

func NewPaymentConfigHandler(database *db.MongoDB, registry *paymentregistry.Registry) *PaymentConfigHandler {
	return &PaymentConfigHandler{db: database, registry: registry}
}

// GET /admin/payment-config/wechat
func (h *PaymentConfigHandler) GetWechat(w http.ResponseWriter, r *http.Request) {
	var doc models.PaymentProviderConfig
	err := h.db.PaymentConfigs().FindOne(r.Context(), bson.M{"key": paymentregistry.KeyWechatPay}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"key":        paymentregistry.KeyWechatPay,
			"configured": false,
			"enabled":    false,
		})
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to load config")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"key":          doc.Key,
		"appId":        doc.AppID,
		"mchId":        doc.MchID,
		"apiV3Key":     maskField(doc.APIv3Key),
		"privateKey":   maskField(doc.PrivateKey),
		"certSerialNo": doc.CertSerialNo,
		"notifyUrl":    doc.NotifyURL,
		"enabled":      doc.Enabled,
		"configured":   doc.MchID != "",
	})
}

// PUT /admin/payment-config/wechat
func (h *PaymentConfigHandler) UpdateWechat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AppID        string `json:"appId"`
		MchID        string `json:"mchId"`
		APIv3Key     string `json:"apiV3Key"`
		PrivateKey   string `json:"privateKey"`
		CertSerialNo string `json:"certSerialNo"`
		NotifyURL    string `json:"notifyUrl"`
		Enabled      bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx := r.Context()
	var existing models.PaymentProviderConfig
	findErr := h.db.PaymentConfigs().FindOne(ctx, bson.M{"key": paymentregistry.KeyWechatPay}).Decode(&existing)
	if findErr != nil && !errors.Is(findErr, mongo.ErrNoDocuments) {
		respondWithError(w, http.StatusInternalServerError, "Failed to load existing config")
		return
	}

	now := time.Now()
	_, upsertErr := h.db.PaymentConfigs().UpdateOne(
		ctx,
		bson.M{"key": paymentregistry.KeyWechatPay},
		bson.M{
			"$set": bson.M{
				"appId":        req.AppID,
				"mchId":        req.MchID,
				"apiV3Key":     keepExisting(req.APIv3Key, existing.APIv3Key),
				"privateKey":   keepExisting(req.PrivateKey, existing.PrivateKey),
				"certSerialNo": req.CertSerialNo,
				"notifyUrl":    req.NotifyURL,
				"enabled":      req.Enabled,
				"updatedAt":    now,
			},
			"$setOnInsert": bson.M{
				"_id":       primitive.NewObjectID(),
				"key":       paymentregistry.KeyWechatPay,
				"createdAt": now,
			},
		},
		options.Update().SetUpsert(true),
	)
	if upsertErr != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save config")
		return
	}

	resp := map[string]string{"status": "saved"}
	if reloadErr := h.registry.Reload(ctx); reloadErr != nil {
		resp["warning"] = "Service initialization failed: " + reloadErr.Error()
	}
	respondWithJSON(w, http.StatusOK, resp)
}

// GET /admin/payment-config/alipay
func (h *PaymentConfigHandler) GetAlipay(w http.ResponseWriter, r *http.Request) {
	var doc models.PaymentProviderConfig
	err := h.db.PaymentConfigs().FindOne(r.Context(), bson.M{"key": paymentregistry.KeyAlipay}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"key":        paymentregistry.KeyAlipay,
			"configured": false,
			"enabled":    false,
			"isSandbox":  true,
		})
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to load config")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"key":        doc.Key,
		"appId":      doc.AppID,
		"privateKey": maskField(doc.PrivateKey),
		"publicKey":  maskField(doc.PublicKey),
		"notifyUrl":  doc.NotifyURL,
		"returnUrl":  doc.ReturnURL,
		"isSandbox":  doc.IsSandbox,
		"enabled":    doc.Enabled,
		"configured": doc.AppID != "",
	})
}

// PUT /admin/payment-config/alipay
func (h *PaymentConfigHandler) UpdateAlipay(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AppID      string `json:"appId"`
		PrivateKey string `json:"privateKey"`
		PublicKey  string `json:"publicKey"`
		NotifyURL  string `json:"notifyUrl"`
		ReturnURL  string `json:"returnUrl"`
		IsSandbox  bool   `json:"isSandbox"`
		Enabled    bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx := r.Context()
	var existing models.PaymentProviderConfig
	findErr := h.db.PaymentConfigs().FindOne(ctx, bson.M{"key": paymentregistry.KeyAlipay}).Decode(&existing)
	if findErr != nil && !errors.Is(findErr, mongo.ErrNoDocuments) {
		respondWithError(w, http.StatusInternalServerError, "Failed to load existing config")
		return
	}

	now := time.Now()
	_, upsertErr := h.db.PaymentConfigs().UpdateOne(
		ctx,
		bson.M{"key": paymentregistry.KeyAlipay},
		bson.M{
			"$set": bson.M{
				"appId":      req.AppID,
				"privateKey": keepExisting(req.PrivateKey, existing.PrivateKey),
				"publicKey":  keepExisting(req.PublicKey, existing.PublicKey),
				"notifyUrl":  req.NotifyURL,
				"returnUrl":  req.ReturnURL,
				"isSandbox":  req.IsSandbox,
				"enabled":    req.Enabled,
				"updatedAt":  now,
			},
			"$setOnInsert": bson.M{
				"_id":       primitive.NewObjectID(),
				"key":       paymentregistry.KeyAlipay,
				"createdAt": now,
			},
		},
		options.Update().SetUpsert(true),
	)
	if upsertErr != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save config")
		return
	}

	resp := map[string]string{"status": "saved"}
	if reloadErr := h.registry.Reload(ctx); reloadErr != nil {
		resp["warning"] = "Service initialization failed: " + reloadErr.Error()
	}
	respondWithJSON(w, http.StatusOK, resp)
}
```

- [ ] **Step 3.4: Run the tests**

```bash
cd backend && go test ./internal/api/handlers/... -run TestPaymentConfig -v
```

Expected: `PASS` for all 4 payment config tests.

- [ ] **Step 3.5: Run full handler test suite to confirm no regressions**

```bash
cd backend && go test ./internal/api/handlers/... -v 2>&1 | tail -20
```

Expected: All tests pass.

- [ ] **Step 3.6: Commit**

```bash
git add backend/internal/api/handlers/payment_config.go backend/internal/api/handlers/payment_config_test.go
git commit -m "feat: add PaymentConfigHandler with GET/PUT for wechat and alipay"
```

---

## Task 4: Refactor billing.go + payment_notify.go

Replace the direct service fields with `*paymentregistry.Registry` in both handlers.

**Files:**
- Modify: `backend/internal/api/handlers/billing.go`
- Modify: `backend/internal/api/handlers/payment_notify.go`
- Modify: `backend/internal/api/handlers/testhelpers_test.go`

- [ ] **Step 4.1: Update billing.go struct and constructor**

In `backend/internal/api/handlers/billing.go`:

**Remove** these import lines:
```go
alipayservice "agentstore/internal/alipay"
wechatservice "agentstore/internal/wechat"
```

**Add** this import:
```go
"agentstore/internal/paymentregistry"
```

**Replace** the `wechat` and `alipay` fields in `BillingHandler`:
```go
// Remove:
wechat       *wechatservice.Service
alipay       *alipayservice.Service

// Add:
registry     *paymentregistry.Registry
```

**Replace** `NewBillingHandler` signature and body:
```go
// Old:
func NewBillingHandler(stripeSvc *stripeservice.Service, wechatSvc *wechatservice.Service, alipaySvc *alipayservice.Service, database *db.MongoDB, emitter events.Emitter, sysLogger *syslog.Logger, store *configstore.Store) *BillingHandler {
    return &BillingHandler{
        stripe:  stripeSvc,
        wechat:  wechatSvc,
        alipay:  alipaySvc,
        ...

// New:
func NewBillingHandler(stripeSvc *stripeservice.Service, reg *paymentregistry.Registry, database *db.MongoDB, emitter events.Emitter, sysLogger *syslog.Logger, store *configstore.Store) *BillingHandler {
    return &BillingHandler{
        stripe:   stripeSvc,
        registry: reg,
        ...
```

**Replace** all `h.wechat` with `h.registry.Wechat()` and `h.alipay` with `h.registry.Alipay()` throughout the file. The usages are:

```go
// enabledPaymentMethods (around line 66):
// Old:
if h.wechat != nil {
    methods = append(methods, "wechat_h5")
}
if h.alipay != nil {
    methods = append(methods, "alipay")
}
// New:
if h.registry.Wechat() != nil {
    methods = append(methods, "wechat_h5")
}
if h.registry.Alipay() != nil {
    methods = append(methods, "alipay")
}

// createDomesticPayOrder case "wechat_h5" (around line 441):
// Old:
if h.wechat == nil { ... }
result, err := h.wechat.CreateH5Order(...)
// New:
if h.registry.Wechat() == nil { ... }
result, err := h.registry.Wechat().CreateH5Order(...)

// createDomesticPayOrder case "alipay" (around line 452):
// Old:
if h.alipay == nil { ... }
result, err := h.alipay.CreatePayOrder(...)
// New:
if h.registry.Alipay() == nil { ... }
result, err := h.registry.Alipay().CreatePayOrder(...)
```

- [ ] **Step 4.2: Update payment_notify.go struct and constructor**

In `backend/internal/api/handlers/payment_notify.go`:

**Remove** imports:
```go
alipayservice "agentstore/internal/alipay"
wechatservice "agentstore/internal/wechat"
```

**Add** import:
```go
"agentstore/internal/paymentregistry"
```

**Replace** struct fields:
```go
// Remove:
wechat *wechatservice.Service
alipay *alipayservice.Service

// Add:
registry *paymentregistry.Registry
```

**Replace** `NewPaymentNotifyHandler` signature:
```go
// Old:
func NewPaymentNotifyHandler(
    wechatSvc *wechatservice.Service,
    alipaySvc *alipayservice.Service,
    stripeSvc *stripeservice.Service,
    ...
) *PaymentNotifyHandler {
    return &PaymentNotifyHandler{
        wechat: wechatSvc,
        alipay: alipaySvc,
        ...

// New:
func NewPaymentNotifyHandler(
    reg *paymentregistry.Registry,
    stripeSvc *stripeservice.Service,
    ...
) *PaymentNotifyHandler {
    return &PaymentNotifyHandler{
        registry: reg,
        ...
```

**Replace** all `h.wechat` with `h.registry.Wechat()` and `h.alipay` with `h.registry.Alipay()`:

```go
// HandleWechatNotify (around line 59):
// Old: if h.wechat == nil { ... }  result, err := h.wechat.ParseAndVerifyNotify(r)
// New: if h.registry.Wechat() == nil { ... }  result, err := h.registry.Wechat().ParseAndVerifyNotify(r)

// HandleAlipayNotify (around line 98):
// Old: if h.alipay == nil { ... }  bm, err := h.alipay.ParseAndVerifyNotify(r)
// New: if h.registry.Alipay() == nil { ... }  bm, err := h.registry.Alipay().ParseAndVerifyNotify(r)
```

- [ ] **Step 4.3: Update testhelpers_test.go**

In `backend/internal/api/handlers/testhelpers_test.go`, find the line:

```go
billingHandler := NewBillingHandler(nil, nil, nil, sharedDB, emitter, sysLogger, cfgStore)
```

Change it to:

```go
billingHandler := NewBillingHandler(nil, nil, sharedDB, emitter, sysLogger, cfgStore)
```

(Remove the two nil service args; the second nil now represents the registry which is nil-safe.)

- [ ] **Step 4.4: Build**

```bash
cd backend && go build ./...
```

Expected: no errors.

- [ ] **Step 4.5: Run full test suite**

```bash
cd backend && go test ./...
```

Expected: all tests pass.

- [ ] **Step 4.6: Commit**

```bash
git add backend/internal/api/handlers/billing.go backend/internal/api/handlers/payment_notify.go backend/internal/api/handlers/testhelpers_test.go
git commit -m "refactor: billing + payment_notify handlers use paymentregistry.Registry"
```

---

## Task 5: Wire main.go

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 5.1: Replace wechat/alipay service init with registry**

In `backend/cmd/server/main.go`:

**Remove** imports:
```go
alipayservice "agentstore/internal/alipay"
wechatservice "agentstore/internal/wechat"
```

**Add** import:
```go
"agentstore/internal/paymentregistry"
```

**Replace** the wechat/alipay service initialization block (currently around lines 246–265):

```go
// Old block to remove:
var wechatSvc *wechatservice.Service
if wechatSvc, err = wechatservice.New(cfg.WeChatPay); err != nil { ... }
if wechatSvc != nil { slog.Info("WeChat Pay configured") }

var alipaySvc *alipayservice.Service
if alipaySvc, err = alipayservice.New(cfg.Alipay); err != nil { ... }
if alipaySvc != nil { slog.Info("Alipay configured") }

// New block to add:
paymentReg := paymentregistry.New(database)

// Seed from env/yaml on first startup (one-time migration)
if seedErr := paymentReg.Seed(ctx, paymentregistry.WechatSeedConfig{
    AppID:        cfg.WeChatPay.AppID,
    MchID:        cfg.WeChatPay.MchID,
    APIv3Key:     cfg.WeChatPay.APIv3Key,
    PrivateKey:   cfg.WeChatPay.PrivateKey,
    CertSerialNo: cfg.WeChatPay.CertSerialNo,
    NotifyURL:    cfg.WeChatPay.NotifyURL,
}, paymentregistry.AlipaySeedConfig{
    AppID:      cfg.Alipay.AppID,
    PrivateKey: cfg.Alipay.PrivateKey,
    PublicKey:  cfg.Alipay.PublicKey,
    NotifyURL:  cfg.Alipay.NotifyURL,
    ReturnURL:  cfg.Alipay.ReturnURL,
    IsSandbox:  cfg.Alipay.IsSandbox,
}); seedErr != nil {
    slog.Warn("payment registry seed failed", "error", seedErr)
}

if reloadErr := paymentReg.Reload(ctx); reloadErr != nil {
    slog.Warn("payment registry initial load failed", "error", reloadErr)
}
if paymentReg.Wechat() != nil {
    slog.Info("WeChat Pay configured (from DB)")
}
if paymentReg.Alipay() != nil {
    slog.Info("Alipay configured (from DB)")
}
```

- [ ] **Step 5.2: Update handler constructors**

Find the billing and payment notify handler construction (around lines 375–377):

```go
// Old:
billingHandler := handlers.NewBillingHandler(stripeSvc, wechatSvc, alipaySvc, database, emitter, sysLogger, cfgStore)
paymentNotifyHandler := handlers.NewPaymentNotifyHandler(wechatSvc, alipaySvc, stripeSvc, database, emitter, sysLogger)

// New:
billingHandler := handlers.NewBillingHandler(stripeSvc, paymentReg, database, emitter, sysLogger, cfgStore)
paymentNotifyHandler := handlers.NewPaymentNotifyHandler(paymentReg, stripeSvc, database, emitter, sysLogger)
```

- [ ] **Step 5.3: Create PaymentConfigHandler and register routes**

After the existing `llmConfigHandler` creation (around line 393), add:

```go
paymentConfigHandler := handlers.NewPaymentConfigHandler(database, paymentReg)
```

Then in the admin route section (after the `llm-config` routes at line 856–857), add:

```go
// Payment provider configuration (admin only)
adminWrite.HandleFunc("/payment-config/wechat", paymentConfigHandler.GetWechat).Methods("GET")
adminWrite.HandleFunc("/payment-config/wechat", paymentConfigHandler.UpdateWechat).Methods("PUT")
adminWrite.HandleFunc("/payment-config/alipay", paymentConfigHandler.GetAlipay).Methods("GET")
adminWrite.HandleFunc("/payment-config/alipay", paymentConfigHandler.UpdateAlipay).Methods("PUT")
```

- [ ] **Step 5.4: Build and test**

```bash
cd backend && go build ./... && go vet ./... && go test ./...
```

Expected: clean build, all tests pass.

- [ ] **Step 5.5: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat: wire PaymentServiceRegistry and PaymentConfigHandler routes in main.go"
```

---

## Task 6: Frontend — API client, route, sidebar

**Files:**
- Modify: `frontend/src/api/client.ts`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/AdminLayout.tsx`

- [ ] **Step 6.1: Add API methods to client.ts**

In `frontend/src/api/client.ts`, find the `adminApi` object. Add these two methods alongside the existing `getLLMConfig` / `updateLLMConfig`:

```ts
getPaymentConfig: (provider: 'wechat' | 'alipay') =>
  api.get<Record<string, unknown>>(`/admin/payment-config/${provider}`).then(r => r.data),

updatePaymentConfig: (provider: 'wechat' | 'alipay', data: Record<string, unknown>) =>
  api.put<{ status: string; warning?: string }>(`/admin/payment-config/${provider}`, data).then(r => r.data),
```

- [ ] **Step 6.2: Add route to App.tsx**

In `frontend/src/App.tsx`:

Add lazy import near the existing admin lazy imports:
```ts
const AdminPaymentConfigPage = lazy(() => import('./pages/admin/PaymentConfigPage'));
```

Add route inside the admin `<Route path="admin">` block, next to the `llm-config` route:
```tsx
<Route path="payment-config" element={<Suspense fallback={<LazyFallback />}><AdminPaymentConfigPage /></Suspense>} />
```

- [ ] **Step 6.3: Add sidebar nav item to AdminLayout.tsx**

In `frontend/src/components/AdminLayout.tsx`:

Add `Wallet` to the lucide-react import (or use `CreditCard` which is already imported):

Find the nav items array (around line 59) and add the payment-config item after `financial`:

```ts
{ path: '/admin/payment-config', icon: CreditCard, label: 'Payment Config' },
```

Place it after the `{ path: '/admin/financial', icon: DollarSign, label: 'Financial' },` line.

The LLM Config item is owner-only; make Payment Config also owner-only to match:
```ts
...(role === 'owner' ? [
  { path: '/admin/llm-config', icon: Cpu, label: 'LLM Config' },
  { path: '/admin/payment-config', icon: CreditCard, label: 'Payment Config' },
] : []),
```

(Replace the existing `...(role === 'owner' ? [{ path: '/admin/llm-config'...}] : [])` with the above.)

- [ ] **Step 6.4: TypeScript check**

```bash
cd frontend && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 6.5: Commit**

```bash
git add frontend/src/api/client.ts frontend/src/App.tsx frontend/src/components/AdminLayout.tsx
git commit -m "feat: add payment-config route and sidebar nav to admin"
```

---

## Task 7: PaymentConfigPage.tsx

**Files:**
- Create: `frontend/src/pages/admin/PaymentConfigPage.tsx`

- [ ] **Step 7.1: Create the page**

Create `frontend/src/pages/admin/PaymentConfigPage.tsx`:

```tsx
import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Save, Loader2, CreditCard, ShieldAlert, CheckCircle2, Circle } from 'lucide-react';
import { toast } from 'sonner';
import { adminApi } from '../../api/client';
import LoadingSpinner from '../../components/LoadingSpinner';

const MASKED = '••••••';

function ConfiguredBadge({ configured }: { configured: boolean }) {
  return configured ? (
    <span className="flex items-center gap-1 text-xs font-medium text-emerald-400">
      <CheckCircle2 className="w-3.5 h-3.5" /> 已配置
    </span>
  ) : (
    <span className="flex items-center gap-1 text-xs font-medium text-dark-500">
      <Circle className="w-3.5 h-3.5" /> 未配置
    </span>
  );
}

// ── WeChat Pay Section ───────────────────────────────────────────────────────

interface WechatForm {
  appId: string;
  mchId: string;
  apiV3Key: string;
  privateKey: string;
  certSerialNo: string;
  notifyUrl: string;
  enabled: boolean;
}

function WechatSection() {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<WechatForm>({
    appId: '', mchId: '', apiV3Key: '', privateKey: '',
    certSerialNo: '', notifyUrl: '', enabled: false,
  });

  const { data, isLoading } = useQuery({
    queryKey: ['payment-config', 'wechat'],
    queryFn: () => adminApi.getPaymentConfig('wechat'),
  });

  useEffect(() => {
    if (data) {
      setForm({
        appId:        (data.appId as string) || '',
        mchId:        (data.mchId as string) || '',
        apiV3Key:     (data.apiV3Key as string) === MASKED ? '' : (data.apiV3Key as string) || '',
        privateKey:   (data.privateKey as string) === MASKED ? '' : (data.privateKey as string) || '',
        certSerialNo: (data.certSerialNo as string) || '',
        notifyUrl:    (data.notifyUrl as string) || '',
        enabled:      (data.enabled as boolean) ?? false,
      });
    }
  }, [data]);

  const mutation = useMutation({
    mutationFn: (d: WechatForm) => adminApi.updatePaymentConfig('wechat', d),
    onSuccess: (res) => {
      if (res.warning) {
        toast.warning('已保存，但服务初始化失败：' + res.warning);
      } else {
        toast.success('微信支付配置已保存');
      }
      queryClient.invalidateQueries({ queryKey: ['payment-config', 'wechat'] });
    },
    onError: () => toast.error('保存失败'),
  });

  if (isLoading) return <LoadingSpinner size="sm" className="py-8" />;

  const configured = !!(data?.configured);

  return (
    <div className="bg-dark-900/50 border border-dark-800 rounded-2xl p-6">
      <div className="flex items-center justify-between mb-5">
        <h2 className="text-lg font-semibold text-white flex items-center gap-2">
          <CreditCard className="w-5 h-5 text-primary-400" />
          微信支付 (WeChat Pay H5)
        </h2>
        <ConfiguredBadge configured={configured} />
      </div>

      <form onSubmit={(e) => { e.preventDefault(); mutation.mutate(form); }} className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <Field label="App ID" value={form.appId} onChange={v => setForm(f => ({ ...f, appId: v }))} placeholder="wx_xxxxxxxxxxxxxxxxxx" />
          <Field label="商户号 (Mch ID)" value={form.mchId} onChange={v => setForm(f => ({ ...f, mchId: v }))} placeholder="1234567890" />
        </div>
        <Field
          label="API v3 密钥"
          value={form.apiV3Key}
          onChange={v => setForm(f => ({ ...f, apiV3Key: v }))}
          type="password"
          placeholder={data?.apiV3Key === MASKED ? '已保存 (不修改请留空)' : '32位字符'}
        />
        <div>
          <label className="block text-sm font-medium text-dark-300 mb-2">商户私钥 PEM</label>
          <textarea
            rows={4}
            value={form.privateKey}
            onChange={e => setForm(f => ({ ...f, privateKey: e.target.value }))}
            placeholder={data?.privateKey === MASKED ? '已保存 (不修改请留空)' : '-----BEGIN RSA PRIVATE KEY-----\n...'}
            className="w-full px-4 py-3 bg-dark-800 border border-dark-700 rounded-lg text-white placeholder-dark-500 focus:outline-none focus:border-primary-500 font-mono text-xs resize-none"
          />
        </div>
        <Field label="证书序列号" value={form.certSerialNo} onChange={v => setForm(f => ({ ...f, certSerialNo: v }))} placeholder="ABCDEF123456..." />
        <Field label="回调通知 URL" value={form.notifyUrl} onChange={v => setForm(f => ({ ...f, notifyUrl: v }))} placeholder="https://yourdomain.com/api/billing/wechat/notify" />

        <label className="flex items-center gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={form.enabled}
            onChange={e => setForm(f => ({ ...f, enabled: e.target.checked }))}
            className="w-5 h-5 rounded border-dark-700 bg-dark-800 text-primary-500 focus:ring-primary-500"
          />
          <span className="text-dark-300 text-sm">启用微信支付</span>
        </label>

        <button
          type="submit"
          disabled={mutation.isPending}
          className="flex items-center gap-2 px-5 py-2.5 bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
        >
          {mutation.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
          保存微信支付配置
        </button>
      </form>
    </div>
  );
}

// ── Alipay Section ───────────────────────────────────────────────────────────

interface AlipayForm {
  appId: string;
  privateKey: string;
  publicKey: string;
  notifyUrl: string;
  returnUrl: string;
  isSandbox: boolean;
  enabled: boolean;
}

function AlipaySection() {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<AlipayForm>({
    appId: '', privateKey: '', publicKey: '',
    notifyUrl: '', returnUrl: '', isSandbox: true, enabled: false,
  });

  const { data, isLoading } = useQuery({
    queryKey: ['payment-config', 'alipay'],
    queryFn: () => adminApi.getPaymentConfig('alipay'),
  });

  useEffect(() => {
    if (data) {
      setForm({
        appId:      (data.appId as string) || '',
        privateKey: (data.privateKey as string) === MASKED ? '' : (data.privateKey as string) || '',
        publicKey:  (data.publicKey as string) === MASKED ? '' : (data.publicKey as string) || '',
        notifyUrl:  (data.notifyUrl as string) || '',
        returnUrl:  (data.returnUrl as string) || '',
        isSandbox:  (data.isSandbox as boolean) ?? true,
        enabled:    (data.enabled as boolean) ?? false,
      });
    }
  }, [data]);

  const mutation = useMutation({
    mutationFn: (d: AlipayForm) => adminApi.updatePaymentConfig('alipay', d),
    onSuccess: (res) => {
      if (res.warning) {
        toast.warning('已保存，但服务初始化失败：' + res.warning);
      } else {
        toast.success('支付宝配置已保存');
      }
      queryClient.invalidateQueries({ queryKey: ['payment-config', 'alipay'] });
    },
    onError: () => toast.error('保存失败'),
  });

  if (isLoading) return <LoadingSpinner size="sm" className="py-8" />;

  const configured = !!(data?.configured);

  return (
    <div className="bg-dark-900/50 border border-dark-800 rounded-2xl p-6">
      <div className="flex items-center justify-between mb-5">
        <h2 className="text-lg font-semibold text-white flex items-center gap-2">
          <CreditCard className="w-5 h-5 text-blue-400" />
          支付宝 (Alipay)
        </h2>
        <ConfiguredBadge configured={configured} />
      </div>

      <form onSubmit={(e) => { e.preventDefault(); mutation.mutate(form); }} className="space-y-4">
        <Field label="App ID" value={form.appId} onChange={v => setForm(f => ({ ...f, appId: v }))} placeholder="2021000000000000" />
        <div>
          <label className="block text-sm font-medium text-dark-300 mb-2">应用私钥 PEM</label>
          <textarea
            rows={4}
            value={form.privateKey}
            onChange={e => setForm(f => ({ ...f, privateKey: e.target.value }))}
            placeholder={data?.privateKey === MASKED ? '已保存 (不修改请留空)' : '-----BEGIN RSA PRIVATE KEY-----\n...'}
            className="w-full px-4 py-3 bg-dark-800 border border-dark-700 rounded-lg text-white placeholder-dark-500 focus:outline-none focus:border-primary-500 font-mono text-xs resize-none"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-dark-300 mb-2">支付宝公钥 PEM</label>
          <textarea
            rows={4}
            value={form.publicKey}
            onChange={e => setForm(f => ({ ...f, publicKey: e.target.value }))}
            placeholder={data?.publicKey === MASKED ? '已保存 (不修改请留空)' : '-----BEGIN PUBLIC KEY-----\n...'}
            className="w-full px-4 py-3 bg-dark-800 border border-dark-700 rounded-lg text-white placeholder-dark-500 focus:outline-none focus:border-primary-500 font-mono text-xs resize-none"
          />
        </div>
        <Field label="异步通知 URL" value={form.notifyUrl} onChange={v => setForm(f => ({ ...f, notifyUrl: v }))} placeholder="https://yourdomain.com/api/billing/alipay/notify" />
        <Field label="同步跳转 URL" value={form.returnUrl} onChange={v => setForm(f => ({ ...f, returnUrl: v }))} placeholder="https://yourdomain.com/billing/success" />

        <label className="flex items-center gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={form.isSandbox}
            onChange={e => setForm(f => ({ ...f, isSandbox: e.target.checked }))}
            className="w-5 h-5 rounded border-dark-700 bg-dark-800 text-primary-500 focus:ring-primary-500"
          />
          <span className="text-dark-300 text-sm">沙箱模式 (开发测试用)</span>
        </label>

        <label className="flex items-center gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={form.enabled}
            onChange={e => setForm(f => ({ ...f, enabled: e.target.checked }))}
            className="w-5 h-5 rounded border-dark-700 bg-dark-800 text-primary-500 focus:ring-primary-500"
          />
          <span className="text-dark-300 text-sm">启用支付宝</span>
        </label>

        <button
          type="submit"
          disabled={mutation.isPending}
          className="flex items-center gap-2 px-5 py-2.5 bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
        >
          {mutation.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
          保存支付宝配置
        </button>
      </form>
    </div>
  );
}

// ── Shared Field Component ───────────────────────────────────────────────────

function Field({
  label, value, onChange, placeholder, type = 'text',
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  type?: string;
}) {
  return (
    <div>
      <label className="block text-sm font-medium text-dark-300 mb-2">{label}</label>
      <input
        type={type}
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        className="w-full px-4 py-3 bg-dark-800 border border-dark-700 rounded-lg text-white placeholder-dark-500 focus:outline-none focus:border-primary-500"
      />
    </div>
  );
}

// ── Page ─────────────────────────────────────────────────────────────────────

export default function PaymentConfigPage() {
  return (
    <div className="max-w-2xl mx-auto">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-white flex items-center gap-3">
          <CreditCard className="w-7 h-7 text-primary-400" />
          Payment Config
        </h1>
        <p className="text-dark-400 mt-1">
          配置国内支付渠道凭证，保存后立即生效（无需重启）
        </p>
      </div>

      <div className="space-y-6">
        <WechatSection />
        <AlipaySection />
      </div>

      <div className="mt-6 p-4 rounded-xl bg-dark-900/30 border border-dark-800 text-xs text-dark-500">
        <ShieldAlert className="inline w-3.5 h-3.5 mr-1" />
        私钥等敏感字段在显示时已脱敏。留空字段保存时不会覆盖原值。订阅套餐计费仍走 Stripe，此处仅管理积分包的国内支付渠道。
      </div>
    </div>
  );
}
```

- [ ] **Step 7.2: TypeScript check**

```bash
cd frontend && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 7.3: Lint check**

```bash
cd frontend && npm run lint
```

Expected: 0 errors.

- [ ] **Step 7.4: Commit**

```bash
git add frontend/src/pages/admin/PaymentConfigPage.tsx
git commit -m "feat: add PaymentConfigPage admin UI for WeChat/Alipay credentials"
```

---

## Task 8: Final Verification

- [ ] **Step 8.1: Full backend build + test**

```bash
cd backend && go build ./... && go vet ./... && go test ./...
```

Expected: clean build, all tests pass.

- [ ] **Step 8.2: Full frontend check**

```bash
cd frontend && npx tsc --noEmit && npm run lint && npm test -- --run
```

Expected: 0 type errors, 0 lint errors, all tests pass.

- [ ] **Step 8.3: Manual smoke test**

1. Start the server: `set -a && source .env && set +a && cd backend && go run ./cmd/server`
2. Start the frontend: `cd frontend && npm run dev`
3. Log in as an owner account, go to `/admin/payment-config`
4. Verify page loads with "未配置" badges on both sections
5. Enter dummy values in the WeChat section with `enabled: false`, click Save
6. Verify toast "微信支付配置已保存" appears
7. Refresh the page — verify mchId / appId are still shown, sensitive fields show empty placeholders
8. Leave apiV3Key empty and save again — verify mchId still shows
9. Toggle "启用微信支付" on then off, save — verify the toggle persists

- [ ] **Step 8.4: Final commit**

```bash
git add -p  # stage any stray changes
git commit -m "chore: payment config admin page complete"
```
