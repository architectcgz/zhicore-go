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

### 拓扑初始化落点

本仓库 RabbitMQ 拓扑统一由以下文件声明：

```text
deploy/docker/
├── docker-compose.yml                     # 本地开发编排，启动 RabbitMQ 并加载 definitions
├── rabbitmq/
│   ├── rabbitmq.conf                      # RabbitMQ 配置，启用 management plugin 并加载 definitions
│   └── definitions.json                   # exchange、queue、binding 的版本化声明
└── README.md
```

分层职责：

- **`definitions.json`**：拓扑唯一事实源，声明所有 exchange、queue 和 binding。新增 consumer queue 或 binding 必须在此文件中同步更新。
- **`docker-compose.yml` + `rabbitmq.conf`**：本地开发环境按 definitions 初始化 RabbitMQ 拓扑。
- **生产环境**：由部署流程（Terraform、Helm、Operator 或 RabbitMQ definitions import）按同一 `definitions.json` 或等价声明创建拓扑，不依赖应用首次启动的隐式 declare。
- **应用启动**：按 `docs/architecture/runtime-operations.md` 做 passive declare 校验——确认必需 exchange、queue、binding 存在且参数一致，不匹配则 fail fast。

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

## Consumer 命名

ZhiCore Go 中每个需要标识身份的 consumer 必须使用统一命名。命名覆盖两种角色：

### Outbox Dispatcher 命名

Outbox dispatcher 是各服务内从 `outbox_events` 表 claim 事件并发布到 RabbitMQ 的后台 worker。其名称用于：

- `outbox_events.claimed_by`：标识当前 claim 了某批 outbox 事件的 dispatcher 实例。
- 日志和 trace 的 operation 名称：例如 `zhicore-content:outbox-dispatcher:publish`。
- 可观测指标标签：`dispatcher` label。

**格式**：

```text
<service-name>:outbox-dispatcher
```

运行时实例标识追加粒度由各服务本地决定，推荐：

```text
<service-name>:outbox-dispatcher:<host>:<worker-id>
```

示例：

- `zhicore-content:outbox-dispatcher`
- `zhicore-auth:outbox-dispatcher`
- `zhicore-user:outbox-dispatcher`
- `zhicore-comment:outbox-dispatcher`

只有 producer 服务需要 outbox dispatcher。纯 consumer 服务（Search、Ranking、Notification）不产生 outbox 事件，不需要 outbox dispatcher。

### RabbitMQ Consumer 命名

RabbitMQ consumer 是订阅 RabbitMQ queue 并处理入站事件的 worker。其名称用于：

- queue 命名中的 `<consumer>` 片段：`<consumer>.<event-family>.p<partition>`。
- DLX、retry queue 等衍生资源命名。
- consumer group 和部署标识。

**格式**：

```text
<service-name>:<event-family>-consumer
```

示例：

| 服务 | 消费的事件族 | RabbitMQ Consumer |
| --- | --- | --- |
| `zhicore-content` | `user.profile.*` | `zhicore-content:user-profile-consumer` |
| `zhicore-content` | `comment.*` | `zhicore-content:comment-consumer` |
| `zhicore-search` | `content.post.*` | `zhicore-search:content-post-consumer` |
| `zhicore-ranking` | `content.post.*` | `zhicore-ranking:content-post-consumer` |
| `zhicore-ranking` | `comment.*` | `zhicore-ranking:comment-consumer` |
| `zhicore-notification` | `content.post.*` | `zhicore-notification:content-post-consumer` |
| `zhicore-notification` | `comment.*` | `zhicore-notification:comment-consumer` |
| `zhicore-notification` | `user.*` | `zhicore-notification:user-consumer` |

### 各模块完整 Consumer 命名清单

| 服务 | Outbox Dispatcher | RabbitMQ Consumer（按事件族） |
| --- | --- | --- |
| `zhicore-content` | `zhicore-content:outbox-dispatcher` | `zhicore-content:user-profile-consumer`、`zhicore-content:comment-consumer` |
| `zhicore-auth` | `zhicore-auth:outbox-dispatcher` | （Auth 当前主要作为 producer，不作为事件 consumer） |
| `zhicore-user` | `zhicore-user:outbox-dispatcher` | （User 当前主要作为 producer，不作为事件 consumer） |
| `zhicore-comment` | `zhicore-comment:outbox-dispatcher` | （Comment 当前主要作为 producer，不作为事件 consumer） |
| `zhicore-search` | （无，纯 consumer 服务） | `zhicore-search:content-post-consumer` |
| `zhicore-ranking` | （无，纯 consumer 服务） | `zhicore-ranking:content-post-consumer`、`zhicore-ranking:comment-consumer` |
| `zhicore-notification` | （无，纯 consumer 服务） | `zhicore-notification:content-post-consumer`、`zhicore-notification:comment-consumer`、`zhicore-notification:user-consumer` |

各模块的具体事件生产/消费关系和事件 payload 登记见对应 `libs/contracts/events/<domain>/` 和模块 `data-events.md`。

### Consumer 实例数建议

以下从消息数量、单条处理成本和吞吐需求三个维度，分析每个 consumer 的推荐实例数。

**分析维度**：

| 维度 | 说明 |
| --- | --- |
| 消息量 | 事件产生的频率。`viewed` / `liked` 属于高频，`published` / `deleted` 属于低频。 |
| 处理成本 | 单条事件的耗时。涉及外部 HTTP 调用或 ES 写入为**重**，纯 PostgreSQL 写入为**轻**。 |
| 推荐实例 | `1` = 单实例足够；`2` = 建议至少 2 个实例竞争消费。 |

**实例数不是部署硬编码**，是水平扩展的起点参考。实际扩缩容由消息积压、consumer lag 和 CPU/内存指标驱动。

---

#### `zhicore-search:content-post-consumer` → **2 实例**

| 因素 | 分析 |
| --- | --- |
| 消息量 | 中低。`published` / `updated` / `deleted` 是人工操作，频率远低于互动事件。 |
| 处理成本 | **重**。`published` / `updated` 需回源 Content 拉取全文 body（HTTP 调用），再写入 Elasticsearch。单条耗时 100–500ms。 |
| 吞吐瓶颈 | 串行时一条慢 ES bulk 会阻塞后续所有事件。2 实例可并行处理，避免 publish 积压。 |

#### `zhicore-ranking:content-post-consumer` → **2 实例**

| 因素 | 分析 |
| --- | --- |
| 消息量 | **高**。消费 `viewed`、`liked`、`favorited` 等高频互动事件，其中 `viewed` 是全站最高频事件。 |
| 处理成本 | 轻。写入 `ranking_event_ledger` + upsert `ranking_delta_bucket`，纯 PostgreSQL 事务，单条 5–20ms。 |
| 吞吐瓶颈 | `viewed` 去重逻辑增加少量开销，但核心瓶颈是消息到达速率而非单条成本。高峰期大量互动可能撑满单实例。 |

#### `zhicore-ranking:comment-consumer` → **1 实例**

| 因素 | 分析 |
| --- | --- |
| 消息量 | 中。`comment.created` / `comment.deleted`，频率低于点赞/浏览。 |
| 处理成本 | 轻。写入 COMMENT delta 到 bucket，纯 PostgreSQL 写入。 |
| 理由 | 当前阶段评论量级不会成为瓶颈。若未来评论量大幅增长，可扩至 2。 |

#### `zhicore-notification:content-post-consumer` → **2 实例**

| 因素 | 分析 |
| --- | --- |
| 消息量 | **高**。`content.post.liked` 是全站最高频的通知触发事件。`content.post.published` 触发 campaign 规划但频率低。 |
| 处理成本 | 中重。创建通知涉及 `consumed_events` 幂等写入 + `notifications` 插入 + `notification_group_state` upsert + 未读数缓存更新 + 实时 fanout，单条 10–30ms。 |
| 吞吐瓶颈 | 点赞高峰时通知创建是典型写密集型路径，多条 DB 操作串行累积。2 实例避免点赞通知被低频的 campaign 规划阻塞。 |

#### `zhicore-notification:comment-consumer` → **1 实例**

| 因素 | 分析 |
| --- | --- |
| 消息量 | 中。`comment.created` 触发通知（通知文章作者或被回复者）。`comment.deleted` 更新通知状态。评论频率低于点赞。 |
| 处理成本 | 中。和点赞通知类似的多步写入，但可能需要回查 Content/Comment 补齐定位字段。 |
| 理由 | 当前阶段评论量级不足以需要 2 实例。若评论量大幅增长，可按需扩至 2。 |

#### `zhicore-notification:user-consumer` → **1 实例**

| 因素 | 分析 |
| --- | --- |
| 消息量 | **低**。`user.followed` 是极低频事件。 |
| 处理成本 | 轻。创建单条关注通知。 |
| 理由 | 关注行为日均量极低，1 实例远未饱和。 |

#### `zhicore-content:user-profile-consumer` → **1 实例**

| 因素 | 分析 |
| --- | --- |
| 消息量 | **低**。`user.profile.updated` 是极低频事件（用户改昵称/头像/简介）。 |
| 处理成本 | 轻。`UPDATE posts SET owner_snapshot ... WHERE owner_id = ?`，单条 SQL。 |
| 理由 | 低频 + 轻量，1 实例足够。 |

#### `zhicore-content:comment-consumer` → **1 实例**

| 因素 | 分析 |
| --- | --- |
| 消息量 | 中。`comment.created` / `comment.deleted` 触发 `post_stats.comment_count` 原子增减。 |
| 处理成本 | 轻。`UPDATE post_stats SET comment_count = comment_count ± 1`，单条原子 SQL。 |
| 理由 | 纯 PG 原子更新，当前阶段 1 实例足够。若评论量大幅增长，可扩至 2。 |

---

**汇总**：

| Consumer | 实例数 | 关键原因 |
| --- | --- | --- |
| `zhicore-search:content-post-consumer` | **2** | 回源 Content + ES 写入重，避免串行阻塞 |
| `zhicore-ranking:content-post-consumer` | **2** | `viewed` / `liked` 高频，高峰期消息量大 |
| `zhicore-ranking:comment-consumer` | 1 | 评论量中，纯 PG 写入轻 |
| `zhicore-notification:content-post-consumer` | **2** | 点赞高频 + 多步 DB 写入，高峰压力大 |
| `zhicore-notification:comment-consumer` | 1 | 评论量中，当前可单实例 |
| `zhicore-notification:user-consumer` | 1 | 关注极低频 |
| `zhicore-content:user-profile-consumer` | 1 | profile 更新极低频 |
| `zhicore-content:comment-consumer` | 1 | 原子 PG 更新，当前量级足够 |

所有 `1 实例` consumer 在消息量增长后均可按需扩至 2。实例数调整不改变拓扑（queue/binding/DLQ 不变），只改变部署副本数。

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
- 不强制每个 consumer 都建立独立 inbox 表，但每条消费链路必须声明自己的幂等边界。幂等边界可以是 inbox、业务 ledger、`source_event_id` 唯一约束、状态版本号或覆盖写；只要能证明重复投递不会制造重复副作用即可。
- 如果事件处理包含多个副作用，必须保证这些副作用在同一个幂等边界下提交，或分别有稳定幂等键保护；不能只让其中一步幂等，却让计数、通知、缓存刷新或外部调用在重复投递时重复生效。
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
