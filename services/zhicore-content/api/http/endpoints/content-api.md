# Content API 详细设计

状态：草案。本文是 `zhicore-content` Go-first HTTP API 的字段级 contract，尚未由 Go handler / contract test 验证。

## 总体方案

Content HTTP API 分为五组：

- 公开阅读：公开文章列表、作者文章、详情、正文、标签文章。
- 作者工作台：草稿、正文 blocks、发布、撤回、定时发布、删除恢复、标签维护。
- 互动：点赞、收藏、互动状态和 reader presence。
- 标签：标签列表、详情、搜索和热门标签。
- 管理运维：管理端文章查询、删除、Content outbox 查询和重试。

核心边界：

- Content 不提供 `GET /api/v1/users/{userId}/posts` facade。用户主页需要文章列表时直接调用 `GET /api/v1/posts?authorId={authorId}`。
- 对外 `postId` 是 string 公开 ID；内部数据库 `id BIGINT` 不进入 HTTP path。
- 公开读接口只返回 `PUBLISHED` 且未删除文章。作者工作台接口由 application 校验 owner。
- 正文只接受 `schemaVersion + blocks`，不接受 raw HTML 作为可信正文。外部链接和 external media 只做安全格式和 provider 白名单校验。
- 发布是用户可见原子操作：PG `published_*` 指针和 MongoDB body snapshot 必须一起成功。
- 限流按 API 情景分层处理：公开读和标签接口主要保护缓存 / DB 回源，草稿保存保护 MongoDB 写入和 autosave 风暴，发布 / 定时 / 管理命令不能在分布式限流不可确认时 fail-open，reader presence 不能影响正文可读性。完整矩阵见 `docs/architecture/services/content/rate-limiting.md`。

## 限流与降级

所有 endpoint 都接受 Gateway 粗限流保护。Content 服务内业务限流按 actor、post、session、service caller、operation 和高成本资源维度执行：

- 公开阅读、标签和互动查询：允许短时本机限流兜底，重点控制缓存穿透和数据库 / MongoDB 回源。
- 作者工作台写路径：`POST /posts`、草稿 meta/body、标签替换和删除等按 actor + post + operation 限制；autosave 可配置短 burst，但要限制持续 QPS 和单位时间 body 字节量。
- 发布生命周期：publish、unpublish、schedule、restore、delete 是高副作用路径；Redis / limiter 不可确认时返回 `1004`，不能放行。
- 点赞 / 收藏：接口幂等不等于无限放行；重复刷写仍要按 actor + post + operation 限制，避免统计和 outbox 被放大。
- Reader presence：heartbeat 可合并为 no-op success；Redis 不可用时 `PUT` / `GET` 返回空 `ReaderPresence` 并标记 `degraded=true`，`DELETE` 返回空成功，不能阻塞文章详情、正文读取或公开列表。
- 管理和运维：管理删除和 outbox retry 必须按 admin actor + target 限流，并保留审计语义。

限流命中返回 `1003 REQUEST_TOO_FREQUENT` / HTTP `429`。限流依赖不可用且当前 API 不允许 fail-open 时返回 `1004 SERVICE_DEGRADED` / HTTP `503`。

## 通用对象

### `PostSummary`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | 文章公开 ID。 |
| `authorId` | string | 是 | 作者 ID，来自 User 的稳定公开或 opaque ID。 |
| `authorName` | string | 否 | Content 作者快照。 |
| `authorAvatarFileId` | string | 否 | Upload 文件引用。 |
| `authorAvatarUrl` | string | 否 | 可展示头像 URL；由 Content 通过 Upload contract 解析或留空。 |
| `title` | string | 是 | 标题。 |
| `summary` | string | 否 | 摘要。 |
| `coverFileId` | string | 否 | 封面文件引用。 |
| `coverUrl` | string | 否 | 封面展示 URL；运行时派生字段，不作为 Content 持久化事实。 |
| `status` | string | 是 | `DRAFT`、`PUBLISHED`、`SCHEDULED`、`DELETED`。公开列表只返回 `PUBLISHED`。 |
| `publishedAt` | string | 否 | RFC3339。 |
| `createdAt` | string | 是 | RFC3339。 |
| `updatedAt` | string | 是 | RFC3339。 |
| `stats` | object | 是 | `viewCount`、`likeCount`、`favoriteCount`、`commentCount`。 |
| `viewer` | object | 否 | 登录用户视角：`liked`、`favorited`。匿名可省略。 |

### `PostDetail`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `post` | `PostSummary` | 是 | 文章摘要。 |
| `body` | `PostBody` | 否 | 公开详情可内联正文；列表接口不返回。 |
| `tags` | `Tag[]` | 否 | 标签列表。 |

### `PostBody`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `bodyId` | string | 是 | Content body UUID，不是产品版本号。 |
| `schemaVersion` | int | 是 | 当前第一阶段为 `1`。 |
| `format` | string | 是 | 固定 `blocks`。 |
| `blocks` | object[] | 是 | 结构化正文 blocks。 |
| `plainText` | string | 是 | 后端 canonicalize 后提取的纯文本。 |
| `contentHash` | string | 是 | `sha256:<hex>`。 |
| `sizeBytes` | int | 是 | canonical JSON 字节数。 |
| `createdAt` | string | 是 | RFC3339。 |

可发布 block 类型：`paragraph`、`heading`、`quote`、`list`、`code_block`、`table`、`collapsible`、`math`、`image`、`external_embed`、`attachment_gallery`。预留但不可发布：`mention`、`poll`、`custom_widget`。

系统内上传文件的正文 block 必须保存 Upload / File Service 返回的 `fileId`；如响应中包含 `url`，它只能作为运行时解析出的展示派生值。第三方外部资源必须使用 `external_embed` 或显式外部 URL 字段，不和系统内文件引用混用。

### `Tag`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `tagId` | string | 是 | 标签公开 ID。 |
| `name` | string | 是 | 展示名。 |
| `slug` | string | 是 | URL 友好唯一标识。 |
| `description` | string | 否 | 描述。 |
| `createdAt` | string | 是 | RFC3339。 |
| `updatedAt` | string | 是 | RFC3339。 |

### `Draft`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | 文章公开 ID。 |
| `postVersion` | int | 是 | 乐观锁版本。 |
| `meta` | object | 是 | `title`、`summary`、`coverFileId`、`topicId`、`categoryId`、`tags`。 |
| `draftBody` | `PostBody` | 否 | 当前草稿正文。空草稿可为空。 |
| `draftBodyHash` | string | 否 | 当前草稿 body hash。 |
| `savedAt` | string | 否 | 最近保存时间。 |

### `CursorPage<T>`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `items` | array | 是 | 当前页。 |
| `nextCursor` | string | 否 | 下一页 opaque cursor。 |
| `hasMore` | boolean | 是 | 是否有更多。 |
| `limit` | int | 是 | 本次 limit。 |

### `Page<T>`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `items` | array | 是 | 当前页。 |
| `page` | int | 是 | 从 `1` 开始。 |
| `size` | int | 是 | 每页大小。 |
| `total` | int | 是 | 总数。 |
| `pages` | int | 是 | 总页数。 |

## 公开阅读 API

### `GET /api/v1/posts`

公开文章列表。用于首页、用户主页文章区和标签 / 分类过滤场景。

Query：

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `authorId` | string | 否 | 无 | 作者过滤。获取某用户发表的前 20 条：`authorId={id}&limit=20`。 |
| `tag` | string | 否 | 无 | 标签 slug 过滤。 |
| `categoryId` | string | 否 | 无 | 分类过滤。 |
| `cursor` | string | 否 | 无 | Opaque cursor。 |
| `limit` | int | 否 | `20` | `1..100`。 |
| `sort` | string | 否 | `latest` | 第一阶段只支持 `latest`。热门/榜单归 Ranking。 |

响应 `data`：`CursorPage<PostSummary>`。

排序：`published_at DESC, public_id DESC`。cursor 内部包含 `published_at + public_id`，consumer 不解析。

错误：`1001` 参数非法。

### `GET /api/v1/posts/{postId}`

公开文章详情。

响应 `data`：`PostDetail`。默认包含 `body`；如果 body miss，返回 `4018 CONTENT_BODY_UNAVAILABLE` 并创建 repair task。

可见性：只允许读取 `PUBLISHED` 且未删除文章。

错误：`4001`、`4018`、`4024`。

### `GET /api/v1/posts/{postId}/body`

读取 published body。Search 等服务也通过 typed client 使用该语义。

响应 `data`：`PostBody`。

错误：`4001`、`4018`、`4019`、`4024`。

### `POST /api/v1/posts/batch-get`

批量获取文章摘要。

Body：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postIds` | string[] | 是 | 最多 100 个。 |
| `includeDeleted` | boolean | 否 | 仅管理员或服务间维护调用可用，默认 `false`。 |

响应 `data`：

```json
{
  "items": [
    { "postId": "p1...", "title": "...", "status": "PUBLISHED" }
  ],
  "missingPostIds": ["p1missing"]
}
```

普通公开调用只返回可见文章；不可见、已删除和不存在统一进入 `missingPostIds`。

### `GET /api/v1/posts/{postId}/tags`

公开读取文章标签。

响应 `data`：`Tag[]`。

可见性：只允许读取 `PUBLISHED` 且未删除文章；不可见、已删除和不存在统一返回 `4001`。

错误：`4001`。

## 作者工作台 API

### `POST /api/v1/posts`

创建草稿。

鉴权：登录用户。

Body：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `title` | string | 否 | 草稿阶段可空；发布时必填。最大 200。 |
| `summary` | string | 否 | 用户摘要。 |
| `coverFileId` | string | 否 | Upload 文件引用。 |
| `topicId` | string | 否 | 话题引用，拆 Topic 服务前由 Content 管理。 |
| `categoryId` | string | 否 | 分类引用。 |
| `tags` | string[] | 否 | 最多 10 个标签 slug 或名称。 |
| `body` | object | 否 | `schemaVersion + blocks`。为空时创建空草稿占位。 |

响应 `data`：

```json
{
  "postId": "p1K8x9Q2",
  "postVersion": 1
}
```

错误：`2006`、`1001`、`4007`、`4012`、`4013`、`4021`。

### `GET /api/v1/me/posts`

我的文章列表。

鉴权：登录用户。

Query：

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `status` | string | 否 | `all` | `all`、`draft`、`published`、`scheduled`、`deleted`。 |
| `cursor` | string | 否 | 无 | Opaque cursor。 |
| `limit` | int | 否 | `20` | `1..100`。 |

响应 `data`：`CursorPage<PostSummary>`。

排序：`updated_at DESC, public_id DESC`。

### `GET /api/v1/me/drafts`

我的草稿列表。只返回草稿摘要，不批量读取 MongoDB 正文。

Query：`cursor`、`limit`。

响应 `data`：`CursorPage<PostSummary>`。

### `GET /api/v1/posts/{postId}/draft`

读取当前草稿。

鉴权：作者。

响应 `data`：`Draft`。

错误：`2006`、`2008`、`4001`、`4004`。

### `PATCH /api/v1/posts/{postId}/draft/meta`

更新草稿元数据。

鉴权：作者。

Body：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `basePostVersion` | int | 是 | 乐观锁版本。 |
| `title` | string | 否 | 最大 200。 |
| `summary` | string | 否 | 最大长度由 Content 配置。 |
| `coverFileId` | string | 否 | 置空表示移除封面。 |
| `topicId` | string | 否 | 置空表示移除话题。 |
| `categoryId` | string | 否 | 分类引用。 |
| `tags` | string[] | 否 | 最多 10 个。 |

响应 `data`：`Draft`。

错误：`2008`、`4001`、`4004`、`4007`、`4012`、`4017`、`4021`。

### `PUT /api/v1/posts/{postId}/draft/body`

保存草稿正文。使用 copy-on-write，不原地覆盖旧 body。

鉴权：作者。

Body：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `basePostVersion` | int | 是 | 乐观锁版本。 |
| `baseDraftBodyId` | string | 否 | 当前草稿 body id；空草稿可为空。 |
| `baseDraftBodyHash` | string | 否 | 当前草稿 hash；空草稿可为空。 |
| `schemaVersion` | int | 是 | 当前为 `1`。 |
| `blocks` | object[] | 是 | 正文 blocks。 |
| `clientSavedAt` | string | 否 | 客户端保存时间，仅作冲突提示参考，不作为事实源。 |

响应 `data`：

```json
{
  "postId": "p1K8x9Q2",
  "postVersion": 2,
  "draftBodyId": "uuid",
  "draftBodyHash": "sha256:...",
  "savedAt": "2026-06-27T10:00:00Z",
  "wordCount": 1200
}
```

错误：`2008`、`4013`、`4014`、`4015`、`4017`、`4019`、`4020`、`4021`、`4022`、`4024`。

### `DELETE /api/v1/posts/{postId}/draft`

删除草稿指针并创建正文清理任务。已发布文章删除草稿不影响线上 published body。

鉴权：作者。

响应 `data`：可省略。

错误：`2008`、`4001`、`4004`。

### `POST /api/v1/posts/{postId}/publish`

发布草稿。

鉴权：作者。

Body：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `basePostVersion` | int | 是 | 发布确认时看到的版本。 |
| `draftBodyId` | string | 是 | 要发布的草稿 body。 |
| `draftBodyHash` | string | 是 | 要发布的草稿 hash。 |
| `idempotencyKey` | string | 否 | 可选；推荐调用方同时传 `Idempotency-Key` header。 |

响应 `data`：

```json
{
  "postId": "p1K8x9Q2",
  "postVersion": 3,
  "publishedAt": "2026-06-27T10:00:00Z"
}
```

错误：`4002`、`4005`、`4006`、`4016`、`4017`、`4018`、`4019`、`4021`、`4023`。

### `POST /api/v1/posts/{postId}/unpublish`

撤回已发布文章，回到草稿或不可公开状态。

鉴权：作者。

Body：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `basePostVersion` | int | 是 | 撤回确认时看到的版本。 |

响应 `data`：

```json
{
  "postId": "p1K8x9Q2",
  "postVersion": 4,
  "status": "DRAFT"
}
```

错误：`4003`、`4004`、`4017`。

### `POST /api/v1/posts/{postId}/schedule`

设置定时发布。

鉴权：作者。

Body：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `basePostVersion` | int | 是 | 定时发布确认时看到的版本。 |
| `draftBodyId` | string | 是 | 定时发布时要上线的草稿 body。 |
| `draftBodyHash` | string | 是 | 定时发布时要上线的草稿 hash。 |
| `scheduledAt` | string | 是 | RFC3339，必须是未来时间。 |

响应 `data`：

```json
{
  "postId": "p1K8x9Q2",
  "postVersion": 4,
  "status": "SCHEDULED",
  "scheduledAt": "2026-06-28T10:00:00Z"
}
```

错误：`4004`、`4016`、`4017`、`4019`、`4021`、`4023`。

### `DELETE /api/v1/posts/{postId}/schedule`

取消定时发布。

鉴权：作者。

Query：可选 `basePostVersion`。

响应 `data`：

```json
{
  "postId": "p1K8x9Q2",
  "postVersion": 5,
  "status": "DRAFT"
}
```

错误：`4003`、`4017`。

### `DELETE /api/v1/posts/{postId}`

作者软删除文章。

鉴权：作者。

Query：可选 `basePostVersion`。

响应 `data`：

```json
{
  "postId": "p1K8x9Q2",
  "postVersion": 6,
  "status": "DELETED"
}
```

错误：`4004`、`4017`。

### `POST /api/v1/posts/{postId}/restore`

恢复作者软删除文章。

鉴权：作者。

Body：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `basePostVersion` | int | 否 | 恢复确认时看到的版本。 |

响应 `data`：

```json
{
  "postId": "p1K8x9Q2",
  "postVersion": 7,
  "status": "DRAFT"
}
```

错误：`4001`、`4017`。

### `PUT /api/v1/posts/{postId}/tags`

替换文章标签集合。

鉴权：作者。

Body：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `basePostVersion` | int | 是 | 乐观锁版本。 |
| `tags` | string[] | 是 | 最多 10 个；空数组表示清空标签。 |

响应 `data`：`Tag[]`。

错误：`2008`、`4001`、`4004`、`4012`、`4017`。

### `DELETE /api/v1/posts/{postId}/tags/{slug}`

删除单个标签关系。

鉴权：作者。

响应 `data`：`Tag[]`。

错误：`2008`、`4001`、`4004`、`4012`。

## 互动和 presence API

### `PUT /api/v1/posts/{postId}/like`

幂等点赞。已点赞时返回成功，不报冲突。

鉴权：登录用户。

响应 `data`：

```json
{ "postId": "p1...", "liked": true, "likeCount": 10 }
```

错误：`2006`、`4001`。

### `DELETE /api/v1/posts/{postId}/like`

幂等取消点赞。未点赞时返回成功。

响应 `data`：

```json
{ "postId": "p1...", "liked": false, "likeCount": 9 }
```

错误：`2006`、`4001`。

### `PUT /api/v1/posts/{postId}/favorite`

幂等收藏。响应包含 `favorited` 和 `favoriteCount`。

响应 `data`：

```json
{ "postId": "p1...", "favorited": true, "favoriteCount": 6 }
```

错误：`2006`、`4001`。

### `DELETE /api/v1/posts/{postId}/favorite`

幂等取消收藏。响应包含 `favorited` 和 `favoriteCount`。

响应 `data`：

```json
{ "postId": "p1...", "favorited": false, "favoriteCount": 5 }
```

错误：`2006`、`4001`。

### `GET /api/v1/posts/{postId}/engagement`

互动状态。

鉴权：匿名可查计数；登录用户额外返回 viewer 状态。

响应 `data`：

```json
{
  "postId": "p1...",
  "stats": {
    "viewCount": 100,
    "likeCount": 10,
    "favoriteCount": 5,
    "commentCount": 3
  },
  "viewer": {
    "liked": true,
    "favorited": false
  }
}
```

错误：`4001`。

### `POST /api/v1/posts/engagement/batch-status`

批量查询当前用户互动状态。

鉴权：登录用户。

Body：`postIds`，最多 100 个。

响应 `data`：

```json
{
  "items": [
    { "postId": "p1...", "liked": true, "favorited": false }
  ]
}
```

错误：`2006`、`1001`。

### Reader presence

| 方法 | 路径 | Body / Query | 响应 |
| --- | --- | --- | --- |
| `PUT` | `/api/v1/posts/{postId}/reader-sessions/{sessionId}` | body 可选 `clientId` | `ReaderPresence` |
| `DELETE` | `/api/v1/posts/{postId}/reader-sessions/{sessionId}` | 无 | 空 |
| `GET` | `/api/v1/posts/{postId}/reader-presence` | 无 | `ReaderPresence` |

`ReaderPresence`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `readingCount` | int | 当前估算阅读人数。 |
| `avatars` | object[] | 最多 3 个头像摘要：`userId`、`nickname`、`avatarUrl`。 |
| `degraded` | boolean | 是否因为 Redis / presence 依赖不可用返回降级空摘要；正常响应为 `false`。 |

Presence 是 Redis 短生命周期状态，不能影响文章可读性。Redis 不可用时：

- `PUT /reader-sessions/{sessionId}` 返回 HTTP `200`，`data` 为 `readingCount=0`、`avatars=[]`、`degraded=true`。
- `DELETE /reader-sessions/{sessionId}` 返回 HTTP `200`，`data` 为空。
- `GET /reader-presence` 返回 HTTP `200`，`data` 为 `readingCount=0`、`avatars=[]`、`degraded=true`。

## 标签 API

### `GET /api/v1/tags`

Query：`cursor`、`limit`，默认 `20`，最大 `100`。

响应 `data`：`CursorPage<Tag>`。

### `GET /api/v1/tags/{slug}`

响应 `data`：`Tag`。

错误：`4012`。

### `GET /api/v1/tags/search`

Query：`keyword` 必填，`limit` 默认 `10`，最大 `50`。

响应 `data`：`Tag[]`。

### `GET /api/v1/tags/hot`

Query：`limit` 默认 `10`，最大 `50`。

响应 `data`：`TagStats[]`，按 `postCount DESC, slug ASC`。

### `GET /api/v1/tags/{slug}/posts`

Query：`cursor`、`limit`。

响应 `data`：`CursorPage<PostSummary>`，只返回公开文章。

`TagStats`：`tagId`、`name`、`slug`、`postCount`。

错误：`4012`。

## 管理和运维 API

### `GET /api/v1/admin/content/posts`

管理端文章查询。

鉴权：管理员。

Query：

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `keyword` | string | 无 | 标题 / 摘要搜索，最大 100 字符。 |
| `status` | string | 无 | `DRAFT`、`PUBLISHED`、`SCHEDULED`、`DELETED`。 |
| `authorId` | string | 无 | 作者过滤。 |
| `page` | int | `1` | 从 1 开始。 |
| `size` | int | `20` | `1..100`。 |

响应 `data`：`Page<AdminPostItem>`。

`AdminPostItem`：`postId`、`title`、`authorId`、`authorName`、`status`、`viewCount`、`likeCount`、`commentCount`、`createdAt`、`publishedAt`。

### `DELETE /api/v1/admin/content/posts/{postId}`

管理端软删除文章。必须记录 Admin 审计事实；Content 负责文章状态校验和 outbox 事件。

鉴权：管理员。

Body：可选 `reason`。

响应 `data`：

```json
{
  "postId": "p1K8x9Q2",
  "status": "DELETED"
}
```

错误：`2007`、`4001`、`4004`。

### `GET /api/v1/admin/content/outbox-events`

查询 Content outbox dead / failed 事件。

鉴权：管理员。

Query：`status=dead|failed`、`eventType`、`page`、`size`。

响应 `data`：`Page<OutboxEventItem>`。

`OutboxEventItem`：`eventId`、`eventType`、`aggregateType`、`aggregateId`、`aggregateVersion`、`retryCount`、`lastError`、`occurredAt`、`createdAt`、`updatedAt`。

错误：`2007`、`1001`。

### `POST /api/v1/admin/content/outbox-events/{eventId}/retry`

手动重试 outbox 事件。

鉴权：管理员。

Body：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `reason` | string | 是 | 重试原因，进入审计。 |

响应 `data`：`eventId`、`status`。

错误：`2007`、`1001`、`1005`。

## 实现切片建议

1. **HTTP contract test scaffold**：建立 Content handler contract test helper，覆盖 envelope、auth header、错误码和 JSON 字段。
2. **草稿和发布主链路**：实现 `POST /posts`、`PATCH /draft/meta`、`PUT /draft/body`、`POST /publish`、`GET /posts/{postId}`。
3. **公开列表和作者列表**：实现 `GET /posts`，覆盖 `authorId + limit=20`、cursor、排序稳定性和空列表。
4. **标签和正文查询**：实现 tags、body、batch-get。
5. **互动和 presence**：实现点赞、收藏、engagement、reader presence。
6. **管理运维**：实现 admin posts、outbox 查询和 retry。

每个切片都必须同步 application use case、ports、repository / adapter 和 handler contract tests。
