# Ranking 运行期 Resilience 设计

本文固定 `zhicore-ranking` 首次微服务化实现时必须落地的 timeout、retry、熔断、降级、健康检查和依赖故障语义。全局规则见 `docs/architecture/runtime-operations.md`；本文只记录 Ranking 自己的业务取舍。

当前状态：本文是设计事实源，不表示 Go 代码已经实现。首次实现 Ranking HTTP、consumer、worker、scheduler、adapter 或 runtime wiring 前，必须为对应行补配置项、adapter 行为测试或 application 编排测试。

## 核心原则

- PostgreSQL 是 `ranking_event_ledger`、`ranking_delta_bucket`、`ranking_post_state` 和 `ranking_period_score` 的权威源；PostgreSQL 不可用时 Ranking 核心读写不可用。
- Redis 只保存可重建榜单、候选集、view dedup、view cap、backfill lock、empty cache 和运行期锁，不保存不可重建的热度事实。
- RabbitMQ 是事件输入通道，不是 Ranking 已接受事实的存储；事件只有在 PostgreSQL 事务提交后才 ack。
- MongoDB archive 只服务冷历史榜单，不参与实时分数计算。
- Content / User 是展示和元数据 owner。Ranking 不伪造文章、作者或话题事实；公开榜单过滤主路径使用 Ranking 本地 `public_visible` 投影，Content 回源只用于详情补齐、事件缺字段解析、repair / reconcile 兜底。
- resilience policy 在 runtime wiring 声明，不能散落到 handler、application、repository 或 adapter。
- 熔断、降级、Redis 回源、consumer retry、DLQ、snapshot 失败和 rebuild partial failure 必须可观测。

## 时间预算

| 场景 | 建议总预算 | 说明 |
| --- | --- | --- |
| 文章总榜 / 分数 / rank 查询 | `1s` 到 `2s` | 优先 Redis；Redis miss / 不可用时受控回源 PostgreSQL。 |
| 文章详情榜查询 | `2s` 到 `3s` | Ranking 查询和 Content 批量详情共享预算。 |
| 候选集查询 | `500ms` 到 `1s` | 优先 Redis candidate；必要时回源 PostgreSQL 构建小批候选。 |
| 周期榜查询 | `1s` 到 `2s` | 活跃窗口 Redis / PostgreSQL；冷历史 MongoDB archive。 |
| 单条事件摄入 | `1s` 到 `3s` | 含 decode、可选 Content publicId 解析、view dedup 和 PostgreSQL 事务。 |
| 单条可见性事件投影 | `1s` 到 `3s` | 含 decode、可选 Content publicId 解析、projection inbox 和 PostgreSQL 投影事务。 |
| bucket flush 单批次 | `5s` 到 `15s` | 按批次 claim，单个 bucket 使用短事务。 |
| snapshot / candidate refresh 单轮 | `5s` 到 `30s` | 使用临时 Redis key + rename；失败保留上一版。 |
| archive 单轮 | `10s` 到 `60s` | 按周期和批次限制，失败可重试。 |
| admin rebuild 提交请求 | `2s` 到 `5s` | 只启动 / 受理任务或同步做快速校验；长任务异步执行并暴露状态。 |

下游 client 的单次 timeout 不得超过上游剩余 deadline。没有明确上游 deadline 时，runtime 必须为 HTTP handler、consumer 和 worker 设置默认 timeout。

## Provider / Operation 矩阵

| Provider | Operation | 调用方 / 场景 | Timeout 基线 | Retry | Circuit breaker key | Max in-flight | 降级策略 | 幂等 / 一致性 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `postgres` | `ranking.ingest_tx` | 插入 ledger、upsert bucket | `1s..3s` | 不在事务外盲重试；依赖 `event_id` 幂等和 broker retry | `postgres.ranking.ingest_tx` | 按 DB pool 和 consumer concurrency 限制 | 失败则 nack / requeue 或进入 DLQ；不 ack 已失败事件 | `event_id` 主键保证重复事件 no-op。 |
| `postgres` | `ranking.visibility_projection` | Content 可见性 / 元数据事件投影 | `1s..3s` | 不在事务外盲重试；依赖 projection inbox 幂等和 broker retry | `postgres.ranking.visibility_projection` | 按 DB pool 和 consumer concurrency 限制 | 失败则 nack / requeue 或进入 DLQ；不更新本地可见性投影 | `ranking_projection_event_inbox.event_id` 保证重复事件 no-op。 |
| `postgres` | `ranking.flush_tx` | bucket flush、state / period 更新、applied 推进 | `1s..3s` | worker 可按批次退避重试；单事务不盲重试 | `postgres.ranking.flush_tx` | flush worker 并发配置 | 本轮失败，bucket 保持可重试 | `applied_*` 只推进 pending delta。 |
| `postgres` | `ranking.query` | hot / score / rank / active period 查询 | `1s..3s` | 对连接抖动最多 2 次总尝试 | `postgres.ranking.query` | 查询路径独立限并发 | 返回 `SERVICE_DEGRADED` / `1004`；不得伪装为空榜 | Redis miss 后回源的权威来源。 |
| `postgres` | `ranking.rebuild` | reset materialized state、ledger replay | `3s..10s` per batch | 按 rebuild batch retry 策略 | `postgres.ranking.rebuild` | rebuild 单实例互斥 | 失败返回 rebuild failedStage；释放 barrier / lock | 不删除 ledger；下一次 rebuild 从头开始。 |
| `redis` | `ranking.zset.query` | 读取榜单 ZSET、候选集、meta | `50ms..200ms` | 不阻塞主查询重试；最多一次快速重试 | `redis.ranking.zset.query` | Redis 查询独立限并发 | miss / error 后回源 PostgreSQL 或 MongoDB；记录 degraded | Redis 不是事实源。 |
| `redis` | `ranking.zset.materialize` | flush 后增量更新、snapshot 原子替换 | `100ms..500ms` | snapshot/job 内退避重试 | `redis.ranking.zset.materialize` | materializer worker 限并发 | flush 后 Redis 失败不回滚 PG；等待 snapshot / rebuild 回填 | PG state / period 已提交才可更新 Redis。 |
| `redis` | `ranking.visibility_remove` | 文章不可公开后移除榜单 / 候选集成员 | `100ms..500ms` | consumer 后置动作最多一次快速重试；reconcile / snapshot 再修复 | `redis.ranking.visibility_remove` | materializer worker 限并发 | Redis 失败不回滚 PG projection；等待 snapshot / visibility reconcile 回填 | PostgreSQL `public_visible=false` 是过滤权威。 |
| `redis` | `ranking.view_dedup` | `content.post.viewed` 去重 | `50ms..200ms` | 不重试放大延迟；可短时本机严格 fallback | `redis.ranking.view_dedup` | view consumer 独立限并发 | 短时本机 fallback；超过窗口 ack 并丢弃 view，不写 ledger | view 是弱热度信号，不能因 dedup 不可确认而无限计数。 |
| `redis` | `ranking.view_cap` | 单篇文章日浏览上限 | `50ms..200ms` | 不重试放大延迟；可短时本机严格 fallback | `redis.ranking.view_cap` | view consumer 独立限并发 | 短时本机 fallback；超过窗口 ack 并丢弃 view，不写 ledger | 防刷边界不可长期 fail-open。 |
| `redis` | `ranking.lock` | rebuild、snapshot、candidate、archive、backfill lock | `50ms..200ms` | 最多一次快速重试 | `redis.ranking.lock` | 锁操作独立限并发 | 可用 PostgreSQL advisory lock 替代；否则跳过本轮或返回 `1004` | 不能多实例并发执行非幂等任务。 |
| `rabbitmq` | `ranking.consume` | Content / Comment 热度、可见性和元数据事件输入 | broker 配置 + handler `1s..3s` | 按 consumer retry / DLQ 策略 | `rabbitmq.ranking.consume` | consumer concurrency 配置 | RabbitMQ 不可用时暂停摄入；HTTP 查询继续服务已有榜单 | 事件未 ack 前可重投；已落账 / 已投影事件靠 `event_id` no-op。 |
| `content-service` | `post.resolve_public_id` | 事件 payload 缺内部 `postId`、HTTP path 入站解析 | `500ms..1s` | 只读调用最多 2 次总尝试 | `content.post.resolve_public_id` | consumer / HTTP 分别限并发 | transient 失败：consumer nack / query 返回 `1004`；not found / deleted：DLQ 或业务 404 | 不把无法解析的 publicId 写进 ledger。 |
| `content-service` | `post.batch_get_details` | `ListHotPostsWithDetails`、元数据补齐 | `1s..2s` | 最多 2 次总尝试 | `content.post.batch_get_details` | 详情查询限并发 | 详情 endpoint 返回 `1004`；分数 / ID endpoint 不依赖详情 | Ranking 不伪造文章详情。 |
| `content-service` | `post.metadata_backfill` | author、publishedAt、topicIds 补齐 | `1s..3s` | job 内退避重试 | `content.post.metadata_backfill` | backfill worker 限并发 | 本轮失败，保留缺失元数据并记录 degraded；creator/topic 投影延后 | 热度账本不因展示元数据缺失回滚。 |
| `content-service` | `post.visibility_reconcile` | 修复可见性投影漂移 | `1s..3s` | job 内退避重试 | `content.post.visibility_reconcile` | reconcile worker 限并发 | 本轮失败，保留现有投影并打 degraded metric；公开榜单仍按本地投影过滤 | reconcile 是兜底，不是查询主路径。 |
| `user-service` | `user.batch_get_summary` | 创作者榜展示摘要 | `1s..2s` | 最多 2 次总尝试 | `user.batch_get_summary` | 查询路径限并发 | contract 允许时省略摘要或返回占位；否则返回 `1004` | User 资料不是 Ranking 权威事实。 |
| `mongo` | `ranking.archive.write` | 日 / 周 / 月榜归档 | `3s..10s` | job 内退避重试 | `mongo.ranking.archive.write` | archive worker 限并发 | 归档失败不影响在线榜单；记录 retry / alert | archive 可重试，唯一键保证幂等写入。 |
| `mongo` | `ranking.archive.read` | 冷历史周期榜查询 | `2s..5s` | 最多 2 次总尝试 | `mongo.ranking.archive.read` | 冷查询限并发 | 返回 `SERVICE_DEGRADED` / `1004`，不得伪装为空历史榜 | 活跃窗口仍走 Redis / PostgreSQL。 |

## API / Worker 降级矩阵

| API / use case | 关键依赖不可用 | 策略 |
| --- | --- | --- |
| `ListHotPosts` / `ListHotPostsWithScore` | Redis 不可用或 miss | 回源 PostgreSQL，写 backfill / degraded metric；回源失败返回 `1004` / `503`。 |
| `ListHotPosts` / `ListHotPostsWithScore` | PostgreSQL 不可用 | 失败 `1004` / `503`；不得返回空榜伪装成功。 |
| `ListHotPostsWithDetails` | Content 详情不可用 | 失败 `1004` / `503`；不把只有 score 的结果伪装成详情结果。 |
| `GetPostRank` / `GetPostScore` | Content publicId 解析不可用 | 失败 `1004` / `503`；Content 确认不存在时返回业务 not found。 |
| 周期榜活跃窗口查询 | Redis 不可用或 miss | 回源 `ranking_period_score` 并尝试回填；PostgreSQL 不可用时失败。 |
| 周期榜冷历史查询 | MongoDB 不可用 | 失败 `1004` / `503`；不伪装为空历史榜。 |
| `GetHotPostCandidates` | Redis candidate 不可用或 stale | 优先回源 PostgreSQL 构建小批候选；回源失败返回 degraded，让 Comment 使用本地旧缓存或空候选。 |
| `IngestRankingEvent` | PostgreSQL 不可用 | nack / requeue 或进入 DLQ；不 ack 未落账事件。 |
| `IngestRankingEvent` | Content publicId 解析 transient 失败 | nack / requeue；Content 确认 not found / deleted 时 DLQ 并告警。 |
| `IngestRankingEvent` view dedup / cap | Redis 短时不可用 | 使用本机严格 fallback；超过降级窗口后 ack 并丢弃 view，不写 ledger。 |
| `IngestRankingEvent` 非 view 事件 | Redis 不可用 | 正常写 ledger / bucket；非 view 指标不依赖 Redis dedup。 |
| `ApplyContentVisibilityEvent` | PostgreSQL 不可用 | nack / requeue 或进入 DLQ；不 ack 未更新 projection 的事件。 |
| `ApplyContentVisibilityEvent` | Content publicId 解析 transient 失败 | nack / requeue；Content 确认 not found 时 DLQ 或 no-op 告警，不更新 projection。 |
| `ApplyContentVisibilityEvent` | Redis 移除榜单成员失败 | PostgreSQL projection 已提交则不回滚；记录 `redis_visibility_remove_failed`，等待 snapshot / visibility reconcile 修复。 |
| `ApplyContentVisibilityEvent` | 事件乱序或迟到 | 以 `occurredAt` / `aggregateVersion` 的目标 contract 规则判定；不能用消费时间覆盖较新的 projection。 |
| `FlushRankingBuckets` | PostgreSQL 不可用 | 本轮失败，bucket 保持未 flushed 或 owner 超时释放；不丢弃 pending delta。 |
| `FlushRankingBuckets` | Redis materialize 不可用 | PostgreSQL commit 不回滚；记录 `redis_materialize_failed`，等待 snapshot / rebuild 修复。 |
| `RefreshRankingSnapshots` | Redis 不可用 | 本轮失败，保留旧 key；如果 Redis 整体不可用，HTTP 查询回源 PostgreSQL。 |
| `RefreshHotPostCandidates` | Redis 不可用 | 本轮失败，保留旧 candidate；连续失败告警。 |
| `ReconcilePostVisibility` | Content 不可用 | 本轮失败并记录 degraded；不作为公开榜单查询主过滤，也不把未知状态当成可见。 |
| `ArchiveRankings` | MongoDB 不可用 | 本轮失败，按 retry / backoff 重试；不影响在线榜单。 |
| `RebuildFromLedger` | Redis lock 不可用且无 PostgreSQL lock fallback | 拒绝启动，返回 `1004` / `503`；不得无锁 rebuild。 |
| `RebuildFromLedger` | Redis refresh 在 PG rebuild 后失败 | 返回 partial failedStage，保留 PG 权威状态，告警并要求 snapshot 重试。 |

## Redis 故障策略

Redis 在 Ranking 中按职责区分故障语义：

| Redis 职责 | 故障策略 |
| --- | --- |
| 榜单 / 候选集 ZSET 查询 | 回源 PostgreSQL 或 MongoDB；回源失败才返回 degraded。 |
| flush 后增量物化 | 不回滚 PostgreSQL；记录失败，等待 snapshot / rebuild 回填。 |
| 可见性移除 | 不回滚 PostgreSQL projection；记录失败，等待 snapshot / visibility reconcile 收敛。 |
| view dedup / view cap | 短窗口本机严格 fallback；超过窗口后 view 事件 ack 丢弃，不写 ledger。 |
| 分布式锁 / backfill lock | 可用 PostgreSQL advisory lock 替代；没有替代时跳过后台任务或拒绝 admin rebuild。 |

建议配置项：

| 配置 | 默认建议 | 说明 |
| --- | --- | --- |
| `RANKING_REDIS_DEGRADED_VIEW_WINDOW` | `60s` | view dedup / cap Redis 不可用时允许本机 fallback 的最大窗口。 |
| `RANKING_REDIS_QUERY_BACKFILL_LOCK_TTL` | `60s` | Redis miss 回源回填锁 TTL。 |
| `RANKING_REDIS_EMPTY_CACHE_TTL` | `60s` | 空榜短 TTL 占位。 |
| `RANKING_REDIS_LOCK_FALLBACK` | `postgres_advisory_lock` 或 `none` | Redis lock 不可用时是否允许 PostgreSQL advisory lock 替代。 |

## RabbitMQ / Consumer 策略

- RabbitMQ 不可用时 Ranking 暂停摄入新事件，已有榜单继续对外查询。
- RabbitMQ 默认不让 Ranking HTTP readiness 失败；如果 deployment 把 consumer 独立部署，可为 consumer readiness 显式要求 RabbitMQ ready。
- consumer handler 只有在 PostgreSQL 事务提交后 ack。
- duplicate `event_id` 直接 ack，不进入 DLQ。
- transient Content / PostgreSQL / handler timeout 使用 nack / requeue 或 broker retry；超过阈值进入 DLQ。
- Content 可见性 / 元数据事件消费失败时不能 ack；重复 `event_id` 命中 projection inbox 后 ack no-op。
- DLQ 必须保留 event id、event type、source service、publicPostId、reason、traceId，不记录敏感 payload。
- rebuild barrier 开启时 consumer 不写业务表；可以暂停拉取或 nack / requeue 等待 broker 重投。

## 健康检查

`/health/live` 只检查进程存活。

`/health/ready` 的默认 HTTP 服务策略：

- PostgreSQL 连接和轻量 ping 必须 ready。
- 必要配置必须 ready，例如 Redis、RabbitMQ、MongoDB、Content/User client、bucket window、权重、worker concurrency 等配置格式。
- Redis、RabbitMQ、MongoDB、Content 和 User 默认不作为 HTTP readiness 硬阻断项；它们进入 health details 和 metrics。

原因：

- Redis 不可用时查询可回源 PostgreSQL，flush 物化可由 snapshot 修复。
- RabbitMQ 不可用时摄入暂停，但现有榜单仍可查询。
- MongoDB 只影响冷历史榜单和归档。
- Content / User 主要影响解析、详情和摘要，不应摘除所有 Ranking 读流量。

如果部署策略要求 worker / consumer pod 在依赖不可用时摘流，必须显式配置：

| 部署形态 | readiness 建议 |
| --- | --- |
| HTTP + worker 同进程 | PostgreSQL 硬依赖；Redis / RabbitMQ / MongoDB 进入 degraded details。 |
| consumer-only | PostgreSQL + RabbitMQ 硬依赖；Content client 配置必须有效。 |
| snapshot / candidate worker | PostgreSQL + Redis lock / Redis materialize 硬依赖，或启用 PostgreSQL lock fallback。 |
| archive worker | PostgreSQL + MongoDB 硬依赖。 |
| rebuild worker | PostgreSQL + rebuild lock 依赖；Redis refresh 失败允许 partial failedStage，但不应静默成功。 |

## Metrics 和告警

最低 metrics：

| metric | 标签 | 说明 |
| --- | --- | --- |
| `ranking_downstream_requests_total` | `provider,operation,result` | 下游调用结果，result 包含 `success/timeout/circuit_open/error/degraded`。 |
| `ranking_downstream_duration_ms` | `provider,operation` | 下游调用耗时。 |
| `ranking_degraded_total` | `operation,reason` | 降级次数，例如 `redis_query_fallback_pg`、`view_dedup_local_fallback`、`candidate_stale`。 |
| `ranking_events_consumed_total` | `eventType,result` | consumer 处理结果，result 包含 `accepted/duplicate/dropped/retry/dlq`。 |
| `ranking_bucket_flush_total` | `result` | bucket flush 成功、失败、跳过。 |
| `ranking_bucket_flush_lag_seconds` | `worker` | 最老未 flushed bucket 的滞后。 |
| `ranking_redis_materialize_failed_total` | `operation` | flush / snapshot / candidate / visibility remove 写 Redis 失败。 |
| `ranking_visibility_projection_total` | `eventType,result` | 可见性 / 元数据投影处理结果，result 包含 `applied/duplicate/retry/dlq/stale_ignored`。 |
| `ranking_visibility_projection_lag_seconds` | `eventType` | 最新已处理可见性事件与当前时间的滞后。 |
| `ranking_visibility_reconcile_total` | `result` | 可见性 reconcile 结果。 |
| `ranking_snapshot_refresh_total` | `rankingType,result` | snapshot 刷新结果。 |
| `ranking_candidate_stale_seconds` | `source` | 候选集距离上次成功刷新时间。 |
| `ranking_rebuild_total` | `stage,result` | rebuild 阶段结果。 |
| `ranking_archive_total` | `periodType,result` | 归档任务结果。 |
| `ranking_dlq_total` | `eventType,reason` | DLQ 事件计数。 |

告警至少覆盖：

- PostgreSQL unavailable 或 ready check 失败。
- consumer lag / DLQ 持续增长。
- 最老未 flushed bucket 超过配置阈值。
- Redis materialize 连续失败，或 query fallback 比例持续升高。
- 可见性 projection lag 持续增长，或 visibility 事件 DLQ 增长。
- candidate 超过 `2 * stale_threshold`。
- rebuild partial failure。
- archive 连续失败导致活跃窗口外数据无法查询。

日志必须带 `requestId` / `traceId`、`operation`、`provider`、`durationMs`、`attempt`、`eventId`、`bucketStart`、`postId`、`degradedReason`，不得记录原始 IP、完整消息 payload、Authorization、cookie 或下游敏感错误堆栈。

## 配置准入

首次实现前，Ranking runtime 配置至少覆盖：

- HTTP server read、write、idle、header 和 shutdown timeout。
- PostgreSQL query timeout、transaction timeout、pool size。
- Redis dial/read/write timeout、pool size、backfill lock TTL、empty cache TTL、view degraded fallback window。
- RabbitMQ queue、routing key、consumer concurrency、prefetch、handler timeout、retry / DLQ 策略、consumer shutdown timeout。
- Content/User client base URL、timeout、retry、breaker、max-in-flight。
- MongoDB archive read / write timeout、pool size、collection 名称。
- bucket window、flush interval、flush delay、flush batch size、claim TTL、worker concurrency。
- snapshot / candidate refresh interval、stale threshold、candidate size。
- visibility reconcile interval、batch size、lag alert threshold、Redis removal retry policy。
- archive cron、batch size、retry backoff。
- rebuild lock TTL、drain timeout、batch size、operation status retention。
- 每个 circuit breaker 的统计窗口、最小请求数、失败率阈值、连续失败阈值、打开时长和半开探测数。

配置加载、默认值和校验必须遵守 `docs/architecture/configuration.md`：handler、repository、adapter 和普通构造函数不得读取环境变量。

## 测试准入

首次实现相关切片时至少覆盖：

- 下游 timeout / circuit open 映射到 application 语义错误，不透出底层错误文本。
- 只读 client retry 不超过配置次数，最终业务结果只计一次失败。
- Redis query miss / error 回源 PostgreSQL；PostgreSQL 失败不伪装为空榜。
- view dedup / cap Redis 不可用时，本机 fallback 和超过窗口 ack-drop 分支可测。
- `IngestRankingEvent` 在 PostgreSQL 失败时不 ack；duplicate `event_id` ack no-op。
- Content 可见性事件在 PostgreSQL 失败时不 ack；duplicate projection `event_id` ack no-op。
- 可见性事件把 `public_visible=false` 写入 PG 后，Redis 移除失败不回滚 PG，snapshot / reconcile 后榜单过滤生效。
- Content 详情 / reconcile 不可用时，公开榜单查询仍按本地 `public_visible` 投影过滤，不临时放行未知文章。
- flush 成功后 Redis materialize 失败不回滚 PG，snapshot 后可回填。
- bucket late event 追加到已 flushed bucket 后只应用新增 pending delta。
- rebuild barrier 期间 consumer 不写业务表；rebuild crash 后锁过期能恢复。
- MongoDB archive read/write 不可用分别返回 degraded 或任务 retry。
- readiness 在 PostgreSQL 不可用时 non-ready；Redis/RabbitMQ/MongoDB 默认只进入 degraded details。
- breaker key 按 `provider + operation` 分开，Content 详情失败不熔断 Content publicId 解析。
