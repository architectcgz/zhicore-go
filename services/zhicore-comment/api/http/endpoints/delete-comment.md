# 删除评论

状态：已验证。本文是 `zhicore-comment` Go-first HTTP API 的字段级 contract，由 handler contract test 覆盖。

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `DELETE` |
| 主路径 | `/api/v1/posts/{postId}/comments/{commentId}` |
| 管理端路径 | `/api/v1/admin/comments/posts/{postId}/comments/{commentId}` |
| 鉴权 | 作者删除需要 `X-User-Id`；管理端需要 `X-User-Id` + `X-User-Roles: ADMIN` |
| 幂等 | 作者重复删除返回 `5001`；Admin 重复删除幂等成功 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 对外文章 ID。 |
| `commentId` | string | 是 | 被删除入口评论对外 ID。 |

## Body 字段

公开作者删除无 body。

Admin 删除可传：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `reason` | string | 是 | 管理删除原因，服务端 trim 后不能为空。 |

## 成功响应 `DeleteCommentResp`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 对外文章 ID。 |
| `commentId` | string | 是 | 被删除入口评论 ID。 |
| `rootCommentId` | string | 否 | 删除回复时返回所属根评论。 |
| `deletedAt` | string | 是 | RFC3339。 |
| `deletedByRole` | string | 是 | `AUTHOR` 或 `ADMIN`。 |
| `affectedCount` | int | 是 | 本次实际从 `NORMAL` 变为 `DELETED` 的评论数量。Admin 重复删除为 `0`。 |
| `alreadyDeleted` | boolean | 否 | 仅 Admin 重复删除时返回 `true`。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | `Invalid request` | path 非法或 Admin `reason` 为空。 |
| `1004` | `503` | `Service unavailable` | PostgreSQL 不可用。 |
| `2006` | `401` | `Authentication required` | 缺少登录态。 |
| `2007` | `403` | `Admin role required` | 管理端缺少管理员角色。 |
| `2008` | `403` | `Forbidden` | 普通用户删除他人评论。 |
| `5001` | `404` | `Comment not found` | 评论不存在、跨文章、已删除或普通用户重复删除。 |

## 事务和事件

- 删除任意评论节点都软删除该节点及整棵子树。
- 同一事务内维护 `reply_count`、`comment_post_stats` 和顶级评论 rank 可见性。
- 实际删除数量大于 `0` 时只写一条 `comment.deleted` outbox，payload 带 `affectedCount` 和 `isRoot`。
- Admin 重复删除不写 outbox、不重复扣减统计。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `DeleteComment`、`AdminDeleteComment` |
| Ports | `CommentCommandRepository`、`CommentStatsRepository`、`CommentPostStatsRepository`、`OutboxPublisher`、`TransactionRunner`、`Clock` |
| 事件 | `comment.deleted` |
