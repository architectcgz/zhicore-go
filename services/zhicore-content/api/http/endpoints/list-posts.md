# 公开文章列表

状态：已验证。本文固定 `GET /api/v1/posts` 的 Go-first HTTP contract，已由 application / handler / repository test 覆盖公开 published 可见性、默认 limit、上限、cursor 透传和稳定排序。

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/posts` |
| 鉴权 | 匿名 |
| Content-Type | 无 body |

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `authorId` | string | 否 | 无 | 作者过滤，当前映射为 Content `owner_id`。 |
| `cursor` | string | 否 | 无 | Opaque cursor，consumer 不解析。 |
| `limit` | int | 否 | `20` | `1..100`，超过上限按 `100`。 |
| `sort` | string | 否 | `latest` | 第一阶段只支持 `latest`。 |
| `tag` | string | 否 | 无 | 标签过滤，任务 8 前传入会返回 `1001`。 |
| `categoryId` | string | 否 | 无 | 分类过滤，任务 8 前传入会返回 `1001`。 |

## 成功响应

`data` 为 `CursorPage<PostSummary>`：`items`、`nextCursor`、`hasMore`、`limit`。

排序固定为 `published_at DESC, public_id DESC`，cursor 内部锚点为 `publishedAt + postId`。

## 错误响应

| code | HTTP status | 触发条件 |
| --- | --- | --- |
| `1001` | `400` | query 参数、cursor 或 sort 非法。 |
| `1004` | `503` | PostgreSQL 等依赖不可用。 |

## 测试

- Handler contract test：`services/zhicore-content/api/http/public_post_queries_handler_test.go`
- Application test：`services/zhicore-content/internal/content/application/public_post_queries_test.go`
- Repository test：`services/zhicore-content/internal/content/infrastructure/postgres/post_queries_test.go`
