# 上传音频

## 来源

- 服务设计：`docs/architecture/services/upload/README.md`
- 当前 API schema：`services/zhicore-upload/api/http/README.md`
- Go handler：`services/zhicore-upload/api/http/handler.go`
- Java 参考：`../zhicore-microservice/zhicore-upload/src/main/java/com/zhicore/upload/controller/FileUploadController.java`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/upload/audio` |
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
| `file` | file | 是 | 不允许为空 | 音频文件。当前默认允许 `audio/mpeg`、`audio/mp3`、`audio/mp4`、`audio/x-m4a`、`audio/aac`、`audio/wav`、`audio/x-wav`、`audio/ogg`、`audio/webm`，最大 10 MiB。 |

访问级别固定为 `PUBLIC`。

## 成功响应 `data`

`data` 为 `UploadFile`，字段见 `services/zhicore-upload/api/http/README.md`。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `400` | `400` | `文件不能为空` | multipart 缺少 `file` 或文件为空。 |
| `400` | `400` | `文件类型不允许: <contentType>` | `file` MIME type 不在音频允许列表。 |
| `400` | `400` | `文件大小超过限制` | 文件超过音频大小限制。 |
| `500` | `500` | `系统内部错误，请稍后重试` | 未分类服务端错误。 |

`400` / `500` 是当前 Go handler 实际输出的兼容例外。目标业务码见服务级 README。

## 权限和可见性

- 本 endpoint 不校验业务 owner。
- 上传成功后返回的 `fileId` 由业务服务保存引用。
- 返回 URL 是否长期公开由 `PUBLIC` 访问级别和 File Service 决定。

## 排序、分页和过滤

无。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `UploadAudio(file, PUBLIC)` |
| Application owner | `services/zhicore-upload/internal/upload/application.Service` |
| Port | `ports.FileService.Upload` |
| 事务边界 | 无本地事务；外部 File Service 拥有文件元数据。 |
| 事件 | 当前不发布事件。 |

## 测试要求

- Handler contract test：待补，至少覆盖成功 envelope、默认 `PUBLIC`、不允许的 MIME type 和大小限制。
- System HTTP test：待补。
