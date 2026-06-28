# Auth 限流设计

本文是 Auth 模块限流策略的专题事实源。字段级 HTTP schema 只引用本文，不重复完整矩阵。

## 目标

Auth 限流同时服务三个目标：

- 防止密码爆破、验证码轰炸、邮箱枚举、refresh replay 和 Admin 批量误操作。
- 在 Redis 短时不可用时明确哪些接口可以用本机限流兜底，哪些必须 fail closed 或进入处理中。
- 避免把安全收敛动作误伤，例如用户正在 logout、踢设备或改密码时，不应被普通刷接口限流挡住止损。

## 两层限流

| 层级 | 归属 | 职责 |
| --- | --- | --- |
| Gateway 粗限流 | `zhicore-gateway` | 按 IP、route、method、基础突发流量限流，阻挡明显洪水流量；不依赖账号是否存在。 |
| Auth 业务安全限流 | `zhicore-auth` | 按 email、account、session、token、purpose、actor、target 和失败结果限流；可以触发锁定、审计、operation 或安全处置。 |

Gateway 不能替代 Auth 业务限流。Gateway 不知道密码校验结果、账号状态、refresh replay、验证码 purpose 或 Admin 目标账号。

Auth 限流 key 只能保存规范化值或 hash。不得在 Redis key、日志或 metrics label 中保存 password、access token、refresh token、cookie、Authorization header、完整请求体或原始敏感输入。

## API 矩阵

| API / 能力 | Gateway 粗限流 | Auth 业务安全限流 | Redis 不可用时 |
| --- | --- | --- | --- |
| `POST /auth/login` | IP + route | `email_normalized + IP`、email、IP；失败达到阈值后设置 `locked_until`。 | 短时使用更严格本机限流；持续不可用后停止登录。 |
| `POST /auth/register` | IP + route | IP、email、可选 device/request fingerprint；同 email pending 重试不因普通限流误伤，User 下游失败重试有冷却窗口。 | 短时本机限流兜底；持续不可用后停止注册或返回临时不可用。 |
| `POST /auth/email-verification/send` | IP + route | email + purpose、IP、device、账号状态；设置 resend cooldown 和日上限。 | 短时本机限流兜底；持续不可用后停止发送。 |
| `POST /auth/email-verification/verify` | IP + route | token/email + IP；失败超过阈值后作废验证码或触发冷却。 | 短时本机限流兜底；验证码状态仍以 DB 为准。 |
| `POST /auth/password-reset/send` | IP + route | email + purpose、IP、device、账号状态；对外保持统一响应，避免邮箱枚举。 | 短时本机限流兜底；持续不可用后停止发送。 |
| `POST /auth/password-reset/verify` | IP + route | token/email + IP；失败超过阈值后作废验证码或触发冷却。 | 短时本机限流兜底；验证码状态仍以 DB 为准。 |
| `POST /auth/password-reset/confirm` | IP + route | reset token、email、IP；成功后进入高风险 `sessionVersion` 增量和会话吊销流程。 | 若无法完成 Redis 撤销投影，返回 `202 PROCESSING` 或失败，不承诺旧会话已失效。 |
| `POST /auth/refresh` | IP + route | sessionId、accountId、IP；refresh replay 不按普通限流吞掉，直接写 audit 并进入 session/账号级风险处置。 | 按 Redis 降级矩阵处理：仅当 Gateway 能回源 Auth 校验 access state 时允许短时降级。 |
| `GET /auth/csrf` | IP + route | IP + sessionId；允许多标签恢复，但防止被刷。 | 可短时本机限流兜底。 |
| `POST /auth/logout` / `DELETE /auth/sessions/current` | IP + route，主要限制重复提交成本 | 当前 session、account、IP；这是降低风险动作，不应仅因普通限流被拒绝。 | 仍应尽力撤销当前 session 和清 cookie；Redis 投影失败时返回 `202 PROCESSING` 或失败。 |
| `POST /auth/logout-all` / `DELETE /auth/sessions/{sessionId}` | IP + route | actor account、target account、sessionId、IP；写 audit，超限时优先合并为已有未完成 operation。 | Redis 投影不可确认时返回 `202 PROCESSING` 或失败。 |
| `POST /auth/password/change` | IP + route | account、session、IP；要求当前密码和 CSRF，多次失败可触发 step-up、临时锁定或更严格审计。 | DB 可完成改密时仍必须处理旧会话失效；Redis 投影失败时返回 `202 PROCESSING` 或失败。 |
| `POST /auth/account/deactivate` | IP + route | account、session、IP；要求当前密码和 CSRF，写 audit。 | Auth 状态可先落 DB；Redis 投影或 User 去激活未完成时返回处理中。 |
| `GET /auth/me` | IP + route | account、session、IP 轻量读限流。 | 可回源 DB 查询；记录 degraded metric，不能改变登录态。 |
| `GET /auth/sessions` | IP + route | account、session、IP；分页和查询窗口限制。 | 可回源 DB 查询；记录 degraded metric。 |
| `GET /auth/security-operations/{operationId}` | IP + route | account、operationId、IP；用户只能查自己的 operation，Admin 走 Admin 审计权限。 | 可回源 DB 查询；记录 degraded metric。 |
| Internal typed client / Admin facade | 服务路由 + IP / service | 服务签名、nonce/timestamp 防重放；按 calling service、actor、target、command 类型限流。 | 高风险命令不能因为缺失分布式限流而 fail-open；可返回处理中或失败。 |

## Redis 故障原则

Redis 不可用时不能统一放行，也不能把所有 Auth API 直接打死。

- 匿名入口和验证码类 API 只能短时用本机限流兜底，阈值必须比正常 Redis 分布式限流更严格。
- 高风险写操作不能因为本机限流缺失而 fail-open；如果无法确认 Redis 撤销投影，返回 `202 PROCESSING` 或失败。
- 查询类 API 可降级到 DB，但必须记录 degraded metric，避免系统长期带病运行而不可见。
- refresh 只在 Gateway 能回源 Auth 校验 access state 时允许短时降级；否则返回 `503`。

## 安全收敛动作

以下动作的目标是降低风险，不应被普通限流直接阻断：

- `logout current`
- `DELETE /auth/sessions/current`
- 被盗 token 场景下的 `logout all`
- 用户主动 revoke 自己的 session

可以限制重复请求的响应成本，例如短时间内返回同一个未完成 operation，但必须尽力执行 DB revoke、清 cookie、写 audit 和 Redis 投影补偿。

## 配置和观测

所有阈值、窗口和冷却时间必须配置化。首批默认值在实现阶段随配置文档固定，不能写死在 handler 中。

每类限流至少记录：

- allow / reject 计数
- 触发维度，例如 `route`、`reason`、`limitType`
- degraded fallback 计数
- Redis unavailable 计数
- 账号锁定、验证码作废、refresh replay、安全 operation 创建等安全事件计数

metrics label 不得包含原始 email、IP、token、cookie、Authorization header 或用户输入文本。
