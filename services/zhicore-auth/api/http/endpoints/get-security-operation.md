# Get Security Operation

## 来源

- 服务总览：`docs/architecture/services/auth/README.md`
- 模块 API 设计：`docs/architecture/module/auth/api.md`
- 数据模型：`docs/architecture/module/auth/data-model.md`
- 当前 API schema：`services/zhicore-auth/api/http/README.md`
- Go handler：`services/zhicore-auth/api/http/handler.go`
- Go contract test：`services/zhicore-auth/api/http/auth_handler_test.go`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/auth/security-operations/{operationId}` |
| 兼容别名 | 无 |
| Content-Type | 无 body |
| 鉴权 | 登录用户；Admin 后续走 Admin 审计权限 |
| 幂等 | 查询接口，天然幂等 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `operationId` | string | 是 | 安全 operation ID。 |

## Query 参数

无。

## Body 字段

无。

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `operationId` | string | 是 | 安全 operation ID。 |
| `type` | string | 是 | 操作类型，例如 `LOGOUT_CURRENT`、`REVOKE_SESSION`、`PASSWORD_CHANGE`、`ACCOUNT_BAN`。 |
| `status` | string | 是 | `PROCESSING`、`SUCCEEDED`、`FAILED`。 |
| `createdAt` | string | 是 | RFC3339 创建时间。 |
| `updatedAt` | string | 是 | RFC3339 更新时间。 |
| `completedAt` | string/null | 是 | RFC3339 完成时间；未完成时为 `null`。 |
| `retryAfterSeconds` | int/null | 是 | 建议轮询间隔；不需要继续轮询时为 `null`。 |
| `errorCode` | string/null | 是 | 可展示/可分支的错误类别；无错误时为 `null`。 |

不返回 Redis key、token、refresh token、access token `jti` 原值、cookie、Authorization header、内部异常堆栈或完整请求体。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `operationId` 缺失或格式非法。 |
| `2006` | `401` | 请先登录 | 缺少可信登录身份上下文。 |
| `1005` | `404` | 数据不存在 | operation 不存在，或不属于当前账号可见范围。 |
| `2005` | `403` | 权限不足 | Admin 查询未通过审计权限；普通用户通常用 `404` 隐藏不可见 operation。 |
| `2015` | `429` | 请求过于频繁 | 触发查询限流。 |
| `1004` | `503` | 服务暂时不可用 | Auth DB 不可用或查询降级失败。 |

## 权限和可见性

- 普通用户只能查询自己账号下的 security operation。
- Admin 查询目标账号 operation 必须走 Admin 权限和审计，首批不在本 endpoint 暴露跨账号查询能力。
- `FAILED` 只返回稳定 `errorCode`，不直接返回内部 Redis/DB 错误文本。

## 排序、分页和过滤

无。

## 测试要求

- Handler contract test：已验证，覆盖可见 operation 返回、`404` 和敏感字段不返回。
- System HTTP test：待补。
