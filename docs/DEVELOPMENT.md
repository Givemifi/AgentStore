# AgentStore Development

This document summarizes the current development workflow for AgentStore. Keep this file focused on how to build, test, and safely change the project; product positioning belongs in `README.md`, and production smoke testing belongs in `LAUNCH_SMOKE_TEST.md`.

## Prerequisites

- Go 1.25+
- Node.js 22+
- MongoDB Atlas or local MongoDB
- Git
- Optional integrations for full local testing: Stripe test-mode keys, Resend API key, OAuth provider credentials, and an OpenAI-compatible LLM provider key

## Local setup

From the repository root:

```bash
./scripts/setup.sh
```

The script creates local configuration and prompts for MongoDB, JWT secrets, app name, frontend URL, and optional integration settings.

To start the backend:

```bash
set -a && source .env && set +a
cd backend
go run ./cmd/server
```

To start the frontend in another terminal:

```bash
set -a && source .env && set +a
cd frontend
npm install
npm run dev
```

To initialize a new database:

```bash
cd backend
go run ./cmd/agentstore setup
```

Default local URLs:

- Backend: `http://localhost:4290`
- Frontend: `http://localhost:4280`

## Required verification after changes

Run these before considering code changes ready:

```bash
cd backend && go build ./... && go vet ./... && go test ./...
cd frontend && npx tsc --noEmit
cd frontend && npm run lint
cd frontend && npm test -- --run
```

For launch-critical changes involving MongoDB, billing, LLM providers, credits, tenant isolation, or chat, also run the manual smoke test in `LAUNCH_SMOKE_TEST.md` against disposable credentials and test-mode billing.

## Backend validation rule

When modifying structs in `backend/internal/models/`:

1. Update Go `validate` struct tags.
2. Update the matching MongoDB JSON Schema in `backend/internal/db/schema.go`.
3. Keep both sets of constraints equivalent.
4. Run:

```bash
cd backend && go test ./internal/validation/...
```

When adding a new collection that accepts user/API writes:

1. Add validation tags to the model struct.
2. Add a schema function to `backend/internal/db/schema.go`.
3. Include it in `AllSchemas()`.
4. Add validation tests in `backend/internal/validation/validate_test.go`.

## Logging rule

Use `syslog.Logger` for significant system events. Supported severities are:

- `critical`
- `high`
- `medium`
- `low`
- `debug`

Avoid adding ad-hoc debug output to request handlers or frontend UI paths. CLI user-facing output in `backend/cmd/agentstore` is expected.

## Cleanup rules

When cleaning the repository:

- Audit first, then delete.
- Delete only what is proven unused.
- Do not remove deployment, Docker, env example, or CI files unless they are proven obsolete.
- Do not remove test fixtures or helpers unless they are proven unreferenced.
- Do not change business logic during cleanup.
- Do not run `git add` or `git commit` unless explicitly asked.

Protected high-risk paths:

- `backend/internal/credits/`
- `backend/internal/stripe/`
- `backend/internal/llm/`
- `backend/internal/api/handlers/chat.go`
- `backend/internal/api/handlers/billing.go`
- `backend/internal/middleware/tenant.go`
- `backend/internal/api/handlers/admin_launch_readiness.go`
- `frontend/src/pages/app/ChatPage.tsx`
- `frontend/src/api/client.ts`
- `.env.example`
- `Dockerfile`
- `.github/workflows/`

## Git workflow

Recommended pre-commit review commands:

```bash
git status --short
git diff --stat
git diff --check
git diff --name-status
```

Only stage files after reviewing the diff. Do not commit generated dependencies, local logs, local AI tool state, secrets, build artifacts, or test output directories.

Local AI tool state is ignored through `.gitignore`:

```gitignore
.claude/worktrees/
.superpowers/
```
