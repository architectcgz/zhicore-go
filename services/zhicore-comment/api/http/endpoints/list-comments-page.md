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

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `page` | int | 否 | `1` | 页码，从 `1` 开始。 |
| `size` | int | 否 | `20` | 每页大小，范围 `1..100`。 |
| `sort` | string | 否 | `TIME` | `TIME` 或 `HOT`。`HOT` 公式由 Comment 查询层拥有。 |
| `includeViewer` | boolean | 否 | `false` | 登录用户可传 `true` 返回 `viewer.liked`；匿名请求传 `true` 按 `false` 处理。 |

## Body / Multipart 字段

无。

## 成功响应 `data`

`data` 为 `Page<CommentItem>`，`CommentItem` 字段见 `services/zhicore-comment/api/http/README.md`。

示例：

```json
{
  "code": 200,
  "message": "操作成功",
  "data": {
    "items": [
      {
        "postId": "p1K8x9Q2",
        "floor": 26,
        "author": {
          "userId": "u1001",
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
    "total": 1,
    "pages": 1
  },
  "timestamp": 1782112892184
}
```

空列表返回 `items: []`，不返回 `null`。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `postId` 格式非法、`page` / `size` 越界、`sort` 非法。 |
| `1004` | `503` | 服务暂时不可用 | Content / User / PostgreSQL / Redis 依赖不可用且无法降级。 |
| `4001` | `404` | 文章不存在 | Content 返回文章不存在或不可见。 |

## 权限和可见性

- 匿名可读取公开可见文章的未删除顶级评论。
- 登录用户读取时可返回 `viewer.liked`。
- 只返回根评论，即 `root_id IS NULL AND parent_id IS NULL`。
- 已删除评论默认不进入列表；如果未来需要展示“评论已删除”占位，必须单独登记 contract。

## 排序、分页和过滤

- Page 分页从 `1` 开始。
- `TIME` 排序：`createdAt DESC, floor DESC`，保证稳定排序。
- `HOT` 排序：按 Comment 查询层固定公式排序，必须有稳定 tie-breaker；第一阶段可先落回 `likeCount DESC, createdAt DESC, floor DESC`。
- `size` 最大 `100`。
- 本 endpoint 不返回回复列表；回复预览如需支持，必须新增字段并在 contract 中登记 `replyLimit`。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `ListTopLevelCommentsByPage` |
| 查询模型 | `CommentQueryRepository` 返回根评论列表和统计；User 批量补作者摘要。 |
| Ports | `CommentQueryRepository`、`CommentListCacheStore`、`UserProfileClient`、`CommentLikeRepository` / `CommentLikeCacheStore` |
| 事务边界 | 只读查询，不开启业务写事务。 |
| 事件 | 无。 |
| 缓存 | 可使用文章顶级评论列表缓存；缓存 miss 时回源 PostgreSQL 并批量补作者摘要。 |

## 测试要求

- Handler contract test：待补，覆盖默认分页、`size` 上限、非法 `sort`、空列表、匿名读取、登录用户 `viewer.liked` 和 envelope。
- Application test：待补，覆盖 `TIME` 排序稳定性、`HOT` 排序 tie-breaker、作者摘要批量查询和缓存 miss 回源。
- System HTTP test：待补。
