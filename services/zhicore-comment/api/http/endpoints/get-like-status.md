# 评论点赞状态

状态：已验证。本文是 `zhicore-comment` Go-first HTTP API 的字段级 contract，由 handler contract test 覆盖。

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/posts/{postId}/comments/{commentId}/liked` |
| 鉴权 | 登录用户，`X-User-Id` 必填 |
| 幂等 | 只读 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 对外文章 ID。 |
| `commentId` | string | 是 | Comment 对外评论 ID。 |

## 成功响应 `CommentLikeStatusResp`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 对外文章 ID。 |
| `commentId` | string | 是 | Comment 对外评论 ID。 |
| `liked` | boolean | 是 | 当前登录用户是否已点赞。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | `Invalid request` | path 非法。 |
| `1004` | `503` | Service unavailable | PostgreSQL 不可用。 |
| `2006` | `401` | `Authentication required` | 缺少登录态。 |
| `5001` | `404` | `Comment not found` | 评论不存在、跨文章或已删除。 |

## 语义

`liked` 必须以 `comment_likes` 为强一致事实；不能用最终一致的 `likeCount` 推断。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `GetLikeStatus` |
| Ports | `CommentQueryRepository` |
| 事件 | 无 |
