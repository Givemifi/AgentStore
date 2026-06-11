# AgentStore 任务清单

> 决定"做什么"看这里。状态以本次归档（2026-06-11）为准。

## V1 已完成

- 多租户 SaaS 底座 + RBAC(owner/admin/user)+ 租户隔离
- 认证全套：密码、OAuth、magic link、MFA/TOTP、JWT(access+refresh)+ 自动刷新
- Agent 市场(预置 24 个精品 agent)+ SSE 流式对话 + Markdown 渲染(含代码块复制)
- 积分系统：订阅积分 + 购买积分包，对话先验后扣，余额不足拦截(402)
- Stripe 计费：checkout、订阅、portal、税、退款、争议、优惠码、积分包购买
- **微信支付 H5 + 支付宝网页积分包支付**：V3 API 验签、金额校验、幂等回调、前端轮询(需真实商户凭证联调)
- admin 后台：用户/租户/套餐/计费/品牌/健康/日志/API key/webhook/遥测/配置/上线就绪
- 品牌白标(应用名、配色、logo)+ 运行时配置(configstore)
- 系统日志(syslog,分级)+ 健康/指标/DataDog 集成
- **多模态 + 语音对话**：按住说话(Web Speech API)、图片(vision base64)、文档(PDF/Word/TXT 前端提取)上传 + 手机浏览器适配
- **i18n 英/中双语**：react-i18next，英文默认，浏览器自动检测，Language Switcher(header + auth 页)
- **Chat Markdown 渲染增强**：代码块 + 语言标签 + 复制按钮、**bold**/*italic*、列表、标题、blockquote；`memo` 优化
- **公开 Agent 市场**(`/agents`, `/agents/:slug`)：无需登录浏览 24 个 agent，搜索/分类/详情/一键开始
- **SEO**：`spaHandler` 动态 OG meta；`/sitemap.xml`；`/robots.txt`
- **后端公开 API**：`GET /api/public/agents`、`GET /api/public/agents/{slug}`
- **Agent 目录扩充**：从 6 扩到 24 个精品 agent，7 大类，system prompt 含多语言自适应
- Signup 支持 `?redirect=` 回跳参数
- **M4a — 注册送积分 configstore 化**：`growth.trial_credits` 可在 admin 后台配置(默认 25)
- **M4b — 对话分享**：`share_links` collection(快照式)；`POST /chat/conversations/:id/share`；`GET /api/public/share/:token`；ChatPage 分享按钮；`/share/:token` 公开分享页
- **M4c — 推荐码系统**：User 模型加 `ReferralCode/ReferredBy`；注册时生成唯一推荐码；邮箱验证后异步发放双向奖励；`growth.referral_reward_referee/referrer` configstore 配置(默认 0)；Settings → Profile 「邀请好友」区块含推荐链接复制
- **M5 — 信任合规**：GDPR 数据导出(`/auth/export-data`)和删号(`/auth/delete-account`)已实现；隐私/条款通过 branding pages(`/p/privacy-policy`, `/p/terms-of-service`)承载；`PublicFooter` 组件加到市场页；User 类型加 `referralCode/referredBy`
- **打磨与开源就绪（2026-06-11）**：修 `/api/admin/promotions` 500（补 nil 守卫）；修前端 lint CI 阻断（PublicFooter 未使用 t）；清 dev.yaml 硬编码 LLM key；删死代码（AdminRoute、ui/Alert）和编译产物。

## 遗留问题

- **多模态图片真实可用性(最高优先)**：需在 Admin → LLM 配置启用确实支持视觉的模型端点再验证。
- **微信/支付宝真实联调待完成**：需提供真实商户凭证 + 公网回调地址(或 ngrok)。支付宝沙箱(`ALIPAY_IS_SANDBOX=true`)可先测。
- **图像/视频生成积分未启用**：字段已有但为 0；若启用生成类能力需补扣费逻辑。
- **admin 角色权限边界**：精确读写/删除边界以 handler 实际检查为准，建议补权限矩阵文档。
- **推荐奖励默认为 0**：需 admin 配置 `growth.referral_reward_referee/referrer` 才能实际发奖励。
- **分享页 OG 注入**：spaHandler 尚未对 `/share/:token` 路径单独注入对话标题 OG（可扩展）。
- **Footer 中的联系邮件**：`PublicFooter` 中 `hello@agentstore.ai` 为占位符，上线前需更新或从 configstore 读取。
- **46 个 lint warning 待清理**：主要是 React 19 `set-state-in-effect`（非 CI 阻断，可单独一轮处理）。
- **admin/settings/支付流 i18n 债务**：约 57 处 CJK 硬编码字符串未走 i18n namespace（对运营方影响小，留作后续里程碑）。

## 下一步优先任务

1. 在 Admin → configstore 配置 `growth.trial_credits`（已可配置，无需开发）；视需求调整推荐奖励数值。
2. 在 Admin → 品牌页面 seed `/p/privacy-policy` 和 `/p/terms-of-service` 内容。
3. 配好视觉模型端点，端到端验证图片多模态对话。
4. 真机(iOS Safari / Android Chrome)回归语音 + 附件 + 键盘适配。
5. 梳理并文档化 RBAC 权限矩阵。
6. 补充多模态/语音前端测试(`ChatPage.test.tsx`)与 `attachments.ts` 单测。
7. 配微信/支付宝真实商户凭证 + 公网域名，端到端联调国内支付。

## 暂缓事项

- 图像/视频生成类 Agent 能力(需先定计费模型)。
- 第三方 OAuth / Resend / DataDog 的生产配置(dev 可选,上线前再配)。
- admin 后台 i18n(运营方自用,英文够用,后续里程碑)。

## 不建议现在做的事项

- 大规模重构高风险路径(credits/stripe/llm/chat/tenant),除非有明确需求并配套测试。
- 在遗留问题关闭前引入新功能。
- 任何"清理"动作触碰业务代码。
