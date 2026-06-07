# AgentStore P0 Operational Readiness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Converge the current AgentStore branch into a buildable, smoke-testable P0 operating baseline for a platform-operated agent marketplace.

**Architecture:** P0 is executed as reviewable batches, not a rewrite. This plan starts with repository convergence and route/setup cleanup, then verifies existing auth, agent, model, chat, credits, billing, and deployment surfaces through targeted tests and smoke-test preparation. Any product behavior fix discovered during verification should be implemented in the smallest touched area and covered by the existing package/page test for that area.

**Tech Stack:** Go 1.25, Gorilla Mux, MongoDB Go driver, go-playground/validator, React 19, Vite 7, TypeScript, TanStack Query, Vitest, Playwright, Fly/Docker.

---

## Scope Check

The P0 operational roadmap spans multiple subsystems: repository convergence, auth/bootstrap, agent management, model configuration, streaming chat, credits/billing, deployment, and runbooks. To keep execution safe, this plan is structured as batches with explicit checkpoints. Batch 1 is concrete cleanup and verification. Later batches are verification-first: run the named tests and only make the minimal code change in the mapped files when a check fails.

Do not add third-party creator marketplace features, revenue sharing, multi-agent orchestration, image/video execution, or a complex subscription matrix while executing this plan.

Do not commit unless the user explicitly asks for commits. Commit steps below are checkpoints with commands prepared for a user-approved commit.

## File Structure Map

### Repository convergence

- Modify `.gitignore` — replace obsolete LastSaaS binary ignores with AgentStore binary ignores.
- Modify `.dockerignore` — replace obsolete LastSaaS binary ignores with AgentStore binary ignores.
- Modify `scripts/setup.sh` — make the local setup script say AgentStore and default to `agentstore-dev`.
- Modify `frontend/e2e/admin.spec.ts` — make admin unauthenticated-route tests target `/admin`, not `/last`.
- Remove `backend/handlers.test` — Go binary artifact, not source.
- Remove `package.json` and `package-lock.json` at repo root — accidental Node files unrelated to the Vite frontend.
- Remove `backend/package.json` and `backend/package-lock.json` — accidental Node files unrelated to the Go backend.

### Backend operational areas to verify

- `backend/cmd/agentstore/main.go` — CLI dispatcher and `setup`, `doctor`, `status`, `health`, `db`, `users`, `tenants`, `financial`, `logs` commands.
- `backend/cmd/server/main.go` — HTTP route wiring for bootstrap, auth, tenant agents, tenant model settings, chat stream, billing, webhook, and admin routes.
- `backend/internal/api/handlers/bootstrap.go` — bootstrap status and guard.
- `backend/internal/api/handlers/auth.go` — register, login, refresh, logout, current user.
- `backend/internal/api/handlers/admin.go` — root admin users, tenants, dashboard, status management.
- `backend/internal/api/handlers/agent.go` — tenant-scoped agent CRUD, publish, archive.
- `backend/internal/api/handlers/model_settings.go` — tenant provider/model APIs.
- `backend/internal/api/handlers/llm_config.go` — root fallback LLM config.
- `backend/internal/api/handlers/chat.go` — static agents compatibility, non-streaming chat, streaming chat, conversations, messages.
- `backend/internal/credits/service.go` — credit deduction and usage event creation.
- `backend/internal/stripe/stripe.go` and `backend/internal/api/handlers/webhook.go` — checkout and webhook fulfillment.
- `backend/internal/db/schema.go` and `backend/internal/validation/validate.go` — schema/validation sync for modified models.

### Frontend operational areas to verify

- `frontend/src/App.tsx` — `/dashboard`, `/chat/:agentId`, `/settings`, `/settings/agents`, `/settings/models`, `/admin/*`, and compatibility `/last/*` redirect.
- `frontend/src/api/client.ts` — `agentsApi`, `tenantAgentsApi`, `tenantModelsApi`, `chatApi`, `billingApi`, `adminApi`.
- `frontend/src/pages/app/DashboardPage.tsx` — published agent discovery.
- `frontend/src/pages/app/ChatPage.tsx` — streaming chat, error display, credit updates, history.
- `frontend/src/pages/app/settings/AgentsTab.tsx` — tenant/admin agent management.
- `frontend/src/pages/app/settings/ModelSettingsTab.tsx` — provider/model config.
- `frontend/src/pages/admin/LLMConfigPage.tsx` — root fallback LLM config.
- `frontend/src/pages/app/BuyCreditsPage.tsx` and `frontend/src/pages/app/settings/BillingTab.tsx` — P0 recharge path.

### Test and verification files

- Backend tests:
  - `backend/internal/api/handlers/auth_test.go`
  - `backend/internal/api/handlers/admin_test.go`
  - `backend/internal/api/handlers/agent_test.go`
  - `backend/internal/api/handlers/chat_test.go`
  - `backend/internal/api/handlers/llm_config_test.go`
  - `backend/internal/api/handlers/model_settings_test.go`
  - `backend/internal/api/handlers/billing_test.go`
  - `backend/internal/credits/service_test.go`
  - `backend/internal/llm/router_test.go`
  - `backend/internal/validation/validate_test.go`
- Frontend tests:
  - `frontend/src/components/AdminLayout.test.tsx`
  - `frontend/src/pages/admin/LLMConfigPage.test.tsx`
  - `frontend/src/pages/app/DashboardPage.test.tsx`
  - `frontend/src/pages/app/ChatPage.test.tsx`
  - `frontend/src/pages/app/settings/AgentsTab.test.tsx`
  - `frontend/src/pages/app/settings/ModelSettingsTab.test.tsx`
- E2E tests:
  - `frontend/e2e/admin.spec.ts`
  - `frontend/e2e/auth.spec.ts`
  - `frontend/e2e/navigation.spec.ts`
  - `frontend/e2e/smoke.spec.ts`
- Smoke doc:
  - `docs/smoke-test-real-mongo-llm-billing.md`

---

## Task 1: Converge repository artifacts and visible setup naming

**Files:**
- Modify: `.gitignore`
- Modify: `.dockerignore`
- Modify: `scripts/setup.sh`
- Modify: `frontend/e2e/admin.spec.ts`
- Delete: `backend/handlers.test`
- Delete: `package.json`
- Delete: `package-lock.json`
- Delete: `backend/package.json`
- Delete: `backend/package-lock.json`

- [ ] **Step 1: Update the admin E2E route test to lock `/admin` as the advertised route**

Replace the full contents of `frontend/e2e/admin.spec.ts` with:

```ts
import { test, expect } from '@playwright/test';

test.describe('Admin panel access', () => {
  test('admin routes redirect unauthenticated users to login', async ({ page }) => {
    await page.goto('/admin');
    await expect(page).toHaveURL(/\/login/, { timeout: 10_000 });
  });

  test('admin users page requires authentication', async ({ page }) => {
    await page.goto('/admin/users');
    await expect(page).toHaveURL(/\/login/, { timeout: 10_000 });
  });

  test('admin tenants page requires authentication', async ({ page }) => {
    await page.goto('/admin/tenants');
    await expect(page).toHaveURL(/\/login/, { timeout: 10_000 });
  });

  test('admin logs page requires authentication', async ({ page }) => {
    await page.goto('/admin/logs');
    await expect(page).toHaveURL(/\/login/, { timeout: 10_000 });
  });
});
```

- [ ] **Step 2: Run the targeted E2E test if Playwright browsers are installed**

Run:

```bash
cd frontend && npx playwright test e2e/admin.spec.ts
```

Expected if the dev/e2e environment is available: PASS.

If Playwright browsers or the dev server are not available, run the static frontend test suite instead and record the E2E environment blocker in the final report:

```bash
cd frontend && npm test -- --run src/components/AdminLayout.test.tsx
```

Expected: PASS.

- [ ] **Step 3: Update `.gitignore` Go binary ignores**

Replace lines 5-12 of `.gitignore` with:

```gitignore
# Go binaries
backend/agentstore
backend/agentstore-cli
backend/agentstore-server
backend/server
backend/vendor/
/agentstore
/bin/
```

The resulting top of `.gitignore` should be:

```gitignore
# Config files with secrets
backend/config/dev.yaml
backend/config/prod.yaml

# Go binaries
backend/agentstore
backend/agentstore-cli
backend/agentstore-server
backend/server
backend/vendor/
/agentstore
/bin/

# Process management
.pids/
```

- [ ] **Step 4: Update `.dockerignore` Go binary ignores**

Replace the obsolete LastSaaS lines in `.dockerignore` with:

```dockerignore
backend/agentstore
backend/agentstore-cli
backend/agentstore-server
```

The resulting top of `.dockerignore` should be:

```dockerignore
.env
.env.local
.git
.gitignore
.idea
.vscode
.pids
.DS_Store
backend/agentstore
backend/agentstore-cli
backend/agentstore-server
backend/server
backend/vendor
backend/config/dev.yaml
frontend/node_modules
frontend/dist
bin
*.swp
*.swo
*~
```

- [ ] **Step 5: Update `scripts/setup.sh` visible defaults**

In `scripts/setup.sh`, make these exact replacements:

```diff
-echo "=== LastSaaS Setup ==="
+echo "=== AgentStore Setup ==="
@@
-read -rp "Database Name [lastsaas-dev]: " DATABASE_NAME
-DATABASE_NAME="${DATABASE_NAME:-lastsaas-dev}"
+read -rp "Database Name [agentstore-dev]: " DATABASE_NAME
+DATABASE_NAME="${DATABASE_NAME:-agentstore-dev}"
@@
-read -rp "App Name [LastSaaS]: " APP_NAME
-APP_NAME="${APP_NAME:-LastSaaS}"
+read -rp "App Name [AgentStore]: " APP_NAME
+APP_NAME="${APP_NAME:-AgentStore}"
```

- [ ] **Step 6: Remove accidental generated artifacts**

Run from the repo root:

```bash
rm -f backend/handlers.test package.json package-lock.json backend/package.json backend/package-lock.json
```

Expected: command exits 0 and the files no longer appear in `git status --short` as untracked files.

- [ ] **Step 7: Verify remaining LastSaaS references are compatibility or historical docs only**

Run:

```bash
grep -RInE 'LastSaaS|lastsaas|/last' --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=dist --exclude-dir=vendor .
```

Expected remaining categories:

```text
frontend/src/App.tsx: compatibility redirect from /last/* to /admin/*
docs/superpowers/...: historical specs/plans mentioning the migration
```

Unexpected categories to fix before continuing:

```text
scripts/setup.sh
.gitignore
.dockerignore
frontend/e2e/admin.spec.ts
README.md
Dockerfile
fly.toml
.github/workflows/ci.yml
backend/cmd/agentstore/*.go user-facing usage text
frontend/src/pages/**/*.tsx user-facing navigation or copy
```

- [ ] **Step 8: Run repository status check**

Run:

```bash
git status --short
```

Expected: deleted generated artifacts are gone; intended modifications include `.gitignore`, `.dockerignore`, `scripts/setup.sh`, and `frontend/e2e/admin.spec.ts` if they were not already updated.

- [ ] **Step 9: User-approved commit checkpoint**

Only if the user explicitly asks for a commit, run:

```bash
git add .gitignore .dockerignore scripts/setup.sh frontend/e2e/admin.spec.ts backend/handlers.test package.json package-lock.json backend/package.json backend/package-lock.json
git commit -m "$(cat <<'EOF'
chore: converge AgentStore setup artifacts

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

Expected: commit succeeds. If hooks fail, fix the underlying issue and create a new commit after user approval.

---

## Task 2: Establish the build and unit-test baseline

**Files:**
- Read/verify: `backend/go.mod`
- Read/verify: `frontend/package.json`
- Fix only if needed: files named in failing compiler/test output

- [ ] **Step 1: Run backend package tests for P0 areas**

Run:

```bash
cd backend && go test ./internal/api/handlers ./internal/credits ./internal/llm ./internal/validation ./internal/config ./internal/middleware
```

Expected: PASS.

If this fails, fix the smallest package named in the failure. Use these target mappings:

```text
agent handler failure       -> backend/internal/api/handlers/agent.go or agent_test.go
chat handler failure        -> backend/internal/api/handlers/chat.go or chat_test.go
LLM/model failure           -> backend/internal/api/handlers/llm_config.go, model_settings.go, backend/internal/llm/router.go, or related tests
credit failure              -> backend/internal/credits/service.go or service_test.go
validation failure          -> backend/internal/validation/validate.go, backend/internal/db/schema.go, or backend/internal/models/*.go
middleware/auth failure     -> backend/internal/middleware/*.go or corresponding *_test.go
```

- [ ] **Step 2: Run backend build**

Run:

```bash
cd backend && go build ./...
```

Expected: PASS.

If this fails because of removed `backend/cmd/lastsaas` references, replace the reference with `backend/cmd/agentstore` in the failing file. Do not recreate `backend/cmd/lastsaas`.

- [ ] **Step 3: Run frontend focused tests**

Run:

```bash
cd frontend && npm test -- --run src/components/AdminLayout.test.tsx src/pages/admin/LLMConfigPage.test.tsx src/pages/app/DashboardPage.test.tsx src/pages/app/ChatPage.test.tsx src/pages/app/settings/AgentsTab.test.tsx src/pages/app/settings/ModelSettingsTab.test.tsx
```

Expected: PASS.

If this fails, fix only the component or API mock named in the failure. Use these target mappings:

```text
AdminLayout failure         -> frontend/src/components/Layout.tsx or AdminLayout.test.tsx
LLM config failure          -> frontend/src/pages/admin/LLMConfigPage.tsx or its test
Dashboard failure           -> frontend/src/pages/app/DashboardPage.tsx or its test
Chat failure                -> frontend/src/pages/app/ChatPage.tsx, frontend/src/api/client.ts, or ChatPage.test.tsx
Agent settings failure      -> frontend/src/pages/app/settings/AgentsTab.tsx or its test
Model settings failure      -> frontend/src/pages/app/settings/ModelSettingsTab.tsx or its test
```

- [ ] **Step 4: Run frontend typecheck/build**

Run:

```bash
cd frontend && npx tsc --noEmit && npm run build
```

Expected: PASS.

If Vite warns about Node version, switch to Node `^20.19.0 || >=22.12.0` before treating the warning as an app failure.

- [ ] **Step 5: Run the full backend build verification required by project instructions**

Run:

```bash
cd backend && go build ./...
```

Expected: PASS.

- [ ] **Step 6: User-approved commit checkpoint**

Only if the user explicitly asks for a commit and Tasks 1-2 pass, run:

```bash
git status --short
git add <only-files-intentionally-changed-for-tasks-1-and-2>
git commit -m "$(cat <<'EOF'
chore: establish AgentStore P0 verification baseline

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

Expected: commit succeeds. Replace `<only-files-intentionally-changed-for-tasks-1-and-2>` with explicit paths; do not use `git add .`.

---

## Task 3: Verify fresh setup, bootstrap, auth, and root admin gates

**Files:**
- Verify: `backend/cmd/agentstore/main.go`
- Verify: `backend/internal/api/handlers/bootstrap.go`
- Verify/fix if needed: `backend/internal/api/handlers/auth.go`
- Verify/fix if needed: `backend/internal/api/handlers/admin.go`
- Verify/fix if needed: `backend/internal/middleware/auth.go`
- Verify/fix if needed: `backend/internal/middleware/tenant.go`
- Verify/fix if needed: `backend/internal/middleware/rbac.go`
- Test: `backend/internal/api/handlers/auth_test.go`
- Test: `backend/internal/api/handlers/admin_test.go`
- Test: `backend/internal/api/handlers/isolation_test.go`
- Test: `frontend/e2e/auth.spec.ts`
- Test: `frontend/e2e/admin.spec.ts`

- [ ] **Step 1: Run auth/admin backend tests**

Run:

```bash
cd backend && go test ./internal/api/handlers -run 'Test(Auth|Admin|Bootstrap|Isolation)' -count=1
```

Expected: PASS.

- [ ] **Step 2: Verify CLI command names compile**

Run:

```bash
cd backend && go run ./cmd/agentstore help
```

Expected output includes:

```text
agentstore - AgentStore system administration tool
setup                Initialize the system
status               Check system and database status
doctor               Run comprehensive system diagnostics
```

- [ ] **Step 3: Verify no LastSaaS CLI command directory remains tracked**

Run:

```bash
git status --short backend/cmd/lastsaas backend/cmd/agentstore
```

Expected: `backend/cmd/lastsaas/*` deletions and `backend/cmd/agentstore/*` additions are intentional. There should be no new code that imports or invokes `cmd/lastsaas`.

- [ ] **Step 4: Run admin route E2E unauthenticated check if browser environment is available**

Run:

```bash
cd frontend && npx playwright test e2e/admin.spec.ts
```

Expected: PASS.

If the browser environment is not available, record the blocker and keep the updated `frontend/e2e/admin.spec.ts` as the executable check for CI.

- [ ] **Step 5: Fresh database manual verification command set**

Use a disposable MongoDB database name. Do not point this at production.

Run in terminal 1:

```bash
export DATABASE_NAME="agentstore-smoke-$(date +%Y%m%d%H%M%S)"
export MONGODB_URI="mongodb://localhost:27017"
export JWT_ACCESS_SECRET="$(openssl rand -hex 32)"
export JWT_REFRESH_SECRET="$(openssl rand -hex 32)"
export SERVER_HOST="localhost"
export SERVER_PORT="4290"
export FRONTEND_URL="http://localhost:4280"
cd backend && go run ./cmd/agentstore setup
```

Expected: setup creates a root tenant and owner account or prompts for the required owner fields. Use test-only credentials.

- [ ] **Step 6: Start backend against the disposable database**

Run:

```bash
cd backend && go run ./cmd/server
```

Expected: server listens on `localhost:4290` and logs no MongoDB connection error.

- [ ] **Step 7: Check bootstrap and health endpoints**

Run in another terminal:

```bash
curl -sS http://localhost:4290/health
curl -sS http://localhost:4290/api/bootstrap/status
```

Expected:

```text
/health returns HTTP 200
/api/bootstrap/status reports initialized=true after setup
```

- [ ] **Step 8: User-approved commit checkpoint**

Only if code changed and the user explicitly asks for a commit, run:

```bash
git status --short
git add <auth-bootstrap-admin-files-that-changed>
git commit -m "$(cat <<'EOF'
fix: verify AgentStore setup and admin gates

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

Expected: commit succeeds.

---

## Task 4: Verify curated agent management and dashboard visibility

**Files:**
- Verify/fix if needed: `backend/internal/api/handlers/agent.go`
- Verify/fix if needed: `backend/internal/models/agent.go`
- Verify/fix if needed: `backend/internal/db/schema.go`
- Verify/fix if needed: `backend/internal/validation/validate.go`
- Verify/fix if needed: `frontend/src/api/client.ts`
- Verify/fix if needed: `frontend/src/pages/app/DashboardPage.tsx`
- Verify/fix if needed: `frontend/src/pages/app/settings/AgentsTab.tsx`
- Test: `backend/internal/api/handlers/agent_test.go`
- Test: `frontend/src/pages/app/DashboardPage.test.tsx`
- Test: `frontend/src/pages/app/settings/AgentsTab.test.tsx`

- [ ] **Step 1: Run tenant agent backend tests**

Run:

```bash
cd backend && go test ./internal/api/handlers -run 'Test(ListAgents|CreateAgent|UpdateAgent|PublishAgent|ArchiveAgent|DeleteAgent)' -count=1
```

Expected: PASS.

If a validation/schema failure occurs after editing `backend/internal/models/agent.go`, update both:

```text
backend/internal/models/agent.go
backend/internal/db/schema.go
```

Then run:

```bash
cd backend && go test ./internal/validation/... ./internal/api/handlers -run 'Test.*Agent' -count=1
```

Expected: PASS.

- [ ] **Step 2: Run frontend agent UI tests**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/DashboardPage.test.tsx src/pages/app/settings/AgentsTab.test.tsx
```

Expected: PASS.

- [ ] **Step 3: Verify frontend APIs use current backend paths**

Check `frontend/src/api/client.ts` contains these calls:

```ts
export const agentsApi = {
  list: () =>
    api.get<Agent[]>('/chat/agents').then(r => r.data),
  get: (agentId: string) =>
    api.get<Agent>(`/chat/agents/${agentId}`).then(r => r.data),
};

export const tenantAgentsApi = {
  list: () => api.get<Agent[]>('/tenant/agents').then(r => r.data),
  get: (id: string) => api.get<Agent>(`/tenant/agents/${id}`).then(r => r.data),
  create: (data: Partial<Agent>) => api.post<Agent>('/tenant/agents', data).then(r => r.data),
  update: (id: string, data: Partial<Agent>) => api.put<Agent>(`/tenant/agents/${id}`, data).then(r => r.data),
  publish: (id: string) => api.post<{ status: string }>(`/tenant/agents/${id}/publish`).then(r => r.data),
  archive: (id: string) => api.post<{ status: string }>(`/tenant/agents/${id}/archive`).then(r => r.data),
  delete: (id: string) => api.delete(`/tenant/agents/${id}`).then(r => r.data),
};
```

Expected: public user discovery uses `/chat/agents` compatibility for P0, and tenant admin CRUD uses `/tenant/agents`.

- [ ] **Step 4: Manual P0 agent flow**

With backend and frontend running against a disposable database:

```text
1. Log in as root/admin.
2. Open /settings/agents.
3. Create an agent with:
   name: Smoke Legal Expert
   slug: smoke-legal-expert
   category: legal
   description: Answers basic legal-information questions for smoke testing.
   systemPrompt: Reply concisely and include the token smoke-agent-ok in the first sentence.
   welcomeMessage: Ask me a legal-information question.
   suggestedPrompts: ["Summarize a simple NDA", "Explain limitation of liability"]
   creditCost.textMessageCredits: 3
   status: draft
   visibility: private
4. Publish the agent.
5. Open /dashboard as a normal user in the same tenant.
6. Confirm Smoke Legal Expert appears.
7. Archive the agent.
8. Refresh /dashboard.
9. Confirm Smoke Legal Expert no longer appears.
```

Expected: draft/archived agents are hidden from normal users; published agents are visible.

- [ ] **Step 5: Confirm public agent responses do not expose system prompts**

From an authenticated browser session or API client, call:

```text
GET /api/chat/agents
GET /api/chat/agents/smoke-legal-expert
```

Expected response fields include public metadata and credit cost, and do not include:

```text
systemPrompt
modelConfig provider secrets
apiKey
apiKeyEncrypted
apiKeyRef
```

- [ ] **Step 6: User-approved commit checkpoint**

Only if code changed and the user explicitly asks for a commit, run:

```bash
git status --short
git add <agent-management-files-that-changed>
git commit -m "$(cat <<'EOF'
fix: validate curated agent management flow

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

Expected: commit succeeds.

---

## Task 5: Verify model configuration and root LLM fallback

**Files:**
- Verify/fix if needed: `backend/internal/api/handlers/model_settings.go`
- Verify/fix if needed: `backend/internal/api/handlers/llm_config.go`
- Verify/fix if needed: `backend/internal/llm/router.go`
- Verify/fix if needed: `backend/internal/llm/openai.go`
- Verify/fix if needed: `backend/internal/models/model_provider.go`
- Verify/fix if needed: `backend/internal/models/model_config.go`
- Verify/fix if needed: `frontend/src/pages/app/settings/ModelSettingsTab.tsx`
- Verify/fix if needed: `frontend/src/pages/admin/LLMConfigPage.tsx`
- Test: `backend/internal/api/handlers/model_settings_test.go`
- Test: `backend/internal/api/handlers/llm_config_test.go`
- Test: `backend/internal/llm/router_test.go`
- Test: `frontend/src/pages/app/settings/ModelSettingsTab.test.tsx`
- Test: `frontend/src/pages/admin/LLMConfigPage.test.tsx`

- [ ] **Step 1: Run backend LLM/model tests**

Run:

```bash
cd backend && go test ./internal/api/handlers ./internal/llm -run 'Test(.*LLM.*|.*Model.*|.*Provider.*|.*Router.*)' -count=1
```

Expected: PASS.

- [ ] **Step 2: Run frontend LLM/model settings tests**

Run:

```bash
cd frontend && npm test -- --run src/pages/admin/LLMConfigPage.test.tsx src/pages/app/settings/ModelSettingsTab.test.tsx
```

Expected: PASS.

- [ ] **Step 3: Verify provider secrets are masked in API responses**

Inspect or test responses from:

```text
GET /api/tenant/model-providers
GET /api/admin/llm-config
```

Expected:

```text
Full provider API keys are not returned.
Masked previews such as sk-***1234 are acceptable.
Empty/missing key previews are acceptable when no key is configured.
```

If plaintext keys are returned, fix the response shape in:

```text
backend/internal/api/handlers/model_settings.go
backend/internal/api/handlers/llm_config.go
```

Then add/adjust tests in:

```text
backend/internal/api/handlers/model_settings_test.go
backend/internal/api/handlers/llm_config_test.go
```

- [ ] **Step 4: Manual real-provider root fallback verification**

Use the cheapest available OpenAI-compatible text model. Do not commit keys.

```text
1. Log in as root/admin.
2. Open /admin/llm-config.
3. Set API key, base URL, model, and enable chat.
4. Save.
5. Refresh the page.
6. Confirm the config reloads without exposing the full API key.
```

Expected: config saves, reloads, and masks secrets.

- [ ] **Step 5: Manual tenant provider verification**

```text
1. Open /settings/models as a tenant admin.
2. Create an OpenAI-compatible provider.
3. Create a text model config for that provider.
4. Set it as the default text model.
5. Run the provider test action.
```

Expected: provider test reports success with valid credentials and a clear failure with invalid credentials.

- [ ] **Step 6: User-approved commit checkpoint**

Only if code changed and the user explicitly asks for a commit, run:

```bash
git status --short
git add <model-llm-files-that-changed>
git commit -m "$(cat <<'EOF'
fix: verify model configuration and LLM fallback

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

Expected: commit succeeds.

---

## Task 6: Verify streaming chat, persistence, and no-charge failures

**Files:**
- Verify/fix if needed: `backend/internal/api/handlers/chat.go`
- Verify/fix if needed: `backend/internal/credits/service.go`
- Verify/fix if needed: `backend/internal/models/chat_message.go`
- Verify/fix if needed: `backend/internal/models/conversation.go`
- Verify/fix if needed: `backend/internal/models/usage_event.go`
- Verify/fix if needed: `frontend/src/api/client.ts`
- Verify/fix if needed: `frontend/src/pages/app/ChatPage.tsx`
- Test: `backend/internal/api/handlers/chat_test.go`
- Test: `backend/internal/credits/service_test.go`
- Test: `frontend/src/pages/app/ChatPage.test.tsx`

- [ ] **Step 1: Run backend chat and credit tests**

Run:

```bash
cd backend && go test ./internal/api/handlers ./internal/credits -run 'Test(.*Chat.*|.*Stream.*|.*Message.*|.*Credit.*|.*Usage.*)' -count=1
```

Expected: PASS.

- [ ] **Step 2: Run frontend chat tests**

Run:

```bash
cd frontend && npm test -- --run src/pages/app/ChatPage.test.tsx
```

Expected: PASS.

- [ ] **Step 3: Verify frontend streaming parser event contract**

Check `frontend/src/api/client.ts` contains this event union:

```ts
export type ChatStreamEvent =
  | { event: 'message_start'; data: { conversationId: string; messageId: string } }
  | { event: 'delta'; data: { text: string } }
  | { event: 'message_done'; data: { conversationId: string; messageId: string; creditsCharged: number; remainingCredits: number; model: string } }
  | { event: 'error'; data: { message: string } };
```

Expected: backend `writeSSE` events in `backend/internal/api/handlers/chat.go` match these names and data shapes.

- [ ] **Step 4: Manual real streaming chat verification**

Using the disposable database and real low-cost LLM config:

```text
1. Give the test tenant exactly 6 total credits.
2. Open /chat/smoke-legal-expert.
3. Send: Reply with exactly: smoke-ok
4. Wait until streaming completes.
5. Refresh the page.
6. Confirm the conversation and assistant message remain visible.
```

Expected UI:

```text
Assistant response appears.
No generic AI unavailable message appears.
Visible credit balance decreases by the agent text credit cost.
Refresh keeps the conversation history.
```

- [ ] **Step 5: Verify MongoDB writes after successful chat**

Run in `mongosh` against the disposable database:

```javascript
const tenant = db.tenants.findOne({ slug: "YOUR_TENANT_SLUG" })
db.usage_events.find({ tenantId: tenant._id, type: "agent_chat" }).sort({ createdAt: -1 }).limit(3)
db.conversations.find({ tenantId: tenant._id }).sort({ updatedAt: -1 }).limit(3)
db.chat_messages.find({ tenantId: tenant._id }).sort({ createdAt: -1 }).limit(6)
db.tenants.findOne({ _id: tenant._id }, { subscriptionCredits: 1, purchasedCredits: 1 })
```

Expected:

```text
usage_events has quantity equal to the agent credit cost.
conversation exists for the selected agent.
chat_messages include one user message and one assistant message.
assistant status is completed.
assistant creditsCharged equals the agent credit cost.
tenant total credits decreased by exactly the agent credit cost.
```

- [ ] **Step 6: Verify provider failure does not charge**

```text
1. Change the configured provider key or base URL to an invalid test value.
2. Record the tenant credit balance.
3. Send a chat message.
4. Confirm the UI shows a clear error.
5. Check MongoDB tenant balance and usage_events.
```

Expected:

```text
No new chargeable usage_event is created.
Tenant credit balance is unchanged.
Assistant message status is error.
creditsCharged is 0.
```

- [ ] **Step 7: Verify insufficient balance blocks before provider call**

```text
1. Set tenant credits to 0.
2. Restore valid provider config.
3. Send a chat message.
4. Check UI and MongoDB.
```

Expected:

```text
Request fails with insufficient-credit behavior.
Provider is not called.
No chargeable usage_event is created.
Tenant credit balance remains 0.
```

- [ ] **Step 8: User-approved commit checkpoint**

Only if code changed and the user explicitly asks for a commit, run:

```bash
git status --short
git add <chat-credit-files-that-changed>
git commit -m "$(cat <<'EOF'
fix: verify streaming chat and credit charging

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

Expected: commit succeeds.

---

## Task 7: Verify recharge path, Stripe fallback decision, and admin credit operations

**Files:**
- Verify/fix if needed: `backend/internal/api/handlers/billing.go`
- Verify/fix if needed: `backend/internal/api/handlers/webhook.go`
- Verify/fix if needed: `backend/internal/stripe/stripe.go`
- Verify/fix if needed: `backend/internal/credits/service.go`
- Verify/fix if needed: `backend/internal/models/credit_bundle.go`
- Verify/fix if needed: `backend/internal/models/billing.go`
- Verify/fix if needed: `frontend/src/pages/app/BuyCreditsPage.tsx`
- Verify/fix if needed: `frontend/src/pages/app/settings/BillingTab.tsx`
- Verify/fix if needed: `frontend/src/pages/admin/FinancialPage.tsx`
- Test: `backend/internal/api/handlers/billing_test.go`
- Test: `backend/internal/stripe/stripe_test.go`
- Test: `backend/internal/credits/service_test.go`

- [ ] **Step 1: Run billing, Stripe, and credit tests**

Run:

```bash
cd backend && go test ./internal/api/handlers ./internal/stripe ./internal/credits -run 'Test(.*Billing.*|.*Checkout.*|.*Webhook.*|.*Stripe.*|.*Credit.*)' -count=1
```

Expected: PASS.

- [ ] **Step 2: Verify public credit bundle API path exists in route wiring**

Check `backend/cmd/server/main.go` contains:

```go
guarded.Handle("/credit-bundles", authMiddleware.RequireAuth(http.HandlerFunc(bundlesHandler.ListBundlesPublic))).Methods("GET")
```

Expected: authenticated users can list credit bundles without admin access.

- [ ] **Step 3: Verify checkout API path exists in route wiring**

Check `backend/cmd/server/main.go` contains:

```go
billingOwner.HandleFunc("/checkout", billingHandler.Checkout).Methods("POST")
```

Expected: tenant owners can start checkout.

- [ ] **Step 4: Choose P0 recharge mode based on test evidence**

Use this decision rule:

```text
If checkout creation, webhook signature verification, webhook idempotency, and credit fulfillment tests pass, P0 recharge mode is Stripe credit packs.
If any of those fail and cannot be fixed in one focused task, P0 recharge mode is manual payment plus admin credit adjustment.
```

Expected: exactly one P0 recharge mode is documented in the final execution report.

- [ ] **Step 5: Manual admin credit adjustment verification**

```text
1. Open the admin tenant profile page.
2. Select the disposable tenant.
3. Set subscriptionCredits to 6 and purchasedCredits to 0, or use the existing credit adjustment UI if available.
4. Save.
5. Reload the tenant profile.
6. Confirm the displayed balance is 6.
7. Send a chat message costing 3 credits.
8. Confirm balance becomes 3.
```

Expected: admin can restore a user/tenant to a usable credit balance without direct database edits.

- [ ] **Step 6: User-approved commit checkpoint**

Only if code changed and the user explicitly asks for a commit, run:

```bash
git status --short
git add <billing-credit-files-that-changed>
git commit -m "$(cat <<'EOF'
fix: verify P0 credit recharge path

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

Expected: commit succeeds.

---

## Task 8: Final deployment and smoke-test readiness

**Files:**
- Verify/fix if needed: `README.md`
- Verify/fix if needed: `.env.example`
- Verify/fix if needed: `Dockerfile`
- Verify/fix if needed: `fly.toml`
- Verify/fix if needed: `.github/workflows/ci.yml`
- Verify/fix if needed: `docs/smoke-test-real-mongo-llm-billing.md`
- Verify/fix if needed: `CLAUDE.md` if project instructions become stale

- [ ] **Step 1: Verify docs name the correct P0 deploy and smoke path**

Read these files:

```text
README.md
.env.example
Dockerfile
fly.toml
.github/workflows/ci.yml
docs/smoke-test-real-mongo-llm-billing.md
```

Expected:

```text
Visible project name is AgentStore.
CLI setup examples use go run ./cmd/agentstore setup.
Backend examples use go run ./cmd/server.
Frontend examples use cd frontend && npm run dev.
Smoke test uses real MongoDB and real low-cost LLM provider.
Secrets are described as environment values, not committed values.
```

- [ ] **Step 2: Run final backend verification**

Run:

```bash
cd backend && go test ./internal/api/handlers ./internal/credits ./internal/llm ./internal/validation ./internal/config ./internal/middleware && go build ./...
```

Expected: PASS.

- [ ] **Step 3: Run final frontend verification**

Run:

```bash
cd frontend && npm test -- --run src/components/AdminLayout.test.tsx src/pages/admin/LLMConfigPage.test.tsx src/pages/app/DashboardPage.test.tsx src/pages/app/ChatPage.test.tsx src/pages/app/settings/AgentsTab.test.tsx src/pages/app/settings/ModelSettingsTab.test.tsx && npx tsc --noEmit && npm run build
```

Expected: PASS.

- [ ] **Step 4: Run final naming scan**

Run:

```bash
grep -RInE 'LastSaaS|lastsaas|/last' --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=dist --exclude-dir=vendor .
```

Expected remaining categories only:

```text
frontend/src/App.tsx compatibility redirect from /last/* to /admin/*
docs/superpowers historical design/plan references
```

- [ ] **Step 5: Run smoke test checklist**

Follow `docs/smoke-test-real-mongo-llm-billing.md` against a disposable database and a low-cost real LLM provider.

Expected smoke-test evidence:

```text
/health returns 200.
Root/admin login works.
Admin LLM config saves and reloads.
A published agent can be chatted with.
Successful chat deducts credits and writes usage_events.
Provider failure does not deduct credits.
Insufficient balance blocks before provider call.
Admin recharge restores usage.
```

- [ ] **Step 6: Prepare final execution report**

Write the final report in the conversation, not a new doc file unless the user asks. Include:

```text
Build status: pass/fail with commands run.
Test status: pass/fail with commands run.
Manual smoke status: pass/fail/not run with reason.
P0 recharge mode: Stripe credit packs or manual payment plus admin recharge.
Remaining blockers: exact file/route/test and why it blocks P0.
Safe next phase: P1 stable operations or remaining P0 blocker fix.
```

- [ ] **Step 7: User-approved commit checkpoint**

Only if code changed and the user explicitly asks for a commit, run:

```bash
git status --short
git add <explicit-final-doc-or-config-files-that-changed>
git commit -m "$(cat <<'EOF'
chore: document AgentStore P0 smoke readiness

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

Expected: commit succeeds.

---

## Self-Review Against Spec

### Spec coverage

- P0 code/naming convergence: Task 1 and Task 2.
- Initialization/auth/admin: Task 3.
- Curated agent store: Task 4.
- Model configuration/routing: Task 5.
- Streaming chat/persistence: Task 6.
- Credits and billing fallback: Task 7.
- Deployment and smoke testing: Task 8.
- Operational admin controls: Task 7 manual admin recharge plus Task 8 smoke/final report. Deeper P1 admin enhancements remain outside this P0 plan.

### Placeholder scan

This plan avoids unresolved placeholder markers and unspecified future implementation placeholders. Where runtime failures may occur, the plan maps each failure to exact files and commands.

### Type/path consistency

- Public agent discovery uses current backend path `/api/chat/agents` through `agentsApi` for P0 compatibility.
- Tenant agent CRUD uses `/api/tenant/agents` through `tenantAgentsApi`.
- Streaming chat uses `/api/chat/stream` and event names `message_start`, `delta`, `message_done`, `error`.
- Admin LLM config uses `/api/admin/llm-config`.
- Tenant model settings use `/api/tenant/model-providers`, `/api/tenant/model-configs`, and `/api/tenant/model-defaults`.
