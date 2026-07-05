# 公开文章详情

状态：已验证。本文固定 `GET /api/v1/posts/{postId}` 的 Go-first HTTP contract，已由 application / handler / repository test 覆盖 published 可见性、正文读取、body miss / hash / schema 错误映射和成功 envelope。

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/posts/{postId}` |
| 鉴权 | 匿名 |
| Content-Type | 无 body |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |

## 成功响应

`data` 为 `PostDetail`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `post` | `PostSummary` | 是 | 文章摘要和统计。 |
| `body` | `PostBody` | 否 | 公开详情默认内联 published body。 |
| `tags` | `Tag[]` | 否 | 待任务 8 标签关系落地后返回。 |

## 错误响应

| code | HTTP status | 触发条件 |
| --- | --- | --- |
| `4001` | `404` | 文章不存在、已删除、未发布或匿名不可见。 |
| `4018` | `500` | published body 指针存在但正文缺失。 |
| `4019` | `409` | body hash 校验失败。 |
| `4024` | `500` | stored body schema 当前服务不可读。 |
| `1004` | `503` | PostgreSQL / MongoDB 等依赖不可用。 |

## 测试

- Handler contract test：`services/zhicore-content/api/http/public_post_queries_handler_test.go`
- Application test：`services/zhicore-content/internal/content/application/public_post_queries_test.go`
- Repository test：`services/zhicore-content/internal/content/infrastructure/postgres/post_queries_test.go`
