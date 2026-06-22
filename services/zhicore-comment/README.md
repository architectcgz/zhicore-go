# zhicore-comment

`zhicore-comment` 是评论服务的 Go 迁移模块。

服务职责：

- 拥有评论、回复、评论媒体引用、评论状态、评论统计和评论点赞。
- 提供评论列表、回复列表、增量查询、点赞状态和管理端评论操作。
- 通过事件通知 Content 更新文章评论计数，通过事件通知 Notification 或 Ranking 构建各自读模型。

数据归属：

- `comments`
- `comment_stats`
- `comment_likes`
- comment 服务自己的 `outbox_events`

迁移注意点：

- Comment 拥有评论树，Content 拥有文章。
- 创建评论前可以调用 Content 验证文章事实。
- 返回评论作者信息时调用 User 或使用明确的本地快照。
