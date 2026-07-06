# zhicore-notification

`zhicore-notification` 是通知服务的 Go 目标服务模块。

服务职责：

- 拥有通知收件箱、通知已读状态、未读数、通知聚合、投递台账、通知偏好、免打扰、作者订阅、广播 campaign、全局公告和小助手消息。
- 消费 User、Content、Comment、Message 等事件生成通知。
- 发布通知实时 fanout 事件或推送消息。

数据归属：

- `notifications`
- `notification_group_state`
- `notification_campaign`
- `notification_campaign_shard`
- `notification_delivery`
- `notification_user_preference`
- `notification_user_dnd`
- `notification_author_subscription`
- `global_announcements`
- `assistant_messages`

Go 设计注意点：

- Notification 不拥有触发通知的原始用户、文章、评论或私信。
- 通知 payload 可保存来源快照，但事实仍以来源服务为准。
- RabbitMQ 消费者必须支持重复投递和乱序事件。
- HTTP `notificationId` 使用 `notifications.public_id`，不暴露内部自增 `id`。
- 当前 Content 的公开 ID 生成仍是服务私有随机 `post_` 方案，不存在稳定可复用短 ID codec；Notification 首批切片先在 `internal/notification/infrastructure/publicid` 落本地短公开 ID codec。该实现只允许 Notification 服务复用，未来若提取到 `libs/kit/publicid`，必须先补跨服务算法、secret version、错误分类和迁移兼容设计。
- Notification public ID 形态固定为 `n` prefix、单字符算法版本、Base58 可逆置换结果和 4 字符 checksum；数据库列长度为 `VARCHAR(32)`。配置必须注入 active version 和版本化 secret，日志和错误不得输出 secret 或内部自增 ID。
