# Ranking API 详细设计

状态：草案。本文是 `zhicore-ranking` HTTP API 的字段级 contract，尚未由 Go handler / contract test 验证。

## 总体方案

Ranking HTTP API 分为五组：

- 文章榜：总榜、日榜、周榜、月榜、分数、排名和详情榜。
- 热门候选集：供 Comment 等下游服务读取热门文章候选。
- 创作者榜：按文章热度派生创作者榜、分数和排名。
- 话题榜：按文章热度派生话题榜、分数和排名。
- 管理运维：管理员触发 `rebuild-from-ledger` 并查询 rebuild 操作状态。

核心边界：

- Ranking 对外 `postId` 是 Content `public_id` 字符串，不暴露内部 `post_id BIGINT`。
- Ranking 对外 `entityId` 对文章榜使用 Content `public_id`，对创作者榜使用 User `userId` 字符串，对话题榜使用 Content/Topic 拥有的 `topicId` 字符串。
- Ranking 只返回公开可见文章对应的榜单项。公开可见性来自本地 `public_visible` projection，不在查询主路径逐条回源 Content。
- `score` 是 Ranking 解释的浮点排序值，只用于展示和排序；consumer 不能用它推导源服务计数。
- 列表分页保留 Ranking 目标 0-based page 语义：`page` 默认 `0`，`size` 默认 `20`，最大 `100`。

## 通用对象

### `RankingPage<T>`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `items` | array | 是 | 当前页项目。空榜返回 `[]`。 |
| `page` | int | 是 | 当前页，从 `0` 开始。 |
| `size` | int | 是 | 本次页大小。 |
| `hasMore` | boolean | 是 | 是否还有后续页。 |
| `generatedAt` | string | 是 | 榜单数据生成或读取时间，RFC3339。 |
| `source` | string | 是 | `REDIS`、`POSTGRES`、`MONGO_ARCHIVE`。 |
| `degraded` | boolean | 是 | 是否发生降级，例如 Redis miss 后回源。 |

`total` 不作为默认返回字段，避免 Redis ZSET 和活跃窗口回源路径为了总数增加额外成本。后续管理端若需要总数，必须在对应 endpoint 单独登记。

### `RankingScoreItem`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `entityId` | string | 是 | 对文章是 Content `public_id`；对创作者是 `userId`；对话题是 `topicId`。 |
| `rank` | int | 是 | 从 `1` 开始的名次。 |
| `score` | number | 是 | Ranking 热度分，允许小数。 |
| `updatedAt` | string | 否 | 该分数最后更新时间，RFC3339。 |

### `PostRankItem`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content `public_id`。 |
| `rank` | int | 是 | 从 `1` 开始。 |
| `score` | number | 是 | Ranking 热度分。 |

### `PostDetailRankItem`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content `public_id`。 |
| `rank` | int | 是 | 从 `1` 开始。 |
| `score` | number | 是 | Ranking 热度分。 |
| `post` | object | 是 | Content `PostSummary` 或 typed client 等价摘要对象，由 Content 批量详情接口返回。 |

Ranking 不复制 Content 字段定义；`post` 的字段形态以 Content typed client / HTTP schema 为准。

### `HotPostCandidate`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `publicPostId` | string | 是 | Content `public_id`。 |
| `rank` | int | 是 | 从 `1` 开始。 |
| `score` | number | 是 | Ranking 热度分。 |
| `generatedAt` | string | 是 | 候选生成时间，RFC3339。 |

外部 HTTP 不返回内部 `post_id`。如果服务间 typed client 需要内部引用，应在 `libs/contracts/clients/ranking` 单独定义包含 `internalId` / `publicId` 的 DTO。

### `HotPostCandidateSet`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `version` | string | 是 | 候选集版本。 |
| `generatedAt` | string | 否 | 上次成功生成时间。缺失表示没有可用候选集。 |
| `sourceKey` | string | 是 | 来源 Redis key，例如 `ranking:posts:hot`。 |
| `sourceCount` | int | 是 | 来源榜单参与候选生成的数量。 |
| `candidateSize` | int | 是 | 返回候选数量。 |
| `minScore` | number | 否 | 入选候选的最低分。 |
| `stale` | boolean | 是 | 候选集是否已超过新鲜度阈值。 |
| `degraded` | boolean | 是 | 是否回源或返回旧候选。 |
| `items` | `HotPostCandidate[]` | 是 | 候选列表。 |

### `RankResult`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `entityId` | string | 是 | 查询对象 ID。 |
| `rank` | int | 否 | 当前排名。从 `1` 开始；未上榜时为空。 |
| `score` | number | 否 | 当前分数；未上榜时为空。 |
| `ranked` | boolean | 是 | 是否在当前榜单中。 |
| `rankingType` | string | 是 | `HOT`、`DAILY`、`WEEKLY`、`MONTHLY`。 |
| `periodKey` | string | 否 | 周期榜 period key；总榜为空。 |
| `updatedAt` | string | 否 | 分数更新时间。 |

未上榜但实体存在时返回 HTTP `200`，`ranked=false`，`rank/score` 为空。实体不存在或不可公开时返回 `1005` / HTTP `404`。

### `RebuildAccepted`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `operationId` | string | 是 | rebuild 操作 ID。 |
| `status` | string | 是 | `ACCEPTED`。 |
| `acceptedAt` | string | 是 | RFC3339。 |
| `requestedBy` | string | 是 | 管理员用户 ID。 |
| `dryRun` | boolean | 是 | 是否仅做校验。 |
| `lockTtlSeconds` | int | 是 | rebuild 锁 TTL。 |
| `statusPath` | string | 是 | 状态查询 path。 |

本 endpoint 只负责快速校验和受理；执行进度和最终结果通过 `operationId` 查询。

### `RebuildOperationStatus`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `operationId` | string | 是 | rebuild 操作 ID。 |
| `status` | string | 是 | `ACCEPTED`、`RUNNING`、`SUCCEEDED`、`PARTIAL_FAILED`、`FAILED`、`CANCELED`。 |
| `dryRun` | boolean | 是 | 是否 dry run。 |
| `force` | boolean | 是 | 是否 force；首期始终为 `false`。 |
| `requestedBy` | string | 是 | 管理员用户 ID。 |
| `reason` | string | 否 | 管理员触发原因。 |
| `failedStage` | string | 否 | 失败阶段，例如 `LOCK`、`DRAIN`、`REPLAY`、`REDIS_REFRESH`、`CANDIDATE_REFRESH`。 |
| `errorCode` | string | 否 | 脱敏后的内部错误分类。 |
| `message` | string | 否 | 面向管理员的状态摘要，不包含底层敏感错误。 |
| `replayedEvents` | int64 | 是 | 已 replay 的 ledger 事件数。 |
| `rebuiltPosts` | int64 | 是 | 已重建文章数。 |
| `refreshedSnapshots` | int | 是 | 已刷新 snapshot 数。 |
| `refreshedCandidates` | boolean | 是 | 是否刷新候选集。 |
| `acceptedAt` | string | 是 | RFC3339。 |
| `startedAt` | string | 否 | RFC3339。 |
| `completedAt` | string | 否 | RFC3339。 |
| `durationMs` | int64 | 否 | 已完成时的耗时。 |

## 通用 Query 参数

### Page 参数

适用于所有榜单列表 endpoint。

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `page` | int | 否 | `0` | 从 `0` 开始。 |
| `size` | int | 否 | `20` | `1..100`。 |

### 周期参数

| 榜单 | 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- | --- |
| 日榜 | `date` | string | 否 | 当前 UTC 日期 | 格式 `YYYY-MM-DD`。 |
| 周榜 | `year` | int | 否 | 当前 ISO week-based year | 例如 `2026`。 |
| 周榜 | `week` | int | 否 | 当前 ISO week | `1..53`。 |
| 月榜 | `month` | string | 否 | 当前 UTC 月份 | 格式 `YYYY-MM`。 |

返回中的 `periodKey` 使用 Ranking 内部稳定格式：日榜 `YYYY-MM-DD`、周榜 `YYYY-Www`、月榜 `YYYY-MM`。

## 文章总榜

### `GET /api/v1/ranking/posts/hot`

返回文章总榜 `postId` 列表。

鉴权：匿名 / 服务间调用。

Query：通用 Page 参数。

成功响应 `data`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `items` | string[] | 是 | Content `public_id` 列表。 |
| `page` | int | 是 | 当前页，从 `0` 开始。 |
| `size` | int | 是 | 页大小。 |
| `hasMore` | boolean | 是 | 是否还有更多。 |
| `generatedAt` | string | 是 | RFC3339。 |
| `source` | string | 是 | `REDIS` 或 `POSTGRES`。 |
| `degraded` | boolean | 是 | 是否降级回源。 |

错误：`1001`、`1003`、`1004`。

排序：`hot_score DESC, post_id ASC`。同分按内部 `post_id` 稳定排序，不对外暴露。

### `GET /api/v1/ranking/posts/hot/scores`

返回文章总榜分数和 rank。

鉴权：匿名 / 服务间调用。

Query：通用 Page 参数。

成功响应 `data`：`RankingPage<RankingScoreItem>`，其中 `entityId` 为 Content `public_id`。

错误：`1001`、`1003`、`1004`。

排序：`score DESC, entityId ASC` 的对外稳定语义；内部可用 `hot_score DESC, post_id ASC` 实现。

### `GET /api/v1/ranking/posts/hot/details`

返回文章总榜详情，Ranking 排序后批量调用 Content 补齐文章摘要。

鉴权：匿名 / 服务间调用。

Query：通用 Page 参数。

成功响应 `data`：`RankingPage<PostDetailRankItem>`。

错误：

| code | HTTP status | 触发条件 |
| --- | --- | --- |
| `1001` | `400` | 分页参数非法。 |
| `1004` | `503` | Ranking 查询或 Content 批量详情不可用。 |

Content 详情不可用时，本 endpoint 失败返回 `1004`，不能把只有 score 的结果伪装成详情结果。

## 文章周期榜

### `GET /api/v1/ranking/posts/daily`

返回文章日榜 `postId` 列表。

Query：通用 Page 参数 + `date`。

成功响应 `data`：`RankingPage<string>`，额外字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `periodType` | string | 是 | `DAY`。 |
| `periodKey` | string | 是 | `YYYY-MM-DD`。 |

错误：`1001`、`1003`、`1004`。

### `GET /api/v1/ranking/posts/weekly`

返回文章周榜 `postId` 列表。

Query：通用 Page 参数 + `year` + `week`。

成功响应 `data`：`RankingPage<string>`，额外字段 `periodType=WEEK`、`periodKey=YYYY-Www`。

错误：`1001`、`1003`、`1004`。

### `GET /api/v1/ranking/posts/monthly`

返回文章月榜 `postId` 列表。

Query：通用 Page 参数 + `month`。

成功响应 `data`：`RankingPage<string>`，额外字段 `periodType=MONTH`、`periodKey=YYYY-MM`。

错误：`1001`、`1003`、`1004`。

### `GET /api/v1/ranking/posts/daily/scores`

返回文章日榜分数。

Query：通用 Page 参数 + `date`。

成功响应 `data`：`RankingPage<RankingScoreItem>`，额外字段 `periodType=DAY`、`periodKey=YYYY-MM-DD`。

错误：`1001`、`1003`、`1004`。

### `GET /api/v1/ranking/posts/weekly/scores`

返回文章周榜分数。

Query：通用 Page 参数 + `year` + `week`。

成功响应 `data`：`RankingPage<RankingScoreItem>`，额外字段 `periodType=WEEK`、`periodKey=YYYY-Www`。

错误：`1001`、`1003`、`1004`。

### `GET /api/v1/ranking/posts/monthly/scores`

返回文章月榜分数。

Query：通用 Page 参数 + `month`。

成功响应 `data`：`RankingPage<RankingScoreItem>`，额外字段 `periodType=MONTH`、`periodKey=YYYY-MM`。

错误：`1001`、`1003`、`1004`。

## 单篇文章排名和分数

### `GET /api/v1/ranking/posts/{postId}/rank`

查询单篇文章在总榜中的排名。

Path：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content `public_id`。 |

成功响应 `data`：`RankResult`，`rankingType=HOT`。

错误：

| code | HTTP status | 触发条件 |
| --- | --- | --- |
| `1001` | `400` | `postId` 格式非法。 |
| `1004` | `503` | Content `public_id` 解析或 Ranking 查询不可用。 |
| `1005` | `404` | Content 确认文章不存在、不可公开，或 Ranking 不存在该文章 state。 |

### `GET /api/v1/ranking/posts/{postId}/score`

查询单篇文章当前总榜分数。

Path 同 `GetPostRank`。

成功响应 `data`：`RankResult`，`rankingType=HOT`。

错误同 `GetPostRank`。

## 热门候选集

### `GET /api/v1/ranking/posts/hot/candidates`

返回面向下游服务的热门文章候选集。

鉴权：匿名 / 服务间调用。服务间调用必须携带 `X-Caller-Service` 和 `X-Caller-Operation`。

Query：

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `size` | int | 否 | 服务配置 | `1..500`，不得超过服务配置上限。 |

成功响应 `data`：`HotPostCandidateSet`。

错误：`1001`、`1003`、`1004`。

候选集 stale 时仍可返回旧数据，并设置 `stale=true`。如果 Redis candidate 不可用且 PostgreSQL 回源失败，返回 `1004`。

## 创作者榜

### `GET /api/v1/ranking/creators/hot`

返回创作者总榜 `userId` 列表。

Query：通用 Page 参数。

成功响应 `data`：`RankingPage<string>`，`items` 为 `userId` 列表。

错误：`1001`、`1003`、`1004`。

### `GET /api/v1/ranking/creators/hot/scores`

返回创作者总榜分数和 rank。

Query：通用 Page 参数。

成功响应 `data`：`RankingPage<RankingScoreItem>`，其中 `entityId` 为 `userId`。

错误：`1001`、`1003`、`1004`。

### `GET /api/v1/ranking/creators/{userId}/rank`

查询创作者在总榜中的排名。

Path：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `userId` | string | 是 | User 拥有的用户 ID。 |

成功响应 `data`：`RankResult`，`rankingType=HOT`。

错误：`1001`、`1004`、`1005`。

### `GET /api/v1/ranking/creators/{userId}/score`

查询创作者当前总榜分数。

Path 同 `GetCreatorRank`。

成功响应 `data`：`RankResult`，`rankingType=HOT`。

错误：`1001`、`1004`、`1005`。

## 话题榜

### `GET /api/v1/ranking/topics/hot`

返回话题总榜 `topicId` 列表。

Query：通用 Page 参数。

成功响应 `data`：`RankingPage<string>`，`items` 为 `topicId` 列表。

错误：`1001`、`1003`、`1004`。

### `GET /api/v1/ranking/topics/hot/scores`

返回话题总榜分数和 rank。

Query：通用 Page 参数。

成功响应 `data`：`RankingPage<RankingScoreItem>`，其中 `entityId` 为 `topicId`。

错误：`1001`、`1003`、`1004`。

### `GET /api/v1/ranking/topics/{topicId}/rank`

查询话题在总榜中的排名。

Path：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `topicId` | string | 是 | Content / Topic 拥有的话题 ID。 |

成功响应 `data`：`RankResult`，`rankingType=HOT`。

错误：`1001`、`1004`、`1005`。

### `GET /api/v1/ranking/topics/{topicId}/score`

查询话题当前总榜分数。

Path 同 `GetTopicRank`。

成功响应 `data`：`RankResult`，`rankingType=HOT`。

错误：`1001`、`1004`、`1005`。

## 管理运维

### `POST /api/v1/ranking/admin/rebuild-from-ledger`

管理员触发从 ledger 全量重建 materialized state、Redis snapshot 和候选集。

鉴权：管理员。必须有 `X-User-Id` 和包含管理员角色的 `X-User-Roles`。

Body：

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `dryRun` | boolean | 否 | 缺失为 `false` | 只做权限、依赖和锁校验，不执行重建。 |
| `reason` | string | 否 | 可缺失 | 管理员触发原因，最长 200。 |
| `force` | boolean | 否 | 缺失为 `false` | 首期不允许 `true`；传 `true` 返回 `1008`。 |

成功响应 `data`：`RebuildAccepted`。

错误：

| code | HTTP status | 触发条件 |
| --- | --- | --- |
| `1001` | `400` | body 字段非法或 `reason` 超长。 |
| `1004` | `503` | PostgreSQL、Redis lock 或必要依赖不可用。 |
| `1008` | `409` | 已有 rebuild 正在运行，或 `force=true` 但当前阶段不允许。 |
| `2006` | `401` | 缺少登录态。 |
| `2007` | `403` | 缺少管理员角色。 |
| `2008` | `403` | 已登录但无权执行 rebuild。 |

`rebuild-from-ledger` 是长任务入口。请求成功只表示任务已受理，不表示 rebuild 已完成。Ranking 必须写入 `ranking_rebuild_operation`，后续通过 `operationId` 查询状态。

### `GET /api/v1/ranking/admin/rebuild-operations/{operationId}`

查询 rebuild 操作状态。

鉴权：管理员。必须有 `X-User-Id` 和包含管理员角色的 `X-User-Roles`。

Path：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `operationId` | string | 是 | rebuild 操作 ID。 |

成功响应 `data`：`RebuildOperationStatus`。

错误：

| code | HTTP status | 触发条件 |
| --- | --- | --- |
| `1001` | `400` | `operationId` 格式非法。 |
| `1005` | `404` | 操作不存在或已按保留策略清理。 |
| `2006` | `401` | 缺少登录态。 |
| `2007` | `403` | 缺少管理员角色。 |
| `2008` | `403` | 已登录但无权查询 rebuild 操作。 |

## 权限和可见性

- 公开榜单只包含 `public_visible=true` 的文章。
- Ranking 不在查询主路径逐条回源 Content 判断可见性。
- `ListHotPostsWithDetails` 只有在 Content 批量详情成功时返回详情；Content 不可用时返回 `1004`。
- 管理 endpoint 只接受 Gateway 注入的可信管理员身份。

## 排序、分页和过滤

- 所有榜单列表使用 page 分页，从 `0` 开始。
- `size` 默认 `20`，最大 `100`；候选集 `size` 最大 `500` 或服务配置上限，两者取小。
- 总榜排序：`score DESC, entityId ASC` 的对外稳定语义。
- 周期榜排序：`deltaScore DESC, entityId ASC` 的对外稳定语义。
- 日榜 `date` 使用 `YYYY-MM-DD`；周榜使用 ISO week-based year + week；月榜 `month` 使用 `YYYY-MM`。

## 测试要求

- Handler contract test：每个 endpoint 覆盖成功 envelope、错误 envelope、分页默认值、分页上限和非法参数。
- Redis miss：列表 endpoint 覆盖 Redis miss 后 PostgreSQL 回源，且 `degraded=true`。
- Visibility：公开榜单过滤 `public_visible=false` 的文章。
- Content 详情：`/posts/hot/details` 覆盖 Content 批量详情失败返回 `1004`。
- Admin rebuild：覆盖未登录、非管理员、lock 不可用、已有任务运行、accepted response、status 查询和 status not found。
- Ranking API 仍未实现时，本文状态保持“草案”；handler 和 contract test 落地后再更新为“已验证”。
