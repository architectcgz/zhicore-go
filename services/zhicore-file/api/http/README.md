# zhicore-file HTTP Schema

本目录记录 `zhicore-file` 的对外 HTTP contract。`zhicore-file` 已登记为 Go-first API reset，HTTP path 以本目录为新事实源；旧 `/api/v1/upload/...` 入口不保留兼容。

## 来源

- 服务设计：`docs/architecture/services/file/README.md`
- 通用 HTTP contract：`docs/contracts/http.md`
- 错误码规则：`docs/contracts/error-codes.md`
- Go handler：`services/zhicore-file/api/http/handler.go`
- Go contract test：`services/zhicore-file/api/http/handler_test.go`
- 历史 Java 参考：`../zhicore-microservice` 中的文件上传 controller / DTO 只作为业务能力参考，不作为 path 兼容约束。

## 公共规则

- 响应 envelope：使用 `docs/contracts/http.md` 的统一 envelope；成功响应 HTTP `200`，`body.code=200`，`message="操作成功"`。
- 当前错误形态：Go handler 现阶段通过 `libs/kit/httpapi.WriteError` 返回 `body.code=HTTP status`，例如参数错误为 `400`。后续迁移到 `1001` / `8001` / `8002` / `8003` 时必须同步更新本目录和 handler contract test。
- 时间：`timestamp` 为 Unix epoch milliseconds；上传结果 `uploadTime` 为 RFC3339 字符串，字段为空时可省略。
- ID：`fileId` 为 `zhicore-file` 生成或返回的 opaque string，调用方不得解析内部结构。
- 鉴权上下文：当前 6 个 endpoint 不从 `Authorization` 或 `X-User-Id` 解析调用者身份；文件访问权限由 `accessLevel` 和 `zhicore-file` 的 URL / delete 语义决定。后续若删除文件需要业务 owner 校验，必须先更新服务设计和本 contract。
- Multipart：上传接口使用 `multipart/form-data`；单文件字段名固定为 `file`，批量字段名固定为 `files`。
- 访问级别：`accessLevel` 只允许 `PUBLIC`、`PRIVATE`；大小写输入由 Go handler 归一为大写。未提供时，除 `/image/with-access` 外默认 `PUBLIC`。
- 文件类型和大小：图片默认允许 `image/jpeg`、`image/jpg`、`image/png`、`image/gif`、`image/webp`，最大 50 MiB；音频默认允许 `audio/mpeg`、`audio/mp3`、`audio/mp4`、`audio/x-m4a`、`audio/aac`、`audio/wav`、`audio/x-wav`、`audio/ogg`、`audio/webm`，最大 10 MiB。若运行时配置覆盖默认值，以服务配置和 handler test 为准。

## 通用上传响应 `UploadFile`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `fileId` | string | 是 | `zhicore-file` 返回的文件 ID。 |
| `url` | string | 是 | 文件访问 URL。PRIVATE 文件可为签名 URL 或受控访问 URL，具体由 `zhicore-file` 决定。 |
| `fileSize` | int | 是 | 文件大小，单位 byte。 |
| `fileHash` | string | 否 | 文件哈希；当前实现未返回时省略。 |
| `instantUpload` | boolean | 是 | 是否秒传；Go 响应固定包含该字段。 |
| `uploadTime` | string | 否 | RFC3339；外部结果没有上传时间时省略。 |
| `accessLevel` | string | 是 | `PUBLIC` 或 `PRIVATE`。 |
| `originalName` | string | 是 | 原始文件名。 |
| `contentType` | string | 是 | 上传文件 MIME type。 |

## Endpoint 索引

| 方法 | 路径 | 文档 | 状态 |
| --- | --- | --- | --- |
| `POST` | `/api/v1/files/image` | [endpoints/upload-image.md](endpoints/upload-image.md) | 已验证 |
| `POST` | `/api/v1/files/audio` | [endpoints/upload-audio.md](endpoints/upload-audio.md) | 草案 |
| `POST` | `/api/v1/files/image/with-access` | [endpoints/upload-image-with-access.md](endpoints/upload-image-with-access.md) | 已验证 |
| `POST` | `/api/v1/files/images/batch` | [endpoints/upload-images-batch.md](endpoints/upload-images-batch.md) | 草案 |
| `GET` | `/api/v1/files/{fileId}/url` | [endpoints/get-file-url.md](endpoints/get-file-url.md) | 已验证 |
| `DELETE` | `/api/v1/files/{fileId}` | [endpoints/delete-file.md](endpoints/delete-file.md) | 已验证 |

## API 到设计追踪

| Endpoint | Use case | 设计文档 | Contract 状态 | 测试状态 |
| --- | --- | --- | --- | --- |
| `POST /api/v1/files/image` | `UploadImage(file, PUBLIC)` | `docs/architecture/services/file/README.md` | 已验证 | `TestUploadImageUsesPublicAccessAndReturnsFileEnvelope`、`TestUploadImageRejectsUnsupportedContentType` |
| `POST /api/v1/files/audio` | `UploadAudio(file, PUBLIC)` | `docs/architecture/services/file/README.md` | 草案 | 待补 handler contract test |
| `POST /api/v1/files/image/with-access` | `UploadImage(file, accessLevel)` | `docs/architecture/services/file/README.md` | 已验证 | `TestUploadImageWithAccessPassesPrivateAccess`、`TestUploadImageWithAccessRequiresAccessLevel` |
| `POST /api/v1/files/images/batch` | `UploadImagesBatch(files, accessLevel)` | `docs/architecture/services/file/README.md` | 草案 | 待补 handler contract test |
| `GET /api/v1/files/{fileId}/url` | `GetFileURL(fileId)` | `docs/architecture/services/file/README.md` | 已验证 | `TestGetFileURLAndDeleteFileUsePathFileID` |
| `DELETE /api/v1/files/{fileId}` | `DeleteFile(fileId)` | `docs/architecture/services/file/README.md` | 已验证 | `TestGetFileURLAndDeleteFileUsePathFileID` |

## 服务级公开错误码

| code | HTTP status | 含义 | 适用场景 | 当前状态 |
| --- | --- | --- | --- | --- |
| `400` | `400` | 参数或上传输入错误 | multipart 缺失、`accessLevel` 缺失或非法、文件为空、类型不允许、大小超限。 | 当前 Go handler 已使用，兼容例外。 |
| `500` | `500` | 系统内部错误 | 未分类错误或对象存储 adapter 未映射错误。 | 当前 Go handler 已使用。 |
| `1001` | `400` | 参数校验失败 | 后续替换 `400` 时用于 multipart、`fileId` 或 `accessLevel` 通用参数错误。 | 待迁移。 |
| `8001` | `400` / `413` | 文件过大 | 文件大小超过业务限制或外部配额限制。 | 待迁移。 |
| `8002` | `400` | 文件类型不允许 | MIME type 不在允许列表。 | 待迁移。 |
| `8003` | `500` / `503` | 文件操作失败 | 上传、删除、哈希、对象存储调用或 URL 解析失败。 | 待迁移。 |

## 待补 contract

- 为 `POST /api/v1/files/audio` 补 handler contract test，覆盖音频 MIME、大小限制和成功 envelope。
- 为 `POST /api/v1/files/images/batch` 补 handler contract test，覆盖多文件字段名、默认 `PUBLIC`、`PRIVATE`、部分失败只返回成功项的当前语义。
- 将 File 错误响应从 HTTP 风格 `body.code` 迁移到业务错误码时，必须作为独立 contract 变更处理。
