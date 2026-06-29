# Ranking 数据事件与投影设计

本文记录 `zhicore-ranking` 的跨服务事件输入、热度 ledger 和 Content 可见性 / 元数据投影。Ranking README 只保留入口和关键结论；事件字段、投影事务和 schema 细节以本文为准。

相关事实源：

- [Ranking 服务设计](README.md)
- [Ranking 设计决策日志](decision-log/2026-06-29-ranking-design-decisions.md)
- [Ranking 运行期 resilience](runtime-resilience.md)
- [Content 数据、事件和契约设计](../content/data-events-contracts.md)
- [事件契约规范](../../../contracts/events.md)

## 核心结论

- 热度指标事件写 `ranking_event_ledger` 和 `ranking_delta_bucket`，用于重放和分数计算。
- 可见性 / 元数据事件不写热度 ledger；它们写 `ranking_projection_event_inbox` 或等价幂等表，并更新 `ranking_post_state` 的本地 projection。
- 公开榜单查询、snapshot 和候选集必须过滤 `ranking_post_state.public_visible = TRUE`。
- Content 回源只用于详情补齐、事件缺字段解析、repair / reconcile 兜底，不作为公开榜单查询主过滤路径。
- Redis 榜单、候选集和移除操作都是可重建投影；Redis 失败不回滚 PostgreSQL projection。

## 本地可见性投影

`ranking_post_state.public_visible` 是 Ranking 对 Content 生命周期事件的本地投影，不是 Content 源事实。Content 的发布、删除、撤回、恢复、管理端下架 / 隐藏 / 重新公开等事件驱动该投影。

推荐字段：

| 字段 | 含义 |
| --- | --- |
| `public_visible` | 是否允许进入公开榜单；默认 `FALSE`。 |
| `content_status` | Content 生命周期状态快照，例如 `PUBLISHED`、`DRAFT`、`DELETED`、`HIDDEN`、`TAKEN_DOWN`。 |
| `visibility_reason` | 不可公开或重新公开原因，来自 Content 事件。 |
| `visibility_updated_at` | 最新可见性事实时间。 |

`public_visible` 默认 `FALSE`，避免 Ranking 在尚未收到 Content 发布 / 恢复 / 可见性事件前把未知文章放入公开榜单。文章恢复公开后，历史热度计数不删除，可按现有分数重新进入公开榜单。

## 消费事件

Ranking 消费的跨服务事件：

| 事件 | 来源 | 指标 | Delta | 关键字段 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `content.post.viewed` | Content | `VIEW` | `+1` | `eventId`、`publicPostId`、`postId?`、`viewerId?`、`ipHash?`、`occurredAt`、`publishedAt` | 先做浏览去重和上限控制 |
| `content.post.published` | Content | 可见性 / 元数据 | - | `eventId`、`publicPostId`、`postId?`、`authorId`、`publishedAt`、`topicIds?`、`occurredAt` | 标记文章可进入公开榜单，刷新 metadata 快照 |
| `content.post.updated` | Content | 元数据 | - | `eventId`、`publicPostId`、`postId?`、`authorId?`、`publishedAt?`、`topicIds?`、`occurredAt` | 刷新 author / publishedAt / topicIds 等 Ranking 所需 metadata |
| `content.post.deleted` | Content | 可见性 | - | `eventId`、`publicPostId`、`postId?`、`deletedAt`、`occurredAt` | 标记文章不可进入公开榜单，不删除热度 ledger |
| `content.post.visibility_changed` | Content | 可见性 | - | `eventId`、`publicPostId`、`postId?`、`oldVisibility`、`newVisibility`、`publicVisible`、`reason`、`occurredAt`、`aggregateVersion?` | 覆盖撤回、下架、隐藏、恢复、重新公开等状态变化 |
| `content.post.tags.updated` | Content | 元数据 | - | `eventId`、`publicPostId`、`postId?`、`topicIds`、`occurredAt` | 刷新 topic 派生榜所需 topicIds |
| `content.post.liked` | Content | `LIKE` | `+1` | `eventId`、`publicPostId`、`postId?`、`authorId`、`likedBy`、`occurredAt` | 上游保证点赞幂等 |
| `content.post.unliked` | Content | `LIKE` | `-1` | `eventId`、`publicPostId`、`postId?`、`authorId`、`unlikedBy`、`occurredAt` | 负增量由源服务显式发出 |
| `content.post.favorited` | Content | `FAVORITE` | `+1` | `eventId`、`publicPostId`、`postId?`、`authorId`、`favoritedBy`、`occurredAt` | 收藏增量 |
| `content.post.unfavorited` | Content | `FAVORITE` | `-1` | `eventId`、`publicPostId`、`postId?`、`authorId`、`unfavoritedBy`、`occurredAt` | 负增量 |
| `comment.created` | Comment | `COMMENT` | `+1` | `eventId`、`publicPostId`、`postId?`、`commentId`、`authorId`、`createdAt` | 评论增量 |
| `comment.deleted` | Comment | `COMMENT` | `-affectedCount` | `eventId`、`publicPostId`、`postId?`、`affectedCount`、`deletedAt` | 根评论删除时按 affectedCount 回滚 |
| `comment.liked` | Comment | 第一阶段不消费 | - | `eventId`、`commentId`、`publicPostId`、`postId?`、`likedBy` | 如产品要求计入文章热度，必须先扩展指标、权重、bucket 列和 replay 规则 |
| `comment.unliked` | Comment | 第一阶段不消费 | - | `eventId`、`commentId`、`publicPostId`、`postId?`、`unlikedBy` | 同上 |

Go 目标默认先实现 Content 互动、Content 可见性控制事件和评论创建 / 删除。评论点赞是否计入文章热度必须在事件 contract 阶段明确，不能在 Ranking 内自行推断。

表中的 `publicPostId` 是稳定契约字段；`postId?` 是生产方可选携带的 Content 内部 `post_id`，仅用于减少 Ranking 解析调用。HTTP API 的 `{postId}` path 值使用 Content `public_id`。

## 热度事件摄入

热度指标事件 payload 必须表达事实和 delta，不能传“当前总数快照”作为 Ranking 的主输入。发生时间使用 UTC RFC3339；落库字段使用 `occurred_at`。

摄入流程摘要：

```text
事务前：
  decode RabbitMQ JSON
  + 校验 eventId、eventType、publicPostId、occurredAt、delta
  + 解析内部 postId；transient error 时 nack/requeue，确定性 not found / deleted 进入 DLQ
  + view dedup / view cap 过滤；被拦截事件 ack 但不写 ledger

单个 PostgreSQL 事务：
  INSERT ranking_event_ledger(event_id, ...)
  + UPSERT ranking_delta_bucket(bucket_start, post_id, delta...)

事务提交后：
  ack RabbitMQ
```

消费侧不直接写 Redis、不直接更新 `ranking_post_state`。`event_id` 是消费幂等键；进入 Ranking 内部后的 `post_id` 是局部顺序和 bucket 聚合键。

## 可见性 / 元数据投影事务

```text
事务前：
  decode RabbitMQ JSON
  + 校验 eventId、eventType、publicPostId、occurredAt
  + 解析内部 postId；transient error 时 nack/requeue，确定性 not found 进入 DLQ 或 no-op 告警

单个 PostgreSQL 事务：
  INSERT ranking_projection_event_inbox(event_id, ...)
    - 插入成功：继续
    - event_id 冲突：duplicate no-op
  + UPSERT ranking_post_state metadata / visibility 字段
    - published / restored / visible：public_visible=true，并刷新 author_id、published_at、topic_ids
    - deleted / unpublished / hidden / taken_down：public_visible=false，记录 content_status、visibility_reason、visibility_updated_at

事务提交后：
  + 如果 public_visible=false，best-effort 从 Redis 榜单和候选集中移除该 post
  + ack RabbitMQ
```

可见性投影更新不写 `ranking_event_ledger`、不更新 `ranking_delta_bucket`、不改变历史热度计数。Redis 移除失败只记录 `redis_visibility_remove_failed` 和 degraded metric，不回滚 PostgreSQL；下一轮 snapshot、candidate refresh 或 visibility reconcile 必须从 `ranking_post_state.public_visible` 收敛。

乱序处理应优先使用事件 envelope 的 `aggregateVersion`。若 provider 暂未提供版本，consumer 至少使用 `occurredAt` 和当前 `visibility_updated_at` 防止较旧事件覆盖较新投影；最终 contract 仍应补齐 `aggregateVersion` 或等价字段。

## 查询与 Redis 收敛

总榜从当前 `ranking_post_state WHERE public_visible = TRUE` 构建。周期榜从 `ranking_period_score` 构建后必须 join / 批量加载 `ranking_post_state` 过滤 `public_visible = TRUE`，并补齐 author/topic 派生榜。

Redis 处理规则：

| 操作 | Redis 处理 |
| --- | --- |
| 事件摄入 | 不更新榜单 Redis，只写 ledger/bucket |
| bucket flush 成功 | 事务提交后增量更新总榜和周期榜；失败等待 snapshot |
| 可见性变为不可公开 | PostgreSQL 投影提交后 best-effort 从总榜、周期榜和候选集中移除；失败等待 snapshot / reconcile |
| 可见性恢复公开 | PostgreSQL 投影提交后可触发单篇重算 / snapshot；不得凭 Redis 旧成员判断可见 |
| snapshot refresh | 原子替换总榜、日/周/月榜 |
| rebuild-from-ledger | 重建完成后 refresh active snapshots 和 hot candidates |
| hot candidate refresh | 使用临时 key + rename 原子替换候选集和 meta |

Redis miss 回源总榜时，拿到 backfill lock 的请求必须回源：

```sql
SELECT *
FROM ranking_post_state
WHERE public_visible = TRUE
ORDER BY hot_score DESC
LIMIT :limit OFFSET :offset;
```

## Schema 草案

`ranking_post_state` 需要保存 Ranking 本地可见性投影字段：

```sql
ALTER TABLE ranking_post_state
  ADD COLUMN public_visible BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN content_status VARCHAR(32) NULL,
  ADD COLUMN visibility_reason VARCHAR(64) NULL,
  ADD COLUMN visibility_updated_at TIMESTAMPTZ NULL;

CREATE INDEX idx_ranking_post_state_hot_score
  ON ranking_post_state(public_visible, hot_score DESC);
```

正式 migration 应直接在 `CREATE TABLE ranking_post_state` 中包含这些字段；上面的 `ALTER TABLE` 只用于表达差异。

可见性 / 元数据 projection inbox 草案：

```sql
CREATE TABLE ranking_projection_event_inbox (
  event_id VARCHAR(128) PRIMARY KEY,
  event_type VARCHAR(128) NOT NULL,
  post_id BIGINT NOT NULL,
  public_post_id VARCHAR(32) NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  processed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ranking_projection_inbox_post
  ON ranking_projection_event_inbox(post_id, occurred_at);
```

正式 migration 可按实现命名调整，但必须保留：

- `event_id` 唯一幂等。
- 按 `post_id + occurred_at` 查询投影处理历史。
- 可区分热度 ledger 和非热度 projection 事件。

## 首个实现切片要求

切片 1 至少覆盖：

- `content.post.liked`
- `content.post.unliked`
- `content.post.published`
- `content.post.deleted`
- `content.post.visibility_changed`
- `comment.created`
- `comment.deleted`

`view` / `favorite` 可在同一切片后半补，但公开榜单过滤不能后置到查询时逐条回源 Content。切片 1 的测试至少覆盖：

- 可见性事件在 PostgreSQL 失败时不 ack。
- duplicate projection `event_id` ack no-op。
- `public_visible=false` 后 Redis 移除失败不回滚 PostgreSQL。
- snapshot / reconcile 后榜单不包含不可公开文章。
