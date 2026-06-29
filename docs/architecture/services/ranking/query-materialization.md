# Ranking 查询、缓存与物化设计

本文记录 `zhicore-ranking` 的查询路径、Redis key、Redis miss 回源、snapshot、候选集和归档策略。运行期故障语义见 [runtime-resilience.md](runtime-resilience.md)。

## 查询原则

PostgreSQL 是 `ledger/bucket/state/period` 的权威源。Redis 只保存可重建榜单和运行期控制状态。

公开榜单查询、snapshot 和候选集必须过滤 `ranking_post_state.public_visible = TRUE`。该投影由 Content 可见性事件维护，细节见 [data-events-projections.md](data-events-projections.md)。

文章详情补齐由 application 批量调用 Content contract；不能在 Redis store 里同步调用 Content，避免查询层隐藏 N+1。

## Redis key

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

## 缓存更新

| 操作 | Redis 处理 |
| --- | --- |
| 事件摄入 | 不更新榜单 Redis，只写 ledger/bucket |
| bucket flush 成功 | 事务提交后增量更新总榜和周期榜；失败等待 snapshot |
| 可见性变化 | 见 [data-events-projections.md](data-events-projections.md)，以 PostgreSQL projection 为过滤权威 |
| snapshot refresh | 原子替换总榜、日 / 周 / 月榜 |
| rebuild-from-ledger | 重建完成后 refresh active snapshots 和 hot candidates |
| hot candidate refresh | 使用临时 key + rename 原子替换候选集和 meta |
| archive | 不改变当前榜单；必要时回填月榜缓存 |

## Redis miss 回源

总榜查询流程：

1. 先读 `ZREVRANGE ranking:posts:hot ... WITHSCORES`。
2. 如果 key 不存在或返回空，尝试获取 `ranking:backfill:posts:hot`，TTL 默认 60 秒。
3. 拿到锁的请求回源 `ranking_post_state WHERE public_visible = TRUE ORDER BY hot_score DESC`，写临时 key 后 `RENAME` 原子替换正式 key。
4. 没拿到锁的请求可以短暂等待或直接回源 PostgreSQL 返回，不重复回填。
5. 回源仍为空时写 `ranking:empty:posts:hot`，TTL 默认 60 秒，避免短时间内击穿。

周期榜查询流程：

1. 先读对应 Redis key，例如 `ranking:posts:daily:2026-06-23`。
2. miss 时，如果 period 在活跃窗口内，回源 `ranking_period_score WHERE period_type=? AND period_key=? ORDER BY delta_score DESC`，并按 `ranking_post_state.public_visible = TRUE` 过滤后回填 Redis。
3. 如果 period 超出活跃窗口，查 MongoDB archive；冷数据默认不回填长期 Redis key，只可写短 TTL 查询缓存。
4. 回源为空时写 `ranking:empty:{period_type}:{period_key}`，TTL 默认 60 秒。

## Snapshot refresh

```text
读取 ranking_post_state + ranking_period_score
构建 post / creator / topic scores
写临时 Redis ZSET
原子替换正式 key
```

总榜从当前 `ranking_post_state WHERE public_visible = TRUE` 构建。周期榜从 `ranking_period_score` 构建后必须 join / 批量加载 `ranking_post_state` 过滤 `public_visible = TRUE`，并补齐 author / topic 派生榜。

刷新失败保留旧 key。活跃窗口参考 Java 设计：日榜最近 2 天、周榜最近 20 天覆盖到的 ISO 周、月榜最近 365 天覆盖到的月份；具体值配置化。

## 候选集 stale 语义

候选集 meta 至少包含：

- `version`
- `generated_at`
- `source_key`
- `source_count`
- `candidate_size`
- `min_score`
- `last_refresh_attempt`
- `consecutive_failures`

`stale` 是查询时派生状态，不单独持久化：

- `now - generated_at < stale_threshold`：fresh。
- `stale_threshold <= now - generated_at < 2 * stale_threshold`：stale，仍返回旧数据，并异步触发 refresh。
- `generated_at` 缺失或 `now - generated_at >= 2 * stale_threshold`：视为过期，Comment 应降级为空候选集或本地旧缓存。

刷新失败时保留上一版候选集，不删除正式 Redis key；连续失败 3 次后发送告警，但不阻塞查询。

## 归档和冷数据

归档任务从 Redis 或 PostgreSQL source store 读取日 / 周 / 月榜，写 MongoDB。归档不是实时查询权威源；归档失败不影响在线榜单，但必须可重试。

月榜冷数据查询可以先查 Redis，缺失时用 lock 回源 MongoDB 并回填 Redis；超出 Redis 保留范围时直接查 MongoDB。

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

字段级 request / response、错误码、权限和返回空榜语义需要后续按 `docs/contracts/http-schema-template.md` 提取到 `services/zhicore-ranking/api/http`。
