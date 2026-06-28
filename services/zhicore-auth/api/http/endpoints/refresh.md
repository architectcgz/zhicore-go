# Refresh

## 来源

- 服务总览：`docs/architecture/services/auth/README.md`
- 模块 API 设计：`docs/architecture/module/auth/api.md`
- 模块 service 设计：`docs/architecture/module/auth/service.md`
- 数据模型：`docs/architecture/module/auth/data-model.md`
- Redis key 设计：`docs/architecture/module/auth/redis-keys.md`
- 限流设计：`docs/architecture/module/auth/rate-limiting.md`
- Gateway route risk：`docs/architecture/services/gateway/route-risk-policy.md`
- 当前 API schema：`services/zhicore-auth/api/http/README.md`
- Go handler：待实现
- Go contract test：待补

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/auth/refresh` |
| 兼容别名 | 无 |
| Content-Type | 无 body |
| 鉴权 | 匿名 + refresh cookie |
| 幂等 | 非幂等；成功 refresh 必须执行 rotation。 |

## Header

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `X-CSRF-Token` | 是 | 与 `csrf_token` cookie 做 double-submit 校验。 |

## Cookie

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `refresh_token` | 是 | HttpOnly opaque refresh token；Auth 只保存 hash，不保存明文。 |
| `csrf_token` | 是 | 非 HttpOnly CSRF token；必须与 `X-CSRF-Token` 一致。 |

## Path 参数

无。

## Query 参数

无。

## Body 字段

无。

## 成功响应 `data`

Redis 正常时，Auth 基于 PostgreSQL refresh session 真相源校验 token hash，执行 rotation，签发新 access token 和新 refresh token，并覆盖 `refresh_token` 与 `csrf_token` cookie。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `accessToken` | string | 是 | 新 Bearer access token。 |
| `tokenType` | string | 是 | 固定 `Bearer`。 |
| `expiresIn` | int | 是 | 固定 `7200`，单位秒。 |
| `csrfToken` | string | 是 | 新 CSRF token，与响应写入的 `csrf_token` cookie 同值。 |
| `principal` | object | 是 | 当前认证主体。 |

`principal`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `accountId` | string | 是 | Auth account ID。 |
| `userId` | string | 是 | User 服务返回并由 Auth 保存的 user ID。 |
| `email` | string | 是 | 账号邮箱。 |
| `roles` | array | 是 | 当前有效角色，空列表返回 `[]`。 |
| `accountStatus` | string | 是 | 账号状态，例如 `ACTIVE`。 |
| `sessionVersion` | int | 是 | 登录态有效性版本。 |
| `principalVersion` | int | 是 | 认证主体快照版本。 |

Redis 短时不可用但 Auth DB 正常且 Gateway 能回源 Auth `ValidateAccessState(claims)` 时，refresh 可以降级成功；服务端必须记录 degraded metric、使用更严格限速，并避免在 body 暴露 Redis 细节。Redis 不可用且 Gateway 不能回源 Auth 时，refresh 必须返回 `503`，不得签发新的 access token。

## 处理中响应 `data`

账号或 session 存在未完成安全 operation，或高风险撤销刚发生且 Gateway 可见投影未完成时，refresh 不签发新 token，可返回 HTTP `202` 表示安全同步中。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `operationId` | string | 是 | 相关安全 operation ID。 |
| `status` | string | 是 | 固定 `PROCESSING`。 |
| `retryAfterSeconds` | int | 是 | 建议前端重试或轮询间隔。 |
| `refreshAccepted` | boolean | 是 | 固定 `false`，表示本次未执行 refresh rotation。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2013` | `403` | CSRF 校验失败 | `X-CSRF-Token` 或 `csrf_token` cookie 缺失、不匹配。 |
| `2001` | `401` | token 无效 | refresh token 格式非法、hash 不匹配或无法定位 session。 |
| `2002` | `401` | token 过期 | refresh session 或 refresh token 已过期。 |
| `2017` | `401` | refresh token replay | 已失效 tokenId/token hash 再次出现，当前 session 被吊销或升级风险处置。 |
| `2018` | `401` | session 已撤销 | refresh session 已撤销、账号级 sessionVersion 已失效或安全处置要求重新登录。 |
| `2004` | `403` | 账号禁用 | 账号状态为 `DISABLED`。 |
| `2019` | `403` | 账号被封禁 | 账号状态为 `BANNED`。 |
| `2014` | `403` | 账号临时锁定 | 账号处于 `locked_until` 风控窗口。 |
| `2015` | `429` | 请求过于频繁 | 触发 sessionId、accountId 或 IP refresh 限流。 |
| `1004` | `503` | 服务暂时不可用 | Auth DB 不可用；或 Redis 不可用且 Gateway 不能回源 Auth 校验 access state。 |

## 权限和可见性

- refresh 只基于 HttpOnly `refresh_token` cookie 定位 session，不接受 body 中的 `accountId/sessionId/token`。
- 成功 refresh 不因 rotation 自动黑名单旧 access token；旧 access token 仍按自身 `exp`、`jti`、session/version 状态由 Gateway 判断。
- refresh replay 是安全事件，不能被普通限流吞掉。
- 不返回 refresh token 明文、refresh token hash、tokenId、Redis key、jti 原值、完整 IP 或完整 User-Agent。

## 排序、分页和过滤

无。

## 测试要求

- Handler contract test：待补，覆盖正常 rotation、CSRF 失败、refresh replay、revoked session、Redis 正常、Redis 短时降级成功、Redis 不可用且 Gateway 不可回源返回 `503`、安全 operation `202`。
- System HTTP test：待补。
