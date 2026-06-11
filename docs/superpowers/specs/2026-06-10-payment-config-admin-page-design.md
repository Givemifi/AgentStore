# Payment Config Admin Page — Design Spec

**Date**: 2026-06-10  
**Status**: Approved  
**Scope**: Admin UI + backend API for hot-reloadable WeChat Pay / Alipay credential management

---

## Problem

WeChat Pay and Alipay credentials are currently only configurable via `.env` / YAML config files, requiring a server restart and filesystem access. Admin operators need to manage these credentials from the admin UI without deploying or restarting the service.

---

## Goals

- Admin can view, set, and update WeChat Pay and Alipay credentials from `/admin/payment-config`
- Changes take effect immediately (hot-reload, no restart)
- Sensitive fields (private keys) are never returned in plaintext via API
- Existing env/yaml-based credentials migrate automatically on first startup
- No impact on users if credentials are invalid — old service stays active, error is surfaced to admin

---

## Backend Architecture

### 1. PaymentServiceRegistry (`internal/paymentregistry/registry.go`)

A new package responsible for holding live service instances and reloading them when config changes.

```go
type Registry struct {
    mu     sync.RWMutex
    db     *db.MongoDB
    wechat *wechatservice.Service
    alipay *alipayservice.Service
}

func New(db *db.MongoDB) *Registry
func (r *Registry) Wechat() *wechatservice.Service   // RLock; nil = not configured
func (r *Registry) Alipay() *alipayservice.Service   // RLock; nil = not configured
func (r *Registry) Reload(ctx context.Context) error  // full lock; reads DB, rebuilds clients
func (r *Registry) Seed(ctx, wechatCfg, alipayCfg)   // one-time migration from env/yaml
```

**Reload behavior**: reads both providers from DB, calls `wechatservice.New()` and `alipayservice.New()`. If `enabled=false` or fields are empty, service is set to nil (same as unconfigured). If initialization fails (bad key format, network error downloading WeChat certs), the old service pointer is preserved and an error is returned. Never leaves the registry in a half-initialized state.

### 2. DB Collection: `payment_configs`

One document per provider. Key field: `"wechat_pay"` or `"alipay"`.

**WechatPayDBConfig fields**: `key`, `appId`, `mchId`, `apiV3Key`, `privateKey`, `certSerialNo`, `notifyUrl`, `enabled`, `createdAt`, `updatedAt`

**AlipayDBConfig fields**: `key`, `appId`, `privateKey`, `publicKey`, `notifyUrl`, `returnUrl`, `isSandbox`, `enabled`, `createdAt`, `updatedAt`

- Added to `db.MongoDB` as `PaymentConfigs()` accessor
- `paymentConfigsSchema()` added to `AllSchemas()` in `schema.go`

### 3. Handler: `internal/api/handlers/payment_config.go`

`PaymentConfigHandler` holds `db *db.MongoDB` and `registry *paymentregistry.Registry`.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/admin/payment-config/wechat` | adminWrite | Returns wechat config; sensitive fields masked |
| PUT | `/admin/payment-config/wechat` | adminWrite | Saves wechat config to DB; triggers `registry.Reload()` |
| GET | `/admin/payment-config/alipay` | adminWrite | Returns alipay config; sensitive fields masked |
| PUT | `/admin/payment-config/alipay` | adminWrite | Saves alipay config to DB; triggers `registry.Reload()` |

**Sensitive field masking**:
- GET: if field has a value in DB, return `"••••••"` instead of plaintext
- PUT: if incoming value is `"••••••"` **or empty string**, preserve existing DB value (no-op for that field). There is intentionally no way to "clear" a sensitive field to empty — to replace a key, submit a new value.

**Reload error response**: if `registry.Reload()` returns an error, respond with `HTTP 200` but include `{ "status": "saved", "warning": "Service initialization failed: <reason>" }` — config is persisted, but admin is warned.

### 4. Changes to Existing Handlers

`billing.go` and `payment_notify.go`: replace direct `h.wechat` / `h.alipay` service fields with `h.registry *paymentregistry.Registry`. Service access becomes `h.registry.Wechat()` / `h.registry.Alipay()`.

`NewBillingHandler` and `NewPaymentNotifyHandler` signatures updated to accept `*paymentregistry.Registry` instead of individual service pointers.

### 5. main.go Changes

```
startup order:
1. Create paymentregistry.Registry
2. Seed from env/yaml if DB has no records (one-time migration)
3. Call registry.Reload(ctx) to initialize services from DB
4. Pass registry to billing/notify handlers
5. Register payment-config routes under adminWrite subrouter
```

---

## Frontend

### PaymentConfigPage.tsx (`frontend/src/pages/admin/PaymentConfigPage.tsx`)

Two vertical card sections, each with an independent save action. Matches LLMConfigPage visual style (`max-w-2xl mx-auto`, `bg-dark-900/50 border border-dark-800 rounded-2xl p-6`).

**WeChat Pay section** (top):
- Status badge top-right: `已配置` (green) / `未配置` (gray) based on whether `appId` is non-empty in API response
- Fields: App ID, 商户号 (Mch ID), API v3 密钥 (password input), 商户私钥 PEM (textarea, rows=4), 证书序列号, 回调通知 URL
- Enabled toggle
- Save button with loading state

**Alipay section** (below, separated by margin):
- Same status badge pattern
- Fields: App ID, 应用私钥 PEM (textarea), 支付宝公钥 PEM (textarea), 异步通知 URL, 同步跳转 URL, 沙箱模式 (checkbox toggle)
- Enabled toggle
- Save button with loading state

**Masked field behavior**: if API returns `"••••••"` for a field, show placeholder `"已保存 (不修改请留空)"`. If user leaves field empty on save, backend preserves existing value.

**Data fetching**: `useQuery` for each provider separately. `useMutation` per save button.

### API Client (`frontend/src/api/client.ts`)

```ts
// Added to adminApi:
getPaymentConfig: (provider: 'wechat' | 'alipay') =>
  api.get(`/admin/payment-config/${provider}`)

updatePaymentConfig: (provider: 'wechat' | 'alipay', data: Record<string, unknown>) =>
  api.put(`/admin/payment-config/${provider}`, data)
```

### Routing

- Route: `/admin/payment-config` → `<PaymentConfigPage />`
- Admin sidebar: new entry "Payment Config" with a credit card or settings icon, placed near the existing Billing/Financial entry

---

## Security Rules (不可误改)

1. Private keys / API keys are **never** returned in plaintext from any GET endpoint
2. `"••••••"` sentinel value on PUT means "keep existing" — never write this literal string to DB
3. `registry.Reload()` on failure must preserve the old service — never leave registry in inconsistent state
4. Routes are under `adminWrite` (root tenant + admin/owner) — same as LLMConfig

---

## Migration Path

On startup, `registry.Seed(ctx, cfg.WeChatPay, cfg.Alipay)` checks if `payment_configs` collection is empty. If empty AND env/yaml credentials are non-empty, it writes them to DB. This runs once. After that, DB is authoritative; env/yaml is ignored for runtime service configuration (but still used for initial seed if DB is wiped).

---

## Out of Scope

- Per-tenant payment config (all tenants share the platform payment credentials)
- Stripe config in this page (Stripe is env-only and managed separately)
- Credential validation / test connection button (can be added later)
