# Ranking 服务设计

## 事实来源

- Java `zhicore-ranking` controller：`RankingController`。
- Java `zhicore-ranking` domain model / service：`HotScore`、`PostStats`、`CreatorStats`、`RankingMetricType`、`HotScoreCalculator`。
- Java `zhicore-ranking` application service：`RankingLedgerIngestionService`、`RankingLedgerFlushService`、`RankingLedgerReplayService`、`RankingSnapshotService`、`RankingArchiveService`、`RankingHotPostCandidateService`、文章/创作者/话题查询服务。
- Java `zhicore-ranking` infrastructure：`PgRankingLedgerRepository`、`RankingRedisRepository`、`RankingRedisKeys`、MongoDB archive、RabbitMQ/RocketMQ consumer、scheduler。
- Java `zhicore-ranking/src/main/resources/db/ranking-schema.sql`。
- `../zhicore-microservice/docs/architecture/zhicore-ranking-detailed-design.md` 和 `../zhicore-microservice/zhicore-ranking/README.md`。
- Go 目标 Content / Comment 事件设计：`content.post.*`、`comment.*`。

## 职责边界

`zhicore-ranking` 拥有热度事实账本、窗口增量聚合、文章当前热度状态、周期榜单分数、Redis ZSET 物化榜单、热门文章候选集、历史榜单归档和 Ranking 自己的 rebuild / replay 运维流程。

Ranking 不拥有文章、评论、用户、话题或关注关系的源事实。它只保存源服务 ID、必要元数据快照和热度计算结果。文章是否存在、是否公开、作者资料、话题名称和评论内容仍由 Content、Comment、User 或未来 Topic 归属服务解释。

DDD 设计用于指导 Go 目标实现，不表示当前 Go 代码已经完成。Java 侧已经落地的 `ledger -> bucket -> state/period -> Redis` 链路是主要事实来源，但 Go 实现按本仓库 `api/http -> application -> domain/ports -> infrastructure` 的依赖方向重新落点。

## DDD 目标设计

Ranking 是独立限界上下文。统一语言以“热度事件、指标增量、账本、窗口桶、当前状态、周期分数、快照、候选集、归档、重放”为核心，不把 Content、Comment、User 或 Redis 的模型引入 Ranking domain。

### 标识策略

Ranking 内部持久化和分数计算统一使用 Content 的内部 `post_id BIGINT` 作为 opaque reference。`ranking_event_ledger`、`ranking_delta_bucket`、`ranking_post_state` 和 `ranking_period_score` 不生成自己的文章 ID，也不把 Content 的内部主键解释为连续业务编号。

HTTP API 中的 `{postId}` 和返回给前端的文章 ID 使用 Content 的外部 `public_id`。Ranking handler 在入站时通过 Content contract 把 `public_id` 解析为内部 `post_id`；出站列表先按内部 `post_id` 排名，再通过本地快照或 Content 批量 contract 转成 `public_id`。Redis ZSET 成员优先使用内部 `post_id`，避免外部 ID 算法变化影响 flush/rebuild。

`public_post_id` 可以作为 Ranking 的稳定快照字段保存到 `ranking_post_state`，用于减少查询时的 Content 批量解析；它不是 Ranking 权威数据，缺失或过期时以 Content 返回为准。

### 限界上下文与子域

| 子域 | 职责 | 主要存储 |
| --- | --- | --- |
| Event Ledger | 消费热度事实事件，去重，记录不可变账本 | `ranking_event_ledger` |
| Delta Bucket | 将高频事件压缩到短时间窗净增量，支持晚到事件和重试 | `ranking_delta_bucket` |
| Score State | 文章累计计数、当前热度分、周期分数和版本 | `ranking_post_state`、`ranking_period_score` |
| Query / Materialization | 文章、创作者、话题总榜和周期榜查询；Redis ZSET 物化 | Redis ZSET、PostgreSQL 回源 |
| Snapshot / Repair | 从权威 state / period 重建 Redis，修复缓存漂移 | PostgreSQL、Redis |
| Replay / Admin | 从 ledger 全量重放，重建 state、period、Redis 和候选集 | `ranking_event_ledger`、锁 |
| Hot Candidate | 为 Comment 等下游提供热门文章候选集快照 | Redis ZSET + meta |
| Archive | 日/周/月历史榜单归档和冷数据查询 | MongoDB archive |

### 聚合

#### `RankingEventLedger` 聚合

`RankingEventLedger` 是 Ranking 接受后的不可变热度事实账本。

- **内部标识**：`EventID`，来自上游事件 envelope 的 `eventId`，作为 `ranking_event_ledger.event_id` 主键。
- **归属**：Ranking 拥有账本行；`PostID`、`ActorID`、`AuthorID` 是源服务 opaque reference；`PublicPostID` 是可选展示快照。
- **事实字段**：`EventType`、`MetricType`、`Delta`、`OccurredAt`、`BucketStart`、`PublishedAt`、`PartitionKey`、`SourceService`、`SourceOpID`、`PublicPostID`。
- **行为**：`Accept`、`IgnoreDuplicate`、`NormalizeOccurredAt`、`BuildPartitionKey`。
- **不变量**：`EventID` 全局唯一；`PostID` 必填；`MetricType` 必须属于受控枚举；`Delta != 0`；`OccurredAt` 必须是业务事实发生时间，不使用消费时间替代，缺失时才降级为当前时间并记录可观测信号。

`ledger` 只追加，不做 `PENDING/FAILED/DEAD` 状态机。消费成功的定义是同一事务内完成 `ledger` 插入和 `bucket` 聚合。重复事件命中 `event_id` 主键时直接返回 duplicate no-op 并 ack。

#### `RankingDeltaBucket` 聚合

`RankingDeltaBucket` 是 `(BucketStart, PostID)` 下的短时间窗净增量聚合。

- **标识**：`(BucketStart, PostID)`。
- **增量字段**：`ViewDelta`、`LikeDelta`、`FavoriteDelta`、`CommentDelta`。
- **已应用字段**：`AppliedViewDelta`、`AppliedLikeDelta`、`AppliedFavoriteDelta`、`AppliedCommentDelta`。
- **claim 字段**：`FlushOwner`、`FlushStartedAt`、`Flushed`、`FlushedAt`。
- **行为**：`Accumulate`、`ClaimForFlush`、`PendingDelta`、`MarkApplied`、`MarkFlushed`、`ReleaseStaleClaim`。
- **不变量**：`pending delta = total delta - applied delta`；flush 只能物化 pending delta，不能把整桶累计值重复写入 state/period/Redis；只有 pending delta 全部应用后才能置 `flushed = true`。

默认 bucket window 参考 Java 设计为 `10s`，低流量可放宽到 `30s`，不建议超过 `60s`。Go 实现必须配置化，例如 `pipeline.bucket_window`。

**晚到事件处理规则**：

- 所有被接受的事件都先进入 `ranking_event_ledger`，保留原始 `occurred_at` 用于审计和 replay。
- `BucketStart` 默认仍由 `occurred_at` 按窗口向下取整，并写入 ledger，保证 live flush 和 `RebuildFromLedger` 对周期榜的归属一致。
- 如果目标 bucket 尚未 flushed，直接累加对应 delta。
- 如果目标 bucket 已 flushed，Go 实现不把事件强制挪到当前窗口；而是在同一个 `(bucket_start, post_id)` 上追加 delta，并把 `flushed` 重新置为 `false`、清空 `flushed_at`。此时 `pending = total - applied` 正好等于晚到事件新增部分，下轮 flush 只物化这部分。
- 这种“重开 bucket”不需要额外 repair 表，也不会破坏日/周/月周期归属。实现时必须用条件更新保护 `applied_*`，避免正在 flush 的 bucket 被并发追加后重复应用。

#### `RankingPostState` 聚合

`RankingPostState` 是单篇文章的当前热度权威状态。

- **标识**：`PostID`。
- **元数据快照**：`PublicPostID`、`AuthorID`、`PublishedAt`、`TopicIDs`。
- **计数字段**：`ViewCount`、`LikeCount`、`FavoriteCount`、`CommentCount`。
- **分数字段**：`RawScore`、`HotScore`。
- **并发字段**：`Version`、`LastBucketStart`、`UpdatedAt`。
- **行为**：`ApplyBucketDelta`、`RecalculateScore`、`AttachMetadata`、`IncrementVersion`。
- **不变量**：所有计数不能为负；`HotScore` 由公式计算，不能由外部直接写入；并发 flush 必须通过 version、post 级锁或等价条件更新避免覆盖。

`RankingPostState` 是 PostgreSQL 权威状态。Redis 总榜只可从它或同一事务提交后的 flush 结果物化，不能反向写回 `post_state`。

#### `RankingPeriodScore`

`RankingPeriodScore` 是周期榜单分数的独立写模型。

- **标识**：`(PeriodType, PeriodKey, PostID)`。
- **字段**：`DeltaScore`、`UpdatedAt`。
- **行为**：`Increment`、`RebuildFromLedger`、`ListTop`。
- **不变量**：`PeriodType` 只能是 `DAY`、`WEEK`、`MONTH`；`PeriodKey` 必须稳定格式化：`YYYY-MM-DD`、`YYYY-Www`、`YYYY-MM`；周期分只记录本周期累计贡献，不保存文章权威元数据。

创作者榜和话题榜第一阶段不单独建 PostgreSQL 权威表，默认从文章分数和 `ranking_post_state.author_id/topic_ids` 派生后物化到 Redis。只有未来需要独立作者/话题历史状态时，才新增对应 state 表。

**周期分存储策略**：

`ranking_period_score` 保留为独立表，不从 ledger 实时聚合查询。原因是热榜查询属于高频读路径，直接扫 ledger 会把 replay/审计账本变成在线查询瓶颈。代价是需要维护额外写模型和清理任务。

默认保留窗口：

| 周期 | PostgreSQL 活跃保留 | 超出窗口 |
| --- | --- | --- |
| 日榜 | 最近 7 天 | 归档到 MongoDB 后清理 |
| 周榜 | 最近 60 天覆盖到的 ISO 周 | 归档到 MongoDB 后清理 |
| 月榜 | 最近 365 天覆盖到的月份 | 归档到 MongoDB 后清理 |

活跃窗口内查询优先 Redis，miss 时回源 `ranking_period_score`；超出窗口的冷数据直接查 MongoDB archive，不再回填长期 Redis key。

#### `RankingSnapshot`

`RankingSnapshot` 是从权威状态构建 Redis 视图的应用级读模型，不是独立事实聚合。

- **来源**：`ranking_post_state`、`ranking_period_score`。
- **输出**：文章、创作者、话题的总榜、日榜、周榜、月榜 Redis ZSET。
- **行为**：`RefreshCurrentSnapshots`、`RefreshActiveSnapshots`、`ReplaceRedisRanking`。
- **不变量**：快照刷新必须原子替换目标 key 或使用临时 key + rename；刷新失败保留上一版成功结果；Redis 为空时查询可回源 PostgreSQL 或返回明确空榜，不得伪造分数。

#### `HotPostCandidateSet`

`HotPostCandidateSet` 是面向下游服务的热门文章候选集快照。

- **来源**：`ranking:posts:hot` 当前总榜。
- **Redis key**：`ranking:posts:hot:candidates` 和 `ranking:posts:hot:candidates:meta`。
- **字段**：`Version`、`GeneratedAt`、`CandidateSize`、`SourceKey`、`SourceCount`、`MinScore`、`Stale`、候选 `PostID/PublicPostID/Rank/Score`。
- **行为**：`RefreshCandidates`、`MarkStale`、`GetCandidates`。
- **不变量**：候选集不维护第二套热度公式；只从总榜截取前 N 条且过滤 `score <= 0`；刷新失败保留上一次成功结果；元信息用于下游判断新鲜度。

候选集服务于 Comment 的首页评论缓存判定，不是前端分页接口的替代品。Comment 同步后可以降维为本地 set，但 Ranking 必须保留 score 和 rank。

#### `RankingArchive`

`RankingArchive` 是历史榜单归档文档。

- **标识**：`(EntityType, RankingType, Period, Rank)` 或等价唯一键。
- **实体类型**：`post`、`creator`、`topic`。
- **榜单类型**：`daily`、`weekly`、`monthly`。
- **存储**：MongoDB。
- **行为**：`ArchiveDaily`、`ArchiveWeekly`、`ArchiveMonthly`、`LoadMonthlyArchive`。
- **不变量**：归档是冷数据查询来源，不参与实时分数计算；归档失败不回滚实时榜单，但必须可重试和可观测。

MongoDB collection 草案：

```json
{
  "_id": "ObjectId",
  "entity_type": "post",
  "ranking_type": "daily",
  "period": "2026-06-23",
  "rank": 1,
  "entity_id": 123456,
  "public_entity_id": "p1K8x9Q2",
  "score": 9876.54,
  "metadata": {
    "view_count": 10000,
    "like_count": 500,
    "favorite_count": 120,
    "comment_count": 80
  },
  "archived_at": "2026-06-24T00:05:00Z"
}
```

索引要求：

```javascript
db.ranking_archives.createIndex(
  { entity_type: 1, ranking_type: 1, period: 1, rank: 1 },
  { unique: true }
);
db.ranking_archives.createIndex(
  { entity_type: 1, ranking_type: 1, period: 1, entity_id: 1 }
);
```

### 非聚合对象

以下对象不建成领域聚合：

- Redis ZSET key、分布式锁 key、临时快照 key：运行时物化和同步机制。
- RabbitMQ delivery tag、consumer offset、DLQ 消息：消息基础设施状态。
- Content/Post/User/Topic 详情 DTO：外部服务 contract 或展示补齐模型。
- Sentinel / 限流配置：运行时保护机制。

### 值对象

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

核心约束：

| 值对象 | 约束 |
| --- | --- |
| `MetricDelta` | 取消点赞、取消收藏、删除评论必须由源服务显式发送负增量；Ranking 不推断补偿 |
| `BucketStart` | 由 `OccurredAt` 和 `BucketWindow` 计算，统一使用 UTC 业务时间 |
| `HotScore` | 允许 double，但只由 Ranking 解释；对外只作为排序/展示值 |
| `PeriodKey` | 周榜使用 ISO week-based year，避免跨年周错误 |
| `RankingMetricType` | 新增指标必须同步补权重、bucket 列、ledger replay、period score 和 Redis 投影规则 |

### 领域服务

领域服务只承载纯业务规则，不依赖 PostgreSQL、Redis、RabbitMQ、MongoDB、HTTP client 或配置读取。

| 领域服务 | 职责 |
| --- | --- |
| `HotScoreCalculator` | 根据计数、权重、发布时间和半衰期计算 `RawScore` / `HotScore` |
| `BucketWindowPolicy` | 根据 `OccurredAt` 和窗口大小计算 `BucketStart` |
| `BucketFlushPolicy` | 判断 bucket 是否可 flush、计算 stale claim 时间和 pending delta |
| `PeriodKeyPolicy` | 生成日/周/月 `PeriodKey` |
| `ViewDedupPolicy` | 判断浏览事件是否进入 ledger；具体 Redis 去重由 infrastructure 承载 |
| `SnapshotBuildPolicy` | 从文章状态构建文章/创作者/话题榜，排序和截断 |
| `CandidatePolicy` | 从总榜生成热门候选集，过滤无效分数并计算 stale |
| `ReplayPolicy` | replay 时的锁、批次和暂停 live ingestion 规则 |

权重和半衰期是 runtime 配置注入到 application/domain service 的业务参数；domain 不直接读取环境变量或 Nacos。

**热度分数公式**：

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

Java 参考默认值：`w_view=1.0`、`w_like=5.0`、`w_favorite=8.0`、`w_comment=10.0`、`halfLifeDays=7.0`。这些值必须配置化。`published_at IS NULL` 时 `timeDecay=1.0`，但未发布、删除或隐藏文章不应进入公开榜单，过滤依据来自 Content 的公开状态快照或详情回源。

**浏览去重策略**：

- 登录用户：`ranking:dedup:view:{internalPostId}:user:{viewerId}`。
- 匿名用户：`ranking:dedup:view:{internalPostId}:anon:{ipHash}`。
- `ipHash` 由 Gateway 或 Content 在事件 payload 中提供，Ranking 不接收原始 IP；建议使用带服务密钥的 HMAC/SHA-256 截断值，不使用明文 IP 或可逆编码。
- TTL 默认 30 分钟，匿名去重允许较高误判率，不追求精确唯一访客。
- 单篇文章浏览上限使用 `ranking:view:cap:{internalPostId}:{yyyyMMdd}` 记录当日已计入的浏览分数，每日自然过期。达到上限后 consumer ack 事件并记录 metrics，但不写 `ranking_event_ledger`，因为 ledger 的 `delta` 不允许为 0。

## Application 用例

**命令 / 消费用例（Commands / Consumers）**：

- `IngestRankingEvent`：由 RabbitMQ consumer 调用，校验事件、可选执行 view 去重/上限、写 `ranking_event_ledger`、upsert `ranking_delta_bucket`。
- `FlushRankingBuckets`：claim 可刷 bucket，计算 pending delta，更新 `ranking_post_state`、`ranking_period_score`，事务提交后增量物化 Redis。
- `RefreshRankingSnapshots`：从 `ranking_post_state` / `ranking_period_score` 重建当前总榜和活跃日/周/月榜 Redis。
- `RefreshHotPostCandidates`：从 `ranking:posts:hot` 生成 `ranking:posts:hot:candidates` 和 meta。
- `RebuildFromLedger`：管理员触发，暂停 live ingestion，清空物化层，从 `ranking_event_ledger` 顺序重放，刷新 Redis 和候选集。
- `ArchiveRankings`：按日/周/月定时把 Redis 或 PostgreSQL 来源榜单归档到 MongoDB。
- `BackfillPostMetadata`：补齐缺失 `author_id`、`published_at`、`topic_ids`，并触发相关文章的 creator/topic 投影修复。

**查询用例（Queries）**：

- `ListHotPosts` / `ListHotPostsWithScore` / `ListHotPostsWithDetails`。
- `ListDailyHotPosts`、`ListWeeklyHotPosts`、`ListMonthlyHotPosts` 及对应 `scores`。
- `GetPostRank`、`GetPostScore`。
- `ListHotCreators` / `ListHotCreatorsWithScore`、`GetCreatorRank`、`GetCreatorScore`。
- `ListHotTopics` / `ListHotTopicsWithScore`、`GetTopicRank`、`GetTopicScore`。
- `GetHotPostCandidates`。

查询用例返回 DTO 或视图模型，不把 Redis key、Mongo document、PostgreSQL row 泄露给 domain。文章详情补齐由 application 批量调用 Content contract；不能在 Redis store 里同步调用 Content，避免查询层隐藏 N+1。

## Ports

Ports 放在 `services/zhicore-ranking/internal/ranking/ports`，按能力和用例族定义 consumer-side interface。

**核心端口**：

| Port | 职责 |
| --- | --- |
| `RankingLedgerRepository` | 插入 ledger、按游标列 ledger、replay 顺序读取、查询审计 |
| `RankingBucketRepository` | upsert bucket、claim flushable bucket、推进 applied delta、释放 stale claim |
| `RankingStateRepository` | 加载/保存 `ranking_post_state`、乐观锁更新、按 post 批量查询 |
| `RankingPeriodScoreRepository` | 增量更新周期分、查询周期 top、清理/重建 |
| `RankingReplayRepository` | reset materialized state、replay barrier、rebuild 事务所需组合操作 |
| `RankingQueryStore` | 读取文章/创作者/话题总榜和周期榜，可以由 Redis + PostgreSQL fallback 实现 |
| `HotPostCandidateStore` | 候选集原子替换、meta 读写、stale 标记 |
| `RankingArchiveStore` | MongoDB 历史榜单归档和冷数据读取 |

**机制端口**：

| Port | 职责 |
| --- | --- |
| `TransactionRunner` | 显式事务边界 |
| `Clock` | UTC 当前时间、业务日期和 ISO week 计算 |
| `RankingLockManager` | replay、scheduler、bucket claim 和 monthly backfill 锁 |
| `RankingEventConsumerCheckpoint` | 如 Go consumer 需要本地 queue checkpoint，可选；业务幂等仍以 ledger/event_id 为准 |
| `MetricsRecorder` | 低基数指标记录；不能影响业务控制流 |

**缓存、事件和外部服务端口**：

| Port | 职责 |
| --- | --- |
| `RankingRedisMaterializer` | 增量更新/原子替换 Redis ZSET、删除/回填 key |
| `ViewDedupStore` | 浏览去重和单篇文章浏览分数上限 |
| `ContentPostClient` | 解析 `public_id` / 内部 `post_id`、批量补齐文章元数据、公开状态和详情 |
| `CommentClient` | 第一阶段通常不需要；只有评论事件缺少必要字段且不能从 payload 获取时再补 |
| `RankingEventDecoder` | 将 Content / Comment 事件 payload 映射成 Ranking 内部 `RankingEvent` |

端口不能暴露 `*gorm.DB`、`*redis.Client`、Mongo driver、RabbitMQ delivery、HTTP DTO 或外部 SDK 类型。底层 duplicate key、not found、Redis nil、MQ nack 由 infrastructure adapter 翻译为 module-local 语义，再由 application 决定 no-op、重试或公开错误。

## 一致性与事务边界

### 事件摄入事务

```text
事务前：
  decode RabbitMQ JSON
  + 校验 eventId、eventType、publicPostId、occurredAt、delta
  + 若 payload 携带内部 postId：校验格式并作为 opaque reference
  + 若 payload 未携带内部 postId：ContentPostClient.ResolvePublicId(publicPostId)
      - not found / deleted：记录告警并投递 DLQ
      - transient error：nack/requeue 或按 consumer retry 策略重试
  + view dedup / view cap 过滤；被拦截事件 ack 但不写 ledger

单个 PostgreSQL 事务：
  INSERT ranking_event_ledger(event_id, ...)
    - 插入成功：继续
    - event_id 冲突：duplicate no-op
  + UPSERT ranking_delta_bucket(bucket_start, post_id, delta...)
    - 新 bucket：正常插入
    - 未 flushed bucket：累加 delta
    - 已 flushed bucket：累加 delta，并置 flushed=false、flushed_at=NULL

事务提交后：
  ack RabbitMQ
```

消费侧不直接写 Redis、不直接更新 `ranking_post_state`。`event_id` 是消费幂等键；进入 Ranking 内部后的 `post_id` 是局部顺序和 bucket 聚合键。事件 payload 中 `publicPostId` 必填，内部 `postId` 是可选优化字段；Content/Comment 生产方如果已经持有内部 `post_id`，应一起携带，避免 consumer 每条事件同步解析。只携带 `publicPostId` 时，Ranking decoder 通过 Content contract 解析后再落账。

`RebuildFromLedger` 运行期间，consumer 必须通过 replay barrier 或分布式锁暂停 live ingestion，避免 replay 和实时落账重复计数。进入 replay 窗口的消息应 nack/requeue 或快速失败后由 broker retry。

### Bucket flush 事务

```text
短事务 1：
  claim flushable buckets

短事务 2..N：
  对每个 bucket 计算 pending delta
  + 更新 ranking_post_state
  + 更新 ranking_period_score(DAY/WEEK/MONTH)
  + 推进 applied_*，必要时标记 flushed=true

事务提交后：
  增量更新 Redis ZSET
  Redis 失败只记录日志和指标，等待 snapshot/rebuild 回填
```

claim 必须只选择窗口已经结束、未 flushed、未被有效 owner 持有或 claim 已过期的 bucket。Go 实现可用条件更新或 `FOR UPDATE SKIP LOCKED`，但必须保留 `flush_owner/flush_started_at` 语义，方便 crash 后回收。

`ranking_post_state` 更新必须使用 version、post 级锁或等价条件写，避免两个 worker 对同一内部 `post_id` 的相邻 bucket 并发覆盖。`bucket` 的 `applied_*` 是幂等关键字段，不能省略。由于 `like/comment/favorite` 都可能出现正负混合增量，`applied_*` 是否越界不使用简单绝对值边界 SQL 约束表达，而由 `PendingDelta`、条件更新和事务测试保证。

推荐的 bucket 级并发控制是“先锁定、再取 pending、同事务应用”：

```sql
WITH locked AS (
  SELECT bucket_start, post_id,
         view_delta, like_delta, favorite_delta, comment_delta,
         applied_view_delta, applied_like_delta,
         applied_favorite_delta, applied_comment_delta,
         view_delta - applied_view_delta AS pending_view_delta,
         like_delta - applied_like_delta AS pending_like_delta,
         favorite_delta - applied_favorite_delta AS pending_favorite_delta,
         comment_delta - applied_comment_delta AS pending_comment_delta
  FROM ranking_delta_bucket
  WHERE bucket_start = :bucket_start
    AND post_id = :post_id
    AND flushed = FALSE
    AND flush_owner = :worker_id
  FOR UPDATE
),
marked AS (
  UPDATE ranking_delta_bucket b
  SET applied_view_delta = l.view_delta,
      applied_like_delta = l.like_delta,
      applied_favorite_delta = l.favorite_delta,
      applied_comment_delta = l.comment_delta,
      flushed = TRUE,
      flushed_at = NOW(),
      updated_at = NOW()
  FROM locked l
  WHERE b.bucket_start = l.bucket_start
    AND b.post_id = l.post_id
  RETURNING l.*
)
SELECT * FROM marked;
```

application 只把 `pending_*_delta` 应用到 `ranking_post_state` 和 `ranking_period_score`。如果返回 0 行，说明 bucket 被其他 worker 处理、owner 失效或状态变化，本轮跳过。状态更新、period 更新和 `applied_*` 推进必须在同一个 PostgreSQL 事务内完成。

### Snapshot refresh

```text
读取 ranking_post_state + ranking_period_score
构建 post / creator / topic scores
写临时 Redis ZSET
原子替换正式 key
```

总榜从当前 `ranking_post_state` 构建；周期榜从 `ranking_period_score` 构建，并通过 `ranking_post_state` 补齐 author/topic 派生榜。刷新失败保留旧 key。活跃窗口参考 Java 设计：日榜最近 2 天、周榜最近 20 天覆盖到的 ISO 周、月榜最近 365 天覆盖到的月份；具体值配置化。

### Rebuild from ledger

```text
1. 管理员权限校验
2. 获取 ranking:lock:rebuild，TTL 默认 30 分钟，长任务必须续期
3. 设置 replay barrier：ranking:replay:active=true
4. Consumer 看到 barrier 后停止拉取新消息，并等待本地 in-flight handler 完成
5. rebuild job 等待 in-flight drain，超过配置超时则失败退出并释放 barrier
6. 清空 ranking_delta_bucket / ranking_post_state / ranking_period_score / 当前 Redis
7. 按 occurred_at,event_id 游标批量读取 ledger，默认 batch_size=1000
8. 重放到 bucket，flush all buckets without Redis
9. refresh active snapshots，refresh hot candidates
10. 释放 barrier 和 rebuild lock
```

进入 replay 窗口的新消息不写业务表：consumer 可以 nack/requeue，也可以停止消费等待 broker 重新投递。rebuild 使用 ledger 中保存的 `bucket_start` 重放，`occurred_at` 只作为排序和审计字段。rebuild 不能删除 `ranking_event_ledger`。返回结果至少包含 `replayedEvents`、`rebuiltAt`、`duration` 和 `failedStage`。如果候选集刷新失败，保留旧候选集并记录告警。若 rebuild 进程 crash，锁超时后 consumer 可以自动恢复；下一次 rebuild 从头开始，不要求断点续传。

### Archive

归档任务从 Redis 或 PostgreSQL source store 读取日/周/月榜，写 MongoDB。归档不是实时查询权威源；归档失败不影响在线榜单，但必须可重试。月榜冷数据查询可以先查 Redis，缺失时用 lock 回源 MongoDB 并回填 Redis；超出 Redis 保留范围时直接查 MongoDB。

## 查询和缓存策略

PostgreSQL 是 `ledger/bucket/state/period` 的权威源。Redis 只保存可重建榜单和运行期控制状态。

| Redis key | 含义 |
| --- | --- |
| `ranking:posts:hot` | 文章总榜 ZSET |
| `ranking:posts:daily:{date}` | 文章日榜 |
| `ranking:posts:weekly:{year}:{week}` | 文章周榜 |
| `ranking:posts:monthly:{year}:{month}` | 文章月榜 |
| `ranking:creators:hot` | 创作者总榜 |
| `ranking:creators:daily:{date}` / `weekly` / `monthly` | 创作者周期榜 |
| `ranking:topics:hot` | 话题总榜 |
| `ranking:topics:daily:{date}` / `weekly` / `monthly` | 话题周期榜 |
| `ranking:posts:hot:candidates` | 热门文章候选集 |
| `ranking:posts:hot:candidates:meta` | 候选集元信息 |
| `ranking:dedup:view:{internalPostId}:user:{viewerId}` | 登录用户浏览去重 |
| `ranking:dedup:view:{internalPostId}:anon:{ipHash}` | 匿名用户浏览去重 |
| `ranking:view:cap:{internalPostId}:{yyyyMMdd}` | 单篇文章单日浏览累计分数上限 |
| `ranking:backfill:*` | Redis miss 回源回填锁 |
| `ranking:empty:*` | 空结果短 TTL 占位 |
| `ranking:lock:*` | replay、scheduler、archive、monthly backfill 锁 |

缓存更新要求：

| 操作 | Redis 处理 |
| --- | --- |
| 事件摄入 | 不更新榜单 Redis，只写 ledger/bucket |
| bucket flush 成功 | 事务提交后增量更新总榜和周期榜；失败等待 snapshot |
| snapshot refresh | 原子替换总榜、日/周/月榜 |
| rebuild-from-ledger | 重建完成后 refresh active snapshots 和 hot candidates |
| hot candidate refresh | 使用临时 key + rename 原子替换候选集和 meta |
| archive | 不改变当前榜单；必要时回填月榜缓存 |

### Redis miss 回源

总榜查询流程：

1. 先读 `ZREVRANGE ranking:posts:hot ... WITHSCORES`。
2. 如果 key 不存在或返回空，尝试获取 `ranking:backfill:posts:hot`，TTL 默认 60 秒。
3. 拿到锁的请求回源 `ranking_post_state ORDER BY hot_score DESC`，写临时 key 后 `RENAME` 原子替换正式 key。
4. 没拿到锁的请求可以短暂等待或直接回源 PostgreSQL 返回，不重复回填。
5. 回源仍为空时写 `ranking:empty:posts:hot`，TTL 默认 60 秒，避免短时间内击穿。

周期榜查询流程：

1. 先读对应 Redis key，例如 `ranking:posts:daily:2026-06-23`。
2. miss 时，如果 period 在活跃窗口内，回源 `ranking_period_score WHERE period_type=? AND period_key=? ORDER BY delta_score DESC` 并回填 Redis。
3. 如果 period 超出活跃窗口，查 MongoDB archive；冷数据默认不回填长期 Redis key，只可写短 TTL 查询缓存。
4. 回源为空时写 `ranking:empty:{period_type}:{period_key}`，TTL 默认 60 秒。

### 候选集 stale 语义

候选集 meta 至少包含 `version`、`generated_at`、`source_key`、`source_count`、`candidate_size`、`min_score`、`last_refresh_attempt` 和 `consecutive_failures`。`stale` 是查询时派生状态，不单独持久化：

- `now - generated_at < stale_threshold`：fresh。
- `stale_threshold <= now - generated_at < 2 * stale_threshold`：stale，仍返回旧数据，并异步触发 refresh。
- `generated_at` 缺失或 `now - generated_at >= 2 * stale_threshold`：视为过期，Comment 应降级为空候选集或本地旧缓存。

刷新失败时保留上一版候选集，不删除正式 Redis key；连续失败 3 次后发送告警，但不阻塞查询。

## 事件

### 消费事件

Ranking 消费的跨服务事件：

| 事件 | 来源 | 指标 | Delta | 关键字段 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `content.post.viewed` | Content | `VIEW` | `+1` | `eventId`、`publicPostId`、`postId?`、`viewerId?`、`ipHash?`、`occurredAt`、`publishedAt` | 先做浏览去重和上限控制 |
| `content.post.liked` | Content | `LIKE` | `+1` | `eventId`、`publicPostId`、`postId?`、`authorId`、`likedBy`、`occurredAt` | 上游保证点赞幂等 |
| `content.post.unliked` | Content | `LIKE` | `-1` | `eventId`、`publicPostId`、`postId?`、`authorId`、`unlikedBy`、`occurredAt` | 负增量由源服务显式发出 |
| `content.post.favorited` | Content | `FAVORITE` | `+1` | `eventId`、`publicPostId`、`postId?`、`authorId`、`favoritedBy`、`occurredAt` | 收藏增量 |
| `content.post.unfavorited` | Content | `FAVORITE` | `-1` | `eventId`、`publicPostId`、`postId?`、`authorId`、`unfavoritedBy`、`occurredAt` | 负增量 |
| `comment.created` | Comment | `COMMENT` | `+1` | `eventId`、`publicPostId`、`postId?`、`commentId`、`authorId`、`createdAt` | 评论增量 |
| `comment.deleted` | Comment | `COMMENT` | `-affectedCount` | `eventId`、`publicPostId`、`postId?`、`affectedCount`、`deletedAt` | 根评论删除时按 affectedCount 回滚 |
| `comment.liked` | Comment | 第一阶段不消费 | - | `eventId`、`commentId`、`publicPostId`、`postId?`、`likedBy` | 如产品要求计入文章热度，必须先扩展指标、权重、bucket 列和 replay 规则 |
| `comment.unliked` | Comment | 第一阶段不消费 | - | `eventId`、`commentId`、`publicPostId`、`postId?`、`unlikedBy` | 同上 |

Go 目标默认先实现 Content 互动和评论创建/删除。评论点赞是否计入文章热度必须在事件 contract 阶段明确，不能在 Ranking 内自行推断。表中的 `publicPostId` 是稳定契约字段；`postId?` 是生产方可选携带的 Content 内部 `post_id`，仅用于减少 Ranking 解析调用。HTTP API 的 `{postId}` path 值使用 Content `public_id`。

事件 payload 必须表达事实和 delta，不能传“当前总数快照”作为 Ranking 的主输入。发生时间使用 UTC RFC3339；落库字段使用 `occurred_at`。

### 生产事件

Ranking 第一阶段默认不生产关键跨服务事件。热门候选集通过同步查询或定时拉取暴露给 Comment，不通过事件广播。若未来需要 `ranking.hot_candidates.updated` 事件，必须新增 `libs/contracts/events/ranking` contract，并明确它是可丢失提示还是权威事实。

## 跨服务依赖

- Content：文章元数据、批量文章详情、作者 ID、发布时间、话题引用、公开状态。
- Comment：默认只通过事件输入，不同步读取评论库；缺字段时优先修事件 contract。
- User：创作者榜需要用户摘要时通过 User contract 批量补齐；Ranking 不保存用户资料权威数据。
- Notification：不直接依赖 Ranking；Notification 可消费源事件自行创建通知。
- Admin：`rebuild-from-ledger` 可由 Admin facade 委托 Ranking，同时保留 Ranking 自己的管理员权限校验。

## Go 包落点

目标目录：

```text
services/zhicore-ranking/
  api/http/
  internal/ranking/
    application/
      commands/
      queries/
      consumers/
      jobs/
    domain/
      ledger/
      bucket/
      score/
      snapshot/
      candidate/
      archive/
      shared/
    ports/
    infrastructure/
      postgres/
      redis/
      rabbitmq/
      mongo/
      clients/
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

第一版不需要机械拆出所有子包。切片 1 可以先保留 `domain/ledger`、`domain/bucket`、`domain/score` 和少量 ports；snapshot、candidate、archive 后续再补。

后台任务归属：

- `application/jobs`：`FlushRankingBuckets`、`RefreshRankingSnapshots`、`RefreshHotPostCandidates`、`ArchiveRankings`。
- `application/consumers`：事件解码后调用 `IngestRankingEvent`。
- `infrastructure/jobs`：ticker、cron、scheduler adapter 和锁实现。
- `runtime`：读取配置，启动/停止 consumer 和 job，管理 lifecycle context、panic、shutdown 和 readiness。

## API 保留范围

Ranking 保留当前前端和服务间使用的路径。Go 目标不要求兼容 Java 旧数据，但 API 形态需要按目标前端 contract 固定。

- `/api/v1/ranking/posts/hot`：文章总榜，返回 Content `public_id` 列表。
- `/api/v1/ranking/posts/hot/details`：文章总榜详情，批量补 Content 详情。
- `/api/v1/ranking/posts/hot/scores`：文章总榜分数和 rank，`entityId` 使用 Content `public_id`。
- `/api/v1/ranking/posts/hot/candidates`：热门文章候选集。
- `/api/v1/ranking/posts/daily`、`/weekly`、`/monthly`：文章周期榜，返回 Content `public_id` 列表。
- `/api/v1/ranking/posts/daily/scores`、`/weekly/scores`、`/monthly/scores`：文章周期榜分数，`entityId` 使用 Content `public_id`。
- `/api/v1/ranking/posts/{postId}/rank`、`/score`：单篇文章排名和分数，`{postId}` 是 Content `public_id`。
- `/api/v1/ranking/creators/hot`、`/hot/scores`、`/{userId}/rank`、`/{userId}/score`。
- `/api/v1/ranking/topics/hot`、`/hot/scores`、`/{topicId}/rank`、`/{topicId}/score`。
- `/api/v1/ranking/admin/rebuild-from-ledger`：管理员全量补算。

分页页码从 `0` 开始；`size/limit` 必须按配置限制最大值，Java 默认最大 `100` 可作为迁移参考。周榜使用 ISO week-based year 和 week number。服务间候选集可同时返回内部 `postId` 和 `publicPostId`，但外部 HTTP response 不暴露内部 `post_id`。

字段级 request/response、错误码、权限和返回空榜语义需要后续按 `docs/contracts/http-schema-template.md` 提取到 `services/zhicore-ranking/api/http`。

## 数据归属

Ranking 拥有：

- `ranking_event_ledger`
- `ranking_delta_bucket`
- `ranking_post_state`
- `ranking_period_score`
- MongoDB `ranking_archive`
- Redis ZSET 榜单、候选集、view 去重、锁和空结果缓存。

目标 schema 草案只固定方向，正式 migration 以 `services/zhicore-ranking/migrations/` 为准：

```sql
CREATE TABLE ranking_event_ledger (
  event_id VARCHAR(128) PRIMARY KEY,
  event_type VARCHAR(128) NOT NULL,
  post_id BIGINT NOT NULL,
  public_post_id VARCHAR(32) NULL,
  bucket_start TIMESTAMPTZ NOT NULL,
  actor_id BIGINT NULL,
  author_id BIGINT NULL,
  metric_type VARCHAR(32) NOT NULL,
  delta INTEGER NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  published_at TIMESTAMPTZ NULL,
  partition_key VARCHAR(64) NOT NULL,
  source_service VARCHAR(64) NULL,
  source_op_id VARCHAR(128) NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT ck_ranking_ledger_delta_nonzero CHECK (delta <> 0),
  CONSTRAINT ck_ranking_metric_type CHECK (metric_type IN ('VIEW', 'LIKE', 'FAVORITE', 'COMMENT'))
);

CREATE INDEX idx_ranking_ledger_post_occurred
  ON ranking_event_ledger(post_id, occurred_at, event_id);

CREATE INDEX idx_ranking_ledger_occurred_event
  ON ranking_event_ledger(occurred_at, event_id);

CREATE INDEX idx_ranking_ledger_bucket
  ON ranking_event_ledger(bucket_start, post_id);
```

```sql
CREATE TABLE ranking_delta_bucket (
  bucket_start TIMESTAMPTZ NOT NULL,
  post_id BIGINT NOT NULL,
  view_delta BIGINT NOT NULL DEFAULT 0,
  like_delta INTEGER NOT NULL DEFAULT 0,
  favorite_delta INTEGER NOT NULL DEFAULT 0,
  comment_delta INTEGER NOT NULL DEFAULT 0,
  applied_view_delta BIGINT NOT NULL DEFAULT 0,
  applied_like_delta INTEGER NOT NULL DEFAULT 0,
  applied_favorite_delta INTEGER NOT NULL DEFAULT 0,
  applied_comment_delta INTEGER NOT NULL DEFAULT 0,
  flush_owner VARCHAR(128) NULL,
  flush_started_at TIMESTAMPTZ NULL,
  flushed BOOLEAN NOT NULL DEFAULT FALSE,
  flushed_at TIMESTAMPTZ NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (bucket_start, post_id),
  CONSTRAINT ck_ranking_bucket_flushed_applied CHECK (
    flushed = FALSE OR (
      applied_view_delta = view_delta
      AND applied_like_delta = like_delta
      AND applied_favorite_delta = favorite_delta
      AND applied_comment_delta = comment_delta
    )
  )
);

CREATE INDEX idx_ranking_bucket_flush
  ON ranking_delta_bucket(flushed, bucket_start, updated_at);
```

`applied_*` 支持负增量和正负混合，不用绝对值比较这类 SQL 约束表达部分应用边界；正式实现必须用 application 层的 `PendingDelta` 计算、条件更新和回归测试保证不会重复物化。

```sql
CREATE TABLE ranking_post_state (
  post_id BIGINT PRIMARY KEY,
  public_post_id VARCHAR(32) NULL,
  author_id BIGINT NULL,
  published_at TIMESTAMPTZ NULL,
  topic_ids JSONB NOT NULL DEFAULT '[]',
  view_count BIGINT NOT NULL DEFAULT 0,
  like_count INTEGER NOT NULL DEFAULT 0,
  favorite_count INTEGER NOT NULL DEFAULT 0,
  comment_count INTEGER NOT NULL DEFAULT 0,
  raw_score DOUBLE PRECISION NOT NULL DEFAULT 0,
  hot_score DOUBLE PRECISION NOT NULL DEFAULT 0,
  version BIGINT NOT NULL DEFAULT 0,
  last_bucket_start TIMESTAMPTZ NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT ck_ranking_post_counts_nonnegative CHECK (
    view_count >= 0 AND like_count >= 0 AND favorite_count >= 0 AND comment_count >= 0
  )
);

CREATE INDEX idx_ranking_post_state_hot_score
  ON ranking_post_state(hot_score DESC);

CREATE UNIQUE INDEX idx_ranking_post_state_public_post
  ON ranking_post_state(public_post_id)
  WHERE public_post_id IS NOT NULL;
```

```sql
CREATE TABLE ranking_period_score (
  period_type VARCHAR(16) NOT NULL,
  period_key VARCHAR(32) NOT NULL,
  post_id BIGINT NOT NULL,
  delta_score DOUBLE PRECISION NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (period_type, period_key, post_id),
  CONSTRAINT ck_ranking_period_type CHECK (period_type IN ('DAY', 'WEEK', 'MONTH'))
);

CREATE INDEX idx_ranking_period_score_lookup
  ON ranking_period_score(period_type, period_key, delta_score DESC);
```

## 服务配置要点

Ranking 的服务私有配置由 `runtime` 读取和校验，字段名最终以 `services/zhicore-ranking/configs/` 模板为准。

| 配置 | 作用 | 默认与约束 |
| --- | --- | --- |
| `ranking.pagination.default_size` / `max_size` | 查询默认和最大 size | Java 参考默认 20、最大 100 |
| `ranking.pipeline.bucket_window` | bucket 时间窗 | 本地默认 `10s`，不建议超过 `60s` |
| `ranking.pipeline.flush_interval` | flush worker 周期 | Java 参考 `5s` |
| `ranking.pipeline.flush_batch_size` | 每轮 claim bucket 数 | Java 参考 `200`，需压测校准 |
| `ranking.pipeline.flush_delay` | 避免刷未结束窗口 | 必须大于等于 bucket window 或明确说明 |
| `ranking.weights.*` | view/like/favorite/comment 权重和 half-life | 必须可配置；变更后需 snapshot/rebuild 生效 |
| `ranking.view_dedup.ttl` | 浏览去重窗口 | Java 参考 30 分钟 |
| `ranking.view_cap.*` | 单篇文章浏览分数上限 | 防刷配置，必须可观测 |
| `ranking.snapshot.refresh_interval` | Redis 快照刷新周期 | Java 参考 `60s` |
| `ranking.candidate.posts.enabled/size/refresh_interval/stale_threshold` | 热门候选集 | size 参考 200 |
| `ranking.query.backfill_lock_ttl` | Redis miss 回源回填锁 TTL | 默认 `60s` |
| `ranking.query.empty_cache_ttl` | 空结果占位 TTL | 默认 `60s` |
| `ranking.period.retention.daily/weekly/monthly` | 周期分活跃保留窗口 | 默认 `7d`、`60d`、`365d` |
| `ranking.archive.*` | 归档 cron、保留数量、锁 | 日/周/月分开配置 |
| `ranking.replay.batch_size` | ledger replay 批次 | 必须有上限 |
| `ranking.replay.lock_ttl/drain_timeout` | rebuild 锁 TTL 和 in-flight drain 超时 | 默认 `30m`，长任务必须续期 |

这些配置遵循 `docs/architecture/configuration.md`：生产依赖地址和敏感项通过环境注入，启动前校验，日志只输出脱敏摘要。后台任务启动、停止、panic 和 shutdown 语义遵循 `docs/architecture/runtime-operations.md`。

## 推荐首个实现切片

**切片 1：事件账本 + bucket + 文章总榜查询**

目标：先验证 Ranking 最核心的事实摄入、幂等和可重建权威状态。

- Domain：`RankingEventLedger`、`RankingDeltaBucket`、`RankingPostState`、`RankingMetricType`、`HotScoreCalculator`。
- Application：`IngestRankingEvent`、`FlushRankingBuckets`、`ListHotPosts`、`ListHotPostsWithScore`、`GetPostRank`、`GetPostScore`。
- Ports：`RankingLedgerRepository`、`RankingBucketRepository`、`RankingStateRepository`、`RankingRedisMaterializer`、`ContentPostClient`、`TransactionRunner`、`Clock`。
- Infrastructure：PostgreSQL repository、Redis ZSET store。
- HTTP：`GET /api/v1/ranking/posts/hot`、`GET /api/v1/ranking/posts/hot/scores`、`GET /api/v1/ranking/posts/{postId}/rank`、`GET /api/v1/ranking/posts/{postId}/score`。
- 事件：先接 `content.post.liked`、`content.post.unliked`、`comment.created`、`comment.deleted`，view/favorite 可在同一切片后半补。

**切片 2：Redis snapshot + rebuild-from-ledger**

目标：补齐 Redis 漂移修复和管理端补算。

- 补 `RefreshRankingSnapshots`、`RebuildFromLedger`、replay lock、flush/snapshot scheduler lock。
- HTTP 补 `POST /api/v1/ranking/admin/rebuild-from-ledger`，管理员权限来自 auth context 或 Admin facade。
- 测试覆盖重复事件、bucket 晚到事件、flush Redis 失败后 snapshot 回填、rebuild 后结果一致。

**切片 3：周期榜 + 创作者/话题派生榜**

目标：补日/周/月文章榜，并从文章 state 派生 creator/topic 榜。

- 补 `ranking_period_score`、`PeriodKeyPolicy`、creator/topic Redis 物化。
- HTTP 补文章 `daily/weekly/monthly` 和 `scores`，补 creators/topics hot/rank/score。
- 明确 Topic 服务拆出前 `topic_ids` 仍来自 Content 元数据快照。

**切片 4：热门候选集 + 文章详情 + 归档**

目标：支持 Comment 下游候选同步和前端详情榜。

- 补 `RefreshHotPostCandidates`、`GetHotPostCandidates`。
- HTTP 补 `/api/v1/ranking/posts/hot/candidates` 和 `/posts/hot/details`。
- 补 Content 批量详情 client、Mongo archive 和月榜冷数据回源。

## 迁移风险

- `bucket` flush 如果按整桶总量物化，会重复放大计数；必须使用 `applied_*` 只刷 pending delta。
- RabbitMQ 没有 RocketMQ `hashKey` 的同等默认语义，可以用 routing key、consistent hash exchange 或 consumer 本地 post 分片降低同一内部 `post_id` 的并发冲突；正确性仍必须容忍重复、乱序和迟到事件。
- `RebuildFromLedger` 必须暂停 live ingestion，否则 replay 和实时消费会重复计数。
- Redis 不是权威源；所有 Redis 榜单必须能从 `ranking_post_state` / `ranking_period_score` 或 ledger 重建。
- view 事件的反刷发生在 ledger 前；被拦截浏览不能进入热度账本。
- 元数据补齐失败不能阻塞 ledger/bucket 落账，但会影响 creator/topic 榜，需要补偿回填。
- 热度权重变更后，当前 `hot_score` 和 Redis 榜单需要 snapshot 或 replay 才能完全一致。
- 评论删除事件必须明确 `affectedCount`；根评论批量删除时不能只发 `-1`。
- 月榜冷数据回源需要锁和空结果缓存，避免 Redis miss 后击穿 MongoDB。
- Admin rebuild、archive、snapshot、candidate refresh 都是后台任务，必须有 owner、锁、超时、可观测和停机语义。

## 下一步

- 按 `docs/contracts/http-schema-template.md` 提取 Ranking HTTP 字段级 contract，重点固定：
  - `page` 从 `0` 开始，`size/limit` 上限。
  - `postId` 对外统一使用 Content `public_id`；handler 入站解析为内部 `post_id`，出站列表和 `HotScore.entityId` 返回 `public_id`。
  - `HotScore` response 中 `entityId`、`score`、`rank` 的类型和空榜语义。
  - 周榜 `year/week` 的 ISO week 规则和错误码。
  - `rebuild-from-ledger` 的管理员权限和返回结构。
- 生成 Ranking migration 草案，重点核对：
  - `ranking_event_ledger.event_id` 主键、`bucket_start`、`occurred_at,event_id` replay 索引。
  - `ranking_delta_bucket.applied_*`、claim 字段和 flush 索引。
  - `ranking_post_state.public_post_id` 快照、`version` 乐观锁、非负计数约束、`topic_ids JSONB`。
  - `ranking_period_score` 的 `period_type/period_key/post_id` 主键和 score lookup 索引。
- 提取 Content / Comment 事件 payload contract：
  - Ranking 需要 `eventId`、`publicPostId`、可选内部 `postId`、`occurredAt`、明确 `metricType/delta` 或可确定映射字段。
  - `content.post.viewed` 需要 `viewerId` 或 `ipHash` 支持 view dedup。
  - `comment.deleted` 需要 `affectedCount`。
- 设计 RabbitMQ 分片策略，明确同一内部 `post_id` 事件的局部顺序优化和乱序容忍测试。
- 先实现“事件账本 + bucket + 文章总榜查询”最小切片，再推进 snapshot/replay、周期榜和候选集。

## 实现前检查清单

### Contract 确认

- [ ] Content 事件是否包含 `publicPostId`、`publishedAt`，并在可用时携带内部 `postId`？
- [ ] Comment 删除事件是否明确 `affectedCount`？
- [ ] 浏览事件是否包含 `viewerId` 或 `ipHash`？

### 并发验证

- [ ] 两个 worker 同时 flush 同一 bucket 是否不会重复计数？
- [ ] 同一 `post_id` 的相邻 bucket 并发更新 state 是否不会覆盖？
- [ ] 晚到事件追加到已 flushed bucket 后是否只应用 pending delta？

### 边界测试

- [ ] `publicPostId` 解析 not found 是否进入 DLQ？
- [ ] Redis 全部失败时查询是否能回源 PostgreSQL？
- [ ] Replay 期间新到事件是否会被 nack/requeue 或暂停消费？
- [ ] 周期榜超出活跃窗口时是否查询 MongoDB archive？

### 监控埋点

- [ ] Bucket flush 延迟 P99。
- [ ] Redis 回源频率和回填失败次数。
- [ ] View dedup 拦截率和 view cap 拦截率。
- [ ] Rebuild 执行时间、失败次数和 in-flight drain 超时次数。

## DDD 设计总结

1. **Ranking 的源事实是热度账本**：文章、评论、用户和话题源数据仍归各自服务。
2. **ledger/bucket/state/Redis 分层不可混淆**：ledger 记录事实，bucket 压缩增量，state/period 是权威结果，Redis 是可重建物化。
3. **flush 幂等依赖 applied delta**：晚到事件和重试只能物化新增部分，不能重复刷整桶。
4. **榜单允许最终一致**：秒级到十秒级延迟可接受，查询优先 Redis，修复靠 snapshot 和 replay。
5. **热门候选集是下游消费视图**：服务于 Comment 缓存判定，不是第二套热度算法。
6. **后台任务必须有 owner 和锁**：flush、snapshot、archive、candidate、replay 都要有明确生命周期和并发保护。
