# AgentStore Architecture

平台自营 AI Agent 市场,跑在生产级多租户 SaaS 底座上。用户发现 Agent、对话、查看积分消耗、购买积分、在租户隔离的工作区继续使用。运营方在 admin 后台管理 Agent、套餐、积分、计费、品牌、健康、遥测、日志、上线就绪。

## 技术栈

- **后端**:Go 1.25,`gorilla/mux`,MongoDB 官方 driver。入口 `backend/cmd/server`。
- **CLI / MCP**:`backend/cmd/agentstore` 提供初始化等管理命令与只读 MCP server。
- **前端**:React 19 + TypeScript + Vite 7 + Tailwind 4,`@tanstack/react-query`、`react-router-dom` 7、`react-hook-form` + `zod`、`recharts`、`sonner`、`dompurify`。
- **多模态依赖**:`pdfjs-dist`(PDF 提取)、`mammoth`(Word .docx 提取),均动态 import 走独立 chunk。
- **数据库**:MongoDB,存租户、用户、成员关系、套餐、积分、会话/消息、遥测、日志、webhook、配置、健康指标。
- **计费**:Stripe(checkout、订阅、portal、税、退款、争议、优惠码)+ **微信支付 V3**(H5 网页支付)+ **支付宝**(电脑网站/手机网站,UA 自动检测);均仅用于积分包购买,订阅走 Stripe。
- **LLM**:OpenAI 兼容 provider,经后端 LLM 服务路由,供 Agent 对话使用。

## 前后端结构

### 后端模块(`backend/internal/`)

- `api/handlers/` — HTTP handler:auth、bootstrap、admin、tenant、billing、chat、agent、knowledge、feedback、annotations、usage、branding、webhook、telemetry、health、logs、docs、promotions、model settings。
- `auth/` — JWT access/refresh、密码哈希、OAuth、magic link、MFA/TOTP、session。
- `middleware/` — 认证、租户解析、RBAC、计费拦截、metrics、recovery、API 版本、安全。
- `models/` — MongoDB 模型结构体(校验须与 JSON Schema 同步)。
- `db/` — 连接、集合、索引、JSON Schema(`schema.go` / `AllSchemas()`)。
- `credits/` — 订阅积分与购买积分核算(`service.go`)。
- `stripe/` — 客户、checkout、价格、订阅、portal、税、退款、争议、webhook。
- `wechat/` — 微信支付 V3 客户端:H5 下单、异步回调解析验签、订单查询。
- `alipay/` — 支付宝客户端:PC/WAP 下单(UA 自动选)、异步回调验签、订单查询。
- `llm/` — provider client、模型/请求路由、流式与非流式 chat、embeddings(`EmbedWithConfig`)、token usage(usage 上报 + 估算兜底)。
- `knowledge/` — 知识库 RAG:文档分块(`Chunk`)、嵌入入库(`IngestDocument`)、进程内余弦检索 + TTL 缓存(`Retrieve`)、注入块拼装(`BuildKnowledgeBlock`)。
- `agents/` — 市场 Agent 领域逻辑。
- `configstore/` — DB 后端的运行时配置。
- `planstore/` — 套餐与积分包种子数据。
- `events/`、`webhooks/` — 内部事件发射与外发 webhook。
- `telemetry/` — 产品分析事件、SDK helper、PM 看板查询。
- `health/`、`metrics/`、`datadog/` — 健康、指标采集、集成上报。
- `syslog/` — 带 severity 的系统日志与注入检测。
- `validation/`、`testutil/` — 校验 helper 与测试工具。

### 前端模块(`frontend/src/`)

- `App.tsx` — 顶层路由与 bootstrap guard。
- `api/client.ts` — API 客户端、token 刷新、按域分组的 typed endpoint。
- `contexts/` — auth、tenant、branding、theme provider。
- `components/`、`components/app/`、`components/ui/` — 布局、admin 布局、共享 UI、骨架屏、错误边界、主题注入、移动导航。
- `pages/public/` — 落地页与自定义公共页。
- `pages/auth/` — 登录、注册、MFA、magic link、密码重置、验证。
- `pages/app/` — 用户工作区:dashboard、chat、onboarding、credits、billing、team、activity、plan、settings。
- `pages/admin/` — admin 各页:dashboard、users、tenants、plans、billing、branding、health、logs、API key、webhook、promotions、telemetry、config、launch readiness。
- `types/`、`utils/`、`hooks/` — 共享类型、工具、hook。
- `e2e/` — Playwright 端到端测试。

## 数据流

1. **认证**:前端登录 → 后端签发 access+refresh JWT。后续请求带 `Authorization: Bearer` 与 `X-Tenant-ID` 头;`client.ts` 在 401 时自动刷新。
2. **租户隔离**:`middleware/tenant.go` 从头解析租户,后续 handler 全部按租户作用域查询。
3. **Agent 对话**:`ChatPage.tsx` → `chatApi.stream`(SSE)→ `api/handlers/chat.go` → 先 `CheckSufficientCredits` → 调 `llm/` provider(流式)→ 成功后 `DeductCredits`(按 `agent.CreditCost`,reason `agent_chat`)。余额不足返回 402。
4. **多模态**:前端 `utils/attachments.ts` 把图片转 base64 data URL、文档前端提取文字;`composeMessagePayload` 组装 → `chat.go` 用 `buildUserMessage` 走 OpenAI vision 多段 content 格式(`type:"text"` / `type:"image_url"`)发给 LLM。DB 只存纯文本,图片不持久化。
5. **计费**:Stripe checkout/webhook → `stripe/` + `billing.go` 更新订阅/积分;积分包购买注入购买积分。微信/支付宝流程:前端 `POST /billing/checkout {bundleId, paymentMethod}` → 后端建 `payment_orders`(状态 pending)→ 返回跳转 URL → 渠道异步回调 `POST /billing/wechat/notify|/billing/alipay/notify` → 验签 + 金额校验 → 原子 pending→completed → 注入积分 + 写 `financial_transactions`；前端轮询 `GET /billing/payment/status?outTradeNo=xxx` 确认到账。
6. **知识库 RAG**:admin 在 `settings/agents` 的知识库弹窗上传文档(前端提取文字)→ `POST /tenant/agents/{id}/knowledge` → `knowledge.IngestDocument` 异步分块 + 嵌入(`EmbedWithConfig`)入 `knowledge_chunks`。对话时 `chat.go` 调 `knowledge.Retrieve`(owner 租户 + agent,进程内余弦相似度)→ `BuildKnowledgeBlock` 拼到 systemPrompt 之后。检索失败降级为无知识对话。
7. **数据标注闭环**:用户在 ChatPage 对 assistant 消息 👍/👎(`PUT /chat/messages/{id}/feedback`)→ admin 在 `/admin/annotations` 审阅队列(负反馈优先)、打分/分类/写理想回答 → 「沉淀为知识」生成 annotation 来源的知识文档(即时嵌入生效)或导出 SFT JSONL(`GET /tenant/annotations/export`)。token 用量随每条 assistant 消息落库,供质量看板聚合。

## 新增集合(本轮)

- `knowledge_documents` — 知识文档元数据(状态 processing/ready/error、chunk 数、字符数、来源类型)。
- `knowledge_chunks` — 文档分块 + 嵌入向量(检索用)。
- `message_feedback` — 用户对 assistant 消息的 👍/👎 + 评论(唯一索引 messageId+userId)。
- `annotations` — admin 质量标注(评分、问题标签、理想回答、沉淀状态)。

`chat_messages` 增 `promptTokens`/`completionTokens`;`tenants` 增 `defaultEmbeddingModelConfigId`;`ModelModality` 增 `embedding`。

## 关键文件

- `backend/cmd/server/` — 服务入口
- `backend/internal/api/handlers/chat.go` — 对话 + 扣费 + 多模态组装
- `backend/internal/credits/service.go` — `CheckSufficientCredits` / `DeductCredits` / `ErrInsufficientCredits`
- `backend/internal/llm/openai.go` — provider client、`NewVisionMessage`、`contentToString`
- `backend/internal/middleware/tenant.go` — 租户解析
- `backend/internal/middleware/bodylimit.go` — `/api/chat` 路径 body 上限 30MB,其余 1MB
- `backend/internal/db/schema.go` — MongoDB JSON Schema(`AllSchemas()`)
- `frontend/src/pages/app/ChatPage.tsx` — 对话 UI、语音、附件、移动适配
- `frontend/src/utils/attachments.ts` — 附件处理与 payload 组装
- `frontend/src/hooks/useSpeechRecognition.ts` — Web Speech API 按住说话封装
- `frontend/src/api/client.ts` — API 客户端

## 外部依赖

- **MongoDB**(Atlas 或本地)— 主数据存储
- **Stripe** — 计费(可选,dev 可不配)
- **微信支付 V3** — 国内 H5 积分包支付(可选,不配则不显示);商户号 + APIv3 密钥 + 私钥 + 证书序列号
- **支付宝** — 国内网页积分包支付(可选,不配则不显示);AppID + RSA2 私钥/公钥
- **OpenAI 兼容 LLM provider** — 对话(本地默认 `https://ai.wuwutech.com`,模型 `MiniMax-M2.5`)
- **Resend** — 邮件(可选,未配则验证 token 打印到控制台)
- **OAuth provider**(Google 等)— 第三方登录(可选)
- **DataDog** — 指标集成(可选)
- **浏览器 Web Speech API** — 语音输入(纯前端,不走后端;不支持的浏览器自动隐藏麦克风按钮)

## 高风险路径(改动需谨慎并跑测试)

- 计费/资金:`api/handlers/billing.go`、`stripe/`、`wechat/`、`alipay/`、Stripe webhook、微信/支付宝回调、checkout 成功流、退款、争议、交易记录、`payment_orders` 状态流转。
- 积分:`credits/`、积分包 checkout、扣费、usage event、余额不足行为。
- 租户隔离:`middleware/tenant.go`、租户作用域 handler、API key scope、model settings、chat、agents、credits、billing。
- LLM 路由:`llm/`、模型/provider 设置、fallback、流式与非流式路径。
- 对话:`api/handlers/chat.go`、`frontend/src/pages/app/ChatPage.tsx`、markdown 渲染、流式状态、重试/中断、扣费、知识注入、历史截断、token 落库。
- 知识库:`knowledge/`、`api/handlers/knowledge.go`、嵌入入库与检索、检索失败降级。
- 上线就绪:`api/handlers/admin_launch_readiness.go` 与对应 UI。

## 校验与 Schema 规则

混合校验:(1) `internal/models/` 结构体 `validate` tag;(2) `internal/db/schema.go` 的 MongoDB JSON Schema。改模型须两者等价,然后:

```bash
cd backend && go test ./internal/validation/...
```

## 部署形态

默认生产构建用根目录 `Dockerfile`,把 Go 后端 + React 前端打进单容器,运行时复制 `backend/config/prod.example.yaml` 为 `config/prod.yaml`,秘密与环境值由环境变量在启动时展开;Go server 从 `/app/static` 提供 SPA,容器暴露 `8080`。必需生产环境值:MongoDB、JWT secrets、`SERVER_PORT=8080`、`FRONTEND_URL`、应用名/邮箱身份、`WEBHOOK_ENCRYPTION_KEY`。OAuth/Resend/Stripe/DataDog 为可选集成。

**依赖项目**(以本仓库为 submodule/fork/copy)必须用 `Dockerfile.saas` + `fly.saas.toml`,绝不裸跑 `fly deploy`(否则 auth 路由缺失、登录静默失败)。详见 `docs/DEPLOYMENT.md`。
