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
- **扣费时序**:`CheckSufficientCredits`(预检,仅用于快速拦截)→ **先扣费 `DeductCredits` → 再调用 LLM**。LLM 失败或客户端取消(流式)时调用 `RefundDeduction` 退款。即**先扣后用、失败退款**,杜绝"回答已发但因并发耗尽余额而未扣到钱"的免费请求。
- **扣减原子性与并发**:`DeductCredits` 不使用多文档事务(兼容单节点 MongoDB)。常见情形(单一余额桶即可覆盖)用一次带 `$gte` 条件的原子 `$inc`,无竞争;仅当需跨"订阅+购买"两桶拆分时回退到精确余额的乐观 CAS,并最多重试 `maxDeductAttempts` 次。任何并发下余额都不会变负,合法并发请求也不会被乐观锁误拒。`usage.go` 的 `RecordUsage` 与对话走同一 `credits.Service`,行为一致。
- **退款语义**:`RefundDeduction` 将积分退回 `purchasedCredits`(避免把订阅余额顶超套餐月额度),并写一条 `<type>_refund` 的补偿 `usage_event`,使流水净额归零。
- **余额不足**:`CheckSufficientCredits` 为假,或 `DeductCredits` 返回 `ErrInsufficientCredits` 时返回 HTTP 402(`Insufficient credits`);此时不调用 LLM、不落任何消息。
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

## 知识库 / RAG 规则(本轮新增)

代码位置:`backend/internal/knowledge/`、`api/handlers/knowledge.go`、`api/handlers/chat.go`、`llm/openai.go`(`EmbedWithConfig`)、`llm/router.go`(`ResolveEmbeddingModel`)。

- **归属与权限**:知识库按 `(tenantId, agentId)` 作用域。仅 **DB agent**(数据库里的 agent,非静态 catalog demo agent)支持知识库;写操作(增/删/重建索引)要求 `admin`,列表对租户成员可读。平台自营 agent 的知识由根租户 admin 管理。
- **文档来源**:沿用「前端提取文字」决策——PDF/Word/TXT 在浏览器用 pdfjs/mammoth 解析后,只把纯文本 `POST` 给后端;后端不引入文档解析依赖。也支持粘贴文本、问答对。
- **嵌入模型**:租户 `defaultEmbeddingModelConfigId`(modality=embedding)→ 回退到环境变量 `OPENAI_EMBEDDING_MODEL` + legacy LLMConfig 的 key/baseURL。**未配置嵌入模型时拒绝上传知识**(返回引导提示),但**不影响对话**。
- **上限**:每 agent 最多 20 个文档、1000 个 chunk;单次上传文本 ≤ 400k 字符。超限报错提示精简。
- **检索注入**:对话时以 owner 租户 + agent 检索 top 6 且余弦相似度 ≥ 0.35、注入文本 ≤ 8000 字符的 chunk,拼到 systemPrompt 之后。**检索失败一律降级为无知识对话,绝不阻断聊天**(仅 `slog.Warn`)。
- **计费**:知识检索与嵌入由**运营方承担成本**,不额外扣用户积分;对话仍按 `agent.CreditCost` 扣文本积分,规则不变。
- **向量存储**:chunk 向量直接存 MongoDB(`knowledge_chunks.embedding`),检索在 Go 进程内算余弦相似度(自托管 mongo:7 无 Atlas Vector Search)。进程内 5 分钟 TTL 缓存,文档增删/重建时主动失效。

## 反馈 / 数据标注规则(本轮新增)

代码位置:`api/handlers/feedback.go`、`api/handlers/annotations.go`、`frontend/src/pages/admin/AnnotationsPage.tsx`。

- **用户反馈**:用户对 **assistant 消息**点 👍/👎(+可选评论),每人每条消息最多一条(`message_feedback` 唯一索引 `messageId+userId`)。只能评价自己会话里的消息。
- **标注工作台(admin)**:租户 admin 审阅本租户的负反馈消息,打 1-5 质量分、问题分类标签、撰写「理想回答」。**admin 可读本租户内其他用户的会话上下文用于质量审计**,该读取写 `syslog.Medium` 审计日志。严格限本租户,不跨租户。
- **沉淀知识**:把标注的「理想回答」一键生成 `问:…\n答:…` 知识文档并立即嵌入(同步),agent 下次对话即生效。已沉淀的标注幂等(重复返回 409)。
- **导出训练集**:导出 SFT 格式 JSONL(每行 system+多轮 user/assistant,assistant 用理想回答优先,否则原回答且仅 score≥minScore),供未来微调。

## token 用量字段语义(本轮新增)

`ChatMessage.promptTokens` / `completionTokens` 记录 assistant 消息的 token 用量:优先取 provider 的 usage 上报(流式经 `stream_options.include_usage`),provider 未上报时按字符估算(ASCII/4 + 非 ASCII 逐字),故为**近似值**,用于成本/质量分析,不用于计费。

## 上下文窗口截断(本轮新增)

对话历史在送入 LLM 前按估算 token 总额 ≤ 12000 从最新往前截断(`truncateHistory`),至少保留最近 2 条消息。防止长对话 token 失控,被截断不报错。

## 不能被后续误改的逻辑(汇总)

1. 租户隔离:所有查询按租户作用域,严禁跨租户。
2. 积分先扣后用、失败退款、余额不足返回 402;扣减原子且并发不变负、不误拒(无多文档事务,兼容单节点 MongoDB)。
3. Stripe webhook 幂等与金额计算;微信/支付宝回调验签、金额校验、pending→completed 幂等。
4. 模型校验双写同步(Go tag + JSON Schema)。
5. 多模态:图片不入库、`/api/chat` body 上限、附件数量/大小限制。
6. JWT + `X-Tenant-ID` 的认证/隔离链路。
7. 知识库:按租户+agent 隔离、检索失败降级不阻断对话、不额外扣用户积分、嵌入未配置时拒绝上传但不影响聊天。
8. 数据标注:admin 仅可审阅本租户会话上下文(有 syslog 审计),严禁跨租户。
