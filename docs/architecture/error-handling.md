# 错误处理架构

本文件定义 Go 服务内部错误分层、对外错误映射和错误处置分级。对外错误响应、错误码和 HTTP status 见 `docs/contracts/errors.md`；通用日志、metrics、trace 和脱敏规则见 `docs/architecture/observability.md`。

## 分层原则

错误依赖方向必须和代码依赖方向一致：

```text
infrastructure error -> ports/domain semantic error -> application public error -> api/http response
```

禁止：

- handler 直接判断 SQL、Redis、RabbitMQ、外部 SDK 的错误。
- application 直接暴露 ORM、driver、SDK sentinel。
- domain 依赖 HTTP status 或响应 envelope。
- `libs/contracts` 中放 fallback、重试、熔断或调用方业务策略。

## Domain

Domain 只定义纯业务语义错误，例如：

- 状态不允许。
- 领域不变量被破坏。
- 值对象非法。

Domain 不知道 HTTP、数据库、Redis、RabbitMQ 或外部 SDK。

## Ports

Ports 可以定义 application 需要分支处理的语义错误，例如：

- `ErrPostNotFound`
- `ErrUserUnavailable`
- `ErrDuplicateLike`
- `ErrFileNotFound`

这些错误表示“调用方需要做业务分支”，不是底层技术错误的透传。

## Infrastructure

Infrastructure 负责把具体技术错误翻译成 ports/domain 语义：

- `sql.ErrNoRows` -> `ports.ErrFileNotFound`
- Redis nil -> cache miss，而不是业务错误
- File metadata / object storage 404 -> `ports.ErrFileNotFound`
- 外部超时 -> `ports.ErrDependencyUnavailable`

Infrastructure 可以记录底层错误日志，但不能把底层错误文本直接作为对外 message。

## Application

Application 拥有 use case 错误映射：

- 判断 domain/ports 语义。
- 决定错误是否对外可见。
- 映射到公开错误码和 HTTP status。
- 决定幂等冲突、重复请求、权限失败、状态冲突的业务结果。

同一个 infrastructure not-found 在不同 use case 中可能映射不同结果，所以不要在 repository 层过早决定 HTTP status。

## API / HTTP

HTTP handler 只处理：

- 参数绑定和基础校验错误。
- application 返回的公开错误。
- 未知错误统一映射为内部错误，并记录 trace。

HTTP handler 不直接访问数据库、缓存、MQ 或外部 SDK。

## 错误日志和观测字段

- 每个请求应有 `traceId` 或 `requestId`，传播规则见 `docs/architecture/observability.md`。
- 对外错误响应可以带 `traceId`，响应形态见 `docs/contracts/errors.md`。
- 错误日志中记录底层错误摘要、traceId / requestId、provider、operation 和关键业务 ID。
- 对外 `message` 不包含底层错误细节。

## 运行期错误处置

错误处置分为四类：忽略、记录、上报、补偿。每个 handler、worker、consumer 和 adapter 必须先判断错误属于哪一类，再决定日志级别和后续动作。

| 场景 | 处置 | 日志级别 | 是否上报 |
| --- | --- | --- | --- |
| 预期内的业务拒绝，例如未登录、权限不足、状态不允许、重复点赞 | 返回公开错误码 | `INFO` 或 `WARN` | 默认不上报 |
| 参数校验、multipart 解析失败、类型错误 | 返回公开错误码 | `INFO`；异常频率升高时 `WARN` | 默认不上报 |
| 资源不存在，且属于正常查询分支 | 返回公开错误码或空结果 | `INFO` 或不记录 | 默认不上报 |
| Redis cache miss、幂等重复消费、已处理事件重复投递 | 按正常分支处理 | 不记录或 `DEBUG` | 不上报 |
| 下游超时、连接失败、限流、熔断打开 | 返回降级错误或进入重试/补偿 | `WARN`；持续失败时 `ERROR` | 持续失败或影响核心路径时上报 |
| 数据库写失败、事务提交失败、outbox 写入失败 | 返回失败，保留底层错误日志 | `ERROR` | 上报 |
| 未分类 panic、未知错误、HTTP 5xx | recovery 后返回内部错误 | `ERROR` | 上报 |
| worker/consumer 可重试失败 | 记录 retry metadata，进入重试 | `WARN` | 超过阈值或进入 dead-letter 时上报 |
| 明确 best-effort 的通知型事件失败 | 记录失败原因和业务 ID | `WARN` | 默认不上报；连续失败时上报 |

当前个人项目阶段，“上报”至少表示结构化 `ERROR` 日志和可统计的错误计数；接入 Sentry、OpenTelemetry Collector、Prometheus Alertmanager 或其他告警系统后，再把同一类事件接入外部告警。不要为了尚未接入告警系统而省略错误分类和关键字段。

## 错误日志要求

- 通用日志字段、日志级别定义、脱敏规则和下游调用日志字段见 `docs/architecture/observability.md`。
- 业务错误日志记录公开 message 和业务 ID；底层错误只进入内部日志字段，不进入对外响应。
- 高频可预期分支不能刷 `ERROR`，避免真正故障被噪声淹没。
- 下游调用失败必须包含 provider、operation、status、durationMs、attempt 和 errorCode。
- 异步消费失败必须包含 eventId、eventType、attempt、consumer、幂等处理结果和下一步动作。

## 重试和降级

- 重试策略属于 infrastructure adapter 或调用方 application，不属于 `libs/contracts`。
- 幂等写入必须先有业务唯一键、idempotency key 或 outbox/inbox 保障，再谈重试。
- 降级结果必须由 application 决定，不能由 HTTP client adapter 自行返回业务 DTO。
