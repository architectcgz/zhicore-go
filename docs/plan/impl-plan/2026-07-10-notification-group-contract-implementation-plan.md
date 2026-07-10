# 通知聚合组 Contract Implementation Plan

> **给 agentic workers：** 使用 @test-driven-development 按任务逐项推进；每个步骤达到预期后立即更新 checkbox。提交前使用 @committing-changes。

**目标：** 以三份 Notification endpoint contract 为事实源，实现列表聚合组、触发者分页、组级已读以及 Vue 通知中心的对应状态流。

**架构：** Content、Comment、User provider 先在事件中输出公开 target/anchor、actor 展示和内容快照；Notification 将这些不可变事实与稳定 `groupId` 一并写入 inbox 和 group state。Go handler 只解析 HTTP 和映射 DTO；application 持有 `groupId + recipient` 权限与命令语义；PostgreSQL 查询、游标和事务留在 store。Vue API adapter 归一化 DTO，feature composable 负责加载、详情、actor 分页、乐观组级已读和回滚，route page 只渲染与派发用户意图。

**技术栈：** Go、Gin、PostgreSQL、Vue 3、TypeScript、Axios、Vitest。

---

## 范围与非目标

- 范围：`GET /api/v1/notifications`、`GET /api/v1/notification-groups/{groupId}/actors`、`POST /api/v1/notification-groups/{groupId}/read` 及前端消费。
- 非目标：改变单条通知已读和全部已读的 HTTP path；为历史内部 ID 建立兼容 API；在通知列表同步 N+1 查询 User。
- 依据：三份 endpoint contract、`docs/architecture/services/notification/README.md`、前端 `docs/design/pages/notification.md`。

## 任务 1：上游事件公开快照 contract

**边界：** `libs/contracts/events/{content,comment,user}` 与 Content、Comment、User producer outbox。

**验收：**

- [ ] 三类事件带 actor `publicId`、`displayName`、可选 `avatarUrl`，及 target resource/anchor 的公开 ID 和历史展示快照。
- [ ] Notification consumer 拒绝缺失快照的 producer contract；不把内部数值转换成 HTTP ID。
- [ ] producer 和 consumer contract test 锁定字段和 JSON 形状。

- [ ] 写失败 contract test 并确认失败。
- [ ] 实现 provider payload、outbox 映射与 Notification 解析。
- [ ] 运行相关 Content、Comment、User、Notification 测试。

## 任务 2：聚合组读写模型

**边界：** migration、`ports`、application、PostgreSQL store。

**验收：**

- [ ] migration 将不可变 `group_id` 同时持久化在 inbox 和 group state；backfill/rebuild 对同一 `(recipient_id, group_key)` 得到同一 ID，并有唯一索引和 owner 限定反查。
- [ ] `groupId` 稳定且不暴露 `group_key`，所有读写按 `recipient_id + groupId` 限定。
- [ ] 列表按 `latestOccurredAt DESC, groupId DESC` cursor 稳定排序，`totalCount`、`unreadCount`、`actorTotalCount` 有不同含义。
- [ ] actor 列表按最新时间、公开 actor ID 稳定排序并合并同一 actor 的 `eventCount`。
- [ ] 单条、组级、全部已读按同一 group-state→通知行→stats 锁顺序执行并失效 unread/aggregation cache；重复调用 `changedCount=0`，并发新通知保留未读。

- [ ] 写 repository/application 失败测试并确认失败。
- [ ] 实现最小 migration、port、query 和 command。
- [ ] 运行 `cd services/zhicore-notification && go test ./internal/notification/... -run 'Group|Aggregated|Actor' -count=1`。

## 任务 3：HTTP contract

**边界：** `api/http` handler、payload、contract test。

**验收：**

- [ ] 三个 path 返回标准 envelope；参数非法为 `1001`，缺登录为 `2006`，非 owner 与不存在统一 `1005`。
- [ ] 列表 JSON 使用 `groupId`、`recentActors`、`target`、RFC3339 UTC，空数组绝不编码为 `null`。
- [ ] actor cursor 与 group read 返回规定字段；未知 `type` 原样输出。

- [ ] 写 handler 失败测试并确认失败。
- [ ] 实现 handler/DTO 映射。
- [ ] 运行 `cd services/zhicore-notification && go test ./api/http -run 'NotificationGroup|ListNotifications' -count=1`。

## 任务 4：Vue contract 迁移

**边界：** `src/api/notification.ts`、notification feature mapper/composable、route page 与贴近 owner 的测试。

**验收：**

- [ ] adapter 使用 `groupId` 和两个新的 group endpoint，不再以单条 `notificationId` 标记组已读。
- [ ] feature 防止同一组重复提交，组级 read 乐观更新后以 `changedCount/unreadCount` 为准；同步更新分类 breakdown，未知或并发状态重拉 unread facts，失败回滚。
- [ ] 详情默认使用 `recentActors`，按需加载 actor cursor；切组或关闭丢弃 stale response、失败保留已加载 actor，未知 target 保留展示并禁用跳转。

- [ ] 写 Vitest 失败用例并确认失败。
- [ ] 实现最小 adapter、mapper、composable 与页面接线。
- [ ] 运行定向 Vitest 与 `pnpm typecheck`。

## 集成验证与回退

- [ ] 运行 `go test ./...`、相关 Vitest、`pnpm typecheck`、`make test-size`、`bash scripts/check-structure.sh` 和必要的 `make check`。
- [ ] 有可用 PostgreSQL 时验证已有数据升级、backfill/rebuild 与 migration `up -> down 1 -> up`；无隔离数据库时在交付说明中记录。
- [ ] migration 和前端 adapter 可独立回退；不回退或改写已有单条已读 API。

## 架构适配评估

- [ ] `groupId` 的生成、解析和 owner 检查属于 Notification application/repository，不泄漏到 HTTP 或 Vue。
- [ ] DTO 归一化只在 API/mapper；route page 不直接调用 adapter。
- [ ] 计划不依赖二次重构：actor 详情是单独 endpoint，列表不做 User N+1。
