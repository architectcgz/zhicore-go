# Content 发布闭环基础实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；涉及 domain、application、HTTP contract、migration、repository、MongoDB store、outbox 和 copy-on-write 失败语义的步骤按 @test-driven-development 执行。每个 checkbox 达到预期后立即更新；如需提交，提交前必须先使用 @committing-changes。

**目标：** 在 `zhicore-content` 中实现创建草稿、保存正文、发布文章和读取 published body 的最小闭环。

**架构：** Content 拥有文章可见性、草稿指针、published 指针、正文 cleanup / repair 任务和发布 outbox。PostgreSQL 是可见性真相源，MongoDB 只保存 body；保存草稿和发布都使用 copy-on-write，不能让失败写入污染线上正文。

**技术栈：** Go 1.26、Gin HTTP router、PostgreSQL migration、MongoDB body store、RabbitMQ outbox、`libs/kit/httpapi`、Content V1 body parser。

---

## 背景依据

- `docs/migration/service-migration-workflow.md`
- `docs/architecture/services/content/README.md`
- `docs/architecture/services/content/domain-model.md`
- `docs/architecture/services/content/body-storage-and-publishing.md`
- `docs/architecture/services/content/application-and-ports.md`
- `docs/architecture/services/content/runtime-resilience.md`
- `docs/architecture/go-service-design.md`
- `docs/architecture/migrations.md`
- `docs/architecture/testing.md`
- `docs/contracts/http-schema-template.md`
- `docs/contracts/events.md`
- `services/zhicore-content/api/http/README.md`
- `services/zhicore-content/api/http/endpoints/create-post.md`
- `services/zhicore-content/api/http/endpoints/save-draft-body.md`
- `services/zhicore-content/api/http/endpoints/publish-post.md`
- `services/zhicore-content/api/http/endpoints/get-post-body.md`

## 范围

本计划只覆盖 Content 首个发布闭环：

- `POST /api/v1/posts`
- `PUT /api/v1/posts/{postId}/draft/body`
- `POST /api/v1/posts/{postId}/publish`
- `GET /api/v1/posts/{postId}/body`
- Content core PostgreSQL schema、MongoDB body store、正文 cleanup / repair 任务记录、发布 outbox 和 runtime 最小装配。

不在本计划处理：

- 文章列表、文章详情元数据、读取草稿、草稿 meta 更新、删除/恢复/撤回/定时发布。
- 点赞、收藏、标签、分类管理、reader presence、管理端、Search / Ranking consumer、link preview。
- 前端 adapter 或页面改动。

## 文件结构

- 新增：`services/zhicore-content/migrations/20260705093000_create_content_publish_core.up.sql`
- 新增：`services/zhicore-content/migrations/20260705093000_create_content_publish_core.down.sql`
- 新增：`services/zhicore-content/migrations/migration_contract_test.go`
- 新增：`services/zhicore-content/internal/content/domain/errors.go`
- 新增：`services/zhicore-content/internal/content/domain/post.go`
- 新增：`services/zhicore-content/internal/content/domain/post_events.go`
- 新增：`services/zhicore-content/internal/content/domain/post_test.go`
- 新增：`services/zhicore-content/internal/content/ports/post.go`
- 新增：`services/zhicore-content/internal/content/ports/body_store.go`
- 新增：`services/zhicore-content/internal/content/ports/outbox.go`
- 新增：`services/zhicore-content/internal/content/ports/tasks.go`
- 新增：`services/zhicore-content/internal/content/ports/clients.go`
- 新增：`services/zhicore-content/internal/content/application/service.go`
- 新增：`services/zhicore-content/internal/content/application/create_post_test.go`
- 新增：`services/zhicore-content/internal/content/application/save_draft_body_test.go`
- 新增：`services/zhicore-content/internal/content/application/publish_post_test.go`
- 新增：`services/zhicore-content/internal/content/application/get_published_body_test.go`
- 新增：`services/zhicore-content/internal/content/infrastructure/postgres/post_repository.go`
- 新增：`services/zhicore-content/internal/content/infrastructure/postgres/post_repository_test.go`
- 新增：`services/zhicore-content/internal/content/infrastructure/mongo/body_store.go`
- 新增：`services/zhicore-content/internal/content/infrastructure/mongo/body_store_test.go`
- 新增：`services/zhicore-content/internal/content/runtime/module.go`
- 新增：`services/zhicore-content/internal/content/runtime/module_test.go`
- 新增：`services/zhicore-content/api/http/handler.go`
- 新增：`services/zhicore-content/api/http/create_post_handler_test.go`
- 新增：`services/zhicore-content/api/http/save_draft_body_handler_test.go`
- 新增：`services/zhicore-content/api/http/publish_post_handler_test.go`
- 新增：`services/zhicore-content/api/http/get_post_body_handler_test.go`
- 修改：`services/zhicore-content/api/http/README.md`
- 修改：`services/zhicore-content/api/http/endpoints/create-post.md`
- 修改：`services/zhicore-content/api/http/endpoints/save-draft-body.md`
- 修改：`services/zhicore-content/api/http/endpoints/publish-post.md`
- 修改：`services/zhicore-content/api/http/endpoints/get-post-body.md`
- 修改：`services/zhicore-content/cmd/server/main.go`

## 任务 1：Content core migration

**测试立场：** TDD - schema、唯一约束、状态枚举、outbox 和 cleanup / repair 任务属于 R4。

- [x] **步骤 1：编写 migration contract 测试**

  覆盖 migration SQL 至少包含 `posts`、`post_stats`、`outbox_events`、`domain_event_tasks`、`content_body_cleanup_tasks`、`content_body_repair_tasks`；检查 `posts.public_id` 唯一、`posts.status` 枚举约束、published / draft 指针字段和 cleanup task 幂等键。

  运行：`cd services/zhicore-content && go test ./migrations -run TestContentPublishCoreMigrationContract`

  预期：失败，因为 migration 尚未落地。

- [x] **步骤 2：新增 migration pair**

  新增 `20260705093000_create_content_publish_core.up.sql` 和 `.down.sql`。`up` 使用显式 `BEGIN` / `COMMIT`，只创建 Content 自有表，不创建跨服务外键；`down` 撤销本次 schema。

- [x] **步骤 3：运行 migration contract 测试**

  运行：`cd services/zhicore-content && go test ./migrations`

  预期：通过。

- [x] **步骤 4：记录真实数据库验证命令**

  有可用 `ZHICORE_CONTENT_POSTGRES_DSN` 时运行：

  ```bash
  migrate -path services/zhicore-content/migrations -database "$ZHICORE_CONTENT_POSTGRES_DSN" up
  migrate -path services/zhicore-content/migrations -database "$ZHICORE_CONTENT_POSTGRES_DSN" down 1
  migrate -path services/zhicore-content/migrations -database "$ZHICORE_CONTENT_POSTGRES_DSN" up
  ```

  预期：`up -> down 1 -> up` 成功，`schema_migrations` 非 dirty。没有 DSN 时，在交付说明中把该项列为未跑的外部依赖验证。

  已验证：`ZHICORE_CONTENT_POSTGRES_DSN` 未设置，因此使用隔离临时 PostgreSQL 容器 `postgres:16.14-alpine` 和 `migrate/migrate:v4.18.3` 执行真实 `up -> down 1 -> up`。最终 `schema_migrations` 为 `20260705093000 dirty=false`，关键表 `posts`、`post_stats`、`outbox_events`、`domain_event_tasks`、`content_body_cleanup_tasks`、`content_body_repair_tasks` 和关键索引已确认存在；`PUBLISHED` check 约束已确认要求 `published_body_hash`、`published_plain_text_length`、`published_at` 非空。

## 任务 2：Post domain 和发布不变量

**测试立场：** TDD - 生命周期状态、发布 guard、标题/正文不变量和领域事件属于 R4。

- [x] **步骤 1：编写 domain 失败测试**

  覆盖 `PostFactory.CreateDraft`、标题 trim / 长度、空标题不能发布、空正文不能发布、已删除不能保存草稿、已发布不能重复发布、发布成功产生 `PostPublished`。

  运行：`cd services/zhicore-content && go test ./internal/content/domain -run 'TestPostFactory|TestPostPublish'`

  预期：失败，因为 domain 尚未实现。

- [x] **步骤 2：实现 domain 类型**

  新增 `PostID`、`PublicPostID`、`OwnerID`、`PostStatus`、`PostTitle`、`PostSummary`、`BodyPointer`、`OwnerSnapshot`、`Post`、`PostFactory` 和 `PostPublishPolicy`。领域层只表达业务不变量，不依赖 HTTP、PostgreSQL、MongoDB、RabbitMQ 或 parser。

- [x] **步骤 3：实现领域事件和错误**

  新增 `PostCreated`、`PostPublished`、`ErrPostNotFound`、`ErrForbidden`、`ErrPostAlreadyPublished`、`ErrPostDeleted`、`ErrTitleRequired`、`ErrBodyRequired`、`ErrBodyTooShort`、`ErrDraftConflict`、`ErrBodyUnavailable`。

- [x] **步骤 4：运行 domain 测试**

  运行：`cd services/zhicore-content && go test ./internal/content/domain`

  预期：通过。

## 任务 3：Ports 和 application 发布编排

**测试立场：** TDD - 权限、copy-on-write、事务、outbox、cleanup / repair 和错误映射属于 R4。

- [x] **步骤 1：定义端口和 application DTO**

  定义 `PostRepository`、`PostQueryRepository`、`PostContentStore`、`OutboxPublisher`、`BodyCleanupTaskStore`、`BodyRepairTaskStore`、`UserProfileClient`、`FileResourceClient`、`TransactionRunner`、`Clock` 和复用现有 `BodyParserRegistry`。Application 对外只暴露自有 command / query / result，不导出 domain alias。

- [x] **步骤 2：编写 `CreatePost` 失败测试**

  覆盖缺少 actor、带 body 创建草稿、空草稿、作者快照获取、初始 `post_stats`、非法标题、非法 body、MongoDB 写 body 失败时不创建可见 post。

  运行：`cd services/zhicore-content && go test ./internal/content/application -run TestCreatePost`

  预期：失败。

- [x] **步骤 3：实现 `CreatePost`**

  使用 actor 作为 owner；可选 body 先经 `BodyParserRegistry` 标准化并写 MongoDB draft body，再在 PostgreSQL 事务内创建 `posts`、`post_stats` 和必要 cleanup 记录。创建草稿不发布公开事件。

- [x] **步骤 4：编写 `SaveDraftBody` 失败测试**

  覆盖 owner 校验、`basePostVersion` 冲突、`baseDraftBodyId` / `baseDraftBodyHash` 冲突、body hash no-op、copy-on-write 写新 draft、旧 draft cleanup task、MongoDB 写成功但 PG 更新失败时新 draft 进入 orphan cleanup。

  运行：`cd services/zhicore-content && go test ./internal/content/application -run TestSaveDraftBody`

  预期：失败。

- [x] **步骤 5：实现 `SaveDraftBody`**

  校验 body schema 和媒体引用；MongoDB 写新 draft body 后，PostgreSQL 事务条件更新 draft 指针和 `post_version`。事务失败时按 `body-storage-and-publishing.md` 记录 orphan cleanup，不修改 published 指针。

- [x] **步骤 6：编写 `PublishPost` 失败测试**

  覆盖 owner 校验、标题为空、正文为空、9 个有效 rune 拒绝、10 个有效 rune 允许、重复发布、草稿冲突、draft body miss、hash 冲突、MongoDB snapshot 写失败、MongoDB snapshot 写成功但 PG 事务失败、发布成功写带完整 payload / `aggregateVersion` 的 `content.post.published` outbox。

  运行：`cd services/zhicore-content && go test ./internal/content/application -run TestPublishPost`

  预期：失败。

- [x] **步骤 7：实现 `PublishPost`**

  读取 draft body 并校验 hash、正文、媒体和封面；写 MongoDB snapshot body 后，用 PostgreSQL 事务切换 `published_*`、清空或推进 draft 指针、递增 `post_version`、写 outbox 和 cleanup task。PG 失败时不直接删除 snapshot，只登记独立 orphan cleanup task，后续 worker 删除前必须确认 PostgreSQL 已无引用。

- [x] **步骤 8：编写 `GetPublishedPostBody` 失败测试**

  覆盖草稿不可见、已删除不可见、published body miss 写 repair task、hash 冲突、schema 不可读和成功返回 body。

  运行：`cd services/zhicore-content && go test ./internal/content/application -run TestGetPublishedPostBody`

  预期：失败。

- [x] **步骤 9：实现 `GetPublishedPostBody`**

  查询 `posts.published_body_id`，只允许 `PUBLISHED` 且未删除正文读取；MongoDB miss 或 hash mismatch 返回对应 application error，并写 repair task。

- [x] **步骤 10：运行 application 收口测试**

  运行：`cd services/zhicore-content && go test ./internal/content/application`

  预期：通过。

## 任务 4：PostgreSQL repository 和 MongoDB body store

**测试立场：** TDD - repository 条件更新、唯一约束、cleanup / repair、MongoDB body hash 和 context 传播属于 R4。

- [x] **步骤 1：编写 PostgreSQL repository 失败测试**

  覆盖 public ID 唯一、owner 条件查询、draft 指针 CAS 更新、publish 指针 CAS 更新、outbox 写入、cleanup / repair task 幂等写入和底层错误翻译。

  运行：`cd services/zhicore-content && go test ./internal/content/infrastructure/postgres`

  预期：失败。

- [x] **步骤 2：实现 PostgreSQL repository**

  Repository 使用显式 mapper，不让 GORM / SQL row model 进入 domain 或 application。所有查询传递 `context.Context`，事务边界由 `TransactionRunner` 或 repository 显式方法承载。

- [x] **步骤 3：编写 MongoDB body store 失败测试**

  覆盖写 draft body、写 snapshot body、读取 published body、hash mismatch、context cancel、精确 body id delete 和删除不存在 body 幂等成功。

  运行：`cd services/zhicore-content && go test ./internal/content/infrastructure/mongo`

  预期：失败。

- [x] **步骤 4：实现 MongoDB body store**

  Body store 只保存 body 文档，不决定 published 可见性；所有公开可见性以 PostgreSQL 指针为准。删除 cleanup 前必须由 application / repository 确认 body 未被 `posts` 引用。

- [x] **步骤 5：运行 infrastructure 收口测试**

  运行：

  ```bash
  cd services/zhicore-content
  go test ./internal/content/infrastructure/postgres ./internal/content/infrastructure/mongo
  ```

  预期：通过。

## 任务 5：Gin HTTP handler 和 contract 状态

**测试立场：** TDD - path、可信身份 header、request / response DTO、envelope 和公开错误码属于 R3 / R4。

- [ ] **步骤 1：编写 `CreatePost` handler 失败测试**

  覆盖 `POST /api/v1/posts`、缺 `X-User-Id` 返回 `2006`、空草稿、带 body 创建、标题过长、非法 body 和成功 envelope。

  运行：`cd services/zhicore-content && go test ./api/http -run TestCreatePost`

  预期：失败。

- [ ] **步骤 2：编写 `SaveDraftBody` handler 失败测试**

  覆盖 `PUT /api/v1/posts/{postId}/draft/body`、作者鉴权、版本冲突、body schema 错误、正文过大、request body 超限、context cancel 和成功 envelope。

  运行：`cd services/zhicore-content && go test ./api/http -run TestSaveDraftBody`

  预期：失败。

- [ ] **步骤 3：编写 `PublishPost` handler 失败测试**

  覆盖 `POST /api/v1/posts/{postId}/publish`、缺登录态、非作者、标题为空、正文为空、草稿冲突、重复发布、发布依赖不可用和成功 envelope。

  运行：`cd services/zhicore-content && go test ./api/http -run TestPublishPost`

  预期：失败。

- [ ] **步骤 4：编写 `GetPublishedPostBody` handler 失败测试**

  覆盖 `GET /api/v1/posts/{postId}/body`、草稿不可见、已删除不可见、body miss、hash 冲突、schema 不可读和成功 envelope。

  运行：`cd services/zhicore-content && go test ./api/http -run TestGetPostBody`

  预期：失败。

- [ ] **步骤 5：实现 Gin handler**

  `api/http` 只做协议绑定、可信 header 映射、请求体大小限制、DTO 转换和错误映射。当前操作者只来自 `X-User-Id`，body 中的 `userId` / `ownerId` / `actor` 字段不得覆盖身份。

- [ ] **步骤 6：更新 endpoint 状态**

  将被 handler contract test 覆盖的 `create-post.md`、`save-draft-body.md`、`publish-post.md`、`get-post-body.md` 和 `services/zhicore-content/api/http/README.md` 状态从“草案”更新为“已验证”，并写明测试文件。

- [ ] **步骤 7：运行 HTTP 收口测试**

  运行：`cd services/zhicore-content && go test ./api/http`

  预期：通过。

## 任务 6：Runtime、server 入口和最终验证

**测试立场：** TDD - runtime 装配、fail fast、worker 生命周期和配置校验属于 R4。

- [ ] **步骤 1：编写 runtime module 失败测试**

  覆盖缺 PostgreSQL / MongoDB / body parser / outbox / clock / config 时 fail fast；构造成功时返回 HTTP handler、cleanup / repair / outbox worker 描述和 health details。

  运行：`cd services/zhicore-content && go test ./internal/content/runtime`

  预期：失败。

- [ ] **步骤 2：实现 runtime module**

  `runtime.Build` 只做依赖装配和配置校验，不执行 migration，不写业务逻辑。cleanup / repair / outbox worker 可以先暴露为可启动组件；如真实 worker 未在本切片闭合，必须 fail fast 或明确返回 disabled 状态，不能伪装生产可运行。

- [ ] **步骤 3：补 `cmd/server` 最小入口**

  `cmd/server/main.go` 只调用 runtime 装配、启动 HTTP server 和处理 shutdown；不直接 new repository、写 handler 业务逻辑或执行 migration。

- [ ] **步骤 4：运行服务内测试**

  运行：`cd services/zhicore-content && go test ./...`

  预期：通过。

- [ ] **步骤 5：运行测试规模检查**

  运行：`python3 scripts/check-test-size.py --files services/zhicore-content`

  预期：通过。

- [ ] **步骤 6：运行结构检查**

  运行：`bash scripts/check-structure.sh`

  预期：`structure ok`。

- [ ] **步骤 7：按完成标准准备 review 证据**

  记录实际执行过的 `go test`、migration 验证、`check-test-size` 和 `check-structure` 输出摘要。若没有真实 PostgreSQL / MongoDB DSN，明确列为残余风险，不写成已验证。

## 架构适配评估

- 计划遵守 `api/http -> application -> domain/ports -> infrastructure -> runtime` 的依赖方向；handler 不持有业务事务，repository 不做权限判断。
- HTTP DTO 只留在 `services/zhicore-content/api/http`，application 对外暴露自有 command / query / result，domain 不被导出别名穿透到入站层。
- PostgreSQL 负责 published / draft 指针和可见性，MongoDB 只保存 body；发布失败方向有 cleanup / repair 收敛路径。
- 本计划只实现 Content 首个闭环，不把点赞、收藏、标签、管理端、presence、Search / Ranking consumer 或 link preview 塞入同一提交链。
- Ranking / Search 后续依赖的 `content.post.published` outbox 边界在本计划中落地，但 consumer 和下游投影不在本计划内实现。

## 交付说明

- 每个任务完成后先更新本计划 checkbox，再进入下一任务。
- 每个可运行切片都要留下最小验证证据；没有跑真实 DB / MongoDB / 全量测试时，不得声称全仓或真实依赖已验证。
- 本计划完成后，长期结论需要回写到 Content 服务文档、HTTP endpoint 状态、review 证据和必要的技术债记录。
