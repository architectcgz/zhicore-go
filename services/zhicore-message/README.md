# zhicore-message

`zhicore-message` 是私信服务的 Go 目标服务模块。

服务职责：

- 拥有私信会话、私信消息、已读状态、撤回状态和消息派发 outbox。
- 提供会话列表、会话详情、消息列表和私信未读数。
- 对接外部 IM 或 WebSocket 推送能力时，由 Message 自己封装 adapter。

数据归属：

- `conversations`
- `messages`
- `message_outbox_task`

Go 设计注意点：

- 私信未读数归 Message，通知未读数归 Notification，两者不能混为一个聚合。
- 发送私信前可以调用 User 判断拉黑关系、陌生人消息权限和用户摘要。
