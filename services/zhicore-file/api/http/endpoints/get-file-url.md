# 获取文件访问 URL

## 来源

- 服务设计：`docs/architecture/services/file/README.md`
- 当前 API schema：`services/zhicore-file/api/http/README.md`
- Go handler：`services/zhicore-file/api/http/handler.go`
- Go contract test：`services/zhicore-file/api/http/handler_test.go`
- 历史 Java 参考：`../zhicore-microservice/zhicore-upload/src/main/java/com/zhicore/upload/controller/FileUploadController.java`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/files/{fileId}/url` |
| 兼容别名 | 无 |
| Content-Type | 无 body |
| 鉴权 | 匿名 / 服务间；不读取 `Authorization` 或 `X-User-Id` |
| 幂等 | 幂等查询 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `fileId` | string | 是 | File service 文件 ID；不能为空白。 |

## Query 参数

无。

## Body / Multipart 字段

无。

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `data` | string | 是 | 文件访问 URL。 |

示例：

```json
{
  "code": 200,
  "message": "操作成功",
  "data": "https://cdn.example.com/file_123.jpg",
  "timestamp": 1782112892184
}
```

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `400` | `400` | `文件ID不能为空` | `fileId` 为空白。 |
| `500` | `500` | `系统内部错误，请稍后重试` | 未分类服务端错误。 |

`400` / `500` 是当前 Go handler 实际输出。对象存储或 metadata 查询返回文件不存在、无权限或不可用时，adapter 后续必须映射为服务级 README 登记的稳定错误。

## 权限和可见性

- 本 endpoint 不校验业务 owner。
- 对 `PRIVATE` 文件，返回 URL 的签名、有效期和访问控制由 `zhicore-file` 决定。
- 业务服务不应把该 URL 当成永久事实保存；长期引用应保存 `fileId`。

## 排序、分页和过滤

无。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `GetFileURL(fileId)` |
| Application owner | `services/zhicore-file/internal/file/application.Service` |
| Port | `ports.FileService.GetFileURL` |
| 事务边界 | 无本地事务。 |
| 事件 | 当前不发布事件。 |

## 测试要求

- Handler contract test：`TestGetFileURLAndDeleteFileUsePathFileID`。
- System HTTP test：待补。
