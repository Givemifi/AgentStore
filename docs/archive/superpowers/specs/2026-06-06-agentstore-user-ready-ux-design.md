# AgentStore User-Ready UX Design

**Date:** 2026-06-06  
**Status:** Draft for user review  
**Primary goal:** Make AgentStore feel complete and immediately usable for ordinary end users while preserving its SaaS-boilerplate/developer value in a secondary layer.

## 1. Context

AgentStore currently has a strong multi-tenant SaaS foundation and a visible transition toward a platform-operated AI Agent marketplace.

Evidence from the current codebase:

- The app already exposes user-facing routes for the core journey: `/dashboard`, `/chat/:agentId`, `/buy-credits`, `/plan`, `/settings`, `/activity`, and `/onboarding` in `frontend/src/App.tsx`.
- The admin surface is broad: `/admin/users`, `/admin/tenants`, `/admin/plans`, `/admin/financial`, `/admin/health`, `/admin/branding`, `/admin/llm-config`, and more in `frontend/src/App.tsx`.
- The dashboard is already shaped as an Agent marketplace, with marketplace hero copy, credit balance, Agent cards, suggested prompts, and chat entry in `frontend/src/pages/app/DashboardPage.tsx`.
- The current P0 operational plan names the target as a “platform-operated agent marketplace” in `docs/superpowers/plans/2026-06-06-agentstore-p0-operational-readiness.md`.
- The public README still leads with “The last SaaS boilerplate you'll ever need,” which conflicts with an ordinary end-user Agent marketplace experience.

This design treats AgentStore as two layers:

1. **Primary end-user product:** a practical AI Agent marketplace where users can discover Agents, chat, understand credit usage, buy credits, and continue working.
2. **Secondary operator/developer layer:** the SaaS foundation, admin tooling, white-label system, billing, MCP, CLI, and deployment story.

## 2. Current development assessment

AgentStore is not an early demo. It already contains substantial backend and frontend functionality.

Approximate readiness by area:

| Area | Current readiness | Notes |
|---|---:|---|
| SaaS foundation | 80-90% | Auth, tenants, RBAC, billing, admin, logs, health, branding, telemetry are broad. |
| Admin backend/frontend | 75-85% | Very feature-rich, but onboarding and UI consistency need productization. |
| Agent management | 65-75% | CRUD, publish/archive, prompts, costs, models exist; creation UX is still utilitarian. |
| Agent marketplace | 55-65% | Users can browse cards and start chat, but search/filter/recommendation/trust cues are missing. |
| Chat | 65-75% | Streaming, history, credits, stop-generation exist; Markdown, retry, mobile polish need work. |
| Billing/credits UX | 55-65% | Backend is strong; user-facing explanation and purchase psychology need improvement. |
| Onboarding | 40-50% | Existing steps are account-oriented, not first-success-oriented. |
| Mobile user UX | 30-45% | Main navigation is hidden below `md` without an obvious replacement. |
| UI consistency | 45-60% | Strong dark aesthetic exists, but primitives are underused and states vary. |
| Product positioning/docs | 40-55% | SaaS-boilerplate and Agent-marketplace narratives need separation. |

## 3. Design principles

1. **Ordinary users should never need platform vocabulary.** Avoid “tenant,” “root,” “config,” “provider,” “boilerplate,” and “bucket” in the end-user path. Use “workspace,” “Agent,” “credits,” and “AI model settings” where needed.
2. **First success beats feature breadth.** The first session should guide the user to choose an Agent and send a useful first message.
3. **Credits must feel safe.** Users should know the balance, expected cost, when credits are charged, and what happens on failure.
4. **Do not overpromise capabilities.** If image/video generation or non-OpenAI-compatible providers are not truly implemented, mark them as coming soon or hide them from ordinary users.
5. **Admin should answer “am I ready to launch?”** Operators need a launch checklist more than more metrics on first use.
6. **Mobile is P0 for ordinary users.** A marketplace and chat product must be navigable on phone-sized screens.
7. **Improve through focused productization, not a rewrite.** Reuse current routes, APIs, and components where possible.

## 4. Target end-user journey

The ideal ordinary-user journey is:

1. User signs up or logs in.
2. User completes a short onboarding flow focused on their goal.
3. User lands on the Agent marketplace.
4. User sees available credits and a clear cost explanation.
5. User searches or filters Agents.
6. User chooses an Agent card or suggested prompt.
7. User enters chat with the prompt ready or easy to select.
8. User sends a message and receives a Markdown-formatted answer.
9. Credits are deducted only after a successful response.
10. If credits are insufficient, the user sees a clear buy-credits path.
11. After purchase, the user can continue the previous conversation.

## 5. Information architecture

### 5.1 Ordinary user navigation

Desktop top navigation:

- Agents
- Credits
- History
- Settings

Mobile bottom navigation:

- Agents
- Credits
- History
- Settings

Admin access should appear only for platform/root admins and should not dominate ordinary user navigation. On mobile, Admin can live under Settings or More.

### 5.2 Settings hierarchy

Separate settings by audience:

**Personal settings**

- Profile
- Security
- Sessions
- Billing

**Workspace admin settings**

- Team
- Agents
- Models
- Usage limits

**Platform admin**

- Branding
- Plans
- Financial
- Health
- Logs
- Config

This avoids presenting ordinary users with platform setup concepts.

## 6. Page-level design

### 6.1 `/dashboard` as Marketplace

Rename the mental model from dashboard to marketplace for ordinary users.

Recommended hero copy:

> Find the right AI agent for your next task.

Recommended subcopy:

> Search practical business agents, try a suggested prompt, and only pay credits after a successful response.

Hero should include:

- Current credit balance.
- Common cost explanation: “Most agents cost 1 credit/message.”
- Failure rule: “Failed responses are not charged.”
- Search box.
- Category chips.

Marketplace sections:

1. Recommended Agents
2. Popular Agents
3. Recently used Agents or recent conversations
4. Category-filtered Agent grid

Agent card content:

- Icon/avatar/color.
- Agent name.
- Category.
- One-line “best for” statement.
- Description.
- Suggested prompt preview.
- Credit cost.
- Capability labels only for capabilities that are actually usable.
- Primary CTA: “Start with this agent.”
- Optional secondary CTA: “Preview prompts.”

Role-aware empty states:

- Ordinary user with no Agents: explain that the workspace has not published Agents yet and provide contact/admin guidance.
- Admin user with no Agents: show “Create your first Agent” CTA.

### 6.2 `/chat/:agentId`

Chat is the core product experience and should feel premium.

P0 improvements:

- Render assistant messages as Markdown with headings, lists, tables, links, code blocks, and blockquotes.
- Show clear suggested prompts in the empty state.
- Let suggested prompts fill the composer or send directly.
- Show composer cost text: “This message costs 1 credit. Failed responses are not charged.”
- Add visible retry actions for failed messages.
- Add “Edit & retry” for user messages when practical.
- Add a buy-credits CTA inline when balance is insufficient.
- Make conversation history a drawer on mobile.
- Keep the composer accessible above the mobile keyboard.

Failure behavior:

- Model not configured: show “AI model is not configured. Contact an administrator.”
- Insufficient credits: show required credits and buy path.
- Stream interrupted: show partial content with a styled interrupted badge and retry action.
- Message load error: show retry, not only static error text.

### 6.3 `/buy-credits`

The page should explain what credits buy, not only list bundles.

Each bundle should show:

- Credit amount.
- Price.
- Estimated number of typical messages.
- Best-use label: occasional use, best value, team usage.
- “Credits are available immediately after checkout.”

If no bundles are available:

- Ordinary user: explain whether credits are available through plan subscription only and link to `/plan`.
- Admin: offer CTA to create or enable a credit bundle.

### 6.4 `/billing/success`

Do not only auto-redirect. Show a useful completion state:

- “Credits added successfully.”
- Updated balance if available.
- CTA: “Continue your conversation.”
- CTA: “Browse Agents.”
- Link: “View receipt” when invoice/transaction is available.

### 6.5 `/onboarding`

Shift onboarding from account setup to first success.

Recommended steps:

1. **Choose your goal** — Legal, Tax, Marketing, Support, E-commerce, Strategy, Other.
2. **Pick a recommended Agent** — show 1-3 Agents matching the goal.
3. **Choose a first prompt** — suggested prompts from the selected Agent.
4. **Start with credits explained** — one sentence about starting balance and cost.

Completion should navigate into the selected chat and preserve the chosen prompt.

### 6.6 `/admin`

Add a Launch Checklist to the admin dashboard.

Checklist items:

- Brand configured → `/admin/branding`
- LLM provider connected → `/admin/llm-config` or model settings
- First Agent published → Agent management
- Credit bundle or plan active → plan/bundle admin
- Stripe webhook healthy → health/config
- Test chat passed → marketplace/chat

Each item should have status, brief explanation, and direct CTA.

This transforms admin from “many tools” into “what remains before launch.”

## 7. Capability honesty

### 7.1 Model providers

The UI currently has provider-type concepts for OpenAI-compatible, Anthropic, and Gemini, while execution is mainly OpenAI-compatible.

For P0, use one of two strategies:

**Recommended P0 strategy:**

- Publicly support OpenAI-compatible providers.
- Mark Anthropic/Gemini as “Coming soon” or hide them from creation flows.
- Replace provider test stub with a real OpenAI-compatible test call.

**Alternative:**

- Implement true Anthropic/Gemini execution and real provider tests before advertising them.

Avoid letting operators configure a provider type that appears supported but fails during real chat.

### 7.2 Image/video generation

Image/video capability fields exist, but generation is currently coming soon.

P0 behavior:

- Do not show Image/Video capability labels on ordinary-user cards unless the capability is truly usable.
- Agent management can show disabled “Coming soon” controls.
- If users reach an image/video endpoint, show a polished coming-soon message and do not charge credits.

## 8. UI system improvements

Add or standardize shared components:

- `PageHeader`
- `EmptyState`
- `ErrorState` with retry
- `MobileBottomNav`
- `AgentCard`
- `CreditBalanceCard`
- `LaunchChecklist`
- `MarkdownMessage`
- `ConfirmModal` usage for destructive actions

Use existing Button/Card/Input/Alert/Badge primitives more consistently. Avoid one-off hand-coded styles for common controls unless a page has a specific visual need.

## 9. Terminology

Recommended terminology map:

| Avoid or limit | Use instead |
|---|---|
| Expert | Agent |
| Current Expert | Current Agent |
| Back to Experts | Back to Agents |
| Launch your AgentStore | Find the right AI agent |
| Credits bucket | Credits |
| Tenant | Workspace |
| Root tenant | Platform admin workspace, only in admin/operator docs |
| LLM Config | AI model settings / Model provider |
| Boilerplate | Open-source foundation, only in developer docs |

## 10. Documentation positioning

Separate public positioning into two lanes:

1. **AgentStore product:** AI Agent marketplace for ordinary users.
2. **AgentStore open-source foundation:** SaaS/AI platform boilerplate for builders.

Short combined positioning:

> AgentStore is a marketplace for practical AI agents, built on an open-source SaaS foundation.

The end-user product should not read like a boilerplate. The developer README can still explain the boilerplate/foundation value, but it should acknowledge the marketplace layer.

## 11. Prioritized roadmap

### P0 — user-ready launch polish

1. Normalize ordinary-user copy around “Agent,” “credits,” and “workspace.”
2. Redesign `/dashboard` into a marketplace page with search, category chips, credit explanation, and stronger Agent cards.
3. Improve `/chat/:agentId` with Markdown rendering, prompt-click flow, retry actions, credit cost text, and mobile drawer behavior.
4. Add mobile bottom navigation for ordinary users.
5. Improve `/buy-credits` and `/billing/success` so credits are understandable and purchase completion continues the user journey.
6. Add Admin Launch Checklist.
7. Make provider support honest: either support only OpenAI-compatible in P0 or implement/test other providers fully.
8. Hide or mark image/video as coming soon on ordinary-user surfaces.
9. Upgrade empty/error states for marketplace, chat, credits, activity, and Agent management.

### P1 — deeper product quality

1. Agent detail page.
2. Recently used and recommended Agents.
3. Agent usage counts, updated timestamps, and platform verified labels.
4. More personalized onboarding.
5. Admin responsive drawer.
6. Shared component refactor across admin and app pages.
7. More polished Agent management form.
8. Billing education improvements for plan vs credit bundle choices.

### P2 — marketplace expansion

1. Ratings/reviews if the marketplace grows beyond curated platform Agents.
2. Public Agent pages.
3. Rich install/publish workflows.
4. Creator marketplace/revenue sharing only after P0/P1 are stable.
5. True multimodal generation.
6. True multi-provider execution and provider-specific model settings.

## 12. Non-goals for this design

- Do not rewrite the app architecture.
- Do not add third-party creator marketplace features yet.
- Do not implement revenue sharing yet.
- Do not expand image/video generation before text-chat P0 is reliable.
- Do not advertise unsupported model providers.
- Do not make the ordinary-user experience depend on understanding admin concepts.

## 13. Verification expectations for later implementation

When this design turns into implementation, verification should include:

- `cd backend && go build ./...`
- `cd backend && go test ./internal/validation/...`
- Relevant backend handler tests for any API behavior changes.
- `cd frontend && npx tsc --noEmit`
- Relevant Vitest tests for changed pages/components.
- Playwright coverage for:
  - unauthenticated redirect
  - marketplace loads
  - mobile nav appears
  - Agent card starts chat
  - insufficient credits path
  - buy credits success path where possible

Manual verification should include a smoke walkthrough:

1. Sign up/log in.
2. Complete onboarding.
3. Browse marketplace.
4. Start a chat.
5. Send a message.
6. See credits explanation.
7. Trigger insufficient credits or buy path.
8. Return to chat after purchase/success.
9. Admin checks Launch Checklist.

## 14. Success criteria

The design is successful when an ordinary user can say:

- “I know what this product does.”
- “I know which Agent to try.”
- “I understand what credits are and when they are charged.”
- “I can chat successfully on desktop or mobile.”
- “If I run out of credits, I know exactly how to continue.”

And an operator can say:

- “I know what remains before launch.”
- “I can configure branding, models, Agents, credits, and billing from clear entry points.”
- “The product no longer promises capabilities that are not actually working.”
