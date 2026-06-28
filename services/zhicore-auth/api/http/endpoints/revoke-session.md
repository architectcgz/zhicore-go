# Revoke Session

## 来源

- 服务总览：`docs/architecture/services/auth/README.md`
- 模块 API 设计：`docs/architecture/module/auth/api.md`
- 模块 service 设计：`docs/architecture/module/auth/service.md`
- Redis key 设计：`docs/architecture/module/auth/redis-keys.md`
- 当前 API schema：`services/zhicore-auth/api/http/README.md`
- Go handler：待实现
- Go contract test：待补

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `DELETE` |
| 主路径 | `/api/v1/auth/sessions/{sessionId}` |
| 兼容别名 | 无 |
| Content-Type | 无 body |
| 鉴权 | 登录用户 |
| 幂等 | 对同一目标 session 重复撤销应尽量返回同一最终语义 |

## Header

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `X-CSRF-Token` | 是 | 变更 session 的浏览器请求必须提交；Auth 校验它与 `csrf_token` cookie 一致。 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `sessionId` | string | 是 | 目标 refresh session ID。 |

## Query 参数

无。

## Body 字段

无。

## 成功响应 `data`

HTTP `200` 表示目标 session 的 DB revoke 和 Gateway 可见撤销投影均已完成。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `status` | string | 是 | 固定 `REVOKED`。 |
| `sessionId` | string | 是 | 被撤销的目标 session ID。 |
| `current` | boolean | 是 | 目标是否为当前请求所属 session。 |

如果 `sessionId` 等于当前 session，本 endpoint 语义等同撤销当前 session，并必须清理当前响应的 `refresh_token` 和 `csrf_token` cookie。

## 处理中响应 `data`

HTTP `202` 表示 DB revoke 已提交或安全操作已受理，但 Redis 撤销投影未确认完成。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `operationId` | string | 是 | 安全 operation ID。 |
| `status` | string | 是 | 固定 `PROCESSING`。 |
| `retryAfterSeconds` | int | 是 | 建议前端轮询间隔。 |
| `sessionId` | string | 是 | 目标 session ID。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `sessionId` 缺失或格式非法。 |
| `2006` | `401` | 请先登录 | 缺少可信登录身份上下文。 |
| `2013` | `403` | CSRF 校验失败 | 缺少或不匹配 `X-CSRF-Token`。 |
| `1005` | `404` | 数据不存在 | 目标 session 不属于当前账号、已被清理或不存在；不得暴露其他账号 session 存在性。 |
| `2015` | `429` | 请求过于频繁 | 触发重复提交成本限制；不得阻断安全收敛补偿。 |
| `1004` | `503` | 服务暂时不可用 | DB 或安全投影依赖不可用，且无法创建 operation。 |

## 权限和可见性

- 用户只能撤销自己账号下的 session。
- 目标 session 不属于当前账号时返回 `404`，不返回 `403`。
- 不返回目标 session 的 refresh token、token hash、access token `jti`、完整 IP 或完整 User-Agent。

## 排序、分页和过滤

无。

## 测试要求

- Handler contract test：待补，覆盖撤销其他 session、撤销当前 session 清 cookie、非本人 session 返回 `404`、`202 PROCESSING`。
- System HTTP test：待补。
