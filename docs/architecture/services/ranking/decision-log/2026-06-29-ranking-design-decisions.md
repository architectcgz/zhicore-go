# Ranking 设计决策日志

本文按 `docs/architecture/services/ranking/README.md` 和相关专题文档中已经固定的 Ranking 设计重建关键 decision log。它不是逐字 transcript，也不替代 Ranking README 或专题文档；本文用于复盘每个关键取舍为什么成立。

相关事实源：

- [Ranking 服务设计](../README.md)
- [Ranking 领域模型设计](../domain-model.md)
- [Ranking Application、Ports 与事务设计](../application-and-ports.md)
- [Ranking 数据事件与投影设计](../data-events-projections.md)
- [Ranking 查询、缓存与物化设计](../query-materialization.md)
- [Ranking Schema、配置与实现切片](../schema-and-implementation.md)
- [服务边界](../../../service-boundaries.md)
- [运行期规范](../../../runtime-operations.md)
- [事件契约规范](../../../../contracts/events.md)

| # | 决策项 | Question | Decision | Rationale | Follow-up |
| --- | --- | --- | --- | --- | --- |
| 1 | Ranking 职责边界 | Ranking 是否拥有文章、评论、用户或话题源事实？ | 不拥有。Ranking 只拥有热度事实账本、bucket、state、period score、Redis 榜单、候选集、归档和 replay / rebuild 运维流程。 | 文章、评论、用户和话题的可见性、详情和生命周期归各自 owner；Ranking 只解释热度贡献和榜单物化，避免跨库读写和事实漂移。 | 查询详情时通过 Content/User 等 contract 批量补齐，不在 Ranking domain 保存外部资料权威副本。 |
| 2 | 内部 post_id 与 public_id | Ranking 内部计算用 Content 内部 `post_id`，还是对外 `public_id`？ | 内部落库、分数计算和 Redis ZSET 成员使用 Content 内部 `post_id` opaque reference；HTTP path 和 response 使用 Content `public_id`。 | 内部 `post_id` 更适合高频聚合、索引和 Redis member；外部 `public_id` 服务前端路由和公开 contract。两者职责不同，不能混用。 | Ranking handler 入站解析 `public_id`，出站列表转换为 `public_id`；Content / Comment 事件 payload 统一用 `publicId` 和 `internalId` 且均必填。 |
| 3 | `public_post_id` 快照 | Ranking 是否在 state 表保存 `public_post_id`？ | 可以保存为稳定快照字段，但不是 Ranking 权威数据。缺失或过期时以 Content contract 返回为准。 | 保存快照能减少查询时批量解析成本；但 `public_id` 仍归 Content 所有，Ranking 不能成为第二事实源。 | `ranking_post_state.public_post_id` 建唯一稀疏索引；元数据补齐失败需要补偿任务。 |
| 4 | 热度源事实 | Redis 榜单能否作为热度事实源？ | 不能。`ranking_event_ledger` 是热度事实输入账本，`ranking_post_state` / `ranking_period_score` 是权威结果，Redis 只是可重建投影。 | Redis 适合读路径和物化榜单，但会丢失历史和审计能力；权威状态必须能从 ledger / PG 重建。 | 所有 Redis key 必须有 snapshot / rebuild 回填路径。 |
| 5 | Ledger 状态机 | `ranking_event_ledger` 是否需要 `PENDING/FAILED/DEAD` 状态？ | 不需要。ledger 只追加；消费成功定义为同一事务内完成 ledger 插入和 bucket 聚合。 | ledger 是已接受事实，不是 producer outbox。失败消息由 RabbitMQ retry / DLQ 处理；重复事件由 `event_id` 主键 no-op。 | consumer 对 transient error nack / requeue；重复 `event_id` 直接 ack。 |
| 6 | 事件幂等键 | Ranking 幂等应依赖 source operation id 还是 event id？ | 以事件 envelope 的 `eventId` 作为 `ranking_event_ledger.event_id` 主键。 | 事件级幂等最稳定，能覆盖 broker 重投、consumer crash 和上游重试；source operation id 只作为审计或关联字段。 | Content / Comment 事件 contract 必须保证 `eventId` 稳定唯一。 |
| 7 | 事件输入形态 | Ranking 消费 delta 还是当前总数快照？ | 消费事实 delta，不消费当前总数快照。点赞、收藏、评论删除等负增量必须由源服务显式发送。 | Ranking 无法可靠推断源服务当前总数快照背后的增删事实；delta 能让 ledger replay、周期分和幂等语义闭合。 | `comment.deleted` 必须携带 `affectedCount`；取消点赞 / 取消收藏必须发 `-1`。 |
| 8 | 浏览去重位置 | view dedup 应该写入 ledger 后再过滤，还是进入 ledger 前过滤？ | 进入 ledger 前过滤。被 dedup 或 view cap 拦截的 view ack 但不写 ledger。 | ledger 的 `delta` 不允许为 0；防刷拦截不是热度事实，不应污染可重放账本。 | `ViewDedupStore` 使用 Redis；记录 dedup / cap metrics。 |
| 9 | IP 处理 | Ranking 是否接收原始 IP 做匿名浏览去重？ | 不接收原始 IP。由 Gateway 或 Content 提供 `ipHash`，建议 HMAC/SHA-256 截断。 | Ranking 不需要也不应保存可识别 IP；匿名去重允许有限误判，不追求唯一访客精确性。 | 事件 contract 需要固定 `viewerId?` / `ipHash?` 字段语义。 |
| 10 | Bucket window | bucket window 是否固定写死？ | 不写死。默认参考 `10s`，低流量可放宽到 `30s`，不建议超过 `60s`，必须配置化。 | 窗口越短越接近实时但写入更多；越长越省写但榜单滞后。配置化便于压测后调整。 | 配置项 `ranking.pipeline.bucket_window`；flush delay 必须大于等于 bucket window 或明确说明。 |
| 11 | Bucket flush 幂等 | flush 能否按整桶累计值刷到 state？ | 不能。必须通过 `applied_*` 只物化 pending delta。 | 重试、晚到事件和 worker crash 都会让同一 bucket 被多次处理；刷整桶会重复放大计数。 | application test 必须覆盖重复 flush、晚到事件和正负混合 delta。 |
| 12 | 晚到事件处理 | 已 flushed bucket 收到晚到事件时，是否挪到当前窗口？ | 不挪。仍追加到原 `(bucket_start, post_id)`，把 `flushed=false`、`flushed_at=NULL`，下轮只刷新增 pending。 | 这样 live flush 和 `RebuildFromLedger` 对周期榜归属一致，不需要额外 repair 表。 | bucket 更新必须用条件更新保护 `applied_*`，避免并发重复应用。 |
| 13 | State 并发控制 | 多 worker 更新同一 post state 时如何避免覆盖？ | 使用 version、post 级锁或等价条件更新；不能靠 broker 顺序假设保证正确性。 | RabbitMQ 不天然提供 RocketMQ `hashKey` 等价保证，同一 `post_id` 事件可能乱序或并发处理。 | 可用 routing key / consistent hash exchange / 本地分片优化顺序，但正确性必须容忍乱序。 |
| 14 | 周期榜存储 | 日/周/月榜是否每次从 ledger 聚合查询？ | 不实时扫 ledger。使用 `ranking_period_score` 独立写模型。 | 热榜查询是高频读路径，直接扫 ledger 会把审计账本变成在线查询瓶颈。 | 维护 `period_type + period_key + post_id` 主键和活跃窗口清理任务。 |
| 15 | 周期保留窗口 | 周期分是否永久保留在 PostgreSQL？ | 不永久保留。日榜保留最近 7 天，周榜保留最近 60 天覆盖的 ISO 周，月榜保留最近 365 天覆盖的月份；超出归档到 MongoDB。 | 在线 PG 只服务活跃查询和近期回源，长期历史交给冷归档降低表膨胀。 | retention 配置化；归档成功后才能清理活跃窗口外数据。 |
| 16 | 创作者 / 话题榜 | 创作者榜和话题榜第一阶段是否建独立 PG 权威表？ | 不建。第一阶段从 `ranking_post_state.author_id/topic_ids` 派生后物化到 Redis。 | 当前创作者 / 话题榜是文章热度的派生视图，没有独立事实生命周期。 | 未来需要独立作者/话题历史状态时再新增 state 表。 |
| 17 | Snapshot 原子性 | 刷新 Redis 榜单时能否直接覆盖正式 key？ | 不直接覆盖。使用临时 key 构建后 rename 原子替换；失败保留上一版。 | 直接覆盖可能让查询看到半成品或空榜。临时 key + rename 能保证读路径稳定。 | `RankingRedisMaterializer` 必须封装原子替换和失败保留策略。 |
| 18 | Redis miss | Redis miss 时返回空榜还是回源？ | 总榜和活跃周期榜优先回源 PostgreSQL 并回填 Redis；超出活跃窗口查 MongoDB archive。回源为空时写短 TTL empty cache。 | Redis 不是权威源，miss 不等于没有数据；但回源需要防击穿。 | 使用 `ranking:backfill:*` 锁和 `ranking:empty:*` 短 TTL。 |
| 19 | 候选集定位 | 热门文章候选集是不是前端分页榜单的替代品？ | 不是。候选集服务于 Comment 等下游缓存判定，只从总榜截取前 N 条并带 meta。 | 候选集是下游消费视图，不维护第二套公式，也不承诺完整分页语义。 | `GetHotPostCandidates` 返回 version、generatedAt、sourceCount、candidateSize、stale 等元信息。 |
| 20 | 候选集 stale | 候选集过期时是否删除旧 key？ | 不删除旧 key。刷新失败保留上一版；stale 是查询时派生状态。 | 删除旧 key 会让下游在短故障中失去可用候选；旧数据比空白更利于降级。 | 连续失败 3 次告警；超过 `2 * stale_threshold` 后 Comment 可降级为空候选或本地旧缓存。 |
| 21 | Archive 权威性 | MongoDB archive 是否参与实时分数计算？ | 不参与。archive 只用于历史榜单冷数据查询。 | 归档是查询和审计视图，不应反向影响实时 state / period score。 | 归档失败不影响在线榜单，但必须可重试、可观测。 |
| 22 | Rebuild 语义 | `RebuildFromLedger` 能否和 live ingestion 并行？ | 不能。rebuild 必须通过 barrier / lock 暂停 live ingestion，等待 in-flight drain 后从 ledger 重放。 | replay 和实时消费同时写 bucket/state 会重复计数或覆盖状态。 | consumer 看到 barrier 后停止拉取或 nack/requeue；rebuild crash 后锁过期可恢复消费。 |
| 23 | Rebuild 来源 | rebuild 是否清空 ledger 后重建？ | 不能删除 ledger。rebuild 清空 materialized state、bucket、period 和当前 Redis，再从 ledger 重放。 | ledger 是审计和重放事实源，删除会失去恢复能力。 | rebuild 执行结果写入 `ranking_rebuild_operation`，状态查询返回 `replayedEvents`、`rebuiltPosts`、`durationMs`、`failedStage` 等字段。 |
| 24 | 热度公式 | Go 第一阶段是否重新设计热度公式？ | 不重新设计。沿用 Java half-life 衰减公式，权重和半衰期配置化。 | 迁移期不应改变榜单语义；配置化允许后续压测和产品调整。 | 权重变更后需要 snapshot 或 replay 才能完全一致。 |
| 25 | 未发布 / 隐藏文章 | `published_at IS NULL` 时是否可进入公开榜单？ | `published_at IS NULL` 时公式可令 `timeDecay=1.0`，但未发布、删除、撤回、下架或隐藏文章不应进入公开榜单。 | 公式处理缺失时间只是防御；公开可见性仍归 Content 状态决定。Ranking 只能保存本地投影用于过滤，不能成为 Content 生命周期源事实。 | 过滤主路径来自 `ranking_post_state.public_visible`；Content 回源只用于详情补齐、事件缺字段解析和 repair / reconcile。 |
| 26 | 评论点赞热度 | `comment.liked` / `comment.unliked` 是否计入文章热度？ | 第一阶段不消费。若产品要求计入，必须先扩展指标、权重、bucket 列、replay 规则和事件 contract。 | 评论点赞属于评论互动，不一定等价于文章热度；在 Ranking 内临时推断会破坏指标可解释性。 | 事件 contract 阶段单独决策，不在 Ranking consumer 内隐藏实现。 |
| 27 | Content / Comment 事件字段 | Ranking 是否能每条事件同步查询 Content / Comment 补字段？ | 事件 payload 统一使用 `publicId` / `internalId` 命名，且两者均必填。Content 事件由 Content 直接携带内部 `post_id`；Comment 在创建评论前通过 Content contract 校验文章并保存 Content 内部 `post_id` opaque reference，随后在 Comment 事件中携带。缺字段时视为 producer contract 错误进入 DLQ，不把同步补查变成常态。 | 高频事件每条同步补查会放大延迟和失败面；Ranking 是下游但内部聚合键是 Content 内部 `post_id`，因此事件必须携带下游可直接落账的 opaque reference。`publicId` 仍服务审计、DLQ、repair 和对外转换。 | Ranking 使用 `internalId` 落账；HTTP path、repair 和 reconcile 可解析 `publicId`，但事件摄入主路径不按 `publicId` 同步补字段。 |
| 28 | Comment 同步依赖 | Ranking 是否同步读取 Comment 服务？ | 第一阶段通常不需要。Comment 只通过事件输入；缺字段时优先修 Comment 事件。 | Ranking 不应把 Comment 查询变成热度摄入的在线依赖。 | `CommentClient` 仅作为未来必要时的可选端口。 |
| 29 | Ranking 生产事件 | Ranking 是否生产关键跨服务事件？ | 第一阶段默认不生产关键事件。热门候选集通过同步查询或定时拉取暴露给 Comment。 | 候选集是可重建视图，不是权威业务事实；事件广播会增加 consumer 幂等和一致性成本。 | 如未来需要 `ranking.hot_candidates.updated`，必须新增 ranking event contract 并定义是否可丢失。 |
| 30 | HTTP API 兼容 | Go Ranking 是否完全兼容 Java 旧数据和 API 形态？ | 不要求兼容 Java 旧数据，但 API 形态需要按目标前端 contract 固定。 | Go 重建阶段以目标 contract 为准；旧 Java 是能力参考，不是字段事实源。 | 字段级 schema 已提取到 `services/zhicore-ranking/api/http/`，当前为草案，待 handler / contract test 验证。 |
| 31 | 分页起点 | Ranking page 分页从 0 还是 1 开始？ | Ranking 保留 page 从 `0` 开始，`size/limit` 必须配置最大值。 | Ranking 当前前端和 Java 参考已有 0-based 语义；切换会影响调用方。 | HTTP schema 中明确 page 起点、最大 size 和空榜语义。 |
| 32 | 首个实现切片 | Ranking 首先实现完整榜单体系还是最小链路？ | 先实现“事件账本 + bucket + 文章总榜查询”，再推进 snapshot/rebuild、周期榜、候选集和归档。 | Ranking 风险集中在幂等、bucket flush、Redis 可重建和公开可见性过滤；先闭合核心链路更容易验证。 | 切片 1 覆盖 `content.post.liked/unliked`、`content.post.published/deleted/visibility_changed`、`comment.created/deleted`，view/favorite 可后续补。 |
| 33 | Ranking 运行韧性专题归属 | Ranking 的 timeout、retry、熔断、降级、健康检查和依赖故障语义写在哪里？ | 新增 `runtime-resilience.md` 作为 Ranking 运行韧性专题事实源；README 和 decision-log 只保留关键结论和入口。 | Ranking 同时有 HTTP 查询、RabbitMQ consumer、bucket flush、snapshot、candidate、archive 和 rebuild，故障语义跨 PostgreSQL、Redis、RabbitMQ、MongoDB、Content/User client，必须集中维护。 | 首次实现任一 adapter、worker、consumer 或 runtime wiring 前必须先读该文档。 |
| 34 | Ranking readiness 依赖 | Redis、RabbitMQ、MongoDB 或 Content/User 不可用时，Ranking HTTP readiness 是否失败？ | 首期默认只有 PostgreSQL 是 HTTP readiness 硬依赖；Redis、RabbitMQ、MongoDB、Content/User 默认进入 degraded details 和 metrics，不摘除全部 HTTP 流量。 | Redis 可回源 PG/Mongo；RabbitMQ 影响新事件摄入但已有榜单仍可查；Mongo 只影响冷历史；Content/User 主要影响解析、详情和摘要。直接摘除 HTTP 会扩大故障影响。 | consumer-only、archive-worker、snapshot-worker 等独立部署可用显式配置把对应依赖设为 readiness 硬依赖。 |
| 35 | Redis 故障语义 | Redis 不可用时 Ranking 是否全站失败？ | 不全站失败。榜单查询回源 PostgreSQL / MongoDB；flush 后 Redis materialize 失败不回滚 PG；view dedup / cap 短时本机严格 fallback，超过窗口后 ack 丢弃 view，不写 ledger；锁不可用时用 PG advisory lock 或跳过 / 拒绝高风险任务。 | Redis 不是热度事实源，但承担物化、去重、防刷和锁。查询可以回源；非 view 事件不依赖 Redis；view 是弱热度信号，不能因 dedup 不可确认长期放大计数。 | 需要配置 `RANKING_REDIS_DEGRADED_VIEW_WINDOW`、backfill lock TTL、empty cache TTL 和 lock fallback。 |
| 36 | RabbitMQ 故障语义 | RabbitMQ 不可用时 Ranking 是否不可用？ | HTTP 查询不因 RabbitMQ 不可用失败；consumer 暂停摄入或重试。事件只有在 PostgreSQL 事务提交后 ack，重复 `event_id` 直接 ack no-op。 | RabbitMQ 是输入通道，不是已接受事实源。暂停摄入会让榜单变旧，但已有 state / Redis / PG 仍可查询。 | consumer lag、retry 和 DLQ 必须有 metrics / alert；consumer-only readiness 可要求 RabbitMQ ready。 |
| 37 | Content 解析依赖 | Ranking 能否在事件缺少 `internalId` 时按 `publicId` 同步解析后继续落账？ | 不能。事件缺少 `internalId` 或 `internalId` 非法时进入 DLQ，并记录 producer contract 错误；只有 HTTP path、repair 和 reconcile 兜底路径可以同步解析 `publicId`。 | 事件摄入是高频主路径，不能把 Content 同步解析变成常态依赖；Ranking ledger / bucket 以内部 `post_id` 为聚合键，必须在消费前拿到明确的 opaque reference。 | Content 和 Comment producer contract test 必须覆盖 `internalId` 必填。 |
| 38 | Rebuild 依赖故障 | Rebuild 时 Redis 或锁不可用怎么办？ | rebuild 必须持有 Redis lock 或配置的 PostgreSQL advisory lock；无锁时拒绝启动。若 PG rebuild 成功后 Redis refresh 失败，operation 标记 `PARTIAL_FAILED` 并记录 `failedStage`，保留 PG 权威状态并告警。 | 无锁 rebuild 会和 live ingestion 或其他 rebuild 重复计数；Redis refresh 失败不应抹掉已重建的 PG 权威状态，也不能静默宣称完全成功。 | Admin rebuild schema 已在 `admin-rebuild.md` 和 HTTP contract 固定。 |
| 39 | 文章可见性同步机制 | 文章删除、撤回、下架、隐藏等是否靠 Ranking 查询时回源 Content 过滤？ | 不靠查询主路径回源。Content 通过 `content.post.published`、`content.post.deleted`、`content.post.visibility_changed`、`content.post.tags.updated` 等事件驱动 Ranking 本地 visibility / metadata projection；公开榜单查询、snapshot 和候选集按该投影过滤。 | 查询时逐条回源 Content 会把榜单可用性绑到 Content 实时查询，并产生 N+1、熔断扩散和不可预测延迟。事件投影让 Ranking 在 Content 短暂不可用时仍能按最后已知可见性稳定服务。 | Content provider 侧需要固定 `visibility_changed` payload；Ranking 需要 visibility reconcile 作为兜底。 |
| 40 | 可见性事件是否写热度 ledger | `content.post.deleted` / `visibility_changed` 是否进入 `ranking_event_ledger`？ | 不进入热度 ledger。它们写 `ranking_projection_event_inbox` 或等价幂等表，并更新 `ranking_post_state.public_visible`、`content_status`、`visibility_reason`、`visibility_updated_at`。 | `ranking_event_ledger` 只记录会产生热度 delta 的事实；可见性变化没有 `MetricDelta`，放入同一 ledger 会破坏 delta 非零、不重放为热度分的约束。 | schema 草案保留 `ranking_projection_event_inbox`；正式 migration 时可按实现命名调整但语义必须保留。 |
| 41 | Redis 榜单移除失败 | 文章变为不可公开后，Redis ZSET / candidate 移除失败是否回滚 PG projection？ | 不回滚。PostgreSQL visibility projection 已提交后即为 Ranking 过滤权威；Redis 移除失败记录 degraded metric，等待 snapshot、candidate refresh、visibility reconcile 或 rebuild 收敛。 | Redis 是可重建投影，不应决定业务可见性事务成败。回滚 PG 会让 Ranking 明知文章不可公开却继续按旧状态服务。 | 查询和 snapshot 都必须过滤 `public_visible=true`；Redis 失败要有 `redis_visibility_remove_failed` 指标和告警。 |
| 42 | Admin rebuild 操作状态 | `rebuild-from-ledger` 是否只返回 accepted，还是要保存可查询状态？ | 保存 `ranking_rebuild_operation`，`POST` 返回 `operationId`，管理员通过 `GET /api/v1/ranking/admin/rebuild-operations/{operationId}` 查询状态。 | rebuild 是长任务和高风险管理操作，只返回 accepted 会失去审计、排障和 partial failure 可见性。Ranking 是 rebuild owner，必须保存状态事实。 | 首期 `force=true` 不允许；必须持有 Redis lock 或 PostgreSQL advisory lock fallback；状态 schema 见 `admin-rebuild.md` 和 HTTP contract。 |

## 需要继续决策的问题

- Ranking HTTP 字段级 schema 已进入草案；后续需要用 handler / contract test 验证 `HotScore.entityId`、`rank`、`score`、空榜和错误码。
- Content / Comment 事件 payload 已提取为草案；后续需要用 producer / consumer contract test 验证。当前策略：事件统一用 `publicId` / `internalId`，两者均必填；Ranking 使用 `internalId` 落账，`publicId` 用于审计、DLQ、repair 和对外转换。
- RabbitMQ 分片策略已在 `event-ordering-and-partitioning.md` 固定为“正确性不依赖 MQ 顺序；首期 direct/local keyed；扩容后 Ranking 私有 shard router”。后续需要用 consumer tests 验证。
- Admin `rebuild-from-ledger` 已固定权限、审计、互斥锁和状态查询 schema；后续需要 handler / worker / repository 测试验证。
