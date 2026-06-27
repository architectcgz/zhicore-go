# 运行期操作规则

本文件定义 `zhicore-go` 服务运行期的启动、健康检查、超时、重试和幂等规则。

仓库目录和服务内分层见 `docs/architecture/repository-layout.md` 和 `docs/architecture/go-service-design.md`。配置和环境变量规则见 `docs/architecture/configuration.md`。日志、metrics 和 trace 规则见 `docs/architecture/observability.md`。错误分层和错误处置规则见 `docs/architecture/error-handling.md`。

## 适用范围

本文件适用于：

- `services/<service>/cmd/server` 进程入口。
- `services/<service>/internal/<domain>/runtime` 运行时组装。
- HTTP server、Gateway、worker、consumer、dispatcher 和定时任务。
- PostgreSQL、Redis、RabbitMQ、MongoDB、Elasticsearch、对象存储和外部 HTTP client。

## 配置规范

服务配置、环境变量、配置模板、必填校验、密钥处理和 `libs/kit/config` 边界以 `docs/architecture/configuration.md` 为准。

运行期规则只依赖已经解析和校验过的配置。启动路径不得在 handler、domain、repository、client adapter 或普通构造函数中临时读取环境变量。

## 启动流程

每个服务启动顺序：

1. 读取并校验配置。
2. 初始化 logger、requestId/traceId 基础设施。
3. 建立数据库、Redis、RabbitMQ、外部 client 等运行时依赖。
4. 构建 `runtime.Module`。
5. 注册 HTTP handler、consumer、worker 和 dispatcher。
6. 启动 health endpoint 和主服务。
7. 记录服务名、版本、环境、监听地址和关键依赖摘要。

规则：

- 启动路径不得自动执行 schema migration 或自动建表改表；正式 schema 演进由 `docs/architecture/migrations.md` 规定的 `golang-migrate` 流程负责。
- 关键依赖不可用时，服务应启动失败或进入 not ready 状态；不要静默降级到错误行为。
- 可选依赖必须在配置和日志中明确标记为 optional。
- `cmd/server/main.go` 不承载业务 wiring，业务组装放在 `internal/<domain>/runtime`。

## 构造和外部副作用

普通 `New...` / `NewService` / `NewRepository` 构造函数只接收已经解析好的依赖、配置和 identity，不主动访问外部世界。

禁止在普通构造函数中隐藏以下行为：

- 读取环境变量、hostname、文件、证书或本机网络状态。
- 打开真实数据库、Redis、RabbitMQ、MongoDB、Elasticsearch、对象存储或 HTTP client 连接。
- 读取当前时间、生成随机 secret / token / identity，或调用外部服务探测默认值。
- 忽略外部读取错误后静默使用空值或默认值。

需要外部事实时，必须放到名字和职责明确的 owner 层，例如 `cmd/server`、`runtime`、`Load*`、`Dial*`、`Open*`、`Build*` 或显式 factory。失败必须向上返回错误，或由调用方明确记录并决定 fallback。

测试中通过依赖注入传入 clock、random、hostname、client、文件内容或配置值，不把本机环境变成行为断言的一部分。

## 优雅停机

服务必须监听 `SIGINT` 和 `SIGTERM`。

停机顺序：

1. 标记 readiness 为 false，停止接收新流量。
2. 停止接收新的队列消息、定时任务和 outbox claim。
3. 等待正在处理的 HTTP 请求、consumer、worker 在 shutdown timeout 内完成。
4. 对未完成任务按业务语义执行 nack、重新入队、释放 claim 或记录可恢复状态。
5. 关闭 HTTP server、数据库连接池、Redis、RabbitMQ 和其他 client。
6. 记录停机结果。

规则：

- HTTP server 必须配置 `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout` 和 `IdleTimeout`。
- shutdown timeout 必须显式配置，默认不超过 `30s`。
- consumer ack 只能在业务处理和必要持久化完成后执行。
- worker claim 必须有过期或可恢复机制，避免进程崩溃导致任务永久卡住。

## 健康检查

每个可部署服务至少提供：

```text
GET /health/live
GET /health/ready
```

`/health/live`：

- 只表示进程仍可响应。
- 不检查数据库、Redis、RabbitMQ 或外部服务。
- 用于进程存活探测。

`/health/ready`：

- 表示服务可以接收业务流量。
- 检查必需依赖，例如数据库连接、Redis、RabbitMQ channel 或关键外部服务配置。
- 检查不应执行昂贵查询，不应写入业务数据。
- 依赖短暂失败时返回非 ready，并记录限频日志。

响应形态可以不使用业务 `ApiResponse` envelope；健康检查优先保持简单、稳定、适合反向代理和部署系统读取。

## 超时

所有外部边界必须有超时：

- HTTP server：read、write、idle 和 shutdown timeout。
- HTTP client：总 timeout，并可按下游 provider 设置更细粒度的 connect/read timeout。
- 数据库：查询使用带 deadline 的 `context.Context`。
- Redis：dial/read/write timeout。
- RabbitMQ：连接、publish confirm、consumer shutdown timeout。
- 对象存储和 File Service：上传、删除、URL 解析分别配置 timeout。

默认建议：

| 场景 | 建议默认 |
| --- | --- |
| 普通下游 HTTP 查询 | `2s` 到 `5s` |
| 写操作或文件元数据操作 | `5s` 到 `10s` |
| 文件上传 | 按大小和部署环境单独配置 |
| 数据库普通查询 | `1s` 到 `3s` |
| shutdown timeout | `15s` 到 `30s` |

具体服务可以调整，但必须在服务 README 或服务级配置中说明理由。

## Context 传播

服务、repository、handler、job、worker、consumer、runner 和 checker 等运行边界必须显式接收并继续传递 `context.Context`。

规则：

- HTTP handler 从 `r.Context()` 进入 application、repository、cache、MQ 和下游 client。
- 后台 worker / consumer 使用服务 lifecycle context，不把某个请求 context 存进 struct 或跨请求复用。
- 正常 application / infrastructure 代码不得自行创建 `context.Background()` 或 `context.TODO()` 来补缺失参数。
- `context.Background()` 只允许出现在进程根、框架入口、测试根或明确的生命周期根。
- 派生 timeout / deadline / cancel context 后必须调用返回的 cancel 函数，通常使用 `defer cancel()`。
- 需要脱离请求继续执行的异步任务，必须显式说明它使用的是 lifecycle context、任务 context 还是补偿任务 context。

## 重试

重试只适用于明确可重试且具备幂等保障的操作。

允许重试：

- 只读查询。
- 带 idempotency key 的写操作。
- outbox publish。
- consumer 处理中的可恢复下游失败。
- 获取锁、短暂网络错误、临时 `503` / timeout。

禁止盲目重试：

- 没有幂等键的创建、支付、发消息、发送通知、上传提交等写操作。
- 事务已提交但响应丢失的操作，除非可以通过唯一键或业务状态查询确认结果。
- 下游明确返回业务拒绝，例如权限不足、参数错误、状态不允许。

重试规则：

- 使用指数退避和 jitter。
- 设置最大尝试次数和最大耗时。
- 每次重试记录 provider、operation、attempt、durationMs 和最终结果。
- 最终失败要按 `docs/architecture/error-handling.md` 的错误处置规则返回、上报或进入补偿。
- retry / backoff / cancel / pacing 是控制流语义，不能依赖 logger、metrics、trace、debug flag 等可选观测组件是否存在。观测组件只能影响“记录什么”，不能影响“失败后怎么跑”。

## 下游 client resilience policy

每个同步下游 client adapter 必须在 runtime wiring 时声明 resilience policy，不能把 timeout、retry、熔断或降级策略散落在 handler、application 或 repository 中。

policy 至少包含：

- `provider`：下游名称，例如 `zhicore-content`、`postgres`、`rabbitmq`、`file-service`。
- `operation`：稳定操作名，和日志、metrics、trace 使用同一套命名。
- `timeout`：单次业务调用总耗时上限；不得超过上游请求 deadline。
- `retry`：最大尝试次数、退避、jitter 和可重试错误分类。
- `circuitBreaker`：统计窗口、最小请求数、失败阈值、打开时长和半开探测数。
- `maxInFlight`：可选但建议配置，防止下游慢调用或重试耗尽本服务 goroutine / 连接池。
- `degradeStrategy`：由 application 使用的降级策略标识；adapter 只返回错误和观测字段。
- `idempotency`：写操作是否具备幂等键、唯一约束、outbox / inbox 或状态机保障。

默认基线：

| 调用类型 | timeout | retry | 熔断 | 降级 |
| --- | --- | --- | --- | --- |
| 普通只读查询 | `2s` 到 `5s` | 最多 2 次总尝试 | 开启 | application 可选择降级错误、旧缓存或 contract 允许的空结果 |
| 聚合查询 / facade 查询 | `2s` 到 `5s` | 最多 2 次总尝试 | 开启 | 默认返回降级错误；只有响应 schema 明确支持 partial 时才返回部分结果 |
| 幂等写操作 | `5s` 到 `10s` | 具备幂等保障时最多 2 次总尝试 | 开启 | 核心写路径返回失败，不伪造成功 |
| 非幂等写操作 | 按业务配置 | 不重试 | 开启 | 返回失败或进入人工/补偿流程 |
| outbox publish / consumer 下游调用 | 按 broker / provider 配置 | 按 outbox / consumer retry 策略 | 可开启 | 进入 retry、dead-letter 或补偿 |
| best-effort 实时提示、通知 fanout | 短 timeout，例如 `100ms` 到 `1s` | 不阻塞主请求重试 | 开启 | 记录日志和 metrics，按业务决定丢弃、延后或单独 retry |

这些默认值不是生产最终答案。服务首次实现必须先有保守默认值和显式配置项，上线或压测后再根据观测结果调整。

## 后台任务和 goroutine owner

每个 goroutine、worker、consumer、dispatcher、poller 和 reconcile loop 必须先明确 owner：

- 谁启动它。
- 谁负责取消、等待和释放资源。
- 错误如何传播或记录。
- panic 后果是什么。

panic recovery 只能放在明确 owner 边界内，不新增跨服务通用 `SafeGo` / `safe goroutine` 默认包装来替 owner 决定失败语义。

常见语义：

- HTTP 请求边界可以 recover，返回 500 并记录 request 上下文。
- root 级关键后台任务 panic 通常是严重进程问题，应至少记录现场后重新 panic，不能静默退出。
- 定时任务或 reconcile 单轮执行可以由 owner 本地 recover，但语义应是“本轮失败，下一轮可重试”。
- 有业务状态的异步任务应把 panic 转成业务失败状态，例如导入任务、报表任务或批处理任务标记失败。
- 只用于等待 `WaitGroup.Wait()` 的辅助 goroutine 保持最小显式写法，不挂 recover 语义。

如果 goroutine 没有明确 logger、ctx、task name、业务失败状态或生命周期 owner，先修 owner 接线，不用 no-op logger、`context.Background()` 或共享 recover helper 掩盖问题。

## 熔断和降级

当前个人项目阶段不强制引入复杂熔断库，但所有下游 client 必须保留以下扩展点：

- timeout。
- retry policy。
- 错误分类。
- 失败计数或指标标签。
- 熔断状态查询或观测字段。

当某个下游持续失败时：

- 核心写路径优先失败返回，不伪造成功。
- 查询类接口可以返回降级错误或明确的空结果，但必须由 application 决定。
- adapter 不得自行构造业务 DTO 作为降级结果。
- 降级行为必须可观测，至少有结构化日志和错误计数。

熔断按 `provider + operation` 维度统计；除非同一个 provider 的所有操作共享相同容量和失败域，否则不要用一个全局开关熔断整个 provider。

计入失败率的错误：

- connect refused、DNS / TLS / 网络错误、连接池耗尽。
- timeout、deadline exceeded、RabbitMQ publish confirm timeout。
- 下游 HTTP `5xx`、`429` 或明确的资源耗尽 / 限流错误。
- 下游 client 返回的 `ErrDependencyUnavailable`、`ErrCircuitOpen` 或等价临时不可用错误。

不计入下游熔断失败率的错误：

- 参数错误、权限不足、资源不存在、状态不允许等确定性业务拒绝。
- 当前请求 context 因客户端断开或上游主动取消而结束；这类错误记录为 `canceled`，不用于判断 provider 健康。
- 本服务配置错误或序列化错误；这类问题应 fail fast 或作为本服务 bug 处理。

推荐初始熔断参数：

| 参数 | 默认建议 | 说明 |
| --- | --- | --- |
| 统计窗口 | `30s` 滚动窗口 | 低 QPS 服务可放宽到 `60s` |
| 最小请求数 | `20` | 未达到最小样本数不按失败率打开熔断 |
| 连续失败阈值 | `5` | 低 QPS 或冷启动时避免等待窗口填满 |
| 失败率阈值 | `50%` | 只统计上面的临时失败分类 |
| 打开时长 | `30s` | 连续打开可指数退避，最高不超过 `5m`，并记录原因 |
| 半开探测数 | `3` | 半开只允许少量请求通过 |
| 恢复条件 | 半开探测全部成功 | 任一探测失败则重新打开 |

熔断统计以一次业务调用的最终结果为主；每次 attempt 仍要记录 metrics，但不要让一次业务调用的多次 retry 把失败率放大成多次独立业务失败。

失败阈值不能等线上事故发生后才第一次设置。真实场景的做法是：

- 首次实现时使用上面的保守默认值，并让所有参数可配置。
- 压测或灰度期间观察 p95 / p99 latency、timeout rate、retry rate、失败率、熔断打开次数和降级次数。
- 根据调用方 SLO、下游容量、流量大小和错误预算调参；高流量核心路径可以更快熔断，低流量后台任务需要更高最小样本或连续失败阈值。
- 如果熔断打开会导致核心业务不可用，应优先缩短 timeout、限制并发和减少 retry，避免雪崩；不要简单把失败率阈值调高来掩盖问题。
- 参数调整原因应记录在服务 README、配置模板或运维变更记录中。

降级策略只能由 application 选择。允许的策略：

- `fail-fast`：返回公开降级错误，例如 `SERVICE_DEGRADED` 或 provider 对应公开错误码。
- `stale-cache`：返回明确允许的旧缓存；必须能标记数据时间或在内部日志中记录旧值来源。
- `partial-response`：只在 HTTP contract 明确支持部分结果时使用。
- `async-compensation`：主事务完成后副作用失败，进入 outbox、retry、dead-letter 或补偿。
- `best-effort-skip`：通知、实时提示、非关键统计等可恢复副作用失败时记录并跳过。

禁止的降级：

- 把失败伪装成业务成功。
- adapter 自行构造空 DTO、空列表或默认值并让 application 误以为下游成功。
- 对权限、资源归属、支付、写入一致性等核心校验做静默跳过。
- 在 facade 查询中把 provider 不可用伪装成“用户没有数据”；例如 User facade 获取用户发表文章时，Content 查询失败应暴露为降级错误，而不是返回空列表。

## 幂等

幂等必须在拥有副作用的边界设计，而不是事后靠重试掩盖。

常见幂等方式：

- HTTP 写请求：`Idempotency-Key`、业务唯一键或自然唯一约束。
- 数据库写入：唯一索引、状态机条件更新、乐观锁版本。
- outbox：`event_id` 唯一。
- consumer：inbox / ledger / `event_id` 唯一约束。
- 上传：文件 hash、外部 File Service 返回的 file id 或业务引用唯一约束。
- 定时任务：任务 key、claim token、lease 过期时间。

要求：

- 每个可能重复提交的写 use case 必须明确幂等策略。
- consumer 必须容忍重复消息、乱序和迟到消息。
- outbox dispatcher 必须保证同一事件多次 publish 不会导致 consumer 产生不可恢复副作用。
- outbox / inbox / saga / ledger 这类 durable journal 的状态迁移必须使用条件更新或等价 compare-and-set，例如只允许 `pending -> sent`、`pending -> failed`；不能让 stale worker 按 id 盲写，把已完成记录复活回待处理状态。
- 幂等冲突应返回稳定错误码或返回已有结果，不能随机失败。

## 服务运行完成标准

一个服务进入可运行状态前，至少补齐：

- 配置结构和必填配置校验。
- HTTP server timeout 和 graceful shutdown。
- `/health/live` 和 `/health/ready`。
- 下游 client timeout。
- 写路径幂等策略。
- worker/consumer ack、retry、dead-letter 或补偿策略。
- 结构化日志字段和 operation 命名。
- 服务 README 中的本地运行命令、配置示例和依赖说明。

未补齐的运行期能力必须记录到 `docs/todos/debt/`，并写明影响和退出条件。
