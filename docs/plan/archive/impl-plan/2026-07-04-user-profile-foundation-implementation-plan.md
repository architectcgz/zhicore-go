# User Profile 基础实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；本计划步骤使用 checkbox 追踪。提交前必须先使用 @committing-changes。

**目标：** 在 `zhicore-user` 中实现 Profile 初始化、查询、更新、状态变更和首批 HTTP Profile endpoints。

**架构：** User 拥有 profile、`publicId`、`nickname`、头像文件引用、简介、陌生人私信设置和 User 业务状态。Auth 只拥有账号和 token；File 只拥有文件事实，User 只保存 `avatarFileId`。

**技术栈：** Go 1.22、标准库 `net/http`、PostgreSQL SQL migration、`libs/kit/httpapi`。

---

## 背景依据

- `docs/architecture/module/user/README.md`
- `docs/architecture/module/user/service.md`
- `docs/architecture/module/user/domain.md`
- `docs/architecture/module/user/data-events.md`
- `services/zhicore-user/api/http/README.md`

## 文件结构

- 新增：`services/zhicore-user/internal/user/domain/profile.go`
- 新增：`services/zhicore-user/internal/user/domain/errors.go`
- 新增：`services/zhicore-user/internal/user/ports/profile.go`
- 新增：`services/zhicore-user/internal/user/application/service.go`
- 新增：`services/zhicore-user/internal/user/application/profile_test.go`
- 新增：`services/zhicore-user/api/http/handler.go`
- 新增：`services/zhicore-user/api/http/profile_handler_test.go`
- 修改：`services/zhicore-user/api/http/README.md`
- 修改：`services/zhicore-user/api/http/endpoints/get-me.md`
- 修改：`services/zhicore-user/api/http/endpoints/get-profile.md`
- 修改：`services/zhicore-user/api/http/endpoints/update-profile.md`
- 新增：`services/zhicore-user/migrations/<timestamp>_create_user_profile_tables.up.sql`
- 新增：`services/zhicore-user/migrations/<timestamp>_create_user_profile_tables.down.sql`

## 任务 1：Profile application

**测试立场：** TDD - profile 初始化、nickname 唯一、状态和 outbox 是核心行为。

- [x] **步骤 1：编写失败测试**

  覆盖 `CreateProfileForAccount` 幂等、默认 nickname 冲突、`GetMyProfile`、`GetUserProfileByPublicId`、`UpdateProfile`、`DeactivateUserProfile`、`MarkUserDeleted`、`RestoreDeletedUserProfile`。

  运行：`cd services/zhicore-user && go test ./internal/user/application -run 'TestCreateProfile|TestUpdateProfile|TestUserStatus'`

  预期：失败，因为 application 尚不存在。

- [x] **步骤 2：实现 Profile domain 和 policy**

  固定 `UserStatus` 为 `ACTIVE`、`DEACTIVATED`、`DELETED`；实现 nickname trim、长度、危险字符和 bio 校验。

- [x] **步骤 3：定义 Profile ports**

  定义 `ProfileRepository`、`ProfileQueryRepository`、`FileReferenceClient`、`PublicIDGenerator`、`OutboxPublisher`、`TransactionRunner`、`Clock`、`CacheStore`。

- [x] **步骤 4：实现 Profile application**

  `UpdateProfile` 在事务前校验非空头像文件；公开资料字段变化才递增 `profileVersion` 并发布 `user.profile.updated`。

- [x] **步骤 5：运行 application 测试**

  运行：`cd services/zhicore-user && go test ./internal/user/application`

  预期：通过。

## 任务 2：Profile HTTP endpoints

**测试立场：** TDD - HTTP contract、可信身份 header、错误码和 avatar URL 派生必须锁定。

- [x] **步骤 1：编写 handler 失败测试**

  覆盖：

  - `GET /api/v1/users/me`
  - `GET /api/v1/users/{publicId}`
  - `PATCH /api/v1/users/me/profile`
  - 缺 `X-User-Id` 返回 `2006`
  - nickname 冲突返回 `3005`
  - avatar URL 解析失败时不伪造 URL

  运行：`cd services/zhicore-user && go test ./api/http -run TestProfile`

  预期：失败，因为 handler 尚不存在。

- [x] **步骤 2：实现 handler**

  当前操作者只来自 Gateway 注入的 `X-User-Id`；request body 中的操作者字段不得覆盖可信身份。

- [x] **步骤 3：更新 endpoint 状态**

  被 handler contract test 覆盖的 Profile endpoint 标记为“已验证”。

- [x] **步骤 4：运行 User Profile 测试**

  运行：`cd services/zhicore-user && go test ./api/http ./internal/user/...`

  预期：通过。

## 任务 3：Profile migration 和 runtime

**测试立场：** TDD / 手动验证 - schema 是服务事实源。

- [x] **步骤 1：编写 Profile migration**

  至少创建 `users` 和 User 本地 `outbox_events`；`users.public_id`、`users.account_id`、`users.nickname` 需要唯一约束。

- [x] **步骤 2：补 runtime module 和 `cmd/server`**

  `cmd/server/main.go` 只负责进程入口；业务 wiring 放 `internal/user/runtime/module.go`。

- [x] **步骤 3：运行收口测试**

  运行：`cd services/zhicore-user && go test ./...`

  预期：通过。

  当前状态：`runtime.Module` 支持依赖注入组装测试，`cmd/server/main.go` 已作为进程根落地，并在生产 repository、File client、outbox dispatcher、cache 和配置加载尚未落地时 fail fast，避免伪装成可运行服务。已使用隔离临时 PostgreSQL 容器和 `migrate/migrate:v4.18.3` 执行真实 `up -> down 1 -> up`，最终 `schema_migrations` 为 `20260704093000 dirty=false`，关键表 `users`、`outbox_events` 和唯一约束已确认存在。

## 架构适配评估

- User 不保存 Auth password、roles、token 或 account ban 状态。
- `avatarUrl` 只作为读取时展示增强，不落库、不进事件。
- Profile 状态先于关系切片，满足 Block / Follow 的 `ACTIVE` guard 依赖。
