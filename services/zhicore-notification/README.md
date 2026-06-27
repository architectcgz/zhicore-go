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
