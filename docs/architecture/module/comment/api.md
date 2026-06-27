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
| 媒体上传 facade | 登录用户 | 只是转发 Upload / File Service，不转移文件事实归属。 |

客户端伪造的 `X-User-*` header 必须由 Gateway 清理后重新注入。Comment handler 不从 request body 接收当前操作者 `userId`。

## Use Case 追踪

| Endpoint | Use case | 主要副作用 |
| --- | --- | --- |
| `POST /api/v1/posts/{postId}/comments` | `CreateComment` / `CreateReply` | 写 `comments`、初始化 `comment_stats`、回复时递增根评论回复数、写 `comment.created` outbox、失效列表缓存。 |
| `GET /api/v1/posts/{postId}/comments/page` | `ListTopLevelCommentsByPage` | 无业务写入；可读取列表缓存和作者摘要。 |
| `GET /api/v1/posts/{postId}/comments/cursor` | `ListTopLevelCommentsByCursor` | 无业务写入；解码 opaque cursor。 |
| `GET /api/v1/posts/{postId}/comments/incremental` | `ListTopLevelCommentsIncremental` | 无业务写入；按稳定锚点补拉。 |
| `GET /api/v1/posts/{postId}/comments/{floor}` | `GetCommentDetail` | 无业务写入；回复详情返回 `rootFloor` / `parentFloor`。 |
| `PUT /api/v1/posts/{postId}/comments/{floor}` | `UpdateComment` | 更新评论内容；事务后失效详情、列表和回复缓存。 |
| `DELETE /api/v1/posts/{postId}/comments/{floor}` | `DeleteComment` | 软删除评论；根评论删除时批量软删除回复；写 `comment.deleted` outbox。 |
| `GET /api/v1/posts/{postId}/comments/{floor}/replies/page` | `ListRepliesByPage` | 无业务写入；`floor` 必须是根评论楼层。 |
| `GET /api/v1/posts/{postId}/comments/{floor}/replies/cursor` | `ListRepliesByCursor` | 无业务写入；`floor` 必须是根评论楼层。 |
| `GET /api/v1/posts/{postId}/comments/{floor}/replies/incremental` | `ListRepliesIncremental` | 无业务写入；按稳定锚点补拉。 |
| `POST /api/v1/posts/{postId}/comments/{floor}/like` | `LikeComment` | 插入点赞关系、递增点赞数、写 `comment.liked` outbox、更新或失效缓存。 |
| `DELETE /api/v1/posts/{postId}/comments/{floor}/like` | `UnlikeComment` | 删除点赞关系、递减点赞数、写 `comment.unliked` outbox、更新或失效缓存。 |
| `GET /api/v1/posts/{postId}/comments/{floor}/liked` | `GetLikeStatus` | 无业务写入。 |
| `GET /api/v1/posts/{postId}/comments/{floor}/like-count` | `GetLikeCount` | 无业务写入。 |
| `POST /api/v1/posts/{postId}/comments/batch/liked` | `BatchGetLikeStatus` | 无业务写入。 |
| `GET /api/v1/admin/comments` | `ListAdminComments` | 无业务写入。 |
| `DELETE /api/v1/admin/comments/posts/{postId}/comments/{floor}` | `AdminDeleteComment` | 委托 Comment mutation 按 `(postId, floor)` 软删除，保存最小删除元数据，Admin 继续记录完整审计。 |
| `GET /api/v1/admin/comment-outbox/summary` | `GetOutboxSummary` | 无业务写入。 |
| `POST /api/v1/admin/comment-outbox/retry-dead` | `RetryDeadOutboxEvents` | 将 DEAD 事件重置为 PENDING，记录操作者和审计字段。 |

## 外部定位

- 对外 path 不使用全局 `commentId`；评论资源由 `(postId, floor)` 定位。
- `postId` 是 Content 的对外文章 ID。Comment application 通过 Content contract 校验文章事实后，在本地表中保存该公开 `postId` 字符串，不保存 Content 内部 `BIGINT` 主键。
- `floor` 是文章内单调递增楼层号，根评论和回复共享同一序列；删除后不复用、不重排。
- 创建回复时，`parentFloor` 可以指向根评论或任意回复；application 在同一事务内解析直接父评论，并校验同文章、未删除、根归属正确。
- 回复列表接口中的 `{floor}` 表示根评论楼层。第一阶段如果传入回复自身楼层，返回参数错误，避免隐式展开带来额外查询和语义混淆。

## 创建评论流程

```text
HTTP handler
-> 解析 postId、Actor、body
-> CreateComment
-> ContentPostClient 校验文章存在、可见、允许评论
-> UserProfileClient 校验作者状态；后续需要拉黑或互动权限时接入 UserRelationClient
-> TransactionRunner:
     CommentFloorAllocator 分配 floor
     CommentFactory 创建根评论或回复
     CommentCommandRepository 保存 comments
     CommentStatsRepository 初始化统计
     根评论时初始化 hot rank 行
     回复时 CommentStatsRepository 递增根评论 reply_count
     OutboxPublisher 写 comment.created，payload 带 postAuthorId、root/parent 楼层和作者事实
-> 提交后失效列表 / 回复 / 首页缓存
```

评论整体必须至少包含文本、图片或语音中的一项。图片最多 9 张。语音不能和图片同时存在。

## 查询流程

- 顶级评论列表只返回未删除根评论。
- 回复列表按根评论 `floor` 展开，不依赖“根评论下第几条回复”的序号。
- 评论详情返回当前评论 `floor`；如果是回复，同时返回 `rootFloor` 和 `parentFloor`，供前端展开根评论、定位父评论和高亮目标回复。
- 评论列表展示作者摘要时，优先批量调用 User contract 或使用本地快照；不得直接读取 User 数据库。

## 分页和排序

- 传统分页用于 Web 固定页码场景，Go-first API 默认 `page` 从 `1` 开始，`size` 必须有上限。
- 游标分页用于移动端无限滚动，cursor 对外不透明。
- 增量补拉第一阶段优先使用 `floor` 锚点；`floor` 已经是同一文章内单调递增创建序号，避免同时暴露 `createdAt` 和 `floor` 两套排序锚点。
- 顶级评论 `TIME` 排序固定为 `floor DESC`。`createdAt` 是展示和审计字段，不作为第一阶段排序锚点。
- 回复列表排序固定为 `floor ASC`，保证对话从早到晚展开。
- 顶级评论 `HOT` 排序固定为 `like_count DESC, floor ASC`。`like_count` 来自 `comment_hot_rank` 读模型；同点赞数下优先展示更早楼层。
- HOT 游标如果后续补齐，必须编码 `likeCount + floor` 两个锚点；如果未来再加入 `createdAt`、时间衰减或其他排序列，cursor 必须同步包含所有排序锚点，不能只编码 `likeCount + id`。

## 管理和媒体边界

- Admin 删除可以从 Admin facade 进入，但最终 mutation 属于 Comment。Comment 只保存 `deletedBy`、`deleteReason`、`deletedAt` 等执行删除所需的最小元数据。
- 评论媒体上传 API 如果保留，只是 Upload / File Service adapter facade；新前端或新 API 可以直接调用 Upload 服务。
