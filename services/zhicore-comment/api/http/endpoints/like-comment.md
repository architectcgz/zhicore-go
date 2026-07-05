# 点赞评论

状态：已验证。本文是 `zhicore-comment` Go-first HTTP API 的字段级 contract，由 handler contract test 覆盖。

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/posts/{postId}/comments/{commentId}/like` |
| 鉴权 | 登录用户，`X-User-Id` 必填 |
| 幂等 | 重复点赞成功，不重复写 delta 或事件 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 对外文章 ID。 |
| `commentId` | string | 是 | Comment 对外评论 ID。 |

## 成功响应 `LikeCommentResp`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 对外文章 ID。 |
| `commentId` | string | 是 | Comment 对外评论 ID。 |
| `liked` | boolean | 是 | 固定为 `true`。 |
| `changed` | boolean | 是 | 本次是否从未点赞变为已点赞。 |
| `occurredAt` | string | 是 | RFC3339。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | `Invalid request` | path 非法。 |
| `1004` | `503` | `Service unavailable` | User relation 或 PostgreSQL 不可用。 |
| `2006` | `401` | `Authentication required` | 缺少登录态。 |
| `2008` | `403` | `Forbidden` | 评论作者拉黑当前用户。 |
| `5001` | `404` | `Comment not found` | 评论不存在、跨文章或已删除。 |

## 事务和事件

- 状态变化时同一事务内写 `comment_likes`、`comment_counter_deltas(+1)` 和 `comment.liked` outbox。
- 重复点赞不写 delta、不写 outbox。
- 点赞请求必须校验评论作者是否拉黑当前用户；关系不可确认时 fail closed。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `LikeComment` |
| Ports | `CommentCommandRepository`、`CommentCounterDeltaRepository`、`UserRelationClient`、`OutboxPublisher`、`TransactionRunner`、`Clock` |
| 事件 | `comment.liked` |
