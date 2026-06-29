# Comment Application Service 设计

Comment application 层按 commands / queries 组织 use case。application 拥有事务边界、权限上下文、幂等、端口调用、缓存失效、outbox 写入和错误映射。

## 命令用例

| Use case | 职责 |
| --- | --- |
| `CreateComment` | 事务外校验文章、作者状态、拉黑关系和媒体引用；事务内校验父评论 / 根评论树结构，创建根评论或回复，插入 `comments` 并由 PostgreSQL identity 生成内部 `comment_id`，初始化 `comment_stats`、维护 `comment_post_stats`，根评论初始化 `comment_hot_rank` / `comment_recommended_rank`，回复时更新根评论回复数，写 `comment.created` outbox。 |
| `UpdateComment` | 校验作者权限，复用创建评论的用户状态、拉黑关系和媒体引用 guard，整体替换文本 / 媒体，设置 `editedAt`，事务后删除详情和列表缓存；不发布 `comment.updated`。 |
| `DeleteComment` | 普通用户删除自己的评论；删除任意评论节点时在同一事务内软删除该节点及整棵子树，维护根评论 `reply_count`、`comment_post_stats` 和 rank 可见性，写一条 `comment.deleted` outbox。 |
| `AdminDeleteComment` | 管理员删除评论，必须携带 `deletedBy` 和非空 `deleteReason`；重复删除返回成功但不产生事件或统计副作用；Admin 继续记录完整审核审计。 |
| `LikeComment` / `UnlikeComment` | 以 `comment_likes` 维护用户点赞事实；实际状态变化时写计数 delta 和 `comment.liked` / `comment.unliked` outbox，不在请求事务内同步更新点赞统计行。 |
| `RetryDeadOutboxEvents` | 管理端把 DEAD 事件重置为 PENDING，记录操作者和审计字段。 |
| `SyncRankingHotCandidates` | 从 Ranking contract 同步热门文章候选到 Comment 本地缓存或物化输入。 |

## 查询用例

| Use case | 职责 |
| --- | --- |
| `GetCommentDetail` | 返回评论详情；回复详情包含 `rootCommentId` 和 `parentCommentId`。 |
| `ListTopLevelCommentsByPage` | 文章顶级评论传统分页，默认 `RECOMMENDED`，支持 `HOT` / `TIME` 排序；返回 `totalComments` 和 `totalTopLevelComments`。 |
| `ListTopLevelCommentsByCursor` | 文章顶级评论游标分页。 |
| `ListRepliesByPage` | 根评论回复传统分页，平铺返回整棵回复子树，默认 `HOT`，支持 `TIME`。 |
| `ListRepliesByCursor` | 根评论回复游标分页。 |
| `ListTopLevelCommentsIncremental` | 按稳定锚点做文章顶级评论增量补拉。 |
| `ListRepliesIncremental` | 按稳定锚点做回复增量补拉。 |
| `GetLikeStatus` / `BatchGetLikeStatus` / `GetLikeCount` | 点赞状态和点赞数查询。 |
| `ListAdminComments` | 管理端按 keyword、postId、userId、status 筛选。 |
| `GetOutboxSummary` | outbox 事件状态摘要。 |

查询用例可以返回 DTO 或视图模型，不把复杂列表查询塞进 domain。

## 后台作业

| Job | 职责 |
| --- | --- |
| `ApplyCommentCounterDeltas` | 后台批量聚合点赞 delta，更新 `comment_stats.like_count`、`comment_hot_rank.like_count` 和 `comment_recommended_rank`。 |
| `DecayRecommendedRank` | 按 `next_decay_at` 批量 claim 推荐 rank 行，更新 `freshness_tier`、`recommended_score` 和下一次衰减时间。 |

## 写路径前置校验

创建 / 更新评论的外部依赖校验在 Comment 数据库事务外执行，校验不通过直接返回，不进入事务：

- Content：校验 `postId` 存在、公开可见且允许评论，返回 `postAuthorId` 和 Content 内部 `post_id BIGINT` opaque reference。
- User / Auth：校验当前用户存在且状态可互动。
- User relation：根评论校验 `postAuthorId -> actorUserId` 拉黑关系；回复校验 `postAuthorId -> actorUserId` 和 `parentAuthorId -> actorUserId`；更新评论复用对应创建场景的拉黑关系 guard。
- Upload：校验 `imageFileIds` / `voiceFileId` 存在、类型匹配且状态可用。

Content / User / Upload 不可用时写请求失败，不降级写入本地评论事实。查询路径可以对作者摘要和媒体 URL 解析做降级。完整 timeout、retry、熔断和降级矩阵见 `runtime-resilience.md`。

## 事务边界

### 创建根评论

```text
单个 PostgreSQL 事务：
  本地限流通过后进入事务
  comments 插入根评论并返回 comment_id
  + comment_stats 初始化
  + comment_post_stats UPSERT 并 total_comments +1、total_top_level_comments +1
  + comment_hot_rank 初始化根评论 HOT 排序行，like_count = 0
  + comment_recommended_rank 初始化根评论默认排序行
  + outbox_events(comment.created，包含 publicId 和 internalId)

事务提交后：
  删除文章评论列表缓存
  best-effort 更新首页评论缓存
```

评论 ID 由 `comments.id BIGINT IDENTITY` 在插入时生成，详见 `comment-id.md`。创建评论不再调用 Redis、segment、每文章 counter 或独立发号服务。

### 创建回复

```text
单个 PostgreSQL 事务：
  锁定或一致性读取父评论，校验父评论仍 NORMAL、同 postId，并推导 root_id
  + comments 插入回复并返回 comment_id
  + comment_stats 初始化回复统计
  + comment_stats(root_comment).reply_count 原子 +1
  + comment_post_stats UPSERT 并 total_comments +1
  + outbox_events(comment.created，包含 publicId 和 internalId)

事务提交后：
  删除根评论回复列表缓存
  删除文章评论列表缓存中对应根评论统计
```

创建回复时，application / repository 必须在同一事务内校验父评论属于同一文章、未删除且根归属正确；父评论通过 `parentCommentId` 解析为内部 `parent_id`。

### 更新评论

```text
comments 整体替换文本 / 媒体引用，更新 updated_at，并设置 edited_at
```

事务提交后删除评论详情、文章评论列表和回复列表缓存。当前设计不发布 `comment.updated` 集成事件；只有未来出现明确 consumer 时再新增 contract。编辑不改变 `TIME`、`HOT` 或 `RECOMMENDED` 排序位置。

### 删除评论

```text
comments 将目标节点及其 descendants 从 NORMAL 标记为 DELETED
+ 按本次实际变更行数计算 affectedCount
+ 如果目标是回复，comment_stats(root_comment).reply_count 原子 -affectedCount
+ comment_post_stats.total_comments 原子 -affectedCount
+ 如果目标是顶级评论，comment_post_stats.total_top_level_comments 原子 -1
+ 如果目标是顶级评论，comment_hot_rank / comment_recommended_rank 标记 visible=false
+ outbox_events(comment.deleted，包含 affectedCount、isRoot)
```

删除任意评论节点都软删除该节点及整棵子树。无论影响多少条回复，只发布一条 `comment.deleted` 集成事件，用 `affectedCount` 表示本次实际从 `NORMAL` 变为 `DELETED` 的评论数量。普通用户重复删除已删除评论返回 404；Admin 重复删除返回成功但 `affectedCount=0` 且不写 outbox。

### 点赞 / 取消点赞

```text
comment_likes 插入/删除
+ comment_counter_deltas 追加 LIKE +/-1 delta
+ outbox_events(comment.liked / comment.unliked)
```

重复点赞、取消未点赞按目标 contract 幂等成功，不重复写 delta 或事件。`viewer.liked` 以 `comment_likes` 为强一致事实；`likeCount`、顶级评论 HOT 排序和缓存中的点赞数允许短暂最终一致。

后台 worker 按批次 claim `comment_counter_deltas`，按 `comment_id` 聚合后更新 `comment_stats.like_count`；如果目标评论是顶级评论，同步更新 `comment_hot_rank.like_count` 和 `comment_recommended_rank.like_count/recommended_score`。worker 必须幂等，delta 应有状态或消费标记，失败可重试，统计可通过 `comment_likes` 重建。

点赞评论时校验评论作者是否拉黑当前用户；取消点赞不校验拉黑关系，只允许撤销自己的历史点赞。删除评论不清理 `comment_likes`，也不扣减历史 `like_count`；已删除评论不能新增点赞，不进入普通查询和排序。

## 缓存失效

| 操作 | 失效或更新 |
| --- | --- |
| 创建根评论 | 删除文章顶级评论推荐、时间、热度排序、文章评论数和首页评论快照。 |
| 创建回复 | 删除根评论回复列表、根评论回复数、根评论详情、文章顶级评论列表和首页评论快照。 |
| 更新评论 | 删除评论详情、所属文章列表、所属根评论回复列表和首页评论快照。 |
| 删除根评论 | 删除目标评论详情、文章顶级评论列表、文章评论数、该根评论回复列表、回复详情缓存、相关点赞状态 / 计数缓存和首页快照。 |
| 删除回复 | 删除回复详情、根评论回复列表、根评论详情、根评论回复数、文章顶级评论列表中根评论统计和首页快照。 |
| 点赞 / 取消点赞 | 立即删除或更新用户点赞状态；点赞数、目标评论详情、所属列表和首页快照由 delta worker 应用后失效或刷新。 |

需要通配删除的 key 由 Comment 本地 cache store 封装实现，application 只表达“失效文章评论列表、根评论回复列表、首页快照”等语义。

## 错误映射

| 语义 | 对外错误码候选 |
| --- | --- |
| 参数缺失、格式错误、枚举非法、分页非法 | `1001` |
| 下游 Content / User / Upload / Ranking 不可用 | `1004` |
| 缺少登录态 | `2006` |
| 需要管理员角色 | `2007` |
| 非作者修改或删除评论 | `2008` |
| 文章不存在或不可评论 | `4001` 或服务级 contract 登记的 Content 错误 |
| 评论不存在或已删除 | `5001` |
| 评论内容为空 | `5003` |
| 评论内容过长 | `5004` |
| 根评论不存在 | `5005` |
| 被回复评论不存在 | `5006` |

底层 not-found、重复键、Redis nil、HTTP 错误由 infrastructure adapter 翻译为 module-local 语义，再由 application 映射为公开错误。普通用户公开 API 不暴露 `DELETED` 状态，已删除评论按 404 处理；Admin 查询 / 删除可以看到删除元数据。HTTP `message` 使用英文稳定文案，前端负责 i18n。

## 推荐首个实现切片

### 切片 1：创建根评论 / 回复 + 文章评论传统分页查询

- Domain：建 `Comment` 聚合、`CommentStats` 聚合、`CommentFactory`、`CommentContentPolicy` 和 `CommentCreated` 领域事件。
- Ports：定义 `TransactionRunner`、`CommentIDCodec`、`CommentCommandRepository`、`CommentQueryRepository`、`CommentStatsRepository`、`CommentPostStatsRepository`、`OutboxPublisher`、`ContentPostClient`、`UserProfileClient`、`UserRelationClient`、`FileReferenceClient`、`RateLimiter`、`Clock`。
- Application：实现 `CreateComment` / `CreateReply` 和 `ListTopLevelCommentsByPage`，用内存 fake 验证 Content/User 校验、`parentCommentId` 解析、统计初始化、根评论 HOT rank 初始化、outbox 写入和传统分页查询。
- Infrastructure：实现 PostgreSQL repository，以及 Content、User client adapter 的最小能力。
- HTTP：实现 `POST /api/v1/posts/{postId}/comments` 和 `GET /api/v1/posts/{postId}/comments/page`。

`POST /api/v1/posts/{postId}/comments` 的 contract 已声明 `parentCommentId`；首个交付切片如果只支持根评论，会造成 contract test 与实现冲突。

### 切片 2：删除评论 + 点赞 + 计数 delta worker

- 补 `DeleteComment`、`AdminDeleteComment`、`LikeComment`、`UnlikeComment`。
- 补 `CommentLike` 关系实体、`CommentLikeRepository`、`CommentCounterDeltaRepository`、`CommentHotRankRepository`、`CommentRecommendedRankRepository` 和 `ApplyCommentCounterDeltas` / `DecayRecommendedRank` worker。
- 补 `OutboxPublisher`、outbox writer 和 dispatcher。
- 补 `comment.deleted`、`comment.liked`、`comment.unliked` 事件落库和发布。

### 切片 3：游标分页 + 回复列表 + 增量补拉

- 补 `ListRepliesByPage`、`ListTopLevelCommentsByCursor`、`ListRepliesByCursor`、`ListTopLevelCommentsIncremental`、`ListRepliesIncremental`。
- 补 `CommentTreePolicy`、`RootCommentID`、`ParentCommentID`。
- 补目标 cursor codec：`RECOMMENDED`、`TIME` 和 `HOT` URL-safe Base64 无 padding。
