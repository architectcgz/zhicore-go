# 错误契约

本文件定义对外 HTTP API、typed client 和跨服务调用中可见的错误模型。公开错误码表见 `docs/contracts/error-codes.md`；Go 内部错误分层和映射规则见 `docs/architecture/error-handling.md`。

## 基本原则

- 迁移阶段保持 Java 外部错误语义兼容。
- 已有 Java endpoint、controller advice 或 gateway/filter 已经定义的 HTTP status 和 body `code` 是迁移事实源，优先级高于本文件的默认映射表。
- 对外错误只暴露稳定错误码、用户可理解消息和必要定位信息。
- 不暴露 SQL、Redis、RabbitMQ、外部 SDK、堆栈、内部 sentinel 或服务内部包名。
- 错误码属于 provider；consumer 只能依赖 provider 公布的错误码。

## HTTP 错误响应形态

HTTP API 使用统一 envelope：

```json
{
  "code": 400,
  "message": "请求参数错误",
  "timestamp": 1782112892184,
  "traceId": "optional-trace-id"
}
```

字段规则：

- `code`：公开稳定的数字错误码，属于业务/契约错误码，不等同于 HTTP status。
- `message`：兼容 Java 当前语义的错误信息。
- `data`：错误响应默认不返回；只有字段级校验错误等明确需要结构化细节时才返回。
- `timestamp`：Unix epoch milliseconds。
- `traceId`：有链路 ID 时返回；没有时可省略。

## `code` 与 HTTP status

响应 body 里的 `code` 是调用方用于业务分支和错误识别的稳定错误码；HTTP status 只表达传输层和粗粒度请求结果。Go 新实现不得为了方便把 HTTP status 直接写入 body `code`，除非对应 Java 既有接口已经把同一个数字作为公开 `code` 暴露，并且服务级 HTTP contract 明确记录这是兼容行为。

迁移和新增实现按以下顺序选择 body `code`：

1. 对应 Java `ResultCode` 或既有 Java endpoint 实际返回的公开错误码。
2. 服务级 HTTP contract 中登记的公开错误码。
3. 本文件的错误码范围分配出来的新业务错误码。

示例：

- 参数校验失败：HTTP status 可以是 `400`，body `code` 应优先使用 `1001`。
- 未登录：HTTP status 可以是 `401`，body `code` 应优先使用 `2006` 或对应 Java 兼容码。
- 文件类型不允许：HTTP status 可以是 `400`，body `code` 应优先使用 `8002`。
- 未分类服务端错误：HTTP status 可以是 `500`，body `code` 应优先使用 `1000` 或 Java 兼容的 `500` 例外。

Java `ResultCode` 中历史上同时包含 `400`、`401`、`404`、`500` 等 HTTP 风格数字和 `1xxx`-`8xxx` 业务错误码。Go 侧不能继续扩大这种混用；保留 HTTP 风格数字只用于兼容已经发布的接口，新服务错误优先使用业务错误码范围。

## 错误码归属

Go 项目公开错误码使用下列范围；完整码值见 `docs/contracts/error-codes.md`。

| 范围 | 归属 |
| --- | --- |
| `1xxx` | 通用错误 |
| `2xxx` | 认证授权 |
| `3xxx` | User |
| `4xxx` | Content |
| `5xxx` | Comment |
| `6xxx` | Message |
| `7xxx` | Notification |
| `8xxx` | Upload |

HTTP status 可以作为粗分类，但不能替代稳定业务错误码。历史接口如果 Java 当前只返回 HTTP status 作为 `code`，Go 迁移必须先保持兼容，并在服务级 HTTP contract 中标记为兼容例外；后续要改成业务码时，作为独立 API 演进任务处理。

服务内部可以有 `UPLOAD_001` 这类内部错误标识，但除非 Java 外部接口已经暴露，否则不要直接作为公开 `code` 输出。

## HTTP status 兼容优先级

迁移已有接口时，按以下顺序确定 HTTP status：

1. 对应 Java controller、exception handler、gateway filter 的实际 HTTP status。
2. 服务级 HTTP contract 中记录的兼容 status。
3. 本文件的默认映射表。

Java common 当前对 `BusinessException` 和 `DomainException` 使用 HTTP `200` + body `code` 表达业务失败；Go 迁移这些接口时必须保持现状。后续如果要改成更标准的 4xx/5xx，必须作为独立 API 演进任务处理。

## HTTP status 映射

下表只作为新接口或 Java 无明确处理时的默认规则。

| HTTP status | 场景 |
| --- | --- |
| `400` | 参数缺失、类型错误、文件类型不允许、multipart 解析失败或框架层上传限制等请求错误 |
| `401` | 未登录、token 无效或过期 |
| `403` | 已登录但无权限 |
| `404` | 归属服务确认资源不存在 |
| `405` | HTTP method 不支持 |
| `409` | 幂等冲突、重复操作、状态冲突 |
| `413` | payload 过大、对象存储或文件服务配额超限；例如 Java Upload 的 `QuotaExceededException` |
| `429` | 限流 |
| `500` | 未分类服务端错误 |
| `503` | 下游依赖不可用、服务降级、超时且可重试 |

## 参数校验错误

已有接口保持 Java 当前错误响应。新接口如需要字段级错误，可以在 `data` 中返回结构化详情：

```json
{
  "code": 1001,
  "message": "参数校验失败",
  "data": {
    "fields": [
      {
        "field": "accessLevel",
        "reason": "required"
      }
    ]
  },
  "timestamp": 1782112892184
}
```

字段级错误不是默认要求；只有前端或 consumer 需要精确定位字段时再引入。

## 外部依赖错误

- 下游超时、连接失败、熔断打开：对外通常映射为 `503`。
- 下游返回资源不存在：如果语义属于当前 provider 对外 contract，可映射为 `404`。
- 下游权限失败：不要直接透传下游 message；由当前 provider 映射成自己的公开错误。

## 服务错误码清单

每个服务的公开错误码清单应随字段级 HTTP contract 放在：

```text
services/<service>/api/http/
```

跨服务 typed client 如需要显式错误枚举，由 provider 在 `libs/contracts/clients/<provider-service>` 中定义稳定错误语义。
