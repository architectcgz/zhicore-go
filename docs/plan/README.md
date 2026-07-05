# 实施计划索引

本目录记录跨服务、跨仓或结构性任务的实施计划。计划不是当前事实源；执行完成后的长期结论必须回写到对应架构、contract、review 或迁移文档。

## 目录

- `impl-plan/`：正式实现计划，要求按任务 checklist 执行、记录验证并接受 review。
- `exploratory/`：探索性设计方案和技术计划，执行前可提升为正式实现计划。
- `archive/`：已完成、已替代或不再作为当前执行入口的计划归档。

## 当前计划

| 计划 | 范围 | 状态 |
| --- | --- | --- |
| `impl-plan/2026-07-05-content-module-completion-implementation-plan.md` | Content 可运行 runtime、worker、系统测试、错误契约、剩余 API family、限流和观测收口 | 待执行 |
| `exploratory/2026-07-04-content-body-parser-typed-schema-design-plan.md` | Content V1 body parser 从动态 `map[string]any` 重构为强类型 schema | 探索方案 |

## 已归档计划

| 计划 | 范围 | 归档原因 |
| --- | --- | --- |
| `archive/impl-plan/2026-07-04-api-req-resp-foundation-implementation-plan.md` | 前后端 API 基础 `Req` / `Resp`、provider adapter、服务级 HTTP schema 和 feature workflow 接入 | 检查项已完成 |
| `archive/impl-plan/2026-07-04-module-architecture-implementation-plan.md` | `auth`、`user`、`comment` 模块拆分路线图和执行顺序 | 路线图已被子计划消化 |
| `archive/impl-plan/2026-07-04-shared-httpapi-error-writer-implementation-plan.md` | 共享 HTTP envelope 支持业务错误码与 HTTP status 分离 | 检查项已完成 |
| `archive/impl-plan/2026-07-04-auth-authentication-foundation-implementation-plan.md` | Auth 注册、登录、refresh、logout、session 和 security operation 基础 | 检查项已完成 |
| `archive/impl-plan/2026-07-04-user-profile-foundation-implementation-plan.md` | User Profile 初始化、查询、更新、状态和 HTTP Profile endpoints | 检查项已完成；生产 runtime 深化另开计划 |
| `archive/impl-plan/2026-07-04-user-relationship-foundation-implementation-plan.md` | User Block / Follow 关系、统计、事件和 HTTP endpoints | 检查项已完成 |
| `archive/impl-plan/2026-07-04-comment-create-page-foundation-implementation-plan.md` | Comment 创建根评论 / 回复、顶级评论分页、核心 schema 和首批 HTTP endpoints | 检查项已完成 |
| `archive/impl-plan/2026-07-04-comment-interaction-outbox-implementation-plan.md` | Comment 删除、点赞、计数 delta、outbox worker 和 runtime 收口 | 检查项已完成 |
| `archive/impl-plan/2026-07-05-gin-http-migration-implementation-plan.md` | 已有 HTTP handler 统一迁移到 Gin 并去除中间状态 | 检查项已完成 |
| `archive/impl-plan/2026-07-05-content-publish-foundation-implementation-plan.md` | Content 创建草稿、保存正文、发布文章和读取 published body 最小闭环 | 检查项已完成 |
