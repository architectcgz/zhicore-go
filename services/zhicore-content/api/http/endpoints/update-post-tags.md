# update-post-tags

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- 应用与端口：`docs/architecture/services/content/application-and-ports.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`
- Go handler：`services/zhicore-content/api/http/taxonomy_handlers.go`
- Go contract test：`services/zhicore-content/api/http/taxonomy_handler_test.go`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `PUT` |
| 主路径 | `/api/v1/posts/{postId}/tags` |
| 兼容别名 | 无 |
| Content-Type | `application/json` |
| 鉴权 | 作者 |
| 幂等 | 完全替换为相同标签集合时幂等 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |

## Body 字段

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `basePostVersion` | int64 | 是 | 不允许为空 | 乐观锁版本，必须 `> 0`。 |
| `tags` | string[] | 是 | 空数组表示清空标签 | 最多 10 个；slug 会 trim、小写、去重后保存。 |

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | 文章公开 ID。 |
| `postVersion` | int64 | 是 | 更新后的版本。 |
| `tags` | `Tag[]` | 是 | 更新后的标签列表。 |
| `updatedAt` | string | 是 | RFC3339 UTC。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | body、版本或 tag 格式非法。 |
| `2006` | `401` | 请先登录 | 缺少可信 `X-User-Id`。 |
| `2008` | `403` | 无权访问该资源 | 当前用户不是作者。 |
| `4001` | `404` | 文章不存在 | 文章不存在。 |
| `4004` | `409` | 文章已删除 | 操作已删除文章。 |
| `4012` | `404` | 分类不存在 | 任一 tag slug 不存在。 |
| `4017` | `409` | 草稿版本冲突 | `basePostVersion` 过期。 |
| `1004` | `503` | 服务暂时不可用 | PostgreSQL 不可用。 |

## 权限和可见性

- application 必须校验 `posts.owner_id == Actor.UserID`。
- 不从 body 接收操作者 ID。

## 排序、分页和过滤

无分页；响应保持请求中规范化后的标签顺序。

## 测试要求

- Handler contract test：覆盖作者身份、重复 tag 去重、缺登录、非作者、slug 不存在、版本冲突和错误码。

状态：已验证。
