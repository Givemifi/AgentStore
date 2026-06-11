# AI 改动日志(CHANGELOG_AI)

> 每次 AI 改动追加一条:日期、做了什么、是否改业务代码、风险、下一步。最新在上。

## 2026-06-11 — 全球化完善第二批(M4a/M4b/M4c/M5)

**本次做了什么**

**M4a — 注册送积分 configstore 化**
- `auth.go:1955` 新增 `trialCredits()` 方法读取 `growth.trial_credits` configstore key（默认 25）。
- `configstore/seed.go` 加 `growth.trial_credits` 系统变量种子（String 类型，默认 "25"）。

**M4b — 对话分享**
- 新建 `models/share_link.go`：`ShareLink` + `SharedMessage` 快照模型。
- `db/mongodb.go` 加 `ShareLinks()` 集合方法 + `share_links` 唯一索引（token）。
- `db/schema.go` 加 `shareLinksSchema()` 并注册到 `AllSchemas()`。
- 新建 `handlers/share.go`：`CreateShare`、`RevokeShare`、`ListMyShares`、`GetPublicShare` 四个 handler。
- `main.go` 注册路由：`POST /chat/conversations/:id/share`、`GET/DELETE /chat/share/:token`、`GET /api/public/share/:token`。
- 前端：`shareApi`（create/list/revoke/getPublic）加入 `client.ts`；ChatPage 加分享按钮（hover 复制链接，2.5s 回退）；新建 `pages/public/SharePage.tsx`（`/share/:token`）。
- `App.tsx` 注册 `/share/:token` 路由。

**M4c — 推荐码系统**
- `models/user.go` 加 `ReferralCode`、`ReferredBy`、`ReferralRewardedAt` 字段。
- `db/schema.go` users schema 加 `referralCode`/`referredBy` 属性。
- `db/mongodb.go` users 索引加 `referralCode` 唯一稀疏索引。
- `RegisterRequest` 加 `RefCode` 字段；注册时调用 `generateReferralCode()` 生成 8 字符推荐码；如 `RefCode` 有效则写入 `ReferredBy`。
- `VerifyEmail` 完成后 goroutine 调用 `grantReferralRewards()`（幂等，`referralRewardedAt` 字段防重）。
- `credits/service.go` 加 `GrantPurchasedCredits(ctx, tenantID, amount, reason)` 方法。
- `AuthHandler` 加 `creditsSvc` 接口字段 + `SetCreditsSvc()` setter；`main.go` 调用 `authHandler.SetCreditsSvc(credits.NewService(database))`。
- `configstore/seed.go` 加 `growth.referral_reward_referee` / `growth.referral_reward_referrer`（默认 0）。
- 前端：`SignupPage` 读取 `?ref=` 参数并传给 register API；`authApi.register` 加 `refCode?` 参数；`types/index.ts` User 加 `referralCode`/`referredBy`；Settings → ProfileTab 加「邀请好友」区块（推荐链接 + 推荐码展示 + 一键复制）。

**M5 — 信任合规**
- 新建 `components/PublicFooter.tsx`（隐私/条款/联系入口），加到 `AgentsMarketPage`。
- GDPR 导出（`GET /auth/export-data`）和删号（`POST /auth/delete-account`）后端已存在、前端 ProfileTab 已有 UI，本轮无需新增。

**是否修改业务代码**：是。涉及 `auth.go`（高风险路径，注册/验证邮件流程）、`credits/service.go`（高风险路径，新增授信方法）、`models/user.go`（模型字段）、`db/schema.go + mongodb.go`（索引/schema）。**不触碰** stripe/llm/billing.go/chat.go/tenant.go。

**当前风险**
- `grantReferralRewards` 在 goroutine 中执行（fire-and-forget），若进程重启可能丢失（低概率，因邮件验证通常秒级完成）。幂等机制已有保障。
- 推荐奖励默认为 0，需 admin 手动配置才生效。
- 分享快照存 messages 内容，大型对话可能产生较大文档（当前无消息数量上限，可后续加）。

**下一步建议**
1. Admin → configstore 配置 `growth.trial_credits` 和推荐奖励数值。
2. Admin → 品牌页面 seed 隐私政策和服务条款内容到 `/p/privacy-policy` 和 `/p/terms-of-service`。
3. 配好视觉模型端点，端到端验证图片多模态对话。



## 2026-06-11 — 全球化完善第一批(M1/M2/M3/M6)

**本次做了什么**

前端：
- 新增 `react-i18next` + `i18next-browser-languagedetector` 依赖，建立 i18n 基座（`frontend/src/i18n/`，3 namespace：common/auth/app，英/中双语）。
- 更新全部 auth 页（LoginPage、SignupPage、ForgotPasswordPage、ResetPasswordPage、VerifyEmailPage）使用 `useTranslation`。
- 更新核心 app 页（DashboardPage、OnboardingPage、SettingsPage）使用 `useTranslation`。
- `Layout.tsx` 加 `LanguageSwitcher` 组件（header 右上角），auth 页加全文切换按钮。
- `MarkdownMessage.tsx` 增强：代码块加复制按钮 + 语言标签、**bold**/*italic* inline 支持、`memo` 优化；`react-markdown/remark-gfm/rehype-highlight` 依赖已装但保留自定义渲染（更好控制暗色主题样式）。
- `ChatPage.tsx` 加 `useTranslation`，修复硬编码中文字符串（附件 alt、图片数量）。
- 新建 `frontend/src/pages/public/AgentsMarketPage.tsx`（路由 `/agents`）：无需登录的完整市场页，支持搜索+分类筛选。
- 新建 `frontend/src/pages/public/AgentDetailPage.tsx`（路由 `/agents/:slug`）：agent 详情 + CTA + 登录/注册回跳。
- `App.tsx` 注册 `/agents`、`/agents/:slug` 公开路由。
- `SignupPage` 支持 `?redirect=` 回跳参数。
- `frontend/src/api/client.ts` 加 `publicApi`（无认证 axios 实例）+ `publicAgentsApi`（list/get）。
- 测试 setup（`src/test/setup.ts`）加 i18n 初始化，确保 166 个测试全部通过。

后端：
- `backend/internal/agents/catalog.go`：Agent struct 加 `WelcomeMessage`/`SuggestedPrompts` 字段；从 6 扩到 24 个精品 agent（7 大类），所有 system prompt 加「Respond in the language the user writes in」；`cloneAgent` 同步 SuggestedPrompts。
- `backend/internal/api/handlers/chat.go`：`toPublicAgent` 映射新字段。
- 新建 `backend/internal/api/handlers/public_agents.go`：`GET /api/public/agents`、`GET /api/public/agents/{slug}`（公开，5 分钟缓存）。
- `backend/cmd/server/main.go`：注册公开路由；`spaHandler` 扩展 `getMetaTags` 字段支持 OG meta 动态注入；新增 `/sitemap.xml`（首页 + 24 个 agent 详情页）和 `/robots.txt` handler；import `agentsInternal`。
- `frontend/index.html`：加 `{{META_TAGS}}` 占位符。

**是否修改业务代码**：是。`catalog.go`（新增字段和 agent）、`chat.go`（toPublicAgent）、`main.go`（新路由/handler）、`client.ts`（新 API）均有改动。未触碰 credits/stripe/billing/tenant/chat 核心逻辑。

**当前风险**
- 公开 Agent API 只暴露 `publicCatalogAgent` 字段（不含 SystemPrompt/ModelConfig），安全隔离满足。
- 静态 catalog 扩充只影响新部署/setup，已有 DB agent 优先。
- `{{META_TAGS}}` 占位符在开发模式(Vite dev server)不生效（需后端 serve index.html），仅生产 build 有效。

**下一步建议**
1. M4a：注册送积分 configstore 化（`auth.go:1955`）。
2. M4b：对话分享公开页（`share_links` collection + `/share/:token`）。
3. M4c：推荐码双向奖励。
4. M5：隐私/条款内容页 + Footer + GDPR 导出/删号。



## 2026-06-10 — 接入微信支付 H5 + 支付宝网页支付(积分包)

**本次做了什么**

后端:
- 新增 `internal/wechat/service.go`:微信 V3 客户端,封装 H5 下单、平台证书验签、订单查询。
- 新增 `internal/alipay/service.go`:支付宝客户端,UA 自动选 PC(`trade.page.pay`)/WAP(`trade.wap.pay`)、RSA2 验签、订单查询。
- 新增 `internal/api/handlers/payment_notify.go`:微信/支付宝异步回调 handler(`HandleWechatNotify` / `HandleAlipayNotify`)+ 前端轮询 `GetPaymentStatus`。
- 扩展 `internal/config/config.go`:新增 `WeChatPayConfig`、`AlipayConfig`。
- 扩展 `internal/models/billing.go`:新增 `PaymentOrder` 结构体;`FinancialTransaction` 加 `PaymentProvider`/`OutTradeNo` 字段。
- 扩展 `internal/db/schema.go` + `mongodb.go`:新增 `payment_orders` collection。
- 扩展 `internal/api/handlers/billing.go`:`Checkout` 支持 `paymentMethod` 参数;`GetConfig` 返回已启用渠道列表;新增 `createDomesticPayOrder` 内部方法。
- `cmd/server/main.go`:初始化两个 service,注册 3 条路由(`/billing/wechat/notify`、`/billing/alipay/notify`、`/billing/payment/status`)。
- `config/dev.example.yaml`、`config/prod.example.yaml`、`.env.example`:补充两套凭证配置项。

前端:
- `src/api/client.ts`:`checkout` 加 `paymentMethod` 参数;新增 `getPaymentStatus`;`getConfig` 返回 `paymentMethods`。
- `src/pages/app/BuyCreditsPage.tsx`:新增支付方式选择 UI(仅展示已配置渠道);国内支付显示 CNY 参考价。
- `src/pages/app/BillingSuccessPage.tsx`:国内支付回跳后轮询状态(最多 60 秒),区分等待中/成功/超时三个状态。
- `src/pages/app/BuyCreditsPage.test.tsx`:补充 `getConfig` mock,修正 `checkout` 调用断言。

依赖:
- 新增 `github.com/go-pay/gopay v1.5.118` 及其传递依赖(`go-pay/crypto`、`go-pay/errgroup` 等)。

**是否修改业务代码**:是。新增计费渠道,涉及高风险路径(`billing.go`、新增 handler、models、schema)。

**当前风险**
- 微信/支付宝真实凭证未配置,回调地址需公网可达,本轮无法完整端到端联调。支付宝可用沙箱(`ALIPAY_IS_SANDBOX=true`)先测。
- CNY 金额展示为参考值(默认 7.20 汇率),实际以支付时渠道汇率为准;汇率偏差超出用户预期时需更新 `billing.cny_per_usd` configstore 配置。
- 微信/支付宝均仅支持积分包,订阅套餐仍走 Stripe;不可在未经测试的情况下扩展到订阅场景。

**下一步**
1. 准备微信商户号/API 凭证或支付宝沙箱 AppID,配 ngrok 公网转发,做端到端联调测试。
2. 测试路径:选积分包 → 选微信/支付宝 → 跳转支付 → 完成 → 回调触发积分到账 → 前端确认状态。
3. 验证幂等:同一 `outTradeNo` 发两次回调,积分只加一次。

## 2026-06-10 — V1 工程归档(仅文档)

**本次做了什么**
- 重写 `CLAUDE.md`(启动须知,160 行内):项目说明、V1 已完成、启动/测试/构建命令、核心目录、必读文档、开发禁区、完成后要更新的文档。
- 新建 `docs/START_HERE.md`:/clear 后第一篇,含一句话说明、V1 状态、必读顺序、下一步、禁区、可复制提示词。
- 重写 `docs/ARCHITECTURE.md`:补充技术栈版本、数据流、关键文件、外部依赖,纳入本轮多模态/语音模块。
- 新建 `docs/BUSINESS_RULES.md`:角色权限、租户隔离、认证、积分/计费规则、多模态/语音规则、不可误改逻辑汇总。
- 新建 `docs/TASKS.md`:V1 已完成、遗留问题、下一步优先、暂缓、不建议现在做。
- 新建 `docs/DECISIONS.md`:已定型的产品/技术/多模态/部署/流程决策与原因。
- 新建本文件 `docs/CHANGELOG_AI.md`。

**是否修改业务代码**:否。本次仅新增/修改文档,未触碰任何 Go/TS 业务代码、配置或部署文件。

**当前风险**
- 多模态图片真实可用性未端到端确认:上游网关曾路由到无视觉能力的模型(`gemini-2.5-flash`),回"无法查看图片"。属网关/模型侧配置,非代码问题。文档上传不受影响。
- RBAC 精确权限边界文档层面标注"待确认",尚未形成权限矩阵。
- 多模态/语音缺少前端自动化测试覆盖。

**下一步建议**
1. 在 Admin → LLM 配置启用确实支持视觉的模型端点,端到端验证图片对话。
2. 真机回归语音/附件/键盘适配(iOS Safari、Android Chrome)。
3. 梳理 RBAC 权限矩阵,补充多模态前端测试。

---

## 此前(摘要,详见 git 历史与 docs/TASKS.md)

- 实现 Agent 对话语音按住说话 + 图片/文档多模态上传 + 手机端适配(前后端 6 文件 + index.html),构建/测试通过,端到端管线验证。
- V1 多租户 SaaS:认证、Agent 市场、积分、Stripe 计费、admin 后台等已完成。
