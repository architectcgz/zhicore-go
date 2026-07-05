# Content 模块补全实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；涉及 runtime、worker、HTTP contract、application、repository、client adapter、限流和系统测试的步骤按 @test-driven-development 执行。每个 checkbox 达到预期后立即更新；如需提交，提交前必须先使用 @committing-changes，且必须按本计划的最小可审阅切片拆分提交，禁止把 runtime、worker、多个 API 家族和文档同步合成一个大提交。

**目标：** 把 `zhicore-content` 从已完成的发布闭环 foundation 推进到可运行、可运维、API 面逐步完整且有系统验证的 Content 服务。

**架构：** 保持 `api/http -> application -> domain/ports -> infrastructure -> runtime` 依赖方向；PostgreSQL 继续作为文章可见性、metadata、outbox 和任务状态真相源，MongoDB 只保存正文 body。HTTP handler 只做协议绑定和错误映射，运行时依赖、配置、worker lifecycle 和下游 resilience policy 统一在 `internal/content/runtime` 与进程根装配。

**技术栈：** Go 1.26、Gin HTTP router、PostgreSQL、MongoDB、RabbitMQ outbox、Redis rate limit/cache、`libs/kit/httpapi`、`libs/kit/postgres/outbox`、Content V1 body parser、黑盒 HTTP system test。

---

## 背景依据

- `docs/plan/impl-plan/2026-07-05-content-publish-foundation-implementation-plan.md`
- `docs/reviews/backend/2026-07-05-content-publish-foundation.md`
- `docs/architecture/services/content/README.md`
- `docs/architecture/services/content/body-storage-and-publishing.md`
- `docs/architecture/services/content/application-and-ports.md`
- `docs/architecture/services/content/data-events-contracts.md`
- `docs/architecture/services/content/rate-limiting.md`
- `docs/architecture/services/content/runtime-resilience.md`
- `docs/architecture/go-service-design.md`
- `docs/architecture/configuration.md`
- `docs/architecture/runtime-operations.md`
- `docs/architecture/observability.md`
- `docs/architecture/testing.md`
- `docs/contracts/http.md`
- `docs/contracts/http-schema-template.md`
- `docs/contracts/errors.md`
- `docs/contracts/events.md`
- `services/zhicore-content/api/http/README.md`

## 当前基线

已完成并提交的 foundation 范围：

- `POST /api/v1/posts`
- `PUT /api/v1/posts/{postId}/draft/body`
- `POST /api/v1/posts/{postId}/publish`
- `GET /api/v1/posts/{postId}/body`
- Content core migration、domain、application、PostgreSQL repository、MongoDB body store、HTTP handler、runtime module fail-fast foundation 和 review 证据。

当前残余缺口：

- 分支 `task/2026-07-05-content-publish-foundation` 尚未合并回主线。
- Content 服务还不是生产可运行进程：缺配置加载、依赖打开、HTTP server listen/shutdown 和真实 readiness。
- cleanup / repair / outbox worker 当前是 disabled descriptor。
- `4012` / `4021` / `4023` 仍等待 application / ports sentinel error 和 handler mapping。
- 缺 create -> save draft -> publish -> get body 的黑盒 HTTP system test 和真实 MongoDB 端到端验证。
- 多数 Content API family 仍停留在草案或未拆单 endpoint contract。
- 架构 README 和服务 README 需要同步当前状态。
- 限流、resilience policy、日志、metrics 和运行配置模板需要落地。

## 范围

本计划覆盖：

- 当前 foundation 的集成收口和文档状态同步。
- Content 可运行 runtime、配置、依赖打开、HTTP server、健康检查和优雅停机。
- File / User 下游 client adapter、分类 / 话题 / 标签引用校验语义错误、媒体和封面错误映射。
- cleanup worker、repair worker、outbox dispatcher 和 admin retry 基础。
- Content 黑盒 HTTP system test 和本地依赖测试 fixture。
- 剩余 Content API family：公开文章查询、作者工作台、发布生命周期、标签/分类/话题、互动、presence、管理端。
- 限流、resilience、observability 和最终 review 证据。

不在本计划处理：

- Search、Ranking、Notification、Comment 的 consumer 实现。
- 前端 adapter 或页面改动。
- Link preview 抓取与 SSRF-safe fetcher。
- 旧 Java API path / DTO 兼容。

## 文件结构

- 修改：`docs/architecture/services/content/README.md`
- 修改：`services/zhicore-content/README.md`
- 修改：`services/zhicore-content/api/http/README.md`
- 新增或修改：`services/zhicore-content/api/http/endpoints/*.md`
- 修改：`services/zhicore-content/api/http/handler.go`
- 新增或修改：`services/zhicore-content/api/http/*_handler_test.go`
- 修改：`services/zhicore-content/cmd/server/main.go`
- 新增：`services/zhicore-content/cmd/server/config_test.go`
- 新增：`services/zhicore-content/cmd/server/server_test.go`
- 新增：`services/zhicore-content/configs/local.example.env`
- 修改：`services/zhicore-content/internal/content/application/service.go`
- 新增或修改：`services/zhicore-content/internal/content/application/*_test.go`
- 修改：`services/zhicore-content/internal/content/ports/*.go`
- 新增：`services/zhicore-content/internal/content/infrastructure/clients/user_client.go`
- 新增：`services/zhicore-content/internal/content/infrastructure/clients/user_client_test.go`
- 新增：`services/zhicore-content/internal/content/infrastructure/clients/file_client.go`
- 新增：`services/zhicore-content/internal/content/infrastructure/clients/file_client_test.go`
- 新增或修改：`services/zhicore-content/internal/content/infrastructure/postgres/*.go`
- 新增或修改：`services/zhicore-content/internal/content/infrastructure/mongo/*.go`
- 新增：`services/zhicore-content/internal/content/infrastructure/rabbitmq/event_publisher.go`
- 新增：`services/zhicore-content/internal/content/infrastructure/rabbitmq/event_publisher_test.go`
- 新增或修改：`services/zhicore-content/internal/content/runtime/*.go`
- 新增或修改：`services/zhicore-content/internal/content/runtime/*_test.go`
- 新增：`tests/system/http/content_publish_flow_test.go`
- 新增或修改：`tests/testkit/**`
- 新增：`docs/reviews/backend/<date>-content-module-completion.md`

## 提交拆分规则

- 每个任务至少一个独立提交；任务内部如果同时修改 contract、application、infrastructure、runtime 或 docs，必须按可审阅边界继续拆分。
- 合理提交粒度示例：
  - `docs(content): 同步发布闭环后续状态`
  - `feat(content): 接入可运行 runtime 配置`
  - `test(content): 覆盖正文发布系统场景`
  - `feat(content): 实现 outbox dispatcher`
  - `feat(content): 实现公开文章查询接口`
- 每次提交前必须使用 @committing-changes，精确暂存路径，不使用 `git add -A`。

## 任务 0：合并前置和文档状态同步

**测试立场：** R0 文档和 Git 集成切片；不改业务行为。

- [ ] **步骤 1：确认 foundation 分支状态**

  运行：`git status --short --branch && git branch --contains HEAD --all`

  预期：worktree 干净；若当前提交仍只在 task 分支，先按项目 finishing 流程决定 merge / PR / 保留 worktree。

- [ ] **步骤 2：同步 Content 架构 README**

  修改 `docs/architecture/services/content/README.md`：把“当前设计状态”和“下一步”更新为发布闭环 foundation 已完成、runtime / worker / API family / system test 待补。

- [ ] **步骤 3：同步服务 README**

  修改 `services/zhicore-content/README.md`：补当前已实现 endpoint、运行状态、验证命令、剩余切片入口和本计划链接。

- [ ] **步骤 4：运行文档结构验证**

  运行：`bash scripts/check-structure.sh && git diff --check`

  预期：`structure ok`，无 whitespace error。

- [ ] **步骤 5：提交文档同步切片**

  提交前使用 @committing-changes。提交只包含本任务文档文件。

## 任务 1：可运行 runtime、配置和 HTTP server

**测试立场：** TDD - 启动配置、依赖打开、健康检查、server timeout 和 graceful shutdown 属于 R4。

- [ ] **步骤 1：编写配置加载失败测试**

  测试 `cmd/server` 或 runtime config loader：缺少 PostgreSQL DSN、Mongo URI、RabbitMQ URL、User/File base URL 时 fail fast；默认 HTTP timeout、shutdown timeout、worker disabled/enabled 开关和最大 request body 可见。

  运行：`cd services/zhicore-content && go test ./cmd/server -run TestLoadContentServerConfig`

  预期：失败。

- [ ] **步骤 2：实现配置结构和加载**

  新增服务私有配置，环境变量遵守 `ZHICORE_CONTENT_*` 命名。敏感字段提供脱敏摘要，不在日志和错误中输出明文 DSN / URL secret。

- [ ] **步骤 3：补本地配置模板**

  新增 `services/zhicore-content/configs/local.example.env`，只写 fake 示例值和本地默认 timeout，不提交真实凭证。

- [ ] **步骤 4：编写 runtime readiness 失败测试**

  覆盖 `/health/live` 不检查依赖，`/health/ready` 检查 PostgreSQL ping、Mongo ping、RabbitMQ publisher 状态和 worker descriptor；依赖失败返回非 ready。

  运行：`cd services/zhicore-content && go test ./internal/content/runtime -run TestHealthReadiness`

  预期：失败。

- [ ] **步骤 5：实现 runtime readiness checker**

  在 `internal/content/runtime` 中引入可注入 `HealthChecker` / dependency checker；ready 不执行昂贵查询、不写业务数据。

- [ ] **步骤 6：编写 HTTP server 生命周期测试**

  覆盖 `http.Server` timeout 配置、SIGINT/SIGTERM shutdown path、readiness 关闭顺序和依赖 close 调用。

  运行：`cd services/zhicore-content && go test ./cmd/server -run TestContentServerLifecycle`

  预期：失败。

- [ ] **步骤 7：实现 `cmd/server` 可运行入口**

  `cmd/server/main.go` 只负责加载配置、打开依赖、调用 runtime、启动 HTTP server 和 shutdown；不放业务 wiring，不执行 migration。

- [ ] **步骤 8：运行 runtime 收口测试**

  运行：`cd services/zhicore-content && go test ./cmd/server ./internal/content/runtime`

  预期：通过。

- [ ] **步骤 9：提交 runtime 切片**

  提交前使用 @committing-changes。配置、runtime 和 server 生命周期可以按“配置加载”和“HTTP server 生命周期”拆成两个提交。

## 任务 2：下游 client adapter 和语义错误补齐

**测试立场：** TDD - 下游错误分类、retry 边界、公开错误码和 handler mapping 属于 R4。

- [ ] **步骤 1：固定 application / ports sentinel error**

  在 `ports` 或 application 层新增可分支错误：分类 / 话题 / 标签引用不存在、媒体引用非法、封面不可用。不得通过匹配下游错误文本做分支。

- [ ] **步骤 2：编写 application 失败测试**

  覆盖 create / save draft / publish 中：
  - 分类 / 话题 / 标签不存在 -> `4012`
  - 正文媒体非法 -> `4021`
  - 发布封面不可用 -> `4023`
  - 下游不可用 -> `1004`

  运行：`cd services/zhicore-content && go test ./internal/content/application -run 'TestCreatePost|TestSaveDraftBody|TestPublishPost'`

  预期：失败。

- [ ] **步骤 3：实现 application 错误传播**

  application 只依赖 sentinel error，不关心 HTTP status 或下游传输细节。

- [ ] **步骤 4：编写 User client adapter 测试**

  使用 `httptest.Server` 覆盖用户快照成功、404、5xx、timeout、context cancel、envelope 错误和敏感 URL 不泄漏。

  运行：`cd services/zhicore-content && go test ./internal/content/infrastructure/clients -run TestUserClient`

  预期：失败。

- [ ] **步骤 5：实现 User client adapter**

  从对应 client contract 读取 path 和 DTO；adapter 只做传输和错误翻译，不构造业务成功假象。

- [ ] **步骤 6：编写 File client adapter 测试**

  覆盖媒体引用校验、封面校验、404/410 语义错误、5xx 依赖错误、timeout 和 retry 次数。

  运行：`cd services/zhicore-content && go test ./internal/content/infrastructure/clients -run TestFileClient`

  预期：失败。

- [ ] **步骤 7：实现 File client adapter**

  明确 `file.validate_ref` 和 cover validation operation，按 `runtime-resilience.md` 配置 timeout / retry / max-in-flight。

- [ ] **步骤 8：补 handler mapping 和 contract test**

  补 `api/http` 测试覆盖 `4012`、`4021`、`4023`，更新 `services/zhicore-content/api/http/README.md` 的“待补错误映射”状态。

  运行：`cd services/zhicore-content && go test ./api/http -run 'TestCreatePost|TestSaveDraftBody|TestPublishPost'`

  预期：通过。

- [ ] **步骤 9：提交 client 和错误契约切片**

  推荐拆分为 sentinel / application、User client、File client、HTTP mapping 四个提交。

## 任务 3：cleanup worker 和 repair worker

**测试立场：** TDD - claim、幂等删除、PG 引用二次确认、retry、dead-letter 和 shutdown 属于 R4。

- [ ] **步骤 1：扩展 PostgreSQL task repository 测试**

  覆盖 cleanup / repair task claim、stale claim 重领、多实例不重复 claim、mark succeeded、mark failed、dead threshold 和条件更新。

  运行：`cd services/zhicore-content && go test ./internal/content/infrastructure/postgres -run 'TestCleanupTask|TestRepairTask'`

  预期：失败。

- [ ] **步骤 2：实现 task repository claim 状态机**

  使用 `FOR UPDATE SKIP LOCKED` 或等价条件更新；不得让多个 worker 同时处理同一任务。

- [ ] **步骤 3：编写 cleanup worker 失败测试**

  覆盖删除前查询 PG 指针未引用、Mongo body 不存在时幂等成功、被引用时跳过并重试/失败、Mongo delete 失败退避、context cancel 后不再 claim 新任务。

  运行：`cd services/zhicore-content && go test ./internal/content/application -run TestBodyCleanupWorker`

  预期：失败。

- [ ] **步骤 4：实现 cleanup worker**

  cleanup worker 只能按 `body_id` 精确删除；删除前必须确认 `posts.published_body_id` 和 `posts.draft_body_id` 都未引用该 body。

- [ ] **步骤 5：编写 repair worker 失败测试**

  覆盖 published body missing、hash mismatch、schema unreadable 的修复任务处理；第一阶段可只标记 `NEEDS_MANUAL_REPAIR` / `DEAD` 并记录告警字段，不伪造自动修复成功。

  运行：`cd services/zhicore-content && go test ./internal/content/application -run TestBodyRepairWorker`

  预期：失败。

- [ ] **步骤 6：实现 repair worker**

  repair worker 不读取 draft 冒充 published，不直接修改线上指针；能重试、dead-letter、暴露状态给 admin 查询。

- [ ] **步骤 7：接入 runtime worker descriptors**

  当配置启用 worker 时返回 enabled descriptor，并由 lifecycle owner 启动 / 停止；未启用时仍明确 disabled reason。

  运行：`cd services/zhicore-content && go test ./internal/content/runtime -run TestContentWorkers`

  预期：通过。

- [ ] **步骤 8：提交 cleanup / repair 切片**

  cleanup 和 repair 分别提交；repository 状态机可单独提交。

## 任务 4：outbox dispatcher、RabbitMQ publisher 和 admin retry 基础

**测试立场：** TDD - outbox claim、publish confirm、retry、dead-letter、admin retry 和事件 envelope 属于 R4。

- [ ] **步骤 1：编写 outbox dispatch repository 测试**

  覆盖 `PENDING / FAILED -> CLAIMING -> PUBLISHED / FAILED / DEAD`、stale claim 重领、`next_retry_at`、claim lost 和多实例不重复 claim。

  运行：`cd services/zhicore-content && go test ./internal/content/infrastructure/postgres -run TestOutboxDispatch`

  预期：失败。

- [ ] **步骤 2：实现 outbox dispatch repository**

  优先复用 `libs/kit/postgres/outbox`；如 Content schema 缺少 dispatch columns，先补 migration，不能用内存状态假装 dispatcher。

- [ ] **步骤 3：编写 RabbitMQ publisher 测试**

  覆盖 exchange、routing key、事件 envelope、payload version、publish confirm timeout 和失败错误脱敏。

  运行：`cd services/zhicore-content && go test ./internal/content/infrastructure/rabbitmq`

  预期：失败。

- [ ] **步骤 4：实现 RabbitMQ publisher**

  publisher 只负责传输；事件业务 payload 已由 application 写入 outbox。

- [ ] **步骤 5：编写 outbox dispatcher application 测试**

  覆盖 batch claim、成功 mark published、publish 失败退避、超过最大次数进入 dead、context cancel 和 shutdown 不 claim 新任务。

  运行：`cd services/zhicore-content && go test ./internal/content/application -run TestOutboxDispatcher`

  预期：失败。

- [ ] **步骤 6：实现 outbox dispatcher**

  publish RabbitMQ 不在持有 DB 行锁的事务中执行；每个事件结果必须落库。

- [ ] **步骤 7：补 admin outbox retry contract 和 handler**

  拆 `GET /api/v1/admin/content/outbox-events`、`POST /api/v1/admin/content/outbox-events/{eventId}/retry` endpoint 文档、handler contract test 和 application 查询/重试 use case。

- [ ] **步骤 8：接入 runtime**

  outbox dispatcher 按配置启用，worker descriptor、readiness 和 shutdown 都可测。

- [ ] **步骤 9：提交 outbox 切片**

  推荐拆分为 repository、publisher、dispatcher、admin API 四个提交。

## 任务 5：Content 黑盒 HTTP system test 和真实依赖验证

**测试立场：** TDD - 端到端 contract、真实依赖 wiring 和迁移闭环属于 R4。

- [ ] **步骤 1：补 testkit 依赖 fixture**

  在 `tests/testkit` 新增 PostgreSQL、MongoDB、HTTP server fixture。优先使用显式 DSN / URI；没有外部依赖时测试可 `t.Skip`，但不能伪造已验证。

- [ ] **步骤 2：编写发布闭环 system test**

  新增 `tests/system/http/content_publish_flow_test.go`，覆盖 create -> save draft -> publish -> get body；断言可信 `X-User-Id`、envelope、postVersion、body hash 和 published body blocks。

  运行：`go test ./tests/system/http -run TestContentPublishFlow`

  预期：失败。

- [ ] **步骤 3：接入可运行 Content server fixture**

  使用 runtime module 和真实 PostgreSQL / MongoDB；User/File client 可用本地 `httptest` fake provider，但必须走真实 HTTP client adapter。

- [ ] **步骤 4：补真实 MongoDB adapter 端到端验证**

  覆盖 Mongo write draft、write snapshot、read published、hash mismatch 和 context cancel。

- [ ] **步骤 5：运行系统测试**

  运行：`go test ./tests/system/http -run TestContentPublishFlow`

  预期：通过；若缺真实依赖，review 证据必须明确列为未验证。

- [ ] **步骤 6：提交 system test 切片**

  testkit 和具体 system test 可分开提交。

## 任务 6：公开文章查询和作者工作台 API

**测试立场：** TDD - pagination、可见性、作者权限、draft/published 分离和 cursor 稳定性属于 R4。

- [ ] **步骤 1：拆 endpoint contract**

  从 `content-api.md` 拆出或更新：
  - `GET /api/v1/posts`
  - `GET /api/v1/posts/{postId}`
  - `POST /api/v1/posts/batch-get`
  - `GET /api/v1/me/posts`
  - `GET /api/v1/me/drafts`
  - `GET /api/v1/posts/{postId}/draft`
  - `PATCH /api/v1/posts/{postId}/draft/meta`
  - `DELETE /api/v1/posts/{postId}/draft`

- [ ] **步骤 2：编写 application 查询测试**

  覆盖公开只读 published、作者读取自己的 draft、非作者 forbidden、deleted 不可见、cursor/page 默认值和上限。

- [ ] **步骤 3：实现 query ports 和 PostgreSQL 查询**

  列表只读 PostgreSQL metadata，不批量读取 MongoDB 正文；排序必须稳定。

- [ ] **步骤 4：编写 handler contract test**

  覆盖 path、query、身份 header、envelope、分页字段和错误码。

- [ ] **步骤 5：实现 handler**

  handler 不从 body 接受当前 actor；公开接口支持匿名读取 published。

- [ ] **步骤 6：更新 endpoint 状态并提交**

  已由 handler test 覆盖的 endpoint 标为“已验证”。公开查询和作者工作台建议拆两个提交组。

## 任务 7：发布生命周期 API

**测试立场：** TDD - 状态机、可见性、outbox、cleanup 和定时任务属于 R4。

- [ ] **步骤 1：拆 endpoint contract**

  固定：
  - `POST /api/v1/posts/{postId}/unpublish`
  - `POST /api/v1/posts/{postId}/schedule`
  - `DELETE /api/v1/posts/{postId}/schedule`
  - `DELETE /api/v1/posts/{postId}`
  - `POST /api/v1/posts/{postId}/restore`

- [ ] **步骤 2：补 domain/application 状态机测试**

  覆盖已发布撤回、删除、恢复、定时发布创建/取消、重复操作幂等或冲突语义。

- [ ] **步骤 3：实现 application 和 repository**

  状态变更必须写 outbox / visibility event；删除和恢复不得破坏 draft / published pointer 语义。

- [ ] **步骤 4：补 handler contract test 和实现**

  覆盖作者鉴权、缺登录态、非作者、已删除、未发布、重复操作和成功 envelope。

- [ ] **步骤 5：提交发布生命周期切片**

  unpublish/delete/restore 和 schedule 建议分开提交。

## 任务 8：标签、分类和话题 API

**测试立场：** TDD - 引用存在性、slug 唯一性、文章标签替换、统计和查询分页属于 R4。

- [ ] **步骤 1：拆 endpoint contract**

  固定：
  - `GET /api/v1/tags`
  - `GET /api/v1/tags/{slug}`
  - `GET /api/v1/tags/search`
  - `GET /api/v1/tags/hot`
  - `GET /api/v1/tags/{slug}/posts`
  - `GET /api/v1/posts/{postId}/tags`
  - `PUT /api/v1/posts/{postId}/tags`
  - `DELETE /api/v1/posts/{postId}/tags/{slug}`

- [ ] **步骤 2：补 schema / migration 差异检查**

  如果当前 core migration 未覆盖 category/topic/tag 所需列、索引或统计表，新增独立 migration pair 和 migration contract test。

- [ ] **步骤 3：实现 taxonomy ports / repository / application**

  分类、话题、标签引用不存在时返回 sentinel，供 HTTP 映射 `4012`。

- [ ] **步骤 4：补 handler contract test 和实现**

  覆盖公开查询、作者替换标签、重复 tag、slug 不存在、分页和错误码。

- [ ] **步骤 5：提交 taxonomy 切片**

  contract、migration、application/repository、handler 分开提交。

## 任务 9：点赞、收藏、互动状态和 reader presence

**测试立场：** TDD - 幂等写、计数一致性、unknown viewer 状态、Redis 降级和 presence no-op 属于 R4。

- [ ] **步骤 1：拆 engagement endpoint contract**

  固定：
  - `PUT /api/v1/posts/{postId}/like`
  - `DELETE /api/v1/posts/{postId}/like`
  - `PUT /api/v1/posts/{postId}/favorite`
  - `DELETE /api/v1/posts/{postId}/favorite`
  - `GET /api/v1/posts/{postId}/engagement`
  - `POST /api/v1/posts/engagement/batch-status`

- [ ] **步骤 2：实现 engagement application**

  点赞 / 收藏幂等；重复请求不重复写 delta / outbox；Redis 不可用时不能把 unknown 伪装成 `false`。

- [ ] **步骤 3：实现 engagement repository 和缓存 adapter**

  PostgreSQL 是事实源，Redis 只是缓存和受控 fallback 协调。

- [ ] **步骤 4：补 handler contract test 和实现**

  覆盖 `liked/favorited=true/false/null`、`degraded=true`、登录态、匿名读取和错误码。

- [ ] **步骤 5：拆 reader presence endpoint contract**

  固定：
  - `PUT /api/v1/posts/{postId}/reader-sessions/{sessionId}`
  - `DELETE /api/v1/posts/{postId}/reader-sessions/{sessionId}`
  - `GET /api/v1/posts/{postId}/reader-presence`

- [ ] **步骤 6：实现 presence application / Redis adapter / handler**

  Presence 是附加能力；Redis 不可用时按 `rate-limiting.md` 返回空成功或 degraded 摘要，不能影响文章详情和正文读取。

- [ ] **步骤 7：提交互动和 presence 切片**

  like/favorite、engagement query、presence 建议分开提交。

## 任务 10：管理端 Content API

**测试立场：** TDD - admin 鉴权、审计字段、查询过滤、删除可见性和 outbox retry 属于 R4。

- [ ] **步骤 1：拆 admin endpoint contract**

  固定：
  - `GET /api/v1/admin/content/posts`
  - `DELETE /api/v1/admin/content/posts/{postId}`
  - `GET /api/v1/admin/content/outbox-events`
  - `POST /api/v1/admin/content/outbox-events/{eventId}/retry`

- [ ] **步骤 2：编写 admin application 测试**

  覆盖缺少 admin role、查询过滤、管理删除、重复删除、outbox retry 冷却窗口和审计字段。

- [ ] **步骤 3：实现 admin application / repository**

  Admin 删除文章必须写 visibility event；outbox retry 不能绕过 rate limit 和状态条件。

- [ ] **步骤 4：补 handler contract test 和实现**

  使用 `X-User-Id` + `X-User-Roles`；客户端伪造 body actor 不生效。

- [ ] **步骤 5：提交 admin 切片**

  posts 管理和 outbox 管理分开提交。

## 任务 11：业务限流、resilience policy 和观测

**测试立场：** TDD - 高副作用写路径 fail-closed、降级语义、metrics/log 字段和 policy owner 属于 R4。

- [ ] **步骤 1：定义 rate limiter port 和 outcome**

  Outcome 至少覆盖 `ALLOW`、`REJECT_TOO_FREQUENT`、`DEGRADED_ALLOW_LOCAL`、`DEGRADED_DENY_UNAVAILABLE`、`NOOP_SUCCESS`。

- [ ] **步骤 2：补 application 限流测试**

  覆盖草稿保存、发布、互动写、presence、admin retry 和内部 body read 的 fail-open / fail-closed / no-op 分支。

- [ ] **步骤 3：实现 Redis rate limit adapter**

  Redis adapter 只返回 typed outcome；application 选择业务降级，不由 adapter 构造 HTTP response。

- [ ] **步骤 4：接入 runtime resilience policy**

  为 postgres、mongo、redis、user-service、file-service、rabbitmq 的 provider + operation 固定 timeout、retry、breaker key、max-in-flight 配置和默认值。

- [ ] **步骤 5：补观测测试或结构检查**

  覆盖关键日志字段、operation 名称、错误脱敏和 worker result counters。若 metrics kit 尚未存在，先以明确接口和测试 fake 固定调用点，不引入无 owner 的全局 metrics helper。

- [ ] **步骤 6：更新 docs 和 configs**

  同步 `runtime-resilience.md`、`rate-limiting.md` 或服务 README 中“已落地 / 待落地”状态，避免设计文档宣称代码已实现。

- [ ] **步骤 7：提交限流和 resilience 切片**

  rate limiter、policy config、observability 分开提交。

## 任务 12：最终验证、review 证据和完成收口

**测试立场：** 验证门禁切片。

- [ ] **步骤 1：运行服务内测试**

  运行：`cd services/zhicore-content && go test ./...`

  预期：通过。

- [ ] **步骤 2：运行系统测试**

  运行：`go test ./tests/system/http -run TestContent`

  预期：有真实依赖时通过；若跳过，review 证据必须写清楚跳过原因。

- [ ] **步骤 3：运行测试规模检查**

  运行：`python3 scripts/check-test-size.py --files services/zhicore-content tests/system/http tests/testkit`

  预期：通过。

- [ ] **步骤 4：运行结构检查**

  运行：`bash scripts/check-structure.sh`

  预期：`structure ok`。

- [ ] **步骤 5：运行最终 diff 检查**

  运行：`git diff --check`

  预期：无 whitespace error。

- [ ] **步骤 6：请求独立 review**

  对完整 diff、计划 checkbox、验证证据和残余风险做代码 review。若有 finding，先用 @receiving-code-review 判断是否有效，再按最小正确修复。

- [ ] **步骤 7：记录 review 证据**

  新增 `docs/reviews/backend/<date>-content-module-completion.md`，记录范围、提交、验证命令、review finding、残余风险和未验证外部依赖。

- [ ] **步骤 8：最终提交 review 证据**

  提交前使用 @committing-changes。review 证据必须独立提交，不和业务代码混在一起。

## 架构适配评估

- 本计划继续保持 Content 的服务内边界：HTTP contract 归 `services/zhicore-content/api/http`，业务规则归 application/domain，PostgreSQL/MongoDB/RabbitMQ/Redis 归 infrastructure，进程和 worker lifecycle 归 runtime/cmd。
- Worker、outbox dispatcher、rate limiter 和 client adapter 都有明确 owner，避免在 handler 或 repository 中散落运行期策略。
- API family 按公开查询、作者工作台、发布生命周期、taxonomy、engagement/presence、admin 分批实现，每批都有 contract、handler test、application/repository 测试和独立提交边界。
- 系统测试在发布闭环上先补最小黑盒场景，后续 API family 可逐步加入 system test，不要求一次性构造全量生产环境。
- 结构性收敛没有被静默推迟：可运行 runtime、worker、错误 sentinel、system test、限流和观测都作为独立任务，有明确完成标准。

## 风险和取舍

- 本计划范围很大，执行时必须按任务拆分 worktree 或阶段分支；不要在一个长分支里积累所有 API family。
- User/File/RabbitMQ/Redis 的 Go 服务或 contract 若尚未完全可用，Content 只能通过 typed client contract 和本地 fake provider 做测试，不能伪造生产 readiness。
- 如果迁移发现当前 `outbox_events` 或 task 表缺少 dispatch 状态字段，应新增独立 migration，而不是在 worker 中用内存状态绕过。
- Engagement 和 presence 引入 Redis 后，降级语义必须严格按 `rate-limiting.md` 和 `engagement-design.md`，不能为了简化前端把 unknown 写成 `false`。
- Admin API 和 outbox retry 是高风险操作，必须有权限、限流和审计字段测试后再实现。
