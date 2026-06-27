# zhicore-user

`zhicore-user` 是用户服务的 Go 目标服务模块。

服务职责：

- 拥有用户身份、登录凭证、用户资料、头像引用、角色、启用/禁用状态和 JWT 签发/刷新行为。
- 拥有关注、粉丝、拉黑、签到和用户相关统计。
- 发布用户注册、关注、资料更新等事件。

数据归属：

- `users`
- `roles`
- `user_roles`
- `user_follows`
- `user_follow_stats`
- `user_blocks`
- `user_check_ins`
- `user_check_in_stats`
- user 服务自己的 `outbox_events`

Go 设计注意点：

- User 不拥有文章、评论、私信、通知或文件资源。
- 当前不提供 `GET /api/v1/users/{userId}/posts` 这类用户中心文章 facade；用户文章列表直接调用 Content 作者过滤接口。
