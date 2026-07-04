# 创建文章草稿

状态：草案。本文从 `content-api.md` 拆出编辑器最小闭环的创建入口，尚未由 Go handler / contract test 验证。

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- Body 存储与发布设计：`docs/architecture/services/content/body-storage-and-publishing.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`
- 大草案：`services/zhicore-content/api/http/endpoints/content-api.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/posts` |
| 兼容别名 | 无 |
| Content-Type | `application/json` |
| 鉴权 | 登录用户，必须由 Gateway 注入 `X-User-Id` |
| 幂等 | 无；重复调用会创建不同草稿 |

## Path 参数

无。

## Query 参数

无。

## Body 字段 `CreatePostReq`

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `title` | string | 否 | 缺失或空字符串表示未填写标题 | 草稿阶段可空；发布时必填，最大 200。 |
| `summary` | string | 否 | 缺失表示未填写摘要 | 用户摘要。 |
| `coverFileId` | string | 否 | 缺失表示无封面 | File 文件引用，不保存 URL。 |
| `topicId` | string | 否 | 缺失表示无话题 | 话题引用，拆 Topic 服务前由 Content 管理。 |
| `categoryId` | string | 否 | 缺失表示无分类 | 分类引用。 |
| `tags` | string[] | 否 | 缺失按空列表处理 | 最多 10 个标签 slug 或名称。 |
| `body` | object | 否 | 缺失时创建空草稿占位 | `schemaVersion + blocks`，结构同 `SaveDraftBodyReq` 中正文写入字段。 |

## 成功响应 `CreatePostResp`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |
| `postVersion` | int | 是 | 初始乐观锁版本，通常为 `1`。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2006` | `401` | 请先登录 | 缺少可信 `X-User-Id`。 |
| `1001` | `400` | 参数校验失败 | body schema、tags 或正文 blocks 非法。 |
| `4007` | `400` | 文章标题过长 | 标题超过限制。 |
| `4012` | `404` | 分类不存在 | 分类、话题或标签引用不存在。 |
| `4013` | `400` | 正文 schema 非法 | blocks schema 不合法。 |
| `4021` | `400` | 媒体引用非法 | File 引用缺失、不可访问或类型不允许。 |

## 权限和可见性

- 当前登录用户成为草稿 owner。
- 草稿创建后只允许作者和未来管理员路径读取或修改。

## 排序、分页和过滤

无。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `CreatePost` |
| 聚合 | Post + Draft body pointer |
| 事务边界 | PostgreSQL post meta 与可选 MongoDB body 初始写入必须由 Content application 定义一致性边界。 |
| 事件 | 首阶段可不发布公开事件；后续草稿审计或 outbox 由 Content 设计补充。 |

## 测试要求

- Handler contract test：待补，覆盖登录态、空草稿、带 body 创建、标题过长和非法媒体引用。
- System HTTP test：待补。
