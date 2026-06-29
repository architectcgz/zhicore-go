# Content Event Contract

本目录记录 `zhicore-content` 作为 producer 拥有的跨服务事件 payload contract。通用 envelope、RabbitMQ exchange、routing key、outbox 和兼容性规则见 `docs/contracts/events.md`。

## 事件文件

| 文件 | 范围 |
| --- | --- |
| [post-events.md](post-events.md) | `content.post.*` 文章生命周期、可见性、标签和互动事件。 |

## 公共约定

- Producer：`zhicore-content`。
- Exchange：`zhicore.events`。
- Routing key：等于 `eventType`。
- 关键事实事件必须通过 Content producer outbox 发布。
- `publicId` 是 Content 对外公开文章 ID，必填。
- `internalId` 是 Content 内部 `post_id BIGINT` opaque reference，必填；consumer 不能依赖它的生成方式或连续性。
