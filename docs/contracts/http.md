# HTTP 契约

本文件只定义 HTTP 协议层规则。服务级字段 schema 模板见 `docs/contracts/http-schema-template.md`；错误码见 `docs/contracts/errors.md`，时间和 ID 等通用类型见 `docs/contracts/data-types.md`。

## HTTP Contract 基线

Go HTTP contract 优先由服务级 schema 固定。未登记 Go-first API reset 的服务在替换既有实现时不得破坏已发布外部接口；Java 只作为核对既有行为的参考来源。当前 `zhicore-content` 已登记为 Go-first API reset，Content 的 Go HTTP schema 是新事实源，Java 只作为业务能力参考。

已发布外部接口需要保留：

- path
- HTTP method
- query/path/body 参数
- multipart 字段名
- 响应 envelope
- HTTP status
- 字段名、字段类型和空值语义
- 错误码和错误信息语义

需要重做的接口必须作为独立 API 演进任务处理，不能夹在服务实现里顺手改变。

## 成功响应

HTTP 成功响应使用 ZhiCore 统一 envelope。承接已发布接口的服务保持当前 envelope 语义；Go-first API reset 服务仍使用同一 envelope 语义，但 `data` 字段以服务级 schema 为准：

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
- 承接已发布接口时，`data` 是否出现以服务级 schema 和既有外部行为为准；Go-first API reset 服务按自己的服务级 schema 记录。
- `timestamp` 使用 Unix epoch milliseconds。
- `traceId` 有则返回，没有则可省略。

## 请求

- `Content-Type: application/json` 用于 JSON body。
- `multipart/form-data` 用于上传。承接已发布接口时字段名保持既有外部 contract；Go-first API reset 服务按服务级 schema 记录。
- path variable 和 query 参数名以服务级 schema 为准；承接已发布接口时不得无登记改名。
- 不用 Gateway 做参数重命名或响应形态转换。

## 认证和内部身份 Header

外部调用方只通过当前前端约定提交认证凭证，通常是：

```text
Authorization: Bearer <access-token>
```

规则：

- `Authorization` 只由 Gateway 和明确拥有凭证语义的 Auth endpoint 解析。普通业务服务不得从 `Authorization` 解析 JWT 作为当前用户身份。
- Gateway 校验 JWT 后，先移除客户端传入的同名内部身份 header，再重新写入下游可信身份 header。
- 下游服务只消费 Gateway 注入的可信身份上下文，并把它映射成 application input，例如 `Actor`、`AuthContext` 或 `Principal`。
- 缺少可信身份上下文的登录态 endpoint 返回认证失败；不得通过解析 `Authorization` 做服务内 fallback。
- Gateway 不判断资源归属权限。资源归属、可见性和业务权限仍由归属服务 application 判断。

当前内部身份 header 固定为：

| Header | 用途 | 可见范围 |
| --- | --- | --- |
| `X-Account-Id` | 当前登录 Auth account ID | Gateway -> 下游服务。 |
| `X-User-Id` | 当前登录用户 ID | Gateway -> 下游服务。 |
| `X-User-Name` | 当前用户名或展示名 | Gateway -> 下游服务，可选。 |
| `X-User-Roles` | 当前用户角色集合，逗号分隔 | Gateway -> 下游服务，可选。 |
| `X-Session-Id` | 当前 access token 所属登录 session ID | Gateway -> Auth 和需要 session 语义的下游服务。 |
| `X-Session-Version` | 当前 access token 携带的 session version | Gateway -> Auth 和需要认证状态校验的下游服务。 |
| `X-Principal-Version` | 当前 access token 携带的 principal version | Gateway -> Auth 和需要 principal 刷新的下游服务。 |
| `X-Request-Id` | 请求关联 ID | 外部可传入，服务间继续传播。 |
| `X-Trace-Id` | 链路关联 ID | 外部可传入，服务间继续传播。 |

服务级 HTTP schema 必须在“鉴权上下文”中说明 endpoint 是匿名、登录用户还是管理员，以及需要哪些身份字段。

## 响应 Header

- JSON 响应使用 `Content-Type: application/json; charset=utf-8`。
- 如果存在请求 ID / trace ID，优先接受 `X-Request-Id` 或 `X-Trace-Id`，响应可回传同名或统一后的 `X-Request-Id`。
- 鉴权相关 header 保持当前前端约定，不因服务内部实现改名。

## 版本化

已有 `/api/v1/...` 保持不变。

破坏性变更必须使用以下方式之一：

- 新增并行 endpoint。
- 新增版本化 endpoint，例如 `/api/v2/...`。
- 新增字段并保留旧字段，等所有 consumer 切换后再独立清理。

## 服务级 HTTP schema

字段级 HTTP contract 不写在本文件。每个服务自己的 schema 放在：

```text
services/<service>/api/http/README.md
services/<service>/api/http/endpoints/<operation>.md
```

这些 schema 至少记录：

- endpoint path 和 method。
- request path/query/body/multipart 字段。
- response `data` 字段。
- 可能的公开错误码。
- 分页、排序、过滤和权限语义。

具体模板和状态标记见 `docs/contracts/http-schema-template.md`。
