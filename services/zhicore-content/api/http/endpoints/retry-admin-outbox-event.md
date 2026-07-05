# 重试 outbox 事件

状态：已验证。本文从 `content-api.md` 拆出管理端 outbox retry 入口，已由 Go application / handler contract test 验证管理员角色、reason 校验、事件 ID 映射、审计上下文和成功响应。

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- 事件契约：`docs/contracts/events.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`
- Go handler：`services/zhicore-content/api/http/handler.go`
- Go contract test：`services/zhicore-content/api/http/admin_outbox_handler_test.go`
- Application test：`services/zhicore-content/internal/content/application/admin_outbox_test.go`
- 大草案：`services/zhicore-content/api/http/endpoints/content-api.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/admin/content/outbox-events/{eventId}/retry` |
| 兼容别名 | 无 |
| Content-Type | `application/json` |
| 鉴权 | 管理员，必须由 Gateway 注入 `X-User-Id` 和包含 `admin` 或 `ROLE_ADMIN` 的 `X-User-Roles` |
| 幂等 | 受 outbox 状态条件约束；重复 retry 不得重复写业务事实 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `eventId` | string | 是 | outbox 事件 ID。 |

## Query 参数

无。

## Body 字段 `AdminOutboxRetryReq`

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `reason` | string | 是 | 空白字符串非法 | 管理员重试原因，进入 retry 审计上下文。 |

## 成功响应 `AdminOutboxRetryResp`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `eventId` | string | 是 | outbox 事件 ID。 |
| `status` | string | 是 | 重试后状态，通常为 `PENDING`。 |
| `retryCount` | int | 是 | retry 前已累计的发布尝试次数。 |
| `retriedAt` | string | 是 | 管理员触发 retry 的时间，RFC3339。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2007` | `403` | 需要特定角色 | 缺少管理员角色。 |
| `1001` | `400` | 参数校验失败 | eventId 或 reason 缺失。 |
| `1005` | `404` | 数据不存在 | 目标 outbox 事件不存在或当前状态不可 retry。 |
| `1004` | `503` | 服务暂时不可用 | PostgreSQL 等依赖不可用。 |

## 权限和可见性

- 只允许管理员手动重试 outbox 事件。
- retry 只改变 outbox dispatch 状态，不重写文章业务事实，不重新生成业务 payload。
- 管理员 ID、原因和时间必须传入 application 端口，用于 `outbox_retry_audit` 或等价审计。

## 排序、分页和过滤

无。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `RetryAdminOutboxEvent` |
| 聚合 | Outbox event |
| 事务边界 | repository 必须用状态条件把 `FAILED` / `DEAD` 事件重置为可 dispatch 状态，并写审计上下文。 |
| 事件 | 不产生新的业务事件；后续 dispatcher 会发布原 outbox payload。 |

## 测试要求

- Handler contract test：`services/zhicore-content/api/http/admin_outbox_handler_test.go`。
- Application test：`services/zhicore-content/internal/content/application/admin_outbox_test.go`。
- Repository test：待 PostgreSQL admin outbox retry 实现时补充状态条件和审计写入。
