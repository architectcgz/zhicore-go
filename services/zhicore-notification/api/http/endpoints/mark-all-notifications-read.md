# Mark All Notifications Read

## 来源

- 服务总览：`docs/architecture/services/notification/README.md`
- 当前 API schema：`services/zhicore-notification/api/http/README.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/notifications/read-all` |
| 兼容别名 | `/api/v1/notifications/mark-all-read` |
| Content-Type | 无 body |
| 鉴权 | 登录用户 |
| 幂等 | 重复调用成功，未读数保持为 0。 |

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `readAll` | bool | 是 | 固定为 `true`。 |
| `readAt` | string | 是 | RFC3339 本次全部已读时间。 |
| `affectedCount` | int | 是 | 本次从未读变为已读的通知数量。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2006` | `401` | 请先登录 | 缺少可信登录身份上下文。 |
| `1004` | `503` | 服务暂时不可用 | Notification DB 不可用。 |

## 权限和可见性

只影响当前登录用户自己的通知、group state 和 Redis key。实现必须删除 `notification:{userId}:*` 相关缓存。

## 测试要求

- Handler contract test：任务 2 补齐 canonical path 和 alias。
- Application / repository test：任务 2 补齐重复调用、group unread 清零和缓存失效。
