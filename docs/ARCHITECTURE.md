# AgentStore Architecture

AgentStore is a Go + React application for operating a curated AI agent marketplace on top of a production SaaS foundation. Users discover agents, chat with them, understand credit usage, buy credits, and continue work inside a tenant-scoped workspace. Operators manage agents, plans, credits, billing, branding, health, telemetry, logs, and launch readiness from the admin interface.

## Runtime shape

- **Backend:** Go HTTP server in `backend/cmd/server` using `gorilla/mux`.
- **CLI / MCP:** `backend/cmd/agentstore` provides administration commands and the read-only MCP server.
- **Frontend:** React 19 + TypeScript + Vite in `frontend/src`.
- **Database:** MongoDB stores tenants, users, memberships, plans, credits, chat conversations/messages, telemetry, logs, webhooks, configuration, and health metrics.
- **Billing:** Stripe handles checkout, subscriptions, billing portal, tax, refunds, disputes, and promotion codes.
- **LLM providers:** OpenAI-compatible provider configuration is routed through backend LLM services and used by agent chat.

## Backend modules

- `backend/internal/api/handlers/` — HTTP handlers for auth, bootstrap, admin, tenant, billing, chat, agents, usage, branding, webhooks, telemetry, health, logs, docs, promotions, and model settings.
- `backend/internal/auth/` — JWT access/refresh tokens, password hashing, OAuth, magic links, MFA/TOTP, sessions, and related identity logic.
- `backend/internal/middleware/` — authentication, tenant resolution, RBAC, billing enforcement, metrics, recovery, API versioning, and security middleware.
- `backend/internal/models/` — MongoDB model structs. Model validation must stay in sync with MongoDB JSON Schema.
- `backend/internal/db/` — MongoDB connection, collections, indexes, and JSON Schema definitions.
- `backend/internal/credits/` — subscription and purchased credit accounting.
- `backend/internal/stripe/` — Stripe customers, checkout, prices, subscriptions, portal, tax, refunds, disputes, and webhook support.
- `backend/internal/llm/` — provider clients and request routing for agent chat.
- `backend/internal/agents/` — marketplace agent domain logic.
- `backend/internal/configstore/` — DB-backed runtime configuration.
- `backend/internal/planstore/` — seed data for plans and credit bundles.
- `backend/internal/events/` and `backend/internal/webhooks/` — internal event emission and outgoing webhook delivery.
- `backend/internal/telemetry/` — product analytics events, SDK helpers, and PM dashboard queries.
- `backend/internal/health/`, `backend/internal/metrics/`, `backend/internal/datadog/` — system health, metrics collection, and integration reporting.
- `backend/internal/syslog/` — system logging with severity levels and injection detection.
- `backend/internal/validation/` — Go-side validation helpers.
- `backend/internal/testutil/` — test support utilities.

## Frontend modules

- `frontend/src/App.tsx` — top-level route wiring and bootstrap guard.
- `frontend/src/api/client.ts` — API client, auth token refresh, and typed endpoint groups.
- `frontend/src/contexts/` — auth, tenant, branding, and theme context providers.
- `frontend/src/components/` — layout, admin layout, shared UI, skeletons, error boundaries, and theme injection.
- `frontend/src/components/app/` — app-facing reusable components such as credit explanations, markdown messages, empty states, error states, and mobile navigation.
- `frontend/src/pages/public/` — landing and custom public pages.
- `frontend/src/pages/auth/` — login, signup, MFA, magic link, password reset, and verification flows.
- `frontend/src/pages/app/` — user workspace pages, including dashboard, chat, onboarding, credits, billing, team, activity, plan, and settings.
- `frontend/src/pages/admin/` — platform admin pages for dashboard, users, tenants, plans, billing, branding, health, logs, API keys, webhooks, promotions, telemetry, config, and launch readiness.
- `frontend/src/types/` — shared TypeScript types.
- `frontend/src/utils/` and `frontend/src/hooks/` — shared utilities and hooks.
- `frontend/e2e/` — Playwright end-to-end tests.

## High-risk paths

Treat these as protected modules during cleanup and refactors:

- Billing and money movement: `backend/internal/api/handlers/billing.go`, `backend/internal/stripe/`, Stripe webhook handlers, checkout success flows, refunds, disputes, and transaction records.
- Credits: `backend/internal/credits/`, credit bundle checkout, credit deduction, usage events, and insufficient-credit behavior.
- Tenant isolation: `backend/internal/middleware/tenant.go`, tenant-scoped handlers, API key scope resolution, model settings, chats, agents, credits, and billing state.
- LLM provider routing: `backend/internal/llm/`, model/provider settings, provider fallback behavior, streaming and non-streaming chat paths.
- Chat: `backend/internal/api/handlers/chat.go`, `frontend/src/pages/app/ChatPage.tsx`, markdown rendering, streaming state, retry/interruption state, and credit charging.
- Launch readiness: `backend/internal/api/handlers/admin_launch_readiness.go` and the admin launch checklist UI.

## Validation and schema rule

AgentStore uses hybrid validation:

1. Go-side validation through `validate` tags on structs in `backend/internal/models/`.
2. MongoDB JSON Schema in `backend/internal/db/schema.go`.

When modifying model structs, keep both systems aligned and run:

```bash
cd backend && go test ./internal/validation/...
```

## Deployment shape

The default production build uses `Dockerfile` to build the Go backend and React frontend into a single container. The Go server serves the SPA. Environment variables configure MongoDB, JWT secrets, frontend URL, app name, OAuth, email, Stripe, and related integrations.

For projects built on top of AgentStore, follow the repository deployment documentation and project instructions before deploying. Do not replace deployment configs during cleanup unless they are proven obsolete.
