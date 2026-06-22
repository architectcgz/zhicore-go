# User 服务设计

## 事实来源

- Java `zhicore-user` controller：`AuthController`、`UserCommandController`、`UserQueryController`、`Follow*Controller`、`Block*Controller`、`CheckIn*Controller`、`AdminUser*Controller`。
- Java `database/init-all-databases.sql` 和 `docker/postgres-init/02-init-tables.sql` 的 user 表。
- `zhicore-client` 中 User 相关 Feign client 和 DTO。

## 职责边界

`zhicore-user` 拥有用户身份、用户资料、角色、关注、拉黑、签到和用户资料摘要查询。

User 不拥有文章、评论、通知或私信。用户主页里“某用户发表的文章”可以作为 User facade 路由存在，但必须委托 Content 查询。

## API 保留范围

必须保留以下 API 族：

- `/api/v1/auth`：注册、登录、当前用户、refresh。
- `/api/v1/users`：用户资料查询、简单资料批量查询、资料更新、私信设置。
- `/api/v1/users/{userId}/followers`、`following`、`follow-stats`、关注检查。
- `/api/v1/users/{userId}/blocking`：拉黑、取消拉黑、拉黑检查。
- `/api/v1/users/{userId}/check-in`：签到、统计、月度记录。
- `/api/v1/admin/users`：管理端用户查询、禁用、启用、token 失效。

字段级 request/response 需要后续从 Java DTO 提取到 `services/zhicore-user/api/http`。

## 数据归属

User 拥有：

- `users`
- `roles`
- `user_roles`
- `user_follows`
- `user_follow_stats`
- `user_blocks`
- `user_check_ins`
- `user_check_in_stats`
- User 服务自己的 `outbox_events`

内部主键使用 PostgreSQL sequence / identity。对外公开 ID 如需要隐藏数量增长，按 `docs/architecture/id-strategy.md` 单独设计。

## 事件

User 生产：

- `user.registered`
- `user.profile.updated`
- `user.followed`
- `user.unfollowed`

关键事件必须用 producer outbox，不能在业务提交后直接发 RabbitMQ。

User 消费其他服务事件不是第一阶段重点；如果要维护本地快照，必须在对应服务文档记录用途和失效方式。

## 跨服务依赖

- Upload：头像、图片资源只保存 `file_id` 或公开 URL 引用，文件事实仍归 Upload/File Service。
- Content：用户文章列表 facade 通过 Content contract 委托。
- Message：私信权限可以查询 User 的拉黑、私信设置和关注关系。

## Go 目标落点

- HTTP：`services/zhicore-user/api/http`
- Application：`services/zhicore-user/internal/user/application`
- Domain：`services/zhicore-user/internal/user/domain`
- Ports：`services/zhicore-user/internal/user/ports`
- Infrastructure：`services/zhicore-user/internal/user/infrastructure`
- Runtime：`services/zhicore-user/internal/user/runtime/module.go`

## 迁移风险

- Java 中部分 ID 依赖 `IdGeneratorFeignClient`；Go 目标不默认迁移该依赖。
- 用户资料更新会影响 Content 作者快照，必须通过事件和版本号处理，不允许 Content 直接读 User 数据库。
- Admin 用户操作必须走 User 的 command contract，不应复制用户状态变更逻辑。

## 下一步

- 提取 User HTTP 字段级 contract。
- 生成 User migration 草案。
- 设计注册、登录、资料更新、关注、拉黑、签到的行为测试。
