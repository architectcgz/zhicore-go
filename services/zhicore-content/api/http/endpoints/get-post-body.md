# 读取文章正文

状态：已验证。本文固定公开 published body 读取入口的 Go-first HTTP contract，已由 Go handler contract test 验证。

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- Body 存储与发布设计：`docs/architecture/services/content/body-storage-and-publishing.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`
- Go handler：`services/zhicore-content/api/http/handler.go`
- Go contract test：`services/zhicore-content/api/http/get_post_body_handler_test.go`
- 大草案：`services/zhicore-content/api/http/endpoints/content-api.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/posts/{postId}/body` |
| 兼容别名 | 无 |
| Content-Type | 无 body |
| 鉴权 | 匿名 / 服务间 |
| 幂等 | 读取接口，幂等 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |

## Query 参数

无。

## Body 字段

无。

## 成功响应 `PostBodyResp`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `bodyId` | string | 是 | Content body UUID。 |
| `schemaVersion` | int | 是 | 当前为 `1`。 |
| `format` | string | 是 | 固定 `blocks`。 |
| `blocks` | object[] | 是 | 结构化正文 blocks。 |
| `plainText` | string | 是 | 后端 canonicalize 后提取的纯文本。 |
| `contentHash` | string | 是 | `sha256:<hex>`。 |
| `sizeBytes` | int | 是 | canonical JSON 字节数。 |
| `createdAt` | string | 是 | RFC3339。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `4001` | `404` | 文章不存在 | 文章不存在、已删除或匿名不可见。 |
| `4018` | `500` | 正文不可用 | published body 指针存在但 body 缺失，需要 repair。 |
| `4019` | `409` | 正文 hash 冲突 | body hash 校验失败。 |
| `4024` | `500` | 正文 schema 版本不支持 | MongoDB body 的 `schemaVersion` 当前服务不可读。 |
| `1004` | `503` | 服务暂时不可用 | MongoDB body 读取超时、熔断或连接失败。 |

## 权限和可见性

- 匿名只允许读取 `PUBLISHED` 且未删除文章正文。
- 服务间调用通过 `X-Caller-Service` / `X-Caller-Operation` 标识调用来源，不绕过可见性规则，除非未来 typed client contract 明确登记维护场景。

## 排序、分页和过滤

无。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `GetPublishedPostBody` |
| 聚合 | Post published pointer + PostBody |
| 事务边界 | 读取路径不创建事务；body miss 应返回错误并由 repair 流程处理。 |
| 事件 | 无。 |

## 测试要求

- Handler contract test：`services/zhicore-content/api/http/get_post_body_handler_test.go`，覆盖公开可见、草稿不可见、已删除不可见、body miss、hash 冲突、schema 不可读、malformed canonical body 防静默空正文和成功 envelope。
- System HTTP test：待补。
