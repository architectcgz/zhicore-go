# Comment API 背后设计

本文只描述 API 背后的业务流程、权限、状态机和副作用；字段级 HTTP schema 放在 `services/zhicore-comment/api/http/`。

## 鉴权上下文

| API | 鉴权 | 说明 |
| --- | --- | --- |
| 创建评论 / 回复 | 登录用户 | Gateway 注入 `X-User-Id`；application 校验文章和用户互动权限。 |
| 查询评论 / 回复 | 匿名 / 登录用户 | 匿名只读可见评论；登录用户可返回 viewer 点赞状态。 |
| 更新评论 | 作者 | 只能更新自己的未删除评论。 |
| 删除评论 | 作者 / 管理员 | 普通用户只能删除自己的评论；管理员删除必须携带操作者。 |
| 点赞 / 取消点赞 | 登录用户 | 已删除评论不能点赞；重复操作按目标 contract 幂等成功。 |
| 管理端查询 / 删除 / outbox 运维 | 管理员 | 需要 `X-User-Id` 和管理员角色。 |

客户端伪造的 `X-User-*` header 必须由 Gateway 清理后重新注入。Gateway 注入的 `X-User-Id` 是 User 内部 `UserID`，Comment handler 将其解析为 application `Actor.UserID`，不从 request body 接收当前操作者 `userId`。HTTP 响应中的作者摘要使用 User `publicId`，不暴露内部 `UserID`。

## Use Case 追踪

| Endpoint | Use case | 主要副作用 |
| --- | --- | --- |
| `POST /api/v1/posts/{postId}/comments` | `CreateComment` / `CreateReply` | 写 `comments`、初始化 `comment_stats`、维护 `comment_post_stats`、根评论初始化 rank 行、回复时递增根评论回复数、写 `comment.created` outbox、失效列表缓存。 |
| `GET /api/v1/posts/{postId}/comments/page` | `ListTopLevelCommentsByPage` | 无业务写入；可读取列表缓存和作者摘要。 |
| `GET /api/v1/posts/{postId}/comments/cursor` | `ListTopLevelCommentsByCursor` | 无业务写入；解码 opaque cursor。 |
| `GET /api/v1/posts/{postId}/comments/incremental` | `ListTopLevelCommentsIncremental` | 无业务写入；按稳定锚点补拉。 |
| `GET /api/v1/posts/{postId}/comments/{commentId}` | `GetCommentDetail` | 无业务写入；回复详情返回 `rootCommentId` / `parentCommentId`。 |
| `PUT /api/v1/posts/{postId}/comments/{commentId}` | `UpdateComment` | 更新评论内容；事务后失效详情、列表和回复缓存。 |
| `DELETE /api/v1/posts/{postId}/comments/{commentId}` | `DeleteComment` | 软删除目标评论及其整棵子树；维护 `reply_count`、`comment_post_stats` 和 rank 可见性；写一条 `comment.deleted` outbox。 |
| `GET /api/v1/posts/{postId}/comments/{commentId}/replies/page` | `ListRepliesByPage` | 无业务写入；`commentId` 必须是根评论。 |
| `GET /api/v1/posts/{postId}/comments/{commentId}/replies/cursor` | `ListRepliesByCursor` | 无业务写入；`commentId` 必须是根评论。 |
| `GET /api/v1/posts/{postId}/comments/{commentId}/replies/incremental` | `ListRepliesIncremental` | 无业务写入；按稳定锚点补拉。 |
| `POST /api/v1/posts/{postId}/comments/{commentId}/like` | `LikeComment` | 插入点赞关系、追加点赞计数 delta、写 `comment.liked` outbox、返回强一致 `liked=true`。 |
| `DELETE /api/v1/posts/{postId}/comments/{commentId}/like` | `UnlikeComment` | 删除点赞关系、追加点赞计数 delta、写 `comment.unliked` outbox、返回强一致 `liked=false`。 |
| `GET /api/v1/posts/{postId}/comments/{commentId}/liked` | `GetLikeStatus` | 无业务写入。 |
| `GET /api/v1/posts/{postId}/comments/{commentId}/like-count` | `GetLikeCount` | 无业务写入。 |
| `POST /api/v1/posts/{postId}/comments/batch/liked` | `BatchGetLikeStatus` | 无业务写入。 |
| `GET /api/v1/admin/comments` | `ListAdminComments` | 无业务写入。 |
| `DELETE /api/v1/admin/comments/posts/{postId}/comments/{commentId}` | `AdminDeleteComment` | 委托 Comment mutation 按 `(postId, commentId)` 软删除，保存最小删除元数据，Admin 继续记录完整审计。 |
| `GET /api/v1/admin/comment-outbox/summary` | `GetOutboxSummary` | 无业务写入。 |
| `POST /api/v1/admin/comment-outbox/retry-dead` | `RetryDeadOutboxEvents` | 将 DEAD 事件重置为 PENDING，记录操作者和审计字段。 |

## 外部定位

- 对外 path 使用 `(postId, commentId)` 定位评论；`commentId` 是 Comment 对外 ID，不是内部自增数字。
- `postId` 是 Content 的对外文章 ID。Comment application 通过 Content contract 校验文章事实后，在本地表中保存公开 `postId` 字符串和 Content 内部 `post_id BIGINT` opaque reference；HTTP contract 不暴露内部 `post_id`。
- `commentId` 由内部 `comments.id BIGINT IDENTITY` 派生，排序和 cursor 的内部锚点使用 `comments.id`。
- 创建回复时，`parentCommentId` 可以指向根评论或任意回复；application 在同一事务内解析直接父评论，并校验同文章、未删除、根归属正确。
- 回复列表接口中的 `{commentId}` 表示根评论；创建回复的 `parentCommentId` 可以指向根评论或任意未删除回复。
- 非 Admin 公开 API 对不存在和已删除评论统一返回 404，不向普通用户暴露 `DELETED` 状态。

## 创建评论流程

```text
HTTP handler
-> 解析 postId、Actor、body
-> CreateComment
-> ContentPostClient 校验文章存在、可见、允许评论
-> UserProfileClient 校验作者状态
-> UserRelationClient 校验文章作者 / 父评论作者是否拉黑当前用户
-> FileReferenceClient 校验媒体文件引用
-> TransactionRunner:
     事务内校验父评论未删除、同文章、根归属正确
     CommentFactory 创建根评论或回复
     CommentCommandRepository 保存 comments
     CommentStatsRepository 初始化统计
     CommentPostStatsRepository 维护文章级评论总数
     根评论时初始化 hot rank / recommended rank 行
     回复时 CommentStatsRepository 递增根评论 reply_count
     OutboxPublisher 写 comment.created，payload 带 publicId、internalId、postAuthorId、root/parent 评论 ID 和作者事实
-> 提交后失效列表 / 回复 / 首页缓存
```

评论整体必须至少包含文本、图片或语音中的一项。图片最多 9 张。语音不能和图片同时存在。

## 查询流程

- 顶级评论列表只返回未删除根评论。
- 回复列表按根评论 `commentId` 展开，不依赖“根评论下第几条回复”的序号。
- 评论详情返回当前评论 `commentId`；如果是回复，同时返回 `rootCommentId` 和 `parentCommentId`，供前端展开根评论、定位父评论和高亮目标回复。
- 评论列表展示作者摘要时，优先批量调用 User contract 或使用本地快照；不得直接读取 User 数据库。作者摘要解析失败时查询可以降级为占位作者，写路径的用户状态和权限校验不能降级。
- 登录用户查询默认返回 `viewer.liked`；匿名用户省略 `viewer`，前端按未点赞态展示。

## 分页和排序

- 传统分页用于 Web 固定页码场景，Go-first API 默认 `page` 从 `1` 开始，`size` 必须有上限。
- 游标分页用于移动端无限滚动，cursor 对外不透明。
- 增量补拉使用 cursor 内部排序锚点；对外不暴露内部 `comments.id`。
- 顶级评论默认 `RECOMMENDED` 排序，固定为 `recommended_score DESC, comment_id DESC`。同分时新评论优先。
- 顶级评论 `TIME` 排序固定为 `comment_id DESC`。`createdAt` 是展示和审计字段，不作为排序锚点。
- 顶级评论 `HOT` 排序固定为 `like_count DESC, comment_id ASC`。`like_count` 来自 `comment_hot_rank` 读模型；同点赞数下优先展示更早评论。
- 回复列表默认 `HOT`，固定为 `like_count DESC, comment_id ASC`；可选 `TIME` 使用 `comment_id ASC`。回复列表返回根评论下整棵回复子树的平铺列表。
- 顶级评论列表返回 `totalComments` 和 `totalTopLevelComments`。`totalComments` 统计根评论和回复的全部未删除评论；`totalTopLevelComments` 只统计未删除根评论。
- Cursor 必须编码对应排序的全部稳定锚点：`RECOMMENDED` 使用 `recommendedScore + 内部 commentId`，`HOT` 使用 `likeCount + 内部 commentId`，`TIME` 使用内部 `commentId`。

## 管理和媒体边界

- Admin 删除可以从 Admin facade 进入，但最终 mutation 属于 Comment。Comment 只保存 `deletedBy`、`deleteReason`、`deletedAt` 等执行删除所需的最小元数据。
- Admin 删除必须携带 `deleteReason`；作者删除不要求原因。完整审核审计仍归 Admin。
- Comment 不提供媒体上传 facade。前端先调用 Upload 获得文件 ID，Comment 创建 / 更新只接收并校验 `imageFileIds` / `voiceFileId`。
