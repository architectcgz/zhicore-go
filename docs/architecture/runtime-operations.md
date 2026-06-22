# 运行期操作规则

本文件定义 `zhicore-go` 服务运行期的配置、启动、健康检查、超时、重试和幂等规则。

仓库目录和服务内分层见 `docs/architecture/repository-layout.md` 和 `docs/architecture/go-service-design.md`。错误分层、日志和上报规则见 `docs/architecture/error-handling.md`。

## 适用范围

本文件适用于：

- `services/<service>/cmd/server` 进程入口。
- `services/<service>/internal/<domain>/runtime` 运行时组装。
- HTTP server、Gateway、worker、consumer、dispatcher 和定时任务。
- PostgreSQL、Redis、RabbitMQ、MongoDB、Elasticsearch、对象存储和外部 HTTP client。

## 配置规范

配置来源优先级：

1. 启动参数或明确传入的测试配置。
2. 环境变量。
3. 本地开发配置文件模板。
4. 代码内安全默认值。

规则：

- 生产密钥、JWT secret、数据库密码、对象存储凭证、外部服务 token 只能来自环境变量或 Secret，不写入仓库。
- 配置模板可以提交，但只能包含示例值、空值或本地开发默认值。
- 服务启动时必须校验必填配置，缺失时直接启动失败，并给出可操作错误信息。
- 端口、超时、最大请求体、数据库连接池大小、Redis/RabbitMQ 地址、外部服务 base URL 都属于显式配置。
- 不允许在业务代码中散写环境变量名；配置读取集中在 runtime 或 `libs/kit/config`。
- 配置字段命名使用稳定前缀，例如 `ZHICORE_UPLOAD_HTTP_ADDR`、`ZHICORE_GATEWAY_JWT_SECRET`。

本地开发阶段可以使用 `.env.example`、`configs/local.example.*` 或 README 记录示例配置；真实 `.env` 和包含凭证的配置文件不得提交。

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

- 启动路径不得自动执行 schema migration 或自动建表改表。
- 关键依赖不可用时，服务应启动失败或进入 not ready 状态；不要静默降级到错误行为。
- 可选依赖必须在配置和日志中明确标记为 optional。
- `cmd/server/main.go` 不承载业务 wiring，业务组装放在 `internal/<domain>/runtime`。

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

## 熔断和降级

当前个人项目阶段不强制引入复杂熔断库，但所有下游 client 必须保留以下扩展点：

- timeout。
- retry policy。
- 错误分类。
- 失败计数或指标标签。

当某个下游持续失败时：

- 核心写路径优先失败返回，不伪造成功。
- 查询类接口可以返回降级错误或明确的空结果，但必须由 application 决定。
- adapter 不得自行构造业务 DTO 作为降级结果。
- 降级行为必须可观测，至少有结构化日志和错误计数。

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
