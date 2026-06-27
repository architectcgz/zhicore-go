# zhicore-comment HTTP Schema

本目录记录 `zhicore-comment` 的 Go-first HTTP contract。Go handler、contract test、typed client 和 Gateway 路由必须以这里记录的字段级 schema 为准。

## 来源

- 服务总览：`docs/architecture/services/comment/README.md`
- 模块设计：`docs/architecture/module/comment/README.md`
- 模块 API 设计：`docs/architecture/module/comment/api.md`
- 模块 service 设计：`docs/architecture/module/comment/service.md`
- Go handler：`services/zhicore-comment/api/http/...`
- Go contract test：待补。
- Java 参考：`../zhicore-microservice/zhicore-comment/src/main/java/com/zhicore/comment/interfaces/controller/`，仅用于核对既有能力，不作为 Go path / 字段事实源。

## 定位

Comment API 是 Go-first 设计，不沿用旧 Java `commentId` path / DTO 作为外部 contract。外部评论定位使用 `(postId, floor)`：

- `postId` 是 Content 对外文章 ID。
- `floor` 是文章内单调递增楼层号，根评论和回复共享同一序列。
- HTTP path 不暴露内部 `comments.id`。

## 公共规则

- 响应 envelope：见 `docs/contracts/http.md`。
- 错误码：见 `docs/contracts/error-codes.md`。
- 时间、ID、枚举、空值和 JSON 字段：见 `docs/contracts/data-types.md`。
- 分页、排序和过滤：见 `docs/contracts/pagination.md`。
- 认证和身份 header：见 `docs/contracts/http.md` 与 `docs/architecture/security.md`。
- 运行期 timeout、retry、熔断、降级和观测：见 `docs/architecture/runtime-operations.md` 与 `docs/architecture/observability.md`。

## 鉴权上下文

| 鉴权类型 | Header | 说明 |
| --- | --- | --- |
| 匿名 | 无需 `X-User-Id` | 只能读取公开可见评论。 |
| 登录用户 | `X-User-Id` 必填 | 创建评论、回复、点赞、取消点赞和查询 viewer 点赞状态。 |
| 作者 | `X-User-Id` 必填 | application 校验评论作者。 |
| 管理员 | `X-User-Id` + `X-User-Roles` | 管理端路由需要管理员角色；删除仍委托 Comment application。 |

客户端伪造的 `X-User-*` header 必须由 Gateway 清理后重新注入。Comment handler 不从 request body 接收当前操作者 `userId`。

## 通用对象

### `CommentItem`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 对外文章 ID。 |
| `floor` | int64 | 是 | 文章内楼层号，对外评论定位字段。 |
| `rootFloor` | int64 | 否 | 回复所属根评论楼层；根评论省略。 |
| `parentFloor` | int64 | 否 | 直接被回复评论楼层；根评论省略。 |
| `author` | object | 是 | 作者摘要，字段见 `AuthorSummary`。 |
| `content` | string | 否 | 评论文本；没有文本时可省略。 |
| `imageUrls` | string[] | 否 | 图片 CDN / 可展示 URL，最多 9 个；空列表可省略。 |
| `voiceUrl` | string | 否 | 语音 CDN / 可播放 URL。 |
| `voiceDuration` | int | 否 | 语音时长秒数。 |
| `status` | string | 是 | `NORMAL`、`DELETED`。 |
| `stats` | object | 是 | `likeCount`、`replyCount`。 |
| `viewer` | object | 否 | 登录用户视角：`liked`。匿名可省略。 |
| `createdAt` | string | 是 | RFC3339。 |
| `updatedAt` | string | 是 | RFC3339。 |

`imageUrls` 和 `voiceUrl` 必须是可展示或可播放的绝对 `http` / `https` 地址，通常为 Upload / File Service 返回的 CDN URL；Comment 不接受媒体文件 ID、对象存储 key 或相对路径。

### `AuthorSummary`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `userId` | string | 是 | User 对外用户 ID。 |
| `displayName` | string | 否 | 展示名。 |
| `avatarFileId` | string | 否 | Upload 文件引用。 |
| `avatarUrl` | string | 否 | 可展示头像 URL。 |

### `Page<T>`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `items` | array | 是 | 当前页。 |
| `page` | int | 是 | 从 `1` 开始。 |
| `size` | int | 是 | 每页大小。 |
| `total` | int64 | 是 | 总数。 |
| `pages` | int | 是 | 总页数。 |

## Endpoint 索引

### 首批 contract

| 方法 | 路径 | 文档 | 状态 |
| --- | --- | --- | --- |
| `POST` | `/api/v1/posts/{postId}/comments` | `endpoints/create-comment.md` | 草案 |
| `GET` | `/api/v1/posts/{postId}/comments/page` | `endpoints/list-comments-page.md` | 草案 |

### 待提取 contract

| 方法 | 路径 | 状态 |
| --- | --- | --- |
| `GET` | `/api/v1/posts/{postId}/comments/cursor` | API 族已识别 |
| `GET` | `/api/v1/posts/{postId}/comments/incremental` | API 族已识别 |
| `GET` | `/api/v1/posts/{postId}/comments/{floor}` | API 族已识别 |
| `PUT` | `/api/v1/posts/{postId}/comments/{floor}` | API 族已识别 |
| `DELETE` | `/api/v1/posts/{postId}/comments/{floor}` | API 族已识别 |
| `GET` | `/api/v1/posts/{postId}/comments/{floor}/replies/page` | API 族已识别 |
| `GET` | `/api/v1/posts/{postId}/comments/{floor}/replies/cursor` | API 族已识别 |
| `GET` | `/api/v1/posts/{postId}/comments/{floor}/replies/incremental` | API 族已识别 |
| `POST` | `/api/v1/posts/{postId}/comments/{floor}/like` | API 族已识别 |
| `DELETE` | `/api/v1/posts/{postId}/comments/{floor}/like` | API 族已识别 |
| `GET` | `/api/v1/posts/{postId}/comments/{floor}/liked` | API 族已识别 |
| `GET` | `/api/v1/posts/{postId}/comments/{floor}/like-count` | API 族已识别 |
| `POST` | `/api/v1/posts/{postId}/comments/batch/liked` | API 族已识别 |
| `POST` | `/api/v1/comments/media/images` | 待定 |
| `POST` | `/api/v1/comments/media/voice` | 待定 |
| `GET` | `/api/v1/admin/comments` | API 族已识别 |
| `DELETE` | `/api/v1/admin/comments/posts/{postId}/comments/{floor}` | API 族已识别 |
| `GET` | `/api/v1/admin/comment-outbox/summary` | API 族已识别 |
| `POST` | `/api/v1/admin/comment-outbox/retry-dead` | API 族已识别 |

## API 到设计追踪

| Endpoint | Use case | 设计文档 | Contract 状态 | 测试状态 |
| --- | --- | --- | --- | --- |
| `POST /api/v1/posts/{postId}/comments` | `CreateComment` / `CreateReply` | `docs/architecture/module/comment/service.md` | 草案 | 待补 |
| `GET /api/v1/posts/{postId}/comments/page` | `ListTopLevelCommentsByPage` | `docs/architecture/module/comment/service.md` | 草案 | 待补 |

## 服务级公开错误码

| code | HTTP status | 含义 | 适用场景 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | path、query、body 字段非法，分页参数非法，排序枚举非法。 |
| `1004` | `503` | 服务暂时不可用 | Content、User、Upload、Ranking、PostgreSQL 或 Redis 不可用。 |
| `2006` | `401` | 请先登录 | 登录态 endpoint 缺少 Gateway 注入身份。 |
| `2007` | `403` | 需要特定角色 | 管理端路由缺少管理员角色。 |
| `2008` | `403` | 无权访问该资源 | 非作者更新或删除评论。 |
| `4001` | `404` | 文章不存在 | 文章不存在、不可见或不可评论。 |
| `5001` | `404` | 评论不存在 | 评论或目标楼层不存在。 |
| `5002` | `409` | 评论已删除 | 操作已删除评论。 |
| `5003` | `400` | 评论内容不能为空 | 文本、图片、语音整体为空。 |
| `5004` | `400` | 评论内容过长 | 文本超过 2000 字。 |
| `5005` | `404` | 根评论不存在 | 回复目标根评论不存在。 |
| `5006` | `404` | 被回复的评论不存在 | 被回复评论不存在。 |
| `5007` / `5008` | `409` | 点赞状态冲突 | 保留给状态机冲突；Go-first `POST` / `DELETE` 默认幂等。 |

## 测试要求

- 每个 endpoint 实现前必须补 handler contract test，覆盖 path、method、query/body、鉴权 header、envelope 和错误码。
- 创建评论必须覆盖缺少 `X-User-Id`、空内容、文本过长、图片 URL 超过 9 张、图片 URL 格式、语音 URL 格式、语音和图片互斥、Content 校验失败和成功创建。
- 列表 endpoint 必须覆盖默认值、上限、非法参数、排序稳定性、空列表和作者摘要批量查询。
- 仅更新本文档和 endpoint schema 时运行 `bash scripts/check-structure.sh` 与 `git diff --check`。
