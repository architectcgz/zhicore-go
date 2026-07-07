# get-post-engagement

状态：草案。

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- Engagement 设计：`docs/architecture/services/content/engagement-design.md`
- 运行期 resilience：`docs/architecture/services/content/runtime-resilience.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/posts/{postId}/engagement` |
| 兼容别名 | 无 |
| Content-Type | 无 |
| 鉴权 | 匿名 / 登录用户 |
| 幂等 | 无副作用。 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | 文章公开 ID。 |
| `stats` | `PostStats` | 是 | 文章统计。 |
| `viewer` | `EngagementViewer` | 登录用户必填，匿名省略 | 当前用户互动状态。 |

`PostStats`：`viewCount`、`likeCount`、`favoriteCount`、`commentCount`。

`EngagementViewer`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `liked` | boolean \| null | 是 | `null` 表示状态不可确认，不等于 `false`。 |
| `favorited` | boolean \| null | 是 | `null` 表示状态不可确认，不等于 `false`。 |
| `degraded` | bool | 是 | `true` 表示 viewer 状态因 Redis / fallback 降级不可确认。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `postId` 为空或格式非法。 |
| `1004` | `503` | 服务暂时不可用 | 统计读取或整体查询依赖不可用。 |
| `4001` | `404` | 文章不存在 | 文章不存在、已删除或不可见。 |

## 降级语义

- 匿名请求只返回 `stats`，不返回 `viewer`。
- 登录请求优先读取缓存，cache miss 可批量回源 PostgreSQL。
- Redis 不可用且 DB fallback 预算耗尽时，`stats` 仍可返回，`viewer.liked=null`、`viewer.favorited=null`、`viewer.degraded=true`。
- 不得把 unknown viewer 状态伪装成 `false`。

## 测试要求

- Application test：Redis 不可用且 fallback 成功返回确定 viewer；fallback 不可用返回 null + degraded。
- Handler contract test：匿名无 viewer、登录确定 viewer、登录 degraded viewer。
