# AgentStore

[![CI](https://github.com/Givemifi/AgentStore/actions/workflows/ci.yml/badge.svg)](https://github.com/Givemifi/AgentStore/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/Givemifi/AgentStore/branch/master/graph/badge.svg)](https://codecov.io/gh/Givemifi/AgentStore)
[![Go Report Card](https://goreportcard.com/badge/github.com/Givemifi/AgentStore)](https://goreportcard.com/report/github.com/Givemifi/AgentStore)

**语言 / Language:** [English](README.md) | 简体中文 | [日本語](README.ja.md)

**面向企业的 AI Agent 市场，基于开源 SaaS 底座构建。**

AgentStore 帮助运营者快速搭建精选 AI Agent 市场：用户可发现 Agent、与之对话、了解积分用量、购买积分，无需关心底层平台细节。项目内置生产级 SaaS 底座：多租户账号管理、身份认证、基于角色的访问控制、白标品牌定制、Stripe 计费、API Key、出站 Webhook、管理后台、系统健康监控、基于积分的用量追踪，以及产品分析与遥测。

对于开发者，AgentStore 是一个可 Fork 的 SaaS/AI 平台底座，通过 [Claude Code](https://claude.ai/claude-code) 对话式开发完成。它让你从一个可用的市场出发，通过智能体工程持续演进，而非从零搭建 SaaS 基础设施。

---

## 为什么要做 AgentStore

每个 SaaS 产品都需要同样繁琐的基础设施：用户账号、团队、角色、认证、管理后台、计费、用量限制、品牌、Webhook、API Key。历史上，搭建这些基础设施往往需要数周，才能写一行真正的产品代码。

AgentStore 消灭了这个问题。Fork 项目，让 AI Agent 接管它，在一个已经处理好以下所有事项的底座上开始构建你的产品：

- 多租户隔离 + 基于角色的访问控制
- JWT 认证（含 refresh token 轮换）
- Google、GitHub、Microsoft OAuth 集成
- 魔法链接（无密码）认证
- MFA/TOTP + 恢复码
- 邮箱验证与密码重置
- 团队邀请与成员管理
- Stripe 计费（订阅、按席定价、试用、积分包）
- 套餐权益与计费拦截中间件
- 白标品牌（自定义主题、Logo、落地页、自定义页面）
- API Key 认证（管理员 & 用户两级作用域）
- 出站 Webhook（19 种事件类型，HMAC-SHA256 签名）
- 积分用量追踪（订阅积分 + 购买积分两桶）
- 优惠码与 Stripe 优惠券管理
- 产品分析看板（转化漏斗、KPI、留存队列、互动指标）
- 遥测事件系统（Go SDK + REST API 自定义事件上报）
- 完整管理界面（管理一切）
- 内置 API 文档（HTML & Markdown）
- 实时系统健康监控
- 财务指标看板（营收、ARR、DAU、MAU）
- MCP（Model Context Protocol）服务器，支持 AI 驱动的管理员操作
- CLI 工具（服务器运维）
- 自动版本管理与数据库迁移
- 支持 Fly.io 生产部署

---

## 功能特性

### 认证
- 密码 + 邮箱验证
- Google / GitHub / Microsoft OAuth
- 魔法链接（无密码）
- MFA/TOTP（含恢复码）
- Passkeys（WebAuthn）
- 会话管理（多设备登出）

### Agent 市场
- 23 个预置精选 Agent（法律、金融、营销、客服、代码、写作、翻译、学习等 7 大类）
- SSE 流式对话（Markdown 渲染、代码块复制）
- 多模态：语音（按住说话）、图片（Vision）、文档（PDF / Word / TXT）
- 公开 Agent 目录（`/agents`）：无需登录即可浏览
- 可分享对话链接

### 计费与积分
- 订阅积分（订阅套餐按月发放）
- 购买积分包（一次性买断）
- 对话按 Agent 成本扣费
- 余额不足实时拦截（402）
- Stripe Checkout / 订阅 / Customer Portal
- WeChat Pay H5 + 支付宝网页支付（积分包）
- 优惠码与 Stripe Coupon

### 管理后台
- 用户 / 租户 / 成员 / 邀请管理
- 套餐与积分包配置
- 计费与发票
- 品牌白标（应用名、配色、Logo、自定义页面）
- 系统健康与指标
- 结构化系统日志（分级：critical / high / medium / low / debug）
- API Key 管理（admin & user 两级）
- 出站 Webhook（19 种事件，AES-256-GCM 加密载荷）
- 产品分析与遥测看板
- 上线就绪检查清单
- LLM 配置（运行时可换 provider）
- 推荐码系统 + 注册送积分（可配置）

---

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.25, gorilla/mux |
| 前端 | React 19, TypeScript, Vite 7, Tailwind CSS 4 |
| 数据库 | MongoDB（Atlas 或本地） |
| 认证 | JWT, bcrypt, OAuth, Magic Link, TOTP MFA |
| 计费 | Stripe (stripe-go v82) |
| 邮件 | Resend |
| 图表 | Recharts |
| 指标 | gopsutil v4 |
| 部署 | Docker, Fly.io |

---

## 快速开始

### 🚀 一键部署（Docker Compose — 推荐）

最快方式运行 AgentStore，无需安装 Go 或 Node.js。

**前提：** 安装 [Docker Desktop](https://docs.docker.com/get-docker/)（含 Compose）

```bash
git clone https://github.com/Givemifi/AgentStore.git
cd AgentStore

# 1. 复制 Docker 环境变量模板
cp .env.docker.example .env

# 2. 生成必需密钥（分别运行，将输出填入 .env 对应字段）
openssl rand -hex 32   # → JWT_ACCESS_SECRET
openssl rand -hex 32   # → JWT_REFRESH_SECRET
openssl rand -hex 32   # → WEBHOOK_ENCRYPTION_KEY

# 3. 可选：在 .env 中填入 OPENAI_API_KEY / OPENAI_BASE_URL / OPENAI_MODEL
#    支持任何 OpenAI 兼容端点，填写后部署即可对话。
#    留空则稍后在 admin 后台配置。

# 4. 启动（自动构建镜像 + 启动 MongoDB）
docker compose up -d

# 5. 浏览器打开 http://localhost:8080，按向导创建管理员账户
```

MongoDB 数据持久化在 Docker named volume（`mongo_data`）中。  
`docker compose down -v` 可完整清除所有数据。

> **全自动初始化？** 在 `.env` 中设置 `AGENTSTORE_SETUP_ORG`、`AGENTSTORE_SETUP_NAME`、`AGENTSTORE_SETUP_EMAIL`、`AGENTSTORE_SETUP_PASSWORD`，首次启动时将自动创建管理员账户，无需浏览器向导。

### 开发模式（从源码运行）

**前提：** Go 1.25+、Node.js 22+、MongoDB、Git

```bash
git clone https://github.com/Givemifi/AgentStore.git
cd AgentStore

# 初始化配置
./scripts/setup.sh

# 启动后端（终端 1）
set -a && source .env && set +a
cd backend && go run ./cmd/server

# 启动前端（终端 2）
set -a && source .env && set +a
cd frontend && npm install && npm run dev

# 初始化系统（首次运行）
cd backend && go run ./cmd/agentstore setup
```

后端运行在 `http://localhost:4290`，前端运行在 `http://localhost:4280`。

---

## 部署到生产

### Fly.io

```bash
fly auth login
fly apps create agentstore

# 设置必要密钥
fly secrets set \
  JWT_ACCESS_SECRET=$(openssl rand -hex 32) \
  JWT_REFRESH_SECRET=$(openssl rand -hex 32) \
  WEBHOOK_ENCRYPTION_KEY=$(openssl rand -hex 32) \
  MONGODB_URI="your-mongodb-atlas-uri" \
  DATABASE_NAME=agentstore \
  APP_NAME=AgentStore \
  FROM_EMAIL=noreply@yourdomain.com \
  FRONTEND_URL=https://your-app.fly.dev

fly deploy
```

### 任意 Docker 主机

```bash
# 构建镜像
docker build -t agentstore .

# 或使用预构建镜像
docker pull ghcr.io/givemifi/agentstore:latest

# 运行（需提供外部 MongoDB）
docker run -d \
  -p 8080:8080 \
  --env-file .env \
  -e MONGODB_URI="your-mongodb-uri" \
  agentstore
```

完整部署文档见 [docs/DEPLOYMENT.zh-CN.md](docs/DEPLOYMENT.zh-CN.md)。

---

## 开发命令

```bash
# 后端
cd backend && go build ./...          # 编译
cd backend && go vet ./...            # 静态分析
cd backend && go test ./...           # 运行测试
cd backend && go run ./cmd/server     # 启动服务

# 前端
cd frontend && npm run dev            # 开发服务器
cd frontend && npm run build          # 生产构建
cd frontend && npm run lint           # ESLint 检查
cd frontend && npm test               # 运行测试
cd frontend && npx tsc --noEmit       # TypeScript 类型检查
```

---

## 项目结构

```
backend/
  cmd/server/          — HTTP 服务器入口
  cmd/agentstore/      — CLI 工具（setup、change-password 等）
  internal/
    api/handlers/      — HTTP handlers（auth / admin / billing / chat …）
    models/            — MongoDB 数据模型
    db/                — 数据库连接、索引、JSON Schema 校验器
    credits/           — 积分系统
    stripe/            — Stripe 计费集成
    llm/               — LLM 客户端（OpenAI 兼容）
    middleware/        — 认证、租户解析、RBAC、计费拦截
    bootstrap/         — 首次初始化逻辑

frontend/
  src/
    pages/
      app/             — 用户端页面（Chat、Settings …）
      admin/           — 管理后台页面
      auth/            — 认证页面（Login、Register …）
      public/          — 公开页面（Agents 目录、分享链接 …）
    api/client.ts      — API 客户端 + Token 自动刷新
    i18n/              — 多语言（中 / 英）
```

---

## License

MIT License — Copyright (c) 2026 Givemifi

详见 [LICENSE](LICENSE)。
