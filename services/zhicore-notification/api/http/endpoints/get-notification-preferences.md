# Get Notification Preferences

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/notification-preferences` |
| 兼容别名 | `/api/v1/notifications/preferences` |
| Content-Type | 无 body |
| 鉴权 | 登录用户 |
| 幂等 | 查询接口，天然幂等 |

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `userId` | int | 是 | 当前用户 ID。 |
| `preferences` | array | 是 | 按通知类型返回通道偏好。 |
| `preferences[].notificationType` | string | 是 | 通知类型，例如 `POST_LIKED`。 |
| `preferences[].channels.inApp` | bool | 是 | 站内通知是否启用。 |
| `preferences[].channels.websocket` | bool | 是 | WebSocket 是否启用。 |
| `preferences[].channels.email` | bool | 是 | Email 是否启用。 |
| `preferences[].channels.sms` | bool | 是 | SMS 第一阶段始终不可启用。 |

## 错误响应

| code | HTTP status | 触发条件 |
| --- | --- | --- |
| `2006` | `401` | 缺少可信登录身份上下文。 |
| `1004` | `503` | Notification DB 不可用。 |
