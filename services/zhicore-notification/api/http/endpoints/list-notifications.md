# List Notifications

## 来源

- 服务总览：`docs/architecture/services/notification/README.md`
- 当前 API schema：`services/zhicore-notification/api/http/README.md`
- 实施计划：`docs/plan/impl-plan/2026-07-06-notification-module-foundation-implementation-plan.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/notifications` |
| 兼容别名 | 无 |
| Content-Type | 无 body |
| 鉴权 | 登录用户 |
| 幂等 | 查询接口，天然幂等 |

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `cursor` | string | 否 | 无 | 不透明游标，consumer 不解析。 |
| `size` | int | 否 | `20` | 每页数量，最大 `50`。 |
| `category` | string | 否 | 无 | `INTERACTION`、`CONTENT`、`SOCIAL`、`SYSTEM`、`SECURITY`。 |
| `unreadOnly` | bool | 否 | `false` | 为 `true` 时只返回仍有未读的聚合组。 |

## 成功响应 `data`

`data` 为 cursor page，`items` 每项是聚合通知组。正常路径优先来自 `notification_group_state`，发现缺失或计数不一致时允许回退 DB 聚合并记录 repair signal。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `type` | string | 是 | 通知类型，例如 `POST_LIKED`、`COMMENT_REPLIED`。 |
| `category` | string | 是 | 通知分类。 |
| `targetType` | string | 是 | 源对象类型，例如 `post`、`comment`、`user`。 |
| `targetId` | string | 是 | 源对象 opaque reference。 |
| `totalCount` | int | 是 | 聚合组总通知数。 |
| `unreadCount` | int | 是 | 聚合组未读数。 |
| `latestTime` | string | 是 | RFC3339 最新通知时间。 |
| `latestContent` | string | 是 | 最新展示内容快照。 |
| `recentActors` | array | 否 | 后续接入 User summary 后返回触发者摘要。 |
| `actorIds` | array | 是 | 首期返回最近触发者内部用户 ID 引用。 |
| `aggregatedContent` | object | 是 | 聚合展示快照，字段由 notification type 决定。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `cursor`、`size`、`category` 或 `unreadOnly` 非法。 |
| `2006` | `401` | 请先登录 | 缺少可信登录身份上下文。 |
| `1004` | `503` | 服务暂时不可用 | Notification DB 或缓存回源不可用。 |

## 权限和可见性

普通用户只能查询自己的通知聚合列表。Notification 不跨服务读取源对象正文，只返回本服务保存的展示快照和 opaque reference。

## 测试要求

- Handler contract test：待补。
- Application / repository test：任务 2 补齐聚合列表、回退和 repair signal 行为。
