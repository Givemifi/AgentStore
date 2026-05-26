# AgentStore Mainline Redesign Design

## Purpose

AgentStore is being reshaped from the LastSaaS boilerplate into an open-source platform for launching, selling, and monetizing AI agents. This implementation pass should create the core product loop: tenant-scoped agent management, AgentStore branding, `/admin` root administration, model configuration groundwork, and modern streaming chat.

The implementation should preserve the existing architecture and avoid unnecessary dependencies. Existing auth, tenant isolation, billing, credits, Stripe, admin, telemetry, and bootstrap behavior must keep working.

## Current State

The backend is Go with Gorilla Mux and MongoDB. It uses hybrid validation: Go `validate` tags plus MongoDB JSON Schema in `backend/internal/db/schema.go`. The frontend is React 19, Vite, TypeScript, React Router, TanStack Query, Tailwind, Vitest, and Playwright.

Current chat is based on a static catalog in `backend/internal/agents/catalog.go`, a singleton LLM config in `llm_configs`, and a non-streaming `POST /api/chat` request. Existing conversation and chat message collections are tenant/user scoped. Credits are checked before the LLM call and deducted after success.

The frontend currently exposes root admin under `/last`, stores several localStorage keys with the `lastsaas_` prefix, and contains visible LastSaaS branding in defaults, bootstrap instructions, admin pages, and tests.

## Scope for This Pass

This pass implements the mainline product experience:

- Rebrand visible product surfaces from LastSaaS to AgentStore.
- Migrate user-facing root admin routing from `/last` to `/admin`.
- Add tenant-scoped agent persistence and management.
- Add tenant-scoped model provider/model configuration groundwork.
- Add a model router for text chat with fallback to the existing legacy LLM config.
- Add SSE-style streaming chat while keeping the non-streaming endpoint.
- Improve dashboard, chat, and agent management mobile usability.
- Add validation, schema, indexes, tests, README, and `.env.example` updates.

Image and video generation are intentionally not fully implemented in this pass. The data model and placeholder API surface should reserve those capabilities and return clear “coming soon” responses without charging credits.

## Non-Goals

This pass will not implement public community marketplace review, per-agent payment routing, plugin systems, multi-agent orchestration, full image/video provider execution, or a full Go module path rename from `lastsaas` to `agentstore`.

The Go module path may remain `lastsaas` for now to avoid broad mechanical import churn. User-facing branding and routes should move to AgentStore.

## Routing Design

User-facing routes should be:

- `/dashboard` for the published agent discovery experience.
- `/chat/:agentSlug` for chat.
- `/chat` redirects to `/dashboard`.
- `/settings/agents` for tenant agent management.
- `/settings/models` for tenant model settings.
- `/admin/*` for root system administration.

The frontend should stop linking to `/last`. Existing `/last/*` routes may be temporarily redirected to `/admin/*` for compatibility, but `/last` should not remain the advertised or navigated surface.

Admin route references in components, tests, links, and branding logic must be updated from `/last` to `/admin`.

## Data Model Design

### Agents

Add a tenant-scoped `agents` collection with this Go model shape:

- `id`
- `tenantId`
- `name`
- `slug`
- `category`
- `description`
- `avatar`
- `icon`
- `color`
- `status`: `draft`, `published`, `archived`
- `visibility`: `private`, `public`
- `systemPrompt`
- `welcomeMessage`
- `suggestedPrompts[]`
- `capabilities[]`: `text_chat`, `image_generation`, `video_generation`
- `creditCost.textMessageCredits`
- `creditCost.imageGenerationCredits`
- `creditCost.videoGenerationCredits`
- `modelConfig.textModelId`
- `modelConfig.imageModelId`
- `modelConfig.videoModelId`
- `createdBy`
- `createdAt`
- `updatedAt`

Agents must be isolated by `tenantId`. Normal users may only see `published` agents. Tenant admins and owners may create, edit, publish, unpublish, and archive agents. Deletion should be implemented as archive, not hard delete.

Indexes:

- Unique `tenantId + slug` for active agent identity.
- `tenantId + status` for dashboard/admin listing.
- `tenantId + updatedAt` for management views.

### Model Providers

Add `model_providers`:

- `id`
- `tenantId`
- `name`
- `providerType`: `openai_compatible`, `anthropic`, `gemini`
- `baseUrl`
- `apiKeyEncrypted` or `apiKeyRef`
- `enabled`
- `createdAt`
- `updatedAt`

In this pass, if the repository does not already have a general encryption facility suitable for provider keys, store the key server-side in the provider record but never return it to the frontend. The API response must only expose a masked preview such as `sk-***1234`. A later pass can add envelope encryption if needed.

Indexes:

- Unique `tenantId + name`.
- `tenantId + enabled`.

### Model Configs

Add `model_configs`:

- `id`
- `tenantId`
- `providerId`
- `name`
- `displayName`
- `modality`: `text`, `image`, `video`
- `modelId`
- `defaultParams`
- `enabled`
- `createdAt`
- `updatedAt`

Indexes:

- `tenantId + providerId + modality`.
- `tenantId + modality + enabled`.

### Tenant Default Models

Add optional tenant-level default model references:

- `defaultTextModelConfigId`
- `defaultImageModelConfigId`
- `defaultVideoModelConfigId`

The model router uses these for fallback.

### Chat Messages

Extend `chat_messages` with:

- `status`: `generating`, `completed`, `error`, `interrupted`

Existing messages without a status should remain readable. New assistant messages created during streaming should begin as `generating`, then move to `completed`, `error`, or `interrupted`.

## API Design

### Public Agent APIs

Add:

- `GET /api/agents`
- `GET /api/agents/{slug}`

These require auth and tenant context, return only published agents for the current tenant, and must not expose `systemPrompt` or any model/provider secret.

The existing `/api/chat/agents` and `/api/chat/agents/{agentId}` can remain temporarily as compatibility wrappers, but frontend code should use `/api/agents`.

### Tenant Agent Admin APIs

Add under tenant scope:

- `GET /api/tenant/agents`
- `POST /api/tenant/agents`
- `GET /api/tenant/agents/{agentId}`
- `PUT /api/tenant/agents/{agentId}`
- `POST /api/tenant/agents/{agentId}/publish`
- `POST /api/tenant/agents/{agentId}/archive`

All write routes require tenant role `admin` or `owner`. Reads in this management namespace should also require at least `admin`, because drafts and full prompts are visible here.

### Tenant Model APIs

Add:

- `GET /api/tenant/model-providers`
- `POST /api/tenant/model-providers`
- `PUT /api/tenant/model-providers/{providerId}`
- `POST /api/tenant/model-providers/{providerId}/test`
- `GET /api/tenant/model-configs`
- `POST /api/tenant/model-configs`
- `PUT /api/tenant/model-configs/{modelId}`
- `POST /api/tenant/model-defaults`

All routes require tenant role `admin` or `owner`. Provider API keys are write-only from the client perspective. API responses must return masked previews only.

### Streaming Chat API

Keep existing `POST /api/chat` as the non-streaming endpoint.

Add:

- `POST /api/chat/stream`

Use `text/event-stream` over POST. The request body should reuse the existing chat request shape: `agentId` or `agentSlug`, optional `conversationId`, and `message`.

Expected event types:

- `message_start`
- `delta`
- `message_done`
- `error`

Server flow:

1. Authenticate user and load tenant.
2. Resolve the published agent in the current tenant.
3. Check active billing where existing chat currently requires it.
4. Resolve a text model via the model router.
5. Check sufficient credits before provider call.
6. Create or validate the conversation.
7. Save the user message.
8. Save assistant placeholder with `status=generating`.
9. Stream provider deltas to the client.
10. On successful completion, update assistant content/status, deduct credits, and emit usage metadata.
11. On provider failure, update assistant status to `error` and do not deduct credits.
12. On client disconnect, update assistant status to `interrupted` and do not deduct credits.

## Model Router Design

Add a backend model router/service with methods:

- `ResolveTextModel(ctx, tenant, agent)`
- `ResolveImageModel(ctx, tenant, agent)`
- `ResolveVideoModel(ctx, tenant, agent)`

Resolution order:

1. Agent-specific model config for the modality.
2. Tenant default model config for the modality.
3. Existing legacy active `llm_configs` singleton for text only.
4. Clear configuration error.

The router should return a per-request immutable provider/model snapshot so a config edit during a request does not change an in-flight generation.

## Frontend Design

### Dashboard

The dashboard remains the main agent discovery page. It should be rebranded from “AI Experts” to AgentStore language and use published agents from `/api/agents`.

Agent cards should show:

- icon/avatar/color
- category
- description
- capability badges
- credits per text message
- suggested prompts preview
- Start Chat button

Mobile should use single-column cards and avoid overflowing badges/prompts.

### Chat Page

The chat page should support streaming with smooth delta rendering. Use `fetch` with `AbortController`, not EventSource, because the request is POST with JSON body and auth headers.

Behavior:

- Streaming assistant message appears immediately and updates as deltas arrive.
- Input is disabled while generating.
- “Stop generating” aborts the request.
- Failed generation keeps useful context and allows retry.
- Composer is fixed near the bottom with mobile safe-area padding.
- Conversation drawer is collapsed by default on small screens.
- Suggested prompts scroll horizontally on mobile.
- `/chat` redirects to `/dashboard`.

### Agent Management

Add tenant-scoped management under `/settings/agents`.

The editor should feel like building a GPT/Gemini Gem without overbuilding:

- Basic info: name, slug, category, description, icon/avatar/color.
- Prompt: system prompt and welcome message.
- Suggested prompts list.
- Capabilities checkboxes.
- Credits per capability.
- Model bindings for text/image/video.
- Status controls: draft, publish, archive.

The form should validate required fields, show loading states while saving, and remain usable on mobile.

### Model Settings

Add tenant-scoped model settings under `/settings/models`.

The page should support:

- Provider list and provider form.
- Model config list and model config form.
- Default model selection per modality.
- Masked API key display.
- Test connection button for provider/model configuration.

Image/video models can be configured now, but generation routes should return a coming-soon response until implemented.

## Rebranding Design

Visible `LastSaaS` references should be replaced with `AgentStore`. This includes:

- Frontend default branding.
- Dashboard and chat copy.
- Admin About page.
- Bootstrap/setup guidance.
- Admin routes and tests.
- Config defaults where safe.
- README and `.env.example`.

Local storage key migration should be compatible: read old `lastsaas_*` keys where necessary, write new `agentstore_*` keys going forward, and clear both on logout where appropriate.

Environment config should support new `AGENTSTORE_*` aliases where practical while preserving existing `LASTSAAS_*` compatibility for current deployments.

## Security Design

- Tenant isolation must be enforced on every agent, model provider, model config, conversation, and message query.
- Public agent APIs must never return `systemPrompt`.
- Provider API keys must never be returned unmasked.
- React-rendered user content should remain escaped by default. Existing custom branding HTML must remain sanitized.
- Streaming errors should not expose provider secrets or raw provider responses.
- Credits must only be deducted after successful model completion.
- Interrupted or failed streaming generations must not deduct credits.

## Migration and Seeding

The implementation should add schemas and indexes through the existing MongoDB schema/index mechanisms.

The static catalog should be migrated into seed data or a fallback bootstrap path for new tenants if needed. The goal is to remove hard dependency on `backend/internal/agents/catalog.go` while preserving a useful first-run experience.

## Tests and Verification

Backend tests should cover:

- Agent validation and Mongo schema compatibility.
- Tenant-scoped agent list/read/write.
- Admin/owner write permission and user denial.
- Public agent API redacts `systemPrompt`.
- Provider API key masking.
- Model router fallback order.
- Streaming provider failure does not deduct credits.
- Streaming client interruption does not deduct credits.
- Successful streaming deducts credits after completion.

Frontend tests should cover:

- `/chat` redirects to `/dashboard`.
- `/admin` route replaces `/last` in admin layout and tests.
- Agent editor required-field validation.
- Model settings API key masking behavior.
- Chat streaming appends deltas and supports abort/retry state.

Required verification commands:

```bash
cd backend && go test ./internal/validation/...
cd backend && go build ./...
cd frontend && npx tsc --noEmit
```

Additional focused Go/Vitest tests should run for changed handlers and pages.

## Implementation Slices

The work can be parallelized safely after this spec is approved:

1. Rebranding and route migration: update `/last` to `/admin`, visible AgentStore copy, tests, README, env examples.
2. Backend data/API: add Agent and model provider/config models, schema, indexes, tenant APIs, public APIs.
3. Model router and streaming: add router fallback, streaming LLM support, chat message status, credits behavior.
4. Frontend product UI: dashboard, chat streaming, settings agents, settings models, mobile layout.

These slices touch overlapping types and API client files, so final integration should reconcile shared frontend types and backend route registration carefully.

## Open Decisions Resolved

- Admin route migration: migrate visible root admin routes from `/last` to `/admin`.
- Agent management scope: tenant-scoped, not root-admin scoped.
- `/chat` without an agent: redirect to `/dashboard`.
- First implementation phase: mainline product loop first; full image/video generation deferred.
