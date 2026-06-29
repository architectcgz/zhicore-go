# Comment 统计和 HOT 排序取舍复盘

本文记录 2026-06-27 对 Comment 统计字段、点赞写入和 HOT 排序读模型的设计讨论。它不是逐字 transcript，而是把“为什么不把 stats 全写进 `comments`”、“为什么 `comments + comment_stats` join 会削弱索引收益”、“为什么引入 `comment_counter_deltas` 和 `comment_hot_rank`”整理成后续可复盘的决策记录。

相关事实源：

- [Comment 服务设计](../README.md)
- [Comment 模块数据和事件设计](../../../module/comment/data-events.md)
- [Comment Application Service 设计](../../../module/comment/service.md)
- [Comment HTTP Schema](../../../../../services/zhicore-comment/api/http/README.md)

> 当前事实源已在 2026-06-29 调整：Comment 不再使用文章内 `floor` 作为定位或排序锚点，改用 PostgreSQL identity 生成的内部 `comments.id` 及其派生的对外 `commentId`。本文保留 2026-06-27 当时的 HOT 读模型取舍背景；涉及 `floor` 的字段和索引以 `docs/architecture/module/comment/comment-id.md` 与 `data-events.md` 为准。

## 结论

Comment 第一阶段采用：

- `comments`：只保存评论内容、树结构、状态、删除元数据和媒体引用。
- `comment_likes`：点赞事实源，用 `(comment_id, user_id)` 唯一约束保证幂等。
- `comment_stats`：评论统计读模型，保存 `like_count`、`reply_count`，可从事实表重建。
- `comment_counter_deltas`：点赞计数 delta 台账，请求事务只写 delta，不同步打统计行。
- `comment_hot_rank`：顶级评论 HOT 排序读模型，保存 `post_id`、`floor`、`like_count`、`visible`，用于 `like_count DESC, floor ASC` 查询。

当前不把 `like_count` 直接放进 `comments`，也不让 HOT 高频查询依赖大范围 `comments + comment_stats` join 排序。

## 问题 1：为什么不把 stats 全放进 `comments`

把 `like_count`、`reply_count` 放进 `comments` 的直接好处是查询简单：

```sql
SELECT *
FROM comments
WHERE post_id = ?
  AND root_id IS NULL
  AND status = 'NORMAL'
ORDER BY like_count DESC, floor ASC
LIMIT 20;
```

这样可以建接近理想的单表索引：

```sql
CREATE INDEX ix_comments_post_hot
  ON comments (post_id, like_count DESC, floor ASC)
  WHERE root_id IS NULL AND status = 'NORMAL';
```

但它把高频点赞写入压到 `comments` 本体行：

```sql
UPDATE comments
SET like_count = like_count + 1
WHERE id = ?;
```

问题是：

- `comments` 是评论内容和树结构的强一致主表，本体行被点赞高频更新会和编辑、删除、状态变更、查询可见性争用同一行版本。
- 热门评论会形成热点行，PostgreSQL 行锁和 MVCC 版本膨胀都会放大写入成本。
- 点赞 QPS 高时，主表频繁更新会增加 WAL、索引更新、autovacuum 压力和缓存抖动。
- `comments` 行越宽，更新代价越高；把高频计数放在宽主表里会扩大写放大。

所以 stats 全放进 `comments` 的主要问题不是读路径，而是把互动写入耦合到评论本体生命周期，降低写路径隔离度。

## 问题 2：为什么 `comment_stats.like_count` join 后索引收益会变弱

这里的“变弱”不是说索引失效，也不是说所有 join 都慢。准确说法是：HOT 查询需要同时满足 `comments` 的过滤条件和 `comment_stats` 的排序字段，但普通 B-tree 索引只能建在单表上，数据库很难用一个索引同时完成跨表过滤、排序和 `LIMIT`。

典型查询：

```sql
SELECT c.*
FROM comments c
JOIN comment_stats s ON s.comment_id = c.id
WHERE c.post_id = ?
  AND c.root_id IS NULL
  AND c.status = 'NORMAL'
ORDER BY s.like_count DESC, c.floor ASC
LIMIT 20;
```

理想索引其实是跨表的：

```text
(comments.post_id, comments.root_id/status, comment_stats.like_count DESC, comments.floor ASC)
```

关系型数据库不能在两张普通表之间建立这种联合 B-tree。实际只能分别建：

```sql
CREATE INDEX ix_comments_post_top
  ON comments (post_id, floor)
  WHERE root_id IS NULL AND status = 'NORMAL';

CREATE INDEX ix_comment_stats_hot
  ON comment_stats (like_count DESC, comment_id);
```

优化器通常只能选择一种驱动方式。

### 方式 A：先过滤 `comments`

如果某篇文章有大量顶级评论：

```text
comments 按 post_id/status 找出候选
-> join comment_stats
-> 对候选按 like_count DESC, floor ASC 排序
-> LIMIT 20
```

这能很好利用 `comments(post_id, status)` 过滤，但 `ORDER BY s.like_count` 来自另一张表，排序值要 join 后才完整。候选很多时仍可能需要中间结果排序，`comment_stats(like_count)` 不能直接让结果天然有序。

### 方式 B：先按 `comment_stats.like_count` 扫

另一种路径是：

```text
comment_stats 按 like_count DESC 扫
-> join comments
-> 过滤 post_id/status/root_id
-> 收集到 20 条后停止
```

这能利用 `like_count` 排序索引，但 `post_id/status/root_id` 在 `comments` 表。数据库可能按全局 like_count 扫到很多评论后，才发现它们不属于当前文章或不可见。此时 `LIMIT 20` 不等于只扫描 20 行。

### `floor` tie-breaker 也跨表

HOT 排序最终是：

```text
like_count DESC, floor ASC
```

`like_count` 在 `comment_stats`，`floor` 在 `comments`。即使 `comment_stats(like_count DESC, comment_id)` 有序，也不能天然保证同点赞数下按 `floor ASC` 有序，仍可能需要 join 后补排序。

### 小结

join 后索引收益变弱，本质是：

- 过滤列在 `comments`；
- 排序主列在 `comment_stats`；
- 排序 tie-breaker 又在 `comments`；
- `LIMIT` 必须在过滤和排序都成立后才可靠。

这会让单表索引很难覆盖整条查询链路，容易退化为“过滤后排序”或“排序后大量回表过滤”。

## 问题 3：为什么用 `comment_hot_rank`

`comment_hot_rank` 把 HOT 查询需要的过滤和排序锚点放进一张窄表：

```sql
CREATE TABLE comment_hot_rank (
  comment_id BIGINT PRIMARY KEY,
  post_id VARCHAR(32) NOT NULL,
  floor BIGINT NOT NULL,
  like_count BIGINT NOT NULL DEFAULT 0,
  visible BOOLEAN NOT NULL DEFAULT TRUE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

查询索引可以直接服务目标排序：

```sql
CREATE INDEX ix_comment_hot_rank_post_hot
  ON comment_hot_rank (post_id, like_count DESC, floor ASC)
  WHERE visible = TRUE;
```

查询流程变成：

```text
comment_hot_rank 用 post_id + like_count DESC + floor ASC 取一页 comment_id
-> comments 按 comment_id 批量补正文、状态、媒体引用
-> comment_stats 批量补展示计数
-> User 批量补作者摘要
```

这样排序发生在窄读模型上，`LIMIT 20` 可以更接近“从目标索引取 20 条”，后续 join 只发生在一页结果上。

## 问题 4：为什么还保留 `comment_stats`

`comment_hot_rank` 只解决顶级评论 HOT 排序，不替代完整统计读模型。

`comment_stats` 仍然需要承担：

- 评论详情展示 `likeCount`、`replyCount`。
- TIME 列表展示统计。
- 回复列表展示统计。
- 管理端和修复任务读取统计。
- 从 `comment_likes` / `comments` 重建统计后的落点。

所以当前结构是：

- `comment_stats`：通用统计读模型。
- `comment_hot_rank`：顶级评论 HOT 排序专用读模型。

## 问题 5：为什么点赞请求不直接同步更新 `comment_stats`

同步更新 `comment_stats.like_count` 比更新 `comments.like_count` 好一些，因为它把热点从宽主表移到了窄统计表。但热门评论仍然会形成单行热点：

```sql
UPDATE comment_stats
SET like_count = like_count + 1
WHERE comment_id = ?;
```

如果点赞 QPS 高，热门评论的统计行仍会被频繁行锁竞争。为降低请求写路径压力，当前设计是：

```text
请求事务：
  comment_likes 插入/删除
  + comment_counter_deltas 追加 LIKE +/-1
  + outbox_events(comment.liked/comment.unliked)

后台 worker：
  claim delta
  -> 按 comment_id 聚合
  -> 批量更新 comment_stats.like_count
  -> 顶级评论同步更新 comment_hot_rank.like_count
```

这带来的语义选择是：

- `viewer.liked` 强一致，以 `comment_likes` 为准。
- `likeCount` 最终一致，允许短暂延迟。
- HOT 排序最终一致，允许短暂延迟。
- 统计漂移可以从 `comment_likes` 重建。

## 被明确放弃的方案

| 方案 | 放弃原因 |
| --- | --- |
| stats 全放进 `comments` | 查询简单，但点赞高 QPS 会频繁更新评论主表，主表行锁、MVCC 版本、WAL 和索引写放大都更重。 |
| 只用 `comment_stats` join 排 HOT | 数据归属清楚，但 HOT 查询的过滤、排序、tie-breaker 分散在两张表，难以用一个索引覆盖，`LIMIT` 收益变弱。 |
| 同步更新 `comment_stats.like_count` | 比写 `comments` 好，但热门评论仍会形成统计行热点；请求延迟和锁竞争受点赞峰值影响。 |
| 直接用 Redis 计数作为真相源 | 写入快，但可靠性、重建、事务一致性和 outbox 事件关联都更弱；Redis 只能做缓存或派生读模型。 |

## 当前复盘问题

后续实现或压测时重点验证：

- `comment_hot_rank(post_id, like_count DESC, floor ASC)` 是否覆盖 HOT 查询主路径。
- delta worker 批量大小、claim 策略和失败重试是否会造成统计延迟过大。
- 热门评论点赞峰值下，`comment_likes` 唯一约束插入/删除是否成为新瓶颈。
- `likeCount` 最终一致延迟是否符合前端体验。
- 如果回复创建 QPS 也升高，是否需要把 `reply_count` 也迁移到 delta 机制。
