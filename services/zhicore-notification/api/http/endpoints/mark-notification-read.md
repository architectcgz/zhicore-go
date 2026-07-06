# Mark Notification Read

## 来源

- 服务总览：`docs/architecture/services/notification/README.md`
- 当前 API schema：`services/zhicore-notification/api/http/README.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/notifications/{notificationId}/read` |
| 兼容别名 | 无 |
| Content-Type | 无 body |
| 鉴权 | 登录用户 |
| 幂等 | 重复标记同一通知已读成功且不重复扣减未读数。 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `notificationId` | string | 是 | Notification `public_id`，不是内部 `BIGINT id`。 |

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `notificationId` | string | 是 | 已读通知 public ID。 |
| `read` | bool | 是 | 固定为 `true`。 |
| `readAt` | string | 是 | RFC3339 已读时间；重复已读返回原已读时间。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `notificationId` 缺失、prefix/version/checksum 非法或超长。 |
| `2006` | `401` | 请先登录 | 缺少可信登录身份上下文。 |
| `1005` | `404` | 数据不存在 | 通知不存在或不属于当前用户可见范围。 |
| `1004` | `503` | 服务暂时不可用 | Notification DB 不可用。 |

## 权限和可见性

application 必须按 `public_id + recipient_id` 限定更新，不能只按内部 ID 或 public ID 标记已读。

## 测试要求

- Handler contract test：任务 2 补齐 public ID 参数错误和 envelope。
- Application / repository test：任务 2 补齐权限、幂等和 group unread 非负。
