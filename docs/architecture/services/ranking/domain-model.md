# Ranking 领域模型设计

本文记录 `zhicore-ranking` 的 DDD 目标模型。服务入口和关键结论见 [README.md](README.md)；事件输入和可见性投影见 [data-events-projections.md](data-events-projections.md)。

## 统一语言

Ranking 是独立限界上下文。统一语言以“热度事件、指标增量、账本、窗口桶、当前状态、周期分数、快照、候选集、归档、重放”为核心，不把 Content、Comment、User 或 Redis 的模型引入 Ranking domain。

## 标识策略

Ranking 内部持久化和分数计算统一使用 Content 的内部 `post_id BIGINT` 作为 opaque reference。`ranking_event_ledger`、`ranking_delta_bucket`、`ranking_post_state` 和 `ranking_period_score` 不生成自己的文章 ID，也不把 Content 的内部主键解释为连续业务编号。

HTTP API 中的 `{postId}` 和返回给前端的文章 ID 使用 Content 的外部 `public_id`。Ranking handler 在入站时通过 Content contract 把 `public_id` 解析为内部 `post_id`；出站列表先按内部 `post_id` 排名，再通过本地快照或 Content 批量 contract 转成 `public_id`。

`public_post_id` 可以作为 Ranking 的稳定快照字段保存到 `ranking_post_state`，用于减少查询时的 Content 批量解析；它不是 Ranking 权威数据，缺失或过期时以 Content 返回为准。

## 子域

| 子域 | 职责 | 主要存储 |
| --- | --- | --- |
| Event Ledger | 消费热度事实事件，去重，记录不可变账本 | `ranking_event_ledger` |
| Delta Bucket | 将高频事件压缩到短时间窗净增量，支持晚到事件和重试 | `ranking_delta_bucket` |
| Score State | 文章累计计数、当前热度分、周期分数、可见性投影和版本 | `ranking_post_state`、`ranking_period_score`、`ranking_projection_event_inbox` |
| Query / Materialization | 文章、创作者、话题总榜和周期榜查询；Redis ZSET 物化 | Redis ZSET、PostgreSQL 回源 |
| Snapshot / Repair | 从权威 state / period 重建 Redis，修复缓存漂移 | PostgreSQL、Redis |
| Replay / Admin | 从 ledger 全量重放，重建 state、period、Redis 和候选集 | `ranking_event_ledger`、锁 |
| Hot Candidate | 为 Comment 等下游提供热门文章候选集快照 | Redis ZSET + meta |
| Archive | 日/周/月历史榜单归档和冷数据查询 | MongoDB archive |

## 聚合

### `RankingEventLedger`

`RankingEventLedger` 是 Ranking 接受后的不可变热度事实账本。

- **内部标识**：`EventID`，来自上游事件 envelope 的 `eventId`，作为 `ranking_event_ledger.event_id` 主键。
- **归属**：Ranking 拥有账本行；`PostID`、`ActorID`、`AuthorID` 是源服务 opaque reference；`PublicPostID` 是可选展示快照。
- **事实字段**：`EventType`、`MetricType`、`Delta`、`OccurredAt`、`BucketStart`、`PublishedAt`、`PartitionKey`、`SourceService`、`SourceOpID`、`PublicPostID`。
- **行为**：`Accept`、`IgnoreDuplicate`、`NormalizeOccurredAt`、`BuildPartitionKey`。
- **不变量**：`EventID` 全局唯一；`PostID` 必填；`MetricType` 必须属于受控枚举；`Delta != 0`；`OccurredAt` 必须是业务事实发生时间。

`ledger` 只追加，不做 `PENDING/FAILED/DEAD` 状态机。消费成功的定义是同一事务内完成 `ledger` 插入和 `bucket` 聚合。重复事件命中 `event_id` 主键时直接返回 duplicate no-op 并 ack。

### `RankingDeltaBucket`

`RankingDeltaBucket` 是 `(BucketStart, PostID)` 下的短时间窗净增量聚合。

- **标识**：`(BucketStart, PostID)`。
- **增量字段**：`ViewDelta`、`LikeDelta`、`FavoriteDelta`、`CommentDelta`。
- **已应用字段**：`AppliedViewDelta`、`AppliedLikeDelta`、`AppliedFavoriteDelta`、`AppliedCommentDelta`。
- **claim 字段**：`FlushOwner`、`FlushStartedAt`、`Flushed`、`FlushedAt`。
- **行为**：`Accumulate`、`ClaimForFlush`、`PendingDelta`、`MarkApplied`、`MarkFlushed`、`ReleaseStaleClaim`。
- **不变量**：`pending delta = total delta - applied delta`；flush 只能物化 pending delta，不能把整桶累计值重复写入 state / period / Redis。

默认 bucket window 参考 Java 设计为 `10s`，低流量可放宽到 `30s`，不建议超过 `60s`。Go 实现必须配置化。

晚到事件仍追加到原 `(bucket_start, post_id)`。如果目标 bucket 已 flushed，则累加 delta，并把 `flushed=false`、`flushed_at=NULL`，下轮只物化新增 pending delta。

### `RankingPostState`

`RankingPostState` 是单篇文章的当前热度权威状态。

- **标识**：`PostID`。
- **元数据快照**：`PublicPostID`、`AuthorID`、`PublishedAt`、`TopicIDs`。
- **可见性投影**：`PublicVisible`、`ContentStatus`、`VisibilityReason`、`VisibilityUpdatedAt`。
- **计数字段**：`ViewCount`、`LikeCount`、`FavoriteCount`、`CommentCount`。
- **分数字段**：`RawScore`、`HotScore`。
- **并发字段**：`Version`、`LastBucketStart`、`UpdatedAt`。
- **行为**：`ApplyBucketDelta`、`RecalculateScore`、`AttachMetadata`、`ApplyVisibility`、`IncrementVersion`。
- **不变量**：所有计数不能为负；`HotScore` 由公式计算，不能由外部直接写入；并发 flush 必须通过 version、post 级锁或等价条件更新避免覆盖。

`RankingPostState` 是 PostgreSQL 权威状态。Redis 总榜只可从它或同一事务提交后的 flush 结果物化，不能反向写回 `post_state`。公开榜单必须过滤 `public_visible = true`。

### `RankingPeriodScore`

`RankingPeriodScore` 是周期榜单分数的独立写模型。

- **标识**：`(PeriodType, PeriodKey, PostID)`。
- **字段**：`DeltaScore`、`UpdatedAt`。
- **行为**：`Increment`、`RebuildFromLedger`、`ListTop`。
- **不变量**：`PeriodType` 只能是 `DAY`、`WEEK`、`MONTH`；`PeriodKey` 必须稳定格式化：`YYYY-MM-DD`、`YYYY-Www`、`YYYY-MM`。

创作者榜和话题榜第一阶段不单独建 PostgreSQL 权威表，默认从文章分数和 `ranking_post_state.author_id/topic_ids` 派生后物化到 Redis。

默认保留窗口：

| 周期 | PostgreSQL 活跃保留 | 超出窗口 |
| --- | --- | --- |
| 日榜 | 最近 7 天 | 归档到 MongoDB 后清理 |
| 周榜 | 最近 60 天覆盖到的 ISO 周 | 归档到 MongoDB 后清理 |
| 月榜 | 最近 365 天覆盖到的月份 | 归档到 MongoDB 后清理 |

### `RankingSnapshot`

`RankingSnapshot` 是从权威状态构建 Redis 视图的应用级读模型，不是独立事实聚合。

- **来源**：`ranking_post_state`、`ranking_period_score`。
- **输出**：文章、创作者、话题的总榜、日榜、周榜、月榜 Redis ZSET。
- **行为**：`RefreshCurrentSnapshots`、`RefreshActiveSnapshots`、`ReplaceRedisRanking`。
- **不变量**：快照刷新必须原子替换目标 key 或使用临时 key + rename；刷新失败保留上一版成功结果。

### `HotPostCandidateSet`

`HotPostCandidateSet` 是面向下游服务的热门文章候选集快照。

- **来源**：`ranking:posts:hot` 当前总榜。
- **Redis key**：`ranking:posts:hot:candidates` 和 `ranking:posts:hot:candidates:meta`。
- **字段**：`Version`、`GeneratedAt`、`CandidateSize`、`SourceKey`、`SourceCount`、`MinScore`、`Stale`、候选 `PostID/PublicPostID/Rank/Score`。
- **行为**：`RefreshCandidates`、`MarkStale`、`GetCandidates`。
- **不变量**：候选集不维护第二套热度公式；只从总榜截取前 N 条且过滤 `score <= 0`；刷新失败保留上一次成功结果。

候选集服务于 Comment 的首页评论缓存判定，不是前端分页接口的替代品。

### `RankingArchive`

`RankingArchive` 是历史榜单归档文档。

- **标识**：`(EntityType, RankingType, Period, Rank)` 或等价唯一键。
- **实体类型**：`post`、`creator`、`topic`。
- **榜单类型**：`daily`、`weekly`、`monthly`。
- **存储**：MongoDB。
- **行为**：`ArchiveDaily`、`ArchiveWeekly`、`ArchiveMonthly`、`LoadMonthlyArchive`。
- **不变量**：归档是冷数据查询来源，不参与实时分数计算；归档失败不回滚实时榜单。

MongoDB archive 索引至少覆盖：

```javascript
db.ranking_archives.createIndex(
  { entity_type: 1, ranking_type: 1, period: 1, rank: 1 },
  { unique: true }
);
db.ranking_archives.createIndex(
  { entity_type: 1, ranking_type: 1, period: 1, entity_id: 1 }
);
```

## 非聚合对象

- Redis ZSET key、分布式锁 key、临时快照 key：运行时物化和同步机制。
- RabbitMQ delivery tag、consumer offset、DLQ 消息：消息基础设施状态。
- Content/Post/User/Topic 详情 DTO：外部服务 contract 或展示补齐模型。
- Sentinel / 限流配置：运行时保护机制。

## 值对象

| 值对象 | 含义 |
| --- | --- |
| `PostID` | Content 拥有的文章内部引用，落库为 `post_id BIGINT` |
| `PublicPostID` | Content 对外文章 ID，用于 HTTP path、response 和前端路由 |
| `ActorID` | 触发热度事件的用户引用 |
| `AuthorID` | 文章作者引用 |
| `TopicID` | Content 当前拥有的话题引用；未来 Topic 服务拆出后仍作为 opaque reference |
| `EventID` | 上游事件 envelope 的 `eventId` |
| `RankingMetricType` | 第一阶段为 `VIEW`、`LIKE`、`FAVORITE`、`COMMENT` |
| `MetricDelta` | 指标增量，支持正负，不允许 0 |
| `BucketWindow` | bucket 时间窗配置 |
| `BucketStart` | `OccurredAt` 按窗口向下取整后的时间 |
| `PeriodType` | `DAY`、`WEEK`、`MONTH` |
| `PeriodKey` | 周期键，例如 `2026-03-14`、`2026-W11`、`2026-03` |
| `HotScore` | 排行分数和 rank 展示值 |
| `RankingEntityType` | `POST`、`CREATOR`、`TOPIC` |
| `RankingSnapshotVersion` | 候选集或快照版本 |

## 领域服务

| 领域服务 | 职责 |
| --- | --- |
| `HotScoreCalculator` | 根据计数、权重、发布时间和半衰期计算 `RawScore` / `HotScore` |
| `BucketWindowPolicy` | 根据 `OccurredAt` 和窗口大小计算 `BucketStart` |
| `BucketFlushPolicy` | 判断 bucket 是否可 flush、计算 stale claim 时间和 pending delta |
| `PeriodKeyPolicy` | 生成日/周/月 `PeriodKey` |
| `ViewDedupPolicy` | 判断浏览事件是否进入 ledger；具体 Redis 去重由 infrastructure 承载 |
| `SnapshotBuildPolicy` | 从文章状态构建文章 / 创作者 / 话题榜，排序和截断 |
| `CandidatePolicy` | 从总榜生成热门候选集，过滤无效分数并计算 stale |
| `ReplayPolicy` | replay 时的锁、批次和暂停 live ingestion 规则 |

权重和半衰期是 runtime 配置注入到 application / domain service 的业务参数；domain 不直接读取环境变量。

## 热度公式

Go 目标第一阶段沿用 Java 侧 half-life 衰减公式，避免迁移时改变榜单语义：

```text
rawScore = w_view * viewCount
         + w_like * likeCount
         + w_favorite * favoriteCount
         + w_comment * commentCount

ageDays = max(0, hours_between(publishedAt, now) / 24)
timeDecay = pow(0.5, ageDays / halfLifeDays)
hotScore = rawScore * timeDecay
```

Java 参考默认值：`w_view=1.0`、`w_like=5.0`、`w_favorite=8.0`、`w_comment=10.0`、`halfLifeDays=7.0`。这些值必须配置化。`published_at IS NULL` 时 `timeDecay=1.0`，但未发布、删除、下架或隐藏文章不应进入公开榜单。

## 浏览去重策略

- 登录用户：`ranking:dedup:view:{internalPostId}:user:{viewerId}`。
- 匿名用户：`ranking:dedup:view:{internalPostId}:anon:{ipHash}`。
- `ipHash` 由 Gateway 或 Content 在事件 payload 中提供，Ranking 不接收原始 IP。
- TTL 默认 30 分钟，匿名去重允许较高误判率，不追求精确唯一访客。
- 单篇文章浏览上限使用 `ranking:view:cap:{internalPostId}:{yyyyMMdd}` 记录当日已计入的浏览分数，每日自然过期。
- 达到上限后 consumer ack 事件并记录 metrics，但不写 `ranking_event_ledger`，因为 ledger 的 `delta` 不允许为 0。
