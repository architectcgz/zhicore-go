# unlike-post

状态：草案。

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- Engagement 设计：`docs/architecture/services/content/engagement-design.md`
- 限流设计：`docs/architecture/services/content/rate-limiting.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `DELETE` |
| 主路径 | `/api/v1/posts/{postId}/like` |
| 兼容别名 | 无 |
| Content-Type | 无 |
| 鉴权 | 登录用户 |
| 幂等 | 是；未点赞时取消点赞返回当前确定状态，不重复减少统计或写 outbox。 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | 文章公开 ID。 |
| `liked` | bool | 是 | 固定为 `false`。 |
| `favorited` | bool | 是 | 当前用户收藏状态；必须是确定值。 |
| `stats` | `PostStats` | 是 | 最新互动统计。 |

`PostStats`：`viewCount`、`likeCount`、`favoriteCount`、`commentCount`。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `postId` 为空或格式非法。 |
| `1003` | `429` | 请求过于频繁 | Content 业务限流拒绝。 |
| `1004` | `503` | 服务暂时不可用 | PostgreSQL 或限流依赖不可用，不能确认写入配额。 |
| `2006` | `401` | 请先登录 | 缺少可信 `X-User-Id`。 |
| `4001` | `404` | 文章不存在 | 文章不存在、已删除或不可互动。 |

## 副作用

- 已点赞时在同一 PostgreSQL 事务内删除 `post_likes`、减少 `post_stats.like_count`、写 `content.post.unliked` outbox。
- 未点赞时不写统计 delta，不写 outbox。
- `like_count` 不允许减成负数。

## 测试要求

- Application test：重复取消点赞幂等成功，不重复减少统计或写 outbox。
- Repository test：删除关系、统计 delta 和 outbox 写入在同一事务内。
- Handler contract test：登录态、成功 envelope、文章不存在、限流/依赖错误。
