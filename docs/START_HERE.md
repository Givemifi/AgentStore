# START HERE — /clear 之后先读这里

## 一句话

AgentStore 是一个平台自营的 AI Agent 市场,跑在多租户 SaaS 底座上:用户发现并对话 Agent、按积分计费、买积分包;运营方在 admin 后台管理一切。Go 后端 + React 前端 + MongoDB + Stripe + OpenAI 兼容 LLM。

## 当前 V1 状态

V1 功能已完成并可本地端到端运行(后端 :4290 / 前端 :4280 / MongoDB)。最近一轮新增:Agent 对话的**语音按住说话 + 多模态图片/文档上传 + 手机端适配**,前后端构建与测试通过。详见 `docs/CHANGELOG_AI.md`。

> 本次只做了文档归档,**没有改动业务代码**。

## 继续开发前必须读

1. `CLAUDE.md` — 启动须知、命令、开发禁区(最重要)
2. 本文件 — 当前状态与下一步
3. `docs/TASKS.md` — 待办与遗留问题(决定做什么)
4. `docs/BUSINESS_RULES.md` — 动业务逻辑前必读,避免误改
5. `docs/ARCHITECTURE.md` — 需要定位代码时
6. `docs/DECISIONS.md` — 已定型决策,别重复推翻

## 下一步最该做什么

按 `docs/TASKS.md` 的优先级走。当前最高优先项是验证多模态链路的真实可用性(见"遗留问题"):上游网关需配置确实开启视觉能力的模型,否则图片上传虽走通但模型回"无法查看图片"。文档上传(前端提取文字)不依赖模型视觉,已可用。

## 不要做什么

- 不要在没读 `docs/BUSINESS_RULES.md` 的情况下改积分扣费、计费、租户隔离、LLM 路由、chat。
- 不要改模型结构体而不同步 `internal/db/schema.go` 的 JSON Schema。
- 不要 `git add` / `git commit`,除非用户明确要求。
- 不要为依赖项目裸跑 `fly deploy`(必须 `Dockerfile.saas` + `fly.saas.toml`)。
- 不要把归档/清理变成业务代码改动。

## 下次 /clear 后可直接复制的提示词

```
请先读 CLAUDE.md 和 docs/START_HERE.md 了解 AgentStore 项目现状。
这是一个已完成 V1 的多租户 AI Agent 市场(Go 后端 + React 前端 + MongoDB + Stripe + OpenAI 兼容 LLM)。
继续开发前请读 docs/TASKS.md 决定任务、读 docs/BUSINESS_RULES.md 避免误改业务逻辑。
开发禁区见 CLAUDE.md:高风险路径(credits/stripe/llm/chat/billing/tenant)要谨慎并跑测试;
改模型结构体必须同步 internal/db/schema.go;不要 git commit 除非我明确要求。
改完按 CLAUDE.md 的命令验证,并按要求更新对应文档 + 追加 docs/CHANGELOG_AI.md。
我接下来要做的是:______(在此填写本次任务)。
```
