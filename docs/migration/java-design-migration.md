# Java 设计迁移盘点

本文件记录从 `../zhicore-microservice` 迁移到 `zhicore-go` 时，Java 侧设计中哪些保留、哪些改写、哪些废弃。

## 事实来源

已核对的 Java 来源：

- 根 `pom.xml` 的模块清单和技术栈。
- `docs/architecture/01-system-overview.md`
- `docs/architecture/02-microservices-list.md`
- `docs/architecture/04-service-communication.md`
- `docs/architecture/05-ddd-layered-architecture.md`
- `docs/architecture/06-data-architecture.md`
- `docs/architecture/content-service-design.md`
- `docs/architecture/blog-message-im-integration.md`
- `docs/architecture/zhicore-notification-platform-design.md`
- `docs/architecture/zhicore-ranking-detailed-design.md`
- `zhicore-client/src/main/java`
- `zhicore-integration/src/main/java`
- 各服务 `src/main/resources/application*.yml`
- 各服务 `src/main/resources/db/schema.sql`
- `database/init-all-databases.sql`
- `docker/postgres-init/01-init-databases.sql`
- `docker/postgres-init/02-init-tables.sql`

只读 Java 仓库；不在 Java 仓库内修复历史问题。

## 保留的设计

### 服务边界

保留“每个业务服务独立部署、独立数据归属、通过 contract 通信”的方向。

目标 Go 服务仍按 Java 模块落到：

- `zhicore-gateway`
- `zhicore-user`
- `zhicore-content`
- `zhicore-comment`
- `zhicore-message`
- `zhicore-notification`
- `zhicore-search`
- `zhicore-ranking`
- `zhicore-admin`
- `zhicore-upload`
- `zhicore-id-generator`
- `zhicore-ops`

但 `zhicore-id-generator` 不作为当前默认核心依赖，`zhicore-ops` 也不作为普通业务服务。

### 分层和依赖方向

保留 Java 侧 DDD / ports 思路：

- 接口层只处理传输协议、认证上下文、参数绑定和响应转换。
- 应用层编排用例、事务、端口、事件和外部服务调用。
- 领域层保留核心规则、实体、值对象和领域事件。
- 基础设施层实现数据库、缓存、队列、搜索、对象存储和服务调用 adapter。

Go 侧不照搬 Java 包名。仓库和服务目录按 `docs/architecture/repository-layout.md` 落位，服务内分层按 `docs/architecture/go-service-design.md` 执行。

### 数据存储分工

保留 Java 侧的多存储职责划分：

- PostgreSQL：业务实体、关系、统计、outbox、ledger、审计。
- Redis：缓存、计数器、锁、热榜查询缓存和临时状态。
- MongoDB：文章正文、文档型投影、排行历史归档。
- Elasticsearch：搜索索引、搜索建议和搜索历史相关读模型。

Go 侧必须用显式 migration 管理 schema，不在服务启动路径做自动建表或自动改表。

### 可靠事件方向

保留 `业务事务 -> outbox -> MQ -> consumer idempotency` 方向。

Java 侧已经在 `content`、`user`、`message` 等模块出现 outbox / outbox task 思路；Go 侧应把它标准化：

- 关键跨服务事实必须和业务真相源同事务写入 outbox。
- Dispatcher 异步 claim、投递 RabbitMQ、记录重试和 dead 状态。
- Consumer 通过 `consumed_events`、inbox、ledger 或业务唯一约束处理重复消息。
- Consumer 必须容忍重复、乱序和迟到事件。

### 查询和读模型

保留“归属服务拥有写模型，Search / Ranking 拥有读模型”的设计：

- Search 只拥有 Elasticsearch 索引和搜索相关本地读模型。
- Ranking 只拥有分数、榜单、ledger、snapshot 和归档。
- User / Content / Comment 等归属服务仍是实体事实源。

## 必须改写的设计

### Nacos 和服务发现

Java 文档和配置大量依赖 Nacos 服务注册与配置中心。

Go 目标运行环境优先面向 Kubernetes：

- 服务发现默认使用 Kubernetes Service DNS。
- 配置默认通过 env、ConfigMap、Secret 注入。
- 不把 Nacos 作为 Go 服务的必需组件。

### Feign Client

Java 的 `zhicore-client` 提供 Feign Client、DTO 和事件定义。

Go 侧改写为：

- `libs/contracts/clients/<provider-service>`：provider 拥有的同步 client contract。
- `libs/contracts/events/<domain>`：provider 拥有的事件 payload。
- 调用方服务内部定义 consumer-side port。
- HTTP client、fallback、重试、熔断等实现放调用方 adapter，不放共享 contract。

### RocketMQ

当前 Java 源码主要使用 RocketMQ，部分历史文档提到 RabbitMQ。

Go 侧已确定使用 RabbitMQ：

- 目标 exchange：`zhicore.events`
- 类型：`topic`
- routing key：`<domain>.<event>`，例如 `content.post.published`

迁移时只保留事件语义，不保留 RocketMQ topic/tag/consumer group 的技术形态。

### ID 生成

Java 侧多处服务直接调用 `IdGeneratorFeignClient` 获取 Snowflake ID，包括 User、Content、Comment、Message、Notification、Admin 等。

Go 侧默认策略见 `docs/architecture/id-strategy.md`：

- 内部主键使用各服务数据库 `BIGINT` sequence / identity。
- 外部公开 ID 使用独立 `public_id`、`public_no`、`order_no`。
- `zhicore-id-generator` 仅保留为未来集中发号的可选落点。

### Sentinel

Java 中的 Sentinel 规则、resource name、fallback handler 不直接迁移。

Go 侧先保留能力目标：

- HTTP middleware 层可加限流和熔断。
- service/client adapter 可加超时、重试、熔断、指标。
- 具体库和策略等到服务实现时按风险选择，不提前引入复杂治理。

### 灰度迁移

Java `zhicore-ops` 和 Gateway 包含灰度配置、用户灰度和回滚接口。

当前开发阶段已经确定：

- 不做灰度。
- Java/Go 不运行时并存。
- 前端暂时不修改。
- Gateway 只做路由或环境切换，不承载 API 形态转换。

因此灰度相关设计不迁移为 Go 当前事实源。

## 服务迁移分析

### Gateway

保留：

- 统一外部入口。
- JWT 校验。
- token 黑名单和校验缓存。
- CORS 和路由。

改写：

- Spring Cloud Gateway 路由改为 Go HTTP gateway / reverse proxy / ingress 配置配合。
- 灰度路由不迁移。
- Nacos 服务发现改为 Kubernetes DNS 或本地配置。

### User

保留：

- 认证、用户资料、角色、关注、拉黑、签到。
- `users`、`roles`、`user_roles`、`user_follows`、`user_follow_stats`、`user_blocks`、`user_check_ins`、`user_check_in_stats` 等表归 User。
- 用户资料摘要和批量查询作为 provider contract。
- 用户事件 outbox 方向。

改写：

- 注册和创建用户 ID 改为数据库 sequence。
- 关注、拉黑等事件改用 RabbitMQ payload。
- 用户发表文章查询仍由 Content 拥有；User facade 必须委托 Content。

### Content

保留：

- 文章元数据、标签、分类、互动、统计和作者快照归 Content。
- PostgreSQL 保存元数据和关系，MongoDB 保存正文或文档型投影。
- `owner_name`、`owner_avatar_id`、`owner_profile_version` 是作者快照。
- scheduled publish、outbox、consumed events、MongoDB sync 和 cache-aside 思路。

改写：

- 文章、标签、点赞、收藏 ID 默认使用数据库 sequence。
- `outbox_event` 改投 RabbitMQ。
- Java schema 存在来源漂移，Go 迁移前必须先定每个服务自己的 migration 源。
- `topic_id` 在拆出 Topic 服务前仍由 Content 管理。

### Comment

保留：

- 评论、回复、评论点赞、评论统计归 Comment。
- 分页、游标、增量查询、首页热缓存等读优化方向可以保留。
- Comment 可消费 Content / Ranking / Upload / User 的 provider contract，但不拥有它们的数据。

需要纠偏：

- Java 历史中 comment 事件有直接发 MQ 的路径。Go 侧关键评论事件应补齐 producer outbox，避免业务提交成功但消息丢失。
- 评论图片、语音资源只保存 `file_id`，文件资源仍归 Upload。

### Message

保留：

- 私信会话、私信消息、消息未读统计归 Message。
- 与外部 IM 系统的 bridge / gateway 作为 Message 自己的 adapter。
- `message_outbox_task` 的事务后异步同步方向。
- Message 可以聚合私信会话和通知摘要，但不拥有 Notification 的通知事实。

改写：

- `MessageIdGenerator` 默认改为数据库 sequence 或 repository 返回 ID。
- 外部 IM 失败、补偿和重试语义迁移时要单独建行为测试。

### Notification

保留：

- 通知 inbox、聚合状态、偏好、免打扰、作者订阅、全局公告、assistant message。
- campaign / shard / delivery 的大规模 fanout 设计方向。
- 未读数和聚合通知缓存。

改写：

- 通知、campaign、shard、delivery ID 改为数据库 sequence。
- 消费事件改为 RabbitMQ。
- 具体渠道投递 adapter 在 Go 侧按接口拆分，不把渠道逻辑塞进领域层。

### Search

保留：

- Search 只拥有 Elasticsearch index、suggestion/history 读模型。
- 消费 Content 的发布、更新、删除、标签更新等事件。
- 调用 Content provider contract 获取必要文章详情。

改写：

- RocketMQ consumer 改为 RabbitMQ consumer。
- 索引初始化和重建流程要放入明确运维命令或服务内部 job，不在普通请求路径隐式执行。

### Ranking

保留：

- Ranking 是读模型服务，不拥有文章、评论、用户源数据。
- Redis 保存当前榜单和热查询缓存。
- PostgreSQL 保存 ledger、delta bucket、当前状态和周期分数。
- MongoDB 保存历史榜单归档和冷数据查询。
- 事件消费需要幂等、可 replay。

改写：

- Redis 不是唯一真相，Go 侧文档和实现必须保留从 PostgreSQL / MongoDB 重建的能力。
- RocketMQ consumer 改为 RabbitMQ consumer。
- 文章详情补全仍调用 Content contract，不直接查 Content 数据库。

### Admin

保留：

- Admin 拥有举报、举报处理流程、审计日志和管理编排。
- 用户、文章、评论管理路由是 facade，mutation 必须委托归属服务。

改写：

- 审计 ID、举报 ID 默认使用 Admin 数据库 sequence。
- 不依赖 IdGenerator。

### Upload

保留：

- Upload 是文件资源边界，其他服务只保存 `file_id`。
- 图片、音频、批量上传、URL 解析、删除能力保留。

需要确认：

- Go 侧是继续适配外部 file-service，还是 Upload 自己持久化文件元数据。
- 当前先按 adapter 设计，不把对象存储实现泄漏给业务服务。

### IdGenerator

保留：

- 服务目录和 Java 迁移映射。
- 未来集中发号的可选落点。

不作为当前默认依赖：

- 不作为第一批迁移目标。
- 普通业务实体不通过它发号。

### Ops

保留：

- 迁移对账、回滚记录、CDC 修复、内部运维任务。

不迁移：

- 用户灰度。
- 灰度配置。
- Gateway 灰度判断。

## 已发现的设计风险

### Schema 来源漂移

Java 侧至少存在以下 schema 来源：

- `database/init-all-databases.sql`
- `docker/postgres-init/02-init-tables.sql`
- 各服务 `src/main/resources/db/schema.sql`

这些来源之间存在表结构、字段、命名和归属差异。Go 迁移不能直接全量复制某一个脚本，必须按服务拆分 versioned migrations，并把每个服务的 migration 作为该服务唯一 schema 事实源。

### 共享 contract 存量过宽

Java `zhicore-client` 中存在共享 Feign client、DTO、fallback factory 和 ID Generator client。

Go 侧必须避免：

- 把调用方 fallback 放入共享 contract。
- 把 provider 的内部 DTO 复制给 consumer 私用。
- 让 consumer 通过共享包绕过 provider 归属边界。

### 事件可靠性不一致

Java 侧不同服务事件发布可靠性不一致：

- Content 已经有较完整的 outbox 方向。
- User 有 outbox 模型。
- Message 有 `message_outbox_task`。
- Comment 历史上存在直接发 MQ 的链路。

Go 侧关键跨服务事实应统一为 producer outbox；若某事件被定义为 best-effort，必须在服务文档中显式标注。

### 历史技术选型文档不一致

Java 源码主路径是 RocketMQ，但历史设计文档中也出现 RabbitMQ。Go 侧按当前决策统一使用 RabbitMQ，迁移时只继承事件语义，不继承 RocketMQ 技术细节。

### ID 策略和 Java 现状冲突

Java 多服务调用 IdGenerator，Go 侧默认使用数据库 sequence。迁移实现时需要特别注意：

- 不要把 `IdGeneratorFeignClient` 等价迁移到 Go 服务。
- 新表的主键 migration 要显式定义 identity / sequence。
- 对外 ID 要按 `docs/architecture/id-strategy.md` 单独设计。

## 后续迁移切片

推荐按以下顺序继续把设计迁移到 Go：

1. 外部 API 清单：从 Java controller 和 `zhicore-client` 提取每个服务必须保留的 HTTP contract，落到 `services/<service>/api/http`。
2. Schema 清单：按服务拆分 Java 表结构，生成 Go 目标 migration 草案，并修正 ID 策略。
3. 事件清单：从 `zhicore-integration` 和消费端提取事件 payload、routing key、producer、consumer、幂等存储。
4. Content 设计迁移：优先迁移文章元数据、正文存储、作者快照、outbox 和 scheduled publish。
5. User / Upload 迁移：作为相对清晰的写服务和基础服务，建立第一批 Go 行为测试和运行链路。
