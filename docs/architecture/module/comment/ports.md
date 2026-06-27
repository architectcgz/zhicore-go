# Comment Ports 设计

Ports 放在 `services/zhicore-comment/internal/comment/ports`，按能力和用例族定义 consumer-side interface。

## 核心端口

| Port | 职责 |
| --- | --- |
| `CommentCommandRepository` | `Comment` 聚合加载、保存、编辑、软删除、批量软删除回复。 |
| `CommentFloorAllocator` | 在事务内为指定 `post_id` 分配下一个楼层号。 |
| `CommentQueryRepository` | 详情、文章评论列表、回复列表、游标分页、增量查询和管理端查询。 |
| `CommentStatsRepository` | 初始化统计、原子增减回复数、批量应用点赞 delta、读取统计。 |
| `CommentLikeRepository` | 点赞关系插入、删除、存在性检查和批量状态查询。 |
| `CommentHotRankRepository` | 初始化顶级评论 HOT 排序行，按 `post_id + like_count DESC + floor ASC` 读取候选，批量更新点赞数。 |
| `CommentCounterDeltaRepository` | 追加、claim、标记完成或失败点赞计数 delta，供后台 worker 批量聚合。 |

## 可选端口

| Port | 引入条件 |
| --- | --- |
| `CommentMediaRepository` | 只有当 Go 第一阶段需要独立查询或修复评论媒体引用时才引入；默认媒体引用随 `CommentCommandRepository` 保存。 |

## 基础设施机制端口

| Port | 职责 |
| --- | --- |
| `TransactionRunner` | 显式事务边界。 |
| `OutboxPublisher` | 业务事务内追加 Comment 集成事件。 |
| `OutboxAdminRepository` | outbox summary、dead retry 和状态流转。 |
| `Clock` | 时间源和游标时间比较。 |
| `CursorCodec` | `TIME` / `HOT` 游标编码和解码；具体 codec 落在 application 或 infrastructure，避免 domain 绑定 Base64 兼容细节。 |
| `CommentCounterDeltaWorker` | 后台批量应用 `comment_counter_deltas`，更新 `comment_stats` 和 `comment_hot_rank`；落点是 infrastructure job，不进入 domain。 |

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
| `UserProfileClient` | 获取评论作者摘要、批量用户摘要和用户状态。 |
| `UserRelationClient` | 判断拉黑关系和互动权限。 |
| `FileUploadClient` | 上传评论图片和语音，解析文件 URL。 |
| `RankingClient` | 读取热门文章候选；不拥有 Ranking 分数。 |

## 首个切片端口范围

首个交付切片只锁定创建根评论 / 回复和文章顶级评论传统分页。最小端口集先保持窄接口，`comment_hot_rank` 的读写可以先封装在 command / query repository 内，等 HOT 读模型或 worker 复杂度上升后再拆成独立 `CommentHotRankRepository`：

- 必需端口：`TransactionRunner`、`CommentFloorAllocator`、`CommentCommandRepository`、`CommentQueryRepository`、`CommentStatsRepository`、`OutboxPublisher`、`ContentPostClient`、`UserProfileClient`、`Clock`。
- 可暂缓端口：`CommentHotRankRepository`、`CommentLikeRepository`、`CommentCounterDeltaRepository`、`CommentCounterDeltaWorker`、缓存 store、`RankingClient`、`FileUploadClient` 和媒体 facade。

`OutboxPublisher` 必须进入首个切片，因为 `comment.created` 是 Content / Notification / Ranking 依赖的关键事实；不要先实现只写业务表、不写 outbox 的临时路径。

## 端口约束

- 端口不能暴露 `*gorm.DB`、`*redis.Client`、Gin context、HTTP DTO、ORM sentinel 或外部 SDK 类型。
- repository 返回 module-local 语义错误，例如 `CommentNotFound`、`DuplicateLike`、`StaleCursor`。
- cache store 不把 Redis key 字符串泄漏给 application；application 只表达“失效文章评论列表、根评论回复列表、首页快照”等语义。
- client adapter 负责把 HTTP status、Feign / REST 错误、超时和熔断结果翻译为 module-local 错误。
- `OutboxPublisher` 只负责在业务事务内追加事件，dispatcher 的 claim、发送、retry/dead 状态更新属于 infrastructure job。

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
