# Ranking 服务设计

本文是 `zhicore-ranking` 的服务入口文档，只保留职责边界、关键结论和专题索引。领域模型、事件投影、运行期韧性、schema 和实现切片不在 README 展开。

## 事实来源

- Java `zhicore-ranking` controller：`RankingController`。
- Java `zhicore-ranking` domain / application / infrastructure：`HotScore`、`PostStats`、`CreatorStats`、`RankingMetricType`、`HotScoreCalculator`、ledger / flush / replay / snapshot / archive / candidate 相关 service 和 repository。
- Java `zhicore-ranking/src/main/resources/db/ranking-schema.sql`。
- `../zhicore-microservice/docs/architecture/zhicore-ranking-detailed-design.md` 和 `../zhicore-microservice/zhicore-ranking/README.md`。
- Go 目标 Content / Comment 事件设计：`content.post.*`、`comment.*`。
- 本目录专题文档和 decision-log。

## 专题文档

| 文档 | 内容 |
| --- | --- |
| [decision-log/2026-06-29-ranking-design-decisions.md](decision-log/2026-06-29-ranking-design-decisions.md) | 记录职责边界、ledger / bucket / state / Redis 分层、事件输入、rebuild、候选集和运行韧性的关键取舍。 |
| [domain-model.md](domain-model.md) | DDD 目标模型、标识策略、聚合、值对象、领域服务、热度公式和浏览去重。 |
| [application-and-ports.md](application-and-ports.md) | Application 用例、ports、事务边界、rebuild / archive 流程和 Go 包落点。 |
| [data-events-projections.md](data-events-projections.md) | 消费事件、热度 ledger、Content 可见性 / 元数据投影、Redis 收敛和 projection schema 草案。 |
| [event-ordering-and-partitioning.md](event-ordering-and-partitioning.md) | RabbitMQ 消费分片、同一文章局部顺序优化、retry / DLQ 和乱序容忍测试。 |
| [query-materialization.md](query-materialization.md) | 查询路径、Redis key、Redis miss 回源、snapshot、候选集、归档和 API 保留范围。 |
| [runtime-resilience.md](runtime-resilience.md) | timeout、retry、熔断、降级、readiness、依赖故障、metrics 和测试准入。 |
| [schema-and-implementation.md](schema-and-implementation.md) | 数据归属、schema 草案、配置要点、实现切片、风险和实现前检查清单。 |

## 职责边界

`zhicore-ranking` 拥有：

- 热度事实账本。
- 窗口增量聚合。
- 文章当前热度状态。
- 周期榜单分数。
- Redis ZSET 物化榜单。
- 热门文章候选集。
- 历史榜单归档。
- Ranking 自己的 rebuild / replay 运维流程。

Ranking 不拥有文章、评论、用户、话题或关注关系的源事实。它只保存源服务 ID、必要元数据快照、可见性投影和热度计算结果。文章是否存在、是否公开、作者资料、话题名称和评论内容仍由 Content、Comment、User 或未来 Topic 归属服务解释。

DDD 设计用于指导 Go 目标实现，不表示当前 Go 代码已经完成。Java 侧已经落地的 `ledger -> bucket -> state/period -> Redis` 链路是主要事实来源，但 Go 实现按本仓库 `api/http -> application -> domain/ports -> infrastructure` 的依赖方向重新落点。

## 关键结论

- Ranking 内部持久化和分数计算统一使用 Content 内部 `post_id BIGINT` 作为 opaque reference；HTTP path 和 response 使用 Content `public_id`。
- `ranking_event_ledger` 只记录会产生热度 delta 的事实；可见性 / 元数据事件不写热度 ledger。
- `ranking_delta_bucket` 必须通过 `applied_*` 只物化 pending delta，不能重复刷整桶。
- `ranking_post_state` 和 `ranking_period_score` 是权威结果；Redis 只是可重建投影。
- 公开榜单必须过滤 `ranking_post_state.public_visible = TRUE`；该字段由 Content 可见性事件驱动的本地投影维护。
- Content 回源只用于详情补齐、事件缺字段解析、repair / reconcile 兜底，不作为公开榜单查询主过滤路径。
- Redis 榜单成员移除失败不回滚 PostgreSQL projection，等待 snapshot / candidate refresh / visibility reconcile / rebuild 收敛。
- Rebuild 必须通过 barrier / lock 暂停 live ingestion，不能和实时消费并行写 materialized state。
- 热门候选集服务于 Comment 等下游缓存判定，不是前端分页榜单的替代品。
- 第一阶段默认不生产关键跨服务事件；如未来需要 `ranking.hot_candidates.updated`，必须新增 ranking event contract。

## 当前 API 范围

Ranking 保留当前前端和服务间使用的路径。字段级 request / response、错误码、权限和返回空榜语义已经按 `docs/contracts/http-schema-template.md` 提取到 `services/zhicore-ranking/api/http/`，当前状态为草案，待 Go handler / contract test 验证。

| API 族 | 说明 |
| --- | --- |
| `/api/v1/ranking/posts/hot` | 文章总榜，返回 Content `public_id` 列表。 |
| `/api/v1/ranking/posts/hot/details` | 文章总榜详情，批量补 Content 详情。 |
| `/api/v1/ranking/posts/hot/scores` | 文章总榜分数和 rank，`entityId` 使用 Content `public_id`。 |
| `/api/v1/ranking/posts/hot/candidates` | 热门文章候选集。 |
| `/api/v1/ranking/posts/daily|weekly|monthly` | 文章周期榜。 |
| `/api/v1/ranking/posts/{postId}/rank|score` | 单篇文章排名和分数，`{postId}` 是 Content `public_id`。 |
| `/api/v1/ranking/creators/*` | 创作者榜、分数和排名。 |
| `/api/v1/ranking/topics/*` | 话题榜、分数和排名。 |
| `/api/v1/ranking/admin/rebuild-from-ledger` | 管理员全量补算。 |

分页页码从 `0` 开始；`size/limit` 必须按配置限制最大值。周榜使用 ISO week-based year 和 week number。服务间候选集可同时返回内部 `postId` 和 `publicPostId`，但外部 HTTP response 不暴露内部 `post_id`。

字段级 contract 见 `services/zhicore-ranking/api/http/README.md` 和 `services/zhicore-ranking/api/http/endpoints/ranking-api.md`。

## 跨服务依赖

| 服务 | 依赖方式 |
| --- | --- |
| Content | 事件输入、文章元数据、批量文章详情、作者 ID、发布时间、话题引用和公开状态。 |
| Comment | 默认只通过事件输入；缺字段时优先修事件 contract，不同步读评论库。 |
| User | 创作者榜需要用户摘要时通过 User contract 批量补齐；Ranking 不保存用户资料权威数据。 |
| Notification | 不直接依赖 Ranking；Notification 可消费源事件自行创建通知。 |
| Admin | `rebuild-from-ledger` 可由 Admin facade 委托 Ranking，同时保留 Ranking 自己的管理员权限校验。 |

## 首个实现切片

首个切片只闭合“事件账本 + bucket + 文章总榜查询”：

- Domain：`RankingEventLedger`、`RankingDeltaBucket`、`RankingPostState`、`RankingMetricType`、`HotScoreCalculator`。
- Application：`IngestRankingEvent`、`ApplyContentVisibilityEvent`、`FlushRankingBuckets`、`ListHotPosts`、`ListHotPostsWithScore`、`GetPostRank`、`GetPostScore`。
- Ports：`RankingLedgerRepository`、`RankingBucketRepository`、`RankingStateRepository`、`RankingProjectionInboxRepository`、`RankingRedisMaterializer`、`ContentPostClient`、`TransactionRunner`、`Clock`。
- HTTP：`GET /api/v1/ranking/posts/hot`、`GET /api/v1/ranking/posts/hot/scores`、`GET /api/v1/ranking/posts/{postId}/rank`、`GET /api/v1/ranking/posts/{postId}/score`。
- 事件：`content.post.liked`、`content.post.unliked`、`content.post.published`、`content.post.deleted`、`content.post.visibility_changed`、`comment.created`、`comment.deleted`；view / favorite 可在同一切片后半补。

完整实现切片、schema、配置和检查清单见 [schema-and-implementation.md](schema-and-implementation.md)。

## 实现前必读

实现任一 Ranking handler、consumer、worker、adapter 或 runtime wiring 前，至少先读：

1. [decision-log/2026-06-29-ranking-design-decisions.md](decision-log/2026-06-29-ranking-design-decisions.md)
2. [domain-model.md](domain-model.md)
3. [application-and-ports.md](application-and-ports.md)
4. [data-events-projections.md](data-events-projections.md)
5. [event-ordering-and-partitioning.md](event-ordering-and-partitioning.md)
6. [query-materialization.md](query-materialization.md)
7. [runtime-resilience.md](runtime-resilience.md)
8. [schema-and-implementation.md](schema-and-implementation.md)
