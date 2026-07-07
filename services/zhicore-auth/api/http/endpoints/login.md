# Login

## 来源

- 服务总览：`docs/architecture/services/auth/README.md`
- 模块 API 设计：`docs/architecture/module/auth/api.md`
- 模块 service 设计：`docs/architecture/module/auth/service.md`
- 数据模型：`docs/architecture/module/auth/data-model.md`
- 限流设计：`docs/architecture/module/auth/rate-limiting.md`
- Redis 降级决策：`docs/architecture/module/auth/decision-log.md`
- 当前 API schema：`services/zhicore-auth/api/http/README.md`
- Go handler：`services/zhicore-auth/api/http/handler.go`
- Go contract test：`services/zhicore-auth/api/http/auth_handler_test.go`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/auth/login` |
| 兼容别名 | 无 |
| Content-Type | `application/json` |
| 鉴权 | 匿名 |
| 幂等 | 无；成功登录会创建新的 refresh session。 |

## Path 参数

无。

## Query 参数

无。

## Body 字段 `LoginReq`

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `email` | string | 是 | 不允许为空 | 登录邮箱。 |
| `password` | string | 是 | 不允许为空 | 明文密码只用于本次校验。 |
| `rememberMe` | boolean | 是 | 不允许为空 | `false` 创建标准 7 天 refresh session；`true` 创建记住我 30 天 refresh session。该字段只影响 refresh session / cookie TTL，不影响 access token TTL。 |

## 成功响应 `LoginResp`

登录成功返回 access token；refresh token 只通过 HttpOnly `refresh_token` cookie 写入，不出现在 body。CSRF token 同时通过非 HttpOnly `csrf_token` cookie 和 `csrfToken` body 字段返回，后续变更 session 的请求必须通过 `X-CSRF-Token` header 提交。`refresh_token` cookie 的 `Expires/Max-Age` 与 PostgreSQL refresh session `expiresAt` 对齐：`rememberMe=false` 为 `now + 7d`，`rememberMe=true` 为 `now + 30d`。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `accessToken` | string | 是 | Bearer access token。 |
| `tokenType` | string | 是 | 固定 `Bearer`。 |
| `expiresIn` | int | 是 | 固定 `7200`，单位秒。 |
| `csrfToken` | string | 是 | 与非 HttpOnly `csrf_token` cookie 同值，用于后续 session 变更请求。 |
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

Redis 短时不可用时，登录可在 Auth DB 校验通过后降级成功，但必须使用本机内存限流兜底、阈值更严格，并记录 degraded metric/告警；降级细节不暴露在响应 body 中。Redis 不可用超过配置窗口时返回 `1004`。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | 字段缺失或类型非法。 |
| `2010` | `400` | email 格式非法 | `email` 不符合邮箱格式。 |
| `2003` | `401` | 登录凭证错误 | email 不存在或 password 错误，普通登录失败不区分二者。 |
| `2004` | `403` | 账号禁用 | 账号状态为 `DISABLED`。 |
| `2019` | `403` | 账号被封禁 | 账号状态为 `BANNED`。 |
| `2014` | `403` | 账号临时锁定 | 登录失败次数或风控策略触发 `locked_until`。 |
| `2015` | `429` | 请求过于频繁 | 触发 IP、email、失败结果等登录安全限流。 |
| `1004` | `503` | 服务暂时不可用 | Auth DB 不可用，或 Redis 不可用超过允许降级窗口。 |

## 权限和可见性

- 匿名调用，不接受客户端提交的 `accountId`、`userId`、`roles` 或账号状态字段。
- 不返回 refresh token 明文、password hash、token hash、Redis key、完整 IP 或完整 User-Agent。
- 登录失败审计不得记录 password、refresh/access token、cookie 或 Authorization header。

## 排序、分页和过滤

无。

## 测试要求

- Handler / application contract test：已验证，覆盖 `rememberMe` 字段必填、`rememberMe=true` 传入 service、标准 7 天 / 记住我 30 天 refresh session TTL、session 持久化策略、成功登录 Set-Cookie、refresh token 不进 body、无效凭证统一错误、禁用/封禁/锁定。
- System HTTP test：待补。
