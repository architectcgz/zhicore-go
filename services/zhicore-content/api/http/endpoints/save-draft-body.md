# 保存草稿正文

状态：草案。本文固定编辑器服务端保存正文的 Go-first HTTP contract，尚未由 Go handler / contract test 验证。

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- Body 存储与发布设计：`docs/architecture/services/content/body-storage-and-publishing.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`
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
| `4014` | `400` | 正文过大 | canonical JSON 超过限制。 |
| `4015` | `400` | 正文类型不允许 | block 类型不允许发布或保存。 |
| `4017` | `409` | 文章版本冲突 | `basePostVersion` 落后。 |
| `4019` | `409` | 正文 hash 冲突 | `baseDraftBodyHash` 与服务端当前草稿不一致。 |
| `4020` | `400` | 客户端保存时间非法 | `clientSavedAt` 格式非法。 |
| `4021` | `400` | 媒体引用非法 | File 引用缺失、不可访问或类型不允许。 |
| `4022` | `400` | 外部链接非法 | external embed URL 或 provider 不允许。 |
| `4024` | `500` | 正文存储不可用 | MongoDB body 写入或读取失败。 |

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

- Handler contract test：待补，覆盖作者鉴权、`basePostVersion`、`baseDraftBodyId`、`baseDraftBodyHash`、正文 schema、媒体引用和成功 envelope。
- System HTTP test：待补。
