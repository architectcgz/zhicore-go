# Admin 管理审核 Facade 基础实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；涉及 provider contract、migration、repository、application 编排、handler contract 和 runtime 的步骤按 @test-driven-development 执行。每个 checkbox 达到预期后立即更新；如需提交，提交前必须先使用 @committing-changes。

**目标：** 把 `zhicore-admin` 从占位模块推进到拥有举报查询 / 处理、管理端 facade、审计日志和可运行 runtime 的首个可交付切片。

**架构：** Admin 只拥有 `reports`、`audit_logs` 和管理编排记录；禁用账号、删除文章、删除评论等真实业务 mutation 必须委托 Auth、Content、Comment 等归属服务。Admin 本地 application 负责权限、事务、审计和 provider 调用结果编排，handler 只做协议绑定和错误映射。

**技术栈：** Go 1.26、Gin、PostgreSQL、provider-owned `libs/contracts/clients/*`、Admin HTTP schema、服务本地 runtime。

---

## 背景依据

- `docs/architecture/services/admin/README.md`
- `services/zhicore-admin/api/http/README.md`
- `docs/architecture/service-boundaries.md`
- `docs/migration/service-migration-workflow.md`
- `docs/contracts/http-schema-template.md`
- `docs/architecture/security.md`
- `docs/architecture/testing.md`
- 需要核对既有行为时读取 `../zhicore-microservice/zhicore-admin/src/main/java/com/zhicore/admin/interfaces/controller/*.java`

## 当前基线

- 生产 Go 源码只有 `services/zhicore-admin/internal/admin/doc.go`。
- `services/zhicore-admin/api/http/README.md` 仍是计划化占位，没有 `endpoints/`。
- 当前 Go 占位把举报处理写成 `resolve`，但既有 Java controller 是 `/admin/reports/{reportId}/handle`，还存在 `/admin/reports/pending` 和 `/admin/users/{userId}/enable`。
- `libs/contracts/clients/auth`、`content`、`user` 只有 README，`comment` contract 目录还不存在。

## 不可并行修改文件

- `libs/contracts/clients/auth/contract.go`：必须等 `2026-07-06-gateway-routing-auth-foundation-implementation-plan.md` 任务 2 合并后再追加 Admin disable / enable contract。
- `libs/contracts/clients/content/contract.go`：必须等 `2026-07-06-ranking-ledger-hot-posts-foundation-implementation-plan.md` 任务 1 合并后再追加 Admin 管理端查询 / 删除 contract。
- `libs/contracts/clients/user/contract.go`：必须等 Message 计划任务 1 的 User guard contract 合并后再追加 Admin 用户列表 / 摘要查询 contract。
- `libs/contracts/clients/comment/contract.go`：由本计划任务 2 首次创建；其他计划不得并行创建同名文件。

## 文件结构

- 修改：`docs/architecture/services/admin/README.md`
- 修改：`services/zhicore-admin/README.md`
- 修改：`services/zhicore-admin/api/http/README.md`
- 新增：`services/zhicore-admin/api/http/endpoints/*.md`
- 修改 / 追加：`libs/contracts/clients/auth/contract.go`
- 修改 / 追加：`libs/contracts/clients/content/contract.go`
- 修改 / 追加：`libs/contracts/clients/user/contract.go`
- 新增：`libs/contracts/clients/comment/contract.go`
- 新增：`services/zhicore-admin/migrations/*_create_admin_core_tables.up.sql`
- 新增：`services/zhicore-admin/migrations/*_create_admin_core_tables.down.sql`
- 新增：`services/zhicore-admin/internal/admin/{domain,application,ports,infrastructure,runtime}/**`
- 新增：`services/zhicore-admin/api/http/*.go`
- 新增：`services/zhicore-admin/cmd/server/*.go`
- 新增：`services/zhicore-admin/configs/local.example.env`

## 任务 1：Contract 纠偏与 endpoint 拆单

**测试立场：** R0 文档切片；不改运行行为。

**文件：**
- 修改：`docs/architecture/services/admin/README.md`
- 修改：`services/zhicore-admin/README.md`
- 修改：`services/zhicore-admin/api/http/README.md`
- 新增：`services/zhicore-admin/api/http/endpoints/list-users.md`
- 新增：`services/zhicore-admin/api/http/endpoints/disable-user.md`
- 新增：`services/zhicore-admin/api/http/endpoints/enable-user.md`
- 新增：`services/zhicore-admin/api/http/endpoints/list-posts.md`
- 新增：`services/zhicore-admin/api/http/endpoints/delete-post.md`
- 新增：`services/zhicore-admin/api/http/endpoints/list-comments.md`
- 新增：`services/zhicore-admin/api/http/endpoints/delete-comment.md`
- 新增：`services/zhicore-admin/api/http/endpoints/list-pending-reports.md`
- 新增：`services/zhicore-admin/api/http/endpoints/list-reports.md`
- 新增：`services/zhicore-admin/api/http/endpoints/handle-report.md`

**验收清单：**
- [ ] 举报处理路径固定为 `POST /admin/reports/{reportId}/handle`，不使用 `resolve`。
- [ ] 补齐 `GET /admin/reports/pending` 和 `POST /admin/users/{userId}/enable`。
- [ ] Admin 服务内 path 保持历史管理端路径，不强行加 `/api/v1`。
- [ ] 每个 endpoint 写清 request 字段、response `data`、公开错误码、管理员权限、是否 facade、是否依赖 provider contract。
- [ ] `delete-comment` 明确当前 provider contract 需要 `postId + commentId + reason`；如果外部 path 只有 `commentId`，本计划后续切片必须先解决定位问题。

- [ ] **步骤 1：读取 Java controller / DTO 参考并记录 path 差异**
- [ ] **步骤 2：补 endpoint 文档和 API 索引**
- [ ] **步骤 3：运行结构和空白验证**

运行：`bash scripts/check-structure.sh && git diff --check`

预期：结构检查通过，无 whitespace error。

## 任务 2：Provider-owned contract 补齐

**测试立场：** TDD - 跨服务 typed client contract 属于 R4。

**文件：**
- 修改 / 追加：`libs/contracts/clients/auth/contract.go`
- 修改 / 追加：`libs/contracts/clients/auth/contract_test.go`
- 修改：`libs/contracts/clients/auth/README.md`
- 修改 / 追加：`libs/contracts/clients/content/contract.go`
- 修改 / 追加：`libs/contracts/clients/content/contract_test.go`
- 修改：`libs/contracts/clients/content/README.md`
- 修改 / 追加：`libs/contracts/clients/user/contract.go`
- 修改 / 追加：`libs/contracts/clients/user/contract_test.go`
- 修改：`libs/contracts/clients/user/README.md`
- 新增：`libs/contracts/clients/comment/contract.go`
- 新增：`libs/contracts/clients/comment/contract_test.go`
- 新增：`services/zhicore-admin/internal/admin/ports/clients.go`

**验收清单：**
- [ ] Auth contract 明确账号禁用、启用、角色 / 状态查询 owner，Admin 不直接修改 Auth 表。
- [ ] 执行前确认 Gateway 计划任务 2 已合并；未合并前本任务只能编辑 Admin 本地 ports，不得触碰 `libs/contracts/clients/auth/contract.go`。
- [ ] Content contract 明确管理端文章列表、删除文章 command、失败错误分类和 caller operation。
- [ ] 执行前确认 Ranking 计划任务 1 已合并；未合并前本任务只能在计划中登记 Content contract 待追加，不得触碰 `libs/contracts/clients/content/contract.go`。
- [ ] User contract 明确 `/admin/users` 所需用户列表、状态、角色、最近登录 / 创建时间、分页、搜索字段、失败错误分类和 caller operation；Admin 不复制 User profile 表。
- [ ] 执行前确认 Message 计划任务 1 已合并；未合并前不得并行编辑 `libs/contracts/clients/user/contract.go` 或 `services/zhicore-user/api/http/internal_handlers.go`。
- [ ] Comment contract 明确删除评论需要 `postId`、`commentId`、`reason` 和 operator；Admin 不裸写 Comment URL。
- [ ] 所有 provider contract 固定 `X-Caller-Service=zhicore-admin` 和 `X-Caller-Operation` 语义。
- [ ] Admin 本地 `ports` 只依赖 consumer-side interface，不暴露 provider HTTP DTO 或 SDK 类型。

- [ ] **步骤 1：先写 contract test**
- [ ] **步骤 2：运行 test 确认失败**

运行：`cd libs/contracts && go test ./clients/auth ./clients/content ./clients/user ./clients/comment -count=1`

预期：新增 contract 尚未实现导致失败。

- [ ] **步骤 3：实现最小 typed contract**
- [ ] **步骤 4：运行 contract test**

运行：`cd libs/contracts && go test ./clients/auth ./clients/content ./clients/user ./clients/comment -count=1`

预期：通过。

## 任务 3：reports / audit_logs 持久化基础

**测试立场：** TDD - migration、repository 和审计字段属于 R4。

**文件：**
- 新增：`services/zhicore-admin/migrations/20260706xxxx_create_admin_core_tables.up.sql`
- 新增：`services/zhicore-admin/migrations/20260706xxxx_create_admin_core_tables.down.sql`
- 新增：`services/zhicore-admin/migrations/migration_contract_test.go`
- 新增：`services/zhicore-admin/internal/admin/domain/report.go`
- 新增：`services/zhicore-admin/internal/admin/domain/audit_log.go`
- 新增：`services/zhicore-admin/internal/admin/domain/errors.go`
- 新增：`services/zhicore-admin/internal/admin/ports/repositories.go`
- 新增：`services/zhicore-admin/internal/admin/infrastructure/postgres/*.go`

**验收清单：**
- [ ] `reports` 支持 Go 可用资源标识，不直接照抄 Java `BIGINT target_id` 作为唯一目标字段。
- [ ] `reports` 至少包含 `status`、`target_type`、`target_public_id`、`target_internal_id`、`parent_target_id`、`post_id`、`reason`、`reporter_id`、`handler_id`、`handle_action`、`handle_remark`、`handled_at`。
- [ ] `target_type=comment` 时必须能通过 `target_public_id + post_id` 或 `target_public_id + parent_target_id` 定位 Comment；缺少定位字段时返回参数错误，不调用 provider、不写成功审计。
- [ ] `audit_logs` 至少包含 `operator_id`、`operation`、`target_type`、`target_public_id`、`target_internal_id`、`result`、`reason`、`request_id`、`trace_id`、`created_at`。
- [ ] repository 把 duplicate / not found / constraint error 翻译为 Admin 本地语义错误。
- [ ] migration up/down 可通过 `golang-migrate` 往返验证；文件名中的 `20260706xxxx` 必须在实施时替换为真实单调递增时间戳。

- [ ] **步骤 1：写 migration / repository 失败测试**
- [ ] **步骤 2：实现 migration 和 repository**
- [ ] **步骤 3：运行持久化验证**

运行：`cd services/zhicore-admin && go test ./migrations ./internal/admin/infrastructure/postgres -count=1`

预期：通过。

## 任务 4：举报查询与处理编排闭环

**测试立场：** TDD - provider 编排、事务和审计属于 R4。

**文件：**
- 新增：`services/zhicore-admin/internal/admin/application/service.go`
- 新增：`services/zhicore-admin/internal/admin/application/report_queries.go`
- 新增：`services/zhicore-admin/internal/admin/application/report_commands.go`
- 新增：`services/zhicore-admin/internal/admin/application/*_test.go`
- 新增：`services/zhicore-admin/internal/admin/infrastructure/clients/auth_client.go`
- 新增：`services/zhicore-admin/internal/admin/infrastructure/clients/content_client.go`
- 新增：`services/zhicore-admin/internal/admin/infrastructure/clients/comment_client.go`
- 新增：`services/zhicore-admin/api/http/handler.go`
- 新增：`services/zhicore-admin/api/http/errors.go`
- 新增：`services/zhicore-admin/api/http/payloads.go`
- 新增：`services/zhicore-admin/api/http/request_helpers.go`
- 新增：`services/zhicore-admin/api/http/reports_handlers.go`
- 新增：`services/zhicore-admin/api/http/reports_handler_test.go`

**验收清单：**
- [ ] `GET /admin/reports/pending`、`GET /admin/reports?status=`、`POST /admin/reports/{reportId}/handle` 都有 handler contract test。
- [ ] `IGNORE` 只更新本地 report 状态并写审计。
- [ ] `DELETE_CONTENT` 按 `targetType=post/comment` 委托 Content 或 Comment。
- [ ] `DELETE_CONTENT targetType=comment` 缺 `postId` / 父目标定位字段时返回参数错误，report 保持 `PENDING`，不得写成功审计。
- [ ] `BAN_USER` 委托 Auth，不在 Admin 复制账号状态。
- [ ] 任一下游 mutation 失败时 report 不能从 `PENDING` 变成已处理，不能写成功审计。
- [ ] application 注释说明“委托归属服务 + 本地审计”的业务规则，避免后续维护者把 Admin 做成 owner。

- [ ] **步骤 1：写 application 编排失败测试**
- [ ] **步骤 2：写 handler contract 失败测试**
- [ ] **步骤 3：实现 application、client adapter 和 handler**
- [ ] **步骤 4：运行 Admin 行为测试**

运行：`cd services/zhicore-admin && go test ./internal/admin/application ./api/http -count=1`

预期：通过。

## 任务 5：Users / Posts / Comments facade 与 runtime

**测试立场：** TDD - 新 endpoint、runtime 配置和 health 属于 R3/R4。

**文件：**
- 新增：`services/zhicore-admin/api/http/users_handlers.go`
- 新增：`services/zhicore-admin/api/http/posts_handlers.go`
- 新增：`services/zhicore-admin/api/http/comments_handlers.go`
- 新增：`services/zhicore-admin/api/http/*_handler_test.go`
- 新增：`services/zhicore-admin/internal/admin/runtime/module.go`
- 新增：`services/zhicore-admin/internal/admin/runtime/module_test.go`
- 新增：`services/zhicore-admin/cmd/server/main.go`
- 新增：`services/zhicore-admin/cmd/server/config.go`
- 新增：`services/zhicore-admin/cmd/server/config_test.go`
- 新增：`services/zhicore-admin/configs/local.example.env`

**验收清单：**
- [ ] `/admin/users`、`/admin/users/{userId}/disable`、`/admin/users/{userId}/enable` 有 handler contract test。
- [ ] `/admin/users` 通过 User provider 查询用户列表 / 摘要，Auth provider 只负责账号状态 mutation 或认证状态查询；测试覆盖 User provider degraded、分页默认值和搜索条件。
- [ ] `/admin/posts`、`/admin/posts/{postId}` 有 handler contract test。
- [ ] `/admin/comments/{commentId}` 只有在 provider contract 已解决 `postId` 定位后才实现；否则 endpoint 文档标为待确认并延期。
- [ ] `/health/live` 只表示进程可响应；`/health/ready` 检查 PostgreSQL 和必需 provider client 配置。
- [ ] `cmd/server` 只负责加载配置、打开依赖、调用 runtime、启动和 shutdown，不执行 migration。

- [ ] **步骤 1：写 facade handler 和 runtime 失败测试**
- [ ] **步骤 2：实现 handler、runtime 和配置**
- [ ] **步骤 3：运行服务内验证**

运行：`cd services/zhicore-admin && go test ./... -count=1`

预期：通过。

## 集成验证

- [ ] 运行 `cd libs/contracts && go test ./... -count=1`。
- [ ] 运行 `cd services/zhicore-admin && go test ./... -count=1`。
- [ ] 有可用 `ZHICORE_ADMIN_POSTGRES_DSN` 时运行 `migrate -path services/zhicore-admin/migrations -database "$ZHICORE_ADMIN_POSTGRES_DSN" up && migrate -path services/zhicore-admin/migrations -database "$ZHICORE_ADMIN_POSTGRES_DSN" down 1 && migrate -path services/zhicore-admin/migrations -database "$ZHICORE_ADMIN_POSTGRES_DSN" up`；没有 DSN 时必须用隔离 PostgreSQL 容器执行同等 `up -> down 1 -> up`，或在交付说明中列为未验证的外部依赖。
- [ ] 运行 `make test-size`。
- [ ] 运行 `bash scripts/check-structure.sh`。
- [ ] 完整执行本计划或触达共享 contract / runtime / migration 后，交付前运行 `make check`。
