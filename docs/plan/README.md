# 实施计划索引

本目录记录跨服务、跨仓或结构性任务的实施计划。计划不是当前事实源；执行完成后的长期结论必须回写到对应架构、contract、review 或迁移文档。

## 目录

- `impl-plan/`：正式实现计划，要求按任务 checklist 执行、记录验证并接受 review。

## 当前计划

| 计划 | 范围 | 状态 |
| --- | --- | --- |
| `impl-plan/2026-07-04-api-req-resp-foundation-implementation-plan.md` | 前后端 API 基础 `Req` / `Resp`、provider adapter、服务级 HTTP schema 和 feature workflow 接入 | 待执行 |
| `impl-plan/2026-07-04-module-architecture-implementation-plan.md` | `auth`、`user`、`comment` 模块拆分路线图和执行顺序 | 路线图 |
| `impl-plan/2026-07-04-shared-httpapi-error-writer-implementation-plan.md` | 共享 HTTP envelope 支持业务错误码与 HTTP status 分离 | 待执行 |
| `impl-plan/2026-07-04-auth-authentication-foundation-implementation-plan.md` | Auth 注册、登录、refresh、logout、session 和 security operation 基础 | 待执行 |
| `impl-plan/2026-07-04-user-profile-foundation-implementation-plan.md` | User Profile 初始化、查询、更新、状态和 HTTP Profile endpoints | 部分完成，生产 runtime 待补 |
| `impl-plan/2026-07-04-user-relationship-foundation-implementation-plan.md` | User Block / Follow 关系、统计、事件和 HTTP endpoints | 待执行 |
| `impl-plan/2026-07-04-comment-create-page-foundation-implementation-plan.md` | Comment 创建根评论 / 回复、顶级评论分页、核心 schema 和首批 HTTP endpoints | 已完成 |
| `impl-plan/2026-07-04-comment-interaction-outbox-implementation-plan.md` | Comment 删除、点赞、计数 delta、outbox worker 和 runtime 收口 | 待执行 |
