# Auth HTTP Schema

本目录记录 `zhicore-auth` 的服务级 HTTP contract。Go handler 实现必须以这里记录的字段级 schema 为准。

## 服务级规则

- 服务拥有 `/api/v1/auth` API family。
- `register`、`login`、`refresh` 直接处理客户端凭证；普通业务服务不得解析 `Authorization` 作为身份来源。
- 登录态 endpoint 读取 Gateway 注入的可信身份上下文，或在 Auth 服务自身入口中由 Auth middleware 校验 access token 后构造等价上下文。
- Auth 受保护 endpoint 至少需要 `X-Account-Id`、`X-User-Id`、`X-Session-Id`、`X-Session-Version`、`X-Principal-Version`；角色相关 endpoint 还需要 `X-User-Roles`。
- 变更 session 的浏览器 endpoint 需要 `X-CSRF-Token` 与 `csrf_token` cookie 做 double-submit 校验。
- `refresh_token` 只通过 HttpOnly cookie 传输，不在响应 body 返回。
- `login` 的 `rememberMe` 只决定 refresh session / cookie 的滑动 TTL：`false` 为 7 天，`true` 为 30 天；access token TTL 不受影响。
- `refresh` 不接收 `rememberMe`；成功 rotation 时沿用当前 refresh session 在登录时保存的原始持久化策略续期。
- `logout`、`DELETE /sessions/current` 和撤销当前 session 的响应必须尽力清理 `refresh_token` 和 `csrf_token` cookie。
- `202 PROCESSING` 表示安全撤销已受理但 Gateway 可见的 Redis 撤销投影尚未确认完成；前端不能提示“被盗 token 已失效”，应按 `operationId` 查询。
- `202 PROCESSING` 仍使用成功 envelope：HTTP status 为 `202`，但 body `code` 固定为 `200`，异步处理标识放在 `data.operationId`，不新增专用错误码。
- 成功和失败响应使用 `docs/contracts/http.md` 定义的 ZhiCore envelope。

## Endpoint 索引

| Endpoint | Use case | 设计文档 | Contract 状态 |
| --- | --- | --- | --- |
| `POST /api/v1/auth/register` | `RegisterAccount` | `endpoints/register.md` | 草案 |
| `POST /api/v1/auth/login` | `Login` | `endpoints/login.md` | 已验证 |
| `POST /api/v1/auth/refresh` | `RefreshToken` | `endpoints/refresh.md` | 草案 |
| `POST /api/v1/auth/logout` | `Logout` | `endpoints/logout.md` | 已验证 |
| `GET /api/v1/auth/me` | `GetCurrentPrincipal` | `endpoints/me.md` | 已验证 |
| `GET /api/v1/auth/csrf` | `GetCSRFToken` | `endpoints/csrf.md` | 已验证 |
| `GET /api/v1/auth/sessions` | `ListSessions` | `endpoints/list-sessions.md` | 已验证 |
| `DELETE /api/v1/auth/sessions/current` | `RevokeCurrentSession` | `endpoints/revoke-current-session.md` | 已验证 |
| `DELETE /api/v1/auth/sessions/{sessionId}` | `RevokeSession` | `endpoints/revoke-session.md` | 已验证 |
| `GET /api/v1/auth/security-operations/{operationId}` | `GetSecurityOperation` | `endpoints/get-security-operation.md` | 已验证 |

## 服务级公开错误码子集

| code | symbolic code | HTTP status | 含义 |
| --- | --- | --- | --- |
| `1001` | `VALIDATION_ERROR` | `400` | 请求字段、path 或 query 参数非法。 |
| `1004` | `SERVICE_DEGRADED` | `503` | DB、Redis 投影或安全同步依赖不可用。 |
| `1005` | `DATA_NOT_FOUND` | `404` | session / operation 不存在或不可见。 |
| `2001` | `AUTH_TOKEN_INVALID` | `401` | token 无效。 |
| `2002` | `AUTH_TOKEN_EXPIRED` | `401` | token 过期。 |
| `2003` | `AUTH_INVALID_CREDENTIALS` | `401` | 登录凭证错误。 |
| `2004` | `AUTH_ACCOUNT_DISABLED` | `403` | 账号禁用。 |
| `2005` | `AUTH_PERMISSION_DENIED` | `403` | 权限不足。 |
| `2006` | `AUTH_LOGIN_REQUIRED` | `401` | 缺少可信登录身份上下文。 |
| `2007` | `AUTH_ROLE_REQUIRED` | `403` | 需要特定角色。 |
| `2008` | `AUTH_RESOURCE_ACCESS_DENIED` | `403` | 无权访问目标资源。 |
| `2009` | `AUTH_EMAIL_EXISTS` | `409` | email 已被占用。 |
| `2010` | `AUTH_EMAIL_INVALID` | `400` | email 格式非法。 |
| `2011` | `AUTH_PASSWORD_INVALID` | `400` | password 不符合策略。 |
| `2012` | `AUTH_REGISTER_PENDING_RETRYABLE` | `503` / `409` | 注册 pending 可重试。 |
| `2013` | `AUTH_CSRF_INVALID` | `403` | CSRF header/cookie 缺失或不匹配。 |
| `2014` | `AUTH_ACCOUNT_LOCKED` | `403` | 账号临时锁定。 |
| `2015` | `AUTH_RATE_LIMITED` | `429` | 触发 Auth 安全限流。 |
| `2016` | `AUTH_PRINCIPAL_UNAVAILABLE` | `503` | 认证主体状态暂时不可确认。 |
| `2017` | `AUTH_TOKEN_REPLAYED` | `401` | refresh token replay。 |
| `2018` | `AUTH_SESSION_REVOKED` | `401` | session 已撤销或失效。 |
| `2019` | `AUTH_ACCOUNT_BANNED` | `403` | 账号被封禁。 |
| `2020` | `AUTH_DEACTIVATION_PROCESSING` | `202` | 注销处理中。 |
