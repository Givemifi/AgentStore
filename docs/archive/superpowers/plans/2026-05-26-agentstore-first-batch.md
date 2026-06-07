# AgentStore First Batch Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert the current AgentStore-derived app into the first AgentStore launch batch: product identity rename, marketplace-focused frontend, platform-published agent source, and a working chat path.

**Architecture:** Keep the existing Go + React + MongoDB architecture, but rename the project identity to AgentStore and route normal users through a curated agent marketplace. Platform agents are root/platform-published supply; users keep their own tenant for auth, credits, billing, and conversations. Chat resolves agents through a single backend resolver that accepts slug or ID and persists a canonical agent ID.

**Tech Stack:** Go 1.25+/1.26, MongoDB, Gorilla Mux, React 19, TypeScript, Vite, TanStack Query, Axios, Stripe, OpenAI-compatible LLM API.

---

## File Structure

### Rename and identity files

- Modify: `backend/go.mod` — change Go module from `agentstore` to `agentstore`.
- Modify: `backend/**/*.go` — change imports from `agentstore/internal/...` to `agentstore/internal/...` and update product strings.
- Rename: `backend/cmd/agentstore/` to `backend/cmd/agentstore/` — CLI command package path.
- Modify: `Dockerfile` — build and run `agentstore` binary; update ldflags package path; use `AGENTSTORE_ENV`.
- Modify: `.goreleaser.yaml` — binary/project IDs and command path.
- Modify: `.env.example`, `backend/config/*.yaml`, `backend/Makefile`, `.gitignore`, `.dockerignore`, `server.json`, `manifest.json`, `smithery.yaml`, `fly.toml`, `codecov.yml` — AgentStore names and `AGENTSTORE_*` env vars.
- Modify: `README.md`, `VERSIONS.md`, `docs/**/*.md` — public documentation identity.

### Runtime configuration files

- Modify: `backend/internal/config/config.go` — read `AGENTSTORE_ENV`, with `AGENTSTORE_ENV` fallback.
- Modify: `backend/internal/db/mongodb.go` — update default database names and any old environment names.
- Modify: `backend/cmd/agentstore/cmd_mcp.go` — read `AGENTSTORE_URL`/`AGENTSTORE_API_KEY`, with old fallback if needed.

### Marketplace and navigation files

- Modify: `frontend/src/components/Layout.tsx` — default normal-user nav becomes Agents, Credits, History, Settings; keep Admin for root members.
- Modify: `frontend/src/App.tsx` — remove normal exposure of test entitlement route and add credits/history route behavior if needed.
- Modify: `frontend/src/pages/app/DashboardPage.tsx` — make it an AgentStore marketplace page and use platform agents.
- Modify: `frontend/src/pages/app/ChatPage.tsx` — use canonical agent IDs from the backend for chat requests while keeping slug URLs.
- Modify: `frontend/src/pages/app/SettingsPage.tsx` — do not expose normal-user Agent/Model management in the first user-facing settings tabs.
- Modify: `frontend/src/contexts/BrandingContext.tsx`, auth pages, bootstrap page, admin about/branding pages — default AgentStore identity.

### Agent/chat backend files

- Modify: `backend/internal/api/handlers/chat.go` — add a unified agent resolver; list platform-published agents; store canonical agent ID in conversations/messages/usage metadata; keep slug lookup for URLs.
- Modify: `backend/internal/api/handlers/agent.go` — keep tenant admin management, but marketplace uses platform published source.
- Modify: `backend/internal/api/handlers/auth.go` — grant trial credits on personal tenant creation.
- Modify: `backend/internal/models/agent.go` — add helper methods only if necessary; do not change schema unless required.
- Modify: `backend/internal/api/handlers/chat_test.go` — add regression tests for slug routing/canonical ID and marketplace platform agents.
- Modify: `frontend/src/pages/app/ChatPage.test.tsx`, `frontend/src/pages/app/DashboardPage.test.tsx` — update frontend expectations.

---

### Task 1: Rename Go module, imports, CLI path, and runtime identifiers

**Files:**
- Modify: `backend/go.mod`
- Rename: `backend/cmd/agentstore/` to `backend/cmd/agentstore/`
- Modify: all `backend/**/*.go`
- Modify: `Dockerfile`, `.goreleaser.yaml`, `backend/Makefile`, `.gitignore`, `.dockerignore`, `.env.example`, `backend/config/*.yaml`

- [ ] **Step 1: Write a rename audit output**

Run:

```bash
grep -R "AgentStore\|agentstore\|LASTSAAS\|last-saas" -n --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=dist . > /tmp/agentstore-rename-before.txt || true
```

Expected: `/tmp/agentstore-rename-before.txt` contains old-name references.

- [ ] **Step 2: Rename CLI directory**

Run:

```bash
mv backend/cmd/agentstore backend/cmd/agentstore
```

Expected: `backend/cmd/agentstore/main.go` exists.

- [ ] **Step 3: Rewrite Go module/imports and command strings**

Use a scripted replacement over tracked text files:

```bash
python3 - <<'PY'
from pathlib import Path
pairs = [
    ('agentstore/internal/', 'agentstore/internal/'),
    ('module agentstore', 'module agentstore'),
    ('./cmd/agentstore', './cmd/agentstore'),
    ('cmd/agentstore', 'cmd/agentstore'),
    ('agentstore setup', 'agentstore setup'),
    ('agentstore start', 'agentstore start'),
    ('agentstore stop', 'agentstore stop'),
    ('agentstore restart', 'agentstore restart'),
    ('agentstore change-password', 'agentstore change-password'),
    ('agentstore send-message', 'agentstore send-message'),
    ('agentstore transfer-root-owner', 'agentstore transfer-root-owner'),
    ('agentstore config', 'agentstore config'),
    ('agentstore users', 'agentstore users'),
    ('agentstore tenants', 'agentstore tenants'),
    ('agentstore logs', 'agentstore logs'),
    ('agentstore health', 'agentstore health'),
    ('agentstore stats', 'agentstore stats'),
    ('agentstore financial', 'agentstore financial'),
    ('agentstore doctor', 'agentstore doctor'),
    ('agentstore db', 'agentstore db'),
    ('agentstore mcp', 'agentstore mcp'),
    ('agentstore-server', 'agentstore-server'),
    ('agentstore-cli', 'agentstore-cli'),
    ('AGENTSTORE_ENV', 'AGENTSTORE_ENV'),
    ('AGENTSTORE_URL', 'AGENTSTORE_URL'),
    ('AGENTSTORE_API_KEY', 'AGENTSTORE_API_KEY'),
    ('agentstore-test', 'agentstore-test'),
    ('agentstore-dev', 'agentstore-dev'),
    ('agentstore-admin', 'agentstore-admin'),
    ('agentstore://', 'agentstore://'),
    ('AgentStore-Test', 'AgentStore-Test'),
    ('AgentStore', 'AgentStore'),
    ('agentstore', 'agentstore'),
]
text_suffixes = {'.go','.mod','.sum','.yaml','.yml','.toml','.json','.md','.txt','.tsx','.ts','.js','.jsx','.example','.gitignore','.dockerignore','.Dockerfile'}
for path in Path('.').rglob('*'):
    if path.is_dir() or any(part in {'.git','node_modules','dist'} for part in path.parts):
        continue
    if path.name in {'package-lock.json'}:
        continue
    if path.suffix not in text_suffixes and path.name not in {'Dockerfile','Makefile','CLAUDE.md','README.md','VERSIONS.md','VERSION','server.json','manifest.json','smithery.yaml','codecov.yml','fly.toml','.env.example','.goreleaser.yaml'}:
        continue
    try:
        data = path.read_text()
    except UnicodeDecodeError:
        continue
    next_data = data
    for old, new in pairs:
        next_data = next_data.replace(old, new)
    if next_data != data:
        path.write_text(next_data)
PY
```

Expected: imports use `agentstore/internal/...`; product strings mostly say AgentStore.

- [ ] **Step 4: Add env fallback for runtime compatibility**

In `backend/internal/config/config.go`, ensure `GetEnv()` reads `AGENTSTORE_ENV` first and falls back to `AGENTSTORE_ENV`.

Expected code:

```go
func GetEnv() string {
    if env := os.Getenv("AGENTSTORE_ENV"); env != "" {
        return env
    }
    if env := os.Getenv("AGENTSTORE_ENV"); env != "" {
        return env
    }
    return "dev"
}
```

- [ ] **Step 5: Add MCP env fallback**

In `backend/cmd/agentstore/cmd_mcp.go`, ensure URL/API key read `AGENTSTORE_*` first and old names second.

Expected helper:

```go
func getEnvWithFallback(primary, legacy string) string {
    if value := os.Getenv(primary); value != "" {
        return value
    }
    return os.Getenv(legacy)
}
```

- [ ] **Step 6: Verify rename build state**

Run:

```bash
cd backend && go test ./...
cd backend && go build ./...
```

Expected: both commands pass or expose specific failures to fix before continuing.

---

### Task 2: Rename public docs and frontend identity to AgentStore

**Files:**
- Modify: `README.md`, `VERSIONS.md`, `docs/**/*.md`
- Modify: `frontend/src/contexts/BrandingContext.tsx`
- Modify: `frontend/src/pages/BootstrapPage.tsx`
- Modify: `frontend/src/pages/admin/AboutPage.tsx`
- Modify: `frontend/src/pages/app/TestEntitlementsPage.tsx`
- Modify: `backend/internal/api/handlers/docs.go`, `backend/internal/api/handlers/openapi.go`

- [ ] **Step 1: Rewrite README as AgentStore product doc**

Replace the old boilerplate-focused introduction with AgentStore marketplace positioning:

```markdown
# AgentStore

AgentStore is a self-hosted marketplace for commercial AI agents. It combines user accounts, platform-managed agents, LLM configuration, usage credits, Stripe-powered credit packs, and an admin console so a small team can launch and operate an AI-agent business quickly.
```

Expected: README no longer introduces the project as AgentStore.

- [ ] **Step 2: Update frontend default branding**

In `frontend/src/contexts/BrandingContext.tsx`, default app name should be `AgentStore` and tagline should describe commercial agents.

Expected defaults:

```ts
appName: 'AgentStore',
tagline: 'Launch, sell, and monetize AI agents',
```

- [ ] **Step 3: Update setup instructions**

In `frontend/src/pages/BootstrapPage.tsx`, commands must reference `go run ./cmd/agentstore setup`, `status`, and `change-password`.

- [ ] **Step 4: Update API docs titles**

In backend API docs handlers, replace public API titles and examples with AgentStore.

- [ ] **Step 5: Verify visible old-name grep**

Run:

```bash
grep -R "AgentStore\|agentstore\|LASTSAAS" -n --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=dist . || true
```

Expected: only allowed compatibility fallback/history references remain.

---

### Task 3: Focus normal-user frontend on marketplace, credits, history, and settings

**Files:**
- Modify: `frontend/src/components/Layout.tsx`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/pages/app/DashboardPage.tsx`
- Modify: `frontend/src/pages/app/SettingsPage.tsx`

- [ ] **Step 1: Update default nav items**

In `Layout.tsx`, default nav should be:

```ts
const defaultNavItems = [
  { path: '/dashboard', icon: MessageCircle, label: 'Agents' },
  { path: '/buy-credits', icon: Zap, label: 'Credits' },
  { path: '/activity', icon: FileText, label: 'History' },
  { path: '/settings', icon: Settings, label: 'Settings' },
];
```

Do not include Team or Plan in the default normal-user nav.

- [ ] **Step 2: Disable branding nav from reintroducing hidden default SaaS items**

When `branding.navItems` is present, filter out built-in `team`, `plan`, and `test-entitlements` for normal users unless they are root members.

Expected logic:

```ts
const isPlatformAdmin = memberships.some(m => m.isRoot);
const hiddenNormalUserItems = new Set(['team', 'plan', 'test-entitlements']);
...
.filter(item => isPlatformAdmin || !hiddenNormalUserItems.has(item.id))
```

- [ ] **Step 3: Remove user-facing test entitlement route**

In `App.tsx`, remove or redirect `/test-entitlements` to `/dashboard`.

Expected route:

```tsx
<Route path="/test-entitlements" element={<Navigate to="/dashboard" replace />} />
```

- [ ] **Step 4: Update marketplace copy**

In `DashboardPage.tsx`, update hero copy to AgentStore marketplace language and keep `Start Chat` as primary CTA.

Expected hero title:

```tsx
Launch your AgentStore
```

Expected body:

```tsx
Choose a platform-published business agent, start a conversation, and turn specialist AI workflows into outcomes your team can use immediately.
```

- [ ] **Step 5: Hide normal-user agent/model management tabs**

In `SettingsPage.tsx`, show `Agents` and `Models` tabs only for root/admin roles if the current launch does not allow normal users to manage marketplace supply.

Expected behavior: normal users see Profile, Security if enabled, Sessions, Billing.

- [ ] **Step 6: Frontend typecheck**

Run:

```bash
cd frontend && npx tsc --noEmit
```

Expected: PASS.

---

### Task 4: Make platform-published agents visible to normal-user marketplace

**Files:**
- Modify: `backend/internal/api/handlers/chat.go`
- Test: `backend/internal/api/handlers/chat_test.go`

- [ ] **Step 1: Add failing backend test for platform agent visibility**

Add a test that creates a root/platform published agent and verifies a normal tenant can list it through `/api/chat/agents`.

Test intent:

```go
func TestChatListAgentsIncludesRootPublishedAgentsForNormalTenants(t *testing.T) {
    // create root tenant, normal tenant, root published agent
    // request /api/chat/agents as normal tenant user
    // expect the root published agent slug/name in response
}
```

- [ ] **Step 2: Run the test and verify failure**

Run:

```bash
cd backend && go test ./internal/api/handlers -run TestChatListAgentsIncludesRootPublishedAgentsForNormalTenants -count=1
```

Expected: FAIL until platform source support exists.

- [ ] **Step 3: Implement platform agent lookup**

In `ChatHandler.ListAgents`, after querying current-tenant published agents, query root tenant published agents and return them when the current tenant has no published agents or when platform marketplace mode is active.

Implementation rules:

- Root/platform agents are read-only marketplace supply for normal users.
- Current user's tenant still owns conversations and credits.
- Response includes both `id` and `slug`.

- [ ] **Step 4: Run targeted backend test**

Run:

```bash
cd backend && go test ./internal/api/handlers -run TestChatListAgentsIncludesRootPublishedAgentsForNormalTenants -count=1
```

Expected: PASS.

---

### Task 5: Unify chat agent identifiers and fix Start Chat path

**Files:**
- Modify: `backend/internal/api/handlers/chat.go`
- Modify: `frontend/src/pages/app/ChatPage.tsx`
- Modify: `frontend/src/pages/app/DashboardPage.tsx` if needed
- Test: `backend/internal/api/handlers/chat_test.go`
- Test: `frontend/src/pages/app/ChatPage.test.tsx`

- [ ] **Step 1: Add failing backend test for slug request and canonical persistence**

Add a test where the frontend sends `agentId: agent.slug`, and verify the created conversation stores the canonical agent ObjectID hex.

Test intent:

```go
func TestStreamMessageStoresCanonicalAgentIDForSlugRequests(t *testing.T) {
    // create published DB agent with slug
    // send stream request with agentId equal to slug
    // expect conversation.AgentID == dbAgent.ID.Hex()
}
```

- [ ] **Step 2: Run the test and verify failure**

Run:

```bash
cd backend && go test ./internal/api/handlers -run TestStreamMessageStoresCanonicalAgentIDForSlugRequests -count=1
```

Expected: FAIL until canonical resolution exists.

- [ ] **Step 3: Add resolved agent type and resolver**

In `chat.go`, add a private resolved type:

```go
type resolvedChatAgent struct {
    Agent       agents.Agent
    CanonicalID string
    Slug        string
}
```

Add a resolver that accepts static IDs, DB ObjectIDs, and slugs, and returns canonical ID and slug.

- [ ] **Step 4: Use canonical ID for conversations/messages/usage metadata**

In `SendMessage` and `StreamMessage`, replace direct `req.AgentID` persistence with `resolved.CanonicalID`. Use request slug only for lookup.

Expected persisted fields:

```go
AgentID: resolved.CanonicalID
```

Expected metadata:

```go
"agentId": resolved.CanonicalID,
"agentSlug": resolved.Slug,
```

- [ ] **Step 5: Frontend sends canonical ID after loading agent**

In `ChatPage.tsx`, compute:

```ts
const requestAgentId = agent?.id ?? agentId!;
```

Use `requestAgentId` in `chatApi.send` and `chatApi.stream` payloads, while leaving the URL as slug.

- [ ] **Step 6: Run backend/frontend tests**

Run:

```bash
cd backend && go test ./internal/api/handlers -run 'Test.*Chat|Test.*Agent' -count=1
cd frontend && npm test -- ChatPage DashboardPage
```

Expected: relevant tests pass.

---

### Task 6: Grant trial credits on user workspace creation

**Files:**
- Modify: `backend/internal/api/handlers/auth.go`
- Test: `backend/internal/api/handlers/auth_test.go`

- [ ] **Step 1: Add failing test for signup trial credits**

Add a test that registers a user and verifies the created personal tenant has subscription or purchased credits greater than zero.

Test intent:

```go
func TestRegisterGrantsTrialCreditsToPersonalTenant(t *testing.T) {
    // register user
    // load tenant from membership
    // expect tenant.PurchasedCredits or SubscriptionCredits equals default trial grant
}
```

- [ ] **Step 2: Run test and verify failure**

Run:

```bash
cd backend && go test ./internal/api/handlers -run TestRegisterGrantsTrialCreditsToPersonalTenant -count=1
```

Expected: FAIL if current tenant starts with zero credits.

- [ ] **Step 3: Add default trial credits constant**

In `auth.go`, add:

```go
const defaultTrialCredits int64 = 25
```

- [ ] **Step 4: Grant credits in personal tenant creation**

In `createPersonalTenant`, set:

```go
PurchasedCredits: defaultTrialCredits,
```

- [ ] **Step 5: Run targeted test**

Run:

```bash
cd backend && go test ./internal/api/handlers -run TestRegisterGrantsTrialCreditsToPersonalTenant -count=1
```

Expected: PASS.

---

### Task 7: Verify locally in browser

**Files:**
- No code changes unless verification reveals a root-cause bug.

- [ ] **Step 1: Run backend and frontend verification commands**

Run:

```bash
cd backend && go test ./...
cd backend && go build ./...
cd frontend && npx tsc --noEmit
cd frontend && npm test -- --run
```

Expected: all pass or failures are fixed before continuing.

- [ ] **Step 2: Start local services**

Use existing running services if they are current, otherwise start:

```bash
cd backend && AGENTSTORE_ENV=dev go run ./cmd/server
cd frontend && npm run dev -- --host 0.0.0.0 --port 4280
```

Expected: backend health returns `{"status":"ok"}` and frontend loads.

- [ ] **Step 3: Browser smoke**

Using Playwright, verify:

1. `/login` loads with AgentStore title/branding.
2. A new user can sign up.
3. User lands on marketplace.
4. At least one agent is visible.
5. `Start Chat` opens chat page.
6. Sending a message either succeeds or gives a clear model-not-configured/credit error.

Expected: no AgentStore visible in user-facing UI; chat path is usable or clearly blocked by missing external LLM config.
