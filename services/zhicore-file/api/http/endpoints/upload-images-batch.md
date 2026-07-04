# 批量上传图片

## 来源

- 服务设计：`docs/architecture/services/file/README.md`
- 当前 API schema：`services/zhicore-file/api/http/README.md`
- Go handler：`services/zhicore-file/api/http/handler.go`
- 历史 Java 参考：`../zhicore-microservice/zhicore-upload/src/main/java/com/zhicore/upload/controller/FileUploadController.java`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/files/images/batch` |
| 兼容别名 | 无 |
| Content-Type | `multipart/form-data` |
| 鉴权 | 匿名 / 服务间；不读取 `Authorization` 或 `X-User-Id` |
| 幂等 | 当前 HTTP contract 不保证幂等；秒传由 `zhicore-file` 后续 metadata / hash 规则决定 |

## Path 参数

无。

## Query 参数

无。

## Body / Multipart 字段

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `files` | file[] | 是 | 缺失或空列表返回 `文件不能为空` | 图片文件列表。每个文件按图片规则校验。 |
| `accessLevel` | string | 否 | 空或缺失按 `PUBLIC` | `PUBLIC` 或 `PRIVATE`；Go handler 接受大小写输入并归一为大写。 |

当前 Go handler 解析 multipart 上限为 100 MiB；单个文件仍按图片最大 50 MiB 校验。

## 成功响应 `data`

`data` 为 `UploadFile[]`。当前 Go application 对一批文件执行全有或全无的 contract：任一文件上传失败或校验失败时整批返回错误，不返回部分成功列表，避免调用方误以为整批资源都可引用。

示例：

```json
{
  "code": 200,
  "message": "操作成功",
  "data": [
    {
      "fileId": "file_1",
      "url": "https://cdn.example.com/file_1.jpg",
      "fileSize": 12,
      "instantUpload": false,
      "accessLevel": "PUBLIC",
      "originalName": "a.jpg",
      "contentType": "image/jpeg"
    }
  ],
  "timestamp": 1782112892184
}
```

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `400` | `400` | `无效的 multipart 请求` | multipart 解析失败或超过解析上限。 |
| `400` | `400` | `文件不能为空` | `files` 字段缺失或为空。 |
| `400` | `400` | `访问级别必须是 PUBLIC 或 PRIVATE` | `accessLevel` 不是允许值。 |
| `400` | `400` | `批量上传存在失败文件: ...` | 任一文件为空、类型不允许、大小超限或对象存储返回可归类输入错误。 |
| `500` | `500` | `系统内部错误，请稍后重试` | 未分类服务端错误。 |

单个文件为空、类型不允许或大小超限时，当前 Go application 会返回整批错误；后续如需返回逐文件失败详情，必须作为 contract 变更处理。

## 权限和可见性

- 本 endpoint 不校验业务 owner。
- `PUBLIC` / `PRIVATE` 对所有成功项统一生效。
- 单个文件能否被头像、封面、评论媒体等业务资源引用，由对应业务服务判断。

## 排序、分页和过滤

- 无分页。
- 成功时返回数组顺序保持文件在请求 `files` 字段中的相对顺序。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `UploadImagesBatch(files, accessLevel)` |
| Application owner | `services/zhicore-file/internal/file/application.Service` |
| Port | `ports.FileService.Upload` |
| 事务边界 | 当前无本地事务；每个文件独立处理，后续 metadata 写入和对象存储写入必须在 File service 内定义补偿边界。 |
| 事件 | 当前不发布事件。 |

## 测试要求

- Handler contract test：已覆盖字段名 `files`、默认 `PUBLIC`、`PRIVATE`、部分失败整批返回错误；待补非法 `accessLevel` 和全部失败错误。
- System HTTP test：待补。
