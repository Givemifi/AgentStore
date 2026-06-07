# AgentStore P0 Productization Closeout Design

**Date:** 2026-06-07  
**Status:** Approved design, awaiting implementation plan  
**Scope:** Fast P0 productization closeout for ordinary-user launch readiness.

## 1. Goal

Bring the current AgentStore branch from a mostly complete Agent marketplace experience to a tighter P0 launch baseline for ordinary users and platform operators.

This design intentionally targets the smallest productization slice that removes launch-blocking confusion and user-flow friction:

1. Mobile chat history and retry behavior.
2. Backend provider-support honesty during real chat execution.
3. A real Admin Launch Checklist backed by readiness data instead of local manual state.
4. README positioning that matches the product direction.
5. Focused tests and verification for the changed areas.

The goal is not to expand the marketplace feature set. The goal is to make the already-built marketplace/chat/credits/admin experience more trustworthy, understandable, and launchable.

## 2. Current Context

The existing UX design in `docs/superpowers/specs/2026-06-06-agentstore-user-ready-ux-design.md` defines AgentStore as two layers:

1. Primary ordinary-user product: a practical AI Agent marketplace.
2. Secondary operator/developer layer: SaaS foundation, admin tooling, billing, MCP, CLI, and deployment.

Static inspection shows many P0 UX items are already implemented:

- `/dashboard`, `/chat/:agentId`, `/buy-credits`, `/billing/success`, `/onboarding`, and `/admin` are routed in `frontend/src/App.tsx`.
- `/dashboard` already presents marketplace copy, search, category chips, credit explanation, and Agent cards.
- `/chat/:agentId` already supports streaming, Markdown rendering, suggested prompts, credit copy, insufficient-credit CTA, and stop generation.
- `/buy-credits` and `/billing/success` already explain credits and continuation paths.
- `/onboarding` already follows a goal → Agent → prompt → credits flow.
- A mobile bottom nav exists.
- An Admin Launch Checklist UI exists, but some statuses are local/manual rather than real readiness signals.

The remaining P0 gaps are concentrated and can be closed without a rewrite.

## 3. Non-Goals

This closeout will not implement:

- Popular Agents, Recently Used Agents, recommendation ranking, or a full marketplace merchandising system.
- Agent detail pages.
- Creator marketplace, ratings/reviews, revenue sharing, or public Agent pages.
- True Anthropic/Gemini execution.
- Image/video generation execution.
- A broad shared-component refactor.
- Full Playwright hardening for every user journey.
- Automatic real-LLM test chat from the admin checklist.

These are P1/P2 follow-ups after the fast P0 closeout is stable.

## 4. Design Overview

The closeout is four implementation packages plus verification:

1. **Chat mobile and retry closeout** — make chat usable on phone-sized screens and make interrupted/failed output recoverable.
2. **Provider execution guard** — prevent non-OpenAI-compatible providers from being used in real chat execution until implemented.
3. **Launch readiness API** — replace local/manual Launch Checklist status with backend-derived readiness items.
4. **README positioning** — make the public repo introduction match AgentStore's marketplace + foundation identity.

Each package should be implemented with focused tests in the nearest existing test files.

## 5. Package 1 — Chat Mobile and Retry Closeout

### 5.1 Desired User Experience

On desktop, chat keeps the current two-column structure: conversation history on the left, active chat on the right.

On mobile, the conversation list should no longer occupy permanent vertical space above the chat. The mobile experience should default to the active conversation and expose history through a drawer.

Expected mobile behavior:

- The chat header includes a visible conversation-history button.
- Tapping the button opens a left-side drawer containing:
  - New chat action.
  - Conversation list.
  - Current Agent summary.
- Tapping outside the drawer or pressing close hides it.
- Opening a conversation closes the drawer.
- Starting a new chat closes the drawer.
- The composer remains visible and usable at the bottom of the chat area.

### 5.2 Interrupted and Failed Messages

Interrupted output should not appear as plain text with a trailing `[interrupted]` marker. It should render as a normal assistant message with an explicit status affordance.

Recommended display:

- Badge: `Interrupted · no charge`
- Partial content remains visible.
- Retry action is shown in the assistant message footer.

Failed assistant responses should similarly show:

- A clear failure badge or inline error state.
- No credits charged.
- Retry action.

### 5.3 Retry Behavior

P0 retry should avoid a full message-editing system. The retry action should use the previous user message for the same failed/interrupted assistant response.

Acceptable P0 behavior:

- Direct retry: immediately resends the previous user message.
- If direct retry cannot safely resolve the previous user message, restore that message into the composer and focus the composer.

The retry should preserve the current conversation where possible. If the failed/interrupted response happened in a new optimistic conversation that does not yet have a server conversation ID, the retry may start a new conversation.

### 5.4 Data Shape

Prefer using existing `ChatMessage.status` if already available in frontend types. If the frontend type does not include status, add it with known backend statuses:

- `generating`
- `completed`
- `error`
- `interrupted`

Messages with `status === 'interrupted'` or `status === 'error'` and `role === 'assistant'` should trigger the status badge and retry UI.

For client-side interruption during streaming, the temporary message placed in the cache should include `status: 'interrupted'` and `creditsCharged: 0`, not a textual suffix.

### 5.5 Files Likely Touched

- `frontend/src/pages/app/ChatPage.tsx`
- `frontend/src/pages/app/ChatPage.test.tsx`
- `frontend/src/types/index.ts` if `ChatMessage.status` is missing or incomplete

## 6. Package 2 — Provider Execution Guard

### 6.1 Problem

The UI now largely hides unsupported provider types for new provider creation and marks existing Anthropic/Gemini providers as coming soon. The backend `TestProvider` endpoint also returns `unsupported` for non-OpenAI-compatible providers.

However, the execution path must also enforce this. If historical data or API-created data binds an Anthropic/Gemini provider to a text model, real chat should fail with a deliberate, understandable unsupported-provider error rather than accidentally trying to use it as an OpenAI-compatible endpoint or silently falling back.

### 6.2 Backend Behavior

Add an explicit provider type check in the model resolution path after loading a `ModelProvider` and before returning `llm.RequestConfig`.

Rules:

- `openai_compatible` is supported for text chat execution.
- `anthropic` is not supported for execution in P0.
- `gemini` is not supported for execution in P0.

When a non-supported provider is explicitly bound through the Agent model config or tenant default model config:

- Return a typed or sentinel unsupported-provider error.
- Do not silently fall back to legacy LLM config.
- Chat handler should return a friendly error such as:
  - `This model provider is not supported for chat yet. Use an OpenAI-compatible provider.`
- No credits should be deducted.

Legacy LLM config fallback remains valid when no explicit Agent or tenant model config can be resolved.

### 6.3 Image and Video

Image and video remain coming soon. Existing endpoints should continue returning a not-implemented response that clearly says no credits were charged.

### 6.4 Files Likely Touched

- `backend/internal/llm/router.go`
- `backend/internal/llm/router_test.go`
- `backend/internal/api/handlers/chat.go`
- `backend/internal/api/handlers/chat_test.go`
- Possibly `backend/internal/apierror` or local error helpers if a shared typed error improves clarity

## 7. Package 3 — Admin Launch Checklist Readiness API

### 7.1 Problem

The Admin Launch Checklist exists visually, but several statuses are derived from `localStorage` manual state. A launch checklist should answer what remains before launch based on real system state.

### 7.2 Backend Endpoint

Add a root/admin-only readiness endpoint:

```http
GET /api/admin/launch-readiness
```

Response shape:

```json
{
  "items": [
    {
      "id": "brand",
      "label": "Brand configured",
      "status": "complete",
      "description": "Review app name, logo, landing copy, dashboard copy, and auth page text before inviting users.",
      "actionPath": "/admin/branding"
    }
  ],
  "summary": {
    "complete": 4,
    "warning": 2,
    "pending": 1,
    "total": 7
  }
}
```

Allowed statuses:

- `complete`
- `warning`
- `pending`

### 7.3 Readiness Rules

P0 rules must be deterministic and cheap to compute.

#### Brand configured

Status: `complete` if branding appears meaningfully customized from defaults. Acceptable signals include non-default app name, logo URL, dashboard HTML, landing copy, or auth copy.

Status: `pending` otherwise.

#### Model provider connected

Status: `complete` if either:

- There is an enabled OpenAI-compatible provider and enabled text model for the current workspace/root workspace context; or
- Legacy LLM config is active and complete.

Status: `warning` otherwise.

#### First Agent published

Status: `complete` if at least one Agent in the relevant workspace is:

- `published`
- includes `text_chat` capability

Status: `pending` otherwise.

#### Credit bundle or plan active

Status: `complete` if there is at least one active plan or active credit bundle available for purchase/use.

Status: `warning` otherwise.

#### Stripe webhook healthy

Status uses existing integration health data:

- `complete` when Stripe is healthy.
- `warning` when Stripe is missing, not configured, degraded, or unhealthy.

#### Email provider ready

Status uses existing Resend/email integration health data:

- `complete` when healthy.
- `warning` otherwise.

This item is not a chat blocker but affects invites, verification, and password reset reliability.

#### Test chat passed

Status: `complete` if there is evidence of a successful real chat, such as a completed assistant message with credits charged or a successful `agent_chat` usage event.

Status: `pending` otherwise.

Do not automatically perform a real LLM call from the checklist.

### 7.4 Frontend Behavior

- `frontend/src/pages/admin/DashboardPage.tsx` should fetch readiness via `adminApi.getLaunchReadiness()` or equivalent.
- Remove `getManualItemStatus()` and localStorage-derived checklist status.
- `LaunchChecklist` should render backend items.
- If the readiness endpoint fails, show a visible warning/error state rather than silently showing stale manual state.
- Existing metric cards and integration warning banner can remain.

### 7.5 Files Likely Touched

- `backend/internal/api/handlers/admin.go` or a new focused handler file if existing patterns support it
- `backend/cmd/server/main.go` route wiring
- `backend/internal/api/handlers/admin_test.go` or a new readiness handler test
- `frontend/src/api/client.ts`
- `frontend/src/types/index.ts`
- `frontend/src/pages/admin/DashboardPage.tsx`
- `frontend/src/pages/admin/DashboardPage.test.tsx`
- `frontend/src/pages/admin/components/LaunchChecklist.tsx` if prop shape needs adjustment
- `frontend/src/pages/admin/components/LaunchChecklist.test.tsx`

## 8. Package 4 — README Positioning

### 8.1 Problem

The repository README still opens with a pure SaaS-boilerplate message. That conflicts with the current product direction and ordinary-user Agent marketplace experience.

### 8.2 Desired Positioning

The README should lead with:

> AgentStore is a marketplace for practical AI agents, built on an open-source SaaS foundation.

The opening section should explain both layers in this order:

1. Agent marketplace product: discover Agents, chat, understand credits, buy credits, and operate a curated Agent marketplace.
2. SaaS foundation: auth, tenants/workspaces, RBAC, billing, branding, health, logs, telemetry, webhooks, API keys, and admin tooling.

The phrase “boilerplate” can remain in developer/foundation sections, but should not be the first product impression.

### 8.3 Files Likely Touched

- `README.md`

## 9. Testing and Verification Strategy

This closeout uses targeted verification. It does not require full E2E hardening before implementation is considered ready for review.

### 9.1 Backend Tests

Required targeted tests:

- `backend/internal/llm/router_test.go`
  - OpenAI-compatible provider resolves successfully.
  - Anthropic/Gemini provider returns unsupported-provider error.
  - Explicit unsupported provider binding does not fall back to legacy LLM config.

- `backend/internal/api/handlers/chat_test.go`
  - Unsupported provider returns friendly chat error.
  - Credits are not deducted when provider is unsupported.

- Admin readiness handler tests:
  - Empty/unconfigured state returns pending/warning items.
  - Published text-chat Agent + valid model config + healthy integrations return expected complete statuses.

Required backend verification commands after implementation:

```bash
cd backend && go test ./internal/llm/...
cd backend && go test ./internal/api/handlers/...
cd backend && go build ./...
```

If model or schema validation changes are made, also run:

```bash
cd backend && go test ./internal/validation/...
```

### 9.2 Frontend Tests

Required targeted tests:

- `frontend/src/pages/app/ChatPage.test.tsx`
  - Mobile conversation drawer can be opened.
  - Interrupted assistant message shows badge and retry action.
  - Retry uses the previous user message or restores it to the composer.

- Admin dashboard/checklist tests:
  - Dashboard renders readiness API checklist items.
  - Readiness API failure is visible to the operator.
  - LocalStorage manual status is no longer the source of truth.

Required frontend verification commands after implementation:

```bash
cd frontend && npx tsc --noEmit
cd frontend && npm test -- --run <changed-test-files>
```

### 9.3 Playwright

Playwright coverage for marketplace/mobile/buy-success remains valuable, but it is not required for this fast P0 closeout. It should be handled in the next hardening pass unless implementation changes make an existing Playwright test fail.

## 10. Rollout and Risk Management

### 10.1 Migration Risk

There is no database migration required for the planned P0 behavior if existing message status fields and model provider fields are already present. If frontend types lack `ChatMessage.status`, only TypeScript types need to be expanded.

### 10.2 Provider Risk

The provider guard may make previously misconfigured tenants fail earlier and more explicitly. This is desired behavior. The error must be clear enough for admins to fix configuration by choosing an OpenAI-compatible provider.

### 10.3 Checklist Risk

Readiness rules are intentionally simple. They should be treated as launch guidance, not a formal compliance score. Avoid hiding the checklist if one signal cannot be computed; return warning/pending with helpful descriptions.

### 10.4 Credit Risk

Retry actions must not double-charge failed or interrupted responses. Credits should only be charged after a completed assistant response, consistent with existing chat behavior.

## 11. Success Criteria

This closeout succeeds when:

- On mobile, users can chat without the conversation list dominating the screen.
- Users can recover from failed or interrupted assistant responses with a visible retry path.
- Unsupported provider types cannot accidentally run through the OpenAI-compatible execution path.
- Admins see a Launch Checklist based on real readiness data.
- The README no longer presents AgentStore primarily as a generic SaaS boilerplate.
- Targeted backend and frontend tests cover the changed behavior.
- Build and compile verification are run and reported honestly after implementation.

## 12. Follow-Up Work After This Closeout

Recommended next steps after P0 closeout:

1. Marketplace sections: Recommended, Popular, Recently Used.
2. Agent detail page.
3. Billing success updated balance and receipt link.
4. Full Playwright coverage for marketplace, mobile nav, chat, insufficient credits, and buy credits success.
5. Broader component extraction: `AgentCard`, `PageHeader`, and shared admin/app empty states.
6. True Anthropic/Gemini execution only if provider-specific clients and tests are implemented.
