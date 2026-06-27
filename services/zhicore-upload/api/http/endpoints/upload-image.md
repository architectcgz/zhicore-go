# 上传图片

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
| 主路径 | `/api/v1/upload/image` |
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

访问级别固定为 `PUBLIC`。

## 成功响应 `data`

`data` 为 `UploadFile`，字段见 `services/zhicore-upload/api/http/README.md`。

示例：

```json
{
  "code": 200,
  "message": "操作成功",
  "data": {
    "fileId": "file_123",
    "url": "https://cdn.example.com/file_123.jpg",
    "fileSize": 12,
    "instantUpload": false,
    "uploadTime": "2026-06-22T10:00:00Z",
    "accessLevel": "PUBLIC",
    "originalName": "avatar.jpg",
    "contentType": "image/jpeg"
  },
  "timestamp": 1782112892184
}
```

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `400` | `400` | `文件不能为空` | multipart 缺少 `file` 或文件为空。 |
| `400` | `400` | `文件类型不允许: <contentType>` | `file` MIME type 不在图片允许列表。 |
| `400` | `400` | `文件大小超过限制` | 文件超过图片大小限制。 |
| `500` | `500` | `系统内部错误，请稍后重试` | 未分类服务端错误。 |

`400` / `500` 是当前 Go handler 实际输出的兼容例外。目标业务码见服务级 README。

## 权限和可见性

- 本 endpoint 不校验业务 owner。
- 上传成功后返回的 `fileId` 由业务服务保存引用；头像、封面、评论媒体等归属校验不属于 Upload。
- 返回 URL 是否长期公开由 `PUBLIC` 访问级别和 File Service 决定。

## 排序、分页和过滤

无。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `UploadImage(file, PUBLIC)` |
| Application owner | `services/zhicore-upload/internal/upload/application.Service` |
| Port | `ports.FileService.Upload` |
| 事务边界 | 无本地事务；外部 File Service 拥有文件元数据。 |
| 事件 | 当前不发布事件。 |

## 测试要求

- Handler contract test：`TestUploadImageUsesPublicAccessAndReturnsJavaCompatibleEnvelope`、`TestUploadImageRejectsUnsupportedContentType`。
- System HTTP test：待补。
