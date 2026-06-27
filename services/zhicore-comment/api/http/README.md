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
- Comment 本地持久化保存 Content 公开 `postId` 字符串，不保存 Content 私有数据库主键。

## 公共规则

- 响应 envelope：见 `docs/contracts/http.md`。
- 错误码：见 `docs/contracts/error-codes.md`。
- 时间、ID、枚举、空值和 JSON 字段：见 `docs/contracts/data-types.md`。
- 分页、排序和过滤：见 `docs/contracts/pagination.md`。
- 认证和身份 header：见 `docs/contracts/http.md` 与 `docs/architecture/security.md`。
- 运行期 timeout、retry、熔断、降级和观测：见 `docs/architecture/runtime-operations.md` 与 `docs/architecture/observability.md`。

## 实现前置约束

Comment 公开错误码必须使用 `5001`、`5003` 等服务级业务码作为响应 body `code`。当前 `libs/kit/httpapi.WriteError(w, status, message)` 会把 HTTP status 写入 body `code`，不能直接用于 Comment handler。实现首个 handler 前必须先补一个可显式传入业务错误码的 shared writer，或在 Comment HTTP 层做局部封装，并用 contract test 覆盖 HTTP status 与 body `code` 不同的场景。

## 鉴权上下文

| 鉴权类型 | Header | 说明 |
| --- | --- | --- |
| 匿名 | 无需 `X-User-Id` | 只能读取公开可见评论。 |
| 登录用户 | `X-User-Id` 必填 | Gateway 注入 User 内部 `UserID`；用于创建评论、回复、点赞、取消点赞和查询 viewer 点赞状态。 |
| 作者 | `X-User-Id` 必填 | application 校验评论作者。 |
| 管理员 | `X-User-Id` + `X-User-Roles` | 管理端路由需要管理员角色；删除仍委托 Comment application。 |

客户端伪造的 `X-User-*` header 必须由 Gateway 清理后重新注入。Comment handler 不从 request body 接收当前操作者 `userId`；HTTP 作者摘要只返回 User `publicId`。

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
| `imageFileIds` | string[] | 否 | 图片文件引用，最多 9 个；空列表可省略。 |
| `imageUrls` | string[] | 否 | 图片展示 URL；运行时派生字段，不作为 Comment 持久化事实。 |
| `voiceFileId` | string | 否 | 语音文件引用。 |
| `voiceUrl` | string | 否 | 语音播放 URL；运行时派生字段，不作为 Comment 持久化事实。 |
| `voiceDuration` | int | 否 | 语音时长秒数。 |
| `status` | string | 是 | `NORMAL`、`DELETED`。 |
| `stats` | object | 是 | `likeCount`、`replyCount`。 |
| `viewer` | object | 否 | 登录用户视角：`liked`。匿名可省略。 |
| `editedAt` | string | 否 | RFC3339；评论被作者成功编辑后返回，未编辑时省略。 |
| `createdAt` | string | 是 | RFC3339。 |
| `updatedAt` | string | 是 | RFC3339。 |

`imageFileIds` 和 `voiceFileId` 是 Comment 持久化和写入 contract 的媒体事实。`imageUrls` 和 `voiceUrl` 只在读取响应中作为展示派生字段出现，不能作为创建 / 更新评论的长期事实输入。

### `AuthorSummary`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `publicId` | string | 是 | User 对外公开用户 ID。 |
| `displayName` | string | 否 | 展示名。 |
| `avatarFileId` | string | 否 | Upload 文件引用。 |
| `avatarUrl` | string | 否 | 可展示头像 URL。 |
| `unavailable` | boolean | 否 | User 摘要解析失败时可返回 `true`，前端据此展示占位。 |

### `Page<T>`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `items` | array | 是 | 当前页。 |
| `page` | int | 是 | 从 `1` 开始。 |
| `size` | int | 是 | 每页大小。 |
| `total` | int64 | 是 | 总数。 |
| `pages` | int | 是 | 总页数。 |

### `TopLevelCommentPage`

顶级评论列表不直接使用通用 `Page<T>.total`，因为文章总评论数和顶级评论分页总数语义不同。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `items` | `CommentItem[]` | 是 | 当前页顶级评论。 |
| `page` | int | 是 | 从 `1` 开始。 |
| `size` | int | 是 | 每页大小。 |
| `totalComments` | int64 | 是 | 文章下全部未删除评论数，包含根评论和回复。 |
| `totalTopLevelComments` | int64 | 是 | 文章下未删除根评论数，不包含回复。 |
| `pages` | int | 是 | 按 `totalTopLevelComments` 计算的总页数。 |

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

| code | HTTP status | message | 适用场景 |
| --- | --- | --- | --- |
| `1001` | `400` | `Invalid request` | path、query、body 字段非法，分页参数非法，排序枚举非法。 |
| `1004` | `503` | `Service unavailable` | Content、User、Upload、Ranking、PostgreSQL 或 Redis 不可用。 |
| `2006` | `401` | `Authentication required` | 登录态 endpoint 缺少 Gateway 注入身份。 |
| `2007` | `403` | `Admin role required` | 管理端路由缺少管理员角色。 |
| `2008` | `403` | `Forbidden` | 非作者更新或删除评论。 |
| `4001` | `404` | `Post not found` | 文章不存在、不可见或不可评论。 |
| `5001` | `404` | `Comment not found` | 评论、目标楼层或已删除评论对普通用户不可见。 |
| `5003` | `400` | `Comment content is required` | 文本、图片、语音整体为空。 |
| `5004` | `400` | `Comment content is too long` | 文本超过 2000 字。 |
| `5005` | `404` | `Root comment not found` | 回复目标根评论不存在或已删除。 |
| `5006` | `404` | `Parent comment not found` | 被回复评论不存在或已删除。 |
| `5007` / `5008` | `409` | `Like state conflict` | 保留给状态机冲突；Go-first `POST` / `DELETE` 默认幂等。 |

## 测试要求

- 每个 endpoint 实现前必须补 handler contract test，覆盖 path、method、query/body、鉴权 header、envelope 和错误码。
- 创建评论必须覆盖缺少 `X-User-Id`、空内容、文本过长、图片文件引用超过 9 张、图片文件引用格式、语音文件引用格式、语音和图片互斥、Content 校验失败和成功创建。
- 列表 endpoint 必须覆盖默认值、上限、非法参数、排序稳定性、空列表、作者摘要批量查询、匿名省略 `viewer` 和登录用户默认返回 `viewer.liked`。
- 仅更新本文档和 endpoint schema 时运行 `bash scripts/check-structure.sh` 与 `git diff --check`。
