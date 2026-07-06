# Update Notification DND

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `PUT` |
| 主路径 | `/api/v1/notification-dnd` |
| 兼容别名 | `/api/v1/notifications/dnd` |
| Content-Type | `application/json` |
| 鉴权 | 登录用户 |
| 幂等 | 同一 payload 重复提交结果一致 |

## Request Body

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `enabled` | bool | 是 | 是否启用。 |
| `startTime` | string | 是 | `HH:MM`，不得等于 `endTime`。 |
| `endTime` | string | 是 | `HH:MM`；小于 `startTime` 表示跨日窗口。 |
| `timezone` | string | 是 | IANA timezone。 |
| `categories` | array | 否 | 空数组表示不按分类收窄。 |
| `channels` | array | 否 | 空数组表示不按通道收窄。 |

## 规则

- `startTime == endTime` 返回参数错误。
- 免打扰只影响主动通道，不删除站内通知事实。
- 提交成功后失效 `notification:{userId}:dnd`。
