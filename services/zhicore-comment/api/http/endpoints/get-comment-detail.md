# 评论详情

状态：已验证。本文是 `zhicore-comment` Go-first HTTP API 的字段级 contract，由 handler contract test 覆盖。

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/posts/{postId}/comments/{commentId}` |
| 鉴权 | 匿名 / 登录用户 |
| 幂等 | 只读 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 对外文章 ID。 |
| `commentId` | string | 是 | Comment 对外评论 ID。 |

## Query / Body

无。

## 成功响应

`data` 为 `CommentItem`，字段见 `services/zhicore-comment/api/http/README.md`。

回复详情必须返回 `rootCommentId` 和 `parentCommentId`；根评论省略这两个字段。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | `Invalid request` | `postId` 或 `commentId` 格式非法。 |
| `1004` | `503` | `Service unavailable` | Content / PostgreSQL 不可用且无法降级。 |
| `4001` | `404` | `Post not found` | 文章不存在、不可见或不可评论。 |
| `5001` | `404` | `Comment not found` | 评论不存在、跨文章或已删除。 |

## 权限和可见性

- 匿名可读取公开可见文章下的未删除评论。
- 登录用户读取时返回 `viewer.liked`；匿名省略 `viewer`。
- 普通公开 API 对不存在、跨文章和已删除评论统一返回 `5001`，不暴露删除元数据。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `GetCommentDetail` |
| Ports | `CommentQueryRepository`、`CommentPostStatsRepository`、`UserProfileClient`、登录用户点赞查询端口 |
| 事件 | 无 |
