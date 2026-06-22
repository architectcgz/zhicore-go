# 服务边界与数据归属

本文件定义 `zhicore-go` 中各 Go 服务的数据归属、权威查询、事件归属和跨服务 DTO 放置规则。

在对应 Go 服务完全替换 Java 服务前，`../zhicore-microservice` 仍是接口和行为的事实源。

## 事实来源

第一版边界基于以下 Java 事实整理：

- `../zhicore-microservice/zhicore-*` 下的 Java 服务模块。
- `../zhicore-microservice/database/init-all-databases.sql` 中按服务划分的 PostgreSQL 表。
- 各 Java 服务 `interfaces/controller` 包下的 controller route。
- `../zhicore-microservice/zhicore-client/src/main/java/com/zhicore/clients` 下的共享 Feign contract。
- `../zhicore-microservice/zhicore-common/src/main/java/com/zhicore/common/mq/TopicConstants.java` 中的 Java RocketMQ topic/tag 常量。
- Go 目标消息模型使用 RabbitMQ；Java topic/tag 名称只作为迁移来源事实，不作为 Go 目标 broker contract。

## 核心规则

谁拥有数据，谁就拥有该数据的权威查询。

其他服务可以调用归属服务的查询，也可以出于产品 API 便利暴露 facade 路由，但不能拥有另一个服务聚合的数据模型、持久化 schema 或 repository。

推论：

- 服务可以保存其他服务的 ID 作为引用。
- 服务可以保存来自其他服务的稳定快照，但前提是这个快照服务于自己的本地聚合生命周期。
- 快照不是权威源数据，源数据仍由原归属服务负责更新和兼容性规则。
- 新 Go 代码禁止跨服务数据库 join。
- `libs/kit` 不能放业务模型、repository 或服务数据归属规则。

## 归属层级

### 服务私有数据

服务私有数据放在所属服务内：

- 领域模型：`services/<service>/internal/...`
- 应用层读写模型：`services/<service>/internal/...`
- 持久化模型和 repository：`services/<service>/internal/...`
- schema migration：`services/<service>/migrations/`
- 服务拥有的 HTTP/API 形态：`services/<service>/api/`

其他服务不得导入 `services/<service>/internal`。

### 同步跨服务 contract

服务间同步调用的 provider-owned client contract 放在：

```text
libs/contracts/clients/<provider-service>/
```

例子：

- `libs/contracts/clients/content/`：Content 提供的查询 DTO 和 typed client contract。
- `libs/contracts/clients/user/`：User 提供的用户资料 DTO 和 typed client contract。

Provider 拥有 contract，因为 provider 拥有 API 行为和数据生命周期。Consumer 可以依赖 contract，但不拥有 contract。

### 事件 contract

跨服务事件 payload 放在：

```text
libs/contracts/events/<domain>/
```

例子：

- `libs/contracts/events/content/`：文章创建、发布、删除等事件。
- `libs/contracts/events/user/`：用户注册、用户资料更新等事件。

事件应包含 consumer 需要的稳定事实，不包含 provider 私有持久化细节。

## 服务归属矩阵

### `zhicore-gateway`

拥有：

- 边缘路由、请求认证拦截、CORS、网关 filter、边缘灰度路由判断。
- 网关使用的 token 校验缓存和 token 黑名单缓存。

不拥有：

- 用户身份、角色、登录凭证或用户资料。
- 下游业务服务拥有的 API schema。

允许依赖：

- JWT 配置和校验规则。
- 共享 Redis/cache 中的 token 黑名单和校验缓存。
- 当显式接入时，可以读取 Ops 拥有的灰度发布状态。

说明：

- Gateway route 是部署/API 表面 contract，不是领域数据 contract。
- Gateway 只做入口控制和转发，不实现业务规则。

### `zhicore-user`

拥有：

- 用户身份、凭证、资料、头像文件引用、角色、启用/禁用状态、JWT 签发和刷新。
- 社交关系：关注、粉丝/关注统计、拉黑。
- 用户签到记录和签到统计。
- User 服务自己的 outbox 事件。

拥有的表：

- `users`
- `roles`
- `user_roles`
- `user_follows`
- `user_follow_stats`
- `user_blocks`
- `user_check_ins`
- `user_check_in_stats`
- User 服务自己的 `outbox_events`

权威查询：

- 用户详情和用户摘要。
- 批量用户摘要查询。
- 粉丝/关注列表和关系判断。
- 拉黑判断。
- 陌生人消息权限。
- 签到统计。

Provider contract：

- `libs/contracts/clients/user`
- `libs/contracts/events/user`

说明：

- User 不拥有文章、评论、私信、通知或文件资源。
- 用户中心 facade 路由可以存在，但必须委托给真正的数据归属服务。

### `zhicore-content`

拥有：

- 文章、草稿、文章内容、生命周期状态、定时发布、发布、删除、恢复、标签、分类和话题引用。
- 文章统计、文章点赞、文章收藏、标签统计、文章作者快照和 Content 自己的读模型。
- Content 自己的 outbox、consumed-event、scheduled-publish、internal event task 表。
- Content 使用的 MongoDB 内容/文章投影。

拥有的表：

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

权威查询：

- 文章详情、文章摘要、批量文章摘要、文章正文、草稿、我的文章、公开文章列表、作者文章列表。
- 标签详情、标签搜索、热门标签、标签下文章。
- 文章点赞/收藏状态和计数。
- 文章作者 ID。
- 文章上下文中的读者在线状态。

Provider contract：

- `libs/contracts/clients/content`
- `libs/contracts/events/content`

允许依赖：

- 调用 User 获取作者/用户摘要，或获取本地未维护的用户关系事实。
- 调用 Upload 处理封面或正文图片相关的文件 URL/删除行为。
- 消费 User profile update 事件刷新本地作者快照。
- 消费 Comment 事件更新文章评论统计。

说明：

- `posts` 中的 `owner_name`、`owner_avatar_id`、`owner_profile_version` 是 Content 拥有的作者快照，不是 User 的源数据。
- 在明确引入独立 Topic 服务前，`topic_id` 归 Content 管理。

### `zhicore-comment`

拥有：

- 评论、回复、评论媒体引用、评论状态、评论统计、评论点赞和 Comment 自己的 outbox 事件。

拥有的表：

- `comments`
- `comment_stats`
- `comment_likes`
- Comment 服务自己的 `outbox_events`

权威查询：

- 评论详情。
- 按文章查询评论列表、根评论回复、游标/分页/增量评论查询。
- 评论点赞状态和点赞计数。
- 管理端评论查询和删除命令。

Provider contract：

- `libs/contracts/clients/comment`
- `libs/contracts/events/comment`

允许依赖：

- 调用 Content 验证或获取文章事实。
- 调用 User 获取作者摘要。
- 调用 Upload 处理评论媒体上传和删除。
- 可以读取 Ranking 拥有的热门文章候选，作为评论缓存或物化逻辑的输入。

说明：

- Comment 拥有评论树，Content 拥有文章和文章聚合统计。
- Comment 事件是更新 Content 文章评论计数的首选方式。

### `zhicore-message`

拥有：

- 私信会话、私信消息、消息已读状态、撤回状态和消息派发 outbox task。

拥有的表：

- `conversations`
- `messages`
- `message_outbox_task`

权威查询：

- 用户会话列表、会话详情、会话数量。
- 会话消息列表。
- 私信未读数。

Provider contract：

- `libs/contracts/clients/message`
- `libs/contracts/events/message`

允许依赖：

- 调用 User 获取用户摘要、关系判断、拉黑判断和陌生人消息权限。
- 可以通过 Message 自己拥有的 adapter 对接外部 IM。

说明：

- Message 未读数不是 Notification 未读数，这两个聚合必须分开。

### `zhicore-notification`

拥有：

- 通知收件箱、已读状态、未读数、通知聚合状态、投递台账、用户通知偏好、免打扰设置、作者订阅、广播 campaign、全局公告和小助手消息。

拥有的表：

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

权威查询：

- 用户通知收件箱。
- 通知未读数和未读 breakdown。
- 通知偏好、免打扰和作者订阅。
- 通知投递状态和重试。

Provider contract：

- `libs/contracts/clients/notification`
- `libs/contracts/events/notification`

允许依赖：

- 消费 User、Content、Comment 事件创建通知。
- 调用 User 获取 fanout 所需的粉丝列表或用户摘要。
- 可以发布通知实时 fanout 事件。

说明：

- Notification 不拥有触发通知的源用户、源文章、源评论或源私信。
- Notification payload 可以包含来源快照，但事实源仍属于来源服务。

### `zhicore-search`

拥有：

- 搜索索引、搜索建议、热门搜索词、搜索历史和 Search 自己的读模型。

拥有的存储：

- Elasticsearch index。
- Search 服务本地 suggestion/history 存储或缓存。

权威查询：

- 文章全文搜索。
- 搜索建议。
- 热门搜索词。
- 用户搜索历史。

Provider contract：

- `libs/contracts/clients/search`
- `libs/contracts/events/search`，仅当 Search 发布自己拥有的事实时使用。

允许依赖：

- 消费 Content 事件索引、更新、标签更新和删除文章文档。
- 在索引修复或结果补全时调用 Content 获取文章权威详情。

说明：

- Search 结果是派生读模型。文章详情和可见性语义仍由 Content 决定。

### `zhicore-ranking`

拥有：

- 排行榜 ledger、分数、快照、热门文章/创作者/话题榜读模型、Redis 物化榜单和 MongoDB 排行榜归档。

拥有的存储：

- Ranking Redis key。
- Ranking MongoDB archive。
- Ranking PostgreSQL ledger/snapshot 表，如果后续在部署 schema 中正式存在。

权威查询：

- 热门文章。
- 日榜、周榜、月榜。
- 创作者榜。
- 话题榜。
- 排名和分数查询。

Provider contract：

- `libs/contracts/clients/ranking`
- `libs/contracts/events/ranking`，仅当 Ranking 发布自己拥有的事实时使用。

允许依赖：

- 消费 Content 和 Comment 的互动事件。
- 当排行榜响应需要文章详情时，调用 Content 获取文章权威数据。

说明：

- Ranking 拥有分数计算，不拥有文章、点赞、收藏、评论、用户或标签源数据。

### `zhicore-admin`

拥有：

- 举报、举报处理流程、审核审计日志和管理编排记录。

拥有的表：

- `reports`
- `audit_logs`

权威查询：

- 举报列表和详情。
- 审计日志查询。

Provider contract：

- `libs/contracts/clients/admin`，仅当其他服务消费 Admin 拥有的行为时使用。
- 暴露 User/Content/Comment 管理能力的 Admin facade 路由不是所有权声明。

允许依赖：

- 调用 User 完成管理端用户查询、禁用、启用、token 失效。
- 调用 Content 完成管理端文章查询和删除。
- 调用 Comment 完成管理端评论查询和删除。
- 如果继续使用集中发号策略，可以调用 IdGenerator 生成举报和审计 ID。

说明：

- Admin 不直接拥有用户、文章或评论的 mutation 语义。
- Admin 命令 facade 必须委托给归属服务，再在本地记录审计或举报状态。

### `zhicore-upload`

拥有：

- 文件 ID、已上传对象资源、访问级别、文件 URL 解析和文件删除。

拥有的存储：

- 对象存储资源。
- Upload 服务本地文件元数据，如果后续确定需要。

权威查询：

- 文件 URL 查询。
- 文件存在性和访问检查，如果后续加入。

Provider contract：

- `libs/contracts/clients/upload`
- `libs/contracts/events/upload`，仅当文件生命周期事件成为跨服务事实时使用。

允许依赖：

- 正常上传和 URL 查询不应依赖业务服务。

说明：

- 其他服务只保存 `file_id` 引用。头像、封面、评论图片、语音等业务归属仍属于包含它们的业务服务。

### `zhicore-id-generator`

拥有：

- Snowflake/segment ID 发号行为、worker 配置和 segment 分配状态。

权威查询：

- 单个 Snowflake ID。
- 批量 Snowflake ID。
- 按业务 tag 生成 segment ID。

Provider contract：

- `libs/contracts/clients/idgenerator`

说明：

- ID 发出后，具体实体及其 ID 字段归实体所属服务管理。
- IdGenerator 不拥有用户、文章、评论、消息、举报或通知。

### `zhicore-ops`

拥有：

- 迁移灰度发布状态、用户灰度标记、回滚/对账记录、CDC 修复流程和迁移运维状态。

拥有的存储：

- Ops Redis key，例如灰度配置、灰度状态、用户灰度标记、对账历史、回滚历史。
- Ops-only CDC/checkpoint 状态，如果后续需要持久化。

权威查询：

- 灰度配置和状态。
- 用户灰度分配判断。
- 最新对账和回滚状态。

Provider contract：

- `libs/contracts/clients/ops`，仅当其他服务直接调用 Ops 时使用。

说明：

- Ops 可以在迁移期检查或对账业务数据，但不会因此成为业务数据归属方。
- 长期产品功能不应依赖 Ops 的迁移内部状态。

## 表和存储归属汇总

| 归属方 | 表 / 存储 |
| --- | --- |
| User | `users`, `roles`, `user_roles`, `user_follows`, `user_follow_stats`, `user_blocks`, `user_check_ins`, `user_check_in_stats`, User `outbox_events` |
| Content | `posts`, `post_stats`, `post_likes`, `post_favorites`, `categories`, `tags`, `post_tags`, `tag_stats`, `scheduled_publish_event`, `outbox_event`, `outbox_retry_audit`, `consumed_events`, `domain_event_task`, Content MongoDB projection |
| Comment | `comments`, `comment_stats`, `comment_likes`, Comment `outbox_events` |
| Message | `conversations`, `messages`, `message_outbox_task` |
| Notification | `notifications`, `notification_group_state`, `notification_campaign`, `notification_campaign_shard`, `notification_delivery`, `notification_user_preference`, `notification_user_dnd`, `notification_author_subscription`, `global_announcements`, `assistant_messages` |
| Admin | `reports`, `audit_logs` |
| Search | Elasticsearch index 和 Search 本地 suggestion/history 存储 |
| Ranking | Ranking Redis key、Ranking MongoDB archive、Ranking ledger/snapshot 存储 |
| Upload | 对象存储资源和 Upload 本地文件元数据 |
| Gateway | 路由/auth 缓存 key、token 黑名单、token 校验缓存 |
| IdGenerator | ID worker/segment 分配状态 |
| Ops | 灰度发布、CDC、对账和迁移运维状态 |

如果相同表名出现在多个服务数据库中，它仍然是服务私有表。例如 User 和 Comment 都可以有 `outbox_events`，每张表归所在服务数据库拥有。

## 查询归属例子

| 用例 | 归属服务 | 允许的 facade |
| --- | --- | --- |
| 查询某个用户发表的文章 | Content | User 可以暴露 `GET /api/v1/users/{userId}/posts`，但必须委托给 Content |
| 查询某篇文章的评论 | Comment | Content 可以暴露文章中心 facade，但必须委托给 Comment |
| 查询用户资料摘要 | User | 其他服务调用 User contract，或维护明确文档化的快照 |
| 搜索结果补全文章详情 | Content | Search 可以调用 Content，或只返回 Search 自己拥有的预览数据 |
| 按关键词搜索文章 | Search | Gateway 只转发；Content 不拥有全文索引行为 |
| 查询热门文章或分数 | Ranking | Content 可以展示 Ranking 输出，但不能拥有分数计算 |
| 查询通知未读数 | Notification | Gateway 或 User 只能通过委托暴露 facade |
| 查询私信未读数 | Message | Notification 不能在没有产品 contract 的情况下合并它 |
| Admin 禁用用户 | User | Admin facade 委托给 User，并在本地记录审计 |
| Admin 删除文章 | Content | Admin facade 委托给 Content，并在本地记录审计 |
| 解析文件 URL | Upload | 其他服务保存 file ID 并调用 Upload |

## 示例：查询某用户发表的文章

问题：查询某个用户发表的全部文章时，是 Content 调用 User，还是 User 调用 Content？数据定义在哪里？

结论：

- `zhicore-content` 拥有文章，所以权威查询归 Content。
- 查询端点归 Content，例如 `GET /api/v1/posts/authors/{authorId}`。
- 持久化 `Post` 模型、post repository、分页规则、可见性规则、post DTO mapping 放在 `services/zhicore-content/internal` 和 `services/zhicore-content/api`。
- 跨服务 DTO 和 client contract 放在 `libs/contracts/clients/content`。
- `zhicore-user` 可以出于产品导航便利暴露 `GET /api/v1/users/{userId}/posts`，但该路由只能调用 Content contract，并返回或浅层转换 Content 拥有的结果。

正确方向：

```text
user facade route -> content client contract -> content service query -> content-owned post store
```

错误方向：

```text
content service -> user service -> user-owned post query
```

原因是 User 不拥有文章。

## 什么时候可以同步调用其他服务

当一个服务需要另一个服务拥有的数据，并且调用方无法通过本地 read model 正确维护该数据时，可以同步调用归属服务。

允许的例子：

- Content 创建文章时调用 User 校验或快照作者身份。
- User 为了用户中心 API 暴露已发布文章 facade，并调用 Content。
- Search 在索引修复时调用 Content 获取文章权威详情。

如果数据可以通过事件维护，并且最终一致性可接受，优先使用事件和本地 read model，避免同步调用。

## 事件归属

事件来源跟随数据归属。

Go 目标消息模型使用 RabbitMQ：

- Exchange：`zhicore.events`
- Exchange 类型：`topic`
- Routing key 格式：`<domain>.<event>`，例如 `content.post.published`
- Queue 归属：每个消费服务拥有自己的 queue、dead-letter queue、retry 行为和幂等存储
- 投递规则：除非用例显式记录更强保证，否则 consumer 必须容忍重复投递和乱序投递

| 事件族 | 归属方 | RabbitMQ routing key 示例 | Java 现有来源 |
| --- | --- | --- | --- |
| 用户注册、关注、取消关注、资料更新 | User | `user.registered`, `user.followed`, `user.unfollowed`, `user.profile.updated` | `ZhiCore-user-events` |
| 文章发布、更新、删除、标签更新、点赞、取消点赞、收藏、取消收藏、浏览 | Content | `content.post.published`, `content.post.updated`, `content.post.deleted`, `content.post.tags.updated`, `content.post.liked`, `content.post.unliked`, `content.post.favorited`, `content.post.unfavorited`, `content.post.viewed` | `ZhiCore-post-events` |
| 评论创建、删除、点赞 | Comment | `comment.created`, `comment.deleted`, `comment.liked` | `ZhiCore-comment-events` |
| 私信发送、已读 | Message | `message.sent`, `message.read` | `ZhiCore-message-events` |
| 通知实时 fanout | Notification | `notification.realtime.comment_stream`, `notification.realtime.user_notification`, `notification.realtime.unread_count` | `ZhiCore-notification-events` |

事件 consumer 可以更新自己的 read model 或 projection，但事件不会转移源聚合的所有权。

例子：

- Search 消费文章事件并拥有搜索索引，但文章事实仍归 Content。
- Ranking 消费文章/评论互动事件并拥有榜单分数，但源互动事实仍归 Content 和 Comment。
- Notification 消费文章、评论、用户事件并拥有通知收件箱，但触发通知的事实仍归源服务。
- Content 消费用户资料更新事件刷新作者快照，但用户资料仍归 User。

## 跨服务引用和快照

允许：

- 保存其他服务的 ID 作为 opaque reference，例如 `author_id`、`post_id`、`file_id`、`target_id`。
- 保存本地聚合需要的快照，例如 Content 中的文章作者昵称和头像。
- 保存本地聚合拥有的派生计数和 read model，例如 `post_stats`、`comment_stats`、ranking score、search index。

禁止：

- 导入另一个服务的 `internal` Go 包。
- 新 Go 代码直接查询另一个服务的数据库表。
- 把另一个服务的 persistence model 复制进 consumer 服务。
- 把本地快照当成权威源数据。
- 添加跨服务 SQL 外键。

## 跨服务读取模式选择

当一个流程需要多个归属方的数据时，只能选择一种明确模式：

1. Provider query：通过 `libs/contracts/clients/<provider-service>` 调用数据归属服务。
2. Facade route：在另一个服务暴露产品友好的路由，但委托给归属服务。
3. Event-backed read model：当最终一致性可接受时，消费归属方事件维护本地 projection。
4. 归属服务新增 API：现有 contract 不匹配时，在归属服务新增窄查询。

不要通过共享 repository、共享数据库连接或把业务模型移动到 `libs/kit` 来解决跨服务读取。

## Facade 规则

Facade 路由只有同时满足以下条件才允许：

- 它是为了产品/API 易用性存在，不是数据所有权声明。
- 它不复制另一个服务的持久化逻辑。
- 它通过 `libs/contracts/clients/<provider-service>` 委托给归属服务。
- 任何返回形态转换都必须是浅层转换，并在 facade 边界说明。
- 对归属服务错误的转换必须一致，不能隐藏数据归属。

## 提升到 contract 的规则

DTO 默认保留在服务本地，直到至少一个外部服务确实需要它。

只有满足以下条件时，才提升到 `libs/contracts/clients/<provider-service>`：

- 它是同步跨服务 API 的一部分。
- Provider 愿意对它做版本管理并保持兼容。
- 多个 consumer 或 facade 路由需要同一个稳定形态。

不要把内部领域模型、数据库实体或 repository filter 提升到 `libs/contracts`。

## Go 迁移规则

从 Java 迁移服务到 Go 时：

1. 只把归属服务自己的表、存储、repository 和领域规则迁移到 `services/<service>/internal`。
2. 对外可见的 provider contract 放在 `services/<service>/api`；跨服务 typed contract 放在 `libs/contracts/clients/<provider-service>`。
3. Provider 拥有的事件放在 `libs/contracts/events/<domain>`。
4. 在 consumer 服务内部定义服务本地 consumer-side port，并在边缘实现 adapter。
5. 数据库 migration 放在 `services/<service>/migrations`。
6. 除非前端、网关和已知调用方在同一迁移切片中明确调整，否则保留 Java API 形态。
7. 用 contract test 证明 Go provider 能替代 Java provider 行为。

## 开放决策

实现相关切片前必须解决：

- Upload 的文件元数据是落在自己的数据库、只依赖对象存储元数据，还是继续接外部 file-service backend。
- token blacklist 写入长期归 Gateway 还是 User，例如 logout 和 invalidate-token 流程。
- `topic_id` 后续是否拆出独立 Topic 服务。拆出前由 Content 拥有话题引用。
- Ranking ledger/snapshot 是否需要在 `services/zhicore-ranking/migrations` 中补正式 PostgreSQL migration。
