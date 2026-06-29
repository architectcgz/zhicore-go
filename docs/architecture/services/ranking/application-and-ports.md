# Ranking Application、Ports 与事务设计

本文记录 `zhicore-ranking` 的 application 用例、consumer-side ports、事务边界和 Go 包落点。领域模型见 [domain-model.md](domain-model.md)，事件与投影见 [data-events-projections.md](data-events-projections.md)。

## Application 用例

**命令 / 消费用例（Commands / Consumers）**：

- `IngestRankingEvent`：由 RabbitMQ consumer 调用，校验事件、可选执行 view 去重 / 上限、写 `ranking_event_ledger`、upsert `ranking_delta_bucket`。
- `ApplyContentVisibilityEvent`：消费 Content 可见性 / 元数据事件，写 projection inbox，更新 `ranking_post_state.public_visible` 和元数据快照。
- `FlushRankingBuckets`：claim 可刷 bucket，计算 pending delta，更新 `ranking_post_state`、`ranking_period_score`，事务提交后增量物化 Redis。
- `RefreshRankingSnapshots`：从 `ranking_post_state` / `ranking_period_score` 重建当前总榜和活跃日 / 周 / 月榜 Redis。
- `RefreshHotPostCandidates`：从 `ranking:posts:hot` 生成 `ranking:posts:hot:candidates` 和 meta。
- `RebuildFromLedger`：管理员触发，暂停 live ingestion，清空物化层，从 `ranking_event_ledger` 顺序重放，刷新 Redis 和候选集。
- `ArchiveRankings`：按日 / 周 / 月定时把 Redis 或 PostgreSQL 来源榜单归档到 MongoDB。
- `BackfillPostMetadata`：补齐缺失 `author_id`、`published_at`、`topic_ids`，并触发相关文章的 creator / topic 投影修复。

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
| `RankingPeriodScoreRepository` | 增量更新周期分、查询周期 top、清理 / 重建 |
| `RankingReplayRepository` | reset materialized state、replay barrier、rebuild 事务所需组合操作 |
| `RankingProjectionInboxRepository` | 记录可见性 / 元数据事件消费幂等，避免非热度事件重复更新投影 |
| `RankingQueryStore` | 读取文章 / 创作者 / 话题总榜和周期榜，可以由 Redis + PostgreSQL fallback 实现 |
| `HotPostCandidateStore` | 候选集原子替换、meta 读写、stale 标记 |
| `RankingArchiveStore` | MongoDB 历史榜单归档和冷数据读取 |

**机制端口**：

| Port | 职责 |
| --- | --- |
| `TransactionRunner` | 显式事务边界 |
| `Clock` | UTC 当前时间、业务日期和 ISO week 计算 |
| `RankingLockManager` | replay、scheduler、bucket claim 和 monthly backfill 锁 |
| `RankingEventConsumerCheckpoint` | 如 Go consumer 需要本地 queue checkpoint，可选；业务幂等仍以 ledger / event_id 为准 |
| `MetricsRecorder` | 低基数指标记录；不能影响业务控制流 |

**缓存、事件和外部服务端口**：

| Port | 职责 |
| --- | --- |
| `RankingRedisMaterializer` | 增量更新 / 原子替换 Redis ZSET、删除 / 回填 key |
| `ViewDedupStore` | 浏览去重和单篇文章浏览分数上限 |
| `ContentPostClient` | 解析 `public_id` / 内部 `post_id`、批量补齐文章元数据、公开状态和详情 |
| `CommentClient` | 第一阶段通常不需要；只有评论事件缺少必要字段且不能从 payload 获取时再补 |
| `RankingEventDecoder` | 将 Content / Comment 事件 payload 映射成 Ranking 内部 `RankingEvent` |

端口不能暴露 `*gorm.DB`、`*redis.Client`、Mongo driver、RabbitMQ delivery、HTTP DTO 或外部 SDK 类型。底层 duplicate key、not found、Redis nil、MQ nack 由 infrastructure adapter 翻译为 module-local 语义，再由 application 决定 no-op、重试或公开错误。

## 事件摄入事务

```text
事务前：
  decode RabbitMQ JSON
  + 校验 eventId、eventType、publicId、internalId、occurredAt、delta
  + 将 internalId 作为 Content 内部 post_id opaque reference
      - 缺失或非法：记录 producer contract 错误并投递 DLQ
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

消费侧不直接写 Redis、不直接更新 `ranking_post_state`。`event_id` 是消费幂等键；进入 Ranking 内部后的 `post_id` 是局部顺序和 bucket 聚合键。事件缺少 `internalId` 时不能落账，也不在高频消费路径上通过 `publicId` 同步补字段。

可见性 / 元数据投影是独立于热度 ledger 的消费路径：它写 projection inbox，更新 `ranking_post_state.public_visible` 和元数据快照，Redis 移除失败不回滚 PostgreSQL。完整流程见 [data-events-projections.md](data-events-projections.md)。

## Bucket flush 事务

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

Flush worker 按 `ranking.pipeline.flush_interval` 周期运行，默认参考值为 `5s`。每轮只 claim 满足条件的 bucket；如果 `ranking.pipeline.bucket_window` 是 `10s`，且 `ranking.pipeline.flush_delay >= bucket_window`，刚写入的当前窗口不会立即物化，必须等窗口结束并经过 flush delay 后，才进入可 claim 范围。

`ranking_post_state` 更新必须使用 version、post 级锁或等价条件写，避免两个 worker 对同一内部 `post_id` 的相邻 bucket 并发覆盖。`bucket` 的 `applied_*` 是幂等关键字段，不能省略。

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

## Rebuild from ledger

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

进入 replay 窗口的新消息不写业务表：consumer 可以 nack / requeue，也可以停止消费等待 broker 重新投递。rebuild 使用 ledger 中保存的 `bucket_start` 重放，`occurred_at` 只作为排序和审计字段。rebuild 不能删除 `ranking_event_ledger`。

执行结果写入 `ranking_rebuild_operation`，状态查询至少返回 `replayedEvents`、`rebuiltPosts`、`durationMs` 和 `failedStage`。如果候选集刷新失败，保留旧候选集并记录告警，operation 标记为 `PARTIAL_FAILED`。若 rebuild 进程 crash，锁超时后 consumer 可以自动恢复；下一次 rebuild 从头开始，不要求断点续传。

## Archive

归档任务从 Redis 或 PostgreSQL source store 读取日 / 周 / 月榜，写 MongoDB。归档不是实时查询权威源；归档失败不影响在线榜单，但必须可重试。月榜冷数据查询可以先查 Redis，缺失时用 lock 回源 MongoDB 并回填 Redis；超出 Redis 保留范围时直接查 MongoDB。

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
- `runtime`：读取配置，启动 / 停止 consumer 和 job，管理 lifecycle context、panic、shutdown 和 readiness。
