# 查询 outbox 事件

状态：已验证。本文从 `content-api.md` 拆出管理端 outbox 查询入口，已由 Go application / handler contract test 验证管理员角色、query DTO、分页响应、事件字段和核心错误码。

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
| 方法 | `GET` |
| 主路径 | `/api/v1/admin/content/outbox-events` |
| 兼容别名 | 无 |
| Content-Type | 无 |
| 鉴权 | 管理员，必须由 Gateway 注入 `X-User-Id` 和包含 `admin` 或 `ROLE_ADMIN` 的 `X-User-Roles` |
| 幂等 | 查询幂等 |

## Path 参数

无。

## Query 参数

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `status` | string | 是 | 不允许为空 | 仅支持 `failed` 或 `dead`，application 归一化为 `FAILED` / `DEAD`。 |
| `eventType` | string | 否 | 缺失表示不过滤事件类型 | 例如 `content.post.published`。 |
| `page` | int | 否 | 缺失由 application 使用默认页 | 从 `1` 开始。 |
| `size` | int | 否 | 缺失由 application 使用默认大小 | application 会限制最大值。 |

## 成功响应 `AdminOutboxListResp`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `items` | `OutboxEventItem[]` | 是 | 当前页 outbox 事件。 |
| `page` | int | 是 | 当前页码。 |
| `size` | int | 是 | 页大小。 |
| `total` | int | 是 | 符合条件的总数。 |

`OutboxEventItem`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `eventId` | string | 是 | outbox 事件 ID。 |
| `eventType` | string | 是 | 事件类型，也是 RabbitMQ routing key。 |
| `aggregateType` | string | 是 | 聚合类型，例如 `post`。 |
| `aggregateId` | string | 是 | 聚合公开 ID。 |
| `aggregateVersion` | int | 否 | 聚合版本；Content 发布事件使用文章 `postVersion`。 |
| `status` | string | 是 | `FAILED` 或 `DEAD`。 |
| `retryCount` | int | 是 | 当前尝试次数。 |
| `lastError` | string | 否 | 已脱敏的最近发布失败原因。 |
| `occurredAt` | string | 是 | 业务事件发生时间，RFC3339。 |
| `createdAt` | string | 是 | outbox 行创建时间，RFC3339。 |
| `updatedAt` | string | 是 | outbox 行更新时间，RFC3339。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2007` | `403` | 需要特定角色 | 缺少管理员角色。 |
| `1001` | `400` | 参数校验失败 | status、page 或 size 非法。 |
| `1004` | `503` | 服务暂时不可用 | PostgreSQL 等依赖不可用。 |

## 权限和可见性

- 只允许管理员查看 outbox 发布状态和最近失败原因。
- `lastError` 必须由下层写入前脱敏；HTTP 层不得暴露 RabbitMQ URL、账号、主机或底层 SQL。

## 排序、分页和过滤

- 当前排序由 repository 固定，默认按 outbox 更新时间或 ID 的稳定顺序返回。
- `status` 是必填过滤，避免管理端默认全表扫描。
- `eventType` 是可选低基数字段过滤。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `ListAdminOutboxEvents` |
| 聚合 | Outbox event |
| 事务边界 | 只读查询，不改变 outbox 状态。 |
| 事件 | 不产生新事件。 |

## 测试要求

- Handler contract test：`services/zhicore-content/api/http/admin_outbox_handler_test.go`。
- Application test：`services/zhicore-content/internal/content/application/admin_outbox_test.go`。
- Repository test：待 PostgreSQL admin outbox 查询实现时补充。
