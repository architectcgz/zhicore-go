# zhicore-content HTTP Schema

本目录记录 `zhicore-content` 的 Go-first HTTP contract。Go handler、contract test、typed client 和 Gateway 路由必须以这里记录的 schema 为准。

## 定位

Content API 是 Go-first 设计，不沿用旧 Java path / DTO 作为约束。Java 只作为业务能力参考，用来确认“有哪些文章、草稿、标签、互动、管理端能力”，不作为 Go 对外 contract 的 path、字段或分页形态约束。

本目录是 Content 对外 HTTP 事实源；字段级总览见 [endpoints/content-api.md](endpoints/content-api.md)，进入实现切片的 endpoint 会拆成单独 schema。

## 设计原则

- 文章、草稿、正文、标签、点赞、收藏、文章统计、作者快照和 Content outbox 归 `zhicore-content`。
- 不提供 User facade。用户发表文章列表直接调用 Content，例如 `GET /api/v1/posts?authorId={authorId}&limit=20`。
- Gateway 负责 JWT 校验和分流；Content 只消费可信身份 header，不解析客户端 `Authorization`。
- 外部文章 ID 使用 string `postId`，语义为 Content 的公开 ID，不暴露数据库内部自增主键。
- 正文使用 versioned blocks schema。HTTP 不再接受 `contentType=html/rich` 作为可信正文形态，也不把 raw HTML 作为正文事实。
- 公开列表默认 cursor 分页；管理端和低频维护列表使用 page 分页。
- Search、Ranking、Notification、Comment 等服务通过 Content typed client contract 调用，不复制 Content DTO。

## 公共规则

- 响应 envelope：见 `docs/contracts/http.md`。
- 错误码：见 `docs/contracts/error-codes.md`。
- 时间、ID、枚举、空值和 JSON 字段：见 `docs/contracts/data-types.md`。
- JSON request body 上限：`512KB`；超过后由 HTTP 层返回 HTTP `413`、body `code=4015`，不进入 application parser。
- 分页、排序和过滤：见 `docs/contracts/pagination.md`。
- 认证和身份 header：见 `docs/contracts/http.md` 与 `docs/architecture/security.md`。
- 运行期 timeout、retry、熔断、降级和观测：见 `docs/architecture/runtime-operations.md`、`docs/architecture/observability.md` 和 `docs/architecture/services/content/runtime-resilience.md`。
- 限流和 Redis 故障策略：见 `docs/architecture/services/content/rate-limiting.md`。
- 互动状态和产品降级语义：见 `docs/architecture/services/content/engagement-design.md`。

## 限流上下文

Content API 需要 Gateway 粗限流和 Content 服务内业务限流两层保护。Gateway 按 IP、route、method 阻挡明显洪水流量；Content 按 actor、post、service caller、operation 和高成本资源维度保护草稿保存、发布、正文读取、互动统计、管理端和内部调用。

业务限流命中时返回 HTTP `429`，body `code` 使用 `1003`。Redis 或 limiter 依赖不可用导致高副作用写路径不能确认配额时，返回 HTTP `503`，body `code` 使用 `1004`。

Engagement 读路径中，当前用户点赞 / 收藏状态不可确认时不把 unknown 伪装成 `false`。详情页和批量状态使用 `liked=null`、`favorited=null` 和 `degraded=true` 表示状态不可确认；命令接口仍必须返回确定成功或错误。

## 鉴权上下文

| 鉴权类型 | Header | 说明 |
| --- | --- | --- |
| 匿名 | 无需 `X-User-Id` | 只能读取公开发布内容。 |
| 登录用户 | `X-User-Id` 必填 | 创建草稿、保存草稿、点赞、收藏、读取我的草稿等。 |
| 作者 | `X-User-Id` 必填 | application 校验 `posts.owner_id == Actor.UserID`。 |
| 管理员 | `X-User-Id` + `X-User-Roles` | 管理端路由需要管理员角色；状态校验仍由 Content application 执行。 |
| 服务间调用 | `X-Caller-Service` + `X-Caller-Operation` | Content typed client 调用必填，用于内部调用限流、审计和观测；不表示当前用户身份。 |

客户端伪造的 `X-User-*` header 必须由 Gateway 清理后重新注入。Content handler 不从 request body 接收当前操作者 `userId`。
客户端伪造的 `X-Caller-*` header 必须由 Gateway 清理；服务间 typed client adapter 从服务配置和调用点常量生成 caller header。

## Endpoint 索引

### 公开文章查询

| 方法 | 路径 | 鉴权 | 用途 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/posts` | 匿名 | 公开文章列表；字段级 schema 见 [endpoints/list-posts.md](endpoints/list-posts.md)。 |
| `GET` | `/api/v1/posts/{postId}` | 匿名 | 文章详情元数据和可展示正文；字段级 schema 见 [endpoints/get-post-detail.md](endpoints/get-post-detail.md)。 |
| `GET` | `/api/v1/posts/{postId}/body` | 匿名 / 服务间 | 读取 published body，供详情页和 Search 使用。字段级 schema 见 [endpoints/get-post-body.md](endpoints/get-post-body.md)。 |
| `POST` | `/api/v1/posts/batch-get` | 匿名 / 服务间 | 批量获取文章摘要；字段级 schema 见 [endpoints/batch-get-posts.md](endpoints/batch-get-posts.md)。 |
| `GET` | `/api/v1/posts/{postId}/tags` | 匿名 | 文章标签列表；字段级 schema 见 [endpoints/get-post-tags.md](endpoints/get-post-tags.md)。 |

### 作者工作台

| 方法 | 路径 | 鉴权 | 用途 |
| --- | --- | --- | --- |
| `POST` | `/api/v1/posts` | 登录用户 | 创建文章草稿。字段级 schema 见 [endpoints/create-post.md](endpoints/create-post.md)。 |
| `GET` | `/api/v1/me/posts` | 登录用户 | 我的文章列表，含 draft / published / scheduled / deleted；字段级 schema 见 [endpoints/list-my-posts.md](endpoints/list-my-posts.md)。 |
| `GET` | `/api/v1/me/drafts` | 登录用户 | 我的草稿列表；字段级 schema 见 [endpoints/list-my-drafts.md](endpoints/list-my-drafts.md)。 |
| `GET` | `/api/v1/posts/{postId}/draft` | 作者 | 读取当前草稿；字段级 schema 见 [endpoints/get-post-draft.md](endpoints/get-post-draft.md)。 |
| `PATCH` | `/api/v1/posts/{postId}/draft/meta` | 作者 | 更新草稿元数据；字段级 schema 见 [endpoints/update-draft-meta.md](endpoints/update-draft-meta.md)。 |
| `PUT` | `/api/v1/posts/{postId}/draft/body` | 作者 | 保存草稿正文 blocks。字段级 schema 见 [endpoints/save-draft-body.md](endpoints/save-draft-body.md)。 |
| `DELETE` | `/api/v1/posts/{postId}/draft` | 作者 | 删除草稿指针并创建正文清理任务；字段级 schema 见 [endpoints/delete-post-draft.md](endpoints/delete-post-draft.md)。 |
| `POST` | `/api/v1/posts/{postId}/publish` | 作者 | 发布草稿。字段级 schema 见 [endpoints/publish-post.md](endpoints/publish-post.md)。 |
| `POST` | `/api/v1/posts/{postId}/unpublish` | 作者 | 撤回已发布文章；字段级 schema 见 [endpoints/unpublish-post.md](endpoints/unpublish-post.md)。 |
| `POST` | `/api/v1/posts/{postId}/schedule` | 作者 | 创建或更新定时发布；字段级 schema 见 [endpoints/schedule-post.md](endpoints/schedule-post.md)。 |
| `DELETE` | `/api/v1/posts/{postId}/schedule` | 作者 | 取消定时发布；字段级 schema 见 [endpoints/cancel-post-schedule.md](endpoints/cancel-post-schedule.md)。 |
| `DELETE` | `/api/v1/posts/{postId}` | 作者 | 软删除文章；字段级 schema 见 [endpoints/delete-post.md](endpoints/delete-post.md)。 |
| `POST` | `/api/v1/posts/{postId}/restore` | 作者 | 恢复软删除文章；字段级 schema 见 [endpoints/restore-post.md](endpoints/restore-post.md)。 |
| `PUT` | `/api/v1/posts/{postId}/tags` | 作者 | 替换文章标签集合；字段级 schema 见 [endpoints/update-post-tags.md](endpoints/update-post-tags.md)。 |
| `DELETE` | `/api/v1/posts/{postId}/tags/{slug}` | 作者 | 删除单个文章标签；字段级 schema 见 [endpoints/delete-post-tag.md](endpoints/delete-post-tag.md)。 |

### 互动

| 方法 | 路径 | 鉴权 | 用途 |
| --- | --- | --- | --- |
| `PUT` | `/api/v1/posts/{postId}/like` | 登录用户 | 幂等点赞；字段级 schema 见 [endpoints/like-post.md](endpoints/like-post.md)。 |
| `DELETE` | `/api/v1/posts/{postId}/like` | 登录用户 | 幂等取消点赞；字段级 schema 见 [endpoints/unlike-post.md](endpoints/unlike-post.md)。 |
| `PUT` | `/api/v1/posts/{postId}/favorite` | 登录用户 | 幂等收藏；字段级 schema 见 [endpoints/favorite-post.md](endpoints/favorite-post.md)。 |
| `DELETE` | `/api/v1/posts/{postId}/favorite` | 登录用户 | 幂等取消收藏；字段级 schema 见 [endpoints/unfavorite-post.md](endpoints/unfavorite-post.md)。 |
| `GET` | `/api/v1/posts/{postId}/engagement` | 匿名 / 登录用户 | 互动计数和当前用户状态；字段级 schema 见 [endpoints/get-post-engagement.md](endpoints/get-post-engagement.md)。 |
| `POST` | `/api/v1/posts/engagement/batch-status` | 登录用户 | 批量查询点赞 / 收藏状态；字段级 schema 见 [endpoints/batch-get-engagement-status.md](endpoints/batch-get-engagement-status.md)。 |

### 标签

| 方法 | 路径 | 鉴权 | 用途 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/tags` | 匿名 | 标签列表；字段级 schema 见 [endpoints/list-tags.md](endpoints/list-tags.md)。 |
| `GET` | `/api/v1/tags/{slug}` | 匿名 | 标签详情；字段级 schema 见 [endpoints/get-tag.md](endpoints/get-tag.md)。 |
| `GET` | `/api/v1/tags/search` | 匿名 | 标签搜索 / 自动补全；字段级 schema 见 [endpoints/search-tags.md](endpoints/search-tags.md)。 |
| `GET` | `/api/v1/tags/hot` | 匿名 | 热门标签；字段级 schema 见 [endpoints/list-hot-tags.md](endpoints/list-hot-tags.md)。 |
| `GET` | `/api/v1/tags/{slug}/posts` | 匿名 | 标签下公开文章列表；字段级 schema 见 [endpoints/list-posts-by-tag.md](endpoints/list-posts-by-tag.md)。 |

### 管理和运维

| 方法 | 路径 | 鉴权 | 用途 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/admin/content/posts` | 管理员 | 管理端文章查询；字段级 schema 见 [endpoints/list-admin-posts.md](endpoints/list-admin-posts.md)。 |
| `DELETE` | `/api/v1/admin/content/posts/{postId}` | 管理员 | 管理端删除文章；字段级 schema 见 [endpoints/delete-admin-post.md](endpoints/delete-admin-post.md)。 |
| `GET` | `/api/v1/admin/content/outbox-events` | 管理员 | 查询 dead / failed outbox 事件。 |
| `POST` | `/api/v1/admin/content/outbox-events/{eventId}/retry` | 管理员 | 手动重试 outbox 事件。 |

## 已验证 endpoint

本节的“已验证”表示对应 endpoint 的路由、可信身份 header、请求 DTO、基础错误 envelope、成功响应和本切片指定的核心错误码已经由 handler contract test 覆盖。File / Cover 依赖的专用语义错误已经通过 application / ports sentinel 固定，不通过匹配下游错误文本实现。

| 方法 | 路径 | 文档 | Handler contract test | 状态 |
| --- | --- | --- | --- | --- |
| `POST` | `/api/v1/posts` | [endpoints/create-post.md](endpoints/create-post.md) | `services/zhicore-content/api/http/create_post_handler_test.go` | 已验证 |
| `GET` | `/api/v1/posts` | [endpoints/list-posts.md](endpoints/list-posts.md) | `services/zhicore-content/api/http/public_post_queries_handler_test.go` | 已验证 |
| `GET` | `/api/v1/posts/{postId}` | [endpoints/get-post-detail.md](endpoints/get-post-detail.md) | `services/zhicore-content/api/http/public_post_queries_handler_test.go` | 已验证 |
| `POST` | `/api/v1/posts/batch-get` | [endpoints/batch-get-posts.md](endpoints/batch-get-posts.md) | `services/zhicore-content/api/http/public_post_queries_handler_test.go` | 已验证 |
| `GET` | `/api/v1/me/posts` | [endpoints/list-my-posts.md](endpoints/list-my-posts.md) | `services/zhicore-content/api/http/author_workbench_handler_test.go` | 已验证 |
| `GET` | `/api/v1/me/drafts` | [endpoints/list-my-drafts.md](endpoints/list-my-drafts.md) | `services/zhicore-content/api/http/author_workbench_handler_test.go` | 已验证 |
| `GET` | `/api/v1/posts/{postId}/draft` | [endpoints/get-post-draft.md](endpoints/get-post-draft.md) | `services/zhicore-content/api/http/author_workbench_handler_test.go` | 已验证 |
| `PATCH` | `/api/v1/posts/{postId}/draft/meta` | [endpoints/update-draft-meta.md](endpoints/update-draft-meta.md) | `services/zhicore-content/api/http/author_workbench_handler_test.go` | 已验证 |
| `DELETE` | `/api/v1/posts/{postId}/draft` | [endpoints/delete-post-draft.md](endpoints/delete-post-draft.md) | `services/zhicore-content/api/http/author_workbench_handler_test.go` | 已验证 |
| `GET` | `/api/v1/admin/content/outbox-events` | [endpoints/list-admin-outbox-events.md](endpoints/list-admin-outbox-events.md) | `services/zhicore-content/api/http/admin_outbox_handler_test.go` | 已验证 |
| `POST` | `/api/v1/admin/content/outbox-events/{eventId}/retry` | [endpoints/retry-admin-outbox-event.md](endpoints/retry-admin-outbox-event.md) | `services/zhicore-content/api/http/admin_outbox_handler_test.go` | 已验证 |
| `PUT` | `/api/v1/posts/{postId}/draft/body` | [endpoints/save-draft-body.md](endpoints/save-draft-body.md) | `services/zhicore-content/api/http/save_draft_body_handler_test.go` | 已验证 |
| `POST` | `/api/v1/posts/{postId}/publish` | [endpoints/publish-post.md](endpoints/publish-post.md) | `services/zhicore-content/api/http/publish_post_handler_test.go` | 已验证 |
| `POST` | `/api/v1/posts/{postId}/unpublish` | [endpoints/unpublish-post.md](endpoints/unpublish-post.md) | `services/zhicore-content/api/http/post_lifecycle_handler_test.go` | 已验证 |
| `POST` | `/api/v1/posts/{postId}/schedule` | [endpoints/schedule-post.md](endpoints/schedule-post.md) | `services/zhicore-content/api/http/post_schedule_handler_test.go` | 已验证 |
| `DELETE` | `/api/v1/posts/{postId}/schedule` | [endpoints/cancel-post-schedule.md](endpoints/cancel-post-schedule.md) | `services/zhicore-content/api/http/post_schedule_handler_test.go` | 已验证 |
| `DELETE` | `/api/v1/posts/{postId}` | [endpoints/delete-post.md](endpoints/delete-post.md) | `services/zhicore-content/api/http/post_lifecycle_handler_test.go` | 已验证 |
| `POST` | `/api/v1/posts/{postId}/restore` | [endpoints/restore-post.md](endpoints/restore-post.md) | `services/zhicore-content/api/http/post_lifecycle_handler_test.go` | 已验证 |
| `GET` | `/api/v1/posts/{postId}/body` | [endpoints/get-post-body.md](endpoints/get-post-body.md) | `services/zhicore-content/api/http/get_post_body_handler_test.go` | 已验证 |
| `GET` | `/api/v1/tags` | [endpoints/list-tags.md](endpoints/list-tags.md) | `services/zhicore-content/api/http/taxonomy_handler_test.go` | 已验证 |
| `GET` | `/api/v1/tags/{slug}` | [endpoints/get-tag.md](endpoints/get-tag.md) | `services/zhicore-content/api/http/taxonomy_handler_test.go` | 已验证 |
| `GET` | `/api/v1/tags/search` | [endpoints/search-tags.md](endpoints/search-tags.md) | `services/zhicore-content/api/http/taxonomy_handler_test.go` | 已验证 |
| `GET` | `/api/v1/tags/hot` | [endpoints/list-hot-tags.md](endpoints/list-hot-tags.md) | `services/zhicore-content/api/http/taxonomy_handler_test.go` | 已验证 |
| `GET` | `/api/v1/tags/{slug}/posts` | [endpoints/list-posts-by-tag.md](endpoints/list-posts-by-tag.md) | `services/zhicore-content/api/http/taxonomy_handler_test.go` | 已验证 |
| `GET` | `/api/v1/posts/{postId}/tags` | [endpoints/get-post-tags.md](endpoints/get-post-tags.md) | `services/zhicore-content/api/http/taxonomy_handler_test.go` | 已验证 |
| `PUT` | `/api/v1/posts/{postId}/tags` | [endpoints/update-post-tags.md](endpoints/update-post-tags.md) | `services/zhicore-content/api/http/taxonomy_handler_test.go` | 已验证 |
| `DELETE` | `/api/v1/posts/{postId}/tags/{slug}` | [endpoints/delete-post-tag.md](endpoints/delete-post-tag.md) | `services/zhicore-content/api/http/taxonomy_handler_test.go` | 已验证 |
| `GET` | `/api/v1/admin/content/posts` | [endpoints/list-admin-posts.md](endpoints/list-admin-posts.md) | `services/zhicore-content/api/http/admin_posts_handler_test.go` | 已验证 |
| `DELETE` | `/api/v1/admin/content/posts/{postId}` | [endpoints/delete-admin-post.md](endpoints/delete-admin-post.md) | `services/zhicore-content/api/http/admin_posts_handler_test.go` | 已验证 |

## 已验证依赖语义错误映射

| code | 场景 | 验证 |
| --- | --- | --- |
| `4012` | 分类 / 话题 / 标签引用不存在 | `application.ErrTaxonomyReferenceNotFound` -> HTTP `404` / code `4012`，由 `create_post_handler_test.go` 覆盖。 |
| `4021` | File 媒体引用非法 | `application.ErrMediaRefInvalid` -> HTTP `400` / code `4021`，由 create / save draft / publish handler test 覆盖。 |
| `4023` | 发布封面不可用 | `application.ErrCoverUnavailable` -> HTTP `400` / code `4023`，由 `publish_post_handler_test.go` 覆盖。 |

## 服务级公开错误码

| code | HTTP status | 含义 | 适用场景 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | path/query/body 字段非法、cursor 非法、blocks schema 非法。 |
| `1003` | `429` | 请求过于频繁 | Content 业务限流命中，包含公开读、草稿保存、发布、互动、管理端和内部调用频控。 |
| `1004` | `503` | 服务暂时不可用 | MongoDB、PostgreSQL、User、File service、限流依赖或其他核心依赖不可用。 |
| `1005` | `404` | 数据不存在 | outbox event 等非文章资源不存在。 |
| `2006` | `401` | 请先登录 | 登录态 endpoint 缺少 Gateway 注入身份。 |
| `2007` | `403` | 需要特定角色 | 管理端路由缺少管理员角色。 |
| `2008` | `403` | 无权访问该资源 | 非作者访问草稿、编辑、发布、删除等作者资源。 |
| `4001` | `404` | 文章不存在 | 文章不存在、已删除或匿名不可见。 |
| `4002` | `409` | 文章已发布 | 重复发布。 |
| `4003` | `409` | 文章未发布 | 需要已发布状态但当前未发布。 |
| `4004` | `409` | 文章已删除 | 操作已删除文章。 |
| `4005` | `400` | 文章标题不能为空 | 发布时标题为空。 |
| `4006` | `400` | 文章内容不能为空 | 发布时正文有效内容为空。 |
| `4007` | `400` | 文章标题过长 | 标题超过限制。 |
| `4008` / `4009` | `409` | 点赞状态冲突 | 保留给状态机冲突；Go-first `PUT` / `DELETE` 默认幂等。 |
| `4010` / `4011` | `409` | 收藏状态冲突 | 保留给状态机冲突；Go-first `PUT` / `DELETE` 默认幂等。 |
| `4012` | `404` | 分类不存在 | 分类、话题或标签引用不存在。 |
| `4013`-`4024` | `400` / `409` / `500` | 正文错误 | blocks schema、正文大小、媒体引用、hash、repair 等错误。 |

## 测试要求

- 每个 endpoint 实现前必须补 handler contract test，覆盖 path、method、query/body、鉴权 header、envelope 和错误码。
- 作者资源 endpoint 必须覆盖缺少 `X-User-Id`、非作者访问和作者访问。
- 列表 endpoint 必须覆盖默认值、上限、非法参数、排序稳定性、空列表和 cursor 透传。
- 正文 endpoint 必须覆盖 blocks schema、`basePostVersion`、`draftBodyId`、`draftBodyHash`、媒体引用和 raw HTML 拒绝。
- Engagement endpoint 必须覆盖 `liked/favorited` 的 `true`、`false`、`null` 和 `degraded=true` 降级语义。
- 仅更新本文档和 endpoint schema 时运行 `bash scripts/check-structure.sh` 与 `git diff --check`。
