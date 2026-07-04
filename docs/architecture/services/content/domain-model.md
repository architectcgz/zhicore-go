# Content 领域模型设计

本文记录 `zhicore-content` 的 DDD 目标设计：限界上下文、子域、聚合、值对象、领域事件、领域服务和工厂。设计用于指导 Go 目标实现，不表示当前 Go 代码已经完成。

## DDD 战术模式

- **聚合（Aggregate）**：`Post`、`Tag`、`Category`、`PostStats`
- **值对象（Value Object）**：封装业务概念，避免到处传裸 `string` / `int64`
- **领域服务（Domain Service）**：承载不自然归属单个实体的纯业务规则
- **领域事件（Domain Event）**：聚合内发生的业务事实，由 application 转换为集成事件
- **工厂（Factory）**：封装复杂聚合创建逻辑，确保创建时不变量
- **仓储（Repository）**：隔离领域模型和 PostgreSQL / MongoDB 等基础设施
- **端口（Port）**：application 对基础设施的抽象依赖

## 子域

| 子域 | 职责 | 主要存储 |
| --- | --- | --- |
| Post Lifecycle | 文章创建、元数据修改、发布、定时发布、撤回、删除、恢复、归档和作者快照更新 | `posts`、`scheduled_publish_event` |
| Post Body / Draft | 正文、草稿、富文档块、媒体引用和 PostgreSQL / MongoDB 指针状态 | MongoDB `post_bodies`、`posts.published_body_id`、`posts.draft_body_id` |
| Tag / Category | 标签、分类、slug、标签关系和标签统计投影 | `tags`、`categories`、`post_tags`、`tag_stats` |
| Engagement | 点赞、取消点赞、收藏、取消收藏和文章本地统计 | `post_likes`、`post_favorites`、`post_stats` |
| Projection / Integration | 服务内投影任务、跨服务 outbox、消费幂等和管理端重试 | `domain_event_task`、`outbox_event`、`consumed_events`、`outbox_retry_audit` |
| Reader Presence | 读者在线 presence session、离开和在线状态查询 | Redis |

**Reader Presence 隐私边界：**

- **可见范围**：presence 数据聚合后只暴露"当前在线人数"（匿名数字），不暴露具体读者身份列表，即使对文章作者也不例外。
- **匿名用户**：匿名/未登录读者不记录 presence（无法关联身份，也不聚合到计数，防止 fingerprinting）。
- **TTL**：presence key 默认 TTL = 30s（心跳续期），停止心跳后 30s 自动过期；不需要主动 leave 接口也能自然清理。
- **用户选项**：第一阶段不提供"关闭 presence 追踪"的用户设置；如未来引入，User 服务的 presence 偏好设置由 Content 消费并在写 presence 前检查。
- **不得暴露**：presence 数据不得包含 `userId`、设备信息或 IP 进入 API 响应；Redis key 中的 `userId` 是服务内部实现细节，不对外。

## 聚合

### `Post`

`Post` 是文章主数据聚合根，负责维护文章生命周期和必须强一致的状态。

- **标识**：内部 `PostID` 由数据库生成；如需要短公开标识，使用独立 `PublicPostID`，不依赖 `zhicore-id-generator`。
- **归属**：`OwnerID` 引用 User；`OwnerSnapshot` 是 Content 本地快照，不是用户资料事实源。
- **状态**：`Draft`、`Published`、`Scheduled`、`Deleted`。
- **行为**：`UpdateMeta`、`UpdateTags`、`SetTopic`、`SetCoverImage`、`Publish`、`SchedulePublish`、`ExecuteScheduledPublish`、`Unpublish`、`CancelSchedule`、`Delete`、`Restore`、`MarkArchived`、`UpdateOwnerSnapshot`。
- **领域事件**：如 `PostCreated`、`PostPublished`、`PostDeleted`。

核心不变量：

- 已删除文章不能编辑、发布、定时发布或更新标签。
- 只有草稿文章可以设置定时发布。
- 只有定时发布文章可以取消定时或执行定时发布。
- 已发布文章不能重复发布。
- 标题、正文、审核、并发版本等发布 guard 由 application 在调用聚合行为前编排；MongoDB 正文是否存在、hash 是否匹配、blocks schema 是否可解析不属于 `Post` 聚合职责。

为什么高频计数不放在 `Post` 聚合：

- 浏览、点赞、收藏、评论计数更新频率高，如果修改 `Post` 聚合会导致热点锁和乐观锁重试。
- 这些计数的一致性要求低于文章发布状态，允许最终一致或短暂不准确。

### `PostBody` / `DraftSnapshot`

`PostBody` 和 `DraftSnapshot` 表达正文、草稿、内容类型、富文档块和媒体引用，但不单独成为跨事务聚合根。

正文详细存储、copy-on-write、发布原子切换见 [body-storage-and-publishing.md](body-storage-and-publishing.md)。将这部分从领域模型文档拆出去的原因是正文存储涉及 PostgreSQL / MongoDB 跨存储失败语义、cleanup / repair worker 和 schema migration，属于 application / infrastructure 协作设计，不应把领域模型文档变成运行机制清单。

### `Tag`

`Tag` 是标签聚合根：

- `TagID` 是内部标识。
- `TagName` 是展示名称。
- `TagSlug` 是全局唯一自然键，创建后不可变。
- 描述可以修改。

`Tag` 只负责标签自身规则，不拥有文章列表。标签下文章由 `post_tags` 关系和查询模型提供。

### `Category`

`Category` 是分类聚合根或受控参考数据：

- 维护名称、slug、描述、父分类和排序。
- Go 第一阶段如果没有分类管理 API，可以先作为只读参考数据实现。
- `Post` 只保存分类或话题引用，不把分类树嵌入文章聚合。

### `PostStats`

`PostStats` 是独立统计聚合根，负责文章高频计数：

- **标识**：`PostID`，与文章一对一。
- **字段**：`ViewCount`、`LikeCount`、`FavoriteCount`、`CommentCount`。
- **行为**：`IncrementViews`、`IncrementLikes`、`DecrementLikes`、`IncrementFavorites`、`DecrementFavorites`、`UpdateCommentCount`。
- **不变量**：计数不能为负。

点赞事务示例：

```text
单个 PostgreSQL 事务：
  post_likes 表（插入）
  + post_stats.like_count（原子 +1）
  + outbox_event（集成事件）
```

该事务不修改 `Post` 聚合，`Post` 的乐观锁版本号不会因点赞递增。

### `PostLike` / `PostFavorite`

点赞和收藏不是聚合根，而是以 `(PostID, UserID)` 为自然唯一键的互动关系实体：

- `PostLike`
- `PostFavorite`

这些关系总是由 application 在同一个 PostgreSQL 事务里和 `PostStats` 聚合一起修改。`PostStats` 只维护计数不能为负等统计不变量，不拥有 `likedBy` / `favoritedBy` 这类操作者事实；带操作者的集成事件由 application 在关系表插入或删除确认成功后映射并写入 outbox。这样可以避免重复点赞只改统计、却仍误发 `content.post.liked`。

Redis 点赞 / 收藏状态和计数缓存只在 PostgreSQL 事务提交后 best-effort 更新，失败不回滚业务事务。

当前用户视角的 `liked` / `favorited` 查询状态可以因为 Redis 或受控 DB fallback 故障而返回 unknown；unknown 只是查询降级状态，不是领域事实。领域模型只表达关系存在或不存在，不能把 unknown 写入 `post_likes`、`post_favorites`、`post_stats`、outbox 或 Redis 事实缓存。完整互动产品语义见 [engagement-design.md](engagement-design.md)。

### 非领域聚合

以下对象不建成领域聚合：

- `outbox_event`：跨服务集成事件投递台账。
- `domain_event_task`：Content 服务内投影任务。
- `consumed_events`：消费幂等记录。
- Reader presence：短生命周期 Redis 状态。
- `scheduled_publish_event`：定时发布任务记录。
- `outbox_retry_audit`：管理端重试审计。
- `content_body_cleanup_tasks` / `content_body_repair_tasks`：正文资源回收和数据修复任务。

## 领域事件

核心领域事件：

| 领域事件 | 触发聚合行为 | 业务含义 |
| --- | --- | --- |
| `PostCreated` | `Post.Create()` | 文章草稿创建 |
| `PostPublished` | `Post.Publish()` | 文章发布，内容对外可见 |
| `PostUnpublished` | `Post.Unpublish()` | 文章撤回，回到草稿状态 |
| `PostDeleted` | `Post.Delete()` | 文章软删除，不再可见 |
| `PostRestored` | `Post.Restore()` | 文章从删除状态恢复 |
| `PostMetaUpdated` | `Post.UpdateMeta()` | 标题、摘要、封面等元数据更新 |
| `PostTagsUpdated` | `Post.UpdateTags()` | 文章标签关系变更 |
| `OwnerSnapshotRefreshed` | `Post.UpdateOwnerSnapshot()` | 作者快照版本更新 |
| `PostLiked` | application 确认 `PostLike` 新关系插入成功，并调用 `PostStats.IncrementLikes()` | 用户点赞 |
| `PostUnliked` | application 确认 `PostLike` 关系删除成功，并调用 `PostStats.DecrementLikes()` | 用户取消点赞 |
| `PostFavorited` | application 确认 `PostFavorite` 新关系插入成功，并调用 `PostStats.IncrementFavorites()` | 用户收藏 |
| `PostUnfavorited` | application 确认 `PostFavorite` 关系删除成功，并调用 `PostStats.DecrementFavorites()` | 用户取消收藏 |
| `TagCreated` | `Tag.Create()` | 标签创建 |
| `PostTagAssociated` | `Post.UpdateTags()` | 文章关联标签 |

生命周期：

```text
聚合根行为或 application 事务确认的业务事实产生领域事件
-> application 保存聚合 / 关系实体后收集事件
-> application 转换为集成事件并写入 outbox
-> infrastructure dispatcher 投递到 RabbitMQ
-> Search / Ranking / Notification 等服务消费
```

领域事件是领域层纯业务概念，不依赖 JSON、RabbitMQ 或 Protobuf；集成事件是跨服务契约消息，有明确 schema、routing key 和 envelope。

## 值对象

核心值对象：

| 值对象 | 含义 |
| --- | --- |
| `PostID`、`PublicPostID` | 文章内部标识和可选外部公开标识 |
| `UserID`、`OwnerID` | 作者或操作者引用 |
| `TagID`、`TagName`、`TagSlug` | 标签标识、名称和唯一 slug |
| `CategoryID`、`TopicID` | 分类和话题引用 |
| `FileID` | File service 拥有的文件引用，例如封面或正文媒体 |
| `PostTitle`、`PostExcerpt` | 标题和摘要，封装长度、空值和摘要规则 |
| `PostStatus` | 文章生命周期状态 |
| `OwnerSnapshot` | 作者昵称、头像文件引用和资料版本快照 |
| `PostContent`、`ContentBlock`、`MediaResource` | 正文、富文档块和媒体引用 |
| `ScheduledAt`、`PublishedAt` | 发布相关时间点 |
| `CursorToken` | 列表游标编码值 |

使用值对象的原因：

- 编译期区分 `PostID`、`UserID`、`TagID` 等不同 ID，防止参数传错。
- 让代码签名表达领域概念，减少注释解释。
- 将长度、空值、slug 格式、状态等验证逻辑集中在类型构造处。

## 领域服务

领域服务只承载纯业务规则，不依赖 HTTP、PostgreSQL、MongoDB、Redis 或 MQ。

| 领域服务 | 职责 |
| --- | --- |
| `PostPublishPolicy` | 判断 `Post` 聚合状态是否允许发布、定时发布或撤回；只检查聚合内字段 |
| `ExcerptGenerator` | 从正文生成摘要，去除多余空白，控制最大长度 |
| `TagSlugPolicy` | 校验标签名称并生成规范化 slug |
| `OwnerSnapshotPolicy` | 比较作者资料版本，决定是否允许刷新作者快照 |
| `EngagementPolicy` | 判断文章是否可点赞或收藏 |
| `InternalEventPriorityPolicy` | 为 Content 内部事件分级 |

基础设施检查由 application 编排。例如正文是否存在需要查询 MongoDB，这不属于领域服务职责；File 文件状态、User 作者状态、审核服务可用性也都不进入 domain。

## 工厂

工厂负责封装复杂聚合创建逻辑，确保创建时不变量：

- `PostFactory`：创建草稿文章、初始化作者快照、初始状态和 `PostCreated` 事件。
- `TagFactory`：创建标签、通过 `TagSlugPolicy` 生成 slug、产生 `TagCreated` 事件。

选择工厂的原因是避免 application 手动拼装聚合导致不变量遗漏，同时让聚合构造逻辑在 domain 层可测试。
