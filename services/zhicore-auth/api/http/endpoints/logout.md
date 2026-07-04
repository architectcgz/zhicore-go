# Logout

## 来源

- 服务总览：`docs/architecture/services/auth/README.md`
- 模块 API 设计：`docs/architecture/module/auth/api.md`
- 模块 service 设计：`docs/architecture/module/auth/service.md`
- Redis key 设计：`docs/architecture/module/auth/redis-keys.md`
- 当前 API schema：`services/zhicore-auth/api/http/README.md`
- Go handler：`services/zhicore-auth/api/http/handler.go`
- Go contract test：`services/zhicore-auth/api/http/auth_handler_test.go`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/auth/logout` |
| 兼容别名 | `DELETE /api/v1/auth/sessions/current` |
| Content-Type | 无 body |
| 鉴权 | 登录用户或 refresh cookie；无有效凭证时只执行本地清 cookie 语义 |
| 幂等 | 对同一浏览器重复调用应尽量返回同一最终语义 |

## Header

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `Authorization` | 否 | 有 access token 时经 Gateway 验证并注入身份上下文；Auth handler 不直接解析客户端 JWT。 |
| `X-CSRF-Token` | 条件必填 | 只要请求携带可信身份上下文或 refresh cookie 并需要服务端撤销 session，就必须提交。 |

## Cookie

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `refresh_token` | 否 | 有效时 Auth 校验 opaque refresh token 并撤销当前 refresh session。 |
| `csrf_token` | 条件必填 | 只要请求携带可信身份上下文或 refresh cookie 并需要服务端撤销 session，就必须与 `X-CSRF-Token` 做 double-submit 校验。 |

## Path 参数

无。

## Query 参数

无。

## Body 字段

无。

## 成功响应 `LogoutResp`

HTTP `200` 表示可执行的当前 session revoke 和 Gateway 可见撤销投影已完成；如果请求没有任何有效登录凭证，也可返回 `200` 表示已尽力清理本地 cookie。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `loggedOut` | boolean | 是 | 固定 `true`。 |
| `serverRevoked` | boolean | 是 | 是否实际撤销了服务端 refresh session 或 access token 撤销投影。 |

所有 `logout` 响应都应尽量用与写入一致的 `Domain/Path/SameSite/Secure` 清理 `refresh_token` 和 `csrf_token` cookie。

## 处理中响应 `data`

HTTP `202` 表示 DB revoke 已提交或安全操作已受理，但 Redis 撤销投影未确认完成；调用方不能承诺旧 access token 已失效。

`202` 仍使用成功 envelope，body `code` 固定为 `200`；异步处理标识放在 `data.operationId`。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `operationId` | string | 是 | 安全 operation ID。 |
| `status` | string | 是 | 固定 `PROCESSING`。 |
| `retryAfterSeconds` | int | 是 | 建议前端轮询间隔。 |
| `loggedOut` | boolean | 是 | 固定 `true`，表示本地 cookie 已尽力清理。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2013` | `403` | CSRF 校验失败 | 请求携带 refresh cookie 且需要服务端 session revoke，但 CSRF header/cookie 缺失或不匹配。 |
| `2018` | `401` | 会话已失效 | refresh cookie 指向的 session 已撤销、过期或 replay 风险已触发。 |
| `2015` | `429` | 请求过于频繁 | 触发重复提交成本限制；不得阻断安全收敛补偿。 |
| `1004` | `503` | 服务暂时不可用 | DB 或安全投影依赖不可用，且无法创建 operation。 |

无 access token、无 refresh cookie 或二者都无效但未触发 replay 风险时，`logout` 仍返回 `200` 并清 cookie，不使用 `401` 阻止前端清理本地登录态。

## 权限和可见性

- 有 Gateway 注入身份时，按当前身份撤销当前 session 和当前 access token。
- 只有 refresh cookie 时，Auth 只基于 refresh token 定位当前 session，不接收客户端提交的 `accountId/sessionId`。
- 无有效凭证时不得操作任何服务端账号/session，只做清 cookie 幂等响应。

## 排序、分页和过滤

无。

## 测试要求

- Handler contract test：已验证，覆盖 access + refresh、成功清 cookie、CSRF 失败，以及无可信身份 header 且无 refresh cookie 时只做本地清理、不调用 service。
- System HTTP test：待补。
