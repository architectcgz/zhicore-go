# delete-post-tag

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- 应用与端口：`docs/architecture/services/content/application-and-ports.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`
- Go handler：`services/zhicore-content/api/http/taxonomy_handlers.go`
- Go contract test：`services/zhicore-content/api/http/taxonomy_handler_test.go`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `DELETE` |
| 主路径 | `/api/v1/posts/{postId}/tags/{slug}` |
| 兼容别名 | 无 |
| Content-Type | 无 |
| 鉴权 | 作者 |
| 幂等 | 删除不存在的已关联标签返回成功并保持集合不变；标签 slug 本身不存在返回 `4012`。 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |
| `slug` | string | 是 | 标签 slug。 |

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `basePostVersion` | int64 | 是 | 无 | 乐观锁版本，必须 `> 0`。 |

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | 文章公开 ID。 |
| `postVersion` | int64 | 是 | 更新后的版本。 |
| `tags` | `Tag[]` | 是 | 删除后的标签列表。 |
| `updatedAt` | string | 是 | RFC3339 UTC。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | path 或 `basePostVersion` 非法。 |
| `2006` | `401` | 请先登录 | 缺少可信 `X-User-Id`。 |
| `2008` | `403` | 无权访问该资源 | 当前用户不是作者。 |
| `4001` | `404` | 文章不存在 | 文章不存在。 |
| `4004` | `409` | 文章已删除 | 操作已删除文章。 |
| `4012` | `404` | 分类不存在 | 标签 slug 不存在。 |
| `4017` | `409` | 草稿版本冲突 | `basePostVersion` 过期。 |
| `1004` | `503` | 服务暂时不可用 | PostgreSQL 不可用。 |

## 权限和可见性

- application 必须校验 `posts.owner_id == Actor.UserID`。

## 排序、分页和过滤

无分页；响应保持剩余关系顺序。

## 测试要求

- Handler contract test：覆盖成功、缺登录、slug 不存在、版本冲突和参数错误。

状态：已验证。
