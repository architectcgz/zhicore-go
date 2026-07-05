# 保存草稿正文

状态：已验证。本文固定编辑器服务端保存正文的 Go-first HTTP contract，已由 Go handler contract test 验证本切片指定的路由、身份、DTO、request body limit、envelope 和核心错误码；媒体依赖错误待 application / ports 固定 sentinel 后补测。

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- Body 存储与发布设计：`docs/architecture/services/content/body-storage-and-publishing.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`
- Go handler：`services/zhicore-content/api/http/handler.go`
- Go contract test：`services/zhicore-content/api/http/save_draft_body_handler_test.go`
- 大草案：`services/zhicore-content/api/http/endpoints/content-api.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `PUT` |
| 主路径 | `/api/v1/posts/{postId}/draft/body` |
| 兼容别名 | 无 |
| Content-Type | `application/json` |
| 鉴权 | 作者，必须由 Gateway 注入 `X-User-Id` 后由 Content 校验 owner |
| 幂等 | 非幂等；依赖 `basePostVersion`、`baseDraftBodyId`、`baseDraftBodyHash` 做乐观并发控制 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |

## Query 参数

无。

## Body 字段 `SaveDraftBodyReq`

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `basePostVersion` | int | 是 | 不允许为空 | 调用方保存前看到的 post 乐观锁版本。 |
| `baseDraftBodyId` | string | 否 | 空草稿或首次保存可缺失 | 调用方保存前看到的草稿 body ID。 |
| `baseDraftBodyHash` | string | 否 | 空草稿或首次保存可缺失 | 调用方保存前看到的草稿 body hash；必须是服务端返回的 `sha256:` hash。 |
| `schemaVersion` | int | 是 | 不允许为空 | 当前为 `1`。 |
| `blocks` | object[] | 是 | 空数组表示空正文草稿 | 结构化正文 blocks；不接受 raw HTML 作为可信正文。 |
| `clientSavedAt` | string | 否 | 缺失表示不提供客户端保存时间 | RFC3339，仅用于冲突提示，不作为服务端事实源。 |

## 成功响应 `SaveDraftBodyResp`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |
| `postVersion` | int | 是 | 保存成功后的新 post 版本。 |
| `draftBodyId` | string | 是 | 新草稿 body ID。 |
| `draftBodyHash` | string | 是 | 新草稿 body hash，格式 `sha256:<hex>`。 |
| `savedAt` | string | 是 | 服务端保存时间，RFC3339。 |
| `wordCount` | int | 是 | 服务端 canonicalize 后统计的字数。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2006` | `401` | 请先登录 | 缺少可信 `X-User-Id`。 |
| `2008` | `403` | 无权访问该资源 | 当前用户不是作者。 |
| `4001` | `404` | 文章不存在 | post 不存在、已删除或不可编辑。 |
| `4004` | `409` | 文章已删除 | 已删除文章不允许保存草稿。 |
| `4013` | `400` | 正文 schema 非法 | blocks schema 不合法。 |
| `4014` | `400` | 正文类型不允许 | block 类型不允许发布或保存。 |
| `4015` | `400` | 正文过大 | canonical JSON 超过限制。 |
| `4015` | `413` | 请求体过大 | HTTP request body 超过 `512KB`，在进入 application parser 前拒绝。 |
| `4017` | `409` | 草稿冲突 | `basePostVersion`、`baseDraftBodyId` 或 `baseDraftBodyHash` 与服务端当前草稿不一致。 |
| `4019` | `409` | 正文 hash 冲突 | 服务端读取到的 body hash 与 PostgreSQL 指针记录不一致。 |
| `1001` | `400` | 参数校验失败 | `clientSavedAt` 格式非法或普通字段校验失败。 |
| `4021` | `400` | 媒体引用非法 | File 引用缺失、不可访问或类型不允许。 |
| `4020` | `400` | external embed provider 不允许 | `external_embed.provider` 不在白名单内。 |
| `4022` | `400` | 校验错误过多 | 字段级或 block 级错误超过返回上限。 |
| `4024` | `400` | 正文 schema 版本不支持 | 请求的 `schemaVersion` 当前服务不可写。 |
| `1004` | `503` | 服务暂时不可用 | MongoDB body 写入、File 校验或限流依赖不可用。 |

## 权限和可见性

- 只有作者可保存草稿正文。
- 保存草稿不改变 published body；公开读仍读取已发布指针。

## 排序、分页和过滤

无。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `SaveDraftBody` |
| 聚合 | Post draft pointer + PostBody |
| 事务边界 | 使用 copy-on-write 新建 body，再推进草稿指针和 post version；失败时不能让公开正文指针漂移。 |
| 事件 | 当前草稿保存不对外发布事件。 |

## 测试要求

- Parser unit test：待补，覆盖 `V1BodyParser` 正文 schema、未启用 block、外链 scheme、external embed provider、媒体引用提取、容器深度、表格尺寸、字段长度、canonical JSON 大小和错误数量截断。
- Parser benchmark：待补，覆盖 `small`、`medium`、`near_limit`、`many_blocks`、`large_table`、`many_links`、`large_code`、`reject_oversize`、`reject_many_errors`，用于确定正文阈值；基准命令为 `go test -bench=BenchmarkV1BodyParser -benchmem ./services/zhicore-content/...`。
- Handler contract test：`services/zhicore-content/api/http/save_draft_body_handler_test.go`，覆盖作者鉴权、`basePostVersion`、正文 schema、正文过大、超大请求体拒绝、request context cancel 和成功 envelope。
- 待补 handler contract test：`4021` 媒体引用非法；需要 application / ports 先固定可分支语义错误。
- System HTTP test：待补。
- Autosave load test：待补，模拟 10 / 50 / 100 个作者并发保存，正文分布使用 `70% small`、`25% medium`、`5% near_limit`，记录 p95 / p99、错误码分布、CPU、GC、MongoDB 写入耗时和限流命中率。
