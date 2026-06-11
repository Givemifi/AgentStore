# AgentStore 业务规则

> 动业务逻辑前必读。规则均基于真实代码;不确定处标注「待确认」。

## 角色与权限(RBAC)

成员角色定义于 `backend/internal/models/membership.go`,按权重排序:

- `owner`(权重 3)— 完全控制,含破坏性操作
- `admin`(权重 2)— 读写,无删除(待确认:具体边界以各 handler 实际检查为准)
- `user`(权重 1)— 受限/只读

权限以权重比较为基础。改动角色或权重会影响所有 handler 的访问判定,属高风险。

## 多租户与隔离

- 一切数据按租户作用域,通过请求头 `X-Tenant-ID` 解析(`middleware/tenant.go`)。
- 根租户 = 平台运营方;客户租户 = 每个客户/组织一个。
- 所有租户作用域 handler 必须按租户过滤查询。**不可误改**:任何跨租户读写都是隔离漏洞。

## 认证

- JWT 双 token:access + refresh。请求带 `Authorization: Bearer` + `X-Tenant-ID`。
- 前端 `api/client.ts` 在 401 时自动用 refresh token 刷新。
- 支持密码、OAuth、magic link、MFA/TOTP。
- 未配 Resend 时,验证 token 打印到后端控制台(dev 行为)。
- auth 端点有 rate limit(安全限流);触发返回 429,属正常保护行为。

## 用户能做什么

- 注册/登录、管理资料与安全设置(含 MFA)。
- 发现并对话市场 Agent(预置 6 个 demo agent)。
- 查看每次对话的积分消耗与剩余余额。
- 购买积分包 / 升级订阅(经 Stripe;或微信支付/支付宝购买积分包)。
- 管理团队成员与角色、查看会话历史与活动。
- **对话中**:文本输入、按住说话语音输入、上传图片(vision)、上传文档(PDF/Word/TXT,前端提取文字)。

## 系统自动做什么

- 按 `X-Tenant-ID` 解析并强制租户隔离。
- 对话前检查积分,成功后扣费。
- Stripe webhook 驱动订阅状态与购买积分注入。
- 显著系统事件经 `syslog.Logger` 记录(severity:critical/high/medium/low/debug)。
- 401 时前端自动刷新 token。

## 积分规则(高风险,不可误改)

代码位置:`backend/internal/credits/service.go`、`api/handlers/chat.go`。

- 两类积分:**订阅积分**(套餐每月分配)+ **购买积分**(积分包一次性,经 Stripe)。
- **对话扣费**:每次对话按 `agent.CreditCost`(即 `agent.TextCreditCost()`)扣费,reason 为 `agent_chat`。
- **扣费时序**:`CheckSufficientCredits` → 调用 LLM(流式)→ 成功后 `DeductCredits`。即**先验后扣**,LLM 失败不应扣费。
- **余额不足**:`CheckSufficientCredits` 为假时返回 HTTP 402(`Insufficient credits`)。
- `ErrInsufficientCredits` / `ErrBalanceChanged` 映射到对应 HTTP 状态(见 `mapCreditDeductionErrorStatus`)。
- Agent 成本模型 `AgentCreditCost` 含 `TextMessageCredits`/`ImageGenerationCredits`/`VideoGenerationCredits`;当前对话只用文本积分,图像/视频生成积分为 0(待确认:后续若启用需补扣费逻辑)。

## 订单 / 计费规则

代码位置:`backend/internal/stripe/`、`internal/wechat/`、`internal/alipay/`、`api/handlers/billing.go`、`api/handlers/payment_notify.go`。

- Stripe 负责 checkout、订阅、billing portal、税、退款、争议、优惠码。
- 积分包购买可经 Stripe(国际卡)或微信支付 H5/支付宝网页(国内)。**订阅套餐仅走 Stripe**。
- Dev 环境计费可选;不配对应渠道凭证不影响其余功能,前端也不显示该支付方式。

**微信支付/支付宝积分包购买流程:**
1. 前端 `POST /billing/checkout {bundleId, paymentMethod: "wechat_h5"|"alipay"}` → 后端在 `payment_orders` 插入 pending 订单,调渠道 API 返回跳转 URL 与 `outTradeNo`。
2. 前端跳转到支付页面(微信拉起 App;支付宝 PC/WAP)。
3. 渠道异步回调 `POST /billing/wechat/notify` 或 `POST /billing/alipay/notify`(无 JWT,经渠道签名验证)。
4. 后端验签 → 金额校验(误差 >1 分触发 `syslog.Critical` 并拒绝)→ 原子 `pending→completed`(幂等,重复通知跳过)→ 注入购买积分 → 写 `financial_transactions`。
5. 前端回跳后轮询 `GET /billing/payment/status?outTradeNo=xxx`,最多等 60 秒。

**不可误改:**
- 微信/支付宝回调验签(缺失或绕过是严重安全漏洞)。
- 金额校验:回调金额必须与 `payment_orders.amountCents` 一致。
- `pending→completed` 原子过渡:防止重复加积分。
- `payment_orders` 状态:pending / completed / failed;只有 pending 才处理回调。

## 多模态 / 语音规则(本轮新增)

代码位置:`frontend/src/utils/attachments.ts`、`hooks/useSpeechRecognition.ts`、`pages/app/ChatPage.tsx`、`backend/internal/api/handlers/chat.go`、`llm/openai.go`、`middleware/bodylimit.go`。

- **附件上限**:最多 4 个;图片 ≤5MB;文档 ≤2MB(前端常量 `MAX_ATTACHMENTS` / `MAX_IMAGE_BYTES` / `MAX_DOC_BYTES`)。
- **图片**:转 base64 data URL,按 OpenAI vision 格式发给 LLM;后端 `validImageAttachments` 过滤 `data:image/...;base64,` 前缀并截断到 4 个。
- **文档**:PDF/Word/TXT/MD 在**前端提取文字**后拼进消息(`【文档:name】` 块),不依赖模型视觉能力。
- **持久化**:DB `ChatMessage.Content` 只存纯文本,**图片不入库**(无 schema 变更)。
- **body 上限**:`/api/chat` 路径 30MB,其余 1MB(`bodylimit.go`)。
- **语音**:纯前端 Web Speech API,按住说话(push-to-talk),不走后端、不计费;不支持的浏览器隐藏麦克风按钮。

## 校验规则(不可误改)

模型结构体的 `validate` tag 与 `internal/db/schema.go` 的 MongoDB JSON Schema 必须等价。任何模型改动须同步两处并跑 `go test ./internal/validation/...`。

## 不能被后续误改的逻辑(汇总)

1. 租户隔离:所有查询按租户作用域,严禁跨租户。
2. 积分先验后扣、余额不足返回 402、LLM 失败不扣费。
3. Stripe webhook 幂等与金额计算;微信/支付宝回调验签、金额校验、pending→completed 幂等。
4. 模型校验双写同步(Go tag + JSON Schema)。
5. 多模态:图片不入库、`/api/chat` body 上限、附件数量/大小限制。
6. JWT + `X-Tenant-ID` 的认证/隔离链路。
