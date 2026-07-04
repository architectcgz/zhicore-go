# Revoke Current Session

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
| 方法 | `DELETE` |
| 主路径 | `/api/v1/auth/sessions/current` |
| 兼容别名 | `POST /api/v1/auth/logout` 也可撤销当前 session |
| Content-Type | 无 body |
| 鉴权 | 登录用户 |
| 幂等 | 对同一当前 session 重复调用应尽量返回同一最终语义 |

## Header

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `X-CSRF-Token` | 是 | 变更 session 的浏览器请求必须提交；Auth 校验它与 `csrf_token` cookie 一致。 |

## Path 参数

无。

## Query 参数

无。

## Body 字段

无。

## 成功响应 `data`

HTTP `200` 表示当前 session 的 DB revoke 和 Gateway 可见撤销投影均已完成。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `status` | string | 是 | 固定 `REVOKED`。 |
| `sessionId` | string | 是 | 被撤销的当前 session ID。 |

响应必须使用与写入一致的 `Domain/Path/SameSite/Secure` 清理 `refresh_token` 和 `csrf_token` cookie。

## 处理中响应 `data`

HTTP `202` 表示 DB revoke 已提交或安全操作已受理，但 Redis 撤销投影未确认完成；调用方不能向用户承诺“被盗 access token 已失效”。

`202` 仍使用成功 envelope，body `code` 固定为 `200`；异步处理标识放在 `data.operationId`。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `operationId` | string | 是 | 安全 operation ID。 |
| `status` | string | 是 | 固定 `PROCESSING`。 |
| `retryAfterSeconds` | int | 是 | 建议前端轮询间隔。 |

`202` 响应仍应尽力清理当前浏览器的 `refresh_token` 和 `csrf_token` cookie。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2006` | `401` | 请先登录 | 缺少可信登录身份上下文。 |
| `2013` | `403` | CSRF 校验失败 | 缺少或不匹配 `X-CSRF-Token`。 |
| `2018` | `401` | 会话已失效 | 当前 session 已撤销或过期，且无法作为成功幂等处理。 |
| `2015` | `429` | 请求过于频繁 | 触发重复提交成本限制；不得阻断安全收敛补偿。 |
| `1004` | `503` | 服务暂时不可用 | DB 或安全投影依赖不可用，且无法创建 operation。 |

## 权限和可见性

- 只能撤销当前请求所属 session。
- Auth 不接收 `accountId`、`userId` 或 `sessionId` body 参数来决定目标，目标必须来自可信身份上下文。

## 排序、分页和过滤

无。

## 测试要求

- Handler contract test：已验证，覆盖 `200` 清 cookie。
- System HTTP test：待补。
