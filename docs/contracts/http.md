# HTTP 契约

本文件只定义 HTTP 协议层规则。错误码见 `docs/contracts/errors.md`，时间和 ID 等通用类型见 `docs/contracts/data-types.md`。

## 兼容基线

迁移阶段默认保留 Java 外部接口：

- path
- HTTP method
- query/path/body 参数
- multipart 字段名
- 响应 envelope
- HTTP status
- 字段名、字段类型和空值语义
- 错误码和错误信息语义

需要重做的接口必须作为独立 API 演进任务处理，不能夹在服务迁移里顺手改变。

## 成功响应

HTTP 成功响应使用 Java `ApiResponse` 兼容形态：

```json
{
  "code": 200,
  "message": "操作成功",
  "data": {},
  "timestamp": 1782112892184,
  "traceId": "optional-trace-id"
}
```

规则：

- 普通成功响应使用 HTTP `200`，不使用 `204`，避免破坏前端对 envelope 的解析。
- `data` 是否出现以 Java 当前行为为准；新接口无返回体时可以省略 `data`。
- `timestamp` 使用 Unix epoch milliseconds。
- `traceId` 有则返回，没有则可省略。

## 请求

- `Content-Type: application/json` 用于 JSON body。
- `multipart/form-data` 用于上传，字段名必须和 Java controller 保持一致。
- path variable 和 query 参数名保持 Java controller 现状。
- 不用 Gateway 做参数重命名或响应形态转换。

## 响应 Header

- JSON 响应使用 `Content-Type: application/json; charset=utf-8`。
- 如果存在请求 ID / trace ID，优先接受 `X-Request-Id` 或 `X-Trace-Id`，响应可回传同名或统一后的 `X-Request-Id`。
- 鉴权相关 header 保持当前前端约定，不因服务迁移改名。

## 版本化

已有 `/api/v1/...` 保持不变。

破坏性变更必须使用以下方式之一：

- 新增并行 endpoint。
- 新增版本化 endpoint，例如 `/api/v2/...`。
- 新增字段并保留旧字段，等所有 consumer 迁移后再独立清理。

## 服务级 HTTP schema

字段级 HTTP contract 不写在本文件。每个服务自己的 schema 放在：

```text
services/<service>/api/http/
```

这些 schema 至少记录：

- endpoint path 和 method。
- request path/query/body/multipart 字段。
- response `data` 字段。
- 可能的公开错误码。
- 分页、排序、过滤和权限语义。
