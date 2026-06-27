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
| 幂等 | 无；重复提交会分配新 `floor` |

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
| `parentFloor` | int64 | 否 | 省略表示创建根评论 | 被回复评论的文章内楼层。传入时创建回复；被回复评论必须存在、未删除且属于同一 `postId`。 |
| `imageFileIds` | string[] | 否 | 省略或空数组表示无图片 | 图片文件引用，最多 9 个；语音评论不能同时传图片。 |
| `voiceFileId` | string | 否 | 省略表示无语音 | 语音文件引用；不能与 `imageFileIds` 同时存在。 |
| `voiceDuration` | int | 否 | `voiceFileId` 为空时必须省略 | 语音时长秒数；`voiceFileId` 非空时必须为正数，第一阶段上限按 Upload contract 固定为 60 秒候选值。 |

评论整体必须至少包含 `content`、`imageFileIds` 或 `voiceFileId` 中的一项。

`imageFileIds` 和 `voiceFileId` 必须是 Upload / File Service 返回的 opaque file ID。Comment 不接受系统内媒体的 CDN URL、对象存储 key、签名 URL 或相对路径作为写入事实；读取响应可以返回派生的 `imageUrls` / `voiceUrl`。

`parentFloor` 可以指向根评论或任意回复。application 必须在同一事务内解析直接父评论，推导所属根评论，并校验同文章、未删除、根归属正确。

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 对外文章 ID。 |
| `floor` | int64 | 是 | 新评论文章内楼层号。 |
| `rootFloor` | int64 | 否 | 回复所属根评论楼层；根评论省略。 |
| `parentFloor` | int64 | 否 | 直接被回复评论楼层；根评论省略。 |
| `createdAt` | string | 是 | RFC3339。 |

示例：

```json
{
  "code": 200,
  "message": "操作成功",
  "data": {
    "postId": "p1K8x9Q2",
    "floor": 26,
    "createdAt": "2026-06-27T10:30:00Z"
  },
  "timestamp": 1782112892184
}
```

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `postId` 格式非法、`parentFloor` 非正数、`imageFileIds` 数量超过 9、文件 ID 格式非法、`voiceDuration` 非法、图片和语音同时存在。 |
| `1004` | `503` | 服务暂时不可用 | Content / User / PostgreSQL / outbox 依赖不可用。 |
| `2006` | `401` | 请先登录 | 缺少 Gateway 注入的 `X-User-Id`。 |
| `4001` | `404` | 文章不存在 | Content 返回文章不存在、不可见或不可评论。 |
| `5003` | `400` | 评论内容不能为空 | 文本、图片、语音整体为空。 |
| `5004` | `400` | 评论内容过长 | `content` 超过 2000 字。 |
| `5005` | `404` | 根评论不存在 | `parentFloor` 指向的根评论或其根评论不存在。 |
| `5006` | `404` | 被回复的评论不存在 | `parentFloor` 指向的评论不存在。 |
| `5002` | `409` | 评论已删除 | `parentFloor` 指向的评论已删除，不能继续回复。 |

## 权限和可见性

- 需要登录用户。
- application 必须通过 Content contract 校验文章存在、可见且允许评论。
- application 必须通过 User contract 校验作者存在、状态可互动，必要时校验拉黑关系。
- request body 中出现 `userId` 时不作为当前操作者，handler 应按未知字段策略拒绝或忽略；当前操作者只来自可信 `X-User-Id`。

## 排序、分页和过滤

无。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `CreateComment` / `CreateReply` |
| 聚合 | `Comment`、`CommentStats` |
| Ports | `ContentPostClient`、`UserProfileClient`、`CommentFloorAllocator`、`CommentCommandRepository`、`CommentStatsRepository`、`OutboxPublisher`、`TransactionRunner`、`Clock` |
| 事务边界 | 分配 `floor`、写 `comments`、初始化统计、根评论初始化 `comment_hot_rank`、回复时递增根评论回复数、写 `comment.created` outbox 在同一 PostgreSQL 事务内完成。 |
| 事件 | `comment.created`，关键事件，必须进入 producer outbox；payload 必须包含 `postAuthorId`，回复时包含 `rootFloor/rootAuthorId`、`parentFloor/parentAuthorId`。 |
| 缓存 | 提交后失效文章评论列表；回复时额外失效根评论回复列表和根评论统计。 |

## 测试要求

- Handler contract test：待补，覆盖登录态缺失、空内容、文本过长、图片文件引用数量、图片文件引用格式、语音文件引用格式、语音图片互斥、父评论不存在、父评论已删除、Content 校验失败和成功响应。
- Application test：待补，覆盖 `floor` 分配、根评论创建、回复创建、统计初始化、outbox 写入和缓存失效语义。
- System HTTP test：待补。
