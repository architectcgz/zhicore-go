# Auth / User / Comment 模块实现路线图

> **给 agentic workers：** 本文件是路线图，不是直接执行计划。执行时请选择下方某一个小计划，并使用 @subagent-driven-development 或 @executing-plans 逐任务推进；如需提交，提交前必须先使用 @committing-changes。

**目标：** 把原先一次性覆盖 `auth`、`user`、`comment` 的大计划拆成多个可独立执行、验证和 review 的小计划。

**架构：** 三个模块仍遵守 `api/http -> application -> domain/ports -> infrastructure -> runtime`。共享层只承载稳定技术原语，业务模型、repository、状态机和事件写入保持在各服务内部。

**技术栈：** Go 1.22 workspace、标准库 `net/http`、PostgreSQL SQL migration、Redis、RabbitMQ outbox、`libs/kit/httpapi`、`libs/contracts`。

---

## 拆分原则

- 每个小计划只覆盖一个模块内一个交付切片，或一个真正共享的前置能力。
- 每个小计划必须能独立运行最窄相关测试，并能单独提交。
- 优先顺序按依赖推进：共享 HTTP 错误 writer -> Auth 认证基础 -> User Profile -> User Relationship -> Comment 创建 / 分页 -> Comment 互动 / worker。
- 跨服务 system test、Gateway 路由、Admin facade、前端页面和完整部署不塞进首批小计划。

## 执行顺序

| 顺序 | 小计划 | 范围 | 可独立验证 |
| --- | --- | --- | --- |
| 1 | `2026-07-04-shared-httpapi-error-writer-implementation-plan.md` | `libs/kit/httpapi` 支持业务错误码与 HTTP status 分离 | `cd libs/kit && go test ./...` |
| 2 | `2026-07-04-auth-authentication-foundation-implementation-plan.md` | Auth 注册、登录、refresh、logout、session 查询和撤销的最小可测基础 | `cd services/zhicore-auth && go test ./api/http ./internal/auth/...` |
| 3 | `2026-07-04-user-profile-foundation-implementation-plan.md` | User Profile 初始化、查询、更新、状态和 HTTP Profile endpoints | `cd services/zhicore-user && go test ./api/http ./internal/user/...` |
| 4 | `2026-07-04-user-relationship-foundation-implementation-plan.md` | User Block / Follow 关系、统计、事件和 HTTP endpoints | `cd services/zhicore-user && go test ./api/http ./internal/user/...` |
| 5 | `2026-07-04-comment-create-page-foundation-implementation-plan.md` | Comment 创建根评论 / 回复、顶级评论分页、核心 schema 和首批 HTTP endpoints | `cd services/zhicore-comment && go test ./api/http ./internal/comment/...` |
| 6 | `2026-07-04-comment-interaction-outbox-implementation-plan.md` | Comment 删除、点赞、计数 delta、outbox worker 和运行时收口 | `cd services/zhicore-comment && go test ./api/http ./internal/comment/...` |

## 已知张力

- Auth `service.md` 明确注册后 User profile 初始化由 `auth.account.registered` 事件驱动；User `api.md` 仍保留同步 `CreateProfileForAccount` 描述。执行 Auth / User 小计划时，以 Auth 当前 service 决策为准，并在 User 文档同步任务中修正旧口径。
- Comment 是 Go-first API reset；Auth / User 默认不破坏已登记 HTTP contract。
- 当前仓库还没有 CI，交付证据以本地验证命令为准。

## 路线图验证

修改本路线图或新增小计划后运行：

```bash
bash scripts/check-structure.sh
git diff --check -- docs/plan
```

