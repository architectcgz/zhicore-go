# Go 服务设计规则

本文件记录 `zhicore-go` 的 Go 服务内分层、依赖方向、运行时组装、持久化和事件规则。

仓库目录和服务目录模板见 `docs/architecture/repository-layout.md`。本文件不作为目录树事实源。

## 设计目标

- 服务内业务代码保持框架无关，技术实现集中在 `infrastructure`。
- 每个服务可独立构建、测试、部署和迁移。
- 服务之间通过 provider-owned contract、RabbitMQ 事件或明确的 HTTP client 通信。
- 外部 API 形态默认保持稳定，前端暂时不修改。

## 服务内分层职责

完整目录落点见 `docs/architecture/repository-layout.md`。服务内分层职责如下：

| 层 | 典型路径 | 职责 |
| --- | --- | --- |
| HTTP 入站层 | `services/<service>/api/http` | 路由注册、handler、请求 DTO、响应 DTO、参数校验、认证上下文映射和外部 API 兼容。 |
| Application | `services/<service>/internal/<domain>/application` | use case、事务编排、权限上下文、幂等、事件写入和端口调用。 |
| Domain | `services/<service>/internal/<domain>/domain` | 实体、值对象、领域规则、领域事件和领域错误。 |
| Ports | `services/<service>/internal/<domain>/ports` | application 需要的 consumer-side interface，例如 repository、cache、lock、event publisher、外部服务 client、clock。 |
| Infrastructure | `services/<service>/internal/<domain>/infrastructure` | PostgreSQL、Redis、RabbitMQ、MongoDB、Elasticsearch、对象存储和同步 HTTP client 实现。 |
| Runtime | `services/<service>/internal/<domain>/runtime` | 服务内部组装入口，创建 infrastructure、application、HTTP handler、worker 和 consumer。 |

`api/http` 放在服务根目录下，不放进 `internal/<domain>`。它仍然属于本服务边界，可以导入本服务的 application，但不能直接访问数据库、缓存、MQ 或外部 SDK。

HTTP 入站路由统一使用 Gin 组装。Gin 只属于 `api/http` 和进程 runtime 挂载边界；`*gin.Context` 不得传入 application、domain、ports 或 infrastructure。进入 application 前，handler 必须把请求解析成显式 `context.Context`、command、query 或 typed actor/principal，避免业务层依赖 Web 框架生命周期和参数容器。

Application 对外暴露给 `api/http`、runtime 或其他入站 adapter 的类型必须是 application 自有 DTO / command / query；不得用导出的 type alias 重新暴露 domain 类型，例如 `type UserID = domain.UserID` 或给 domain import 起别名后的等价写法。需要跨层传递领域值时，在 application 内部用显式 mapper / 类型转换进入 domain，避免入站层绕过 application 边界直接拿到 domain contract。

## 依赖方向

允许：

```text
api/http -> application -> domain
application -> ports
infrastructure/postgres -> ports/domain model mapping
infrastructure/rabbitmq -> application use case
infrastructure/clients -> ports implementation
runtime -> api/http/application/infrastructure wiring
```

禁止：

- `domain` 依赖 HTTP、PostgreSQL、Redis、RabbitMQ、MongoDB、Elasticsearch 或 Kubernetes。
- consumer 服务导入 provider 服务的 `internal` 包。
- shared contract 中放 fallback、重试、熔断、缓存或调用方业务策略。
- 在 handler 中直接写数据库或发 MQ，绕过 application use case。

## 组装入口

每个服务模块优先使用 `runtime/module.go` 作为组装入口，而不是把 wiring 散落在 `cmd/server/main.go`。

推荐形态：

```go
package runtime

type Deps struct {
	Config Config
	DB     *sql.DB
	Cache  RedisClient
	Broker RabbitMQConnection
	Logger Logger
}

type Module struct {
	HTTPHandler http.Handler
	Consumers   []Consumer
	Workers     []Worker
}

func Build(deps Deps) (*Module, error) {
	// 创建 infrastructure 实现
	// 创建 application use case
	// 创建 HTTP handler / consumer / worker
	// 返回可由 cmd/server 挂载的模块
}
```

`cmd/server/main.go` 只负责进程入口：

```go
func main() {
	cfg := loadConfig()
	deps := openRuntimeDeps(cfg)
	module, err := runtime.Build(deps)
	if err != nil {
		log.Fatal(err)
	}
	run(module)
}
```

如果某个服务将来在一个进程内组合多个业务模块，可以再引入进程级 `composition` 包；当前 `zhicore-go` 以 `services/<service>` 独立进程为主，不需要提前新增全局 `internal/app/composition`。

## Interface 放置规则

Go 里 interface 默认放消费方，而不是放实现方，也不是默认放 `domain`。

推荐：

- application 需要持久化、缓存、锁、MQ、外部服务、时间源时，在 `ports` 定义小而明确的 consumer-side interface。
- infrastructure 包实现这些 interface。
- domain 尽量保持纯业务模型，不依赖 repository、Redis、HTTP client、MQ publisher 这类外部能力。
- 只有某个抽象本身就是领域概念，并且 domain 的纯业务规则确实需要它时，才考虑放入 `domain`。

不要把“仓储接口必须属于领域层”当成默认规则。多数 ZhiCore 用例由 application 编排事务和持久化，因此 repository interface 默认属于 `ports`。

## Contract 放置

同步调用：

```text
libs/contracts/clients/<provider-service>/
```

事件：

```text
libs/contracts/events/<domain>/
```

规则：

- Provider 拥有 contract。
- Consumer 可以依赖 contract，但 consumer 内部仍定义自己的 port。
- 同步 HTTP client 的 path、caller operation、请求 DTO 和响应 DTO 放在 provider contract 目录；consumer adapter 只引用 contract 并适配自己的 port。
- Provider DTO、数据库实体、repository filter、内部 command/query 不进入 `libs/contracts`。
- 外部 HTTP API schema 放在 `services/<service>/api/http`，必须保持当前 API 兼容基线。
- HTTP envelope、错误、数据类型、分页和事件的通用契约规则见 `docs/contracts/` 下的专题文档。

## 运行时依赖约定

服务配置和环境变量规则见 `docs/architecture/configuration.md`；日志、metrics 和 trace 规则见 `docs/architecture/observability.md`；启动、健康检查、优雅停机、超时、重试、熔断和幂等规则见 `docs/architecture/runtime-operations.md`。

| 能力 | 约定 |
| --- | --- |
| 服务发现 | 当前阶段使用本地配置或环境变量；进入 Kubernetes 后再使用 Kubernetes Service DNS |
| 配置注入 | 当前阶段使用 env 和本地配置模板；进入 Kubernetes 后再映射为 ConfigMap、Secret |
| 边缘入口 | 当前阶段使用薄 Go Gateway 作为应用入口，Nginx 或本地反向代理放在其前面；Kubernetes Ingress 只在部署进入 Kubernetes 后使用 |
| 同步调用 | `libs/contracts/clients` + 调用方 `infrastructure/clients` 实现 |
| 异步消息 | RabbitMQ topic exchange |
| 限流和熔断 | Go middleware、client timeout/retry/circuit breaker、指标告警 |
| 日志 | 结构化日志；生产环境默认 JSON，本地开发可用文本格式；字段和脱敏规则见 `docs/architecture/observability.md` |
| 链路追踪 | 当前阶段至少传递 `X-Request-Id` / `X-Trace-Id`；进入统一观测后再接 OpenTelemetry |
| 指标 | 当前阶段先保留稳定 operation 名称和低基数标签；接入 Prometheus 后按服务、operation、status、errorCode 聚合 |
| 数据访问 | 显式 SQL、repository 实现、必要时使用轻量 query helper |
| 对象映射 | 显式 struct、构造函数和 mapper 函数 |
| 内部主键 | PostgreSQL `BIGINT GENERATED BY DEFAULT AS IDENTITY`，外部 ID 单独设计 |

## 命名和映射归属

- Go 内部包名、类型名、字段名、变量名和 receiver 名属于代码风格，默认遵循 Go 习惯和服务内现有写法。
- JSON、HTTP query/path/body 字段、事件 payload 字段属于 contract，按 `docs/contracts/` 中对应专题文档定义。
- 数据库列、outbox / inbox / ledger 表字段属于 migration/schema，默认使用 snake_case；承接已发布数据结构时，以服务级 migration 和数据归属设计为准。
- Redis key 片段和 RabbitMQ routing key 属于运行时契约，由对应服务的 resolver、event contract 或服务文档统一定义。
- Go struct 字段、JSON 字段和数据库列允许命名不同，但必须通过 `json` tag、显式 mapper、SQL alias 或 adapter 转换固定映射，不依赖隐式命名猜测。

## 数据库和 migration

数据库 migration 工具、文件命名、事务、down migration、seed 和 GORM 边界见 `docs/architecture/migrations.md`。

每个服务自己的 schema 放在：

```text
services/<service>/migrations/
```

核心规则：

- migration 是该服务 schema 的唯一事实源。
- 正式 migration 使用 `golang-migrate` 管理 SQL migration。
- 不在服务启动路径执行自动建表或自动改表。
- 不使用 GORM `AutoMigrate` 作为正式 schema 演进方式。
- 不把全量初始化脚本原样复制到每个服务库。
- 导入既有表结构时必须先确认数据归属，再拆成服务本地 migration。
- 内部主键默认 `BIGINT GENERATED BY DEFAULT AS IDENTITY`。
- 对外公开字段按 `docs/architecture/id-strategy.md` 设计并加唯一索引。

示例：

```sql
CREATE TABLE posts (
  id BIGINT PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
  public_id VARCHAR(32) NOT NULL UNIQUE,
  owner_id BIGINT NOT NULL,
  title VARCHAR(200) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## 缓存规则

默认使用 cache-aside：

- 读：先读缓存，未命中回源数据库，再写缓存。
- 写：先提交数据库，再删除相关缓存。
- 需要避免缓存穿透时，可以使用三态缓存或短 TTL 空值。
- 缓存 key 必须由服务本地 resolver 统一生成，不在 handler 或 repository 中散写字符串。

Redis key 命名建议：

```text
<service>:<entity>:<id>:<field>
```

例如：

```text
content:post:123:detail
user:profile:456:simple
ranking:posts:hot:daily
```

## RabbitMQ 事件规则

统一使用：

```text
exchange: zhicore.events
type: topic
routing key: <domain>.<event>
```

示例：

```text
content.post.published
content.post.liked
comment.created
user.profile.updated
```

事件名称、payload 版本、envelope、outbox 和兼容性规则见 `docs/contracts/events.md`。

关键跨服务事实必须使用 producer outbox：

```text
业务表 + outbox 同事务提交
-> dispatcher claim pending event
-> publish RabbitMQ
-> update outbox status / retry / dead
-> consumer idempotent handling
```

**Dispatcher claim 模式（防崩溃安全）：**

outbox dispatcher 必须使用 claim 机制，避免多个实例重复发布同一批事件，也避免在 publish RabbitMQ 期间长时间持有数据库行锁。PostgreSQL 的 `UPDATE` 不支持直接 `ORDER BY ... LIMIT`，因此这里用 CTE 先确定本轮要 claim 的一批 `id`，再在同一个 SQL 语句中更新这些行；`FOR UPDATE SKIP LOCKED` 会跳过已被其他 dispatcher 锁定的行，从而让多实例并发 claim 时拿到互不重叠的 outbox 事件：

1. Dispatcher 用 PostgreSQL CTE 原子 claim 一批事件，示例：

```sql
WITH picked AS (
  SELECT id
  FROM outbox_events
  WHERE status = 'PENDING'
     OR (status = 'CLAIMING' AND claim_started_at < now() - $1::interval)
  ORDER BY id
  FOR UPDATE SKIP LOCKED
  LIMIT $2
)
UPDATE outbox_events AS e
SET claim_owner = $3,
    claim_started_at = now(),
    status = 'CLAIMING'
FROM picked
WHERE e.id = picked.id
RETURNING e.*;
```

2. `claim_owner` 是实例唯一标识（例如 `hostname:goroutine`），`claimTTL` 典型值 `30s`。
3. publish RabbitMQ 成功后，`UPDATE ... SET status = 'SENT', sent_at = now()` 释放 claim。
4. publish 失败，更新 retry metadata；超过阈值标记 `DEAD`。
5. 进程崩溃后，过了 `claimTTL` 的 `CLAIMING` 行自动被其他实例重新 claim（步骤1的 `claim_started_at < now() - $claimTTL` 条件）。
6. 状态迁移必须使用条件更新（`WHERE status = 'CLAIMING' AND claim_owner = $workerID`），不允许盲写覆盖已完成或已被其他实例接管的行。

每个有 outbox 的服务（Auth、User、Content、Comment）都必须遵守此 claim 模式。`FOR UPDATE SKIP LOCKED` 只能用于短事务内选中待 claim 行；claim 提交后再 publish RabbitMQ。不要在持有数据库行锁的事务里执行外部 publish，避免 broker 慢调用阻塞 outbox 表。

outbox dispatcher 必须按 `docs/architecture/observability.md` 暴露 publish result、publish confirm duration、retry、pending、oldest pending、dead 和 stale claim 指标。MQ 有 publish confirm / nack / returned 等确认信号，但可靠性判断不能只看发布失败率；pending 持续增长、最老 pending 变旧或 dead 增加，才表示 outbox 已经无法按预期清空，需要告警和排障。

Consumer 要求：

- 用事件 JSON 的 `eventId`，或落库后的 `event_id`、业务唯一约束保证幂等。
- 容忍重复消息。
- 容忍乱序和迟到消息。
- 不在消费端直接修改 provider 写模型。

只有明确可丢弃的通知型事件可以标为 best-effort；标记位置必须在服务文档或 event contract 中说明。

## 事务边界

应用层拥有事务边界。

常见写入规则：

- 创建/修改业务实体和写 outbox 必须在同一个数据库事务内。
- 业务数据库提交成功后，再异步删除缓存或投递外部消息。
- 不在事务内调用外部 HTTP 服务，除非已经明确接受阻塞和失败语义。
- 如果必须跨资源写入，例如 PostgreSQL + MongoDB，需要用可恢复状态、补偿任务或 outbox，而不是假装它们是一个本地事务。

## API Contract

Go 服务默认不破坏已发布外部 API contract：

- HTTP path
- method
- query/path/body 参数
- 响应封装
- 字段语义
- 错误码
- 权限行为

Go 服务可以在内部使用更清晰的 use case 和 port，但不能把未登记的 API 变化传递给现有前端。

需要重做的 API 必须作为独立 API 演进任务处理。

HTTP 协议层规则见 `docs/contracts/http.md`；错误响应和公开错误码见 `docs/contracts/errors.md`；时间、ID 和 JSON 字段序列化规则见 `docs/contracts/data-types.md`；分页、排序和过滤见 `docs/contracts/pagination.md`。

## 错误处理

Go 服务内部错误分层、底层错误翻译和 application 到 HTTP 的错误映射边界见 `docs/architecture/error-handling.md`。对外错误响应和公开错误码仍以 `docs/contracts/errors.md` 为准。

## 服务实现步骤

每实现一个服务，按以下顺序推进：

1. 明确外部 API 清单和已发布 contract 约束。
2. 按服务边界确认表归属和 migration。
3. 为 Go 服务补齐 `api/http`、`migrations/` 和必要 contract。
4. 先写 handler/use case/repository 的行为测试。
5. 实现 domain/application。
6. 实现 postgres/redis/rabbitmq/clients 等 infrastructure。
7. 用 system HTTP 测试证明已发布 API 或目标 Go-first schema 可调用。
8. 更新服务 README、contract 文档和必要服务替换记录。

## 当前开放点

- File metadata schema、MinIO adapter 和文件生命周期清理 worker 的实现切片。
- 每个服务的第一版 migration 需要逐服务核对数据归属和目标表结构。
