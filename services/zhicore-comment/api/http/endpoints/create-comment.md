# 创建评论

状态：草案。本文是 `zhicore-comment` Go-first HTTP API 的字段级 contract，尚未由 Go handler / contract test 验证。

## 来源

- 服务总览：`docs/architecture/services/comment/README.md`
- 模块 API 设计：`docs/architecture/module/comment/api.md`
- 模块 service 设计：`docs/architecture/module/comment/service.md`
- 模块 domain 设计：`docs/architecture/module/comment/domain.md`
- 当前 API schema：`services/zhicore-comment/api/http/README.md`
- Go handler：待补。
- Go contract test：待补。
- Java 参考：`../zhicore-microservice/zhicore-comment/src/main/java/com/zhicore/comment/interfaces/controller/CommentCommandController.java`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/posts/{postId}/comments` |
| 兼容别名 | 无 |
| Content-Type | `application/json` |
| 鉴权 | 登录用户，`X-User-Id` 必填 |
| 幂等 | 无；重复提交会创建新评论并返回新的 `commentId` |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 对外文章 ID。 |

## Query 参数

无。

## Body 字段

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `content` | string | 否 | 可省略；省略时必须有图片或语音 | 评论文本，最多 2000 字；服务端 trim 后判断空白。 |
| `parentCommentId` | string | 否 | 省略表示创建根评论 | 被回复评论的对外评论 ID。传入时创建回复；被回复评论必须存在、未删除且属于同一 `postId`。 |
| `imageFileIds` | string[] | 否 | 省略或空数组表示无图片 | 图片文件引用，最多 9 个；语音评论不能同时传图片。 |
| `voiceFileId` | string | 否 | 省略表示无语音 | 语音文件引用；不能与 `imageFileIds` 同时存在。 |
| `voiceDuration` | int | 否 | `voiceFileId` 为空时必须省略 | 语音时长秒数；`voiceFileId` 非空时必须为正数，上限以 File contract 固定值为准。 |

评论整体必须至少包含 `content`、`imageFileIds` 或 `voiceFileId` 中的一项。

`imageFileIds` 和 `voiceFileId` 必须是 File service 返回的 opaque file ID。Comment 不接受系统内媒体的 CDN URL、对象存储 key、签名 URL 或相对路径作为写入事实；读取响应可以返回派生的 `imageUrls` / `voiceUrl`。

`parentCommentId` 可以指向根评论或任意回复。application 必须在同一事务内解析直接父评论，推导所属根评论，并校验同文章、未删除、根归属正确。

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 对外文章 ID。 |
| `commentId` | string | 是 | 新评论对外评论 ID。 |
| `rootCommentId` | string | 否 | 回复所属根评论 ID；根评论省略。 |
| `parentCommentId` | string | 否 | 直接被回复评论 ID；根评论省略。 |
| `createdAt` | string | 是 | RFC3339。 |

示例：

```json
{
  "code": 200,
  "message": "OK",
  "data": {
    "postId": "p1K8x9Q2",
    "commentId": "c1K8x9Q2",
    "createdAt": "2026-06-27T10:30:00Z"
  },
  "timestamp": 1782112892184
}
```

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | `Invalid request` | `postId` 格式非法、`parentCommentId` 格式非法、`imageFileIds` 数量超过 9、文件 ID 格式非法、`voiceDuration` 非法、图片和语音同时存在。 |
| `1004` | `503` | `Service unavailable` | Content / User / File service / PostgreSQL / outbox 依赖不可用。 |
| `2006` | `401` | `Authentication required` | 缺少 Gateway 注入的 `X-User-Id`。 |
| `4001` | `404` | `Post not found` | Content 返回文章不存在、不可见或不可评论。 |
| `5001` | `404` | `Comment not found` | `parentCommentId` 指向的评论已删除或对当前用户不可见。 |
| `5003` | `400` | `Comment content is required` | 文本、图片、语音整体为空。 |
| `5004` | `400` | `Comment content is too long` | `content` 超过 2000 字。 |
| `5005` | `404` | `Root comment not found` | `parentCommentId` 指向的根评论或其根评论不存在 / 已删除。 |
| `5006` | `404` | `Parent comment not found` | `parentCommentId` 指向的评论不存在 / 已删除。 |

## 权限和可见性

- 需要登录用户。
- application 必须通过 Content contract 校验文章存在、可见且允许评论。
- application 必须通过 User contract 校验作者存在、状态可互动。
- 创建根评论时校验文章作者是否拉黑当前用户；创建回复时校验文章作者和直接父评论作者是否拉黑当前用户。
- `imageFileIds` / `voiceFileId` 必须通过 File service contract 校验存在、类型匹配且状态可用；File service 不可用时写请求失败。
- request body 中出现 `userId` 时不作为当前操作者，handler 应按未知字段策略拒绝或忽略；当前操作者只来自可信 `X-User-Id`。

## 排序、分页和过滤

无。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `CreateComment` / `CreateReply` |
| 聚合 | `Comment`、`CommentStats` |
| Ports | `ContentPostClient`、`UserProfileClient`、`UserRelationClient`、`FileReferenceClient`、`RateLimiter`、`CommentIDCodec`、`CommentCommandRepository`、`CommentStatsRepository`、`CommentPostStatsRepository`、`OutboxPublisher`、`TransactionRunner`、`Clock` |
| 事务边界 | 本地父评论 / 根评论树校验通过后写 `comments` 并由 PostgreSQL identity 生成内部 `comment_id`，初始化统计、维护 `comment_post_stats`、根评论初始化 `comment_hot_rank` / `comment_recommended_rank`、回复时递增根评论 `reply_count`、写 `comment.created` outbox 在同一 PostgreSQL 事务内完成。 |
| 事件 | `comment.created`，关键事件，必须进入 producer outbox；payload 必须包含 `publicId`、`internalId`、`postAuthorId`，回复时包含 `rootId/rootAuthorId`、`parentId/parentAuthorId`。 |
| 缓存 | 提交后失效文章评论列表；回复时额外失效根评论回复列表和根评论统计。 |

## 测试要求

- Handler contract test：待补，覆盖登录态缺失、空内容、文本过长、图片文件引用数量、图片文件引用格式、语音文件引用格式、语音图片互斥、父评论不存在 / 已删除、Content 校验失败、File 校验失败、User 状态失败和成功响应。
- Application test：待补，覆盖 `commentId` 生成 / 编码、根评论创建、回复创建、统计初始化、文章级统计、outbox 写入和缓存失效语义。
- System HTTP test：待补。
