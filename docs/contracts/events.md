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
