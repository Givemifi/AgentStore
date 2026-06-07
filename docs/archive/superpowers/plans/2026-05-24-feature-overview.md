# AgentStore 功能概览

> 最后更新: 2026-05-24

## 当前架构

```
┌─────────────────────────────────────────────────────────┐
│                    Frontend (React)                     │
│  http://localhost:4280                                  │
├─────────────────────────────────────────────────────────┤
│  - Dashboard (Agent Marketplace)                        │
│  - Chat (对话界面)                                       │
│  - Settings (设置)                                       │
│    ├── Profile / Security / Sessions / Billing          │
│    ├── Agents (Agent管理) ← 新增                         │
│    └── Models (Model配置) ← 新增                         │
│  - Admin (/admin)                                        │
├─────────────────────────────────────────────────────────┤
│                    Backend (Go + MongoDB)               │
│  API服务, 认证, 数据库                                    │
└─────────────────────────────────────────────────────────┘
```

## 已实现的核心功能

### 1. 用户认证 (Auth)
- [x] 邮箱/密码注册登录
- [x] OAuth (Google, GitHub, Microsoft)
- [x] Magic Link 免密登录
- [x] MFA/TOTP 双因素认证
- [x] JWT Token (Access + Refresh)
- [x] Session 管理

### 2. 多租户 (Multi-Tenant)
- [x] 团队/租户创建与管理
- [x] 角色权限 (Owner/Admin/User)
- [x] 成员邀请
- [x] 租户切换

### 3. 计费与积分 (Billing & Credits)
- [x] Stripe 订阅
- [x] 按席位计费
- [x] Credit 积分系统
- [x] 购买积分包
- [x] 使用量追踪

### 4. Agent 功能 (新增)
- [x] Agent 列表 (Marketplace)
- [x] Agent 创建/编辑/发布/归档
- [x] Agent 配置 (Name, System Prompt, Capabilities)
- [x] Credit 费用设置
- [ ] Streaming Chat (实现中)
- [ ] Image/Video Generation (Coming Soon)

### 5. Model 配置 (新增)
- [x] Provider 管理 (OpenAI Compatible, Anthropic, Gemini)
- [x] Model 配置 (Text/Image/Video)
- [x] API Key 托管与屏蔽
- [ ] Test Connection
- [ ] Default Model 设置

### 6. 管理后台 (/admin)
- [x] Dashboard
- [x] 用户/租户管理
- [x] 消息管理
- [x] 日志查看
- [x] 系统健康监控
- [x] 财务仪表盘
- [x] Branding 配置

### 7. Webhooks & API
- [x] 19种事件类型
- [x] HMAC-SHA256 签名
- [x] API Keys 管理

### 8. Branding (白标)
- [x] 自定义 Logo/颜色
- [x] 自定义 Landing Page
- [x] 自定义 CSS/HTML

## 正在进行的工作

| 功能 | 状态 | 说明 |
|-----|------|-----|
| Agent Streaming | 50% | 后端已实现 SSE，前端 UI 进行中 |
| Image Generation | 0% | 后端占位 API，前端 Coming Soon |
| Video Generation | 0% | 后端占位 API，前端 Coming Soon |
| Model Router 完善 | 80% | Fallback 逻辑完成 |
| Admin 路由迁移 | 90% | /last → /admin 完成 |

## 待完成

- [ ] 完整的 Agent 管理 UI (CRUD + 表单验证)
- [ ] Model Settings 完整 UI
- [ ] Streaming Chat 前端
- [ ] 测试覆盖

## 技术栈

- **后端**: Go 1.25, Gorilla Mux, MongoDB Driver
- **前端**: React 19, Vite, TypeScript, TanStack Query, Tailwind
- **验证**: go-playground/validator + MongoDB JSON Schema
- **监控**: DataDog 集成 (可选)
- **部署**: Fly.io (Docker)

## 路由结构

```
/                           → Landing Page
/login, /signup            → 认证页面
/dashboard                 → Agent 市场
/chat/:slug                → 对话页面
/settings                  → 设置 (Profile, Security, Billing)
/settings/agents           → Agent 管理
/settings/models           → Model 配置
/admin                     → 管理后台
```