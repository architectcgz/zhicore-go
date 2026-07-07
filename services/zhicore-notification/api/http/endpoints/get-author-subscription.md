# Get Author Subscription

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/author-subscriptions/{authorId}` |
| 兼容别名 | `/api/v1/notifications/author-subscriptions/{authorId}` |
| Content-Type | 无 body |
| 鉴权 | 登录用户 |
| 幂等 | 查询接口，天然幂等 |

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `authorId` | int | 是 | 作者用户 ID。 |
| `level` | string | 是 | `ALL`、`DIGEST_ONLY`、`MUTED`。 |
| `inAppEnabled` | bool | 是 | 站内通道。 |
| `websocketEnabled` | bool | 是 | WebSocket 通道。 |
| `emailEnabled` | bool | 是 | Email 通道。 |
| `digestEnabled` | bool | 是 | 摘要通道。 |
