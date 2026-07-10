# List Notifications

状态：Contract 草案（字段已定稿，待 handler、查询模型和 contract test 实现）。本草案替换首批开发中的 `targetId`、`actorIds` 和无 `groupId` 聚合项形态；该首批接口尚未对外发布，因此不保留该开发中形态。

## 来源

- 服务总览：`docs/architecture/services/notification/README.md`
- 当前 API schema：`services/zhicore-notification/api/http/README.md`
- 前端消费规则：`zhicore-frontend-vue/docs/design/pages/notification.md`（独立前端仓）
- 聚合触发者分页：`list-notification-group-actors.md`
- 聚合组已读：`mark-notification-group-read.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 路径 | `/api/v1/notifications` |
| Content-Type | 无 body |
| 鉴权 | 登录用户 |
| 幂等 | 查询接口，天然幂等 |

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `cursor` | string | 否 | 无 | 不透明游标，consumer 不解析。 |
| `size` | int | 否 | `20` | 每页聚合组数，最大 `50`。 |
| `category` | string | 否 | 无 | `INTERACTION`、`CONTENT`、`SOCIAL`、`SYSTEM`、`SECURITY`。 |
| `unreadOnly` | bool | 否 | `false` | `true` 时只返回 `unreadCount > 0` 的聚合组。 |

## 成功响应 `data`

`data` 为 cursor page：`{ items, nextCursor?, hasMore }`。聚合组按 `latestOccurredAt DESC, groupId DESC` 稳定排序；空列表必须返回 `[]`，不返回 `null`。

```json
{
  "items": [
    {
      "groupId": "ng1F7qK2m",
      "type": "POST_LIKED",
      "category": "INTERACTION",
      "totalCount": 5,
      "unreadCount": 2,
      "actorTotalCount": 4,
      "latestOccurredAt": "2026-07-10T05:30:00Z",
      "content": {
        "title": "新的互动",
        "body": "陈立等 4 人赞了你的文章"
      },
      "recentActors": [
        {
          "publicId": "user_8x7K2m",
          "displayName": "陈立",
          "avatarUrl": "https://cdn.example/avatar.png"
        }
      ],
      "target": {
        "resource": { "type": "POST", "id": "post_9xk2" },
        "anchor": { "type": "COMMENT", "id": "comment_3d" },
        "snapshot": {
          "title": "信息架构的边界",
          "excerpt": "..."
        }
      }
    }
  ],
  "nextCursor": "opaque-cursor",
  "hasMore": true
}
```

### 聚合组字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `groupId` | string | 是 | Notification 生成的稳定、不透明公开聚合组 ID。可作为 Vue key、actor 分页和组级已读 path 参数；不得返回内部 `group_key`。 |
| `type` | string | 是 | 通知类型，例如 `POST_LIKED`、`COMMENT_REPLIED`、`USER_FOLLOWED`。未知值必须允许前端降级展示。 |
| `category` | string | 是 | 通知分类。 |
| `totalCount` | int | 是 | 组内通知事件总数，不等同于触发用户数。 |
| `unreadCount` | int | 是 | 组内当前未读通知事件数，范围 `0..totalCount`。 |
| `actorTotalCount` | int | 是 | 组内去重后的触发用户数；“等 N 人”只使用此字段。系统通知为 `0`。 |
| `latestOccurredAt` | string | 是 | 最新事件发生时间，UTC RFC3339。 |
| `content` | object | 是 | Notification 保存的展示快照，包含非空 `title`、`body`。它是历史通知展示事实，不是目标资源当前事实。 |
| `recentActors` | array | 是 | 按最近触发时间倒序的去重用户快照，长度 `0..3`；永远不以 `null` 表示空值。 |
| `target` | object or `null` | 是 | 可导航资源的语义描述符；系统类或没有资源目标的通知为 `null`。后端不返回前端 URL。 |

`recentActors` 的每项为 `{ publicId, displayName, avatarUrl? }`。`publicId` 和 `displayName` 是事件发生时保存的 User 展示快照；`avatarUrl` 是可过期的派生展示 URL，可省略或为 `null`。前端只展示前三项，查看完整触发者集合必须调用 actor 分页接口，不能把 `recentActors` 当作全量数据。

### `target` 描述符

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `resource.type` | string | 是 | 主资源类型，例如 `POST`、`USER`、`CONVERSATION`。 |
| `resource.id` | string | 是 | 主资源的公开 ID；不得使用数据库内部 ID 或 Notification 私有 opaque reference。 |
| `anchor` | object | 否 | 主资源内的定位点，例如 `{ type: "COMMENT", id: "comment_3d" }`。 |
| `snapshot` | object | 是 | Notification 保存的目标展示快照；允许 `title?`、`excerpt?`、`coverUrl?`，均不作为目标页的当前事实。 |

前端在单一 route registry 中把 `resource.type` 映射为自己的路由，并使用 `anchor` 生成片段或页面内定位。例如 `POST + COMMENT` 映射为 `/posts/{postId}#comment-{commentId}`。未识别的资源类型必须保留通知展示，但禁用“打开目标”；不得猜测 URL。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `cursor`、`size`、`category` 或 `unreadOnly` 非法。 |
| `2006` | `401` | 请先登录 | 缺少可信登录身份上下文。 |
| `1004` | `503` | 服务暂时不可用 | Notification DB、聚合读模型或必需的展示快照依赖不可用。 |

## 权限、可见性和实现约束

- 只返回当前登录用户的聚合组；`groupId` 在响应和后续请求中都必须再次按 `recipient_id` 限定。
- `groupId` 必须独立于现有内部 `group_key`。推荐给 `notification_group_state` 增加公开 ID，并在 inbox fallback 聚合时稳定解析到同一公开 ID。
- `target.resource.id`、`target.anchor.id` 和 `recentActors[].publicId` 必须由来源服务的公开 ID 提供。事件 payload 必须携带这些稳定事实及所需展示快照；Notification 不得用内部 `int64` 直接作为 HTTP 字段，也不应在列表请求中同步 N+1 查询 User。
- 源资源在事件后被删除、隐藏或变更权限时，列表仍可展示 `content` / `target.snapshot`；用户点击后由目标服务重新校验可见性并展示实际结果。

## 测试要求

- Handler contract test：覆盖完整聚合组 JSON、最多三名且去重的 `recentActors`、`target: null`、未知 `type` 和 envelope。
- Application / repository test：覆盖 `groupId` 稳定性、`totalCount` 与 `actorTotalCount` 区分、按公开 ID 返回 target、cursor 稳定排序，以及 group-state 缺失时 inbox fallback 仍返回同一组 ID。
