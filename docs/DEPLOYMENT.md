# AgentStore Deployment Guide

This guide covers production deployment for AgentStore. It assumes you are deploying the upstream AgentStore app as a single Docker container: Go backend + built React frontend served by the Go process.

For local development, see `docs/DEVELOPMENT.md`. For the launch smoke test, see `LAUNCH_SMOKE_TEST.md`.

## Runtime model

The production container:

1. Builds the Go backend from `backend/cmd/server`.
2. Builds the React frontend with Vite.
3. Copies `backend/config/prod.example.yaml` into the image as `/app/config/prod.yaml`.
4. Expands `${ENV_VAR}` placeholders from runtime environment variables.
5. Serves the React SPA from `/app/static`.
6. Listens on `SERVER_PORT` (set this to `8080` on Fly.io).

There is no nginx, Caddy, or separate frontend server in the default AgentStore deployment.

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
- OpenAI-compatible LLM provider configured in the admin UI after deploy.

## Required environment variables

| Variable | Required | Notes |
|----------|----------|-------|
| `SERVER_PORT` | Yes | `8080` on Fly.io; otherwise match the port exposed by your container host. |
| `DATABASE_NAME` | Yes | Logical project identity. Two apps sharing the same name share the same user base. |
| `MONGODB_URI` | Yes | MongoDB connection string. |
| `JWT_ACCESS_SECRET` | Yes | Generate with `openssl rand -hex 32`. |
| `JWT_REFRESH_SECRET` | Yes | Generate with `openssl rand -hex 32`. |
| `WEBHOOK_ENCRYPTION_KEY` | Yes | Generate with `openssl rand -hex 32`. Used for outgoing webhook secret material. |
| `FRONTEND_URL` | Yes | Public HTTPS URL for CORS, OAuth redirects, and emails. |
| `APP_NAME` | Yes | Product/app name shown in UI and email. |
| `FROM_EMAIL` | Yes for email | Sender email address. |
| `FROM_NAME` | Yes for email | Sender display name. |

Optional variables are listed in `.env.example` and `README.md`.

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

After the first deploy, create the root tenant and root owner account.

Run the CLI from your workstation using production environment variables and a network path to the same MongoDB database:

```bash
cp .env.example .env.production
# Edit .env.production so it matches the production secrets above.

set -a && source .env.production && set +a
export AGENTSTORE_ENV=prod
cd backend
go run ./cmd/agentstore setup
```

Then open:

```text
https://your-app-name.fly.dev
```

Log in with the root owner account created by the setup command.

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
