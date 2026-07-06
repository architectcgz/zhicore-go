# Get Notification DND

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/notification-dnd` |
| 兼容别名 | `/api/v1/notifications/dnd` |
| Content-Type | 无 body |
| 鉴权 | 登录用户 |
| 幂等 | 查询接口，天然幂等 |

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `enabled` | bool | 是 | 是否启用免打扰。 |
| `startTime` | string | 是 | `HH:MM` 本地时间。 |
| `endTime` | string | 是 | `HH:MM` 本地时间，可早于 `startTime` 表示跨日。 |
| `timezone` | string | 是 | IANA timezone，例如 `Asia/Shanghai`。 |
| `categories` | array | 是 | 适用通知分类。 |
| `channels` | array | 是 | 适用主动通道。 |
