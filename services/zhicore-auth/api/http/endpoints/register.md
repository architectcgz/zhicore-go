# Register

## 来源

- 服务总览：`docs/architecture/services/auth/README.md`
- 模块 API 设计：`docs/architecture/module/auth/api.md`
- 模块 service 设计：`docs/architecture/module/auth/service.md`
- 数据模型：`docs/architecture/module/auth/data-model.md`
- 限流设计：`docs/architecture/module/auth/rate-limiting.md`
- Redis 降级决策：`docs/architecture/module/auth/decision-log.md`
- 当前 API schema：`services/zhicore-auth/api/http/README.md`
- Go handler：待实现
- Go contract test：待补

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/auth/register` |
| 兼容别名 | 无 |
| Content-Type | `application/json` |
| 鉴权 | 匿名 |
| 幂等 | 以 `emailVerificationToken` 和 `email` 绑定关系控制；未过期 `PENDING_PROFILE` 可按注册重试语义继续。 |

## Path 参数

无。

## Query 参数

无。

## Body 字段 `RegisterReq`

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `email` | string | 是 | 不允许为空 | 注册邮箱；必须与 `emailVerificationToken` 绑定的 normalized email 一致。 |
| `nickname` | string | 是 | 不允许为空 | 初始化 User profile 的展示名；Auth 不在 `auth/me` 中返回 profile 字段。 |
| `password` | string | 是 | 不允许为空 | 明文密码只在本次请求中使用；服务端只保存 password hash。 |
| `emailVerificationToken` | string | 是 | 不允许为空 | 邮箱验证码 verify 成功后签发的短期一次性 opaque token，`purpose=register`。 |

## 成功响应 `RegisterResp`

Redis 正常且 Gateway 可见的 session/version/principal 投影写入成功时，注册成功后可自动登录，返回 access token 并通过 `Set-Cookie` 写入 `refresh_token` 和 `csrf_token`。refresh token 不进入 body；CSRF token 只作为 `csrfToken` body 字段和 `csrf_token` cookie 给浏览器后续提交 `X-CSRF-Token` 使用。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `registered` | boolean | 是 | 固定 `true`。 |
| `authenticated` | boolean | 是 | 已自动登录时为 `true`。 |
| `accessToken` | string | 条件必填 | `authenticated=true` 时返回 Bearer access token；不返回 refresh token。 |
| `tokenType` | string | 条件必填 | `authenticated=true` 时固定 `Bearer`。 |
| `expiresIn` | int | 条件必填 | `authenticated=true` 时固定 `7200`，单位秒。 |
| `csrfToken` | string | 条件必填 | `authenticated=true` 时返回，并同步写入非 HttpOnly `csrf_token` cookie。 |
| `principal` | object | 条件必填 | `authenticated=true` 时返回当前认证主体。 |
| `loginDeferredReason` | string/null | 是 | Redis/Gateway 投影不可用导致首期不自动登录时返回稳定原因码，例如 `AUTH_PRINCIPAL_UNAVAILABLE`；正常为 `null`。 |

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

Redis 不可用时，首期不走注册成功自动登录。账号和 User profile 已成功创建仍可返回 HTTP `200`，但 `authenticated=false`，不得签发 `accessToken`，不得写入 `refresh_token` cookie；前端应提示注册成功并稍后登录。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | 字段缺失、格式非法、`emailVerificationToken` 过期/已使用/email 不匹配。 |
| `2010` | `400` | email 格式非法 | `email` 不符合邮箱格式。 |
| `2011` | `400` | password 不符合策略 | 密码长度、复杂度或黑名单策略不满足要求。 |
| `2009` | `409` | email 已被占用 | email 已被 `ACTIVE`、`DISABLED`、`BANNED` 等可占用账号使用。 |
| `2012` | `503` / `409` | 注册 pending 可重试 | 命中未闭合的 `PENDING_PROFILE`，调用方可按重试语义继续；具体 status 由 pending 场景固定。 |
| `2015` | `429` | 请求过于频繁 | 触发注册 IP、email、设备/请求指纹等安全限流。 |
| `1004` | `503` | 服务暂时不可用 | Auth DB、User profile 初始化依赖、验证码 token 存储或长期 Redis 降级窗口不可用。 |

## 权限和可见性

- 匿名调用，不接受客户端提交的 `accountId`、`userId`、`roles` 或账号状态字段。
- 注册时可以明确返回 `AUTH_EMAIL_EXISTS`；枚举风险由验证码、限流和风控控制。
- `nickname` 只用于初始化 User profile；Auth principal 不复制头像、bio、profile summary 等 User 字段。

## 排序、分页和过滤

无。

## 测试要求

- Handler contract test：待补，覆盖正常注册并自动登录、Redis 不可用不自动登录、email token 无效、email 冲突、pending retry、密码策略失败。
- System HTTP test：待补。
