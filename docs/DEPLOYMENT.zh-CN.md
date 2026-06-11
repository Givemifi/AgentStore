# AgentStore 部署指南

本文档介绍如何将 AgentStore 部署到生产环境。

**语言：** [English](DEPLOYMENT.md) | 简体中文 | [日本語](DEPLOYMENT.ja.md)

---

## 运行模型

生产容器将 Go 后端二进制与编译好的 React 前端打包在一个镜像中，监听 `SERVER_PORT`（Fly.io 推荐 `8080`）。无 nginx / 独立前端服务器。

## 预构建镜像

每次 Release 自动发布到 GHCR：

```bash
docker pull ghcr.io/givemifi/agentstore:latest
docker pull ghcr.io/givemifi/agentstore:1.3.0  # 指定版本
```

---

## Docker Compose（自托管推荐方案）

无需安装 Go 或 Node.js，MongoDB 随容器自动启动。

### 1. 克隆并配置

```bash
git clone https://github.com/Givemifi/AgentStore.git
cd AgentStore

cp .env.docker.example .env
```

编辑 `.env`，填写必要字段：

```bash
# 必填密钥（分别运行命令生成）
JWT_ACCESS_SECRET=$(openssl rand -hex 32)
JWT_REFRESH_SECRET=$(openssl rand -hex 32)
WEBHOOK_ENCRYPTION_KEY=$(openssl rand -hex 32)

# 可选：填写后部署即可对话（支持任何 OpenAI 兼容端点）
OPENAI_API_KEY=sk-...
OPENAI_BASE_URL=https://api.openai.com
OPENAI_MODEL=gpt-4o
```

### 2. 启动

```bash
docker compose up -d
```

如需使用预构建镜像（跳过本地构建），在 `docker-compose.yml` 中注释掉 `build: .`，取消注释 `image:` 行。

### 3. 首次初始化

浏览器打开 `http://localhost:8080`，按安装向导创建管理员账户。

**全自动初始化（CI/CD 场景）：** 在 `.env` 中设置以下变量，首次启动时自动跳过向导：

```bash
AGENTSTORE_SETUP_ORG=我的公司
AGENTSTORE_SETUP_NAME=张三
AGENTSTORE_SETUP_EMAIL=admin@example.com
AGENTSTORE_SETUP_PASSWORD=YourSecurePass123!
```

### 4. 配置 LLM（如未通过环境变量配置）

登录 → Admin → LLM Configuration → 填入 provider URL、API Key、模型名称。

---

## Fly.io 部署

### 1. 安装 Fly CLI

```bash
curl -L https://fly.io/install.sh | sh
flyctl auth login
```

### 2. 创建 Fly 应用

```bash
flyctl apps create your-app-name --org your-org
```

### 3. 设置必要密钥

```bash
flyctl secrets set \
  SERVER_PORT="8080" \
  DATABASE_NAME="agentstore" \
  MONGODB_URI="mongodb+srv://..." \
  JWT_ACCESS_SECRET="$(openssl rand -hex 32)" \
  JWT_REFRESH_SECRET="$(openssl rand -hex 32)" \
  WEBHOOK_ENCRYPTION_KEY="$(openssl rand -hex 32)" \
  FRONTEND_URL="https://your-app-name.fly.dev" \
  APP_NAME="AgentStore" \
  FROM_EMAIL="noreply@yourdomain.com" \
  FROM_NAME="AgentStore"
```

### 4. 设置可选集成密钥

```bash
flyctl secrets set \
  OPENAI_API_KEY="sk-..." \
  OPENAI_BASE_URL="https://api.openai.com" \
  OPENAI_MODEL="gpt-4o" \
  RESEND_API_KEY="re_..." \
  STRIPE_SECRET_KEY="sk_live_..." \
  STRIPE_PUBLISHABLE_KEY="pk_live_..." \
  STRIPE_WEBHOOK_SECRET="whsec_..." \
  GOOGLE_CLIENT_ID="..." \
  GOOGLE_CLIENT_SECRET="..." \
  GOOGLE_REDIRECT_URL="https://your-app-name.fly.dev/api/auth/google/callback"
```

### 5. 部署

```bash
flyctl deploy
```

### 6. 验证健康状态

```bash
curl -fsS https://your-app-name.fly.dev/health
```

返回 HTTP 200 即成功。

---

## 首次初始化

部署完成后，浏览器打开应用 URL，安装向导将自动出现（系统未初始化时）。

**CLI 方式（可从本机访问 MongoDB 时）：**

```bash
set -a && source .env.production && set +a
export AGENTSTORE_ENV=prod
cd backend
go run ./cmd/agentstore setup
```

---

## 必要环境变量

| 变量 | 必填 | 说明 |
|------|------|------|
| `SERVER_PORT` | 是 | Fly.io 用 `8080` |
| `DATABASE_NAME` | 是 | MongoDB 数据库名 |
| `MONGODB_URI` | 是 | MongoDB 连接字符串 |
| `JWT_ACCESS_SECRET` | 是 | `openssl rand -hex 32` |
| `JWT_REFRESH_SECRET` | 是 | `openssl rand -hex 32` |
| `WEBHOOK_ENCRYPTION_KEY` | 是 | `openssl rand -hex 32` |
| `FRONTEND_URL` | 是 | 公开 HTTPS 地址 |
| `APP_NAME` | 是 | 应用名称 |
| `OPENAI_API_KEY` | 可选 | LLM provider key |
| `OPENAI_BASE_URL` | 可选 | LLM provider 地址 |
| `OPENAI_MODEL` | 可选 | 模型名称 |

三个 `OPENAI_*` 变量同时设置时，首次启动自动 seed LLM 配置。留空则在 admin 后台手动配置。

其余可选变量（OAuth / Stripe / 微信/支付宝 / DataDog）见 `.env.docker.example`。

---

## Stripe Webhook 配置

计费功能需要配置 Stripe Webhook 端点：

```
https://your-app.com/api/billing/webhook
```

订阅以下事件：
- `checkout.session.completed`
- `customer.subscription.created` / `updated` / `deleted`
- `invoice.payment_succeeded` / `invoice.payment_failed`
- `charge.refunded` / `charge.dispute.created`

将 Signing Secret 设置到 `STRIPE_WEBHOOK_SECRET`。
