# Comment 模块架构

`comment` 模块对应 `zhicore-comment` 服务内的评论上下文。Go 实现按 `api/http -> application -> domain/ports -> infrastructure` 的依赖方向落点；本文档描述目标设计，不表示当前 Go 代码已经完成。

## 模块职责

- 管理评论、根评论、回复、评论树、评论状态和删除元数据。
- 管理评论文本、图片文件引用、语音文件引用、语音时长和编辑标记。
- 管理评论点赞关系、回复数、点赞数读模型、文章级评论统计、点赞计数 delta 和统计修正。
- 提供评论详情、文章评论分页、回复分页、游标分页、增量补拉、点赞状态和管理端查询。
- 生产 `comment.created`、`comment.deleted`、`comment.liked`、`comment.unliked` 集成事件。

## 边界

Comment 不把 Content、User、File service、Ranking 或 Notification 的模型引入领域层：

- 文章存在性、可见性和是否允许评论通过 Content contract 校验。
- Gateway 注入的 `X-User-Id` 使用 User 内部 `UserID`；Comment 内部持久化使用该内部 ID，对外 HTTP 作者摘要使用 User `publicId`。
- 作者摘要、用户状态、拉黑关系和互动权限通过 User contract 校验；查询路径可降级展示占位作者，写路径不能降级放行。
- 文件事实归 File service，Comment 图片和语音只保存系统内媒体文件 ID；创建 / 更新时校验文件引用，查询时批量解析或缓存派生可展示 / 可播放 URL。
- Ranking 拥有榜单和分数，Comment 只读取热门候选作为本地缓存输入。
- Notification 拥有通知读模型，Comment 只发布评论事件。

## 子域

| 子域 | 职责 | 主要存储 |
| --- | --- | --- |
| Comment Tree | 创建根评论、创建回复、维护 `rootId` / `parentId`、删除评论和评论状态 | `comments` |
| Comment Content / Media | 评论文本、图片文件引用、语音文件引用、语音时长和内容校验 | `comments.image_file_ids`、`comments.voice_file_id`、`comments.voice_duration` |
| Comment Interaction | 点赞、取消点赞和用户点赞状态 | `comment_likes` |
| Comment Stats | 根评论回复总数、点赞数读模型、文章级评论总数和统计修正；点赞数由 delta worker 批量更新 | `comment_stats`、`comment_post_stats`、`comment_counter_deltas` |
| Comment Hot Rank | 顶级评论 HOT 排序读模型，避免高频查询大范围 join 排序 | `comment_hot_rank` |
| Comment Recommended Rank | 默认评论流排序读模型，按点赞数和新鲜度加权生成 `RECOMMENDED` 排序 | `comment_recommended_rank` |
| Comment Query | 详情、分页、游标分页、增量补拉和管理端查询 | `comments`、`comment_stats`、`comment_post_stats`、rank 表、Redis cache |
| Homepage / Hot Candidate | 首页评论缓存和热门文章候选输入 | Redis cache、Ranking contract |
| Integration | 评论事件、outbox 派发、dead retry、缓存失效和幂等 | Comment `outbox_events` |

## API Family

Comment 是 Go-first API reset。外部评论定位使用 `(postId, commentId)`：

- `postId` 是 Content 公开文章 ID 字符串，Comment 本地保存该字符串用于 HTTP 定位和查询分区；同时保存 Content 内部 `post_id BIGINT` opaque reference，用于跨服务事件下游直接落账。
- `commentId` 是 Comment 对外评论 ID，由内部 `comments.id BIGINT IDENTITY` 派生；HTTP 不直接暴露内部自增数字。
- 评论 ID 和排序锚点设计见 `comment-id.md`；第一版不保留文章内 `floor`。
- `POST /api/v1/posts/{postId}/comments`：创建根评论或回复。
- `GET /api/v1/posts/{postId}/comments/page`：文章顶级评论传统分页。
- `GET /api/v1/posts/{postId}/comments/cursor`：文章顶级评论游标分页。
- `GET /api/v1/posts/{postId}/comments/incremental`：文章顶级评论增量补拉。
- `GET /api/v1/posts/{postId}/comments/{commentId}`：评论详情。
- `PUT /api/v1/posts/{postId}/comments/{commentId}`：更新评论。
- `DELETE /api/v1/posts/{postId}/comments/{commentId}`：删除评论。
- `GET /api/v1/posts/{postId}/comments/{commentId}/replies/page`：回复传统分页。
- `GET /api/v1/posts/{postId}/comments/{commentId}/replies/cursor`：回复游标分页。
- `GET /api/v1/posts/{postId}/comments/{commentId}/replies/incremental`：回复增量补拉。
- `POST /api/v1/posts/{postId}/comments/{commentId}/like`：点赞。
- `DELETE /api/v1/posts/{postId}/comments/{commentId}/like`：取消点赞。
- `GET /api/v1/posts/{postId}/comments/{commentId}/liked`：点赞状态。
- `GET /api/v1/posts/{postId}/comments/{commentId}/like-count`：点赞数。
- `POST /api/v1/posts/{postId}/comments/batch/liked`：批量点赞状态。
- `/api/v1/admin/comments`、`/api/v1/admin/comment-outbox/*`：管理端和 outbox 运维。

顶级评论默认排序为 `RECOMMENDED`，另支持严格 `HOT` 和 `TIME`：

- `RECOMMENDED`：默认评论流，按 `recommended_score DESC, comment_id DESC`。
- `HOT`：严格热门，按 `like_count DESC, comment_id ASC`。
- `TIME`：最新，按 `comment_id DESC`。

回复列表默认 `HOT`，按 `like_count DESC, comment_id ASC` 平铺返回根评论下整棵回复子树；可选 `TIME` 使用 `comment_id ASC`。

Comment 不提供媒体上传 facade。前端先调用 File service 获得文件 ID，再把 `imageFileIds` / `voiceFileId` 传给 Comment。

## 文档拆分

| 文档 | 内容 |
| --- | --- |
| `api.md` | API 背后的业务流程、权限、状态机、副作用和 use case 追踪。 |
| `service.md` | Application service、事务边界、幂等、错误映射、缓存失效和实现切片。 |
| `domain.md` | 聚合、实体、值对象、不变量、领域服务和工厂。 |
| `ports.md` | repository、cache、client、event publisher、outbox 和 external adapter 端口归属。 |
| `data-events.md` | 数据归属、目标 schema 草案、缓存 key、事件 payload 和跨服务一致性。 |
| `comment-id.md` | 评论内部 ID、对外 `commentId`、排序锚点和 cursor 锚点。 |
| `runtime-resilience.md` | timeout、retry、熔断、降级、限流、健康检查和依赖故障语义。 |
| `decision-log.md` | 设计压测中已确认的决策、原因和后续依赖。 |

## 当前状态

- 已固定：模块边界、DDD 聚合拆分、外部定位方式、用户标识边界、评论计数语义、删除子树语义、默认排序、首批 API contract、运行韧性策略和首个实现切片。
- 待实现：Go handler、domain/application/infrastructure、migration、contract test 和 system HTTP test。
- 首批 contract：`services/zhicore-comment/api/http/README.md`、`services/zhicore-comment/api/http/endpoints/create-comment.md`、`services/zhicore-comment/api/http/endpoints/list-comments-page.md`；首个切片必须同时支持根评论和 `parentCommentId` 回复创建。
