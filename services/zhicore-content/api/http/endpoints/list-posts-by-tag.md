# list-posts-by-tag

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- 应用与端口：`docs/architecture/services/content/application-and-ports.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`
- Go handler：`services/zhicore-content/api/http/taxonomy_handlers.go`
- Go contract test：`services/zhicore-content/api/http/taxonomy_handler_test.go`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/tags/{slug}/posts` |
| 兼容别名 | 无 |
| Content-Type | 无 |
| 鉴权 | 匿名 |
| 幂等 | 无 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `slug` | string | 是 | 标签 slug。 |

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `cursor` | string | 否 | 无 | 不透明 cursor。 |
| `limit` | int | 否 | `20` | 最大 `100`。 |

## 成功响应 `data`

`CursorPage<PostSummary>`，字段同 `GET /api/v1/posts`。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `slug`、`cursor` 或 `limit` 非法。 |
| `4012` | `404` | 分类不存在 | 标签不存在。 |
| `1004` | `503` | 服务暂时不可用 | PostgreSQL 不可用。 |

## 排序、分页和过滤

- 只返回公开 `PUBLISHED` 文章。
- Cursor 分页，排序为 `published_at DESC, public_id DESC`。

## 测试要求

- Handler contract test：覆盖成功分页和标签不存在。

状态：已验证。
