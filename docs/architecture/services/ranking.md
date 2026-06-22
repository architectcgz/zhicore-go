# Ranking 服务设计

## 事实来源

- Java `zhicore-ranking` controller：`RankingController`。
- `zhicore-ranking-detailed-design.md`
- Java Ranking consumer 和 ledger/bucket 设计。

## 职责边界

`zhicore-ranking` 拥有热度事实账本、窗口聚合、文章/作者/话题榜单状态、Redis ZSET 物化和历史榜单归档。

Ranking 不拥有文章、评论、用户或话题源事实。

## API 保留范围

必须保留以下 API 族：

- `/api/v1/ranking/posts/hot`、`details`、`scores`、`candidates`
- `/api/v1/ranking/posts/daily`、`weekly`、`monthly` 及对应 `scores`
- `/api/v1/ranking/posts/{postId}/rank`、`score`
- `/api/v1/ranking/creators/hot`、`scores`、`{userId}/rank`、`{userId}/score`
- `/api/v1/ranking/topics/hot`、`scores`、`{topicId}/rank`、`{topicId}/score`
- `/api/v1/ranking/admin/rebuild-from-ledger`

## 数据归属

Ranking 拥有：

- `ranking_event_ledger`
- `ranking_delta_bucket`
- `ranking_post_state`
- `ranking_period_score`
- Redis ZSET 当前榜单。
- MongoDB 历史榜单归档。

Go migration 需要把这些表从 Java 设计转成服务自己的 migration。

## 主链路

目标主链路：

```text
RabbitMQ consumer
-> ranking_event_ledger
-> ranking_delta_bucket
-> flush worker
-> ranking_post_state / ranking_period_score
-> Redis ZSET
-> query API
```

`ledger` 记录不可变事实，`bucket` 做短时间窗净增量，`post_state` 是 PostgreSQL 权威状态，Redis 只负责高频查询。

## 事件

Ranking 消费：

- `content.post.viewed`
- `content.post.liked`
- `content.post.unliked`
- `content.post.favorited`
- `content.post.unfavorited`
- `comment.created`
- `comment.deleted`

事件 payload 必须表达事实和 delta，不传“当前总数快照”。

## 元数据补齐

Ranking 需要 `author_id`、`published_at`、`topic_ids` 等元数据：

- 优先从事件携带。
- 缺失时通过 Content contract 批量补齐。
- 补齐失败不阻塞 ledger/bucket 落账，后续补偿回填。

## Go 目标落点

- HTTP：`services/zhicore-ranking/api/http`
- Application：`services/zhicore-ranking/internal/ranking/application`
- Domain：`services/zhicore-ranking/internal/ranking/domain`
- Ports：`services/zhicore-ranking/internal/ranking/ports`
- Infrastructure：`postgres`、`redis`、`mongo`、`rabbitmq`、`clients`
- Runtime：`services/zhicore-ranking/internal/ranking/runtime/module.go`

## 迁移风险

- `bucket` flush 必须只应用 pending delta，不能重复物化整桶累计值。
- RabbitMQ 没有 RocketMQ hashKey 的同等语义时，需要用 routing key、consistent hash exchange 或 consumer 本地分片保证同 post 局部顺序。
- 榜单查询依赖 Redis，但 Redis 不是权威源，必须有 PostgreSQL 重建和回填路径。

## 下一步

- 把 Ranking ledger/bucket/state migration 草案落到服务。
- 设计 RabbitMQ 分片和 consumer 幂等策略。
- 补 ledger replay、bucket flush、Redis 回填、rebuild-from-ledger 行为测试。
