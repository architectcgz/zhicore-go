# Content Engagement 设计

本文记录 `zhicore-content` 的点赞、收藏、互动统计和当前用户互动状态设计。`engagement` 在 Content 语境中表示文章互动信息：文章级统计、当前登录用户视角状态，以及点赞 / 收藏命令产生的事件和缓存。

当前状态：本文固定产品语义、后端事实源、缓存降级、HTTP contract 约束和实现准入条件，不表示 Go 代码已经实现。

## 目标和边界

Engagement 覆盖：

- 点赞 / 取消点赞。
- 收藏 / 取消收藏。
- 文章统计：浏览数、点赞数、收藏数、评论数。
- 当前登录用户视角：是否已点赞、是否已收藏。
- 批量查询当前用户对多篇文章的互动状态。

Engagement 不覆盖：

- 评论点赞。评论点赞归 `zhicore-comment`。
- 热榜分数。热榜由 Ranking 消费 Content 事件后计算。
- 通知收件箱。Notification 只消费点赞事件并维护自己的投递事实。

## 产品语义

### 状态三值

当前用户视角的点赞 / 收藏状态必须区分三种语义：

| 状态 | JSON 表达 | 含义 | 前端行为 |
| --- | --- | --- | --- |
| 已互动 | `true` | Content 确认当前用户已点赞或已收藏。 | 按已选中状态展示按钮。 |
| 未互动 | `false` | Content 确认当前用户未点赞或未收藏。 | 按未选中状态展示按钮。 |
| 未知 | `null` + `viewer.degraded=true` | Redis 不可用且受控 DB 回源未能确认，或 fallback budget 已耗尽。 | 文章继续展示；按钮展示中性态或轻量禁用态，不把 `unknown` 当成 `false`。 |

`unknown` 不是领域事实，只是查询降级状态。它不能写入数据库、事件或 Redis 缓存，也不能作为业务判断依据。

### 详情页体验

文章详情页按两阶段加载：先加载文章主体，再加载 engagement。文章主体包括公开可见性、标题、作者、正文或正文摘要等内容展示所需信息；engagement 是附加信息，不作为文章可读性的前置条件。

- 文章主体不可用、不可见或不存在时，前端不请求 engagement，也不展示点赞、收藏、统计和当前用户状态。
- 文章主体可用后再请求 engagement；engagement 加载中时，按钮可显示骨架、中性态或轻量 loading。
- 文章主体可用，engagement 不可用时，文章继续展示；前端只降级互动区域。
- 文章主体和统计可用，viewer 状态不可确认：返回 `viewer.liked=null`、`viewer.favorited=null`、`viewer.degraded=true`。
- 前端不弹全局错误，不把按钮展示成“未点赞 / 未收藏”。
- 用户点击点赞 / 收藏按钮时，如果后端无法确认限流或写事务，命令接口返回 `1004 SERVICE_DEGRADED`，前端提示“互动服务暂不可用，请稍后再试”，不做乐观成功。

如果未来某个客户端无法处理 `null` viewer 状态，该客户端必须走兼容适配或独立 contract 演进，不能要求后端把未知伪装成 `false`。

### 列表页和批量状态

列表页允许正文和统计先展示，viewer 状态晚到或缺失。批量状态接口降级时：

- 对能确认的文章返回 `true` / `false`。
- 对不能确认的文章返回 `liked=null` / `favorited=null`，并在 item 上标记 `degraded=true`。
- 不能因为一个文章状态查询失败而让整个列表不可读。

批量查询请求本身参数非法、未登录或服务整体不可用时仍按 HTTP contract 返回错误。

## 事实源和一致性

PostgreSQL 是 engagement 事实源：

- `post_likes` 记录点赞关系。
- `post_favorites` 记录收藏关系。
- `post_stats` 记录文章统计投影；点赞 / 收藏计数由 Content 内部 delta task worker 最终一致更新。
- `outbox_event` 记录跨服务事件。
- `domain_event_tasks` 记录 Content 内部 engagement stats delta task。

Redis 只保存可丢弃缓存：

- 当前用户对文章的点赞 / 收藏状态缓存。
- 文章点赞数 / 收藏数等热点计数缓存。
- 批量状态查询的辅助缓存。

点赞 / 收藏命令必须在单个 PostgreSQL 事务内完成关系表、跨服务 outbox 记录和 Content 内部 stats delta task。`post_stats.like_count` / `favorite_count` 由 Content 内部 `content-engagement-stats` worker 消费 delta task 后投影更新，不通过 RabbitMQ 自消费，也不在命令事务里锁 `posts` 或递增 `posts.post_version`。事务提交后 best-effort 更新 Redis；Redis 失败不回滚业务事务。

## 命令流程

### 点赞

`LikePost`：

1. 校验登录态、文章存在且可互动。
2. 执行业务限流：actor + post + operation，以及 actor 全局互动写频控。
3. 在 PostgreSQL 事务内插入 `(post_id, user_id)` 点赞关系。
4. 如果关系已存在，幂等成功，不重复写 stats delta，不发布重复事件。
5. 如果关系首次创建，写 `content.post.liked` outbox 事件和内部 `LIKE +1` stats delta task。
6. 事务提交后 best-effort 更新 Redis 状态和计数缓存。

### 取消点赞

`UnlikePost`：

1. 校验登录态、文章存在且可互动。
2. 执行业务限流。
3. 在 PostgreSQL 事务内删除 `(post_id, user_id)` 点赞关系。
4. 如果关系不存在，幂等成功，不重复写 stats delta，不发布事件。
5. 如果关系被删除，写 `content.post.unliked` outbox 事件和内部 `LIKE -1` stats delta task。
6. 事务提交后 best-effort 更新 Redis 状态和计数缓存。

收藏 / 取消收藏与点赞 / 取消点赞同构，只替换关系表、stats delta metric 和事件类型。

## 查询流程

### 单篇 engagement

`GetPostEngagement(postID, viewer)`：

1. 从 `post_stats` 或统计缓存读取文章统计。
2. 匿名请求不返回 `viewer`。
3. 登录请求优先从 Redis 批量读取当前用户对该文章的点赞 / 收藏状态。
4. Redis miss 时允许按 cache-aside 从 PostgreSQL 读取，并回填 Redis。
5. Redis 不可用或熔断打开时，进入受控 DB fallback。
6. DB fallback 成功时返回确定状态；失败或预算耗尽时返回 `viewer.degraded=true` 和 `null` 状态，文章统计仍可返回。

统计不可用和 viewer 状态不可用是两类错误。统计属于文章展示核心信息，统计读取失败可返回 `1004`；viewer 状态不可用优先降级为 unknown。

`GetPostEngagement` 默认假设调用方已经通过文章详情接口确认文章存在且对当前请求可见。后端仍必须校验 `postId` 对应文章可互动；文章不存在、已删除或不可见时返回 `4001`，不能返回空统计或 unknown viewer 伪装成可互动资源。

### 批量状态

`BatchGetEngagementStatus(userID, postIDs)`：

1. `postIds` 必须先做数量上限校验，当前上限沿用 HTTP contract 的 100。
2. 请求中重复的 `postId` 按首次出现位置去重；响应按去重后的请求顺序返回，便于前端按 `postId` 建 map。
3. Redis 正常时使用批量读取，禁止对每个 `postId` 单独发起网络往返。
4. Redis miss 的子集可以批量回源 PostgreSQL。
5. Redis 不可用时只允许在 fallback budget 内执行一次批量 SQL，禁止循环逐条 `EXISTS(user_id, post_id)`。
6. DB fallback 成功时返回每个 item 的确定状态。
7. DB fallback 部分失败或预算耗尽时，未确认 item 返回 `null` 状态和 `degraded=true`。

推荐 SQL 形态是按 user 维度一次取回命中的 post 集合，例如：

```sql
SELECT post_id
FROM post_likes
WHERE user_id = $1
  AND post_id = ANY($2);
```

收藏关系使用同样形态查询 `post_favorites`。实现可以按数据库方言调整，但必须保持批量语义。

## Redis 故障降级

Redis 故障不能直接转化成无界 DB 回源。Engagement 查询的降级必须同时满足：

- 本机 fallback limiter 允许。
- `postgres.engagement.query` breaker 未打开。
- DB fallback max-in-flight 未耗尽。
- 请求 context deadline 未过期。
- 批量请求没有超过 `postIds` 上限。

满足条件时可以短时 DB 回源；不满足时返回 viewer unknown 或 `1004`：

| 场景 | 降级结果 |
| --- | --- |
| 单篇详情 viewer 状态不可确认 | 返回 `viewer.degraded=true`，`liked/favorited=null`。 |
| 批量状态部分 item 不可确认 | item 返回 `degraded=true`，状态为 `null`。 |
| 批量状态服务整体不可用，例如 DB breaker open | 返回 `1004`，避免前端误认为所有 item 都未互动。 |
| 点赞 / 收藏命令限流依赖持续不可用 | 返回 `1004`，不执行写事务。 |
| PostgreSQL 事务失败 | 返回失败，不写缓存伪装成功。 |

禁止行为：

- Redis 不可用时无条件逐条查 DB。
- 把查询失败解释为 `liked=false` 或 `favorited=false`。
- 只写 Redis 不写 PostgreSQL 事实表。
- 因为接口幂等就绕过限流反复刷写关系表、stats delta 和 outbox。

## Cache key 和失效原则

具体 key 格式由 Redis adapter 持有，设计约束如下：

- key 只能包含规范化的 `postId`、内部 `post_id`、`user_id` hash 或低基数字段，不包含 token、cookie、原始 URL、标题、摘要或正文。
- 状态缓存 TTL 必须短于产品可接受的状态陈旧窗口，并支持命令提交后的主动更新 / 删除。
- 计数缓存是展示优化，不能作为写事务的计数事实源。
- Redis 写失败只记录 degraded metric，不回滚 PostgreSQL 事务。

## Ports 和配置

Engagement 实现至少需要：

- `PostEngagementRepository`：写入 / 删除点赞收藏关系，批量读取当前用户状态。
- `EngagementStatsTaskStore`：追加、claim、应用和失败标记 Content 内部 stats delta task。
- `PostStatsRepository`：读取统计；由内部 worker 原子应用 delta。
- `EngagementCacheStore`：批量读取状态、回填状态、更新计数缓存。
- `RateLimiter`：互动写和互动读 fallback 决策。
- `TransactionRunner`：包裹关系表、outbox 和内部 stats delta task 事务。
- `OutboxPublisher`：在事务内追加 `content.post.liked` / `content.post.unliked` / favorite 事件。

配置必须覆盖：

- Redis engagement read / write timeout。
- Redis engagement cache breaker 和 max-in-flight。
- DB engagement query timeout。
- DB engagement fallback max-in-flight。
- 本机 fallback limiter 容量、窗口和 Redis 故障 fallback 时长。
- batch `postIds` 上限。
- `content-engagement-stats` worker 开关、batch size、claim lease、retry backoff 和 dead-letter 阈值。

## HTTP contract 要求

`GET /api/v1/posts/{postId}/engagement`：

- 调用方应先加载文章详情；文章详情不可用时不应继续请求 engagement。
- 匿名请求返回 `stats`，可不返回 `viewer`。
- 登录请求返回 `viewer`。
- `viewer.liked` 和 `viewer.favorited` 类型为 `boolean | null`。
- `viewer.degraded` 必填；正常为 `false`，状态不可确认时为 `true`。

`POST /api/v1/posts/engagement/batch-status`：

- 每个 item 返回 `postId`、`liked`、`favorited` 和 `degraded`。
- `liked` / `favorited` 类型为 `boolean | null`。
- `degraded=true` 表示当前 item 的状态不可确认，前端不能当作未互动。

命令接口：

- `PUT /like`、`DELETE /like`、`PUT /favorite`、`DELETE /favorite` 返回确定的当前用户状态和当前统计快照；统计计数不承诺包含本次写入后的强一致最新值。
- 命令失败不能返回 `null` 状态伪装成功。

## 测试准入

首次实现 Engagement 前至少覆盖：

- 重复点赞幂等成功，不重复写 stats delta，不重复写 outbox。
- 重复取消点赞幂等成功，不重复写 stats delta，不重复写 outbox；worker 应用 delta 时不把 `like_count` 减成负数。
- Redis 写缓存失败不回滚 PostgreSQL 事务。
- Redis 读不可用时，单篇 viewer 状态在 fallback budget 内从 DB 确认。
- Redis 读不可用且 fallback budget 耗尽时，单篇 viewer 返回 `null` + `degraded=true`，不返回 `false`。
- 批量状态查询使用批量 repository 方法，不逐条 `EXISTS`。
- 批量状态部分不可确认时，只把对应 item 标记 degraded。
- 点赞 / 收藏命令在限流依赖持续不可用时返回 `1004`，不执行写事务。
- HTTP contract test 覆盖 `viewer.liked=null`、`viewer.favorited=null` 和 `viewer.degraded=true`。
