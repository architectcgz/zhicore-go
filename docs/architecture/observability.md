# 日志与可观测性规范

本文件定义 `zhicore-go` 的结构化日志、请求关联 ID、trace 传播、metrics 标签、错误上报边界和 `libs/kit/observability` 职责。

## 基本原则

- 当前阶段先建立轻量可观测性基线，不强制引入完整 OpenTelemetry、Prometheus、Sentry 或集中日志平台。
- 可观测性不能改变业务控制流。logger、metrics、trace、debug flag 只能影响“记录什么”，不能影响“失败后怎么跑”。
- 日志、metrics 和 trace 使用同一套稳定 `operation` 命名，便于排查同一行为。
- 生产日志默认结构化 JSON；本地开发可以使用文本格式，但字段语义必须一致。
- 敏感信息默认不记录；需要记录配置、请求或响应摘要时必须显式脱敏。
- 审计日志是业务事实，归属对应服务 schema 和用例；普通运行日志不能替代审计日志。

## 适用范围

- HTTP handler、middleware、Gateway、application use case。
- repository、cache、MQ、外部 HTTP client、对象存储 adapter。
- worker、consumer、dispatcher、定时任务和后台 goroutine owner。
- `libs/kit/observability` 跨服务日志、metrics、trace 和脱敏原语。

错误分层、错误映射和错误日志级别见 `docs/architecture/error-handling.md`。配置字段、日志格式和级别配置见 `docs/architecture/configuration.md`。

## 日志字段

通用字段：

| 字段 | 含义 | 要求 |
| --- | --- | --- |
| `ts` | 日志时间 | UTC 时间或日志库标准时间字段 |
| `level` | 日志级别 | `DEBUG`、`INFO`、`WARN`、`ERROR` |
| `service` | 服务名 | 例如 `zhicore-upload` |
| `env` | 运行环境 | 例如 `local`、`dev`、`prod` |
| `operation` | 稳定操作名 | 例如 `upload.image`、`content.post.publish` |
| `requestId` | HTTP 请求关联 ID | 有请求边界时必须有 |
| `traceId` | 链路关联 ID | 有上游传入或本服务生成时必须传播 |
| `durationMs` | 操作耗时 | 请求、下游调用、任务处理必须记录 |
| `errorCode` | 公开错误码或内部稳定错误标识 | 错误场景记录 |
| `error` | 内部错误摘要 | 只记录脱敏后的内部排查信息 |

业务关联字段：

- `userId`：已认证用户 ID；不要记录 token、密码、验证码或完整个人敏感载荷。
- `resourceId`：关键业务 ID，例如 `postId`、`fileId`、`eventId`。
- `provider`：下游 provider，例如 `postgres`、`redis`、`rabbitmq`、`file-service`。
- `status`：HTTP status、任务状态、下游响应分类或业务状态。
- `attempt`：重试次数或消费尝试次数。

规则：

- 字段名保持稳定，避免同一语义在不同服务写成 `request_id`、`requestId`、`reqId` 多套。
- 不把高基数字段写入 metrics label；日志可以记录必要业务 ID。
- 日志 message 保持短句，机器检索依赖字段，不依赖长文本。

## 日志级别

| 级别 | 用途 | 示例 |
| --- | --- | --- |
| `DEBUG` | 本地或临时排查细节，生产默认关闭或采样 | cache miss、解析中间状态 |
| `INFO` | 正常业务分支和生命周期事件 | 服务启动、请求完成、幂等重复已处理 |
| `WARN` | 可恢复异常、降级、重试、依赖短暂不可用 | 下游 timeout 后重试、ready check 短暂失败 |
| `ERROR` | 需要排查或上报的失败 | 数据库写失败、事务提交失败、panic、HTTP 5xx |

规则：

- 预期内业务拒绝、参数校验失败、资源不存在等不能默认刷 `ERROR`。
- `ERROR` 日志必须包含 `operation`、`traceId` 或 `requestId`、错误分类和必要业务 ID。
- 高频失败必须考虑限频、聚合或采样，避免日志风暴掩盖真正故障。
- root 级关键后台任务 panic 应记录现场后按 owner 语义处理，不能静默吞掉。

## 请求和 trace 传播

HTTP 入口：

- 优先接受上游 `X-Request-Id` 和 `X-Trace-Id`。
- 上游没有时生成新的 `requestId`；没有 `traceId` 时可以复用或派生一个 trace ID。
- 响应 header 应回传 `X-Request-Id`；对外错误响应可以带 `traceId`，具体 envelope 见 `docs/contracts/errors.md`。
- `requestId` / `traceId` 从 HTTP handler 进入 `context.Context`，继续传给 application、repository、cache、MQ 和下游 client。

下游调用：

- HTTP client 调用下游时传播 `X-Request-Id` / `X-Trace-Id`。
- RabbitMQ 事件 envelope 应携带可用于关联的 trace 字段；事件 contract 见 `docs/contracts/events.md`。
- worker / consumer 从消息或任务 metadata 恢复 trace 字段；没有时生成新的任务级关联 ID。

当前阶段不强制接 OpenTelemetry SDK；未来接入时应沿用已有 `operation`、`service`、`env`、`traceId` 语义，不重写业务层接口。

## Operation 命名

`operation` 使用小写点分命名：

```text
<domain>.<action>
<domain>.<resource>.<action>
```

示例：

```text
upload.image
upload.audio
content.post.publish
comment.create
rabbitmq.outbox.publish
postgres.post.insert
```

规则：

- 名称表示稳定行为，不带用户 ID、资源 ID、分页参数或错误文本。
- HTTP handler、application、repository、下游 client、worker 可以在同一行为下使用相同前缀。
- 新增公开 endpoint、worker、consumer 或下游 adapter 时必须先确定 operation 名。
- `libs/kit/observability` 不登记服务私有 operation 清单；operation 由服务或 contract owner 维护。

## Metrics

当前阶段代码只需保留稳定指标语义和低基数标签；接入 Prometheus 或其他平台时再落具体 exporter。

推荐基础指标：

- HTTP：请求数、错误数、duration histogram、in-flight。
- 下游 client：请求数、错误数、duration、retry 次数。
- DB / cache / MQ：操作数、错误数、duration、连接池摘要。
- worker / consumer：处理数、失败数、重试数、dead-letter 数、lag 或 backlog。
- producer outbox：publish 结果、confirm 耗时、retry、pending、oldest pending、dead 和 stale claim。

标签规则：

- 允许：`service`、`env`、`operation`、`status`、`errorCode`、`provider`、`method`、`route`、`eventType`、`consumer`。
- 禁止：`userId`、`fileId`、`postId`、`requestId`、`traceId`、原始 URL、错误文本、完整 SQL、完整 routing key 中的高基数片段。
- route 使用模板，例如 `/api/v1/posts/{id}`，不要使用原始 path。
- error label 使用稳定错误码或错误分类，不使用动态错误字符串。

指标命名采用 Prometheus 风格语义；如果未来接入其他平台，也要保持同等语义映射：

| 指标 | 类型 | 标签 | 含义 |
| --- | --- | --- | --- |
| `zhicore_http_requests_total` | counter | `service`、`env`、`operation`、`method`、`route`、`status`、`errorCode` | HTTP 请求结果计数 |
| `zhicore_http_request_duration_seconds` | histogram | `service`、`env`、`operation`、`method`、`route` | HTTP 请求耗时 |
| `zhicore_http_inflight` | gauge | `service`、`env`、`operation`、`method`、`route` | 正在处理的 HTTP 请求数 |
| `zhicore_client_requests_total` | counter | `service`、`env`、`provider`、`operation`、`status`、`errorCode` | 下游 client 调用结果计数 |
| `zhicore_client_request_duration_seconds` | histogram | `service`、`env`、`provider`、`operation`、`status` | 下游 client 调用耗时 |
| `zhicore_client_retries_total` | counter | `service`、`env`、`provider`、`operation`、`reason` | 下游 client retry 次数 |
| `zhicore_client_circuit_state` | gauge | `service`、`env`、`provider`、`operation` | 熔断状态，`0=closed`、`1=open`、`2=half_open` |
| `zhicore_client_circuit_open_total` | counter | `service`、`env`、`provider`、`operation`、`reason` | 熔断打开次数 |
| `zhicore_client_degraded_total` | counter | `service`、`env`、`provider`、`operation`、`strategy` | 降级执行次数 |
| `zhicore_client_inflight` | gauge | `service`、`env`、`provider`、`operation` | 正在执行的下游调用数 |
| `zhicore_worker_jobs_total` | counter | `service`、`env`、`worker`、`operation`、`status` | worker / job 处理结果计数 |
| `zhicore_worker_job_duration_seconds` | histogram | `service`、`env`、`worker`、`operation`、`status` | worker / job 处理耗时 |
| `zhicore_mq_consumer_lag` | gauge | `service`、`env`、`consumer`、`eventType` | consumer backlog / lag 摘要 |
| `zhicore_mq_dead_letter_total` | counter | `service`、`env`、`consumer`、`eventType`、`reason` | dead-letter 计数 |
| `zhicore_outbox_publish_total` | counter | `service`、`env`、`eventType`、`result`、`reason` | outbox dispatcher 发布结果计数 |
| `zhicore_outbox_publish_duration_seconds` | histogram | `service`、`env`、`eventType`、`result` | outbox dispatcher publish confirm 耗时 |
| `zhicore_outbox_retry_total` | counter | `service`、`env`、`eventType`、`reason` | outbox 事件进入 retry 的次数 |
| `zhicore_outbox_pending_total` | gauge | `service`、`env`、`eventType` | outbox 当前待发送事件数 |
| `zhicore_outbox_oldest_pending_seconds` | gauge | `service`、`env`、`eventType` | outbox 最老待发送事件年龄 |
| `zhicore_outbox_dead_total` | counter | `service`、`env`、`eventType`、`reason` | outbox 事件超过重试阈值后进入 dead 的计数 |
| `zhicore_outbox_claim_stale_total` | counter | `service`、`env` | dispatcher 重新接管过期 claim 的计数 |

`status` 标签必须使用稳定枚举。下游 client 推荐值：

```text
success
timeout
canceled
network_error
rate_limited
dependency_4xx
dependency_5xx
dependency_unavailable
circuit_open
degraded
business_error
unknown
```

outbox publish 的 `result` 必须使用稳定枚举：

```text
success
timeout
nack
returned
network_error
broker_unavailable
confirm_timeout
serialization_error
unknown
```

规则：

- counter 只递增；当前状态使用 gauge。
- duration 使用 seconds 语义；日志字段可以继续用 `durationMs` 便于人工阅读。
- histogram bucket 由 exporter 或部署配置决定，但必须覆盖服务 timeout 附近的延迟区间。
- `errorCode` 没有公开错误码时使用稳定内部分类，例如 `DEPENDENCY_TIMEOUT`，不要使用原始错误文本。
- retry 指标同时保留每次 attempt 的日志字段和总 retry 次数；最终业务结果仍记录到 `zhicore_client_requests_total`。
- outbox publish 失败率不能单独代表事件可靠性；必须同时看 pending 数、最老 pending 年龄、dead 数、retry 次数和 confirm latency。

## 运行期观测方式

运行期问题按三层观测：

1. 日志定位单次请求或任务：用 `requestId` / `traceId` 串起 Gateway、业务服务、repository、下游 client 和 worker，查看 `provider`、`operation`、`status`、`durationMs`、`attempt`、`errorCode`、`circuitState`、`degradeStrategy`。
2. Metrics 判断趋势和阈值：按 `service`、`provider`、`operation` 聚合请求量、错误率、timeout rate、retry rate、p95 / p99 latency、熔断打开次数、降级次数和 in-flight。
3. Trace 还原跨服务路径：当前阶段可以先通过 `X-Request-Id` / `X-Trace-Id` 传播实现；未来接入 OpenTelemetry 时沿用相同 operation 和关联 ID。

最小 dashboard 视图：

- 服务入口：HTTP QPS、错误率、p95 / p99 latency、in-flight，按 `operation` 拆分。
- 下游依赖：client QPS、错误率、timeout rate、retry rate、p95 / p99 latency，按 `provider + operation` 拆分。
- resilience：`zhicore_client_circuit_state`、`zhicore_client_circuit_open_total`、`zhicore_client_degraded_total`、`zhicore_client_inflight`。
- 异步处理：worker 成功 / 失败 / retry / dead-letter、consumer lag、处理耗时。
- outbox 发布：publish 成功 / 失败 / nack / returned / confirm timeout、retry、pending、oldest pending、dead、stale claim 和 publish p95 / p99 latency。

最小告警信号只定义“应被看见”的事件，不在本文件固化具体平台规则：

- 核心路径 provider 熔断打开。
- 核心路径出现持续降级或降级次数快速增长。
- 下游失败率、timeout rate 或 retry rate 持续高于服务 README / 运维配置中记录的阈值。
- HTTP `5xx`、`SERVICE_DEGRADED` 或 `DEPENDENCY_UNAVAILABLE` 持续出现。
- worker dead-letter 增加、consumer lag 持续增长或 outbox 长时间无法清空。
- outbox dead 数增加。
- outbox pending 持续增长，或 oldest pending 超过服务 README / 运维配置中记录的阈值。
- outbox publish failure / timeout / nack / returned rate 持续升高。
- outbox publish p95 / p99 latency 接近 RabbitMQ publish confirm timeout。

## 脱敏规则

禁止写入日志、metrics label 或 trace attribute：

- `Authorization` header、cookie、JWT、session、token、验证码。
- 密码、secret、private key、access key、refresh token。
- 完整请求 body、完整文件 URL、对象存储签名 URL。
- 生产 DSN、连接串密码、云厂商凭证。

允许记录脱敏摘要：

- 文件大小、MIME type、扩展名、hash 前缀。
- 资源 ID、公开 ID、业务状态。
- URL 的 host、path 模板和 provider 名，不记录 query 中的签名参数。
- 配置摘要中的非敏感字段，例如 timeout、pool size、enabled。

脱敏逻辑优先复用 `libs/kit/observability` 或 `libs/kit/config` 的通用 helper。服务私有敏感字段由服务 owner 明确补充。

## `libs/kit/observability` 边界

允许放入：

- logger 初始化、字段名常量、上下文取放 `requestId` / `traceId` 的小工具。
- HTTP middleware 的通用日志字段抽取。
- metrics recorder interface、no-op recorder、低基数 label 校验 helper。
- trace header 提取 / 注入 helper。
- 通用脱敏 helper 和日志字段 builder。

禁止放入：

- 服务私有 operation 名、业务错误码清单、路由表、SQL 名称、bucket 名。
- 具体日志平台、告警平台或部署环境的硬编码配置。
- 会替代业务 owner 决策的通用 retry、fallback、panic recovery 或 `SafeGo` 语义。
- 为了“日志方便”把业务 DTO、数据库实体或请求 body 复制进共享库。

共享库只提供原语；服务负责决定哪些行为需要记录、哪些错误要上报、哪些业务 ID 可出现。

## 必须记录的边界

- 服务启动：service、version、env、listen address、关键依赖摘要、配置脱敏摘要。
- 服务停机：shutdown reason、timeout、未完成任务数量或资源关闭结果。
- HTTP 请求完成：method、route、status、durationMs、requestId、traceId、operation。
- 未知错误、panic、HTTP 5xx：operation、traceId/requestId、errorCode、内部错误摘要。
- 下游调用：provider、operation、status、durationMs、attempt、errorCode。
- 熔断和降级：provider、operation、circuitState、degradeStrategy、status、durationMs、requestId、traceId、errorCode。
- worker / consumer：eventId、eventType、consumer、attempt、durationMs、幂等处理结果。
- migration / 管理任务执行：命令、目标服务、目标版本、结果；不得记录生产连接串。

## 测试和验证

修改日志、metrics、trace、脱敏或 `libs/kit/observability` 时：

- 优先补最窄单元测试，覆盖字段名、trace 传播、脱敏和低基数 label 规则。
- 修改 HTTP middleware 时，测试 `X-Request-Id` / `X-Trace-Id` 传入、生成和响应回传。
- 修改 worker / consumer trace 处理时，测试 metadata 缺失和存在两种路径。
- 涉及 `libs/kit/observability` 时，运行 `cd libs/kit && go test ./...`。
- 涉及单个服务 handler、worker 或 adapter 时，运行该服务最窄相关 `go test`。
- 仅修改文档、索引或结构检查时，运行 `bash scripts/check-structure.sh`；交付前按 `docs/reviews/quality-gates.md` 选择是否运行 `make check`。
