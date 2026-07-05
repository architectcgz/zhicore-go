# 批量获取公开文章摘要

状态：已验证。本文固定 `POST /api/v1/posts/batch-get` 的 Go-first HTTP contract，已由 application / handler / repository test 覆盖最多 100 个 ID、公开可见性和 `missingPostIds` 语义。

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/posts/batch-get` |
| 鉴权 | 匿名 / 服务间 |
| Content-Type | `application/json` |

## Body 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postIds` | string[] | 是 | 最多 100 个公开文章 ID。 |
| `includeDeleted` | boolean | 否 | 公开匿名调用忽略该字段；管理端维护语义后续单独登记。 |

## 成功响应

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `items` | `PostSummary[]` | 是 | 可见的 published 文章摘要，按请求 ID 顺序返回。 |
| `missingPostIds` | string[] | 是 | 不存在、未发布、已删除或不可见的 ID。 |

## 错误响应

| code | HTTP status | 触发条件 |
| --- | --- | --- |
| `1001` | `400` | body 非法、`postIds` 为空或超过 100。 |
| `1004` | `503` | PostgreSQL 等依赖不可用。 |

## 测试

- Handler contract test：`services/zhicore-content/api/http/public_post_queries_handler_test.go`
- Application test：`services/zhicore-content/internal/content/application/public_post_queries_test.go`
- Repository test：`services/zhicore-content/internal/content/infrastructure/postgres/post_queries_test.go`
