# Mark Notification Group Read

状态：Contract 草案（字段已定稿，待实现）。列表和详情页打开聚合组时使用本接口；它不能用某条“最新通知”的已读替代。

## 来源

- 聚合组列表：`list-notifications.md`
- 服务 API 索引：`../README.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 路径 | `/api/v1/notification-groups/{groupId}/read` |
| Content-Type | 无 body |
| 鉴权 | 登录用户 |
| 幂等 | 重复标记同一聚合组已读成功，不重复扣减未读数。 |

### Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `groupId` | string | 是 | Notification 聚合组公开 ID。 |

## 成功响应 `data`

```json
{
  "groupId": "ng1F7qK2m",
  "read": true,
  "changedCount": 2,
  "unreadCount": 0,
  "readAt": "2026-07-10T05:31:00Z"
}
```

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `groupId` | string | 是 | 已处理的聚合组公开 ID。 |
| `read` | bool | 是 | 固定为 `true`。 |
| `changedCount` | int | 是 | 此次从未读变为已读的通知事件数；重复调用时为 `0`。 |
| `unreadCount` | int | 是 | 命令完成后的组内未读数。并发新通知可使其大于 `0`，前端不得强制写为 `0`。 |
| `readAt` | string | 是 | 本次命令处理时间，UTC RFC3339。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `groupId` 格式非法。 |
| `2006` | `401` | 请先登录 | 缺少可信登录身份上下文。 |
| `1005` | `404` | 数据不存在 | 聚合组不存在或不属于当前用户。 |
| `1004` | `503` | 服务暂时不可用 | Notification DB 或未读计数依赖不可用。 |

## 权限和并发语义

- 更新必须在一个事务中按 `recipient_id + groupId` 定位组，再只更新该组当前未读的通知事件；不得按全局 `groupId` 或内部 `group_key` 裸更新。
- `notification_stats`、`notification_group_state.unread_count` 与通知行状态必须在同一事务内保持非负一致；返回的 `unreadCount` 是事务后实际值。
- 前端可先用响应更新该组未读数，并用 `changedCount` 调整已知总数；收到实时未读提示或状态未知时，必须重新请求 unread count / breakdown。

## 测试要求

- Handler contract test：覆盖 public `groupId`、幂等重复调用、owner 限定、404 防枚举和 envelope。
- Application / repository test：覆盖含多条未读的聚合组一次清零、并发新通知保留未读、group / category / total 未读计数不为负。
