# Ranking 事件账本与文章热榜基础实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；涉及 shared contract、migration contract、ledger/bucket/state、consumer、Redis materialization、handler、runtime 和 rebuild 的步骤按 @test-driven-development 执行。每个 checkbox 达到预期后立即更新；如需提交，提交前必须先使用 @committing-changes。

**目标：** 把 `zhicore-ranking` 从“已有 schema / HTTP contract 但无 Go 实现”推进到事件账本、bucket、文章总榜查询、热榜分数 / rank、flush worker、snapshot 和 rebuild foundation 可运行。

**架构：** Ranking 拥有热度 ledger、delta bucket、post state、period score、Redis 榜单和 rebuild operation；Content / Comment 事件是输入事实，Content 仍拥有文章详情和公开 ID。Redis 是物化层，不是权威源；所有公开榜单必须过滤 `ranking_post_state.public_visible=true`。

**技术栈：** Go 1.26、Gin、PostgreSQL、Redis ZSET、RabbitMQ、Content / Comment events、Content typed client、Ranking HTTP schema。

---

## 背景依据

- `docs/architecture/services/ranking/README.md`
- `docs/architecture/services/ranking/schema-and-implementation.md`
- `docs/architecture/services/ranking/application-and-ports.md`
- `docs/architecture/services/ranking/domain-model.md`
- `docs/architecture/services/ranking/data-events-projections.md`
- `docs/architecture/services/ranking/event-ordering-and-partitioning.md`
- `docs/architecture/services/ranking/query-materialization.md`
- `docs/architecture/services/ranking/runtime-resilience.md`
- `services/zhicore-ranking/api/http/README.md`
- `services/zhicore-ranking/api/http/endpoints/ranking-api.md`

## 当前基线

- `services/zhicore-ranking/api/http` 已有字段级 schema，但 Go handler / contract test 待实现。
- `services/zhicore-ranking/migrations/` 已有 core tables 和 rebuild operation migration，但无 migration contract test。
- 生产 Go 源码只有 `services/zhicore-ranking/internal/ranking/doc.go`。
- Content events Go 类型只覆盖部分 payload，Ranking hot foundation 需要补齐。

## 不可并行修改文件

- `libs/contracts/clients/content/contract.go`：由本计划任务 1 先固定 Ranking hot foundation 所需 `publicId -> internalId` 和批量摘要；Search / Admin 计划必须等待本任务合并后再追加各自字段。
- `libs/contracts/events/content/contract.go`：由本计划任务 1 先补 Ranking hot foundation 事件；Search 计划必须等待本任务合并后再追加索引事件。

## 任务 1：共享 contract 与 migration 验证

**测试立场：** TDD - migration、Content event contract 和 Content typed client 属于 R4。

**文件：**
- 新增：`services/zhicore-ranking/migrations/migration_contract_test.go`
- 修改：`libs/contracts/events/content/contract.go`
- 修改：`libs/contracts/events/content/contract_test.go`
- 修改：`libs/contracts/events/comment/contract.go`
- 修改：`libs/contracts/events/comment/contract_test.go`
- 新增或修改：`libs/contracts/clients/content/contract.go`
- 新增或修改：`libs/contracts/clients/content/contract_test.go`

**验收清单：**
- [ ] migration test 覆盖 `ranking_event_ledger`、`ranking_delta_bucket`、`ranking_post_state`、`ranking_projection_event_inbox`、`ranking_period_score`、`ranking_rebuild_operation`。
- [ ] migration test 覆盖关键 index、`delta <> 0`、`public_visible` 默认值、`applied_*`、status check。
- [ ] migration up/down 可通过 `golang-migrate` 往返验证；新增 migration 文件名必须使用真实单调递增时间戳，不保留 `20260706xxxx` 占位。
- [ ] Content event Go 类型覆盖 hot foundation 需要的 `published`、`deleted`、`visibility_changed`、`liked`、`unliked`。
- [ ] Content ranking event payload 字段矩阵固定：所有事件必须有 `eventId`、`eventType`、`publicId`、`internalId`、`occurredAt`；`liked` / `unliked` 等排名摄入事件必须有 `delta`；`published` 必须有 `publishedAt`；`visibility_changed` 必须有 `publicVisible`、`oldVisibility`、`newVisibility`、`reason`、`aggregateVersion` 或可排序发生时间。
- [ ] Comment event Go 类型能提供 `comment.created` 和 `comment.deleted.affectedCount`。
- [ ] Comment ranking event payload 字段矩阵固定：`comment.created` 必须有 `eventId`、`eventType`、`postPublicId`、`postInternalId`、`commentId`、`occurredAt`、`delta=+1`；`comment.deleted` 必须有 `affectedCount`、`occurredAt` 和 `delta=-affectedCount`。
- [ ] Content typed client 支撑 `publicId -> internalId` 解析和批量文章摘要查询。
- [ ] 本任务只固定 Ranking hot foundation 需要的 Content client / event 字段；不要混入 Search 索引回源字段、Admin 管理端字段或其他服务私有需求。

- [ ] **步骤 1：写 migration 和 shared contract 失败测试**
- [ ] **步骤 2：实现缺失 contract 类型和 migration assertions**
- [ ] **步骤 3：运行验证**

运行：`cd services/zhicore-ranking && go test ./migrations -count=1 && cd ../../libs/contracts && go test ./events/content ./events/comment ./clients/content -count=1`

预期：通过。

## 任务 2：ledger / bucket / state / projection 核心链路

**测试立场：** TDD - 幂等、迟到事件、pending delta、可见性投影属于 R4。

**文件：**
- 新增：`services/zhicore-ranking/internal/ranking/domain/metric_type.go`
- 新增：`services/zhicore-ranking/internal/ranking/domain/ledger.go`
- 新增：`services/zhicore-ranking/internal/ranking/domain/delta_bucket.go`
- 新增：`services/zhicore-ranking/internal/ranking/domain/post_state.go`
- 新增：`services/zhicore-ranking/internal/ranking/domain/hot_score_calculator.go`
- 新增：`services/zhicore-ranking/internal/ranking/ports/*.go`
- 新增：`services/zhicore-ranking/internal/ranking/application/ingest_ranking_event.go`
- 新增：`services/zhicore-ranking/internal/ranking/application/apply_content_visibility_event.go`
- 新增：`services/zhicore-ranking/internal/ranking/application/flush_ranking_buckets.go`
- 新增：`services/zhicore-ranking/internal/ranking/infrastructure/postgres/*.go`

**验收清单：**
- [ ] `event_id` 重复 no-op。
- [ ] `comment.deleted` 按 `affectedCount` 扣减。
- [ ] 已 flushed bucket 收到迟到事件后，下轮只应用 pending delta。
- [ ] `public_visible=false` 只改 projection，不写热度 ledger。
- [ ] `visibility_changed` 的旧 `aggregateVersion` 或旧时间戳被忽略。
- [ ] 计数不能被负增量扣到负数；错误分类可观测。
- [ ] application 注释说明 `applied_*` 是避免重复物化的幂等关键字段。

- [ ] **步骤 1：写 duplicate、late event、negative delta、stale visibility 失败测试**
- [ ] **步骤 2：实现 domain、ports、repository 和 application**
- [ ] **步骤 3：运行核心链路验证**

运行：`cd services/zhicore-ranking && go test ./internal/ranking/domain ./internal/ranking/application ./internal/ranking/infrastructure/postgres -count=1`

预期：通过。

## 任务 3：RabbitMQ consumer 与写路径闭合

**测试立场：** TDD - ack/nack/DLQ、事务提交后 ack、解码错误属于 R4。

**文件：**
- 新增：`services/zhicore-ranking/internal/ranking/infrastructure/rabbitmq/content_post_consumer.go`
- 新增：`services/zhicore-ranking/internal/ranking/infrastructure/rabbitmq/comment_consumer.go`
- 新增：`services/zhicore-ranking/internal/ranking/infrastructure/rabbitmq/event_decoder.go`
- 新增：`services/zhicore-ranking/internal/ranking/infrastructure/rabbitmq/*_test.go`

**验收清单：**
- [ ] 缺 `internalId` 进入 DLQ，不在消费路径同步解析 public ID。
- [ ] consumer 只有 PostgreSQL 事务提交后才 ack。
- [ ] 摄入事务内不写 Redis。
- [ ] `content.post.deleted`、`visibility_changed` 走 projection inbox。
- [ ] `liked`、`unliked`、`comment.created`、`comment.deleted` 走 ledger + bucket。
- [ ] bad payload、duplicate、retry 行为有测试覆盖。

- [ ] **步骤 1：写 consumer 失败测试**
- [ ] **步骤 2：实现 decoder 和 RabbitMQ adapter**
- [ ] **步骤 3：运行 consumer 验证**

运行：`cd services/zhicore-ranking && go test ./internal/ranking/infrastructure/rabbitmq -count=1`

预期：通过。

## 任务 4：公开热榜 4 个 endpoint

**测试立场：** TDD - handler contract、Redis miss fallback、公开 ID 解析属于 R4。

**文件：**
- 新增：`services/zhicore-ranking/api/http/handler.go`
- 新增：`services/zhicore-ranking/api/http/payloads.go`
- 新增：`services/zhicore-ranking/api/http/errors.go`
- 新增：`services/zhicore-ranking/api/http/hot_posts_handler_test.go`
- 新增：`services/zhicore-ranking/api/http/post_rank_handler_test.go`
- 新增：`services/zhicore-ranking/internal/ranking/application/list_hot_posts.go`
- 新增：`services/zhicore-ranking/internal/ranking/application/list_hot_posts_with_score.go`
- 新增：`services/zhicore-ranking/internal/ranking/application/get_post_rank.go`
- 新增：`services/zhicore-ranking/internal/ranking/application/get_post_score.go`
- 新增：`services/zhicore-ranking/internal/ranking/ports/query_store.go`
- 新增：`services/zhicore-ranking/internal/ranking/ports/redis_materializer.go`
- 新增：`services/zhicore-ranking/internal/ranking/ports/content_post_client.go`
- 新增：`services/zhicore-ranking/internal/ranking/infrastructure/redis/query_store.go`
- 新增：`services/zhicore-ranking/internal/ranking/infrastructure/clients/content_client.go`
- 修改：`services/zhicore-ranking/api/http/README.md`
- 修改：`services/zhicore-ranking/api/http/endpoints/ranking-api.md`

**验收清单：**
- [ ] `GET /api/v1/ranking/posts/hot`、`/hot/scores`、`/{postId}/rank`、`/{postId}/score` 都有 contract test。
- [ ] handler contract test 覆盖 ZhiCore success / error envelope、默认分页 `page=0`、`size=20`、最大 `size=100`、非法分页参数、非法 period / postId、Redis miss fallback、Content client degraded 和公开可见性过滤。
- [ ] 入站 `{postId}` 只接受 Content `public_id`。
- [ ] Redis miss 回源 PostgreSQL 时返回 `degraded=true`。
- [ ] `public_visible=false` 永不出现在榜单。
- [ ] 排序稳定，不暴露内部 `post_id`。
- [ ] 未上榜但实体存在返回 `200 ranked=false`；实体不存在或不可公开返回 schema 登记的 `1005`。

- [ ] **步骤 1：写 4 个 handler contract 失败测试**
- [ ] **步骤 2：实现 query use case、Redis query store、Content client 和 handler**
- [ ] **步骤 3：运行 endpoint 验证**

运行：`cd services/zhicore-ranking && go test ./api/http ./internal/ranking/application ./internal/ranking/infrastructure/redis ./internal/ranking/infrastructure/clients -count=1`

预期：通过。

## 任务 5：runtime、flush worker 和 snapshot

**测试立场：** TDD - worker lifecycle、Redis failure、runtime config 属于 R4。

**文件：**
- 新增：`services/zhicore-ranking/internal/ranking/runtime/module.go`
- 新增：`services/zhicore-ranking/internal/ranking/runtime/module_test.go`
- 新增：`services/zhicore-ranking/internal/ranking/application/refresh_ranking_snapshots.go`
- 新增：`services/zhicore-ranking/internal/ranking/application/refresh_ranking_snapshots_test.go`
- 新增：`services/zhicore-ranking/cmd/server/main.go`
- 新增：`services/zhicore-ranking/cmd/server/config.go`
- 新增：`services/zhicore-ranking/cmd/server/config_defaults.go`
- 新增：`services/zhicore-ranking/cmd/server/config_loader.go`
- 新增：`services/zhicore-ranking/cmd/server/config_test.go`
- 新增：`services/zhicore-ranking/cmd/server/server.go`
- 新增：`services/zhicore-ranking/configs/local.example.env`

**验收清单：**
- [ ] 配置覆盖 bucket window、flush interval、flush delay、snapshot interval、Redis / RabbitMQ / Postgres timeout。
- [ ] `/health/live` 只查进程。
- [ ] `/health/ready` 默认只硬依赖 PostgreSQL；Redis / RabbitMQ 默认进入 degraded details，只有 worker-only / consumer-only 进程或配置 `hardDependency=true` 时才作为 ready 硬依赖。
- [ ] flush 成功后 Redis materialize 失败不回滚 PostgreSQL。
- [ ] Redis snapshot 原子替换或明确临时 key + rename 语义。
- [ ] 启动路径不执行 migration。

- [ ] **步骤 1：写 config、health、flush/materialize failure 失败测试**
- [ ] **步骤 2：实现 runtime、worker 和 snapshot**
- [ ] **步骤 3：运行 runtime 验证**

运行：`cd services/zhicore-ranking && go test ./cmd/server ./internal/ranking/runtime ./internal/ranking/application -run 'Config|Health|Flush|Snapshot' -count=1`

预期：通过。

## 任务 6：admin rebuild、operation status 和后续榜单扩展闸门

**测试立场：** TDD - rebuild lock、barrier、operation 状态属于 R4。

**文件：**
- 新增：`services/zhicore-ranking/internal/ranking/application/rebuild_from_ledger.go`
- 新增：`services/zhicore-ranking/internal/ranking/application/rebuild_from_ledger_test.go`
- 新增：`services/zhicore-ranking/internal/ranking/ports/lock_manager.go`
- 新增：`services/zhicore-ranking/internal/ranking/ports/replay_repository.go`
- 新增：`services/zhicore-ranking/internal/ranking/ports/rebuild_operation_repository.go`
- 新增：`services/zhicore-ranking/internal/ranking/infrastructure/postgres/rebuild_operation_repository.go`
- 新增：`services/zhicore-ranking/internal/ranking/infrastructure/redis/lock_manager.go`
- 新增：`services/zhicore-ranking/api/http/admin_rebuild_handler_test.go`
- 修改：`services/zhicore-ranking/api/http/README.md`
- 修改：`services/zhicore-ranking/api/http/endpoints/ranking-api.md`

**验收清单：**
- [ ] `POST /api/v1/ranking/admin/rebuild-from-ledger` 返回 `ACCEPTED + operationId`。
- [ ] `GET /api/v1/ranking/admin/rebuild-operations/{operationId}` 有 handler contract test，覆盖管理员鉴权、not found、ZhiCore envelope、`operationId`、`status`、`startedAt`、`finishedAt`、`processedEvents`、`errorClass`、`errorMessage`。
- [ ] 无 rebuild lock 时拒绝启动。
- [ ] barrier 期间 live ingestion 不写业务表；consumer nack / requeue 或暂停消费。
- [ ] PostgreSQL rebuild 成功但 Redis refresh 失败时记录 `PARTIAL_FAILED`。
- [ ] status 查询支持 `ACCEPTED`、`RUNNING`、`SUCCEEDED`、`PARTIAL_FAILED`、`FAILED`、`CANCELED`。
- [ ] 本计划完成后，`daily/weekly/monthly`、creator/topic、hot candidates、archive 仍作为后续计划，不混入本 foundation。
- [ ] rebuild / live ingestion / snapshot race 使用 `go test -race` 覆盖，至少包含 barrier 期间 consumer requeue、operation status 更新和 Redis refresh 并发。

- [ ] **步骤 1：写 rebuild 和 status 失败测试**
- [ ] **步骤 2：实现 lock、barrier、operation repository 和 handler**
- [ ] **步骤 3：运行验证**

运行：`cd services/zhicore-ranking && go test ./internal/ranking/application ./api/http -run 'Rebuild|Operation' -count=1`

预期：通过。

## 集成验证

- [ ] 运行 `cd libs/contracts && go test ./events/content ./events/comment ./clients/content -count=1`。
- [ ] 运行 `cd services/zhicore-ranking && go test ./... -count=1`。
- [ ] 运行 `cd services/zhicore-ranking && go test -race ./internal/ranking/... -run 'Consumer|Flush|Snapshot|Rebuild' -count=1`。
- [ ] 有可用 `ZHICORE_RANKING_POSTGRES_DSN` 时运行 `migrate -path services/zhicore-ranking/migrations -database "$ZHICORE_RANKING_POSTGRES_DSN" up && migrate -path services/zhicore-ranking/migrations -database "$ZHICORE_RANKING_POSTGRES_DSN" down 1 && migrate -path services/zhicore-ranking/migrations -database "$ZHICORE_RANKING_POSTGRES_DSN" up`；没有 DSN 时必须用隔离 PostgreSQL 容器执行同等 `up -> down 1 -> up`，或在交付说明中列为未验证的外部依赖。
- [ ] 运行 `make test-size`。
- [ ] 运行 `bash scripts/check-structure.sh`。
- [ ] 完整执行本计划或触达共享 contract、migration、worker 或 runtime 后，交付前运行 `make check`。
