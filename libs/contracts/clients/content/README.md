# Content Typed Client Contract

本目录记录其他服务同步调用 `zhicore-content` 时可依赖的 Go-first typed client contract。Provider 是 Content；consumer 不能复制 Content DTO、不能访问 Content 数据库，也不能导入 `services/zhicore-content/internal`。

当前状态为设计草案，尚未生成 Go client 代码。Go 实现时应把本文件拆成稳定 DTO、client interface、HTTP adapter 和 contract tests。

## 使用场景

- Search 消费 Content 事件后，调用 Content 获取 published body 做全文索引。
- Comment 验证文章存在性、评论权限，并保存 Content 内部 opaque reference 供事件下游使用。
- Ranking / Notification 需要文章摘要时，批量获取 Content summary。
- User 不提供用户文章 facade。用户主页文章列表直接走 Content HTTP API。
- Admin 如需文章管理入口，可以作为 facade 调用 Content admin contract，但不拥有文章 mutation 语义。

## Caller Identity

Content typed client 调用必须携带服务间 caller 身份，供 Content 做内部调用限流、审计和观测。caller 身份不是用户身份，不能替代 `Actor` / `AuthContext` 或资源权限校验。

| Header | 必填 | 来源 | 说明 |
| --- | --- | --- | --- |
| `X-Caller-Service` | 是 | consumer 服务静态配置 | 稳定服务名，例如 `zhicore-search`、`zhicore-comment`、`zhicore-ranking`、`zhicore-notification`、`zhicore-admin`。 |
| `X-Caller-Operation` | 是 | typed client 调用点常量 | 稳定低基数字符串，例如 `search.index_post_body`、`comment.check_post_visible`、`ranking.batch_post_summary`。不得包含用户输入、`postId`、cursor 或错误文本。 |
| `X-Request-Id` | 否 | 上游请求或任务 metadata | 用于单次请求关联。 |
| `X-Trace-Id` | 否 | 上游请求或任务 metadata | 用于跨服务链路关联。 |

规则：

- typed client adapter 负责写入 `X-Caller-Service` / `X-Caller-Operation`，业务代码不手写 header。
- Content 对内部高成本接口按 `callerService + operation + target` 限流；未知 caller 或缺少 caller header 的服务间-only endpoint 默认按未认证内部调用处理，返回 `SERVICE_DEGRADED` 或权限类错误，而不是落到匿名公开配额。
- 如果某个 consumer 需要代表当前用户调用 Content，必须在 Content HTTP schema 中显式登记允许的用户身份 header；普通 typed client 查询默认只使用服务身份。

## Client interface 草案

```go
type Client interface {
    GetPostSummary(ctx context.Context, postID string) (PostSummary, error)
    GetPostDetail(ctx context.Context, postID string) (PostDetail, error)
    GetPublishedBody(ctx context.Context, postID string) (PostBody, error)
    BatchGetPostSummaries(ctx context.Context, postIDs []string) (BatchPostSummaryResult, error)
    ListPublishedPosts(ctx context.Context, query ListPublishedPostsQuery) (CursorPage[PostSummary], error)
    CheckPostsVisible(ctx context.Context, postIDs []string) (map[string]bool, error)
    CheckPostCommentable(ctx context.Context, postID string) (PostCommentContext, error)
}
```

`CheckPostsVisible` 是 Go 侧建议新增的窄 contract，用于 Notification 等服务只需要验证文章是否公开可见的场景；consumer 不应为了存在性校验拉取完整正文。Comment 写路径必须使用 `CheckPostCommentable`，拿到 Content 内部 `internalId` 后再写入本地评论和 outbox。

## DTO

### `PostSummary`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `PostID` | string | 文章公开 ID。 |
| `AuthorID` | string | 作者 ID。 |
| `AuthorName` | string | Content 作者快照。 |
| `Title` | string | 标题。 |
| `Summary` | string | 摘要。 |
| `CoverFileID` | string | Upload 文件引用。 |
| `Status` | string | `PUBLISHED` 等状态。 |
| `PublishedAt` | time | 发布时间。 |
| `Stats` | struct | 浏览、点赞、收藏、评论计数。 |

### `PostCommentContext`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `PostID` | string | Content 公开 `public_id`，与请求 `postID` 对应。 |
| `InternalID` | int64 | Content 内部 `post_id` opaque reference；Comment 保存后只用于事件、统计下游和服务间内部引用。 |
| `AuthorID` | int64 | 文章作者 User 内部 ID，用于通知和拉黑 guard。 |
| `Commentable` | bool | 是否允许创建评论；不可评论时返回业务错误或 `false` 的选择由 HTTP schema 固定。 |
| `Status` | string | Content 生命周期状态快照，用于错误映射和审计。 |

### `PostDetail`

`PostSummary + Tags + Body`。普通 consumer 优先使用 `PostSummary`；只有确实需要展示或索引正文时才读取 `Body`。

### `PostBody`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `BodyID` | string | Content body UUID。 |
| `SchemaVersion` | int | blocks schema 版本。 |
| `Blocks` | []Block | 结构化正文。 |
| `PlainText` | string | canonical 纯文本。 |
| `ContentHash` | string | `sha256:<hex>`。 |
| `SizeBytes` | int64 | canonical JSON 字节数。 |

Consumer 不得把 Content blocks 复制成自己的领域模型。Search 可以把它转换成索引文档；其他服务只读取自己需要的稳定字段。

### `ListPublishedPostsQuery`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `AuthorID` | string | 可选作者过滤。 |
| `Tag` | string | 可选标签 slug。 |
| `CategoryID` | string | 可选分类过滤。 |
| `Cursor` | string | Opaque cursor。 |
| `Limit` | int | `1..100`，默认 20。 |

Consumer 不解析 cursor，只保存并回传给 Content。

## 错误语义

| 错误 | 语义 | Consumer 处理 |
| --- | --- | --- |
| `POST_NOT_FOUND` | 文章不存在或不可见 | 按业务返回 not found、跳过或进入 DLQ。 |
| `CONTENT_BODY_UNAVAILABLE` | published body 不可读 | Search 应 retry 或 DLQ；普通查询返回 Content 降级错误。 |
| `SERVICE_DEGRADED` | Content 或其依赖不可用 | 按调用方 resilience policy retry、熔断或降级。 |
| `PARAM_ERROR` | consumer 请求 contract 错误 | 视为 consumer bug，不应重试。 |

## Resilience

- HTTP client policy 必须按 `docs/architecture/runtime-operations.md` 配置 timeout、retry、circuit breaker 和观测字段。
- 查询类调用可以重试；mutation 不通过 typed client 暴露给普通 consumer。
- Content 失败不能被 User、Admin 或其他 facade 伪装为空列表或成功。
