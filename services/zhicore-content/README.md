# zhicore-content

`zhicore-content` 是内容服务的 Go 迁移模块。

服务职责：

- 拥有文章、草稿、文章内容、发布生命周期、定时发布、删除恢复、标签、分类和话题引用。
- 拥有文章点赞、收藏、统计、作者快照和内容服务内部读模型。
- 发布内容相关事件，供 Search、Ranking、Notification、Comment 等服务消费。

数据归属：

- `posts`
- `post_stats`
- `post_likes`
- `post_favorites`
- `categories`
- `tags`
- `post_tags`
- `tag_stats`
- `scheduled_publish_event`
- `outbox_event`
- `outbox_retry_audit`
- `consumed_events`
- `domain_event_task`

迁移注意点：

- 用户资料归 User，`posts` 中的作者昵称和头像只是 Content 拥有的快照。
- 文件资源归 Upload，Content 只保存 `file_id`。
- 查询某个用户发表的文章由 Content 提供权威查询，User 只能做 facade。
