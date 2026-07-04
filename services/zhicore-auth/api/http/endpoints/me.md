# Get Current Principal

## 来源

- 服务总览：`docs/architecture/services/auth/README.md`
- 模块 API 设计：`docs/architecture/module/auth/api.md`
- 模块 service 设计：`docs/architecture/module/auth/service.md`
- 当前 API schema：`services/zhicore-auth/api/http/README.md`
- Go handler：待实现
- Go contract test：待补

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/auth/me` |
| 兼容别名 | 无 |
| Content-Type | 无 body |
| 鉴权 | 登录用户 |
| 幂等 | 查询接口，天然幂等 |

## Header

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `Authorization` | 条件必填 | 外部请求需提交 Bearer access token；由 Gateway 或 Auth middleware 校验后构造可信身份上下文，Auth handler 不直接解析客户端 JWT。 |
| `X-Account-Id` | 是 | Gateway 注入的当前 Auth account ID。 |
| `X-User-Id` | 是 | Gateway 注入的当前 user ID。 |
| `X-Session-Id` | 是 | Gateway 注入的当前 session ID。 |
| `X-Session-Version` | 是 | Gateway 注入的 token session version。 |
| `X-Principal-Version` | 是 | Gateway 注入的 token principal version。 |
| `X-User-Roles` | 否 | Gateway 注入的当前角色集合；Auth 可按 DB 当前事实重新构造响应。 |

## Path 参数

无。

## Query 参数

无。

## Body 字段

无。

## 成功响应 `AuthPrincipalResp`

`auth/me` 只返回认证主体事实，不返回 User profile 字段。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `principal` | object | 是 | 当前认证主体。 |

`principal`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `accountId` | string | 是 | Auth account ID。 |
| `userId` | string | 是 | User 服务返回并由 Auth 保存的 user ID。 |
| `email` | string | 是 | 账号邮箱。 |
| `roles` | array | 是 | 当前有效角色，空列表返回 `[]`。 |
| `accountStatus` | string | 是 | 账号状态，例如 `ACTIVE`。 |
| `sessionId` | string | 是 | 当前 access token 所属 session ID。 |
| `sessionVersion` | int | 是 | 登录态有效性版本。 |
| `principalVersion` | int | 是 | 认证主体快照版本。 |

不返回 `accessToken`、`refreshToken`、`csrfToken`、nickname、avatar、bio、avatarUrl 或 User profile summary。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2006` | `401` | 请先登录 | 缺少 Gateway 注入的可信身份上下文。 |
| `2001` | `401` | token 无效 | Gateway 或 Auth middleware 判定 access token 无效。 |
| `2002` | `401` | token 过期 | access token 已过期。 |
| `2018` | `401` | session 已撤销 | 当前 session 已撤销或 sessionVersion 已失效。 |
| `2004` | `403` | 账号禁用 | 账号状态为 `DISABLED`。 |
| `2019` | `403` | 账号被封禁 | 账号状态为 `BANNED`。 |
| `2015` | `429` | 请求过于频繁 | 触发 Auth 读限流。 |
| `2016` | `503` | 认证主体暂时不可确认 | Gateway/Auth 需要刷新 principal，但 Auth DB 或必要状态源不可用。 |

## 权限和可见性

- 只能返回当前调用者自己的 Auth principal。
- 资源归属由 Gateway 注入的 `X-Account-Id` 和 Auth 当前 DB 事实共同确认。
- User profile 由 `GET /api/v1/users/me` 提供；Auth 不复制 profile DTO。

## 排序、分页和过滤

无。

## 测试要求

- Handler contract test：待补，覆盖成功返回 principal、缺身份 header、已撤销 session、禁用/封禁状态、profile 字段不返回。
- System HTTP test：待补。
