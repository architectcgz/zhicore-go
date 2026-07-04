# Auth 认证基础实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；本计划步骤使用 checkbox 追踪。提交前必须先使用 @committing-changes。

**目标：** 在 `zhicore-auth` 中实现注册、登录、refresh、logout、当前主体、CSRF、session list、session revoke 和 security operation 查询的最小可测基础。

**架构：** Auth 拥有账号、凭证、角色、refresh session、JWT 签发和安全撤销投影。HTTP handler 只做协议绑定，application 拥有事务和安全语义，repository / Redis / token signer 通过 ports 注入。

**技术栈：** Go 1.22、标准库 `net/http`、PostgreSQL SQL migration、Redis 投影端口、`libs/kit/httpapi`。

---

## 背景依据

- `docs/architecture/module/auth/README.md`
- `docs/architecture/module/auth/service.md`
- `docs/architecture/module/auth/data-model.md`
- `docs/architecture/module/auth/redis-keys.md`
- `services/zhicore-auth/api/http/README.md`
- `docs/migration/service-migration-workflow.md`

## 文件结构

- 新增：`services/zhicore-auth/internal/auth/domain/account.go`
- 新增：`services/zhicore-auth/internal/auth/domain/session.go`
- 新增：`services/zhicore-auth/internal/auth/domain/errors.go`
- 新增：`services/zhicore-auth/internal/auth/ports/repositories.go`
- 新增：`services/zhicore-auth/internal/auth/ports/security.go`
- 新增：`services/zhicore-auth/internal/auth/ports/outbox.go`
- 新增：`services/zhicore-auth/internal/auth/application/service.go`
- 新增：`services/zhicore-auth/internal/auth/application/register_login_test.go`
- 新增：`services/zhicore-auth/api/http/handler.go`
- 新增：`services/zhicore-auth/api/http/auth_handler_test.go`
- 新增：`services/zhicore-auth/migrations/<timestamp>_create_auth_core_tables.up.sql`
- 新增：`services/zhicore-auth/migrations/<timestamp>_create_auth_core_tables.down.sql`
- 新增：`services/zhicore-auth/internal/auth/runtime/module.go`
- 新增：`services/zhicore-auth/cmd/server/main.go`

## 任务 1：domain、ports 和 application 骨架

**测试立场：** TDD - 账号、凭证、token、session 和 outbox 是核心行为。

- [x] **步骤 1：编写 application 失败测试**

  覆盖：

  - `RegisterAccount` 先创建 `PENDING_PROFILE` account / credential，再同步调用 User `CreateProfileForAccount`，最后激活账号、授予默认 `ROLE_USER` 并写 `auth.account.registered`
  - `Login` 校验密码后，先校验 refresh material contract（`sessionId/tokenId/plaintext/hash` 非空且 `expiresAt > now`），再创建 refresh session metadata，最后签 access token
  - 账号不存在、凭证不存在、密码错误都映射 `ErrInvalidCredentials`
  - 禁用、封禁、锁定账号不能登录

  运行：`cd services/zhicore-auth && go test ./internal/auth/application -run 'TestRegisterAccount|TestLogin'`

  预期：失败，因为 application 尚不存在。

- [x] **步骤 2：补 domain 和 module-local 错误**

  实现 `AccountID`、`UserID`、`Email`、`AccountStatus`、`RoleName`、`RefreshSession`、`SecurityOperation` 和 `ErrInvalidCredentials` 等错误。

- [x] **步骤 3：定义 ports**

  定义 `AccountRepository`、`CredentialRepository`、`RoleRepository`、`RefreshTokenMaterialIssuer`、`RefreshSessionStore`、`TokenIssuer`、`PasswordHasher`、`AccessTokenBlacklist`、`AuthCacheStore`、`TransactionRunner`、`OutboxPublisher`、`UserProfileClient`、`Clock`、`RateLimiter`。`RefreshTokenMaterialIssuer` 先生成 `sessionId/tokenId/plaintext/hash/expiresAt`，application 在落库前校验 contract；`RefreshSessionStore` 只持久化已确定 metadata，`TokenIssuer` 只签 access token；`RateLimiter` 使用 typed outcome，避免后续 HTTP 无法区分 reject / degraded / unavailable，并且作为 register/login 的首批安全依赖在构造期必填。`AccessTokenBlacklist` 和 `AuthCacheStore` 在本计划后续 logout / revoke / Gateway projection 步骤继续使用，不是无主端口，但 task 1 不强制 no-op 接线。

- [x] **步骤 4：实现 application 最小行为**

  `RegisterAccount` 采用“两段本地事务 + 同步 User client”：

  - 事务 A：创建或复用未过期 `PENDING_PROFILE` account / credential。
  - 同步调用 User `CreateProfileForAccount` 获取 `userId`。
  - 事务 B：重新获取 `activatedAt`，写 `auth_accounts.user_id`、切 `ACTIVE`、授予默认 `ROLE_USER`、写 `auth.account.registered` outbox。

  关键注释说明同步调用原因：避免返回注册成功但 User profile 未闭合，导致不能登录或资料接口 404。`auth.account.registered` 只表达 ACTIVE 且已有 `userId` 的账号事实；User 返回零 `userId` 视为 contract violation，不进入事务 B；`ACTIVE` 但缺失 `userId` 的账号也必须拒绝登录。

- [x] **步骤 5：运行 application 测试**

  运行：`cd services/zhicore-auth && go test ./internal/auth/application`

  预期：通过。

## 任务 2：HTTP handler 和 contract test

**测试立场：** TDD - cookie、CSRF、refresh rotation、session revoke 和 `202 PROCESSING` 是安全 contract。

- [x] **步骤 1：编写 handler 失败测试**

  覆盖 `register`、`login`、`refresh`、`logout`、`me`、`csrf`、`sessions`、`revoke-session`、`security-operation` 的成功和核心错误码。

  运行：`cd services/zhicore-auth && go test ./api/http`

  预期：失败，因为 handler 尚不存在。

- [x] **步骤 2：实现 handler DTO 和路由**

  使用标准库 `http.ServeMux`。handler 不直接访问 DB、Redis 或 token signer。

- [x] **步骤 3：实现错误映射**

  使用 `libs/kit/httpapi.WriteErrorCode` 写入 Auth 业务错误码；Redis 投影未确认时返回 `202` 和 `operationId`。

- [x] **步骤 4：更新 HTTP schema 状态**

  已被 handler contract test 覆盖的 endpoint 标记为“已验证”。

- [x] **步骤 5：运行 Auth 测试**

  运行：`cd services/zhicore-auth && go test ./api/http ./internal/auth/...`

  预期：通过。

## 任务 3：migration 和 runtime 最小入口

**测试立场：** TDD / 手动验证 - schema 和进程入口影响部署与替换。

- [x] **步骤 1：编写 Auth 核心 migration**

  首批表至少包含 `auth_accounts`、`auth_password_credentials`、`auth_account_roles`、`auth_refresh_sessions`、`auth_used_refresh_tokens`、`auth_security_operations`、`auth_audit_logs`、`auth_outbox_events`。

- [x] **步骤 2：补 runtime module 和 `cmd/server`**

  `cmd/server/main.go` 只负责进程入口和运行时装配；业务 wiring 放 `internal/auth/runtime/module.go`。

- [ ] **步骤 3：验证 migration**

  有本地数据库时验证 `up` 和最近一条 `down 1`。无数据库时记录未运行原因，不声称通过。

  未运行：本机已安装 `migrate` CLI，但当前环境缺少 `ZHICORE_AUTH_POSTGRES_DSN`，因此没有执行 `up` / `down 1`。

- [x] **步骤 4：运行收口测试**

  运行：`cd services/zhicore-auth && go test ./...`

  预期：通过。

## 架构适配评估

- Auth 只拥有认证事实，不复制 User profile。
- 注册链路同步闭合 User profile，再发布 `auth.account.registered`。
- Handler、application、domain、ports、infrastructure 的依赖方向清晰。
