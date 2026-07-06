# Search 文章索引基础实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；涉及 HTTP contract、Content typed client、PostgreSQL FTS / trigram 读模型、事件消费、repository、runtime 和 handler 的步骤按 @test-driven-development 执行。每个 checkbox 达到预期后立即更新；如需提交，提交前必须先使用 @committing-changes。

**目标：** 把 `zhicore-search` 从占位模块推进到拥有文章搜索 contract、PostgreSQL 搜索读模型、Content 事件索引链路、最小查询面、suggest / hot / history 和可运行 runtime 的 foundation。

**架构：** Search 只拥有搜索索引、搜索建议、热门搜索词、搜索历史和相关读模型；文章详情和可见性权威仍归 Content。首期不引入外部搜索引擎，查询主路径不逐条回源 Content 做可见性判断，Search 通过 Content 事件维护 PostgreSQL 派生读模型，删除事件必须硬删除本地索引行。

**技术栈：** Go 1.26、Gin、PostgreSQL FTS / `pg_trgm`、Redis、RabbitMQ Content events、Content typed client、Search HTTP schema。

---

## 背景依据

- `docs/architecture/services/search/README.md`
- `services/zhicore-search/api/http/README.md`
- `docs/architecture/service-boundaries.md`
- `docs/contracts/http-schema-template.md`
- `docs/contracts/pagination.md`
- `docs/architecture/runtime-operations.md`
- 需要固定旧字段时读取 `../zhicore-microservice/zhicore-search` controller / DTO。
- PostgreSQL 搜索语义首期使用 `tsvector`、GIN index、`pg_trgm` 或等价显式索引；如后续引入外部搜索引擎，另开替换计划。

## 当前基线

- 生产 Go 源码只有 `services/zhicore-search/internal/search/doc.go`。
- HTTP schema 是计划化占位，没有 `endpoints/`。
- `libs/contracts/clients/content` 只有 README，没有 Search 需要的 Go typed client。
- Content 事件 Go 类型未完全覆盖 Search 需要的 `updated`、`deleted`、`tags.updated`。

## 不可并行修改文件

- `libs/contracts/clients/content/contract.go`：必须等 `2026-07-06-ranking-ledger-hot-posts-foundation-implementation-plan.md` 任务 1 合并后，再追加 Search 索引回源字段。
- `libs/contracts/events/content/contract.go`：必须等 Ranking 计划任务 1 合并后，再追加 Search 需要的 `updated` / `tags.updated` 事件类型或字段。

## 任务 1：Search HTTP contract 和 Content typed client 固化

**测试立场：** HTTP 文档 R0；Content typed client 属于 R4，采用 TDD。

**文件：**
- 修改：`services/zhicore-search/api/http/README.md`
- 新增：`services/zhicore-search/api/http/endpoints/search-posts.md`
- 新增：`services/zhicore-search/api/http/endpoints/search-suggest.md`
- 新增：`services/zhicore-search/api/http/endpoints/search-hot.md`
- 新增：`services/zhicore-search/api/http/endpoints/search-history.md`
- 新增：`services/zhicore-search/api/http/endpoints/clear-search-history.md`
- 修改：`docs/architecture/services/search/README.md`
- 修改：`libs/contracts/clients/content/README.md`
- 修改 / 追加：`libs/contracts/clients/content/contract.go`
- 修改 / 追加：`libs/contracts/clients/content/contract_test.go`

**验收清单：**
- [ ] 5 个 operation 各自有 endpoint 文档：`search-posts`、`search-suggest`、`search-hot`、`search-history`、`clear-search-history`。
- [ ] `search-history.md` 只记录 `GET /api/v1/search/history`；`clear-search-history.md` 只记录 `DELETE /api/v1/search/history`。
- [ ] `/api/v1/search/posts` 写清可见性滞后 SLA、降级语义、Content 补详情边界。
- [ ] Search 结果可包含索引预览字段，但不能把它当 Content 权威详情。
- [ ] Content client 至少固定 `BatchGetPostSummaries`、`ResolvePublicID`、`GetPublishedBodyForIndex` 或等价查询。
- [ ] 执行前确认 Ranking 计划任务 1 已合并；未合并前本任务只能补 Search HTTP schema，不得编辑 `libs/contracts/clients/content/contract.go` 或 `libs/contracts/events/content/contract.go`。
- [ ] 搜索历史的登录态、保留时间和清理语义写入 endpoint 文档。

- [ ] **步骤 1：核对 Java controller / DTO 与前端页面字段**
- [ ] **步骤 2：补 Search endpoint schema**
- [ ] **步骤 3：写 Content typed client contract 失败测试**
- [ ] **步骤 4：实现 Content typed client contract**
- [ ] **步骤 5：运行验证**

运行：`cd libs/contracts && go test ./clients/content -count=1 && cd ../.. && bash scripts/check-structure.sh`

预期：通过。

## 任务 2：PostgreSQL 搜索读模型与本地存储基础

**测试立场：** TDD - migration、FTS / trigram 索引和 repository 属于 R4。

**文件：**
- 新增：`services/zhicore-search/migrations/20260706xxxx_create_search_core_tables.up.sql`
- 新增：`services/zhicore-search/migrations/20260706xxxx_create_search_core_tables.down.sql`
- 新增：`services/zhicore-search/migrations/migration_contract_test.go`
- 新增：`services/zhicore-search/internal/search/domain/post_document.go`
- 新增：`services/zhicore-search/internal/search/ports/search_index_repository.go`
- 新增：`services/zhicore-search/internal/search/ports/history_repository.go`
- 新增：`services/zhicore-search/internal/search/ports/hot_terms_repository.go`
- 新增：`services/zhicore-search/internal/search/infrastructure/postgres/search_index_repository.go`
- 新增：`services/zhicore-search/internal/search/infrastructure/postgres/search_index_repository_test.go`

**验收清单：**
- [ ] migration 只创建 Search 自有 `search_post_documents`、`search_event_inbox`、history、hot-term 和 rebuild 状态表。
- [ ] `search_post_documents` 明确 `public_id`、`internal_id`、`title`、`summary`、`plain_text`、`author_id`、`topic_ids`、`published_at`、`public_visible`、`content_status`、`published_body_hash`、`aggregate_version`、`event_occurred_at`、`index_generation`、`rebuild_run_id`。
- [ ] `tsvector` 字段可使用 generated column 或显式更新；无论哪种方案，测试必须覆盖 title / summary / plain_text 权重和空文本处理。
- [ ] GIN / trigram index 覆盖关键词查询、suggest 前缀匹配、公开可见性过滤和 `published_at` 排序。
- [ ] `search_event_inbox` 记录 `event_id`、`event_type`、`aggregate_id`、`aggregate_version`、`occurred_at`、`processed_at`、`status`、`error_class`，用于幂等和乱序保护。
- [ ] migration up/down 可通过 `golang-migrate` 往返验证；文件名中的 `20260706xxxx` 必须在实施时替换为真实单调递增时间戳。
- [ ] Search 不创建或复制 Content `posts`、User `users`、Comment `comments` 表。
- [ ] repository test 覆盖 FTS 命中、trigram suggest、公开过滤、排序稳定、重复 event no-op 和 stale event ignored。

- [ ] **步骤 1：写 migration 和 repository 失败测试**
- [ ] **步骤 2：实现 migration、domain document 和 PostgreSQL repository**
- [ ] **步骤 3：运行验证**

运行：`cd services/zhicore-search && go test ./migrations ./internal/search/infrastructure/postgres -count=1`

预期：通过。

## 任务 3：Content 事件索引链路

**测试立场：** TDD - event consumer、幂等、硬删除和乱序属于 R4。

**文件：**
- 新增：`services/zhicore-search/internal/search/application/index_post_event.go`
- 新增：`services/zhicore-search/internal/search/application/rebuild_post_index.go`
- 新增：`services/zhicore-search/internal/search/application/*_test.go`
- 新增：`services/zhicore-search/internal/search/infrastructure/rabbitmq/content_post_consumer.go`
- 新增：`services/zhicore-search/internal/search/infrastructure/rabbitmq/content_post_consumer_test.go`
- 新增：`services/zhicore-search/internal/search/infrastructure/clients/content_client.go`
- 新增：`services/zhicore-search/internal/search/infrastructure/clients/content_client_test.go`
- 修改：`libs/contracts/events/content/contract.go`
- 修改：`libs/contracts/events/content/contract_test.go`

**验收清单：**
- [ ] `content.post.published`、`updated`、`deleted`、`visibility_changed`、`tags.updated` 都有明确处理。
- [ ] 重复 `eventId` ack no-op。
- [ ] `content.post.deleted` 必须硬删除 `search_post_documents` 对应行。
- [ ] 旧可见性事件不能覆盖新状态；需要 `aggregateVersion` 或发生时间保护。
- [ ] 正文回源只发生在索引写入 / 修复，不进入搜索查询主路径。
- [ ] producer payload 缺关键字段进入 retry / DLQ，不默默写入半残索引；DLQ envelope 至少包含 `eventId`、`eventType`、`routingKey`、`consumer`、`errorClass`、`occurredAt`、`failedAt`、`retryCount`、`payloadHash`，不得包含正文全文、raw token 或 credential。
- [ ] rebuild 使用 `index_generation` / `rebuild_run_id` 和 high-water mark 隔离重建写入；live consumer 在 barrier 期间暂停或只写新 generation，禁止 stale rebuild 覆盖新事件。
- [ ] rebuild 完成前 read path 继续读上一代公开索引；切换必须是单事务更新 active generation 或等价原子动作。

- [ ] **步骤 1：写事件 contract 和 application 失败测试**
- [ ] **步骤 2：实现 Content event Go types、index application 和 consumer adapter**
- [ ] **步骤 3：运行验证**

运行：`cd libs/contracts && go test ./events/content -count=1 && cd ../../services/zhicore-search && go test ./internal/search/application ./internal/search/infrastructure/rabbitmq -count=1`

预期：通过。

## 任务 4：最小查询面与 runtime

**测试立场：** TDD - handler contract、PostgreSQL 查询、配置和 health 属于 R4。

**文件：**
- 新增：`services/zhicore-search/api/http/handler.go`
- 新增：`services/zhicore-search/api/http/payloads.go`
- 新增：`services/zhicore-search/api/http/handler_test.go`
- 新增：`services/zhicore-search/internal/search/application/search_posts.go`
- 新增：`services/zhicore-search/internal/search/application/search_posts_test.go`
- 新增：`services/zhicore-search/internal/search/runtime/module.go`
- 新增：`services/zhicore-search/internal/search/runtime/module_test.go`
- 新增：`services/zhicore-search/cmd/server/main.go`
- 新增：`services/zhicore-search/cmd/server/config.go`
- 新增：`services/zhicore-search/cmd/server/server.go`
- 新增：`services/zhicore-search/configs/local.example.env`

**验收清单：**
- [ ] `/health/live` 与 `/health/ready` 落地。
- [ ] `GET /api/v1/search/posts` 的分页语义与 endpoint schema 一致，不在实现中临时切换 0-based / 1-based。
- [ ] PostgreSQL / Content 失败返回 `1004` 或 schema 登记的等价错误，不伪装空结果。
- [ ] 搜索结果只返回公开索引项；`publicVisible=false` 不进入结果。
- [ ] 启动路径不执行 migration。
- [ ] `cmd/server` 不写业务逻辑。
- [ ] `/health/ready` 依赖矩阵写清：PostgreSQL 为 HTTP 查询硬依赖；Redis、RabbitMQ、Content client 按启用的 suggest cache、consumer、rebuild worker 配置进入 ready 或 degraded details。

- [ ] **步骤 1：写 search handler / config / health 失败测试**
- [ ] **步骤 2：实现 query use case、handler 和 runtime**
- [ ] **步骤 3：运行验证**

运行：`cd services/zhicore-search && go test ./api/http ./internal/search/application ./cmd/server ./internal/search/runtime -count=1`

预期：通过。

## 任务 5：suggest、hot 和 history 闭合

**测试立场：** TDD - 新 endpoint、登录态、幂等清理和统计窗口属于 R3/R4。

**文件：**
- 新增：`services/zhicore-search/internal/search/application/list_suggestions.go`
- 新增：`services/zhicore-search/internal/search/application/list_hot_terms.go`
- 新增：`services/zhicore-search/internal/search/application/list_search_history.go`
- 新增：`services/zhicore-search/internal/search/application/clear_search_history.go`
- 新增：`services/zhicore-search/api/http/suggest_history_handlers.go`
- 新增：`services/zhicore-search/api/http/suggest_history_handlers_test.go`
- 新增或修改：`services/zhicore-search/internal/search/infrastructure/postgres/*.go`

**验收清单：**
- [ ] `GET /api/v1/search/history` 和 `DELETE /api/v1/search/history` 需要登录态。
- [ ] 清理历史幂等，多次调用返回同一成功语义。
- [ ] `/suggest` 固定前缀匹配、空词和最小长度语义。
- [ ] `/hot` 固定统计窗口、去重口径、空列表语义。
- [ ] hot term / history 数据归属在 Search 本地，不写入 Content 或 User。

- [ ] **步骤 1：写 handler / application 失败测试**
- [ ] **步骤 2：实现 suggest、hot、history**
- [ ] **步骤 3：运行验证**

运行：`cd services/zhicore-search && go test ./internal/search/application ./api/http -run 'Suggest|Hot|History' -count=1`

预期：通过。

## 集成验证

- [ ] 运行 `cd libs/contracts && go test ./clients/content ./events/content -count=1`。
- [ ] 运行 `cd services/zhicore-search && go test ./... -count=1`。
- [ ] 运行 `cd services/zhicore-search && go test -race ./internal/search/... -run 'Consumer|Rebuild|Generation|Search' -count=1`。
- [ ] 有可用 `ZHICORE_SEARCH_POSTGRES_DSN` 时运行 `migrate -path services/zhicore-search/migrations -database "$ZHICORE_SEARCH_POSTGRES_DSN" up && migrate -path services/zhicore-search/migrations -database "$ZHICORE_SEARCH_POSTGRES_DSN" down 1 && migrate -path services/zhicore-search/migrations -database "$ZHICORE_SEARCH_POSTGRES_DSN" up`；没有 DSN 时必须用隔离 PostgreSQL 容器执行同等 `up -> down 1 -> up`，或在交付说明中列为未验证的外部依赖。
- [ ] 运行 `make test-size`。
- [ ] 运行 `bash scripts/check-structure.sh`。
- [ ] 触达共享 contract、PostgreSQL 搜索索引、事件 consumer 后运行 `make check`。
