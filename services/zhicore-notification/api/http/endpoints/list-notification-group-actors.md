# List Notification Group Actors

状态：Contract 草案（字段已定稿，待实现）。本接口是聚合组详情的触发者全量视图；通知列表只返回 `recentActors` 的前三名。

## 来源

- 聚合组列表：`list-notifications.md`
- 服务 API 索引：`../README.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 路径 | `/api/v1/notification-groups/{groupId}/actors` |
| Content-Type | 无 body |
| 鉴权 | 登录用户 |
| 幂等 | 查询接口，天然幂等 |

### 参数

| 字段 | 位置 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `groupId` | path | string | 是 | - | Notification 聚合组公开 ID。 |
| `cursor` | query | string | 否 | 无 | 不透明游标，consumer 不解析。 |
| `size` | query | int | 否 | `20` | 每页去重触发者数，最大 `50`。 |

## 成功响应 `data`

`data` 是 cursor page：`{ items, nextCursor?, hasMore }`。按 `latestOccurredAt DESC, actor.publicId DESC` 稳定排序，每个用户最多出现一次。

```json
{
  "items": [
    {
      "actor": {
        "publicId": "user_8x7K2m",
        "displayName": "陈立",
        "avatarUrl": null
      },
      "eventCount": 2,
      "latestOccurredAt": "2026-07-10T05:30:00Z"
    }
  ],
  "nextCursor": "opaque-cursor",
  "hasMore": false
}
```

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `actor.publicId` | string | 是 | 触发者 User 公开 ID。 |
| `actor.displayName` | string | 是 | 事件发生时保存的展示名快照。 |
| `actor.avatarUrl` | string or `null` | 是 | 当前可用时的派生头像 URL；不可用或未保存时为 `null`。 |
| `eventCount` | int | 是 | 此用户在该组中触发的事件数，最小为 `1`。 |
| `latestOccurredAt` | string | 是 | 此用户最后一次触发该组事件的 UTC RFC3339 时间。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `groupId`、`cursor` 或 `size` 非法。 |
| `2006` | `401` | 请先登录 | 缺少可信登录身份上下文。 |
| `1005` | `404` | 数据不存在 | 聚合组不存在或不属于当前用户；两种情况使用相同响应，防止枚举。 |
| `1004` | `503` | 服务暂时不可用 | Notification DB 或必需依赖不可用。 |

## 权限和实现约束

- 查询必须以 `groupId + recipient_id` 限定；不得只按 `groupId` 查询。
- 返回的是去重用户列表，不是原始通知事件时间线。原始通知事件详情如有产品需求，应另设权限和隐私语义明确的 endpoint。
- 如果 User 后续不可访问，仍返回事件时的展示快照；本接口不承诺该用户资料页当前可访问。

## 测试要求

- Handler contract test：覆盖 owner、非 owner 的同样 `1005`、cursor、空页和 envelope。
- Application / repository test：覆盖同一 actor 多次事件合并、稳定排序、`eventCount` 和 `recentActors` 前三名与列表接口一致。
