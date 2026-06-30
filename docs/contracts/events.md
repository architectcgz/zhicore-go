# 事件契约

本文件定义 RabbitMQ 事件 contract 的通用规则。具体事件 payload 放在 `libs/contracts/events/<domain>/`。

## Broker 约定

Go 目标统一使用 RabbitMQ：

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

## RabbitMQ 拓扑声明与配置归属

跨服务事件拓扑必须版本化声明，不能只靠人工在 RabbitMQ 控制台创建，也不能把 exchange、queue、binding、DLX 或 retry 规则隐式散在业务代码里。

推荐分层：

```text
项目仓库：声明 exchange / queue / binding / DLX / retry / routing key 规则
部署/IaC：按环境创建这些 RabbitMQ 资源
应用代码：使用这些名字，并在启动时校验或幂等 declare
```

规则：

- 事件 contract 记录 `eventType`、routing key、payload version、producer、已知 consumer 和兼容性要求。
- RabbitMQ topology 必须在项目内有可审查来源，至少记录 exchange、queue、binding、DLX、retry queue、TTL、durable、quorum/classic、prefetch 和 `Single Active Consumer` 等关键参数。
- 生产和共享环境优先由部署流程创建拓扑，例如 RabbitMQ definitions、Terraform、Helm、Operator 或版本化部署脚本；不要依赖一次性手工点击。
- 应用启动可以做 passive declare / 校验：确认必需 exchange、queue 和 binding 存在，参数不匹配时 fail fast。
- 本地开发、测试或临时环境可以允许应用幂等 declare 拓扑；生产是否允许自动 declare 必须由部署策略显式决定，避免代码静默改动运行拓扑。
- RabbitMQ 默认 exchange（空字符串 `""`）只适合简单点对点或本地临时场景；跨服务业务事件必须显式使用已声明的 exchange、queue 和 binding。

Owner 约定：

| 拓扑元素 | Owner | 说明 |
| --- | --- | --- |
| 共享事件 exchange | 平台 / 项目事件规范 | 当前公共跨服务事件默认使用 `zhicore.events` topic exchange。 |
| producer routing key | producer 服务 | routing key 表达 producer 发布的业务事实，必须和事件 contract 同步演进。 |
| consumer queue | consumer 服务 | 每个消费服务拥有自己的 queue，不能让多个业务服务共享同一个消费状态。 |
| binding | consumer 服务 | binding 表达“我订阅哪些事件”，由 consumer 设计和部署声明。 |
| retry queue / DLQ / DLX | consumer 服务 | 重试、dead-letter 和补偿语义跟随 consumer 的处理策略和幂等存储。 |
| vhost、账号、权限、broker 地址 | 部署 / 运维配置 | 环境差异不进入事件 payload contract，必须通过配置和部署资产管理。 |

变更 RabbitMQ topology 时，必须同步事件 contract、受影响服务文档和部署资产。新增 binding 或 queue 通常是 consumer 变更；新增事件类型或 routing key 通常是 producer contract 变更；修改 DLX、retry、TTL、prefetch 或 single-active 语义属于 consumer 运行策略变更。

## 有序事件分区

默认事件 contract 仍要求 consumer 容忍重复、乱序和迟到消息。只有当某个事件族明确需要“同一业务对象局部有序”时，才启用分区有序方案；该方案是优化投递和处理冲突，不替代 `eventId` 幂等、状态机校验、`aggregateVersion` 或 `occurredAt` 乱序保护。

ZhiCore Go 采用固定 `64` 个逻辑分片作为有序事件族的默认方案：

```text
partition = hash(partitionKey) % 64
queue = <consumer>.<event-family>.p<partition>
```

规则：

- `partitionKey` 必须是稳定业务对象 ID，例如 `accountId`、`postId`、`commentId`、`recipientId`；不能使用 worker 实例编号、Pod 名称、delivery tag 或运行时 consumer 数量。
- `64` 是长期稳定的逻辑分片数，不是当前 worker 实例数。日常扩缩容 worker 不改变 `partition = hash(key) % 64` 的结果。
- 同一分片队列同一时间只能有一个 active consumer；可使用 RabbitMQ `Single Active Consumer`，或由部署配置保证每个队列只有一个活跃消费线程。
- 严格局部有序时，分片队列使用 `manual ack`，业务处理和必要持久化提交后再 ack；同一队列内不得并发完成多条消息。
- `prefetch` 默认按严格有序取 `1`。如果某个事件族把有序性降级为冲突优化而非正确性前提，必须在对应服务文档说明更高 `prefetch` / 并发的理由和乱序保护。
- 失败重试不能让后续同分片消息越过失败消息；若选择 DLQ 后继续处理，必须在服务文档说明补偿、告警和状态机兜底。

不要使用 `hash(key) % workerCount`。例如原来部署 `5` 个 worker，后来增加到 `6` 个 worker，如果路由按 worker 数取模，大量 key 会被重新映射，旧队列积压和新队列新消息会并行，破坏同一 key 的顺序。固定 64 分片时，`hash(key) % 64 = 6` 的消息始终进入第 `6` 个逻辑分片队列；新增 worker 只可能改变“由哪个 worker 实例成为该队列的 active consumer”，不改变消息进入哪个分片。

真实 worker 数量属于部署容量，不属于事件 contract。文档和配置应记录 `partitionCount=64`、队列命名、`partitionKey`、`Single Active Consumer` / 单活策略、`prefetch`、ack / retry / DLQ 语义；具体线上当前部署了几个 worker 只记录在部署清单、运行配置或运维面板中，不写成路由规则。

## 事件归属

- 事件归属跟随数据归属。
- Provider 拥有事件名称、payload、版本和兼容性。
- Consumer 只能依赖 `libs/contracts/events/<domain>/` 中公布的稳定字段。
- 事件 payload 不包含 provider 私有数据库 row、repository filter 或内部 command。

## 消息字段和落库映射

- RabbitMQ 消息体是 JSON payload，字段使用 lowerCamelCase，例如 `eventId`、`occurredAt`、`payloadVersion`。
- outbox / inbox / ledger 等落库字段属于对应服务的 migration/schema；记录事件 metadata 时使用 snake_case，例如 `event_id`、`occurred_at`、`payload_version`。
- JSON 字段和落库字段的对应关系属于事件 contract 与服务 schema 的关联规则，不能由 consumer 自行猜测。

## Envelope

新事件推荐使用统一 envelope：

```json
{
  "eventId": "uuid-or-public-event-id",
  "eventType": "content.post.published",
  "payloadVersion": 1,
  "producer": "zhicore-content",
  "occurredAt": "2026-06-22T10:30:00Z",
  "aggregateType": "post",
  "aggregateId": "123",
  "aggregateVersion": 7,
  "correlationId": "optional",
  "causationId": "optional",
  "requestId": "optional-request-id",
  "traceId": "optional-trace-id",
  "payload": {}
}
```

规则：

- `eventId` 全局唯一，consumer 用它做消费幂等。
- `occurredAt` 是业务事实发生时间，不是发送时间或消费时间。
- `payloadVersion` 从 `1` 开始，破坏性变化必须新增版本或新事件类型。
- `aggregateId` 可以是字符串，以兼容内部 bigint 和外部公开 ID。
- `correlationId` / `causationId` 描述业务事件链路；`requestId` / `traceId` 只用于观测关联，规则见 `docs/architecture/observability.md`。
- producer 有请求或任务上下文时应携带 `requestId` / `traceId`；consumer 必须容忍缺失，不能用它们做幂等、权限或业务分支判断。
- 落库时 `eventId` 映射到 `event_id`，`occurredAt` 映射到 `occurred_at`，`payloadVersion` 映射到 `payload_version`。

承接已发布事件时，如果 payload 已被 consumer 使用，先保持语义兼容，再逐步收敛到统一 envelope；Java 事件定义只作为核对历史语义的参考。

## 可靠性

关键跨服务事实必须使用 producer outbox：

```text
业务表 + outbox 同事务提交
-> dispatcher claim pending event
-> publish RabbitMQ
-> update outbox status / retry / dead
-> consumer idempotent handling
```

Consumer 要求：

- 消费时以 JSON `eventId` 作为幂等键；落库到 inbox、ledger 或业务唯一约束时使用 `event_id`。
- 容忍重复消息。
- 容忍乱序和迟到消息。
- 不直接修改 provider 写模型。

只有明确可丢弃的通知型事件可以标为 best-effort；标记位置必须在事件 contract 中说明。

## 兼容性

兼容变更：

- 新增可选字段。
- 新增 consumer 可忽略的 metadata。
- 新增可选观测字段，例如 `requestId` 或 `traceId`。
- 新增事件类型。

破坏性变更：

- 重命名字段。
- 改变字段类型或语义。
- 复用旧事件名表达新事实。
- 删除仍被 consumer 使用的字段。

破坏性变化优先新增新事件类型，例如 `content.post.visibility_changed.v2`，而不是原地改旧事件。
