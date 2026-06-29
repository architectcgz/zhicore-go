# Ranking Schema、配置与实现切片

本文记录 `zhicore-ranking` 的数据归属、schema 草案、服务配置、首个实现切片、风险和实现前检查清单。正式 migration 以 `services/zhicore-ranking/migrations/20260629042338_create_ranking_core_tables.*.sql` 为准。

## 数据归属

Ranking 拥有：

- `ranking_event_ledger`
- `ranking_delta_bucket`
- `ranking_post_state`
- `ranking_period_score`
- `ranking_projection_event_inbox`
- MongoDB `ranking_archive`
- Redis ZSET 榜单、候选集、view 去重、锁和空结果缓存

## Schema 草案

### `ranking_event_ledger`

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

### `ranking_delta_bucket`

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

### `ranking_post_state`

```sql
CREATE TABLE ranking_post_state (
  post_id BIGINT PRIMARY KEY,
  public_post_id VARCHAR(32) NULL,
  author_id BIGINT NULL,
  published_at TIMESTAMPTZ NULL,
  topic_ids JSONB NOT NULL DEFAULT '[]',
  public_visible BOOLEAN NOT NULL DEFAULT FALSE,
  content_status VARCHAR(32) NULL,
  visibility_reason VARCHAR(64) NULL,
  visibility_updated_at TIMESTAMPTZ NULL,
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
  ON ranking_post_state(public_visible, hot_score DESC);

CREATE UNIQUE INDEX idx_ranking_post_state_public_post
  ON ranking_post_state(public_post_id)
  WHERE public_post_id IS NOT NULL;
```

`public_visible` 默认 `FALSE`，避免 Ranking 在尚未收到 Content 发布 / 恢复 / 可见性事件前把未知文章放入公开榜单。`content_status` 和 `visibility_reason` 只保存 Ranking 过滤所需的状态快照，不替代 Content 生命周期权威查询。

### `ranking_projection_event_inbox`

可见性 / 元数据 projection inbox 草案见 [data-events-projections.md](data-events-projections.md)。正式 migration 可按实现命名调整，但必须保留：

- `event_id` 唯一幂等。
- 按 `post_id + occurred_at` 查询投影处理历史。
- 可区分热度 ledger 和非热度 projection 事件。

### `ranking_period_score`

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
| `ranking.archive.*` | 归档 cron、保留数量、锁 | 日 / 周 / 月分开配置 |
| `ranking.replay.batch_size` | ledger replay 批次 | 必须有上限 |
| `ranking.replay.lock_ttl/drain_timeout` | rebuild 锁 TTL 和 in-flight drain 超时 | 默认 `30m`，长任务必须续期 |

这些配置遵循 `docs/architecture/configuration.md`：生产依赖地址和敏感项通过环境注入，启动前校验，日志只输出脱敏摘要。后台任务启动、停止、panic 和 shutdown 语义遵循 `docs/architecture/runtime-operations.md`。

## 推荐首个实现切片

**切片 1：事件账本 + bucket + 文章总榜查询**

目标：先验证 Ranking 最核心的事实摄入、幂等、公开可见性过滤和可重建权威状态。

- Domain：`RankingEventLedger`、`RankingDeltaBucket`、`RankingPostState`、`RankingMetricType`、`HotScoreCalculator`。
- Application：`IngestRankingEvent`、`ApplyContentVisibilityEvent`、`FlushRankingBuckets`、`ListHotPosts`、`ListHotPostsWithScore`、`GetPostRank`、`GetPostScore`。
- Ports：`RankingLedgerRepository`、`RankingBucketRepository`、`RankingStateRepository`、`RankingProjectionInboxRepository`、`RankingRedisMaterializer`、`ContentPostClient`、`TransactionRunner`、`Clock`。
- Infrastructure：PostgreSQL repository、Redis ZSET store。
- HTTP：`GET /api/v1/ranking/posts/hot`、`GET /api/v1/ranking/posts/hot/scores`、`GET /api/v1/ranking/posts/{postId}/rank`、`GET /api/v1/ranking/posts/{postId}/score`。
- 事件：先接 `content.post.liked`、`content.post.unliked`、`content.post.published`、`content.post.deleted`、`content.post.visibility_changed`、`comment.created`、`comment.deleted`，view / favorite 可在同一切片后半补。

**切片 2：Redis snapshot + rebuild-from-ledger**

- 补 `RefreshRankingSnapshots`、`RebuildFromLedger`、replay lock、flush / snapshot scheduler lock。
- HTTP 补 `POST /api/v1/ranking/admin/rebuild-from-ledger`，管理员权限来自 auth context 或 Admin facade。
- 测试覆盖重复事件、bucket 晚到事件、flush Redis 失败后 snapshot 回填、rebuild 后结果一致。

**切片 3：周期榜 + 创作者 / 话题派生榜**

- 补 `ranking_period_score`、`PeriodKeyPolicy`、creator / topic Redis 物化。
- HTTP 补文章 `daily/weekly/monthly` 和 `scores`，补 creators / topics hot / rank / score。
- 明确 Topic 服务拆出前 `topic_ids` 仍来自 Content 元数据快照。

**切片 4：热门候选集 + 文章详情 + 归档**

- 补 `RefreshHotPostCandidates`、`GetHotPostCandidates`。
- HTTP 补 `/api/v1/ranking/posts/hot/candidates` 和 `/posts/hot/details`。
- 补 Content 批量详情 client、Mongo archive 和月榜冷数据回源。

## 实现风险

- `bucket` flush 如果按整桶总量物化，会重复放大计数；必须使用 `applied_*` 只刷 pending delta。
- RabbitMQ 没有 RocketMQ `hashKey` 的同等默认语义，可以用 routing key、consistent hash exchange 或 consumer 本地 post 分片降低同一内部 `post_id` 的并发冲突；正确性仍必须容忍重复、乱序和迟到事件。
- `RebuildFromLedger` 必须暂停 live ingestion，否则 replay 和实时消费会重复计数。
- Redis 不是权威源；所有 Redis 榜单必须能从 `ranking_post_state` / `ranking_period_score` 或 ledger 重建。
- 公开榜单必须过滤 `ranking_post_state.public_visible=true`；查询主路径不能逐条回源 Content 临时判断可见性。
- view 事件的反刷发生在 ledger 前；被拦截浏览不能进入热度账本。
- 元数据补齐失败不能阻塞 ledger / bucket 落账，但会影响 creator / topic 榜，需要补偿回填。
- 热度权重变更后，当前 `hot_score` 和 Redis 榜单需要 snapshot 或 replay 才能完全一致。
- 评论删除事件必须明确 `affectedCount`；根评论批量删除时不能只发 `-1`。
- 月榜冷数据回源需要锁和空结果缓存，避免 Redis miss 后击穿 MongoDB。
- Admin rebuild、archive、snapshot、candidate refresh 都是后台任务，必须有 owner、锁、超时、可观测和停机语义。
- 运行期 timeout、retry、熔断、降级、健康检查和依赖故障语义以 [runtime-resilience.md](runtime-resilience.md) 为准。

## 下一步

- 用 Go handler / contract test 验证 `services/zhicore-ranking/api/http/` 下的 Ranking HTTP 字段级 contract 草案。
- 基于 Ranking migration 补 repository / application 测试，重点验证 ledger 幂等、bucket pending delta、state 可见性过滤、projection inbox 和 period score 更新。
- 提取 Content / Comment 事件 payload contract。
- 设计 RabbitMQ 分片策略，明确同一内部 `post_id` 事件的局部顺序优化和乱序容忍测试。
- 按 [runtime-resilience.md](runtime-resilience.md) 落地 Ranking runtime 配置、health details、metrics 和 adapter / worker 测试。
- 先实现“事件账本 + bucket + 文章总榜查询”最小切片，再推进 snapshot / replay、周期榜和候选集。

## 实现前检查清单

### Contract 确认

- [ ] Content 事件是否包含 `publicPostId`、`publishedAt`，并在可用时携带内部 `postId`？
- [ ] Content 可见性事件是否包含 `publicVisible`、`oldVisibility/newVisibility`、`reason` 和可用于乱序保护的版本或时间？
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
