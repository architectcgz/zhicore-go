# Content Application、Ports 与实现切片

本文记录 `zhicore-content` 的 application use case、ports、事务边界、Go 包落点和推荐首个实现切片。

## Application 用例

Content application 层按命令、查询分层组织 use case。application 拥有事务边界、权限上下文、幂等、端口调用和错误映射。

认证上下文由 Gateway 解析 JWT 后注入，Content 自身不解析客户端 JWT。HTTP 入站层只把可信身份 header 映射为 application input，例如：

```text
type Actor struct {
    UserID int64
    Roles  []string
}
```

规则：

- 登录态命令必须显式携带 `Actor` 或等价 `AuthContext`，不能从 request body 接收 `userId` 作为当前操作者。
- `CreatePost` 的作者 ID 来自 `Actor.UserID`，并作为 `posts.owner_id` 和作者快照的归属引用。
- `PublishPost`、`UpdateDraftBody`、`DeleteDraft`、`UpdatePostTags` 等作者写操作必须在 application 层加载文章并校验 `owner_id == Actor.UserID`。
- 缺少 `Actor` 映射为 `LOGIN_REQUIRED`；操作者不是作者映射为 `RESOURCE_ACCESS_DENIED` 或 Content 服务级 contract 登记的兼容错误。
- application 不读取 HTTP header、不依赖框架上下文、不解析 `Authorization`。

### Commands

- `CreatePost`：创建文章草稿，保存作者快照，初始化统计，必要时写正文/草稿和内部投影任务。
- `UpdateDraftMeta`：更新草稿标题、摘要、封面、话题和分类引用；标题、摘要、封面由 PostgreSQL 充当真相源。
- `UpdateDraftBody` / `SaveDraft`：校验 blocks schema，写入新的 MongoDB draft body，PostgreSQL 用乐观锁切换 `draft_body_id` / `draft_body_hash`。
- `DeleteDraft`：删除服务端草稿指针并写正文清理任务；客户端本地草稿只属于前端 UX，不进入服务端事实。
- `PublishPost`：校验作者、状态、`post_version`、`draft_body_id`、`draft_body_hash`、正文 schema、审核和正文最小长度，先写 MongoDB snapshot，再用 PostgreSQL 事务原子切换 `published_*`、outbox 和 cleanup task。
- `UnpublishPost`：撤回已发布文章，回到草稿状态并写投影任务。
- `SchedulePost` / `CancelSchedule` / `ExecuteScheduledPublish`：维护定时发布记录和最终发布。
- `DeletePost` / `RestorePost` / `PurgePost`：软删除、恢复和清理文章，删除/恢复事件属于内部投影 P0。
- `UpdatePostTags` / `RemovePostTag`：维护文章标签关系，触发标签统计投影。
- `LikePost` / `UnlikePost`：维护点赞关系、`PostStats` 统计和互动事件。
- `FavoritePost` / `UnfavoritePost`：维护收藏关系、`PostStats` 统计和互动事件。
- `SyncAuthorSnapshot`：消费用户资料更新事件，按版本刷新作者快照。
- `UpdateCommentCount`：消费评论事件，更新 `PostStats` 评论计数。

### Queries

- `GetPostDetail`、`GetPostContent`、`GetDraft`
- `ListPublishedPosts`、`ListAuthorPosts`、`ListMyPosts`、`CursorListPosts`、`BatchGetPosts`
- `GetPostTags`、`GetTagDetail`、`SearchTags`、`ListHotTags`、`ListPostsByTag`
- `GetPostEngagement`、`BatchGetEngagementStatus`
- `GetLikeStatus`、`BatchGetLikeStatus`、`GetLikeCount`（内部可作为 engagement query 的窄能力）
- `GetFavoriteStatus`、`BatchGetFavoriteStatus`、`GetFavoriteCount`（内部可作为 engagement query 的窄能力）
- `GetReaderPresence`
- `ListAdminPosts`、`ListOutboxDeadOrFailedEvents`

命令用例修改状态，返回简单成功/失败或聚合 ID。查询用例只读取，返回 DTO 或视图模型。不要在命令用例里嵌套复杂查询逻辑。

## Ports

Ports 放在 `services/zhicore-content/internal/content/ports`，按聚合或用例族定义接口，避免过度碎片化。

### 核心端口

| Port | 职责 | 说明 |
| --- | --- | --- |
| `PostRepository` | Post 聚合持久化 | 加载、保存、按作者校验所有权、乐观锁更新 |
| `PostQueryRepository` | Post 查询 | 详情、列表、批量、作者文章、管理端查询 |
| `PostStatsRepository` | PostStats 统计读模型 | 初始化、读取统计；点赞 / 收藏计数由内部 stats delta worker 原子投影 |
| `PostContentStore` | 正文和草稿存储 | 保存、读取、删除 MongoDB 正文和草稿 |
| `TagRepository` | Tag 聚合持久化和查询 | 按 slug 查找、创建、批量查询 |
| `PostTagRepository` | 文章标签关系 | 替换、删除、批量查询文章标签 |
| `CategoryRepository` | 分类查询 | 查询分类或话题引用合法性 |
| `PostEngagementRepository` | 点赞/收藏关系 | 插入、删除、批量查询当前用户与多个 `post_id` 的关系；Redis 不可用时禁止逐条 `EXISTS` 回源 |

### 基础设施机制端口

| Port | 职责 | 说明 |
| --- | --- | --- |
| `TransactionRunner` | 显式事务边界 | 避免 handler 或 repository 偷偷拥有业务事务 |
| `OutboxPublisher` | 跨服务事件发布 | 业务事务内追加 outbox 记录 |
| `EngagementStatsTaskStore` | 互动统计内部任务 | 业务事务内追加 stats delta task，worker claim 后投影到 `post_stats` |
| `ConsumedEventStore` | 消费幂等 | 记录消费过的事件 ID |
| `BodyCleanupTaskStore` | 正文清理任务 | 创建和查询未引用 MongoDB body 的清理任务；发布事务失败后可用独立短事务记录 orphan snapshot cleanup |
| `BodyRepairTaskStore` | 正文修复任务 | 记录正文缺失、hash 不一致等数据一致性事故 |
| `RateLimiter` | Content 业务限流 | 按 actor、post、session、service caller、operation 和高成本资源维度返回 typed decision；必须能表达 allow、`1003` reject、Redis 降级放行、`1004` fail-closed 和 presence no-op，策略见 `rate-limiting.md`。 |

### 缓存和外部服务端口

| Port | 职责 | 说明 |
| --- | --- | --- |
| `PostCacheStore` | Post 缓存 | cache-aside、失效、三态缓存 |
| `TagCacheStore` | Tag 缓存 | 热门标签缓存 |
| `EngagementCacheStore` | 点赞/收藏缓存 | Redis 状态和计数缓存；必须提供批量读取能力，cache error 只返回依赖状态，不把 unknown 伪装成 false |
| `ReaderPresenceStore` | Presence 状态 | session、leave、presence 查询 |
| `UserProfileClient` | User 服务调用 | 获取作者摘要 |
| `FileResourceClient` | File service调用 | 解析或清理文件引用 |
| `BodyParserRegistry` | 正文 schema 解析 | 按 `schemaVersion` 选择 parser，输出 `NormalizedBody` |
| `Clock` | 时间源 | 可测试的时间抽象 |

端口设计原则：

- `PostRepository` 包含 Post 聚合持久化方法，不拆成 10 个小接口。
- `PostQueryRepository` 独立于 `PostRepository`，避免写模型被查询需求污染。
- Outbox、InternalEventTask dispatcher、cleanup worker 属于 infrastructure；application 只依赖发布端口或任务记录端口。
- HTTP 入站层可以先做 route 级限流和身份上下文提取；涉及 owner、post、operation、body size、presence session、outbox event 等业务维度时，通过 `RateLimiter` 或 application use case 前置 guard 执行，不在 handler 中散写限流 key。`RateLimiter` 返回的 decision 由 application / handler 映射为公开错误或 no-op success，adapter 不直接构造 HTTP response。
- Engagement 查询在 Redis 不可用时只能走受控 DB fallback：单篇详情可以返回 `viewer.liked/favorited=null` 和 `viewer.degraded=true`，批量状态必须使用批量 repository 方法，不能循环逐条查 `(user_id, post_id)`。完整规则见 [engagement-design.md](engagement-design.md)。
- 不定义宽泛 `Store` 大接口。

不定义的端口：

- `IdGeneratorClient`：内部 ID 由 PostgreSQL 生成，不依赖中心发号服务。
- `ScheduledPublishStore`：定时发布调度属于 infrastructure，第一阶段可合并到 `PostRepository` 或服务内 repository。

## 事务边界

### 文章命令事务

```text
单个 PostgreSQL 事务：
  posts 表（Post 聚合）
  + post_tags（标签关系）
  + outbox_event（集成事件）
  + domain_event_task（内部投影任务）
  + content_body_cleanup_tasks（正文清理任务，按需）
```

`posts` 行承载文章可见性真相源和乐观锁。published / draft 两组元数据的原因是已发布文章再次编辑时草稿不能污染线上列表和详情。

### 点赞/收藏事务

```text
单个 PostgreSQL 事务：
  post_likes / post_favorites（关系表）
  + outbox_event（集成事件）
  + domain_event_tasks（内部 stats delta task）

事务提交后：
  content-engagement-stats worker 投影 post_stats
  best-effort 更新 Redis 缓存（失败不回滚）
```

点赞/收藏事务不修改 `Post` 聚合，也不直接更新 `post_stats`，避免热点聚合、统计行同步写和乐观锁冲突。

### 标签事务

```text
单个 PostgreSQL 事务：
  tags（Tag 聚合）
  + post_tags（关系表）
  + domain_event_task（PostTagsUpdated，触发标签统计投影）
```

标签统计和热门标签缓存由内部投影任务最终一致更新。

## Go 包落点

目标目录：

```text
services/zhicore-content/
  api/http/              # HTTP 入站适配器
  internal/content/
    application/
      commands/          # 命令用例
      queries/           # 查询用例
    domain/
      post/              # Post 聚合、值对象、领域事件
      poststats/         # PostStats 聚合
      tag/               # Tag 聚合
      engagement/        # PostLike、PostFavorite 实体
      shared/            # 跨聚合的值对象和领域服务
      events/            # 领域事件定义
    ports/               # 端口接口定义
    infrastructure/
      postgres/          # PostgreSQL repository 和 mapper
      mongo/             # MongoDB PostContentStore
      redis/             # CacheStore
      rabbitmq/
        consumers/       # user.profile.updated 等
        publishers/      # OutboxPublisher
      body/              # 正文 copy-on-write 协调、parser registry 装配
      jobs/              # OutboxDispatcher、InternalEventWorker、BodyCleanupWorker、BodyRepairWorker
      clients/           # User、File service client
    runtime/
      module.go          # 依赖注入和模块装配
```

依赖方向：

```text
api/http -> application -> domain <- ports <- infrastructure
```

- `domain` 不依赖任何外层。
- `application` 依赖 `domain` 和 `ports`。
- `infrastructure` 实现 `ports`，依赖 domain 类型做 mapper。
- 消费者和后台 worker 属于 infrastructure，调用 application use case 或 infrastructure 任务接口。

第一版可以不机械拆出所有子包。拆包标准是职责和依赖边界，而不是为了看起来像 DDD。

## 推荐首个实现切片

首个切片选择“创建草稿并发布文章”，原因是它覆盖核心 DDD、PostgreSQL / MongoDB 正文指针、outbox、cleanup / repair、body parser 和 HTTP 入站，但不会一开始卷入点赞、标签、管理端和 presence。

实施步骤：

1. **Domain 层**
   - 建 `Post` 聚合和值对象：`PostID`、`OwnerID`、`PostTitle`、`PostStatus`、`OwnerSnapshot`
   - 建 `PostFactory`
   - 建 `PostPublishPolicy`
   - 定义 `PostCreated`、`PostPublished`

2. **Ports 层**
   - 定义 `PostRepository`、`PostContentStore`、`OutboxPublisher`、`BodyCleanupTaskStore`、`BodyRepairTaskStore`、`UserProfileClient`、`Clock`、`TransactionRunner`、`BodyParserRegistry`

3. **测试**
   - domain 测试：`Post.Publish()` 不变量、`PostFactory.CreateDraft()` 前置条件、`PostPublishPolicy`
   - application 测试：`CreatePost`、`UpdateDraftBody`、`PublishPost` 编排、copy-on-write 失败语义、outbox 和 cleanup task 同事务

4. **Infrastructure 层**
   - PostgreSQL `PostRepository`
   - MongoDB `PostContentStore`
   - V1 body parser
   - `BodyCleanupWorker` / `BodyRepairWorker`
   - Outbox publisher

5. **HTTP 层**
   - `POST /api/v1/posts`
   - `PUT /api/v1/posts/{postId}/draft`
   - `POST /api/v1/posts/{postId}/publish`
   - 从 Gateway 注入的身份上下文构造 `Actor`，缺失时返回认证失败；不做服务内 JWT 解析。

该切片完成后再扩展点赞、收藏、标签、投影、管理端和 reader presence。

首个切片的 application 测试必须覆盖发布失败方向：MongoDB snapshot 写入成功但 PostgreSQL transaction 失败时，线上 published 指针不变，并且新 snapshot 通过 immediate delete、独立 cleanup task 或 orphan scanner 可被回收。
