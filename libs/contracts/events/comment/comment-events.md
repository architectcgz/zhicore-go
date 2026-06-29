# Comment Events

状态：草案。本文固定 `comment.*` 事件的字段级 payload contract，尚未由 Go outbox publisher / consumer contract test 验证。

## Envelope

所有事件使用 `docs/contracts/events.md` 的统一 envelope。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `eventId` | string | 是 | 全局唯一事件 ID，consumer 幂等键。 |
| `eventType` | string | 是 | `comment.created`、`comment.deleted`、`comment.liked` 或 `comment.unliked`。 |
| `payloadVersion` | int | 是 | 首版为 `1`。 |
| `producer` | string | 是 | 固定为 `zhicore-comment`。 |
| `occurredAt` | string | 是 | 业务事实发生时间，RFC3339。 |
| `aggregateType` | string | 是 | 固定为 `comment`。 |
| `aggregateId` | string | 是 | `commentId` 的字符串形式。 |
| `payload` | object | 是 | 事件 payload。 |

Comment 事件首版不携带 Content 内部 `post_id`。Ranking 需要把 payload 的 `postId` 当作 Content `public_id` 解析为内部 `post_id` 后再写 ledger / projection。

## 事件索引

| eventType / routing key | 触发事实 | Outbox | 主要 consumer |
| --- | --- | --- | --- |
| `comment.created` | 根评论或回复创建成功并可见 | 必须 | Content、Notification、Ranking |
| `comment.deleted` | 评论节点或子树从 `NORMAL` 变为 `DELETED` | 必须 | Content、Notification、Ranking |
| `comment.liked` | 用户点赞评论 | 必须 | Notification、Ranking |
| `comment.unliked` | 用户取消点赞评论 | 必须 | Ranking |

## `comment.created`

`payload`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `commentId` | int64 | 是 | Comment 内部评论 ID。 |
| `postId` | string | 是 | Content `public_id`。 |
| `postAuthorId` | int64 | 是 | 文章作者 User 内部 ID，用于通知和自评论过滤。 |
| `floor` | int64 | 是 | 文章内楼层号。 |
| `authorId` | int64 | 是 | 评论作者 User 内部 ID。 |
| `rootId` | int64 | 否 | 根评论 ID；根评论本身为空。 |
| `rootFloor` | int64 | 否 | 根评论楼层；根评论本身为空。 |
| `rootAuthorId` | int64 | 否 | 根评论作者；根评论本身为空。 |
| `parentId` | int64 | 否 | 直接父评论 ID；根评论本身为空。 |
| `parentFloor` | int64 | 否 | 直接父评论楼层；根评论本身为空。 |
| `parentAuthorId` | int64 | 否 | 直接父评论作者；根评论本身为空。 |
| `hasImages` | boolean | 是 | 评论是否包含图片。 |
| `hasVoice` | boolean | 是 | 评论是否包含语音。 |
| `createdAt` | string | 是 | 创建时间，RFC3339。 |

约束：

- 根评论：`rootId/rootFloor/rootAuthorId/parentId/parentFloor/parentAuthorId` 全部为空。
- 回复：`root*` 和 `parent*` 全部必填。
- Notification 不应为了创建评论 / 回复通知再同步回查 Comment 或 Content；本事件必须携带文章作者、根评论和直接父评论定位事实。
- Ranking 按本事件写 `COMMENT +1`，不读取评论正文。

## `comment.deleted`

`payload`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `commentId` | int64 | 是 | 被删除入口评论 ID。 |
| `postId` | string | 是 | Content `public_id`。 |
| `floor` | int64 | 是 | 被删除入口评论楼层。 |
| `rootId` | int64 | 否 | 所属根评论 ID；删除根评论时可为空或等于 `commentId`，以实现文档固定为准。 |
| `rootFloor` | int64 | 否 | 所属根评论楼层。 |
| `authorId` | int64 | 是 | 被删除入口评论作者 User 内部 ID。 |
| `deletedBy` | int64 | 否 | 删除操作者 User 内部 ID；系统任务可缺失。 |
| `deletedByRole` | string | 是 | `AUTHOR`、`ADMIN` 或 `SYSTEM`。 |
| `deleteReason` | string | 否 | 删除原因机器可读值或审计摘要。 |
| `deletedAt` | string | 是 | 删除时间，RFC3339。 |
| `isRoot` | boolean | 是 | 被删除入口是否根评论。 |
| `affectedCount` | int | 是 | 本次实际从 `NORMAL` 变为 `DELETED` 的评论数量，必须大于 `0`。 |

约束：

- 删除任意评论节点只发布一条 `comment.deleted`。
- 重复删除已删除评论不得再次写 outbox，也不得让 `affectedCount` 再次扣减。
- Content 和 Ranking 按 `affectedCount` 更新评论计数 / 热度指标。

## `comment.liked`

`payload`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `commentId` | int64 | 是 | Comment 内部评论 ID。 |
| `postId` | string | 是 | Content `public_id`。 |
| `floor` | int64 | 是 | 评论楼层。 |
| `commentAuthorId` | int64 | 是 | 评论作者 User 内部 ID。 |
| `likedBy` | int64 | 是 | 点赞用户 User 内部 ID。 |
| `occurredAt` | string | 是 | 点赞事实时间，RFC3339。 |

上游保证同一用户对同一评论重复点赞不重复发布本事件。Ranking 第一阶段默认不消费评论点赞；如产品要求计入文章热度，必须先扩展 Ranking 指标、权重、bucket、replay 和测试。

## `comment.unliked`

`payload`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `commentId` | int64 | 是 | Comment 内部评论 ID。 |
| `postId` | string | 是 | Content `public_id`。 |
| `floor` | int64 | 是 | 评论楼层。 |
| `commentAuthorId` | int64 | 是 | 评论作者 User 内部 ID。 |
| `unlikedBy` | int64 | 是 | 取消点赞用户 User 内部 ID。 |
| `occurredAt` | string | 是 | 取消点赞事实时间，RFC3339。 |

取消未点赞的幂等成功不发布本事件。
