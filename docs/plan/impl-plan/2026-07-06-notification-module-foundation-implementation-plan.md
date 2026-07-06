# Notification 模块基础实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；涉及 notification public ID、migration、收件箱、已读、consumer、delivery、campaign、runtime 的步骤按 @test-driven-development 执行。每个 checkbox 达到预期后立即更新；如需提交，提交前必须先使用 @committing-changes。

**目标：** 把 `zhicore-notification` 从占位模块推进到拥有站内收件箱、聚合未读、已读操作、runtime、交互通知 consumer、偏好 / DND / delivery 和 campaign fanout 基础的可交付模块。

**架构：** Notification 拥有通知收件箱、group state、未读数、偏好、免打扰、作者订阅、campaign、delivery 和 realtime fanout 语义；源用户、文章、评论、私信和榜单事实仍由来源服务拥有。站内通知写入是权威事实，WebSocket / Email / SMS 只影响 delivery 状态，不回滚 inbox。

**技术栈：** Go 1.26、Gin、PostgreSQL、Redis、RabbitMQ、public ID codec、User / Content / Comment typed client、Notification HTTP schema。

---

## 背景依据

- `docs/architecture/services/notification/README.md`
- `services/zhicore-notification/api/http/README.md`
- `deploy/docker/rabbitmq/definitions.json`
- `libs/contracts/events/content/post-events.md`
- `libs/contracts/events/comment/comment-events.md`
- `libs/contracts/events/user/contract.go`
- `docs/architecture/service-boundaries.md`
- `docs/contracts/http-schema-template.md`
- `docs/architecture/runtime-operations.md`
- `docs/architecture/migrations.md`
- `docs/reviews/quality-gates.md`

## 当前基线

- 生产 Go 源码只有 `services/zhicore-notification/internal/notification/doc.go`。
- HTTP schema 是计划化占位，没有 `endpoints/`。
- 设计要求 `notificationId` 使用 `public_id`，但当前仓库没有可复用 public ID codec。
- User follower shard contract 只停留在 README 草案，campaign 不能直接开工。

## 不可并行修改文件

- `libs/contracts/clients/user/contract.go`：必须等 `2026-07-06-message-module-foundation-implementation-plan.md` 任务 1 合并后，再在任务 6 追加 follower shard contract。
- Public ID codec 落点：任务 1 的 migration / schema 开始前必须先完成本计划的 public ID 决策 checklist；未完成前不得新增 notification migration、`internal/notification/infrastructure/publicid/*.go` 或 `libs/kit/publicid/*`。

## Public ID codec 决策门槛

Notification 不能在未确认算法归属时直接实现一套不可替换的公开 ID 方案。任务 1 开始前必须完成：

- [x] 读取 `docs/architecture/id-strategy.md`、`docs/architecture/services/notification/README.md` 和 Content 当前 public ID 生成实现。
- [x] 若 Content 已有稳定、可复用且不含服务私有规则的算法，则先提取 `libs/kit/publicid`，并让 Notification 通过服务级 prefix / secret 配置复用。
- [x] 若 Content 当前实现仍是服务私有或不稳定，则本切片允许先落 `internal/notification/infrastructure/publicid`，但必须在代码注释和服务 README 写明这是 Notification 本地实现，未来提取到 `libs/kit/publicid` 前不得被其他服务复用。
- [x] 决策必须固定算法、落点、`public_id` column length、active version、secret version、错误分类和 redaction；无论选择本地还是共享，测试必须覆盖 version、secret 轮换、非法输入、round-trip、唯一性和不暴露内部自增 ID。

## 任务 1：通知中心首批 HTTP contract 与 inbox migration

**测试立场：** HTTP 文档 R0；migration contract 属于 R4，采用 TDD。

**文件：**
- 修改：`services/zhicore-notification/README.md`
- 修改：`services/zhicore-notification/api/http/README.md`
- 新增：`services/zhicore-notification/api/http/endpoints/list-notifications.md`
- 新增：`services/zhicore-notification/api/http/endpoints/mark-notification-read.md`
- 新增：`services/zhicore-notification/api/http/endpoints/mark-all-notifications-read.md`
- 新增：`services/zhicore-notification/api/http/endpoints/get-notification-unread-count.md`
- 新增：`services/zhicore-notification/api/http/endpoints/get-notification-unread-breakdown.md`
- 新增：`services/zhicore-notification/migrations/20260706xxxx_create_notification_inbox_core.up.sql`
- 新增：`services/zhicore-notification/migrations/20260706xxxx_create_notification_inbox_core.down.sql`
- 新增：`services/zhicore-notification/migrations/migration_contract_test.go`
- 新增或修改：`libs/kit/publicid/*` 或 `services/zhicore-notification/internal/notification/infrastructure/publicid/*`

**验收清单：**
- [x] 写明 `/read-all` 与 `/mark-all-read`、`/unread/count` 与 `/unread-count` 的 alias。
- [x] `notificationId` 是 `public_id` 字符串，不沿用 Java `Long`。
- [x] 已完成“Public ID codec 决策门槛”，并在 migration 中使用决策后的 `public_id` 长度、唯一索引和错误分类。
- [x] migration 创建 `notifications`、`notification_group_state`、`consumed_events`。
- [x] migration 包含 `dedupe_key` 唯一约束、`source_event_id` 幂等、group unread 非负约束、`expires_at` 清理索引。
- [x] migration up/down 可通过 `golang-migrate` 往返验证；文件名中的 `20260706xxxx` 必须在实施时替换为真实单调递增时间戳。
- [x] 聚合列表字段至少固定 `type`、`targetType`、`targetId`、`totalCount`、`unreadCount`、`latestTime`、`latestContent`、`recentActors` 或 `actorIds`、`aggregatedContent`。

- [x] **步骤 1：补 HTTP schema**
- [x] **步骤 2：写 migration 失败测试**
- [x] **步骤 3：实现 public ID codec 和 migration**
- [x] **步骤 4：运行验证**

运行：`cd services/zhicore-notification && go test ./migrations -count=1`

预期：通过。

## 任务 2：站内收件箱、已读和未读数最小闭环

**测试立场：** TDD - public ID、权限、幂等、缓存一致性属于 R4。

**文件：**
- 新增：`services/zhicore-notification/internal/notification/domain/inbox/notification.go`
- 新增：`services/zhicore-notification/internal/notification/domain/inbox/group_state.go`
- 新增：`services/zhicore-notification/internal/notification/domain/inbox/types.go`
- 新增：`services/zhicore-notification/internal/notification/ports/*.go`
- 新增：`services/zhicore-notification/internal/notification/application/commands/mark_notification_read.go`
- 新增：`services/zhicore-notification/internal/notification/application/commands/mark_all_notifications_read.go`
- 新增：`services/zhicore-notification/internal/notification/application/queries/*.go`
- 新增：`services/zhicore-notification/internal/notification/infrastructure/postgres/*.go`
- 新增：`services/zhicore-notification/internal/notification/infrastructure/redis/*.go`
- 新增：`services/zhicore-notification/internal/notification/infrastructure/publicid/*.go`
- 新增：`services/zhicore-notification/api/http/*.go`

**验收清单：**
- [x] `notificationId` 解析失败返回参数错误，不误判为 404。
- [x] 单条已读按 `public_id + recipient_id` 限权。
- [x] 重复已读幂等成功且不重复扣减。
- [x] `notification_group_state.unread_count` 不能为负。
- [x] 聚合列表优先读 group state；发现缺失或不一致时允许回退 DB 聚合并记录 repair signal。
- [x] Redis key 使用 `notification:{userId}:*`，read / read-all 后失效。
- [x] public ID codec 有 round-trip、版本 secret、唯一性和非法输入测试。
- [x] 已完成“Public ID codec 决策门槛”，并按决策选择 `libs/kit/publicid` 或 `internal/notification/infrastructure/publicid`。

- [x] **步骤 1：写 application、handler、repository、public ID 失败测试**
- [x] **步骤 2：实现 domain、ports、repository、cache 和 handler**
- [x] **步骤 3：运行 inbox 验证**

运行：`cd services/zhicore-notification && go test ./internal/notification/... ./api/http -run 'Notification|Unread|PublicID' -count=1`

预期：通过。

## 任务 3：Notification runtime、配置和 server

**测试立场：** TDD - 配置、脱敏、readiness、worker lifecycle 属于 R4。

**文件：**
- 新增：`services/zhicore-notification/cmd/server/config.go`
- 新增：`services/zhicore-notification/cmd/server/config_loader.go`
- 新增：`services/zhicore-notification/cmd/server/config_defaults.go`
- 新增：`services/zhicore-notification/cmd/server/config_validation.go`
- 新增：`services/zhicore-notification/cmd/server/server.go`
- 新增：`services/zhicore-notification/cmd/server/main.go`
- 新增：`services/zhicore-notification/internal/notification/runtime/module.go`
- 新增：`services/zhicore-notification/internal/notification/runtime/health_test.go`
- 新增：`services/zhicore-notification/configs/local.example.env`

**验收清单：**
- [ ] 必填配置覆盖 Postgres、Redis、RabbitMQ、`public_id.active_version`、`public_id.secrets`、`consumer.consumed_events_retention`、`realtime_fanout.timeout`、campaign claim / batch 参数。
- [ ] 配置日志和 error 不泄露 public ID secret、DSN、RabbitMQ URL、Redis credential。
- [ ] `/health/live` 不探下游。
- [ ] `/health/ready` 检查 Postgres、Redis、RabbitMQ 和 enabled worker descriptor；后续 consumer / campaign worker 增加时必须同步更新 runtime wiring、start / stop、panic recovery、readiness descriptor 和配置校验。
- [ ] 启动路径不自动执行 migration。

- [ ] **步骤 1：写 config / health / lifecycle 失败测试**
- [ ] **步骤 2：实现 runtime 和 server**
- [ ] **步骤 3：运行 runtime 验证**

运行：`cd services/zhicore-notification && go test ./cmd/server ./internal/notification/runtime -count=1`

预期：通过。

## 任务 4：交互通知 consumer、双层幂等和 realtime fanout

**测试立场：** TDD - consumer ack/nack、幂等、fanout 补偿属于 R4。

**文件：**
- 新增：`services/zhicore-notification/internal/notification/application/commands/create_interaction_notification.go`
- 新增：`services/zhicore-notification/internal/notification/application/consumers/content_post_consumer.go`
- 新增：`services/zhicore-notification/internal/notification/application/consumers/comment_consumer.go`
- 新增：`services/zhicore-notification/internal/notification/application/consumers/user_consumer.go`
- 新增：`services/zhicore-notification/internal/notification/application/jobs/cleanup_consumed_events.go`
- 新增：`services/zhicore-notification/internal/notification/infrastructure/rabbitmq/*.go`
- 新增：`services/zhicore-notification/internal/notification/infrastructure/clients/*.go`

**验收清单：**
- [ ] 消费 `content.post.liked`、`comment.created`、`user.followed`。
- [ ] `eventId` 和 `dedupeKey` 双层幂等均落库。
- [ ] duplicate event ack no-op；数据库事务提交成功后才 ack；已判定 no-op 的自交互 ack；临时依赖错误 nack / requeue；producer contract 错误写入 DLQ 后 ack；fanout 失败不 nack。
- [ ] 自己赞自己、自己关注自己、自己回复自己 no-op。
- [ ] `comment.created` 缺 `postAuthorId` / `parentAuthorId` 等必需字段时判 producer contract 错误并 retry / DLQ，不在高频路径静默补查。
- [ ] 本任务 fanout 只做站内 inbox 写入后的 WebSocket / unread hint best-effort，不执行 Email / SMS / digest delivery，不应用偏好 / DND；偏好、DND 和 delivery ledger 在任务 5 后才生效。
- [ ] 交互通知的 actor 展示首期只使用事件 payload 中的 actor ID / snapshot；本任务不新增 User summary contract，若需要实时用户摘要必须另开 User provider-owned 前置切片。
- [ ] DLQ envelope 至少包含 `eventId`、`eventType`、`routingKey`、`consumer`、`errorClass`、`failedAt`、`retryCount`、`payloadHash`，不得包含正文全文、raw token 或 credential。
- [ ] realtime fanout 是 best-effort，失败不回滚 inbox。
- [ ] consumer 名称和队列与 `deploy/docker/rabbitmq/definitions.json` 对齐。

- [ ] **步骤 1：写 consumer、idempotency、fanout 失败测试**
- [ ] **步骤 2：实现 application consumer 和 RabbitMQ adapter**
- [ ] **步骤 3：运行 consumer 验证**

运行：`cd services/zhicore-notification && go test ./internal/notification/... -run 'Consumer|Interaction|Fanout' -count=1`

预期：通过。

## 任务 5：偏好、DND、作者订阅与 delivery ledger

**测试立场：** TDD - 偏好、DND、delivery retry 和权限属于 R4。

**文件：**
- 新增：`services/zhicore-notification/api/http/endpoints/get-notification-preferences.md`
- 新增：`services/zhicore-notification/api/http/endpoints/update-notification-preferences.md`
- 新增：`services/zhicore-notification/api/http/endpoints/get-notification-dnd.md`
- 新增：`services/zhicore-notification/api/http/endpoints/update-notification-dnd.md`
- 新增：`services/zhicore-notification/api/http/endpoints/get-author-subscription.md`
- 新增：`services/zhicore-notification/api/http/endpoints/update-author-subscription.md`
- 新增：`services/zhicore-notification/api/http/endpoints/list-deliveries.md`
- 新增：`services/zhicore-notification/api/http/endpoints/retry-delivery.md`
- 新增：`services/zhicore-notification/migrations/20260706xxxx_add_notification_preference_and_delivery.up.sql`
- 新增：`services/zhicore-notification/migrations/20260706xxxx_add_notification_preference_and_delivery.down.sql`
- 新增：`services/zhicore-notification/internal/notification/domain/preference/*.go`
- 新增：`services/zhicore-notification/internal/notification/domain/delivery/*.go`
- 新增：`services/zhicore-notification/internal/notification/application/**/*.go`
- 新增：`services/zhicore-notification/internal/notification/infrastructure/postgres/*_store.go`

**验收清单：**
- [ ] 最终 path 明确为：`GET /api/v1/notification-preferences`、`PUT /api/v1/notification-preferences`、`GET /api/v1/notification-dnd`、`PUT /api/v1/notification-dnd`、`GET /api/v1/author-subscriptions/{authorId}`、`PUT /api/v1/author-subscriptions/{authorId}`、`GET /api/v1/notification-deliveries`、`POST /api/v1/notification-deliveries/{deliveryId}/retry`；如保留 `/api/v1/notifications/preferences` 等历史 alias，必须在 endpoint 文档中标注 alias 和 canonical path。
- [ ] `SMS` 第一阶段禁止启用。
- [ ] DND 校验 `startTime != endTime`，并写清跨日窗口、timezone、categories、channels 语义。
- [ ] 作者订阅固定 `ALL`、`DIGEST_ONLY`、`MUTED` 及 channel boolean 规范化。
- [ ] delivery retry 区分本人重试自己的记录和管理员重试任意记录。
- [ ] provider unconfigured 只影响 delivery 状态，不影响站内通知真相源。
- [ ] migration up/down 可通过 `golang-migrate` 往返验证；偏好 / delivery 表回滚不删除 inbox 核心表。

- [ ] **步骤 1：写 handler / app / repo / permission 失败测试**
- [ ] **步骤 2：实现偏好、DND、订阅和 delivery ledger**
- [ ] **步骤 3：运行验证**

运行：`cd services/zhicore-notification && go test ./internal/notification/... ./api/http -run 'Preference|Dnd|Subscription|Delivery' -count=1`

预期：通过。

## 任务 6：发文 campaign fanout 与 follower shard contract

**测试立场：** TDD - fanout shard、并发 claim、retry/backoff 属于 R4。

**文件：**
- 新增：`services/zhicore-notification/migrations/20260706xxxx_add_notification_campaign_tables.up.sql`
- 新增：`services/zhicore-notification/migrations/20260706xxxx_add_notification_campaign_tables.down.sql`
- 新增：`services/zhicore-notification/internal/notification/domain/campaign/*.go`
- 新增：`services/zhicore-notification/internal/notification/application/commands/plan_post_published_campaign.go`
- 新增：`services/zhicore-notification/internal/notification/application/jobs/execute_campaign_shard.go`
- 新增：`services/zhicore-notification/internal/notification/application/jobs/process_pending_digest.go`
- 新增：`services/zhicore-notification/internal/notification/infrastructure/postgres/campaign_store.go`
- 新增：`services/zhicore-notification/internal/notification/infrastructure/rabbitmq/post_published_consumer.go`
- 修改：`libs/contracts/clients/user/README.md`
- 修改：`libs/contracts/clients/user/contract.go`
- 修改：`libs/contracts/clients/user/contract_test.go`
- 修改：`services/zhicore-user/api/http/internal_handlers.go`

**验收清单：**
- [ ] `content.post.published` 只能创建 campaign 和初始 shard，不能同步逐粉丝写库。
- [ ] 执行前确认 Message 计划任务 1 已合并，避免并行覆盖 `libs/contracts/clients/user/contract.go`。
- [ ] 执行前确认任务 5 已合并；campaign fanout 必须读取偏好、DND、作者订阅和 delivery 规则，不允许绕过 active delivery 策略。
- [ ] shard claim 使用 `FOR UPDATE SKIP LOCKED` 或等价语义。
- [ ] claim timeout / batch size 从配置读并受上下界约束。
- [ ] `ListFollowerShard` degraded 时 retry / DLQ，不能把空结果当成功。
- [ ] delivery 状态至少区分 `IN_APP`、`WEBSOCKET_PENDING`、`DIGEST_PENDING`、`SKIPPED`。
- [ ] group-state rebuild 锁和单用户 rebuild 路径可验证。
- [ ] campaign worker 纳入 runtime start / stop、panic recovery、readiness descriptor 和配置校验。
- [ ] migration up/down 可通过 `golang-migrate` 往返验证；campaign 表回滚不删除 inbox / preference / delivery 表。
- [ ] race 验证覆盖并发 shard claim、重复 event、read-all 与 fanout 同时更新 unread。

- [ ] **步骤 1：写 follower shard contract 和 campaign 失败测试**
- [ ] **步骤 2：实现 User contract、campaign domain、repository 和 worker**
- [ ] **步骤 3：运行验证**

运行：`cd libs/contracts && go test ./clients/user -count=1 && cd ../../services/zhicore-notification && go test ./internal/notification/... -run 'Campaign|Shard|Follower' -count=1`

预期：通过。

## 集成验证

- [ ] 运行 `cd services/zhicore-notification && go test ./... -count=1`。
- [ ] 运行 `cd libs/contracts && go test ./clients/user -count=1`。
- [ ] 运行 `cd services/zhicore-notification && go test -race ./internal/notification/... -run 'Consumer|Campaign|Unread|ReadAll' -count=1`。
- [ ] 有可用 `ZHICORE_NOTIFICATION_POSTGRES_DSN` 时运行 `migrate -path services/zhicore-notification/migrations -database "$ZHICORE_NOTIFICATION_POSTGRES_DSN" up && migrate -path services/zhicore-notification/migrations -database "$ZHICORE_NOTIFICATION_POSTGRES_DSN" down 1 && migrate -path services/zhicore-notification/migrations -database "$ZHICORE_NOTIFICATION_POSTGRES_DSN" up`；没有 DSN 时必须用隔离 PostgreSQL 容器执行同等 `up -> down 1 -> up`，或在交付说明中列为未验证的外部依赖。
- [ ] 运行 `make test-size`。
- [ ] 运行 `bash scripts/check-structure.sh`。
- [ ] 完整执行本计划或触达共享 contract / runtime / migration / RabbitMQ 定义后，交付前运行 `make check`。
