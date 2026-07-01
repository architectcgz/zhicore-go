# 删除文件

## 来源

- 服务设计：`docs/architecture/services/file/README.md`
- 当前 API schema：`services/zhicore-file/api/http/README.md`
- Go handler：`services/zhicore-file/api/http/handler.go`
- Go contract test：`services/zhicore-file/api/http/handler_test.go`
- 历史 Java 参考：`../zhicore-microservice/zhicore-upload/src/main/java/com/zhicore/upload/controller/FileUploadController.java`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `DELETE` |
| 主路径 | `/api/v1/files/{fileId}` |
| 兼容别名 | 无 |
| Content-Type | 无 body |
| 鉴权 | 匿名 / 服务间；不读取 `Authorization` 或 `X-User-Id` |
| 幂等 | 当前 contract 不保证幂等；由 File service application 决定重复删除语义 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `fileId` | string | 是 | File service 文件 ID；不能为空白。 |

## Query 参数

无。

## Body / Multipart 字段

无。

## 成功响应 `data`

成功响应只返回 envelope；`data` 省略。

示例：

```json
{
  "code": 200,
  "message": "操作成功",
  "timestamp": 1782112892184
}
```

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `400` | `400` | `文件ID不能为空` | `fileId` 为空白。 |
| `500` | `500` | `系统内部错误，请稍后重试` | 未分类服务端错误。 |

`400` / `500` 是当前 Go handler 实际输出。对象存储或 metadata 删除返回文件不存在、无权限、删除失败或不可用时，adapter 后续必须映射为服务级 README 登记的稳定错误。

## 权限和可见性

- 本 endpoint 当前不校验业务 owner，调用方必须在业务服务中先确认文件引用可删除。
- File service 不拥有头像、封面、评论媒体等业务引用关系，不负责级联清理业务实体。
- 删除对象存储失败时必须进入 File service 的补偿或清理机制；首期未实现前由技术债记录退出条件。

## 排序、分页和过滤

无。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `DeleteFile(fileId)` |
| Application owner | `services/zhicore-file/internal/file/application.Service` |
| Port | `ports.FileService.DeleteFile` |
| 事务边界 | 当前无本地事务；后续 metadata 状态变更和对象存储删除必须在 File service 内定义补偿边界。 |
| 事件 | 当前不发布事件。 |

## 测试要求

- Handler contract test：`TestGetFileURLAndDeleteFileUsePathFileID`。
- System HTTP test：待补。
