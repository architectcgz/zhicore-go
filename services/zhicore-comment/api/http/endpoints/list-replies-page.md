# 回复传统分页

状态：已验证。本文是 `zhicore-comment` Go-first HTTP API 的字段级 contract，由 handler contract test 覆盖。

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/posts/{postId}/comments/{commentId}/replies/page` |
| 鉴权 | 匿名 / 登录用户 |
| 幂等 | 只读 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 对外文章 ID。 |
| `commentId` | string | 是 | 根评论对外 ID。 |

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `page` | int | 否 | `1` | 页码，从 `1` 开始。 |
| `size` | int | 否 | `20` | 每页大小，范围 `1..100`。 |
| `sort` | string | 否 | `HOT` | `HOT` 或 `TIME`。 |

## 成功响应

`data` 为通用 `Page<CommentItem>`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `items` | `CommentItem[]` | 是 | 当前页回复，平铺返回根评论下的未删除回复。 |
| `page` | int | 是 | 当前页码。 |
| `size` | int | 是 | 每页大小。 |
| `total` | int64 | 是 | 根评论未删除回复总数。 |
| `pages` | int | 是 | 总页数。 |

每个回复 `CommentItem` 必须返回 `rootCommentId` 和 `parentCommentId`。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | `Invalid request` | path、分页或排序参数非法。 |
| `1004` | `503` | `Service unavailable` | Content / PostgreSQL 不可用且无法降级。 |
| `4001` | `404` | `Post not found` | 文章不存在、不可见或不可评论。 |
| `5005` | `404` | `Root comment not found` | 根评论不存在、跨文章、不是根评论或已删除。 |

## 排序

- `HOT`：`likeCount DESC, commentId ASC`。
- `TIME`：`commentId ASC`，按回复创建顺序展示。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `ListRepliesByPage` |
| Ports | `CommentQueryRepository`、`UserProfileClient`、登录用户点赞查询端口 |
| 事件 | 无 |
