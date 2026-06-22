# Comment 服务设计

## 事实来源

- Java `zhicore-comment` controller：comment command/query、like、media、admin、outbox admin。
- Java `database/init-all-databases.sql` 中 comment 表。
- Ranking、Notification 对评论事件的消费设计。

## 职责边界

`zhicore-comment` 拥有评论、回复、评论树、评论状态、评论统计、评论点赞和评论媒体引用。

Comment 不拥有文章、用户资料、文件存储或榜单分数。

## API 保留范围

必须保留以下 API 族：

- `/api/v1/comments`：创建、更新、删除、详情。
- `/api/v1/comments/post/{postId}`：文章评论分页、游标、增量查询。
- `/api/v1/comments/{commentId}/replies`：回复分页、游标、增量查询。
- `/api/v1/comments/{commentId}/like`：点赞、取消点赞。
- `/api/v1/comments/{commentId}/liked`、`like-count`、`batch/liked`：点赞状态和计数查询。
- `/api/v1/comments/media`：评论图片和语音上传入口。
- `/api/v1/admin/comments`：管理端评论查询和删除。
- `/api/v1/admin/comment-outbox`：outbox summary 和 dead retry。

## 数据归属

Comment 拥有：

- `comments`
- `comment_stats`
- `comment_likes`
- Comment 服务自己的 `outbox_events`

评论图片、语音资源只保存 `file_id` 或资源引用；文件事实归 Upload/File Service。

## 主写流程

评论创建：

1. 调用 Content contract 校验文章存在、可评论和可见性。
2. 调用 User contract 获取作者摘要或校验用户状态。
3. 在 Comment 本地事务内写 `comments`、必要统计和 outbox event。
4. 事务后删除相关缓存。

评论删除：

1. 权限判断在 application 层完成。
2. 本地事务修改评论状态或删除标记。
3. 写 `comment.deleted` outbox event。

点赞/取消点赞：

1. 用 `(comment_id, user_id)` 唯一约束保证幂等。
2. 同事务更新 `comment_stats.like_count`。
3. 同事务写 interaction event。

## 事件

Comment 生产：

- `comment.created`
- `comment.deleted`
- `comment.liked`
- `comment.unliked`

这些事件供 Content 更新文章评论计数，Ranking 更新热度，Notification 生成通知。

## 跨服务依赖

- Content：创建评论前校验文章事实。
- User：评论作者资料摘要、权限和拉黑关系。
- Upload：评论媒体上传和文件 URL 解析。
- Ranking：Comment 可以查询热门候选，但不能拥有分数。

## Go 目标落点

- HTTP：`services/zhicore-comment/api/http`
- Application：`services/zhicore-comment/internal/comment/application`
- Domain：`services/zhicore-comment/internal/comment/domain`
- Ports：`services/zhicore-comment/internal/comment/ports`
- Infrastructure：`postgres`、`redis`、`rabbitmq`、`clients`
- Runtime：`services/zhicore-comment/internal/comment/runtime/module.go`

## 迁移风险

- Java 历史中部分评论事件存在直接 MQ 路径，Go 侧关键评论事件必须补齐 producer outbox。
- 评论列表、回复列表、增量查询的排序和游标语义必须逐项保留，否则前端加载更多会出错。
- 媒体上传入口如果只是转发 Upload，需要明确文件归属不转移。

## 下一步

- 提取评论 API 字段级 contract 和分页语义。
- 生成 Comment migration 草案。
- 补创建、删除、点赞、游标分页、outbox 幂等测试。
