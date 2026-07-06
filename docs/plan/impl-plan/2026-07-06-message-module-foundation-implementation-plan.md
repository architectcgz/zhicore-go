# Message 模块基础实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；涉及 HTTP contract、User typed client、私信发送、会话投影、未读数、runtime、召回和 outbox 的步骤按 @test-driven-development 执行。每个 checkbox 达到预期后立即更新；如需提交，提交前必须先使用 @committing-changes。

**目标：** 把 `zhicore-message` 从占位模块推进到能发送私信、维护会话摘要和未读投影、提供历史空列表兼容、可运行 runtime 和召回 / outbox 基础的首个交付切片。

**架构：** Message 拥有私信会话、私信消息、已读状态、撤回状态和消息派发 outbox；User 拥有拉黑、关注和陌生人消息设置；Notification 未读数不能并入 Message。外部 IM provider 通过 Message 本地 port / adapter 封装，provider 未接入时历史查询按 contract 返回空列表。

**技术栈：** Go 1.26、Gin、PostgreSQL、User typed client、IM provider adapter、RabbitMQ outbox、Message HTTP schema。

---

## 背景依据

- `docs/architecture/services/message/README.md`
- `services/zhicore-message/api/http/README.md`
- `docs/architecture/service-boundaries.md`
- `docs/contracts/http-schema-template.md`
- `docs/architecture/runtime-operations.md`
- `docs/architecture/testing.md`
- `docs/migration/service-migration-workflow.md`
- `docs/reviews/quality-gates.md`
- 需要核对既有 alias 和 DTO 时读取 `../zhicore-microservice/zhicore-message/src/main/java/com/zhicore/message/interfaces/controller/*.java`

## 当前基线

- 生产 Go 源码只有 `services/zhicore-message/internal/message/doc.go`。
- HTTP schema 是计划化占位，没有 `endpoints/`。
- User contract 目前缺 `CheckFollowing`、`GetStrangerMessageSetting` 和 Message 专用 caller operation。
- 召回真相源和外部 IM provider 契约未固定。

## 不可并行修改文件

- `libs/contracts/clients/user/contract.go`：由本计划任务 1 先固定 Message guard contract；Notification 计划任务 6 必须等待本任务合并后再追加 follower shard contract。

## 任务 1：固定 Message HTTP contract 与 User typed client 前置契约

**测试立场：** HTTP 文档 R0；User typed client 和 provider internal route 属于 R4，采用 TDD。

**文件：**
- 修改：`services/zhicore-message/README.md`
- 修改：`services/zhicore-message/api/http/README.md`
- 新增：`services/zhicore-message/api/http/endpoints/send-message.md`
- 新增：`services/zhicore-message/api/http/endpoints/recall-message.md`
- 新增：`services/zhicore-message/api/http/endpoints/list-conversations.md`
- 新增：`services/zhicore-message/api/http/endpoints/get-conversation.md`
- 新增：`services/zhicore-message/api/http/endpoints/get-conversation-by-user.md`
- 新增：`services/zhicore-message/api/http/endpoints/get-message-history.md`
- 新增：`services/zhicore-message/api/http/endpoints/mark-conversation-read.md`
- 新增：`services/zhicore-message/api/http/endpoints/get-message-unread-count.md`
- 修改：`libs/contracts/clients/user/README.md`
- 修改：`libs/contracts/clients/user/contract.go`
- 修改：`libs/contracts/clients/user/contract_test.go`
- 修改：`services/zhicore-user/api/http/handler.go`
- 修改：`services/zhicore-user/api/http/internal_handlers.go`
- 新增：`services/zhicore-user/api/http/internal_message_contract_test.go`
- 新增：`services/zhicore-message/internal/message/ports/im_provider.go`
- 新增：`services/zhicore-message/internal/message/ports/im_provider_test.go`

**验收清单：**
- [ ] 先固定召回真相源：若 Message 本地 `messages` 是公开 API 真相源，则本地 `recalled_at` / `recall_status` 决定查询结果，IM provider recall 只作为补偿；若 provider 是真相源，则发送 schema 必须保存 provider `externalMessageId` 并在不支持 recall 时返回明确业务错误。未决前不得实现发送链路。
- [ ] IM provider contract 固定 `SendMessage` request / response、`externalMessageId`、`providerConversationId`、业务 `idempotencyKey`、timeout、retry policy、错误分类和 provider success 后本地事务失败的补偿语义。
- [ ] provider 错误分类至少区分 `TRANSIENT`、`PERMANENT_VALIDATION`、`UNSUPPORTED`、`RATE_LIMITED`、`UNKNOWN`，并映射公开错误码或 outbox retry。
- [ ] 写实历史 alias：`POST /api/v1/messages` 与 `/send`；`GET /api/v1/messages/conversations/{conversationId}/messages` 与 `/conversation/{conversationId}`；`POST /conversations/{conversationId}/read` 与 `/conversation/{conversationId}/read`；`GET /unread/count` 与 `/unread-count`；`/api/v1/messages/conversations` 与 `/api/v1/conversations`。
- [ ] 发送请求字段固定为 `receiverId`、`type`、`content`、`mediaUrl`。
- [ ] `TEXT` 必须有 `content`；`IMAGE` / `FILE` 必须有 `mediaUrl`。
- [ ] MessageVO / ConversationVO 字段形态写入 endpoint 文档。
- [ ] User internal contract 补 `message.check_blocked`、`message.check_following`、`message.get_stranger_message_setting` 的 path、DTO、caller operation 和 degraded 语义。
- [ ] 本任务只固定 Message guard 所需 User contract；不要混入 Notification follower shard contract。
- [ ] `SERVICE_DEGRADED` 不能被 Message 当成“未拉黑 / 允许私信”。
- [ ] 频控和内容安全过滤首期明确为 out-of-scope，endpoint 文档必须声明当前只做字段校验、关系 guard 和 provider 错误映射；若实施时加入频控 / 过滤，必须补独立错误码和测试。

- [ ] **步骤 1：核对 Java controller / DTO 并补 HTTP schema**
- [ ] **步骤 2：先写 User contract / handler 失败测试**
- [ ] **步骤 3：实现 User typed client contract、IM provider port 和 internal handler**
- [ ] **步骤 4：运行验证**

运行：`cd libs/contracts && go test ./clients/user -count=1 && cd ../../services/zhicore-user && go test ./api/http -run TestInternalMessage -count=1`

预期：通过。

## 任务 2：发送私信与会话投影最小闭环

**测试立场：** TDD - 新 endpoint、use case、repository、权限 guard 属于 R4。

**文件：**
- 新增：`services/zhicore-message/migrations/20260706xxxx_create_message_core.up.sql`
- 新增：`services/zhicore-message/migrations/20260706xxxx_create_message_core.down.sql`
- 新增：`services/zhicore-message/migrations/migration_contract_test.go`
- 新增：`services/zhicore-message/internal/message/domain/message.go`
- 新增：`services/zhicore-message/internal/message/domain/conversation.go`
- 新增：`services/zhicore-message/internal/message/domain/errors.go`
- 新增：`services/zhicore-message/internal/message/ports/repositories.go`
- 新增：`services/zhicore-message/internal/message/ports/user_client.go`
- 新增：`services/zhicore-message/internal/message/ports/transaction.go`
- 新增：`services/zhicore-message/internal/message/application/*.go`
- 新增：`services/zhicore-message/internal/message/infrastructure/postgres/*.go`
- 新增：`services/zhicore-message/internal/message/infrastructure/clients/user_client.go`
- 新增：`services/zhicore-message/api/http/*.go`

**验收清单：**
- [ ] migration 创建 `conversations`、`messages`、`message_outbox_task` 和必要唯一索引。
- [ ] migration up/down 可通过 `golang-migrate` 往返验证；文件名中的 `20260706xxxx` 必须在实施时替换为真实单调递增时间戳。
- [ ] 发送链路顺序固定：User guard -> IM provider -> 本地会话摘要 / 未读投影 -> outbox task。
- [ ] IM provider 成功但本地事务失败时，必须写入可重试补偿任务或调用 provider cancel / recall；测试覆盖补偿任务创建失败时返回 `1004` 且不伪成功。
- [ ] 文本缺 `content`、图片 / 文件缺 `mediaUrl`、被拉黑、陌生人消息关闭、User degraded、IM provider 失败都有公开错误映射。
- [ ] 成功发送不得改 Notification 未读数。
- [ ] Message 不复制 User 关系表，只保存必要用户 ID 引用和会话快照。
- [ ] 关键 application 注释说明 User guard 与 IM provider 调用顺序是业务规则，不能绕过。

- [ ] **步骤 1：写 migration、application、handler、repository 失败测试**
- [ ] **步骤 2：实现 domain、ports、repository、application 和 handler**
- [ ] **步骤 3：运行发送闭环测试**

运行：`cd services/zhicore-message && go test ./migrations ./internal/message/... ./api/http -count=1`

预期：通过。

## 任务 3：历史消息、会话已读和未读数

**测试立场：** TDD - provider 降级、幂等已读和未读投影属于 R4。

**文件：**
- 新增：`services/zhicore-message/internal/message/application/get_message_history.go`
- 新增：`services/zhicore-message/internal/message/application/mark_conversation_read.go`
- 新增：`services/zhicore-message/internal/message/application/get_unread_count.go`
- 新增：`services/zhicore-message/internal/message/infrastructure/providers/noop_im_provider.go`
- 新增：`services/zhicore-message/api/http/message_query_handlers.go`
- 新增：`services/zhicore-message/api/http/message_history_handler_test.go`
- 新增：`services/zhicore-message/api/http/mark_read_handler_test.go`
- 新增：`services/zhicore-message/api/http/unread_count_handler_test.go`

**验收清单：**
- [ ] 历史接口在 provider 未接入时返回 `200 + data: []`，不伪造消息，不返回 5xx。
- [ ] 未读数只来自 Message 本地会话投影。
- [ ] 标记已读幂等，重复调用不能重复扣减。
- [ ] 如果 provider 支持 read receipt，adapter 同步失败不能污染本地 unread 投影；补偿边界写入 endpoint 文档。
- [ ] `conversationId` 不属于当前 actor 时返回权限错误，不泄露会话存在性。

- [ ] **步骤 1：写 handler / application 失败测试**
- [ ] **步骤 2：实现 noop provider、query 和 mark read**
- [ ] **步骤 3：运行验证**

运行：`cd services/zhicore-message && go test ./internal/message/application ./api/http -count=1`

预期：通过。

## 任务 4：可运行 runtime、配置和健康检查

**测试立场：** TDD - 配置、脱敏、readiness 和 server lifecycle 属于 R4。

**文件：**
- 新增：`services/zhicore-message/cmd/server/config.go`
- 新增：`services/zhicore-message/cmd/server/config_loader.go`
- 新增：`services/zhicore-message/cmd/server/config_defaults.go`
- 新增：`services/zhicore-message/cmd/server/config_validation.go`
- 新增：`services/zhicore-message/cmd/server/server.go`
- 新增：`services/zhicore-message/cmd/server/main.go`
- 新增：`services/zhicore-message/cmd/server/*_test.go`
- 新增：`services/zhicore-message/internal/message/runtime/module.go`
- 新增：`services/zhicore-message/internal/message/runtime/health_test.go`
- 新增：`services/zhicore-message/configs/local.example.env`

**验收清单：**
- [ ] 必填 env 至少覆盖 Postgres DSN、User base URL、IM provider base URL / credential、HTTP addr / timeout。
- [ ] secret、DSN、provider credential 在错误、日志、`String()`、`GoString()` 中脱敏。
- [ ] `/health/live` 不探活下游。
- [ ] `/health/ready` 检查必需依赖和 enabled worker descriptor。
- [ ] 启动路径不自动执行 migration。
- [ ] `cmd/server` 只做配置、依赖打开、runtime 装配、server lifecycle，不写业务逻辑。
- [ ] 后续 outbox / recall worker 增加时必须同步更新 runtime worker descriptor、启动 / 停止顺序、panic recovery、readiness degraded details 和配置校验。

- [ ] **步骤 1：写 config / health / lifecycle 失败测试**
- [ ] **步骤 2：实现 runtime 和 server**
- [ ] **步骤 3：运行 runtime 验证**

运行：`cd services/zhicore-message && go test ./cmd/server ./internal/message/runtime -count=1`

预期：通过。

## 任务 5：召回与 outbox / 补偿

**测试立场：** TDD - 召回权限、时间窗、outbox 重试属于 R4。

**文件：**
- 新增：`services/zhicore-message/internal/message/application/recall_message.go`
- 新增：`services/zhicore-message/internal/message/application/outbox_worker.go`
- 新增：`services/zhicore-message/internal/message/ports/outbox.go`
- 新增：`services/zhicore-message/internal/message/ports/im_recall.go`
- 新增：`services/zhicore-message/internal/message/infrastructure/postgres/outbox_repository.go`
- 新增：`services/zhicore-message/internal/message/infrastructure/rabbitmq/outbox_dispatcher.go`
- 新增：`services/zhicore-message/api/http/recall_message_handler_test.go`
- 可选新增：`services/zhicore-message/migrations/*_alter_message_outbox_task_status.up.sql`
- 可选新增：`services/zhicore-message/migrations/*_alter_message_outbox_task_status.down.sql`
- 修改：`services/zhicore-message/internal/message/runtime/module.go`
- 修改：`services/zhicore-message/internal/message/runtime/health_test.go`
- 修改：`services/zhicore-message/cmd/server/config.go`
- 修改：`services/zhicore-message/cmd/server/config_validation.go`

**验收清单：**
- [ ] 召回真相源已在任务 1 固定；本任务不得重新改语义，只能实现对应 local-first 或 provider-first 流程。
- [ ] 只有发送者可召回。
- [ ] 召回窗口如沿用 Java 则为 2 分钟；如改变，必须先登记 API 变更原因。
- [ ] IM provider 不支持 recall 时返回明确业务错误，不得伪成功。
- [ ] outbox worker 按任务状态和幂等键重试，send / read / recall 补偿都有可观察状态。
- [ ] outbox worker 纳入 runtime start / stop；panic 后记录错误并触发 readiness degraded，不允许静默退出。
- [ ] migration up/down 可通过 `golang-migrate` 往返验证；可选 migration 一旦创建，必须同时创建 down 文件。
- [ ] worker / cache / unread 投影 race 使用 `go test -race` 覆盖，至少包含并发 mark read、发送后未读增加和 outbox retry。

- [ ] **步骤 1：写 recall / outbox 失败测试**
- [ ] **步骤 2：实现 recall command、provider adapter 和 outbox worker**
- [ ] **步骤 3：运行验证**

运行：`cd services/zhicore-message && go test ./internal/message/... ./api/http -run 'Recall|Outbox' -count=1`

预期：通过。

## 集成验证

- [ ] 运行 `cd libs/contracts && go test ./clients/user -count=1`。
- [ ] 运行 `cd services/zhicore-user && go test ./api/http -run TestInternalMessage -count=1`。
- [ ] 运行 `cd services/zhicore-message && go test ./... -count=1`。
- [ ] 运行 `cd services/zhicore-message && go test -race ./internal/message/... -run 'Unread|Outbox|MarkRead|Send' -count=1`。
- [ ] 有可用 `ZHICORE_MESSAGE_POSTGRES_DSN` 时运行 `migrate -path services/zhicore-message/migrations -database "$ZHICORE_MESSAGE_POSTGRES_DSN" up && migrate -path services/zhicore-message/migrations -database "$ZHICORE_MESSAGE_POSTGRES_DSN" down 1 && migrate -path services/zhicore-message/migrations -database "$ZHICORE_MESSAGE_POSTGRES_DSN" up`；没有 DSN 时必须用隔离 PostgreSQL 容器执行同等 `up -> down 1 -> up`，或在交付说明中列为未验证的外部依赖。
- [ ] 运行 `make test-size`。
- [ ] 运行 `bash scripts/check-structure.sh`。
- [ ] 完整执行本计划或触达共享 contract / runtime / migration 后，交付前运行 `make check`。
