# Comment Event Contract

本目录记录 `zhicore-comment` 作为 producer 拥有的跨服务事件 payload contract。通用 envelope、RabbitMQ exchange、routing key、outbox 和兼容性规则见 `docs/contracts/events.md`。

## 事件文件

| 文件 | 范围 |
| --- | --- |
| [comment-events.md](comment-events.md) | `comment.*` 评论创建、删除、点赞和取消点赞事件。 |

## 公共约定

- Producer：`zhicore-comment`。
- Exchange：`zhicore.events`。
- Routing key：等于 `eventType`。
- `postId` 是 Content `public_id` 字符串，Comment 不保存也不发布 Content 内部 `post_id`。
- `commentId` 是 Comment 内部 `comments.id`，作为跨服务 opaque reference；外部 HTTP 仍可用 `(postId, floor)` 定位评论。
