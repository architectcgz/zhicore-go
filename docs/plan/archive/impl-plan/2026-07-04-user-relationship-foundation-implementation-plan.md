# User Relationship 基础实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；本计划步骤使用 checkbox 追踪。提交前必须先使用 @committing-changes。

**目标：** 在 `zhicore-user` 中实现 Block / Follow 关系、统计、事件和对应 HTTP endpoints。

**架构：** User 拥有关注、拉黑、关系统计和互动 guard。前端 path 使用 `publicId`，服务间关系判断使用内部 `userId`。

**技术栈：** Go 1.22、标准库 `net/http`、PostgreSQL、User outbox、`libs/kit/httpapi`。

---

## 背景依据

- `docs/architecture/module/user/service.md`
- `docs/architecture/module/user/domain.md`
- `docs/architecture/module/user/data-events.md`
- `docs/architecture/module/user/rate-limiting.md`

## 文件结构

- 新增：`services/zhicore-user/internal/user/domain/relationship.go`
- 新增：`services/zhicore-user/internal/user/application/relationship_test.go`
- 修改：`services/zhicore-user/internal/user/application/service.go`
- 新增：`services/zhicore-user/api/http/relationship_handler_test.go`
- 新增：`services/zhicore-user/api/http/endpoints/block-user.md`
- 新增：`services/zhicore-user/api/http/endpoints/unblock-user.md`
- 新增：`services/zhicore-user/api/http/endpoints/list-blocked-users.md`
- 新增：`services/zhicore-user/api/http/endpoints/follow-user.md`
- 新增：`services/zhicore-user/api/http/endpoints/unfollow-user.md`
- 新增：`services/zhicore-user/api/http/endpoints/list-followers.md`
- 新增：`services/zhicore-user/api/http/endpoints/list-following.md`
- 修改：`services/zhicore-user/api/http/README.md`
- 修改：`services/zhicore-user/migrations/<timestamp>_create_user_profile_tables.up.sql` 或新增关系 migration

## 任务 1：关系 contract

**测试立场：** R0 文档切片，但后续 handler 必须按 contract 测试。

- [x] **步骤 1：补 Block endpoints schema**

  固定 `POST /api/v1/users/{publicId}/block`、`DELETE /api/v1/users/{publicId}/block`、`GET /api/v1/users/me/blocked`。

- [x] **步骤 2：补 Follow endpoints schema**

  固定 `POST /api/v1/users/{publicId}/follow`、`DELETE /api/v1/users/{publicId}/follow`、followers / following cursor 分页。

- [x] **步骤 3：更新服务级 HTTP README 索引**

  新 endpoint 状态先标记为“草案”。

## 任务 2：关系 application

**测试立场：** TDD - 幂等、统计、拉黑解除关注和事件一致性属于 R4。

- [x] **步骤 1：编写失败测试**

  覆盖 `BlockUser`、`UnblockUser`、`ListBlockedUsers`、`BatchCheckBlocked`、`FollowUser`、`UnfollowUser`、`ListFollowers`、`ListFollowing`、`CheckFollowing`。

  运行：`cd services/zhicore-user && go test ./internal/user/application -run 'TestBlock|TestFollow'`

  预期：失败。

- [x] **步骤 2：实现 relationship domain**

  固定自关注 / 自拉黑错误、关系命令幂等成功、任一方向拉黑后禁止关注。

- [x] **步骤 3：实现 application**

  拉黑时同事务删除双方关注关系、修正统计、发布 `user.blocked` 和必要的 `user.unfollowed(reason=BLOCKED)`；解除拉黑不恢复关注。

- [x] **步骤 4：运行 application 测试**

  运行：`cd services/zhicore-user && go test ./internal/user/application`

  预期：通过。

## 任务 3：关系 HTTP endpoints

**测试立场：** TDD - path、cursor、错误码和幂等语义是公开 contract。

- [x] **步骤 1：编写 handler 失败测试**

  覆盖自关注 / 自拉黑 `400`、拉黑后关注 `403`、cursor 非法 `400`、重复提交幂等成功。

  运行：`cd services/zhicore-user && go test ./api/http -run 'TestBlock|TestFollow'`

  预期：失败。

- [x] **步骤 2：实现 handler**

  path `publicId` 只用于解析目标用户；当前操作者只来自 `X-User-Id`。

- [x] **步骤 3：更新 endpoint 状态**

  被 handler contract test 覆盖的关系 endpoint 标记为“已验证”。

- [x] **步骤 4：运行关系收口测试**

  运行：`cd services/zhicore-user && go test ./api/http ./internal/user/...`

  预期：通过。

## 架构适配评估

- 关系事实只在 User 服务内维护。
- 统计是可从 `user_follows` 重建的读模型。
- Block / Follow 依赖 Profile `ACTIVE` guard，因此本计划必须在 Profile 计划之后执行。
