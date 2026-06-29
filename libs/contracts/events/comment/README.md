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
- `publicId` 是 Content `public_id` 字符串，必填，用于 HTTP 关联、审计和 repair。
- `internalId` 是 Content 内部 `post_id BIGINT` opaque reference，必填；Comment 在创建评论前通过 Content contract 校验文章并保存该引用。
- `commentId` 是 Comment 内部 `comments.id`，作为跨服务 opaque reference；外部 HTTP 使用由该内部 ID 派生的 `(postId, commentId)` 字符串定位评论。
