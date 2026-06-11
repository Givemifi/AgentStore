# AI 改动交接日志(CHANGELOG_AI)

> 每次 AI 改动追加一条,保留交接轨迹。最新在上。

## 2026-06-12 — 知识库(RAG)+ 数据标注闭环 + 基础缺口修复

让垂直 Agent 可持续进化:新增知识库与数据标注全闭环,并补齐两个阻碍进化的基础缺口(token 用量记录、对话历史截断)。

**后端**
- `llm/openai.go`:新增 `EmbedWithConfig`(OpenAI 兼容 embeddings,批量 ≤64);`CompleteWithUsage` / `CompleteStreamWithUsage` 返回 `CompletionResult{Content,Model,Usage}`;流式加 `stream_options.include_usage`;provider 未上报 usage 时按字符估算。旧 `CompleteWithConfig`/`CompleteStreamWithConfig` 保留为 3 返回值的薄封装。
- `llm/router.go`:新增 `ResolveEmbeddingModel`(租户默认 → `OPENAI_EMBEDDING_MODEL` 环境变量 + legacy 凭证)。
- 新包 `internal/knowledge/`:`Chunk` 分块(~1600 字符、200 重叠)、`IngestDocument` 异步嵌入入库、`Retrieve` 进程内余弦相似度 + 5 分钟 TTL 缓存、`BuildKnowledgeBlock`。含单测。
- 4 个新集合(model + `db/schema.go` JSON Schema + `mongodb.go` 访问器/索引 + `validation` 测试):`knowledge_documents`、`knowledge_chunks`、`message_feedback`、`annotations`。
- 既有模型增量:`ChatMessage` +`promptTokens`/`completionTokens`;`Tenant` +`defaultEmbeddingModelConfigId`;`ModelModality` +`embedding`。
- handler:`knowledge.go`(文档增删/列表/重建索引,admin 写)、`feedback.go`(消息 👍/👎 upsert/删除/查询)、`annotations.go`(队列/上下文/标注/沉淀知识/导出 JSONL/质量统计,admin)。
- `chat.go`:对话注入检索知识(失败降级不阻断)、token 落库、`truncateHistory`(≤12000 token,留最近 2 条)。
- `middleware/bodylimit.go`:`/api/agents` 路径放宽到 5MB(知识文本上传)。
- `model_settings.go`:默认模型支持 embedding modality。

**前端**
- `api/client.ts`:新增 `feedbackApi` / `knowledgeApi` / `annotationsApi` + 类型;`updateDefaults` 支持 embedding。
- `ChatPage.tsx`:assistant 完成态消息加 👍/👎(乐观更新、可撤销)。
- `settings/KnowledgeModal.tsx`(新):某 agent 的知识库管理(上传文档复用 `utils/attachments.ts` 前端提取文字 / 粘贴文本 / 问答对 / 状态轮询 / 删除 / 重建索引)。
- `settings/AgentsTab.tsx`:每个 agent 加「知识库」入口。
- `settings/ModelSettingsTab.tsx`:embedding modality + 默认嵌入模型选择器。
- `admin/AnnotationsPage.tsx`(新):质量看板 + 标注队列(负反馈优先)+ 标注表单(评分/问题标签/理想回答)+ 沉淀知识 + 导出 JSONL。路由 `/admin/annotations`,AdminLayout 导航入口。

**配置/文档**
- `.env.example`:新增 `OPENAI_EMBEDDING_MODEL`。
- `docs/BUSINESS_RULES.md`、`docs/DECISIONS.md`、`docs/ARCHITECTURE.md` 同步知识库/标注/token/截断规则与决策。

**验证**:`go build/vet/test ./...` 全绿;前端 `tsc --noEmit` / `lint`(0 error)/ `vitest`(166 passed)全绿。
