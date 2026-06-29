# Ranking 事件顺序与分片策略

本文固定 `zhicore-ranking` 消费 Content / Comment 事件时的 RabbitMQ 分片、局部顺序优化和乱序容忍设计。通用事件 contract 见 `docs/contracts/events.md`，运行期降级见 [runtime-resilience.md](runtime-resilience.md)。

当前状态：设计事实源，不表示 Go consumer 已实现。

## 结论

- Ranking 正确性不依赖 RabbitMQ 提供同一文章的严格顺序。
- `event_id` 幂等、`ranking_event_ledger` 主键、`ranking_projection_event_inbox` 主键、bucket pending delta 和 PostgreSQL 行级锁才是正确性边界。
- 同一内部 `post_id` 的局部顺序只是性能和冲突优化，不是唯一安全机制。
- 公共跨服务 exchange 仍使用 `zhicore.events` topic exchange；不改变 provider 事件 routing key。
- 首期可以直接消费 `zhicore.events` 绑定队列；扩容后引入 Ranking 私有 shard router，把事件转入 `zhicore.ranking.ingest` 分片队列。
- visibility / metadata 事件按 `aggregateVersion` 优先、`occurredAt` 兜底处理乱序；热度事件按 delta 和 event id 幂等处理，晚到事件可追加到 bucket 后再 flush。

## 为什么不要求 MQ 严格顺序

RabbitMQ 单队列内对单 consumer 可以提供投递顺序，但一旦增加 consumer concurrency、多实例、retry、nack/requeue、DLQ redrive 或 worker 并发，同一文章事件仍可能乱序、重复或迟到。

Ranking 必须把这些情况当成常态：

- broker 重投导致重复消息。
- 同一文章的 `liked` / `unliked` 跨 consumer 并发处理。
- 已 flushed bucket 收到迟到事件。
- Content `visibility_changed` 旧事件晚于新事件到达。
- rebuild barrier 开启时消息重新入队后再次投递。

因此实现必须先保证乱序正确，再用分片减少冲突。

## 首期消费模式

首期使用一个 Ranking-owned queue：

```text
exchange: zhicore.events (topic)
queue: ranking.events
binding keys:
  content.post.*
  comment.*
```

处理流程：

```text
RabbitMQ delivery
-> RankingEventDecoder
-> resolve partition key
-> optional local keyed worker
-> application use case
-> PostgreSQL transaction commit
-> ack
```

规则：

- consumer ack 只能在 PostgreSQL 事务提交后执行。
- duplicate `event_id` 命中 ledger / inbox 后 ack no-op。
- transient PostgreSQL / Content resolve / handler timeout 使用 nack / retry / DLQ 策略。
- 本地 keyed worker 只能优化单进程内同一 `post_id` 的处理顺序，不能作为跨实例正确性假设。
- 如果运行多个 Ranking consumer 实例，必须假设同一 `post_id` 可能并发落到不同实例。

### 单条消费与 bucket 聚合

- consumer 入口逐条处理 RabbitMQ delivery；同一个 consumer 连续收到多条 event 时也逐条执行，不先在内存里批量聚合。
- 每条 event 单独 decode、校验、执行 application use case、提交 PostgreSQL 事务后 ack。
- 聚合发生在 PostgreSQL `ranking_delta_bucket`，不是 consumer 内存里批量聚合；每条已接受事件先按 `event_id` 写入 `ranking_event_ledger` 做幂等，再按 `(bucket_start, post_id)` upsert bucket 并累加 delta。
- 整体链路是“逐条消费 + bucket 聚合 + 批量 flush”；consumer 不直接写 Redis，不直接更新 `ranking_post_state`，后续 `FlushRankingBuckets` 再按 pending delta 批量物化到 `ranking_post_state`、`ranking_period_score` 和 Redis。

## Partition Key

Ranking 内部统一 partition key：

```text
ranking-post:<internalPostId>
```

解析规则：

| 事件来源 | 输入字段 | 解析策略 |
| --- | --- | --- |
| Content 事件 | `payload.internalId` | 直接使用 Content 内部 `post_id`。 |
| Comment 事件 | `payload.internalId` | 直接使用 Content 内部 `post_id`。 |

解析失败语义：

- 缺少 `publicId` / `internalId` 等 contract 字段：进入 DLQ。
- `internalId` 非法：进入 DLQ，不按 `publicId` 同步补查。
- HTTP path 入站、repair 和 reconcile 仍可通过 Content contract 解析 `publicId`，但事件摄入主路径不做同步解析。

## 扩容模式：Ranking 私有 shard router

当单队列 + 数据库并发控制不能满足吞吐时，引入 Ranking 私有分片层。该层不改变 provider event contract。

```text
zhicore.events(topic)
  -> ranking.router.events(queue)
  -> Ranking shard router
  -> zhicore.ranking.ingest(private exchange)
  -> ranking.ingest.0..N queues
  -> Ranking shard workers
```

router 职责：

1. 消费 `ranking.router.events`。
2. decode envelope，解析内部 `post_id`。
3. 计算 `shard = hash(internalPostId) % partitionCount`。
4. 发布到私有 exchange / queue。
5. publish confirm 成功后 ack 原始消息。

worker 职责：

1. 每个 shard queue 只由一个 active worker 消费，或用配置保证同一 shard 内单线程处理。
2. 处理成功并提交 PostgreSQL 后 ack 私有消息。
3. duplicate `event_id` 仍由 ledger / inbox no-op。

### 私有 exchange 选择

优先级：

| 方案 | 使用条件 | 说明 |
| --- | --- | --- |
| `x-consistent-hash` exchange | RabbitMQ 插件可用 | router 以 `ranking-post:<internalPostId>` 作为 routing key 发布，RabbitMQ 分配到 shard queue。 |
| direct exchange + 显式 shard key | 插件不可用 | router 计算 shard 后用 routing key `shard.<n>` 发布到固定队列。 |
| 仅本地 keyed worker | 低 QPS 或单实例 | 不提供跨实例同 post 顺序，只减少单进程并发冲突。 |

`partitionCount` 是部署级配置。变更 partitionCount 会改变 post 到 shard 的映射，必须 drain 旧队列后切换；不能在有积压时直接滚动修改。

## 顺序保证边界

| 场景 | 是否依赖局部顺序 | 正确性机制 |
| --- | --- | --- |
| 热度 `LIKE` / `FAVORITE` / `COMMENT` delta | 不依赖 | `event_id` 去重，delta 写 ledger / bucket，flush 使用 pending delta。 |
| view 去重 / cap | 不依赖严格顺序 | Redis dedup / cap 或本机严格 fallback；不可确认时可丢弃 view。 |
| Content 可见性投影 | 不依赖 MQ 顺序 | `aggregateVersion` 优先，`occurredAt` 兜底，旧事件 stale ignored。 |
| bucket flush | 不依赖 MQ 顺序 | `applied_*` 和 `FOR UPDATE` / 条件更新只应用 pending delta。 |
| rebuild | 不依赖 MQ 顺序 | barrier 暂停 live ingestion，从 ledger 按 `occurred_at,event_id` replay。 |

局部顺序优化能减少同一 `post_id` 的行锁冲突和 bucket 重开次数，但不能删除上述幂等和乱序保护。

## Retry / DLQ

两段式 router 模式下有两个 retry 边界：

| 边界 | ack 条件 | 失败策略 |
| --- | --- | --- |
| `zhicore.events -> router` | 私有消息 publish confirm 成功 | transient 失败重试；无法解析 post 时按事件类型 DLQ。 |
| `private shard queue -> worker` | PostgreSQL 事务提交成功 | transient 失败重试；duplicate ack no-op；确定性坏消息 DLQ。 |

DLQ payload 必须保留：

- `eventId`
- `eventType`
- `producer`
- `publicId`
- `commentId`（如有）
- `reason`
- `traceId` / `requestId`（如有）

不得在 DLQ 记录原始 IP、Authorization、正文全文或敏感请求体。

## 配置项

| 配置 | 默认建议 | 说明 |
| --- | --- | --- |
| `ranking.consumer.mode` | `direct` | `direct`、`local_keyed`、`sharded_router`。 |
| `ranking.consumer.prefetch` | `50` | 单 consumer 未 ack 消息上限。 |
| `ranking.consumer.concurrency` | `4` | direct / local keyed 模式下 handler 并发。 |
| `ranking.consumer.local_partitions` | `64` | 本地 keyed worker 分片数。 |
| `ranking.consumer.partition_count` | `16` | 私有 shard queue 数；变更需 drain。 |
| `ranking.consumer.router_publish_timeout` | `500ms` | router 发布私有消息 confirm 超时。 |
| `ranking.consumer.retry.max_attempts` | `5` | 进入 DLQ 前最大尝试次数。 |
| `ranking.consumer.retry.backoff` | `exponential+jitter` | broker retry 或应用层 retry 策略。 |

这些配置最终落到 `services/zhicore-ranking/configs/` 和 runtime wiring；本文只固定语义。

## 测试准入

首期 direct / local keyed 模式必须覆盖：

- 重复 `eventId` 只写一次 ledger / inbox，第二次 ack no-op。
- 两个事件乱序到达后，bucket pending delta 和 state 结果正确。
- 已 flushed bucket 收到迟到事件后，下一轮 flush 只应用新增 pending delta。
- `visibility_changed` 旧 `aggregateVersion` 或较旧 `occurredAt` 不覆盖新投影。
- 事件缺少 `internalId` 不写 ledger / projection，并进入 DLQ。
- Comment `internalId` 按 Content 内部 `post_id` opaque reference 使用，`publicId` 不当作内部 `post_id` 使用。

sharded router 模式必须额外覆盖：

- 同一内部 `post_id` 稳定路由到同一 shard。
- router publish confirm 成功前不 ack 原始消息。
- private shard worker duplicate `eventId` no-op。
- `partitionCount` 配置变更被启动校验阻止，除非明确处于 drain / migration 模式。
