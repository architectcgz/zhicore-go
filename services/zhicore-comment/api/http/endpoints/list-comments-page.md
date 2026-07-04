# 文章顶级评论传统分页

状态：草案。本文是 `zhicore-comment` Go-first HTTP API 的字段级 contract，尚未由 Go handler / contract test 验证。

## 来源

- 服务总览：`docs/architecture/services/comment/README.md`
- 模块 API 设计：`docs/architecture/module/comment/api.md`
- 模块 service 设计：`docs/architecture/module/comment/service.md`
- 当前 API schema：`services/zhicore-comment/api/http/README.md`
- Go handler：待补。
- Go contract test：待补。
- Java 参考：`../zhicore-microservice/zhicore-comment/src/main/java/com/zhicore/comment/interfaces/controller/CommentQueryController.java`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/posts/{postId}/comments/page` |
| 兼容别名 | 无 |
| Content-Type | 无 |
| 鉴权 | 匿名 / 登录用户 |
| 幂等 | 只读 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 对外文章 ID。 |

## Query 参数 `ListCommentsPageReq`

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `page` | int | 否 | `1` | 页码，从 `1` 开始。 |
| `size` | int | 否 | `20` | 每页大小，范围 `1..100`。 |
| `sort` | string | 否 | `RECOMMENDED` | `RECOMMENDED`、`HOT` 或 `TIME`。 |

## Body / Multipart 字段

无。

## 成功响应 `TopLevelCommentPageResp`

`data` 为 `TopLevelCommentPage`，`CommentItem` 字段见 `services/zhicore-comment/api/http/README.md`。

示例：

```json
{
  "code": 200,
  "message": "OK",
  "data": {
    "items": [
      {
        "postId": "p1K8x9Q2",
        "commentId": "c1K8x9Q2",
        "author": {
          "publicId": "u_8x7K2m",
          "displayName": "azhi"
        },
        "content": "第一条评论",
        "status": "NORMAL",
        "stats": {
          "likeCount": 0,
          "replyCount": 0
        },
        "createdAt": "2026-06-27T10:30:00Z",
        "updatedAt": "2026-06-27T10:30:00Z"
      }
    ],
    "page": 1,
    "size": 20,
    "totalComments": 3,
    "totalTopLevelComments": 1,
    "pages": 1
  },
  "timestamp": 1782112892184
}
```

空列表返回 `items: []`，不返回 `null`。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | `Invalid request` | `postId` 格式非法、`page` / `size` 越界、`sort` 非法。 |
| `1004` | `503` | `Service unavailable` | Content / PostgreSQL / Redis 依赖不可用且无法降级。User 摘要或 File URL 解析失败时查询可降级。 |
| `4001` | `404` | `Post not found` | Content 返回文章不存在或不可见。 |

## 权限和可见性

- 匿名可读取公开可见文章的未删除顶级评论。
- 登录用户读取时默认返回 `viewer.liked`；匿名用户省略 `viewer`，前端按未点赞态展示。
- 只返回根评论，即 `root_id IS NULL AND parent_id IS NULL`。
- 已删除评论不进入列表；如果未来需要展示删除占位，必须单独登记 contract。

## 排序、分页和过滤

- Page 分页从 `1` 开始。
- `RECOMMENDED` 排序：`recommendedScore DESC, commentId DESC`。`recommendedScore` 来自 `comment_recommended_rank`，首版公式为 `likeCount * 100 + freshnessBoost`；这里的 `commentId` 指内部 `comments.id` 排序锚点，对外 cursor 不透明。
- `TIME` 排序：`commentId DESC`。内部 `comments.id` 作为稳定时间锚点；`createdAt` 只作为展示和审计字段返回。
- `HOT` 排序：`likeCount DESC, commentId ASC`。同点赞数下优先展示更早评论。
- `RECOMMENDED` 查询先从 `comment_recommended_rank` 按 `(post_id, recommended_score DESC, comment_id DESC)` 取一页 `comment_id`，再批量加载 `comments`、`comment_stats` 和作者摘要。
- `HOT` 查询先从 `comment_hot_rank` 按 `(post_id, like_count DESC, comment_id ASC)` 取一页 `comment_id`，再批量加载 `comments`、`comment_stats` 和作者摘要，避免大范围 `comments + stats` 排序 join。
- `likeCount` 来自异步更新的读模型，允许短暂最终一致；`viewer.liked` 如果返回，必须以 `comment_likes` 为强一致事实。
- `size` 最大 `100`。
- `totalComments` 来自 `comment_post_stats.total_comments`，统计根评论和回复的全部未删除评论。
- `totalTopLevelComments` 来自 `comment_post_stats.total_top_level_comments`，只统计未删除根评论，并用于计算 `pages`。
- 本 endpoint 不返回回复列表；回复预览如需支持，必须新增字段并在 contract 中登记 `replyLimit`。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `ListTopLevelCommentsByPage` |
| 查询模型 | `RECOMMENDED` 走 `comment_recommended_rank(post_id, recommended_score DESC, comment_id DESC)`；`TIME` 走 `comments(post_id, id DESC)`；`HOT` 走 `comment_hot_rank(post_id, like_count DESC, comment_id ASC)` 后批量补评论和统计；User 批量补作者摘要。 |
| Ports | 首切必需 `CommentQueryRepository`、`CommentPostStatsRepository`、`CommentRecommendedRankRepository`、`UserProfileClient`；登录用户 `viewer.liked` 需要 `CommentLikeRepository` 或等价批量查询能力。 |
| 事务边界 | 只读查询，不开启业务写事务。 |
| 事件 | 无。 |
| 缓存 | 首切可直接回源 PostgreSQL 并批量补作者摘要；后续可加入文章顶级评论列表缓存。 |

## 测试要求

- Handler contract test：待补，覆盖默认 `RECOMMENDED`、`size` 上限、非法 `sort`、空列表、匿名读取、登录用户默认 `viewer.liked`、`totalComments` / `totalTopLevelComments` 和 envelope。
- Application test：待补，覆盖 `RECOMMENDED` 排序、`TIME` 排序稳定性、`HOT` 排序 tie-breaker、作者摘要批量查询、作者摘要降级和缓存 miss 回源。
- System HTTP test：待补。
