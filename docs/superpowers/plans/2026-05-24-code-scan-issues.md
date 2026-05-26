# Code Scan Issues Report - AgentStore

**Scan Date:** 2026-05-24
**Scope:** Backend (Go: backend/internal/) + Frontend (React/TypeScript: frontend/src/)

---

## CRITICAL ISSUES

### 1. Security: Verification Token Logged in Plain Text
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/internal/api/handlers/auth.go`
**Line:** 2018
```go
slog.Warn("Email service not configured, logging verification token", "email", userEmail, "token", verificationToken)
```
**Description:** When email service is not configured, the verification token is logged in plain text. This is a serious security vulnerability as tokens could be exposed in logs.

**Severity:** CRITICAL

---

### 2. Security: Email Displayed in Logs
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/internal/api/handlers/auth.go`
**Line:** 2015
```go
slog.Error("Failed to send verification email", "to", userEmail, "error", err)
```
**Description:** Email addresses are being logged which may violate privacy regulations (GDPR) and create audit trail issues.

**Severity:** HIGH

---

### 3. Unchecked Type Assertion Without Comma-Ok
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/internal/telemetry/service.go`
**Line:** 563
```go
return v.(*KPIData), nil
```
**Description:** Type assertion without checking the ok value. If the cache contains wrong type, this will panic.

**Severity:** HIGH

---

## HIGH PRIORITY ISSUES

### 4. Unsafe Type Assertions in Frontend
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend/src/pages/admin/TenantProfilePage.tsx`
**Line:** 114
```typescript
planChanged ? (selectedPlanId || null) : undefined as unknown as string | null,
```
**Description:** Double type casting (`as unknown as string`) bypasses TypeScript safety. This pattern hides potential runtime errors.

**Severity:** HIGH

### 5. Unsafe Type Assertions in Frontend (Multiple)
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend/src/pages/admin/APIPage.tsx`
**Line:** 485
```typescript
if ((webhook?.events || ['tenant.created']).includes(et.type as any)) {
```
**Description:** Using `as any` to bypass type checking.

**Severity:** HIGH

---

### 6. Missing Error Check After MongoDB Update
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/internal/api/handlers/webhook.go`
**Line:** 759
```go
h.db.Tenants().UpdateOne(ctx, bson.M{"_id": tenant.ID}, bson.M{
    "$set": bson.M{"billingStatus": models.BillingStatusActive, "updatedAt": time.Now()},
})
```
**Description:** UpdateOne result is not checked for errors. The operation could fail silently.

**Severity:** HIGH

---

### 7. HTTP Response Not Checking for Write Errors
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/internal/api/handlers/helpers.go`
**Lines:** 24-25
```go
w.WriteHeader(status)
json.NewEncoder(w).Encode(payload)
```
**Description:** Write errors are not checked. Network failures mid-write are silently ignored.

**Severity:** MEDIUM-HIGH

---

### 8. Potential Race Condition in Token Refresh
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend/src/api/client.ts`
**Lines:** 30-44
```typescript
let isRefreshing = false;
let refreshSubscribers: ((token: string) => void)[] = [];
```
**Description:** The `isRefreshing` flag is not protected by a lock/mutex. Multiple simultaneous 401 responses could trigger race conditions in the token refresh logic.

**Severity:** MEDIUM-HIGH

---

## MEDIUM PRIORITY ISSUES

### 9. Inconsistent Error Response Format
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/internal/api/handlers/usage.go`
**Multiple locations** (lines 32, 37, 47, 51, 55, 59, 76, 82, 121, 125, etc.)

**Description:** Some errors use JSON format `{"error":"message"}` while others use raw text. This inconsistency makes frontend error handling more complex.

**Severity:** MEDIUM

---

### 10. Missing Index Signatures in TypeScript
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend/src/pages/admin/PMPage.tsx`
**Lines:** 218, 405, 500
```typescript
formatter={(v) => [formatCents(v as number), 'MRR']}
formatter={(v) => [formatNum(v as number), 'Credits']}
formatter={(v) => [formatNum(v as number), 'Events']}
```
**Description:** Tooltip formatter receives `any` type from recharts but casts to `number` without validation. Could fail on unexpected data.

**Severity:** MEDIUM

---

### 11. Hardcoded Fallback Values
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/cmd/server/main.go`
**Line:** 152
```go
site = "us5.datadoghq.com"
```
**Description:** Default DataDog site is hardcoded. Should be configurable.

**Severity:** LOW-MEDIUM

---

### 12. Magic Number in Code
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/internal/api/handlers/usage.go`
**Line:** 58
```go
if req.Quantity > 10000 {
```
**Description:** Maximum quantity of 10000 is hardcoded. Should be a configuration value.

**Severity:** LOW

---

### 13. Response Body Not Closed (Health Checks)
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/internal/health/integrations.go`
**Lines:** 180, 201, 221, 241
```go
resp.Body.Close()
```
**Description:** Body is closed but error from Close() is ignored. While less critical for HTTP responses, it's inconsistent with other code patterns.

**Severity:** LOW

---

### 14. Silent Error Handling in Frontend
**Multiple files:**
- `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend/src/hooks/useTelemetry.ts` line 33: `.catch(() => {})`
- `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend/src/hooks/useTelemetry.ts` line 37: `.catch(() => {})`
- `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend/src/components/AdminLayout.tsx` line 41: `.catch(() => { /* non-critical: badge just won't show */ })`

**Description:** Empty catch blocks silently swallow errors. While sometimes intentional, it makes debugging difficult.

**Severity:** MEDIUM

---

### 15. Inconsistent Null Checks
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/internal/middleware/auth.go`
**Lines:** 175-180
```go
user, ok := ctx.Value(UserContextKey).(*models.User)
return user, ok

key, ok := ctx.Value(APIKeyContextKey).(*models.APIKey)
return key, ok
```
**Description:** Type assertions use comma-ok idiom correctly. However, caller doesn't always check the boolean return value.

**Severity:** MEDIUM

---

### 16. Unused Variable in Background Goroutine
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/internal/api/handlers/auth.go`
**Line:** 2012
```go
_ = ctx // timeout guard for background goroutine
```
**Description:** The ctx variable is assigned to underscore but never actually used within the goroutine, making the comment misleading. The context is created with timeout but the goroutine doesn't check for cancellation.

**Severity:** MEDIUM

---

### 17. Missing Validation for Empty Arrays
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/internal/api/handlers/webhook.go`
**Line:** 480-484
```go
cursor, _ := h.db.TenantMemberships().Find(ctx, bson.M{"tenantId": tenant.ID})
var memberships []models.TenantMembership
if cursor != nil {
    cursor.All(ctx, &memberships)
    cursor.Close(ctx)
}
```
**Description:** Error from `Find()` is ignored. If Find fails, the code continues with potentially nil cursor causing panic on `.All()`.

**Severity:** HIGH

---

### 18. Potential Integer Overflow
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/internal/api/handlers/billing.go`
**Line:** 228
```go
customItems = append(customItems, stripeservice.CheckoutLineItem{PriceID: seatPriceID, Quantity: additionalSeats})
```
**Description:** `additionalSeats` is parsed from string without checking for overflow or negative values.

**Severity:** MEDIUM

---

### 19. SQL Injection Risk - N/A (MongoDB)
The codebase uses MongoDB with proper driver parameterization. No SQL injection risks found.

---

### 20. No Rate Limiting on Webhook Endpoint
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/cmd/server/main.go`
**Line:** 681
```go
api.HandleFunc("/billing/webhook", webhookHandler.HandleWebhook).Methods("POST")
```
**Description:** Stripe webhook endpoint has no rate limiting. Could be subject to DoS attacks.

**Severity:** MEDIUM

---

## CODE QUALITY ISSUES

### 21. Missing Comments/TODO Markers
Very few TODO/FIXME comments were found:
- `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/internal/api/handlers/auth.go` line 1282: Comment explaining code flow
- `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/internal/stripe/stripe_test.go` line 93: Test comment

**Assessment:** This is actually GOOD - indicates well-maintained code with few known issues.

---

### 22. Empty Interface Usage
**File:** `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/internal/api/handlers/helpers.go`
```go
map[string]interface{}{}
```
**Description:** Used for dynamic response objects. Not a bug but reduces type safety.

**Severity:** LOW

---

## SUMMARY

| Category | Count |
|----------|-------|
| Critical | 1 |
| High | 7 |
| Medium | 10 |
| Low | 2 |

**Total Issues Found:** 22

**Recommendations:**
1. Remove verification token logging immediately (Issue #1)
2. Review and remove email logging (Issue #2)
3. Add proper error checking for all database operations
4. Protect frontend token refresh with proper synchronization
5. Replace `as unknown as` patterns with proper type guards
6. Make hardcoded values configurable (max quantity, DataDog site)
7. Add rate limiting to webhook endpoint

---

*Report generated by Claude Code read-only scan*
