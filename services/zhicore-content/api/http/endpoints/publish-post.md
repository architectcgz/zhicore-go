# 发布文章

状态：草案。本文固定编辑器发布入口的 Go-first HTTP contract，尚未由 Go handler / contract test 验证。

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- Body 存储与发布设计：`docs/architecture/services/content/body-storage-and-publishing.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`
- 大草案：`services/zhicore-content/api/http/endpoints/content-api.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/posts/{postId}/publish` |
| 兼容别名 | 无 |
| Content-Type | `application/json` |
| 鉴权 | 作者 |
| 幂等 | 无业务幂等键；重复提交依赖 `basePostVersion`、`draftBodyId`、`draftBodyHash` 和当前发布状态返回冲突 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |

## Query 参数

无。

## Body 字段 `PublishPostReq`

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `basePostVersion` | int | 是 | 不允许为空 | 发布确认时看到的 post 版本。 |
| `draftBodyId` | string | 是 | 不允许为空 | 要发布的草稿 body ID。 |
| `draftBodyHash` | string | 是 | 不允许为空 | 要发布的草稿 body hash，格式 `sha256:<hex>`。 |

## 成功响应 `PublishPostResp`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |
| `postVersion` | int | 是 | 发布后的新 post 版本。 |
| `publishedAt` | string | 是 | 服务端发布时间，RFC3339。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2006` | `401` | 请先登录 | 缺少可信 `X-User-Id`。 |
| `2008` | `403` | 无权访问该资源 | 当前用户不是作者。 |
| `4001` | `404` | 文章不存在 | post 不存在、已删除或不可编辑。 |
| `4002` | `409` | 文章已发布 | 重复发布。 |
| `4005` | `400` | 文章标题不能为空 | 发布时标题为空。 |
| `4006` | `400` | 文章内容不能为空 | 发布时正文有效内容为空。 |
| `4016` | `400` | 正文有效文本不足 | 发布时正文有效 rune 数低于最小要求。 |
| `4017` | `409` | 草稿冲突 | `basePostVersion`、`draftBodyId` 或 `draftBodyHash` 与服务端当前草稿不一致。 |
| `4018` | `500` | 正文不可用 | 待发布 body 缺失或需 repair。 |
| `4019` | `409` | 正文 hash 冲突 | `draftBodyHash` 不匹配。 |
| `4021` | `400` | 媒体引用非法 | File 引用不满足发布要求。 |
| `4023` | `400` | 封面不可用 | 草稿封面引用已经不可用或不可发布。 |
| `1004` | `503` | 发布依赖不可用 | 发布高副作用路径依赖不可确认，不能 fail-open。 |

## 权限和可见性

- 只有作者可发布。
- 发布成功后公开读接口可读取 `PUBLISHED` 内容。

## 排序、分页和过滤

无。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `PublishPost` |
| 聚合 | Post published pointer + PostBody snapshot |
| 事务边界 | PG `published_*` 指针和 MongoDB body snapshot 必须一起成功或有明确 repair / 补偿路径。 |
| 事件 | 发布成功后应由 Content outbox 产生后续搜索、排名、通知等事件，具体事件 contract 单独登记。 |

## 测试要求

- Handler contract test：待补，覆盖作者鉴权、版本冲突、正文 hash 冲突、标题/正文为空、成功 envelope 和重复提交。
- System HTTP test：待补。
