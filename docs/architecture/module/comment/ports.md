# Comment Ports 设计

Ports 放在 `services/zhicore-comment/internal/comment/ports`，按能力和用例族定义 consumer-side interface。

## 核心端口

| Port | 职责 |
| --- | --- |
| `CommentCommandRepository` | `Comment` 聚合加载、保存、编辑、软删除、批量软删除回复。 |
| `CommentFloorAllocator` | 在事务内为指定 `post_id` 分配下一个楼层号。 |
| `CommentQueryRepository` | 详情、文章评论列表、回复列表、游标分页、增量查询和管理端查询。 |
| `CommentStatsRepository` | 初始化统计、原子增减回复数、批量应用点赞 delta、读取统计。 |
| `CommentPostStatsRepository` | 维护文章级评论统计 `total_comments` / `total_top_level_comments`，并提供对账读取。 |
| `CommentLikeRepository` | 点赞关系插入、删除、存在性检查和批量状态查询。 |
| `CommentHotRankRepository` | 初始化顶级评论 HOT 排序行，按 `post_id + like_count DESC + floor ASC` 读取候选，批量更新点赞数。 |
| `CommentRecommendedRankRepository` | 初始化顶级评论 RECOMMENDED 排序行，按 `post_id + recommended_score DESC + floor DESC` 读取候选，更新推荐分和可见性。 |
| `CommentCounterDeltaRepository` | 追加、claim、标记完成或失败点赞计数 delta，供后台 worker 批量聚合。 |

## 可选端口

| Port | 引入条件 |
| --- | --- |
| `CommentMediaRepository` | 只有当当前版本需要独立查询或修复评论媒体引用时才引入；默认媒体引用随 `CommentCommandRepository` 保存。 |

## 基础设施机制端口

| Port | 职责 |
| --- | --- |
| `TransactionRunner` | 显式事务边界。 |
| `OutboxPublisher` | 业务事务内追加 Comment 集成事件。 |
| `OutboxAdminRepository` | outbox summary、dead retry 和状态流转。 |
| `Clock` | 时间源和游标时间比较。 |
| `CursorCodec` | `RECOMMENDED` / `TIME` / `HOT` 游标编码和解码；具体 codec 落在 application 或 infrastructure，避免 domain 绑定 Base64 兼容细节。 |
| `RateLimiter` / `AntiSpamPolicy` | 评论创建和高频互动的业务限流，例如 `actorUserId + postId`、单用户周期配额和同内容短时间重复。 |
| `CommentCounterDeltaWorker` | 后台批量应用 `comment_counter_deltas`，更新 `comment_stats`、`comment_hot_rank` 和 `comment_recommended_rank`；落点是 infrastructure job，不进入 domain。 |
| `RecommendedRankDecayWorker` | 使用分布式锁或 claim 机制处理 `next_decay_at` 到期的推荐 rank 行，重算 `freshness_tier` 和 `recommended_score`。 |

## 缓存端口

| Port | 职责 |
| --- | --- |
| `CommentDetailCacheStore` | 评论详情 cache-aside。 |
| `CommentListCacheStore` | 文章评论列表和回复列表缓存。 |
| `CommentLikeCacheStore` | 点赞状态和点赞数缓存。 |
| `HomepageCommentCacheStore` | 首页评论缓存。 |
| `RankingHotPostCandidateStore` | 热门候选本地缓存。 |

## 外部服务端口

| Port | 职责 |
| --- | --- |
| `ContentPostClient` | 校验文章存在、可见性、是否允许评论；返回 `postAuthorId` 供 `comment.created` 通知事件使用。 |
| `UserProfileClient` | 获取评论作者摘要、批量用户摘要和用户状态；DTO 同时包含内部 `userId` 和外部 `publicId`。 |
| `UserRelationClient` | 批量判断拉黑关系和互动权限。 |
| `FileReferenceClient` | 校验 Upload 文件引用存在、类型和状态；批量解析展示 URL。 |
| `RankingClient` | 读取热门文章候选；不拥有 Ranking 分数。 |

## 首个切片端口范围

首个交付切片只锁定创建根评论 / 回复和文章顶级评论传统分页。最小端口集先保持窄接口，`comment_hot_rank` 的读写可以先封装在 command / query repository 内，等 HOT 读模型或 worker 复杂度上升后再拆成独立 `CommentHotRankRepository`：

- 必需端口：`TransactionRunner`、`CommentFloorAllocator`、`CommentCommandRepository`、`CommentQueryRepository`、`CommentStatsRepository`、`CommentPostStatsRepository`、`OutboxPublisher`、`ContentPostClient`、`UserProfileClient`、`UserRelationClient`、`FileReferenceClient`、`RateLimiter`、`Clock`。
- 可暂缓端口：`CommentLikeRepository`、`CommentCounterDeltaRepository`、`CommentCounterDeltaWorker`、缓存 store、`RankingClient`。
- 首切顶级列表默认 `RECOMMENDED`，因此 `CommentRecommendedRankRepository` 应随首个列表切片进入；如果首切临时只交付 `TIME` 排序，必须在 contract 状态中明确标注，不得假装默认排序已完成。

`OutboxPublisher` 必须进入首个切片，因为 `comment.created` 是 Content / Notification / Ranking 依赖的关键事实；不要先实现只写业务表、不写 outbox 的临时路径。

## 端口约束

- 端口不能暴露 `*gorm.DB`、`*redis.Client`、Gin context、HTTP DTO、ORM sentinel 或外部 SDK 类型。
- repository 返回 module-local 语义错误，例如 `CommentNotFound`、`DuplicateLike`、`StaleCursor`。
- cache store 不把 Redis key 字符串泄漏给 application；application 只表达“失效文章评论列表、根评论回复列表、首页快照”等语义。
- client adapter 负责把 HTTP status、Feign / REST 错误、超时和熔断结果翻译为 module-local 错误；具体 resilience policy 见 `runtime-resilience.md`。
- `OutboxPublisher` 只负责在业务事务内追加事件，dispatcher 的 claim、发送、retry/dead 状态更新属于 infrastructure job。
- Comment 不提供媒体上传 facade；Upload 只通过 `FileReferenceClient` 作为文件事实 owner 被调用。
- 查询路径的 User 摘要和 Upload URL 解析可以降级返回占位或省略 URL；写路径校验不能降级放行。

## Go 包落点

```text
services/zhicore-comment/
  api/http/
  internal/comment/
    application/
      commands/
      queries/
    domain/
      comment/
      stats/
      interaction/
      media/
      cursor/
      shared/
      events/
    ports/
    infrastructure/
      postgres/
      redis/
      rabbitmq/
        publishers/
      clients/
      cursor/
      jobs/
    runtime/
      module.go
```

分层依赖方向：

```text
api/http -> application -> domain
                  \-> ports <- infrastructure
runtime -> api/http/application/infrastructure
```

第一版可以不机械拆出所有子包；拆包标准是职责和依赖边界，而不是为了看起来像 DDD。
