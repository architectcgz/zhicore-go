# Content 服务设计

## 事实来源

- Java `zhicore-content` controller：Post command/query、like/favorite、tag、admin、outbox、reader presence。
- Java `content-service-design.md`、`content-visibility-and-projection-evolution.md`、`post-reading-presence.md`。
- Java `zhicore-content/src/main/resources/db/schema.sql`。
- `zhicore-client` 和 `zhicore-integration` 中 post 事件与 DTO。

## 职责边界

`zhicore-content` 拥有文章主数据、文章发布生命周期、标签、分类、话题引用、文章互动写模型、文章统计、作者快照和内容服务内部投影。

Content 不拥有用户资料事实、评论树、搜索索引、热榜分数或通知收件箱。

## API 保留范围

必须保留以下 API 族：

- `/api/v1/posts`：创建、更新、发布、取消发布、定时发布、删除、恢复、草稿、列表、详情、作者文章、游标和批量查询。
- `/api/v1/posts/{postId}/like`、`favorite`：点赞、取消点赞、收藏、取消收藏、状态和计数查询。
- `/api/v1/posts/{postId}/content`、`draft`：正文和草稿读取。
- `/api/v1/posts/{postId}/tags`：文章标签读写。
- `/api/v1/posts/{postId}/readers`：阅读 presence session、leave、presence 查询。
- `/api/v1/tags`：标签详情、列表、搜索、热门和标签文章。
- `/api/v1/admin/posts`：管理端文章查询和删除。
- `/api/v1/admin/outbox`：outbox dead/failed 查询和 retry。

## 数据归属

PostgreSQL：

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
- `domain_event_task`
- `outbox_retry_audit`
- `consumed_events`

MongoDB：

- 文章正文或富文档投影。

Redis：

- 文章详情缓存。
- 点赞/收藏状态和计数缓存。
- 阅读 presence 短生命周期状态。

作者快照字段如 `owner_name`、`owner_avatar_id`、`owner_profile_version` 保留为列，不默认改成 JSON。原因是这些字段需要参与补偿、查询、索引和精确更新。

## 主写流程

文章创建、编辑、发布、删除、恢复由 application use case 拥有事务边界：

```text
api/http -> application command -> postgres repository
                         -> domain_event_task
                         -> outbox_event
```

点赞/收藏必须在同一事务内完成：

- 关系表写入或删除。
- `post_stats.like_count` / `favorite_count` 原子增减。
- 对应 integration event 写入 `outbox_event`。

Redis 更新在事务提交后执行，失败不回滚业务事务。

## 事件

Content 生产：

- `content.post.published`
- `content.post.updated`
- `content.post.deleted`
- `content.post.tags.updated`
- `content.post.liked`
- `content.post.unliked`
- `content.post.favorited`
- `content.post.unfavorited`
- `content.post.viewed`

Content 消费：

- `user.profile.updated`：刷新作者快照。
- Comment 事件可以更新评论计数，但评论事实仍归 Comment。

关键跨服务事件统一走 producer outbox + RabbitMQ topic exchange。

## 跨服务依赖

- User：创建文章和刷新作者快照时读取用户资料摘要。
- Upload：文章图片、封面和正文资源只保存文件引用。
- Comment：文章详情可聚合评论计数，但不直接读评论库。

## Go 目标落点

- HTTP：`services/zhicore-content/api/http`
- Application：`services/zhicore-content/internal/content/application`
- Domain：`services/zhicore-content/internal/content/domain`
- Ports：`services/zhicore-content/internal/content/ports`
- Infrastructure：`postgres`、`redis`、`rabbitmq`、`mongo`、`clients`
- Runtime：`services/zhicore-content/internal/content/runtime/module.go`

## 迁移风险

- Java schema 来源存在漂移，Go migration 必须以服务归属重新整理，不原样复制全量初始化 SQL。
- 点赞/收藏链路必须保留 outbox 同事务语义，否则 Ranking/Notification 会丢事件。
- `domain_event_task` 是服务内投影，不应和跨服务 outbox 混用。
- Presence 只能作为附加能力，不能影响文章正文主链路。

## 下一步

- 提取 Content 字段级 HTTP contract。
- 生成 Content migration 草案。
- 先做创建/发布/点赞/收藏/outbox/presence 的行为测试。
