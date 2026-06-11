# AgentStore 决策记录

> 已定型的技术/产品决策与原因。除非有充分新理由,别反复推翻。

## 产品决策

- **平台自营、策展式市场**:V1 不是第三方开发者市场,而是平台运营方自己策展、运营一批 Agent。简化首版的信任、质量与计费模型。
- **积分包优先变现**:首版主要靠购买积分包(credit pack)变现,订阅积分作为套餐附带分配。降低用户付费门槛、契合按次消耗的对话场景。
- **国内支付支持微信/支付宝(仅积分包)**:订阅套餐仍走 Stripe;积分包购买额外支持微信 H5、支付宝(PC/WAP)。CNY 金额由后端按 `billing.cny_per_usd` 配置换算(默认 7.20),仅展示用,不影响 USD 原始定价。支付方式仅在对应凭证配置时才在前端显示。
- **预置 6 个 demo Agent**:首版以演示/运营场景为主,Agent 由运营方在 admin 管理。

## 技术决策

- **多租户 + `X-Tenant-ID` 头隔离**:所有数据按租户作用域,中间件统一解析。隔离是安全基线,不可绕过。
- **JWT access + refresh,前端自动刷新**:无状态认证 + 401 透明刷新,避免会话服务端依赖。
- **混合校验(Go validate tag + MongoDB JSON Schema)**:双层防御,DB 层兜底非法写入。代价是改模型须双写同步——这是有意取舍,不要为省事去掉任一层。
- **对话先验后扣积分**:`CheckSufficientCredits` → 调 LLM → 成功 `DeductCredits`。保证 LLM 失败不扣费、余额不足提前 402。
- **OpenAI 兼容 provider 抽象**:LLM 经 `llm/` 路由,可换任意 OpenAI 兼容端点(base URL + key + model 运行时可配)。本地默认 `https://ai.wuwutech.com`(注意:传 base URL 不带 `/v1`,由 `normalizeBaseURL` 自动追加)。

## 多模态 / 语音决策(本轮)

- **语音用浏览器原生 Web Speech API**:零成本、不走后端、不计费,按住说话(push-to-talk)。代价是浏览器兼容性差异——不支持则隐藏按钮,可接受。
- **文档在前端提取文字**:PDF(pdfjs)、Word(mammoth)、TXT 全部前端解析后拼进消息文本,**不依赖模型视觉能力**,任何模型都能用。pdfjs/mammoth 动态 import 走独立 chunk,不拖累首屏。
- **图片走 OpenAI vision 格式**:base64 data URL + 多段 content(`type:"text"`/`type:"image_url"`)。
- **图片不入库**:DB `ChatMessage.Content` 只存纯文本。避免 base64 撑爆存储,也回避了 schema 变更。
- **`/api/chat` body 上限单独放宽到 30MB**:仅该路径,其余仍 1MB,平衡多模态需求与安全。
- **附件硬上限(4 个 / 图 5MB / 文档 2MB)**:前端先校验,后端再过滤截断,双重保护。

## 国内支付决策(本轮)

- **微信用 V3 API(REST/JSON)**:非 V2(XML),更现代、官方推荐。使用 `github.com/go-pay/gopay` 库。
- **回调通知走异步验签 + 金额校验**:微信用平台证书(`WxPublicKeyMap`),支付宝用 RSA2(`VerifySign`);金额误差 >1 分触发 `syslog.Critical` 并拒绝——不验签等于无安全边界。
- **订单状态用独立 `payment_orders` 集合**:`FinancialTransaction` 语义保持"仅在支付成功时写入",与 Stripe 一致。`payment_orders` 跟踪 pending→completed 状态,幂等防止重复加积分。
- **前端轮询(非 WebSocket/SSE)**:支付回跳后前端每 2 秒轮询 `GET /billing/payment/status`,最多 30 次(60 秒)。简单可靠,不引入长连接。
- **支付宝 UA 自动检测**:后端读 `User-Agent` 决定用 `trade.page.pay`(PC)还是 `trade.wap.pay`(WAP),前端无需手动选择。

- **依赖项目必须用 `Dockerfile.saas` + `fly.saas.toml`**:多进程(产品后端 + AgentStore 后端,经 Caddy/supervisord)。裸跑 `fly deploy` 会丢 auth 路由导致登录静默失败。本仓库本体用根目录 `Dockerfile`。

## 知识库 / 数据标注决策(本轮)

- **垂直 Agent 持续进化的两条腿**:知识库(RAG,即时改善回答)+ 数据标注闭环(反馈→标注→沉淀知识/导出 SFT)。先靠知识注入快速见效,再靠标注数据沉淀长期资产,避免一上来就依赖微调。
- **向量存 MongoDB + Go 内存余弦**:部署目标是自托管 `mongo:7`(docker-compose),无 Atlas Vector Search。chunk 向量存 `knowledge_chunks.embedding`,检索时把某 agent 全部 chunk 载入进程内算余弦相似度,5 分钟 TTL 缓存。规模依据:自营市场每 agent 几百~一千 chunk,内存余弦足够快。`knowledge.Service` 通过 `embedder`/`modelResolver` 接口抽象,未来可无痛替换为专业向量库(Qdrant 等)。
- **知识文档前端提取文字**:复用既有多模态决策(pdfjs/mammoth 前端解析),后端不引入文档解析依赖,只接收纯文本 + 嵌入。
- **嵌入模型独立 modality**:`ModelModality` 新增 `embedding`,经 `router.ResolveEmbeddingModel` 走「租户默认 → 环境变量 `OPENAI_EMBEDDING_MODEL`」回退链,与对话文本模型解耦。
- **检索降级不阻断**:任何检索/嵌入失败只降级为无知识对话并 `slog.Warn`,绝不让知识层拖垮聊天主链路。
- **token usage 取数 + 估算兜底**:流式请求加 `stream_options.include_usage` 取 provider 上报;未上报时用字符估算(ASCII/4 + 非 ASCII 逐字)。记录进 `ChatMessage`,为成本/质量分析打基础。早先「无 token 记录」是阻碍进化的盲区,本轮补上。
- **历史截断**:对话历史按估算 token ≤ 12000 截断、至少留最近 2 条。配合知识注入,避免长对话上下文爆炸。
- **SFT 导出格式**:标注导出为 OpenAI chat 微调风格 JSONL(system + 多轮 + assistant 理想回答 + meta),为未来微调留口子,但本轮不做训练。

## 流程决策

- **不自动 git commit**:除非用户明确要求。
- **归档/清理不碰业务代码**:先审计,只删证明无用的。
- **每次 AI 改动追加 `docs/CHANGELOG_AI.md`**:保留交接轨迹。
