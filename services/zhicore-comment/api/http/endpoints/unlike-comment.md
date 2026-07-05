# 取消点赞评论

状态：已验证。本文是 `zhicore-comment` Go-first HTTP API 的字段级 contract，由 handler contract test 覆盖。

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `DELETE` |
| 主路径 | `/api/v1/posts/{postId}/comments/{commentId}/like` |
| 鉴权 | 登录用户，`X-User-Id` 必填 |
| 幂等 | 未点赞时成功，不写 delta 或事件 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 对外文章 ID。 |
| `commentId` | string | 是 | Comment 对外评论 ID。 |

## 成功响应 `UnlikeCommentResp`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 对外文章 ID。 |
| `commentId` | string | 是 | Comment 对外评论 ID。 |
| `liked` | boolean | 是 | 固定为 `false`。 |
| `changed` | boolean | 是 | 本次是否从已点赞变为未点赞。 |
| `occurredAt` | string | 是 | RFC3339。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | `Invalid request` | path 非法。 |
| `1004` | `503` | `Service unavailable` | PostgreSQL 不可用。 |
| `2006` | `401` | `Authentication required` | 缺少登录态。 |
| `5001` | `404` | `Comment not found` | 评论不存在或跨文章。已删除评论允许撤销历史点赞。 |

## 事务和事件

- 状态变化时同一事务内删除 `comment_likes`、写 `comment_counter_deltas(-1)` 和 `comment.unliked` outbox。
- 取消未点赞幂等成功，不写 delta、不写 outbox。
- 取消点赞不调用 User relation guard，允许用户撤销自己的历史点赞。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `UnlikeComment` |
| Ports | `CommentCommandRepository`、`CommentCounterDeltaRepository`、`OutboxPublisher`、`TransactionRunner`、`Clock` |
| 事件 | `comment.unliked` |
