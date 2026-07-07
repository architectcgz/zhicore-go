# get-post-tags

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
| 主路径 | `/api/v1/posts/{postId}/tags` |
| 兼容别名 | 无 |
| Content-Type | 无 |
| 鉴权 | 匿名 |
| 幂等 | 无 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |

## 成功响应 `data`

`Tag[]`，按 `post_tags.position ASC, tags.slug ASC` 返回。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `postId` 为空。 |
| `4001` | `404` | 文章不存在 | 文章不存在、已删除或匿名不可见。 |
| `1004` | `503` | 服务暂时不可用 | PostgreSQL 不可用。 |

## 排序、分页和过滤

无分页；单篇文章最多 10 个标签。

## 测试要求

- Handler contract test：覆盖成功数组和文章不存在。

状态：已验证。
