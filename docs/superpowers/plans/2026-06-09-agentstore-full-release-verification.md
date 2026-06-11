# AgentStore Full Release Verification Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run a long-form automated release verification pass over AgentStore, confirm failures through repeated checks, and save confirmed issues without fixing them.

**Architecture:** The verification is evidence-first and non-mutating: run static checks, builds, unit/integration tests, frontend tests, Docker/deployment checks, and app-level smoke checks where the local environment supports them. Every suspected issue must be re-run at least twice before it is promoted to the final issue report. The only planned file output is the verification issue report and command evidence; no source fixes are allowed during execution.

**Tech Stack:** Go 1.25, MongoDB-dependent Go tests, React 19, TypeScript, Vite, Vitest, ESLint, Playwright, Docker/Fly.io deployment artifacts, Bash.

---

## Scope and Rules

### Hard rules

- Do not fix source code during this verification pass.
- Do not commit, push, deploy, or publish anything.
- Do not mutate production or live third-party services.
- Use disposable/local state only.
- If a command requires unavailable infrastructure (Docker daemon, MongoDB, browser install, secrets), record the blocker as an environment limitation, not as an app defect.
- A suspected issue becomes a **confirmed issue** only after at least two consistent reproductions or one deterministic static contradiction (for example, config references a missing committed file).
- Save confirmed issues to `docs/release-verification-issues.md`.
- Include commands, exit codes, relevant output excerpts, severity, affected area, and recommended next validation.

### Time budget

The session may run continuously for up to 5 hours. Suggested allocation:

1. Baseline and repository hygiene: 20 minutes.
2. Backend build/vet/tests: 60 minutes.
3. Frontend lint/typecheck/build/tests: 60 minutes.
4. Docker/deployment/config verification: 45 minutes.
5. Playwright/E2E and local app smoke, if environment supports it: 90 minutes.
6. Reproduction passes and report writing: 45 minutes.

### Issue severity rubric

- **P0 Blocker:** prevents clean install, build, deploy, login, tenant bootstrap, or core chat/billing operation.
- **P1 High:** breaks an advertised feature, core admin path, or data/credit/billing correctness in common conditions.
- **P2 Medium:** feature bug with workaround, docs/config mismatch that can mislead deployers, flaky automated test.
- **P3 Low:** cleanup, warning, confusing wording, non-blocking local environment friction.

---

## File Structure Map

### Create/overwrite during execution

- `docs/release-verification-issues.md`
  - Final confirmed issue report.
  - Contains only repeated/confirmed problems and environment blockers.
  - Must not include unconfirmed suspicions.

### Read/verify during execution

- `.github/workflows/ci.yml`
  - Confirm CI reflects local release gates.
- `Dockerfile`
  - Confirm clean Docker build path.
- `fly.toml`
  - Confirm internal port matches deployment docs.
- `.env.example`
  - Confirm all required env vars are present.
- `backend/config/dev.example.yaml`
- `backend/config/prod.example.yaml`
  - Confirm runtime config templates match config struct and docs.
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/DEPLOYMENT.md`
- `docs/DEVELOPMENT.md`
- `LAUNCH_SMOKE_TEST.md`
  - Confirm release docs are internally consistent.
- `backend/`
  - Build, vet, tests.
- `frontend/`
  - Lint, typecheck, build, tests, Playwright where possible.

---

## Task 1: Establish Clean Baseline

**Files:**
- Read: repository root, `git status`, top-level docs and config files.
- Output: temporary command evidence in terminal only.

- [ ] **Step 1: Capture git status**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore
git status --short
git diff --name-status
```

Expected:

- Shows current working tree changes from release-prep work.
- No untracked secrets such as `.env`, `backend/config/prod.yaml`, or logs.

- [ ] **Step 2: Check ignored secret files are not tracked**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore
git ls-files .env backend/config/dev.yaml backend/config/prod.yaml frontend/dist 2>/dev/null
```

Expected:

- No output for local secret/config/build artifacts.

- [ ] **Step 3: Scan for obvious secret literals in tracked source**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore
grep -RInE 'sk_live_[A-Za-z0-9]|sk_test_[A-Za-z0-9]|whsec_[A-Za-z0-9]|AKIA[0-9A-Z]{16}|mongodb\+srv://[^[:space:]]+:[^[:space:]@]+@' \
  --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=dist . || true
```

Expected:

- Only placeholder examples, if any.
- Any real-looking credential must be rechecked by inspecting the file context before reporting.

---

## Task 2: Backend Verification Pass

**Files:**
- Read/execute: `backend/`.
- Output: command evidence only.

- [ ] **Step 1: Run backend build**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend
go build ./...
```

Expected: exit code 0.

- [ ] **Step 2: Run backend vet**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend
go vet ./...
```

Expected: exit code 0.

- [ ] **Step 3: Run backend short test suite**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend
go test -short ./...
```

Expected: exit code 0.

- [ ] **Step 4: Run targeted non-short packages once**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend
go test ./internal/config ./internal/validation ./internal/llm ./internal/api/handlers
```

Expected:

- exit code 0, or MongoDB-dependent tests skip only when test DB is unavailable.
- If a package fails, re-run that exact package twice before reporting.

- [ ] **Step 5: Re-run any failing backend command twice**

If any backend command fails, run the exact same command two more times:

```bash
# Replace COMMAND with the exact failing command.
COMMAND
COMMAND
```

Report only if at least two runs fail with the same failure signature.

---

## Task 3: Frontend Verification Pass

**Files:**
- Read/execute: `frontend/`.
- Output: command evidence only.

- [ ] **Step 1: Install dependency integrity check**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend
npm ci --dry-run
```

Expected: exit code 0. If npm version does not support expected behavior, record as environment limitation and continue.

- [ ] **Step 2: Run TypeScript typecheck**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend
npx tsc --noEmit
```

Expected: exit code 0.

- [ ] **Step 3: Run ESLint**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend
npm run lint
```

Expected:

- exit code 0.
- Warnings are acceptable if they are documented React Compiler advisory warnings.
- Any error must be re-run twice before reporting.

- [ ] **Step 4: Run production frontend build**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend
npm run build
```

Expected: exit code 0. Chunk-size warnings are non-blocking unless build exits non-zero.

- [ ] **Step 5: Run frontend tests**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend
npm test
```

Expected: exit code 0 and all tests pass.

- [ ] **Step 6: Re-run any failing frontend command twice**

If any frontend command fails, run the exact same command two more times and record whether the failure is deterministic, flaky, or environmental.

---

## Task 4: Deployment and Config Verification

**Files:**
- Read: `Dockerfile`, `fly.toml`, `.env.example`, `backend/config/*.example.yaml`, `README.md`, `docs/DEPLOYMENT.md`.
- Execute if available: Docker build.

- [ ] **Step 1: Validate setup script syntax**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore
bash -n scripts/setup.sh
```

Expected: exit code 0.

- [ ] **Step 2: Verify clean Docker build if Docker daemon is available**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore
docker build -t agentstore-release-verification .
```

Expected: exit code 0.

If Docker daemon is unavailable, record an environment blocker and do not report it as an app issue.

- [ ] **Step 3: Verify Dockerfile does not copy untracked private configs**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore
grep -n 'COPY backend/config/prod.yaml' Dockerfile || true
grep -n 'prod.example.yaml ./config/prod.yaml' Dockerfile
```

Expected:

- First grep has no output.
- Second grep finds the runtime config-template copy.

- [ ] **Step 4: Verify Fly internal port alignment**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore
grep -n 'internal_port = 8080' fly.toml
grep -n 'SERVER_PORT="8080"' README.md docs/DEPLOYMENT.md
```

Expected:

- `fly.toml` routes to 8080.
- README and deployment docs instruct `SERVER_PORT="8080"`.

- [ ] **Step 5: Verify env template coverage**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore
for key in SERVER_PORT DATABASE_NAME MONGODB_URI JWT_ACCESS_SECRET JWT_REFRESH_SECRET WEBHOOK_ENCRYPTION_KEY FRONTEND_URL APP_NAME FROM_EMAIL FROM_NAME STRIPE_PUBLISHABLE_KEY GITHUB_CLIENT_ID MICROSOFT_CLIENT_ID DATADOG_API_KEY; do
  grep -q "^${key}=" .env.example || echo "missing .env.example: $key"
  grep -q "${key}" README.md || echo "missing README: $key"
  grep -q "${key}" docs/DEPLOYMENT.md || echo "missing DEPLOYMENT: $key"
done
```

Expected: no `missing ...` lines.

---

## Task 5: Local App Smoke Verification

**Files:**
- Execute: backend server, frontend dev server or production Docker container if available.
- Output: environment-dependent evidence.

- [ ] **Step 1: Decide available runtime path**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore
command -v mongosh || command -v mongo || true
command -v docker || true
```

Expected:

- If MongoDB and Docker are unavailable, record environment limitation and skip runtime smoke.
- If MongoDB is available, continue.

- [ ] **Step 2: Start backend only if disposable MongoDB config exists**

Precondition: `.env` points to a disposable local/test database, not production.

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore
set -a && source .env && set +a
cd backend
go run ./cmd/server
```

Expected:

- Server starts without panic.
- `/health` returns HTTP 200.

Do not run if `.env` is missing or points to a production-looking database.

- [ ] **Step 3: Verify backend health**

Run in another shell if server is running:

```bash
curl -fsS http://localhost:4290/health
```

Expected: HTTP 200.

- [ ] **Step 4: Start frontend dev server if backend is running**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend
npm run dev -- --host 127.0.0.1
```

Expected: dev server starts and logs local URL.

- [ ] **Step 5: Browser smoke if Playwright browser tooling is available**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend
npx playwright test e2e/smoke.spec.ts --reporter=line
```

Expected: pass, or skip/report environment limitation if browsers are not installed or servers are unavailable.

---

## Task 6: Playwright / E2E Verification

**Files:**
- Execute: `frontend/e2e/*.spec.ts`.

- [ ] **Step 1: Check Playwright installation**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend
npx playwright --version
```

Expected: prints Playwright version.

- [ ] **Step 2: Run Playwright tests if environment is ready**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend
npx playwright test --reporter=line
```

Expected:

- pass if dev/test servers and browsers are configured.
- if it fails because servers are unavailable, browsers are missing, or MongoDB is unavailable, record as environment blocker.
- if it fails after servers are available, re-run the failing spec twice before reporting.

---

## Task 7: Reproduction and Issue Classification

**Files:**
- Create/overwrite: `docs/release-verification-issues.md`.

- [ ] **Step 1: Group all suspected issues**

For each suspected problem, capture:

```markdown
- Title:
- Area:
- First command:
- First evidence:
- Re-run #1 command and result:
- Re-run #2 command and result:
- Classification: confirmed / flaky / environment blocker / dismissed
```

- [ ] **Step 2: Promote only confirmed issues**

A problem is confirmed if:

- two or more runs fail with the same failure signature, or
- a static contradiction is deterministic and directly inspectable.

- [ ] **Step 3: Write the final report**

Create `docs/release-verification-issues.md` with this structure:

```markdown
# AgentStore Release Verification Issues

Date: 2026-06-09
Scope: automated release verification across backend, frontend, deployment config, docs, and app smoke where available.
Policy: confirmed issues only; no source fixes applied during verification.

## Summary

- Total confirmed issues: N
- P0 blockers: N
- P1 high: N
- P2 medium: N
- P3 low: N
- Environment blockers: N

## Confirmed Issues

### ISSUE-001: <title>

- Severity: P0/P1/P2/P3
- Area: backend/frontend/deployment/docs/tests
- Status: confirmed
- Reproduced: 2/2 or static deterministic
- Commands:
  ```bash
  <command>
  ```
- Evidence:
  ```text
  <short output excerpt>
  ```
- Impact:
- Suggested next validation:

## Environment Blockers

### ENV-001: <title>

- Area:
- Command:
- Evidence:
- Why this is not an app defect:

## Passing Checks

- <command> — PASS
```

- [ ] **Step 4: If no confirmed issues exist**

Still create the report with:

```markdown
## Confirmed Issues

No confirmed application defects were found during this verification pass.
```

---

## Task 8: Final Evidence Pass

**Files:**
- Read: `docs/release-verification-issues.md`.

- [ ] **Step 1: Run non-mutating final checks**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/backend
go build ./...
go vet ./...
go test -short ./...

cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore/frontend
npx tsc --noEmit
npm run lint
npm test
```

Expected:

- Report the result of each command in the final chat response.
- If a final command fails, perform two re-runs and update `docs/release-verification-issues.md`.

- [ ] **Step 2: Verify the issue report exists**

Run:

```bash
cd /Users/duoxianganwenaiyisheng/Desktop/AgnetStore
test -f docs/release-verification-issues.md
```

Expected: exit code 0.

---

## Self-Review Checklist

Spec coverage:

- 5-hour automated verification budget covered by time allocation and broad task list.
- Multiple rechecks covered by backend, frontend, E2E, and final reproduction tasks.
- No direct fixes rule stated in scope and every task is verification/reporting-only.
- Confirmed issues saved to `docs/release-verification-issues.md`.
- Environment blockers separated from app defects.

Placeholder scan:

- No TBD/TODO/fill-in placeholders.
- Every task has exact commands and expected result criteria.
- Report template is complete and explicit.

Type/path consistency:

- All paths are project-root-relative and match current repository layout.
- Report path is consistently `docs/release-verification-issues.md`.
- Plan path is `docs/superpowers/plans/2026-06-09-agentstore-full-release-verification.md`.
