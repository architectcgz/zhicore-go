# zhicore-user

`zhicore-user` 是用户资料和用户关系服务的 Go 目标服务模块。

服务职责：

- 拥有用户公开资料、头像引用、陌生人消息设置、关注、拉黑、签到和用户相关统计。
- 不拥有账号凭证、密码 hash、角色事实、启用/禁用账号状态或 JWT 签发/刷新行为；这些归 `zhicore-auth`。
- 发布用户资料创建、关注、资料更新等事件。

数据归属：

- `users`
- `user_follows`
- `user_follow_stats`
- `user_blocks`
- `user_check_ins`
- `user_check_in_stats`
- user 服务自己的 `outbox_events`

Go 设计注意点：

- User 不拥有文章、评论、私信、通知或文件资源。
- User 不拥有账号、凭证、角色或 token 生命周期。
- 当前不提供 `GET /api/v1/users/{userId}/posts` 这类用户中心文章 facade；用户文章列表直接调用 Content 作者过滤接口。
- 运行期 resilience 和限流设计见 `docs/architecture/module/user/runtime-resilience.md`、`docs/architecture/module/user/rate-limiting.md`。
