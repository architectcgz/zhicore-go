# Update Notification Preferences

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `PUT` |
| 主路径 | `/api/v1/notification-preferences` |
| 兼容别名 | `/api/v1/notifications/preferences` |
| Content-Type | `application/json` |
| 鉴权 | 登录用户 |
| 幂等 | 同一 payload 重复提交结果一致 |

## Request Body

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `preferences` | array | 是 | 要替换保存的偏好列表。 |
| `preferences[].notificationType` | string | 是 | 通知类型。 |
| `preferences[].channels.inApp` | bool | 是 | 站内通道。 |
| `preferences[].channels.websocket` | bool | 是 | WebSocket 通道。 |
| `preferences[].channels.email` | bool | 是 | Email 通道。 |
| `preferences[].channels.sms` | bool | 是 | 第一阶段禁止设置为 `true`。 |

## 规则

- `SMS` 第一阶段保留字段但禁止启用，传 `true` 返回参数错误。
- 提交成功后失效 `notification:{userId}:preferences`。

## 错误响应

| code | HTTP status | 触发条件 |
| --- | --- | --- |
| `1001` | `400` | payload 非法、通知类型为空或启用 SMS。 |
| `2006` | `401` | 缺少可信登录身份上下文。 |
| `1004` | `503` | Notification DB 不可用。 |
