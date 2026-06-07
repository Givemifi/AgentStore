# Smoke Test: Real MongoDB and LLM Billing

This checklist validates AgentStore against a real MongoDB deployment and a real OpenAI-compatible LLM provider. Run it before production deploys, after billing/credit changes, and after changes to chat, tenant, or model configuration.

Do not commit real API keys, JWT secrets, MongoDB URIs, Stripe keys, cookies, or bearer tokens. Use a disposable tenant/user and the cheapest available LLM model.

## Scope

This smoke test verifies:

- The backend connects to the intended MongoDB database.
- The app can create/read/update data in real MongoDB collections.
- Admin LLM configuration is saved and reloaded.
- Expert Chat can call a real LLM provider.
- Chat completion deducts credits and writes `usage_events`.
- Failed provider calls do not deduct credits.
- Insufficient-credit requests fail before provider calls and do not create chargeable usage.

## Prerequisites

- Go 1.25+.
- Node.js 22+ for frontend commands. Vite 7 requires Node `^20.19.0 || >=22.12.0`; Node 18.20.8 will print a warning during builds.
- Access to a real MongoDB database. Use a smoke-test database name such as `agentstore-smoke-YYYYMMDD`.
- A real OpenAI-compatible LLM endpoint and API key.
- A test root/admin user.
- A disposable tenant with at least 6 credits, or permission to edit tenant credits from the admin UI.
- Optional: Stripe test-mode keys if you also want to verify checkout/webhook flows.

## Environment setup

1. Export the real environment values in your shell. Keep them local.

   ```bash
   export DATABASE_NAME="agentstore-smoke-YYYYMMDD"
   export MONGODB_URI="mongodb+srv://..."
   export JWT_ACCESS_SECRET="$(openssl rand -hex 32)"
   export JWT_REFRESH_SECRET="$(openssl rand -hex 32)"
   export SERVER_HOST="localhost"
   export SERVER_PORT="4290"
   export FRONTEND_URL="http://localhost:4280"
   export VITE_API_URL="http://localhost:4290"
   ```

2. Start the backend from the repo root.

   ```bash
   cd backend
   go run ./cmd/server
   ```

   Expected result: the server starts on `http://localhost:4290` and does not log MongoDB connection errors.

3. In another terminal, start the frontend.

   ```bash
   cd frontend
   npm ci
   npm run dev
   ```

   Expected result: the frontend starts on `http://localhost:4280` without Node/Vite engine warnings when using Node 22+.

4. If the database is new, initialize the root tenant and owner account.

   ```bash
   cd backend
   go run ./cmd/agentstore setup
   ```

   Expected result: the CLI creates the root tenant and owner user in the configured MongoDB database.

## Pre-flight checks

1. Open `http://localhost:4290/health`.

   Expected result: HTTP 200.

2. Log in at `http://localhost:4280` as the root/admin user.

   Expected result: the Dashboard loads and admin navigation is available.

3. Confirm you are using the intended MongoDB database before writing data.

   ```javascript
   use agentstore-smoke-YYYYMMDD
   db.tenants.find({}, { name: 1, slug: 1, isRoot: 1, billingStatus: 1, billingWaived: 1, subscriptionCredits: 1, purchasedCredits: 1 }).limit(5)
   ```

   Expected result: the smoke-test root tenant is present. Stop if this points at an unintended production database.

## Configure LLM provider

AgentStore has two LLM configuration paths:

- Root admin fallback config at `/admin/llm-config`, backed by `/api/admin/llm-config` and `llm_configs`.
- Tenant model settings at `/settings` -> Model Settings, backed by `model_providers` and `model_configs`.

For this smoke test, configure the root admin fallback first because static Expert Chat agents use it.

1. Go to `http://localhost:4280/admin/llm-config`.

2. Enter:

   - API Key: your real provider key.
   - Base URL: the provider's OpenAI-compatible base URL, for example `https://api.openai.com/v1`.
   - Model: a cheap text model, for example `gpt-4o-mini` or the provider-specific equivalent.
   - Enable AI Chat Feature: checked.

3. Click **Save Configuration**.

   Expected result: the UI shows `Configuration saved`.

4. Verify the config exists in MongoDB without exposing the full key in notes or screenshots.

   ```javascript
   db.llm_configs.find({}, { baseURL: 1, model: 1, isActive: 1, updatedAt: 1 }).sort({ updatedAt: -1 }).limit(3)
   ```

   Expected result: one active config with the expected base URL and model.

## Prepare tenant credits

1. In the admin UI, open the disposable tenant profile.

2. Set a known balance:

   - `subscriptionCredits`: `6`
   - `purchasedCredits`: `0`
   - `billingStatus`: `none` or `active`, or `billingWaived`: `true`

   The chat and usage routes allow root tenants, billing-waived tenants, active tenants, and `none` billing status tenants. Past-due/canceled tenants should be blocked with `402` unless waived.

3. Save the tenant.

4. Verify the starting balance in MongoDB.

   ```javascript
   db.tenants.findOne(
     { slug: "YOUR_TENANT_SLUG" },
     { name: 1, slug: 1, billingStatus: 1, billingWaived: 1, subscriptionCredits: 1, purchasedCredits: 1 }
   )
   ```

   Expected result: total credits are exactly `6`.

## Golden path: real LLM call and credit deduction

1. Switch to the disposable tenant in the app.

2. Open the Dashboard or an Expert Chat page.

3. Start a chat with the static `Legal Expert` agent, or navigate directly to:

   ```text
   http://localhost:4280/chat/legal-expert
   ```

4. Send this low-cost prompt:

   ```text
   Reply with exactly: smoke-ok
   ```

5. Wait for the streamed response to finish.

   Expected result in the UI:

   - A response appears from the assistant.
   - No `AI service is temporarily unavailable` message appears.
   - The visible credit balance decreases by `3` credits. Static catalog agents currently cost 3 credits per text message.

6. Verify MongoDB writes.

   ```javascript
   const tenant = db.tenants.findOne({ slug: "YOUR_TENANT_SLUG" })
   db.usage_events.find({ tenantId: tenant._id, type: "agent_chat" }).sort({ createdAt: -1 }).limit(3)
   db.conversations.find({ tenantId: tenant._id }).sort({ updatedAt: -1 }).limit(3)
   db.chat_messages.find({ tenantId: tenant._id }).sort({ createdAt: -1 }).limit(6)
   db.tenants.findOne({ _id: tenant._id }, { subscriptionCredits: 1, purchasedCredits: 1 })
   ```

   Expected result:

   - `usage_events` has a new `agent_chat` event with `quantity: 3`.
   - The event metadata includes `agentId`, `conversationId`, and `model`.
   - `conversations` has a new/updated conversation for `legal-expert`.
   - `chat_messages` has one user message and one completed assistant message.
   - The assistant message has `creditsCharged: 3` and a model value.
   - Tenant total credits decreased from `6` to `3`.

7. Verify the usage summary API through the UI.

   - Open the Dashboard or Chat page.
   - Confirm the displayed remaining credits match MongoDB.

   Optional API check from an authenticated browser session:

   ```text
   GET /api/usage/summary
   ```

   Expected result: `totalCreditsUsed` includes the new 3-credit `agent_chat` event, and `subscriptionCredits + purchasedCredits` equals the tenant balance in MongoDB.

## Golden path: second charge reaches zero balance

1. Send one more prompt in the same chat:

   ```text
   Reply with exactly: smoke-ok-2
   ```

2. Wait for completion.

3. Verify MongoDB again.

   ```javascript
   const tenant = db.tenants.findOne({ slug: "YOUR_TENANT_SLUG" })
   db.usage_events.find({ tenantId: tenant._id, type: "agent_chat" }).sort({ createdAt: -1 }).limit(5)
   db.tenants.findOne({ _id: tenant._id }, { subscriptionCredits: 1, purchasedCredits: 1 })
   ```

   Expected result:

   - Another `agent_chat` event with `quantity: 3` exists.
   - Tenant total credits are now `0`.

## Negative path: insufficient credits

1. With the same tenant at `0` total credits, send another chat prompt.

2. Expected result:

   - The request fails with an insufficient-credits error.
   - No real provider response is generated.
   - No new `usage_events` document is inserted.
   - Tenant credits remain `0`.

3. Confirm in MongoDB.

   ```javascript
   const tenant = db.tenants.findOne({ slug: "YOUR_TENANT_SLUG" })
   db.usage_events.find({ tenantId: tenant._id, type: "agent_chat" }).sort({ createdAt: -1 }).limit(5)
   db.tenants.findOne({ _id: tenant._id }, { subscriptionCredits: 1, purchasedCredits: 1 })
   ```

## Negative path: provider failure does not charge credits

1. In `/admin/llm-config`, temporarily replace the API key with an invalid value and save.

2. Restore the disposable tenant to 3 credits.

   ```javascript
   db.tenants.updateOne(
     { slug: "YOUR_TENANT_SLUG" },
     { $set: { subscriptionCredits: 3, purchasedCredits: 0, updatedAt: new Date() } }
   )
   ```

3. Send a chat prompt.

4. Expected result:

   - The UI shows an AI-service failure message.
   - No usage event is created for the failed provider call.
   - Tenant credits remain `3`.
   - The assistant `chat_messages` document, if created, has `status: "error"` and `creditsCharged: 0`.

5. Restore the valid LLM API key immediately after the check.

## Optional: direct API checks

Use these only when you already have a valid auth token and tenant header from the browser session. Do not paste tokens into commits, issue comments, or shared logs.

1. List agents.

   ```bash
   curl -sS "$BASE_URL/api/chat/agents" \
     -H "Authorization: Bearer $ACCESS_TOKEN" \
     -H "X-Tenant-ID: $TENANT_ID"
   ```

   Expected result: response includes `legal-expert` with `textMessageCredits` or equivalent credit cost of `3`.

2. Send non-streaming chat.

   ```bash
   curl -sS "$BASE_URL/api/chat" \
     -H "Authorization: Bearer $ACCESS_TOKEN" \
     -H "X-Tenant-ID: $TENANT_ID" \
     -H "Content-Type: application/json" \
     --data '{"agentId":"legal-expert","message":"Reply with exactly: smoke-api-ok"}'
   ```

   Expected result: JSON includes `conversationId`, `answer`, `creditsCharged: 3`, `remainingCredits`, and `model`.

3. Read usage summary.

   ```bash
   curl -sS "$BASE_URL/api/usage/summary" \
     -H "Authorization: Bearer $ACCESS_TOKEN" \
     -H "X-Tenant-ID: $TENANT_ID"
   ```

   Expected result: JSON includes `usage`, `totalCreditsUsed`, `subscriptionCredits`, and `purchasedCredits` matching MongoDB.

4. Record a manual usage event only if you specifically need to test `/api/usage/record` separately from chat.

   ```bash
   curl -sS "$BASE_URL/api/usage/record" \
     -H "Authorization: Bearer $ACCESS_TOKEN" \
     -H "X-Tenant-ID: $TENANT_ID" \
     -H "Content-Type: application/json" \
     --data '{"type":"smoke_manual_usage","quantity":1,"metadata":{"source":"smoke-test"}}'
   ```

   Expected result: HTTP 200 JSON includes `id`, `type: "smoke_manual_usage"`, and `quantity: 1`; tenant credits decrease by 1. Do not use this as the only billing check because it does not call the LLM provider.

## Optional: Stripe billing smoke

Run this only in Stripe test mode.

1. Configure `STRIPE_SECRET_KEY`, `STRIPE_PUBLISHABLE_KEY`, and `STRIPE_WEBHOOK_SECRET`.

2. Start the Stripe CLI forwarder.

   ```bash
   stripe listen --forward-to localhost:4290/api/billing/webhook
   ```

3. From the app, purchase a credit bundle or start a checkout session for a test plan.

4. Complete checkout using Stripe test card details.

5. Expected result:

   - `/api/billing/webhook` accepts the event.
   - `webhook_events` contains the Stripe event ID once.
   - `financial_transactions` contains the purchase/subscription record.
   - Tenant credits or billing status update according to the plan/bundle.

## Cleanup

1. Restore the valid LLM configuration if you changed it during negative testing.

2. Remove or archive disposable tenants/users if the database will be reused.

3. If the database was created only for smoke testing, drop the whole smoke-test database.

   ```javascript
   use agentstore-smoke-YYYYMMDD
   db.dropDatabase()
   ```

4. Rotate any provider key that was pasted into a shared environment or terminal recording.

## Pass/fail criteria

Pass only if all required checks are true:

- Backend health is HTTP 200 with the real MongoDB URI.
- Root/admin login succeeds.
- `/admin/llm-config` saves an active provider config.
- A real Expert Chat response completes.
- Each successful static agent chat deducts exactly 3 credits.
- Each successful chat writes one `agent_chat` usage event.
- Usage summary and MongoDB tenant balances agree.
- Insufficient-credit requests do not call the provider and do not write chargeable usage.
- Provider failures do not deduct credits and do not write chargeable usage.

Fail the smoke test and investigate before deploy if any balance, usage event, or chat message status disagrees with the expected results above.
