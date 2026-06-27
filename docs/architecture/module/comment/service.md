# Comment Application Service 设计

Comment application 层按 commands / queries 组织 use case。application 拥有事务边界、权限上下文、幂等、端口调用、缓存失效、outbox 写入和错误映射。

## 命令用例

| Use case | 职责 |
| --- | --- |
| `CreateComment` | 校验文章和作者事实，分配文章内 `floor`，创建根评论或回复，写 `comments`、初始化 `comment_stats`、根评论初始化 `comment_hot_rank`、回复时更新根评论回复数，写 `comment.created` outbox。 |
| `UpdateComment` | 校验作者权限和状态，更新评论内容，事务后删除详情和列表缓存。 |
| `DeleteComment` | 普通用户删除自己的评论；顶级评论删除时在同一事务内批量软删除回复，写 `comment.deleted` outbox，事务后清理缓存。 |
| `AdminDeleteComment` | 管理员删除评论，必须携带 `deletedBy`，可携带 `deleteReason`；Admin 继续记录完整审核审计。 |
| `LikeComment` / `UnlikeComment` | 以 `comment_likes` 维护用户点赞事实；实际状态变化时写计数 delta 和 `comment.liked` / `comment.unliked` outbox，不在请求事务内同步更新点赞统计行。 |
| `UploadCommentImage` / `UploadCommentVoice` | 作为 Upload / File Service adapter 的 HTTP facade，不表示 Comment 拥有文件事实。 |
| `RetryDeadOutboxEvents` | 管理端把 DEAD 事件重置为 PENDING，记录操作者和审计字段。 |
| `SyncRankingHotCandidates` | 从 Ranking contract 同步热门文章候选到 Comment 本地缓存或物化输入。 |

## 查询用例

| Use case | 职责 |
| --- | --- |
| `GetCommentDetail` | 返回评论详情；回复详情包含 `rootFloor` 和 `parentFloor`。 |
| `ListTopLevelCommentsByPage` | 文章顶级评论传统分页，支持 `TIME` / `HOT` 排序。 |
| `ListTopLevelCommentsByCursor` | 文章顶级评论游标分页。 |
| `ListRepliesByPage` | 根评论回复传统分页。 |
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
| `ApplyCommentCounterDeltas` | 后台批量聚合点赞 delta，更新 `comment_stats.like_count` 和顶级评论 `comment_hot_rank.like_count`。 |

## 事务边界

### 创建根评论

```text
单个 PostgreSQL 事务：
  comment_post_counters 按 Content 公开 postId 分配 floor
  + comments 插入根评论
  + comment_stats 初始化
  + comment_hot_rank 初始化根评论 HOT 排序行，like_count = 0
  + outbox_events(comment.created)

事务提交后：
  删除文章评论列表缓存
  best-effort 更新首页评论缓存
```

`floor` 分配必须使用 `comment_post_counters` 或等价的事务内计数器，不使用 `SELECT max(floor) + 1`。

### 创建回复

```text
单个 PostgreSQL 事务：
  comment_post_counters 按 Content 公开 postId 分配 floor
  + comments 插入回复
  + comment_stats 初始化回复统计
  + comment_stats(root_comment).reply_count 原子 +1
  + outbox_events(comment.created)

事务提交后：
  删除根评论回复列表缓存
  删除文章评论列表缓存中对应根评论统计
```

创建回复时，application / repository 必须在同一事务内校验根评论是顶级评论、父评论属于同一文章、父评论未删除且属于同一根评论树。

### 更新评论

```text
comments 更新内容和更新时间
```

事务提交后删除评论详情、文章评论列表和回复列表缓存。第一阶段如果没有明确 consumer，可以只做内部缓存失效，不发布 `comment.updated` 集成事件。

### 删除评论

```text
comments 标记目标评论 deleted
+ 如果目标是顶级评论，批量标记所有回复 deleted
+ 如果目标是回复，comment_stats(root_comment).reply_count 原子 -1
+ outbox_events(comment.deleted，包含 affectedCount)
```

顶级评论批量删除回复时，第一阶段发布一条 `comment.deleted` 集成事件，并用 `affectedCount` 表示本次软删除影响的评论总数。

### 点赞 / 取消点赞

```text
comment_likes 插入/删除
+ comment_counter_deltas 追加 LIKE +/-1 delta
+ outbox_events(comment.liked / comment.unliked)
```

重复点赞、取消未点赞按目标 contract 幂等成功，不重复写 delta 或事件。`viewer.liked` 以 `comment_likes` 为强一致事实；`likeCount`、顶级评论 HOT 排序和缓存中的点赞数允许短暂最终一致。

后台 worker 按批次 claim `comment_counter_deltas`，按 `comment_id` 聚合后更新 `comment_stats.like_count`；如果目标评论是顶级评论，同步更新 `comment_hot_rank.like_count`。worker 必须幂等，delta 应有状态或消费标记，失败可重试，统计可通过 `comment_likes` 重建。

## 缓存失效

| 操作 | 失效或更新 |
| --- | --- |
| 创建根评论 | 删除文章顶级评论时间排序、热度排序、文章评论数和首页评论快照。 |
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
| 评论不存在 | `5001` |
| 评论已删除 | `5002` |
| 评论内容为空 | `5003` |
| 评论内容过长 | `5004` |
| 根评论不存在 | `5005` |
| 被回复评论不存在 | `5006` |

底层 not-found、重复键、Redis nil、HTTP 错误由 infrastructure adapter 翻译为 module-local 语义，再由 application 映射为公开错误。

## 推荐首个实现切片

### 切片 1：创建根评论 / 回复 + 文章评论传统分页查询

- Domain：建 `Comment` 聚合、`CommentStats` 聚合、`CommentFactory`、`CommentContentPolicy` 和 `CommentCreated` 领域事件。
- Ports：定义 `TransactionRunner`、`CommentFloorAllocator`、`CommentCommandRepository`、`CommentQueryRepository`、`CommentStatsRepository`、`OutboxPublisher`、`ContentPostClient`、`UserProfileClient`、`Clock`。
- Application：实现 `CreateComment` / `CreateReply` 和 `ListTopLevelCommentsByPage`，用内存 fake 验证 Content/User 校验、`parentFloor` 解析、统计初始化、根评论 HOT rank 初始化、outbox 写入和传统分页查询。
- Infrastructure：实现 PostgreSQL repository，以及 Content、User client adapter 的最小能力。
- HTTP：实现 `POST /api/v1/posts/{postId}/comments` 和 `GET /api/v1/posts/{postId}/comments/page`。

`POST /api/v1/posts/{postId}/comments` 的 contract 已声明 `parentFloor`；首个交付切片如果只支持根评论，会造成 contract test 与实现冲突。

### 切片 2：删除评论 + 点赞 + 计数 delta worker

- 补 `DeleteComment`、`AdminDeleteComment`、`LikeComment`、`UnlikeComment`。
- 补 `CommentLike` 关系实体、`CommentLikeRepository`、`CommentCounterDeltaRepository` 和 `ApplyCommentCounterDeltas` worker。
- 补 `OutboxPublisher`、outbox writer 和 dispatcher。
- 补 `comment.deleted`、`comment.liked`、`comment.unliked` 事件落库和发布。

### 切片 3：游标分页 + 回复列表 + 增量补拉

- 补 `ListRepliesByPage`、`ListTopLevelCommentsByCursor`、`ListRepliesByCursor`、`ListTopLevelCommentsIncremental`、`ListRepliesIncremental`。
- 补 `CommentTreePolicy`、`RootCommentID`、`ParentCommentID`。
- 补目标 cursor codec：`TIME` 和 `HOT` URL-safe Base64 无 padding。
