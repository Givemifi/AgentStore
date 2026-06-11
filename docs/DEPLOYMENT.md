# AgentStore Deployment Guide

This guide covers production deployment for AgentStore. It assumes you are deploying the upstream AgentStore app as a single Docker container: Go backend + built React frontend served by the Go process.

For local development, see `docs/DEVELOPMENT.md`.

**Other languages:** [简体中文](DEPLOYMENT.zh-CN.md) | [日本語](DEPLOYMENT.ja.md)

## Runtime model

The production container:

1. Builds the Go backend from `backend/cmd/server`.
2. Builds the React frontend with Vite.
3. Copies `backend/config/prod.example.yaml` into the image as `/app/config/prod.yaml`.
4. Expands `${ENV_VAR}` placeholders from runtime environment variables.
5. Serves the React SPA from `/app/static`.
6. Listens on `SERVER_PORT` (set this to `8080` on Fly.io).

There is no nginx, Caddy, or separate frontend server in the default AgentStore deployment.

## Pre-built image

Every release publishes a multi-platform Docker image to GitHub Container Registry:

```bash
docker pull ghcr.io/givemifi/agentstore:latest
# or a specific version:
docker pull ghcr.io/givemifi/agentstore:1.3.0
```

Using the pre-built image avoids needing Go or Node.js toolchains locally.

## Required external services

Minimum production deployment:

- MongoDB Atlas or another MongoDB-compatible deployment.
- A production host capable of running Docker.
- Production secrets for JWT signing and webhook encryption.

Optional integrations:

- Resend for email verification, password resets, and invitations.
- Stripe for subscriptions, credit bundle purchases, invoices, refunds, and disputes.
- Google/GitHub/Microsoft OAuth providers.
- DataDog for log and metrics forwarding.
- OpenAI-compatible LLM provider (configurable via env vars or the admin UI).

## Required environment variables

| Variable | Required | Notes |
|----------|----------|-------|
| `SERVER_PORT` | Yes | `8080` on Fly.io; otherwise match the port exposed by your container host. |
| `DATABASE_NAME` | Yes | Logical project identity. |
| `MONGODB_URI` | Yes | MongoDB connection string. |
| `JWT_ACCESS_SECRET` | Yes | Generate with `openssl rand -hex 32`. |
| `JWT_REFRESH_SECRET` | Yes | Generate with `openssl rand -hex 32`. |
| `WEBHOOK_ENCRYPTION_KEY` | Yes | Generate with `openssl rand -hex 32`. |
| `FRONTEND_URL` | Yes | Public HTTPS URL for CORS, OAuth redirects, and emails. |
| `APP_NAME` | Yes | Product/app name shown in UI and email. |
| `FROM_EMAIL` | Yes for email | Sender email address. |
| `FROM_NAME` | Yes for email | Sender display name. |
| `OPENAI_API_KEY` | Optional | Seeds an LLM config on first boot so chat works immediately. |
| `OPENAI_BASE_URL` | Optional | Base URL of any OpenAI-compatible endpoint. |
| `OPENAI_MODEL` | Optional | Model name, e.g. `gpt-4o`. |

All three `OPENAI_*` vars must be set for the auto-seed to trigger. Leave them blank and configure the LLM provider in Admin → LLM Configuration after deploy instead.

Optional OAuth / billing / observability variables are listed in `.env.docker.example`.

## Docker Compose (recommended for self-hosted)

The fastest way to run AgentStore on any machine with Docker installed. MongoDB is managed automatically — no Atlas account needed.

### 1. Clone and configure

```bash
git clone https://github.com/Givemifi/AgentStore.git
cd AgentStore

cp .env.docker.example .env
```

Edit `.env` and set the required secrets plus optional LLM provider:

```bash
JWT_ACCESS_SECRET=$(openssl rand -hex 32)
JWT_REFRESH_SECRET=$(openssl rand -hex 32)
WEBHOOK_ENCRYPTION_KEY=$(openssl rand -hex 32)

# Optional: seed LLM config on first boot (any OpenAI-compatible endpoint)
OPENAI_API_KEY=sk-...
OPENAI_BASE_URL=https://api.openai.com
OPENAI_MODEL=gpt-4o
```

### 2. Start

```bash
docker compose up -d
```

To use the pre-built GHCR image instead of building from source, comment out `build: .` and uncomment the `image:` line in `docker-compose.yml`.

### 3. First-run setup

Open `http://localhost:8080` — a setup wizard creates your admin account.

**Non-interactive (automated deploy):** set these in `.env` before startup:

```bash
AGENTSTORE_SETUP_ORG=My Company
AGENTSTORE_SETUP_NAME=Jane Doe
AGENTSTORE_SETUP_EMAIL=admin@example.com
AGENTSTORE_SETUP_PASSWORD=YourSecurePass123!
```

### 4. Configure LLM (if not using env vars)

Admin → LLM Configuration → fill in your provider URL, API key, and model.

---

## Fly.io deployment

### 1. Install Fly CLI

```bash
curl -L https://fly.io/install.sh | sh
flyctl auth login
```

### 2. Create the Fly app

```bash
flyctl apps create your-app-name --org your-org
```

If you use a different app name than `agentstore`, update `fly.toml`:

```toml
app = 'your-app-name'
```

### 3. Set required secrets

```bash
flyctl secrets set \
  SERVER_PORT="8080" \
  DATABASE_NAME="your-db-name" \
  MONGODB_URI="mongodb+srv://..." \
  JWT_ACCESS_SECRET="$(openssl rand -hex 32)" \
  JWT_REFRESH_SECRET="$(openssl rand -hex 32)" \
  WEBHOOK_ENCRYPTION_KEY="$(openssl rand -hex 32)" \
  FRONTEND_URL="https://your-app-name.fly.dev" \
  APP_NAME="YourApp" \
  FROM_EMAIL="noreply@yourdomain.com" \
  FROM_NAME="YourApp"
```

### 4. Set optional integration secrets

Use only what you plan to enable:

```bash
flyctl secrets set \
  RESEND_API_KEY="re_..." \
  STRIPE_SECRET_KEY="sk_live_..." \
  STRIPE_PUBLISHABLE_KEY="pk_live_..." \
  STRIPE_WEBHOOK_SECRET="whsec_..." \
  GOOGLE_CLIENT_ID="..." \
  GOOGLE_CLIENT_SECRET="..." \
  GOOGLE_REDIRECT_URL="https://your-app-name.fly.dev/api/auth/google/callback" \
  GITHUB_CLIENT_ID="..." \
  GITHUB_CLIENT_SECRET="..." \
  GITHUB_REDIRECT_URL="https://your-app-name.fly.dev/api/auth/github/callback" \
  MICROSOFT_CLIENT_ID="..." \
  MICROSOFT_CLIENT_SECRET="..." \
  MICROSOFT_REDIRECT_URL="https://your-app-name.fly.dev/api/auth/microsoft/callback" \
  DATADOG_API_KEY="..." \
  DATADOG_SITE="us5.datadoghq.com"
```

### 5. Deploy

```bash
flyctl deploy
```

### 6. Verify health

```bash
curl -fsS https://your-app-name.fly.dev/health
```

Expected: HTTP 200.

## First-time initialization

After the first deploy, open your app URL in a browser. A setup wizard will appear if the system is not yet initialized — fill in organization name, admin name, email, and password, then click **Create Account**.

**Alternative — CLI (if you can reach MongoDB from your workstation):**

```bash
set -a && source .env.production && set +a
export AGENTSTORE_ENV=prod
cd backend
go run ./cmd/agentstore setup
```

**Alternative — non-interactive (automated pipelines):** set `AGENTSTORE_SETUP_ORG`, `AGENTSTORE_SETUP_NAME`, `AGENTSTORE_SETUP_EMAIL`, `AGENTSTORE_SETUP_PASSWORD` as environment variables. The CLI will skip the TTY prompts and initialize automatically.

Log in with the admin account credentials you just created.

### Configure LLM provider

If you did not set `OPENAI_*` env vars before deploy, go to Admin → LLM Configuration and enter your provider URL, API key, and model name. Chat will not work until this step is complete.

## Stripe webhook setup

If billing is enabled, create a Stripe webhook endpoint:

```text
https://your-app-name.fly.dev/api/billing/webhook
```

Configure these event types at minimum:

- `checkout.session.completed`
- `customer.subscription.created`
- `customer.subscription.updated`
- `customer.subscription.deleted`
- `invoice.payment_succeeded`
- `invoice.payment_failed`
- `charge.refunded`
- `charge.dispute.created`

Copy the signing secret into:

```bash
flyctl secrets set STRIPE_WEBHOOK_SECRET="whsec_..."
```

Redeploy or restart the app after changing secrets if your platform requires it.

## Launch checklist

Before inviting real users:

1. Log in as the root owner.
2. Open **Admin → Launch Readiness**.
3. Complete each checklist item:
   - Branding configured.
   - OpenAI-compatible text model connected.
   - At least one text-chat Agent published.
   - Credit bundle or plan active.
   - Stripe webhook healthy, if billing is enabled.
   - Resend healthy, if email flows are required.
   - Real chat smoke test passed.
4. Run `LAUNCH_SMOKE_TEST.md` against disposable/test-mode credentials before switching to live billing.

## Docker on another platform

Build:

```bash
docker build -t agentstore .
```

Run:

```bash
docker run --rm -p 8080:8080 \
  -e AGENTSTORE_ENV=prod \
  -e SERVER_PORT=8080 \
  -e DATABASE_NAME="your-db-name" \
  -e MONGODB_URI="mongodb+srv://..." \
  -e JWT_ACCESS_SECRET="$(openssl rand -hex 32)" \
  -e JWT_REFRESH_SECRET="$(openssl rand -hex 32)" \
  -e WEBHOOK_ENCRYPTION_KEY="$(openssl rand -hex 32)" \
  -e FRONTEND_URL="https://your-domain.example" \
  -e APP_NAME="YourApp" \
  -e FROM_EMAIL="noreply@yourdomain.com" \
  -e FROM_NAME="YourApp" \
  agentstore
```

Use your platform's secret manager instead of inline `-e` flags for real deployments.

## Common issues

### App deploys but health check fails

Check `SERVER_PORT`. Fly.io routes to port `8080`, so the app must also listen on `8080`.

### Docker build fails on missing `prod.yaml`

The Dockerfile should copy `backend/config/prod.example.yaml` into the runtime image as `config/prod.yaml`. Do not commit local `prod.yaml` secrets.

### OAuth buttons are missing

Set the provider client ID, client secret, and redirect URL. Redirect URLs must exactly match the provider app configuration.

### Billing checkout fails

Check `STRIPE_SECRET_KEY`, `STRIPE_PUBLISHABLE_KEY`, and `STRIPE_WEBHOOK_SECRET`. Also confirm plans/credit bundles exist and are active in admin settings.

### Emails do not send

Set `RESEND_API_KEY`, `FROM_EMAIL`, and `FROM_NAME`. Verify your sending domain in Resend.

### Users can log in but chat fails

Configure an OpenAI-compatible provider under **Settings → Models**, set a default text model, publish at least one text-chat Agent, and ensure the tenant has credits.
