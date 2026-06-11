# AgentStore Local Testing Guide

## System Status
- **Backend**: http://localhost:4290
- **Frontend**: http://localhost:4280
- **Database**: MongoDB running (docker:27017)

## Credentials

### Admin / Operator Account
```
Email: admin@agentstore.local
Password: Admin#Pass2026
```

### Test Customer Accounts
```
Email: alice@testco.local
Password: Alice#Pass2026

Email: customer@example.local
Password: Customer#Pass2026
```

## Testing Checklist

### 1. Operator / Admin Panel (Role: Platform Operator)

**Entry**: http://localhost:4280 → Login as admin → Admin Dashboard

**What to Test**:
- [ ] **Dashboard**: Check health metrics, user/tenant counts
- [ ] **Users Tab**: See all registered users, search by email
- [ ] **Tenants Tab**: View customer tenants, verify isolation
- [ ] **Plans Tab**: View default "Free" plan, ready to add paid plans
- [ ] **Branding Tab**: Customize app name, colors, logo (white-label)
- [ ] **Config Tab**: Runtime configuration management
- [ ] **System Health**: CPU, memory, disk, HTTP metrics real-time
- [ ] **LLM Config**: Verify provider settings (Admin → Config → LLM)

**Expected Admin Actions**:
- Create new subscription plans
- Manage user accounts and roles
- Customize branding for white-label
- Configure webhooks (when set up)
- View system telemetry and health
- Set promotion codes and coupon rules

### 2. Customer / User Experience (Role: Platform User)

**Entry**: http://localhost:4280 → Login as alice OR register new user

**What to Test**:
- [ ] **Login**: Use alice@testco.local / Alice#Pass2026
- [ ] **Dashboard**: User workspace overview
- [ ] **Agent Discovery**: Browse available agents (6 pre-loaded)
  - Legal Expert
  - Tax Advisor
  - Marketing Copywriter
  - Customer Support Expert
  - Telecom Business Advisor
  - Cross-border E-commerce Expert
- [ ] **Chat Interface**: Select an agent and send a test message
  - **⚠️ Note**: Chat will fail if LLM API key lacks model access (see Note below)
- [ ] **Conversation History**: View past conversations
- [ ] **Credits**: View available credits (should show initial allocation)
- [ ] **Billing**: Upgrade plan, purchase credit bundles (if Stripe configured)
- [ ] **Team / Members**: Invite team members (if applicable)
- [ ] **Settings**: Profile, security, theme, preferences

**Expected User Actions**:
- Discover and chat with marketplace agents
- Track credit usage per agent interaction
- Manage profile and security settings
- Manage team membership and roles
- Purchase credits or upgrade subscription

### 3. API Testing (Programmatic)

**Bootstrap Status**:
```bash
curl http://localhost:4290/api/bootstrap/status
```

**Admin Login**:
```bash
curl -X POST http://localhost:4290/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@agentstore.local",
    "password": "Admin#Pass2026"
  }'
```

**Get Admin Dashboard** (requires JWT + X-Tenant-ID):
```bash
TOKEN="<access_token_from_login>"
TENANT="6a283b883b376530107e76cc"

curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Tenant-ID: $TENANT" \
     http://localhost:4290/api/admin/dashboard
```

**List Plans**:
```bash
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Tenant-ID: $TENANT" \
     http://localhost:4290/api/admin/plans
```

**List Agents** (as customer):
```bash
CUSTOMER_TOKEN="<customer_jwt>"
CUSTOMER_TENANT="<customer_tenant_id>"

curl -H "Authorization: Bearer $CUSTOMER_TOKEN" \
     -H "X-Tenant-ID: $CUSTOMER_TENANT" \
     http://localhost:4290/api/chat/agents
```

**View API Documentation**:
```
http://localhost:4290/api/docs
```

## Known Issues & Workarounds

### ⚠️ Chat Not Working?
**Issue**: Chat returns "AI service temporarily unavailable"

**Root Cause**: The LLM API key provided lacks permissions for the MiniMax-M2.5 model on the provider account.

**Solution**:
1. Use a valid MiniMax API key with model access
2. Or switch to another OpenAI-compatible provider (e.g., OpenAI, Perplexity, LocalAI)
3. Update via Admin UI:
   - Login as admin
   - Go to Admin → Config → LLM Settings
   - Update API key, base URL, model name
   - Test chat again

**Temporary Workaround** (for architecture testing):
- Use a dummy but syntactically correct key to test the UI flow
- The API calls will fail at the provider, but routing/multi-tenancy is verified

### Rate Limiting
If you see "Rate limit exceeded" messages:
- This is the security rate limiter protecting auth endpoints
- Wait the indicated seconds before retrying
- This is normal and expected behavior

### Email Notifications Disabled
Auth flows (signup, password reset, email verification) log verification tokens to console instead of sending emails because Resend API key is not configured.

To enable email:
1. Sign up at [Resend.com](https://resend.com)
2. Get your API key
3. Add to `.env`: `RESEND_API_KEY=re_...`
4. Restart backend

### Stripe Billing Optional
Billing is fully functional but optional for dev. To fully test:
1. Create a Stripe test account
2. Add test keys to `.env`: `STRIPE_SECRET_KEY`, `STRIPE_PUBLISHABLE_KEY`
3. Create plans in Admin → Plans
4. Restart backend

## Environment Variables

See `.env` in the project root:
```bash
# Core
DATABASE_NAME=agentstore-dev
MONGODB_URI=mongodb://localhost:27017
JWT_ACCESS_SECRET=<random_hex>
JWT_REFRESH_SECRET=<random_hex>

# LLM Provider
OPENAI_API_KEY=sk-...
OPENAI_BASE_URL=https://ai.wuwutech.com
OPENAI_MODEL=MiniMax-M2.5

# Optional integrations
RESEND_API_KEY=  # Email
STRIPE_SECRET_KEY=  # Billing
GOOGLE_CLIENT_ID=  # OAuth
```

## Logs & Debugging

**Backend Logs**:
```bash
tail -f /tmp/agentstore-backend.log
```

**Frontend Logs**:
```bash
tail -f /tmp/agentstore-frontend.log
```

**MongoDB**:
```bash
# Access mongo shell
docker exec -it agentstore-mongo mongosh

# In shell:
use agentstore-dev
db.users.find().pretty()  # List users
db.tenants.find().pretty()  # List tenants
db.agents.find().pretty()  # List agents
```

## Architecture Notes

### Multi-Tenancy Model
- **Root Tenant**: System operators (you)
- **Customer Tenants**: One per customer/organization
- **Data Isolation**: All data scoped to tenant via X-Tenant-ID header

### Role-Based Access Control (RBAC)
- **Owner**: Full control (destructive operations)
- **Admin**: Read/write (no deletes)
- **User**: Read-only or limited write

### Credit System
- **Subscription Credits**: Monthly allocation per plan
- **Purchased Credits**: One-time packs (via Stripe)
- **Agent Cost**: Each agent consumes credits (configurable)
- **Enforcement**: Blocked if insufficient credits

### Agent Marketplace
- Pre-seeded with 6 demo agents
- Operator can add/remove agents
- Each agent has name, description, category, cost
- Customer sees agents in their tenant workspace

## Next Steps

### To Test Chat End-to-End
1. Get a valid LLM API key (MiniMax, OpenAI, etc.)
2. Update Admin → Config → LLM Settings
3. Send test message in Chat UI
4. Verify credits deducted

### To Test Billing
1. Set up Stripe test account
2. Add keys to `.env` and restart backend
3. Admin → Plans → Create a $9/month plan
4. Customer → Billing → Upgrade plan
5. Enter Stripe test card `4242 4242 4242 4242`

### To Test Email Integration
1. Get Resend API key
2. Add to `.env` and restart backend
3. Register new user and check email verification flow

### To Enable OAuth (Google/GitHub)
1. Create OAuth apps in provider console
2. Get client ID, secret, redirect URL
3. Add to `.env` and restart backend
4. Auth forms will show provider buttons

---

**Happy testing! 🚀**

For production deployment, see `docs/DEPLOYMENT.md`.
