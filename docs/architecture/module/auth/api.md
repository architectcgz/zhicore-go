# Auth API 背后设计

本文只描述 API 背后的业务流程和 use case 追踪；字段级 HTTP schema 放在 `services/zhicore-auth/api/http/`。

## 鉴权上下文

| API | 鉴权 | 说明 |
| --- | --- | --- |
| `POST /api/v1/auth/register` | 匿名 | 创建账号凭证和默认角色，并触发 User profile 初始化。 |
| `POST /api/v1/auth/login` | 匿名 | 校验登录标识、账号状态和密码，签发 token。 |
| `POST /api/v1/auth/refresh` | 匿名 + refresh token | 校验 refresh token，执行 rotation。 |
| `POST /api/v1/auth/logout` | 登录用户或 refresh token | 吊销当前 refresh token；access token 黑名单按 contract 决定。 |
| `GET /api/v1/auth/me` | 登录用户 | 返回当前认证主体、账号状态和角色；用户资料由 User 查询。 |
| `GET /api/v1/auth/csrf` | 匿名 | 签发非 HttpOnly `csrf_token` cookie，并在 body 返回同值；不改变 refresh session。 |
| `GET /api/v1/auth/sessions` | 登录用户 | 返回当前账号活跃 refresh sessions 的设备展示列表。 |
| `DELETE /api/v1/auth/sessions/current` | 登录用户 | 撤销当前 session，清 refresh/csrf cookie，并让当前 access token 失效。 |
| `DELETE /api/v1/auth/sessions/{sessionId}` | 登录用户 | 撤销当前账号下指定 session，让目标设备 refresh 和 access token 失效。 |
| `GET /api/v1/auth/security-operations/{operationId}` | 登录用户 / Admin | 查询安全撤销类 operation 的处理状态。 |

## Use Case 追踪

| Endpoint | Use case | 主要副作用 |
| --- | --- | --- |
| `POST /api/v1/auth/register` | `RegisterAccount` | 写入账号、凭证、默认角色、outbox；初始化 User profile 或发布事件。 |
| `POST /api/v1/auth/login` | `Login` | 按 `rememberMe` 创建标准 7 天或记住我 30 天 PostgreSQL refresh session，签发 access / refresh token，更新登录安全审计，并尽力写 Redis 缓存。 |
| `POST /api/v1/auth/refresh` | `RefreshToken` | 基于 PostgreSQL session 和 token hash 执行 rotation，并沿用 session 原始持久化策略滑动续期；重放时吊销当前 session 或升级账号级处置。 |
| `POST /api/v1/auth/logout` | `Logout` | 吊销当前 refresh token，必要时写 access token 黑名单。 |
| `GET /api/v1/auth/me` | `GetCurrentPrincipal` | 无业务写入。 |
| `GET /api/v1/auth/csrf` | `GetCSRFToken` | 签发 CSRF token 并覆盖 `csrf_token` cookie；不签发 access/refresh token。 |
| `GET /api/v1/auth/sessions` | `ListSessions` | 无业务写入；只返回设备展示需要的 session 摘要。 |
| `DELETE /api/v1/auth/sessions/current` | `RevokeCurrentSession` | 撤销当前 refresh session，写 access token 撤销投影并清 cookie。 |
| `DELETE /api/v1/auth/sessions/{sessionId}` | `RevokeSession` | 撤销指定 refresh session，写 session revoked 投影和 audit。 |
| `GET /api/v1/auth/security-operations/{operationId}` | `GetSecurityOperation` | 无业务写入；返回安全操作状态，不泄露 Redis key 或 token 材料。 |

## 限流归属

Auth API 限流分两层：Gateway 按 IP、route 和基础突发流量做粗限流；Auth 按账号、email、session、token、purpose、actor 和业务失败结果做安全限流。完整矩阵见 `rate-limiting.md`。

核心规则：

- `login`、`register`、验证码发送、密码找回发送等匿名入口必须同时有 IP 维度和业务标识维度限流。
- `refresh` 按 sessionId、accountId 和 IP 限流；refresh replay 不按普通限流吞掉，必须进入安全审计和 session/账号风险处置。
- `logout current`、`DELETE /auth/sessions/current` 这类降低风险动作不应仅因普通限流被拒绝；可以限制重复提交成本，但仍应尽力撤销当前 session 和清 cookie。
- `password/change`、`account/deactivate`、`logout all`、Admin revoke/ban/role command 等高风险写操作按 actor、target、session、IP 限流，并写 audit。
- `auth/me`、session list 和 security operation 查询只做轻量读限流、分页和查询窗口限制，不改变登录态。
- Redis 不可用时，匿名和验证码类 API 只能短时用本机限流兜底；高风险写操作不能因为缺失分布式限流而 fail-open。

## 注册流程

第一阶段推荐使用同步 profile 初始化，避免注册成功但用户资料缺失：

```text
Auth RegisterAccount
-> Auth 本地事务创建 account / credential / role / outbox
-> 调用 User CreateProfileForAccount
-> 成功后返回 token 或账号摘要
```

如果后续改成事件驱动，必须定义 pending profile 状态、补偿任务和前端可见错误语义。

## `me` 的返回边界

`GET /api/v1/auth/me` 只返回认证主体事实，例如 `accountId`、`username`、`roles`、`status`。昵称、头像、简介和用户展示摘要由 User contract 提供，避免 Auth 复制 profile DTO。
