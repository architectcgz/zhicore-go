# Comment 模块架构

`comment` 模块对应 `zhicore-comment` 服务内的评论上下文。Go 实现按 `api/http -> application -> domain/ports -> infrastructure` 的依赖方向落点；本文档描述目标设计，不表示当前 Go 代码已经完成。

## 模块职责

- 管理评论、根评论、回复、评论树、评论状态和删除元数据。
- 管理评论文本、图片展示 URL、语音播放 URL 和语音时长。
- 管理评论点赞关系、点赞数、回复数和统计修正。
- 提供评论详情、文章评论分页、回复分页、游标分页、增量补拉和管理端查询。
- 生产 `comment.created`、`comment.deleted`、`comment.liked`、`comment.unliked` 集成事件。

## 边界

Comment 不把 Content、User、Upload、Ranking 或 Notification 的模型引入领域层：

- 文章存在性、可见性和是否允许评论通过 Content contract 校验。
- 作者摘要、用户状态、拉黑关系和互动权限通过 User contract 校验。
- 文件事实归 Upload / File Service，Comment 图片和语音只保存可展示 / 可播放的 CDN URL，不保存媒体文件 ID。
- Ranking 拥有榜单和分数，Comment 只读取热门候选作为本地缓存输入。
- Notification 拥有通知读模型，Comment 只发布评论事件。

## 子域

| 子域 | 职责 | 主要存储 |
| --- | --- | --- |
| Comment Tree | 创建根评论、创建回复、维护 `rootId` / `parentId`、删除评论和评论状态 | `comments` |
| Comment Content / Media | 评论文本、图片展示 URL、语音播放 URL、语音时长和内容校验 | `comments.image_urls`、`comments.voice_url`、`comments.voice_duration` |
| Comment Interaction | 点赞、取消点赞和用户点赞状态 | `comment_likes` |
| Comment Stats | 评论点赞数、回复数和统计修正 | `comment_stats` |
| Comment Query | 详情、分页、游标分页、增量补拉和管理端查询 | `comments`、`comment_stats`、Redis cache |
| Homepage / Hot Candidate | 首页评论缓存和热门文章候选输入 | Redis cache、Ranking contract |
| Integration | 评论事件、outbox 派发、dead retry、缓存失效和幂等 | Comment `outbox_events` |

## API Family

Comment 是 Go-first API reset。外部评论定位使用 `(postId, floor)`，不暴露内部 `comments.id`：

- `POST /api/v1/posts/{postId}/comments`：创建根评论或回复。
- `GET /api/v1/posts/{postId}/comments/page`：文章顶级评论传统分页。
- `GET /api/v1/posts/{postId}/comments/cursor`：文章顶级评论游标分页。
- `GET /api/v1/posts/{postId}/comments/incremental`：文章顶级评论增量补拉。
- `GET /api/v1/posts/{postId}/comments/{floor}`：评论详情。
- `PUT /api/v1/posts/{postId}/comments/{floor}`：更新评论。
- `DELETE /api/v1/posts/{postId}/comments/{floor}`：删除评论。
- `GET /api/v1/posts/{postId}/comments/{floor}/replies/page`：回复传统分页。
- `GET /api/v1/posts/{postId}/comments/{floor}/replies/cursor`：回复游标分页。
- `GET /api/v1/posts/{postId}/comments/{floor}/replies/incremental`：回复增量补拉。
- `POST /api/v1/posts/{postId}/comments/{floor}/like`：点赞。
- `DELETE /api/v1/posts/{postId}/comments/{floor}/like`：取消点赞。
- `GET /api/v1/posts/{postId}/comments/{floor}/liked`：点赞状态。
- `GET /api/v1/posts/{postId}/comments/{floor}/like-count`：点赞数。
- `POST /api/v1/posts/{postId}/comments/batch/liked`：批量点赞状态。
- `/api/v1/comments/media/images`、`/api/v1/comments/media/voice`：可选媒体上传 facade。
- `/api/v1/admin/comments`、`/api/v1/admin/comment-outbox/*`：管理端和 outbox 运维。

## 文档拆分

| 文档 | 内容 |
| --- | --- |
| `api.md` | API 背后的业务流程、权限、状态机、副作用和 use case 追踪。 |
| `service.md` | Application service、事务边界、幂等、错误映射、缓存失效和实现切片。 |
| `domain.md` | 聚合、实体、值对象、不变量、领域服务和工厂。 |
| `ports.md` | repository、cache、client、event publisher、outbox 和 external adapter 端口归属。 |
| `data-events.md` | 数据归属、目标 schema 草案、缓存 key、事件 payload 和跨服务一致性。 |

## 当前状态

- 已固定：模块边界、DDD 聚合拆分、外部定位方式、首批 API contract、首个实现切片。
- 待实现：Go handler、domain/application/infrastructure、migration、contract test 和 system HTTP test。
- 首批 contract：`services/zhicore-comment/api/http/README.md`、`services/zhicore-comment/api/http/endpoints/create-comment.md`、`services/zhicore-comment/api/http/endpoints/list-comments-page.md`。
