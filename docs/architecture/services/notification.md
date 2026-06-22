# Notification 服务设计

## 事实来源

- Java `zhicore-notification` controller：Notification command/query、delivery、preference。
- `zhicore-notification-platform-design.md`
- `notification-phase1-implementation-status.md`
- Java 全量初始化 SQL 中 notification 表族。

## 职责边界

`zhicore-notification` 拥有通知收件箱、通知聚合状态、未读数、偏好、免打扰、作者订阅、广播任务和通道投递记录。

Notification 不拥有触发通知的源事实，例如文章、评论、关注或私信。它消费源服务事件生成通知读模型。

## API 保留范围

必须保留以下 API 族：

- `/api/v1/notifications`：通知列表、未读数、未读 breakdown、标记单条已读、全部已读。
- `/api/v1/notification-preferences`：通知偏好查询与更新。
- `/api/v1/notification-authors/{authorId}/subscription`：作者订阅偏好。
- `/api/v1/notification-dnd`：免打扰配置。
- `/api/v1/notifications/deliveries`：投递记录查询和重试。

现有别名如 `/unread/count` 和 `/unread-count`、`/read-all` 和 `/mark-all-read` 必须保留。

## 数据归属

Notification 拥有：

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

Redis 只保存未读数、聚合列表和偏好缓存，不是真相源。

## 内部模块

目标服务内建议拆分：

- Notification Center：查询、已读、未读、聚合视图。
- Preference Service：类型、通道、作者订阅和免打扰。
- Interaction Pipeline：点赞、评论、回复、关注等点对点通知。
- Broadcast Pipeline：作品发布、公告、活动类通知。
- Channel Delivery：站内、WebSocket/App Push、邮件/短信预留。
- Delivery Ledger：投递结果、重试、幂等和审计。

## 事件

Notification 消费：

- `content.post.published`
- `content.post.liked`
- `comment.created`
- `comment.deleted`
- `user.followed`
- `user.profile.updated`

Notification 生产的内部事件默认不提升到跨服务 contract，除非其他服务确实需要消费。

## 一致性与幂等

- PostgreSQL 是通知真相源。
- Redis 未读数和聚合缓存可以重建。
- Push、邮件、短信是 best-effort delivery，不决定站内通知是否存在。
- 幂等至少基于 `event_id`、`dedupe_key` 和 `recipient_id + channel + dedupe_key`。

## Go 目标落点

- HTTP：`services/zhicore-notification/api/http`
- Application：`services/zhicore-notification/internal/notification/application`
- Domain：`services/zhicore-notification/internal/notification/domain`
- Ports：`services/zhicore-notification/internal/notification/ports`
- Infrastructure：`postgres`、`redis`、`rabbitmq`、`clients`
- Runtime：`services/zhicore-notification/internal/notification/runtime/module.go`

## 迁移风险

- 高粉作者发布作品会产生 fan-out 风暴，不能用“逐粉丝同步写库”实现。
- 偏好和免打扰判断如果每次回源，会把通知消费链路拖慢，需要缓存和批量读取。
- 未读数容易漂移，必须设计回源和重建任务。
- 站内通知和 Push 不能耦合在同一个事务结果里。

## 下一步

- 提取 Notification HTTP 字段级 contract。
- 生成通知表族 migration 草案。
- 先实现交互通知，再实现 campaign/shard 广播链路。
