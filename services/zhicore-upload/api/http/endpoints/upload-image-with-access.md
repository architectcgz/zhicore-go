# 上传图片并指定访问级别

## 来源

- 服务设计：`docs/architecture/services/upload/README.md`
- 当前 API schema：`services/zhicore-upload/api/http/README.md`
- Go handler：`services/zhicore-upload/api/http/handler.go`
- Go contract test：`services/zhicore-upload/api/http/handler_test.go`
- Java 参考：`../zhicore-microservice/zhicore-upload/src/main/java/com/zhicore/upload/controller/FileUploadController.java`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/upload/image/with-access` |
| 兼容别名 | 无 |
| Content-Type | `multipart/form-data` |
| 鉴权 | 匿名 / 服务间；不读取 `Authorization` 或 `X-User-Id` |
| 幂等 | 无；秒传由外部 File Service 决定 |

## Path 参数

无。

## Query 参数

无。

## Body / Multipart 字段

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `file` | file | 是 | 不允许为空 | 图片文件。当前默认允许 `image/jpeg`、`image/jpg`、`image/png`、`image/gif`、`image/webp`，最大 50 MiB。 |
| `accessLevel` | string | 是 | 不允许为空 | `PUBLIC` 或 `PRIVATE`；Go handler 接受大小写输入并归一为大写。 |

## 成功响应 `data`

`data` 为 `UploadFile`，字段见 `services/zhicore-upload/api/http/README.md`。`accessLevel` 必须等于请求归一后的访问级别。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `400` | `400` | `文件不能为空` | multipart 缺少 `file` 或文件为空。 |
| `400` | `400` | `accessLevel 不能为空` | multipart 缺少 `accessLevel` 或值为空白。 |
| `400` | `400` | `访问级别必须是 PUBLIC 或 PRIVATE` | `accessLevel` 不是允许值。 |
| `400` | `400` | `文件类型不允许: <contentType>` | `file` MIME type 不在图片允许列表。 |
| `400` | `400` | `文件大小超过限制` | 文件超过图片大小限制。 |
| `500` | `500` | `系统内部错误，请稍后重试` | 未分类服务端错误。 |

`400` / `500` 是当前 Go handler 实际输出的兼容例外。目标业务码见服务级 README。

## 权限和可见性

- 本 endpoint 不校验业务 owner。
- `PUBLIC` 文件可直接返回公开 URL；`PRIVATE` 文件 URL 的签名、过期和访问控制由 File Service 决定。
- 业务服务保存 `fileId` 后，自行决定该文件能否被当前业务资源引用。

## 排序、分页和过滤

无。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `UploadImage(file, accessLevel)` |
| Application owner | `services/zhicore-upload/internal/upload/application.Service` |
| Port | `ports.FileService.Upload` |
| 事务边界 | 无本地事务；外部 File Service 拥有文件元数据。 |
| 事件 | 当前不发布事件。 |

## 测试要求

- Handler contract test：`TestUploadImageWithAccessPassesPrivateAccess`、`TestUploadImageWithAccessRequiresAccessLevel`。
- System HTTP test：待补。
