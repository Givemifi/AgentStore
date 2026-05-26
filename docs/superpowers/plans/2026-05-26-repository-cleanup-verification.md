# Repository Cleanup and Verification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the current large, mixed working tree into a clean, verified, commit-ready set of intentional changes.

**Architecture:** Treat this as a release-readiness audit rather than feature work: first inventory every changed/untracked path, then quarantine obvious generated artifacts, then run backend and frontend verification, and finally produce an explicit staging list. No destructive cleanup happens until the user confirms the file classification.

**Tech Stack:** Git, Go backend (`go build`, `go test`), Vite/React frontend (`vitest`, `tsc`, `npm run build`), repository instructions in `CLAUDE.md`.

---

## Current observed state

Tracked modifications exist across backend validation/schema/server files and frontend app/API/context/page files. Untracked paths include likely real feature files such as `backend/internal/api/handlers/chat.go`, `backend/internal/models/agent.go`, `frontend/src/pages/app/ChatPage.tsx`, and `frontend/src/pages/app/settings/ModelSettingsTab.tsx`, plus likely generated or temporary paths such as `backend/node_modules/`, root `node_modules/`, `backend/main`, root `package.json`, root `package-lock.json`, `backend/package.json`, `backend/package-lock.json`, and fingerprint/debug scripts.

Targeted verification already observed before this plan:

```bash
npm --prefix frontend test -- --run ChatPage ModelSettingsTab
# Expected/observed: 2 files passed, 14 tests passed

cd frontend && npx tsc --noEmit
# Expected/observed: exit 0, no output
```

Fresh verification must still be run during plan execution before claiming the whole project is ready.

## File structure and responsibilities for this cleanup

- `backend/` — Go backend source. Real feature files here should be validated with `go build ./...` and relevant `go test` commands.
- `frontend/` — React/Vite frontend source. Real feature files here should be validated with `npm test`, `npx tsc --noEmit`, and `npm run build` as needed.
- `docs/superpowers/plans/2026-05-26-repository-cleanup-verification.md` — this execution plan.
- `.gitignore` — may need updates for generated artifacts that are currently untracked, such as root `node_modules/`, `backend/node_modules/`, and `backend/main`, after confirming they are not intentional source.
- Root `package.json` / `package-lock.json` and `backend/package.json` / `backend/package-lock.json` — currently untracked; classify before deleting or staging.
- Root debug scripts (`capture_fingerprint.go`, `capture_fingerprint.py`, `show_fingerprint.py`, `tls_fingerprint.py`, `init-db.js`) — currently untracked; classify with the user before deleting or staging.

---

### Task 1: Inventory the working tree

**Files:**
- Inspect only: repository root git status and file list
- No code changes

- [ ] **Step 1: Capture concise status**

Run:

```bash
git -C /Users/duoxianganwenaiyisheng/Desktop/AgnetStore status --short
```

Expected: output lists all tracked modifications and untracked files. Save the output in the conversation or a scratch note, not in a committed file.

- [ ] **Step 2: Capture tracked diff summary**

Run:

```bash
git -C /Users/duoxianganwenaiyisheng/Desktop/AgnetStore diff --stat
```

Expected: output summarizes tracked changes. Use it to identify large-risk areas, especially `frontend/package-lock.json`, `backend/internal/db/schema.go`, and `frontend/src/pages/app/DashboardPage.tsx`.

- [ ] **Step 3: List untracked files without scanning dependency folders**

Run:

```bash
git -C /Users/duoxianganwenaiyisheng/Desktop/AgnetStore ls-files --others --exclude-standard | grep -v '/node_modules/'
```

Expected: output excludes dependency folder contents and lists top-level untracked source/artifact paths.

- [ ] **Step 4: Classify untracked paths into three buckets**

Use this initial classification:

```text
Likely intentional source/test/docs:
- backend/internal/agents/
- backend/internal/api/handlers/agent.go
- backend/internal/api/handlers/agent_test.go
- backend/internal/api/handlers/chat.go
- backend/internal/api/handlers/chat_test.go
- backend/internal/api/handlers/llm_config.go
- backend/internal/api/handlers/llm_config_test.go
- backend/internal/api/handlers/model_settings.go
- backend/internal/api/handlers/model_settings_test.go
- backend/internal/credits/
- backend/internal/db/schema_test.go
- backend/internal/llm/
- backend/internal/models/agent.go
- backend/internal/models/chat_message.go
- backend/internal/models/conversation.go
- backend/internal/models/llm_config.go
- backend/internal/models/model_config.go
- backend/internal/models/model_provider.go
- docs/
- frontend/src/components/AdminLayout.test.tsx
- frontend/src/pages/admin/LLMConfigPage.test.tsx
- frontend/src/pages/admin/LLMConfigPage.tsx
- frontend/src/pages/app/ChatPage.test.tsx
- frontend/src/pages/app/ChatPage.tsx
- frontend/src/pages/app/DashboardPage.test.tsx
- frontend/src/pages/app/settings/AgentsTab.test.tsx
- frontend/src/pages/app/settings/AgentsTab.tsx
- frontend/src/pages/app/settings/ModelSettingsTab.test.tsx
- frontend/src/pages/app/settings/ModelSettingsTab.tsx
- frontend/src/utils/storageKeys.ts

Likely generated or accidental:
- backend/main
- backend/node_modules/
- node_modules/
- root package.json
- root package-lock.json
- backend/package.json
- backend/package-lock.json

Needs user decision:
- backend/handlers.test
- capture_fingerprint.go
- capture_fingerprint.py
- init-db.js
- show_fingerprint.py
- tls_fingerprint.py
```

Expected: the executor asks the user to confirm the classification before deleting or staging anything.

- [ ] **Step 5: Commit checkpoint is not allowed yet**

Do not commit. This task is inventory only.

---

### Task 2: Confirm cleanup decisions with the user

**Files:**
- Potentially modify later: `.gitignore`
- Potentially remove later only after explicit approval: generated artifacts and debug scripts

- [ ] **Step 1: Ask for classification approval**

Ask the user this exact decision prompt:

```text
I found three buckets:
1. Likely intentional source/test/docs: backend agent/chat/model settings/credits/llm files, frontend ChatPage/Dashboard/settings/admin tests and pages, docs.
2. Likely generated or accidental: backend/main, backend/node_modules/, root node_modules/, root package.json/package-lock.json, backend package.json/package-lock.json.
3. Needs your decision: backend/handlers.test, capture_fingerprint.go, capture_fingerprint.py, init-db.js, show_fingerprint.py, tls_fingerprint.py.

Should I remove the generated/accidental bucket, keep the intentional bucket, and leave the decision bucket untouched for now?
```

Expected: user says yes or provides corrections. If corrections are provided, update the classification in the conversation before any file operations.

- [ ] **Step 2: If approved, remove generated dependency/build artifacts only**

Run only if the user explicitly approves removal:

```bash
rm -rf /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/node_modules /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/node_modules /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/main
```

Expected: command exits 0. This is destructive, so it requires explicit user approval in the same conversation.

- [ ] **Step 3: If approved, remove accidental package files only**

Run only if the user explicitly confirms root/backend package files are accidental:

```bash
rm -f /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/package.json /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/package-lock.json /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/package.json /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/package-lock.json
```

Expected: command exits 0. Do not run this if the user wants to keep any Node tooling outside `frontend/`.

- [ ] **Step 4: If approved, update `.gitignore` for recurring generated artifacts**

Modify `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore/.gitignore` by adding these lines if they are not already covered:

```gitignore
# Local dependency installs outside frontend
/node_modules/
/backend/node_modules/

# Local Go build output
/backend/main
```

Expected: `git status --short .gitignore` shows `.gitignore` modified only if new ignore rules were needed.

- [ ] **Step 5: Re-run status after cleanup**

Run:

```bash
git -C /Users/duoxianganwenaiyisheng/Desktop/AgnetStore status --short
```

Expected: generated artifacts are gone from untracked output; intentional source/test/docs remain.

- [ ] **Step 6: Commit checkpoint is not allowed yet**

Do not commit. Verification comes first.

---

### Task 3: Run backend verification

**Files:**
- Verify: `backend/...`
- No code changes unless a command fails and root cause is identified

- [ ] **Step 1: Run required backend build**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend && go build ./...
```

Expected: exit 0, no build errors.

- [ ] **Step 2: Run validation tests required by project instructions**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend && go test ./internal/validation/...
```

Expected: exit 0 with `ok lastsaas/internal/validation` or equivalent package output.

- [ ] **Step 3: Run backend unit tests that do not require external services**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend && go test ./internal/models ./internal/db ./internal/llm ./internal/credits
```

Expected: exit 0 for packages that have tests; packages with no tests may print `[no test files]`.

- [ ] **Step 4: Run backend handler tests in short mode first**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend && go test -short ./internal/api/handlers
```

Expected: exit 0. Integration tests guarded by `testing.Short()` should be skipped.

- [ ] **Step 5: If backend verification fails, stop and debug systematically**

If any backend command fails, do not patch randomly. Record:

```text
Command:
Exit code:
First error line:
Relevant file/line:
Likely failing component:
```

Then use the systematic-debugging workflow before proposing code changes.

---

### Task 4: Run frontend verification

**Files:**
- Verify: `frontend/...`
- No code changes unless a command fails and root cause is identified

- [ ] **Step 1: Run the targeted regression tests from the recent fix**

Run:

```bash
npm --prefix /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend test -- --run ChatPage ModelSettingsTab
```

Expected: exit 0 with `2 passed` test files and `14 passed` tests.

- [ ] **Step 2: Run TypeScript check exactly from the frontend project**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend && npx tsc --noEmit
```

Expected: exit 0, no TypeScript errors.

- [ ] **Step 3: Run full frontend test suite**

Run:

```bash
npm --prefix /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend test
```

Expected: exit 0. If failures occur outside ChatPage/ModelSettingsTab, record failing test names before changing code.

- [ ] **Step 4: Run frontend production build**

Run:

```bash
npm --prefix /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend run build
```

Expected: exit 0. This validates `tsc -b` plus Vite build.

- [ ] **Step 5: If frontend verification fails, stop and debug systematically**

If any frontend command fails, record:

```text
Command:
Exit code:
Failing test or compiler diagnostic:
Relevant file/line:
Recent change most likely related:
```

Then use the systematic-debugging workflow before proposing code changes.

---

### Task 5: Review high-risk diffs before staging

**Files:**
- Inspect: all changed source/test files
- No code changes unless review finds a concrete bug

- [ ] **Step 1: Review backend schema/model validation consistency**

Run:

```bash
git -C /Users/duoxianganwenaiyisheng/Desktop/AgnetStore diff -- backend/internal/models backend/internal/db/schema.go backend/internal/validation/validate.go backend/internal/validation/validate_test.go
```

Expected: model `validate` tags and MongoDB JSON schema constraints are aligned for changed/new models, per `CLAUDE.md`.

- [ ] **Step 2: Review backend handler registration and routes**

Run:

```bash
git -C /Users/duoxianganwenaiyisheng/Desktop/AgnetStore diff -- backend/cmd/server/main.go backend/internal/api/handlers
```

Expected: new handlers are registered, routes are tenant-scoped where required, and test helpers match handler expectations.

- [ ] **Step 3: Review frontend API/types/page consistency**

Run:

```bash
git -C /Users/duoxianganwenaiyisheng/Desktop/AgnetStore diff -- frontend/src/api/client.ts frontend/src/types/index.ts frontend/src/pages/app frontend/src/pages/admin frontend/src/App.tsx
```

Expected: API client methods match types and pages; no references remain to stale `chatApi.send` behavior in ChatPage tests.

- [ ] **Step 4: Search for accidental raw secret display**

Run:

```bash
grep -R "apiKey\|secret\|rawKey" -n /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend/src /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend/internal | grep -v node_modules
```

Expected: frontend displays only preview/masked secret fields except explicit one-time reveal flows such as API key creation. Investigate any raw provider API key display.

- [ ] **Step 5: Search for unresolved debug markers**

Run:

```bash
grep -R "TODO\|FIXME\|console.log\|panic(" -n /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend/src | grep -v node_modules
```

Expected: no new unresolved debug markers in changed/new code. Existing unrelated markers can be noted but should not be fixed as part of this cleanup.

---

### Task 6: Prepare commit-ready staging list

**Files:**
- Inspect/stage only after verification passes
- No commit unless the user explicitly asks for one

- [ ] **Step 1: Produce the final status**

Run:

```bash
git -C /Users/duoxianganwenaiyisheng/Desktop/AgnetStore status --short
```

Expected: output includes only intentional source/test/docs and any approved `.gitignore` change.

- [ ] **Step 2: Produce a recommended staging list**

Build the list explicitly; do not use `git add .`. Start with this candidate list and adjust after cleanup decisions:

```text
backend/cmd/server/main.go
backend/internal/agents/
backend/internal/api/handlers/agent.go
backend/internal/api/handlers/agent_test.go
backend/internal/api/handlers/chat.go
backend/internal/api/handlers/chat_test.go
backend/internal/api/handlers/llm_config.go
backend/internal/api/handlers/llm_config_test.go
backend/internal/api/handlers/model_settings.go
backend/internal/api/handlers/model_settings_test.go
backend/internal/api/handlers/testhelpers_test.go
backend/internal/credits/
backend/internal/db/mongodb.go
backend/internal/db/schema.go
backend/internal/db/schema_test.go
backend/internal/llm/
backend/internal/models/agent.go
backend/internal/models/chat_message.go
backend/internal/models/conversation.go
backend/internal/models/llm_config.go
backend/internal/models/model_config.go
backend/internal/models/model_provider.go
backend/internal/models/tenant.go
backend/internal/planstore/seed.go
backend/internal/testutil/testutil.go
backend/internal/validation/validate.go
backend/internal/validation/validate_test.go
frontend/package.json
frontend/package-lock.json
frontend/src/App.tsx
frontend/src/api/client.ts
frontend/src/components/AdminLayout.tsx
frontend/src/components/AdminLayout.test.tsx
frontend/src/components/BrandingThemeInjector.tsx
frontend/src/components/ImpersonationBanner.tsx
frontend/src/components/Layout.tsx
frontend/src/contexts/AuthContext.tsx
frontend/src/contexts/BrandingContext.tsx
frontend/src/contexts/TenantContext.tsx
frontend/src/contexts/ThemeContext.tsx
frontend/src/hooks/useTelemetry.ts
frontend/src/pages/admin/DashboardPage.tsx
frontend/src/pages/admin/LLMConfigPage.tsx
frontend/src/pages/admin/LLMConfigPage.test.tsx
frontend/src/pages/admin/LogsPage.tsx
frontend/src/pages/admin/TenantProfilePage.tsx
frontend/src/pages/admin/TenantsPage.tsx
frontend/src/pages/admin/UserProfilePage.tsx
frontend/src/pages/admin/UsersPage.tsx
frontend/src/pages/app/ChatPage.tsx
frontend/src/pages/app/ChatPage.test.tsx
frontend/src/pages/app/DashboardPage.tsx
frontend/src/pages/app/DashboardPage.test.tsx
frontend/src/pages/app/SettingsPage.tsx
frontend/src/pages/app/settings/AgentsTab.tsx
frontend/src/pages/app/settings/AgentsTab.test.tsx
frontend/src/pages/app/settings/ModelSettingsTab.tsx
frontend/src/pages/app/settings/ModelSettingsTab.test.tsx
frontend/src/pages/auth/LoginPage.tsx
frontend/src/test/setup.ts
frontend/src/types/index.ts
frontend/src/utils/errors.ts
frontend/src/utils/storageKeys.ts
docs/
```

Expected: generated artifacts and undecided debug scripts are not included unless the user explicitly says they are intentional.

- [ ] **Step 3: If the user asks to stage, stage by explicit path**

Run only after user asks to stage or commit:

```bash
git -C /Users/duoxianganwenaiyisheng/Desktop/AgnetStore add <explicit approved paths only>
```

Expected: no accidental `node_modules`, binaries, local package files, or debug scripts are staged.

- [ ] **Step 4: Verify staged diff before any commit**

Run:

```bash
git -C /Users/duoxianganwenaiyisheng/Desktop/AgnetStore diff --cached --stat
```

Expected: staged files match the approved staging list.

- [ ] **Step 5: Do not commit unless explicitly requested**

Stop after presenting:

```text
Verification results:
- Backend build:
- Backend validation tests:
- Backend unit tests:
- Frontend targeted tests:
- Frontend full tests:
- Frontend build:

Cleanup results:
- Removed generated artifacts:
- Left for user decision:

Recommended staging list:
- ...
```

Expected: user decides whether to commit, split commits, or do more cleanup.

---

## Self-review

- Spec coverage: This plan covers inventory, cleanup approval, backend verification, frontend verification, high-risk diff review, and staging preparation.
- Placeholder scan: No TBD/TODO placeholders are present; each step includes exact commands or exact user prompts.
- Type/path consistency: Paths use `/Users/duoxianganwenaiyisheng/Desktop/AgnetStore` consistently and commands avoid broad staging or destructive cleanup without approval.
