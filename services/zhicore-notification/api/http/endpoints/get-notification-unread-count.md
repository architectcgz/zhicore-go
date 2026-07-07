# Get Notification Unread Count

## 来源

- 服务总览：`docs/architecture/services/notification/README.md`
- 当前 API schema：`services/zhicore-notification/api/http/README.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/notifications/unread-count` |
| 兼容别名 | `/api/v1/notifications/unread/count` |
| Content-Type | 无 body |
| 鉴权 | 登录用户 |
| 幂等 | 查询接口，天然幂等。 |

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `unreadCount` | int | 是 | 当前用户未读通知总数。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2006` | `401` | 请先登录 | 缺少可信登录身份上下文。 |
| `1004` | `503` | 服务暂时不可用 | Notification DB 或 Redis 回源不可用。 |

## 权限和可见性

只返回当前登录用户自己的未读总数。Redis key 使用 `notification:{userId}:*`，缓存缺失时回源 Notification DB。

## 测试要求

- Handler contract test：任务 2 补齐 canonical path 和 alias。
- Application / repository test：任务 2 补齐 cache-aside 与回源。
