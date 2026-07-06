# 未实现服务模块总览实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；进入任一服务计划前按该服务计划声明的技能和 TDD / 验证策略执行。每个 checkbox 达到预期后立即更新；如需提交，提交前必须先使用 @committing-changes。

**目标：** 把 `docs/todos/debt/unimplemented-service-modules.md` 中的未实现服务拆成可执行的服务级 impl-plan，并明确不应进入当前实现排期的模块。

**架构：** 每个可实现服务按 `api/http -> application -> domain/ports -> infrastructure -> runtime` 增量落地；跨服务同步依赖先补 provider-owned `libs/contracts/clients/<provider>`，事件依赖先补 `libs/contracts/events/<domain>`。Gateway 保持薄入口，Admin 只做 provider 委托和本地审计，Search/Ranking 只维护派生读模型，Message/Notification 分别拥有自己的未读聚合。

**技术栈：** Go 1.26、Gin HTTP router、PostgreSQL、Redis、RabbitMQ、MongoDB、`libs/contracts`、服务级 HTTP schema、`make check`。

---

## 背景依据

- `docs/todos/debt/unimplemented-service-modules.md`
- `docs/migration/service-migration-workflow.md`
- `docs/architecture/service-boundaries.md`
- `docs/architecture/repository-layout.md`
- `docs/architecture/go-service-design.md`
- `docs/architecture/testing.md`
- `docs/contracts/http-schema-template.md`
- `docs/reviews/quality-gates.md`

## 模块处理结果

| 模块 | 处理方式 | 计划 |
| --- | --- | --- |
| `zhicore-admin` | 正式实现计划 | `docs/plan/impl-plan/2026-07-06-admin-moderation-facade-foundation-implementation-plan.md` |
| `zhicore-gateway` | 正式实现计划 | `docs/plan/impl-plan/2026-07-06-gateway-routing-auth-foundation-implementation-plan.md` |
| `zhicore-message` | 正式实现计划 | `docs/plan/impl-plan/2026-07-06-message-module-foundation-implementation-plan.md` |
| `zhicore-notification` | 正式实现计划 | `docs/plan/impl-plan/2026-07-06-notification-module-foundation-implementation-plan.md` |
| `zhicore-search` | 正式实现计划 | `docs/plan/impl-plan/2026-07-06-search-post-index-foundation-implementation-plan.md` |
| `zhicore-ranking` | 正式实现计划 | `docs/plan/impl-plan/2026-07-06-ranking-ledger-hot-posts-foundation-implementation-plan.md` |
| `zhicore-ops` | 当前不写服务实现计划 | 未固定首个运维 use case；候选 `/api/v1/ops/*` 不是 contract backlog。 |
| `zhicore-id-generator` | 排除当前实现计划 | 已决策“不迁移 / 不提供 HTTP API”；保留为未来集中发号落点。 |

## 并行执行边界

- `Admin` 与 `Gateway` 都需要 `libs/contracts/clients/auth`，不能同时改同一 contract 文件；应先抽一个共享 Auth contract 前置切片，或让其中一个计划 owning 该切片。
- `Message` 与 `Notification` 都需要补 `libs/contracts/clients/user` 的内部查询能力，不能并行改同一 `contract.go`；建议先执行 Message 的 User message guard contract，再执行 Notification campaign follower shard contract。
- `Message` 与 `Notification` 都会修改 `services/zhicore-user/api/http/internal_handlers.go`；必须按 User provider-owned internal route 分片顺序执行，不能并行编辑同一 handler 文件。
- `Search` 与 `Ranking` 都依赖 Content typed client 和 Content 事件 Go 类型，不能并行改同一 Content contract；建议先独立完成 Ranking hot foundation 需要的事件类型，再让 Search 复用并补索引字段。
- 服务内 handler/application/repository/runtime 切片可以按服务并行执行，前提是共享 `libs/contracts` 前置切片已经合并。

## 共享 contract 前置顺序

| 共享 contract | Owner 计划 | Consumer 计划 | 并行规则 |
| --- | --- | --- | --- |
| `libs/contracts/clients/auth` | Gateway 计划任务 2 先固定 `ValidateAccessState`；Admin 计划只在其后追加管理端 disable / enable contract。 | Gateway、Admin | Gateway 任务 2 未合并前，Admin 不得编辑 `libs/contracts/clients/auth/contract.go`。 |
| `libs/contracts/clients/user` | Message 计划任务 1 先固定 message guard contract；Notification 计划任务 6 只在其后追加 follower shard contract。 | Message、Notification | Message 任务 1 未合并前，Notification 不得编辑 `libs/contracts/clients/user/contract.go`。 |
| `services/zhicore-user/api/http/internal_handlers.go` | Message 计划任务 1 先追加 message guard internal route；Notification 计划任务 6 在其后追加 follower shard internal route。 | Message、Notification | Message 任务 1 未合并前，Notification 不得编辑同一 handler 文件；如需同时推进，先拆 User provider-owned 前置切片。 |
| `libs/contracts/clients/content` | Ranking 计划任务 1 先固定 hot foundation 所需 `publicId -> internalId` 与批量摘要；Search 计划任务 1 在其后追加索引回源字段。 | Ranking、Search、Admin | Ranking 任务 1 未合并前，Search / Admin 不得编辑 `libs/contracts/clients/content/contract.go`。 |
| `libs/contracts/events/content` | Ranking 计划任务 1 先补 hot foundation 事件类型；Search 计划任务 3 在其后追加搜索索引需要的 `updated` / `tags.updated` 字段。 | Ranking、Search、Notification | Ranking 任务 1 未合并前，Search 不得编辑 `libs/contracts/events/content/contract.go`。 |
| `libs/contracts/clients/comment` | Admin 计划任务 2 拥有首个 Comment admin contract；Ranking / Notification 如需新增 Comment typed client 另开后续切片。 | Admin | 首个创建由 Admin 计划执行，其他计划不得并行创建同名目录或文件。 |

## `zhicore-ops` 不进入当前实现计划

**当前证据：**

- `services/zhicore-ops/internal/ops/doc.go` 只有包注释。
- `services/zhicore-ops/api/http/README.md` 明确 Java `/api/gray` 灰度接口当前不迁移。
- `docs/architecture/services/ops/README.md` 把 Ops 定义为内部迁移、检查、对账、修复、回放或运维工具落点。

**恢复条件：**

- [ ] 出现单一、具体的首个任务：`reconcile`、`repair-tasks` 或 `event-replay` 三选一。
- [ ] 写清 owner、caller、provider-consumer 关系，以及是否需要其他服务直接调用 Ops。
- [ ] 写清只能通过归属服务 contract 或受控 repair task 修改业务数据，不能把 Ops 变成跨库修表入口。
- [ ] 补齐对应 endpoint schema、`cmd/server/main.go`、`internal/ops/runtime/module.go` 和最小验证命令后，再单独新建正式 impl-plan。

## `zhicore-id-generator` 排除当前实现计划

**当前证据：**

- `docs/architecture/id-strategy.md` 决策普通服务内部主键使用各服务数据库 identity。
- `services/zhicore-id-generator/api/http/README.md` 当前状态为“不迁移 / 不提供 HTTP API”。
- `docs/architecture/services/id-generator/README.md` 明确当前没有使用 Snowflake 的业务场景。

**恢复条件：**

- [ ] 存在明确业务 owner 和真实调用方。
- [ ] 已证明数据库 identity 不能满足该场景。
- [ ] 已说明跨数据库、离线、多主写入、时钟回拨、worker 分配、segment 缓存、容量和高可用要求。
- [ ] 先更新 `docs/architecture/id-strategy.md`、`docs/architecture/services/id-generator/README.md` 和 `services/zhicore-id-generator/api/http/README.md`。
- [ ] 在上述条件满足前，不创建 `services/zhicore-id-generator/api/http/endpoints/`，不补 `/api/v1/id/*` contract，不创建实现计划。

## 总体验证

- [ ] 新增或修改计划后运行 `bash scripts/check-structure.sh`。
- [ ] 运行 `git diff --check`。
- [ ] 执行具体服务计划时先按服务计划中的最窄 `go test` 验证；完整执行任一服务计划或触达共享 contract / runtime / migration 后，交付前必须运行 `make check`，只有纯文档切片可降级为 `bash scripts/check-structure.sh && git diff --check`。
