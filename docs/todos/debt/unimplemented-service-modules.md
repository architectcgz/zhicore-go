# 技术债：未实现服务模块盘点

状态：未处理
优先级：中
负责人：未分配
来源：2026-07-06 仓库服务目录、Go 源码、服务 README 与 HTTP schema 交叉检查。

## 影响

仓库中部分服务目录已经具备 `go.mod`、`api/http/README.md`、`internal/<module>/doc.go` 和迁移设计文档，但还没有真实 Go handler、application、domain、ports、runtime 或测试实现。后续排期如果只看目录存在，容易误判为服务已经进入交付阶段。

本次检查把服务分成三类：

- **完全未实现 / 只有脚手架**：目录存在，但生产 Go 源码只有 `doc.go` 或 `.gitkeep`，HTTP schema 仍是占位或 API 族识别状态。
- **有设计或 schema，但 Go 实现未开始**：已有字段级 schema 或 migration 草案，但 handler / application / runtime 仍未实现。
- **明确不迁移 / 不默认实现**：保留映射目录，当前不是服务交付缺口。

## 当前未实现服务模块

| 服务模块 | 当前证据 | 后续入口 |
| --- | --- | --- |
| `zhicore-admin` | 生产 Go 源码仅 `internal/admin/doc.go`；`api/http/README.md` 标注“当前仅做计划化占位”，endpoint 只有“API 族已识别”。 | 先提取 Admin API contract，再补 `reports` / `audit_logs` migration、编排测试和 provider client。 |
| `zhicore-gateway` | 生产 Go 源码仅 `internal/gateway/doc.go`；Gateway 自有 endpoint 仍为候选。 | 先固定路由清单、认证失败 / 权限失败 / 限流失败 envelope、最小 middleware 链和身份 header 注入规则。 |
| `zhicore-message` | 生产 Go 源码仅 `internal/message/doc.go`；HTTP schema 仍是私信和会话切片占位。 | 先明确外部 IM provider 契约，再提取发送、会话摘要、未读数和历史空列表兼容 contract / 测试。 |
| `zhicore-notification` | 生产 Go 源码仅 `internal/notification/doc.go`；HTTP schema 仍是通知中心切片占位。 | 先固定通知列表、未读、偏好、DND、作者订阅和 fanout 的字段级 contract。 |
| `zhicore-search` | 生产 Go 源码仅 `internal/search/doc.go`；HTTP schema 仍是搜索 API 族占位。 | 先提取 Search API contract、PostgreSQL 搜索读模型、索引重建和事件消费幂等测试。 |
| `zhicore-ops` | 生产 Go 源码仅 `internal/ops/doc.go`；HTTP schema 只记录运维候选和“不迁移 Java 灰度接口”。 | 当前不迁移灰度 API；如果要做运维工具，先按对账、修复或事件回放定义具体任务。 |

## 有 schema 但 Go 实现未开始

| 服务模块 | 当前证据 | 后续入口 |
| --- | --- | --- |
| `zhicore-ranking` | 已有 HTTP 字段级 schema、endpoint 文档和 migration 草案；生产 Go 源码仍只有 `internal/ranking/doc.go`，`api/http/README.md` 明确 `Go handler：待实现`、`Go contract test：待实现`。 | 首个切片闭合“事件账本 + bucket + 文章总榜查询”，先实现 `ListHotPosts`、`ListHotPostsWithScore`、`GetPostRank`、`GetPostScore` 和对应 handler contract test。 |

## 明确不作为当前缺口的模块

| 服务模块 | 当前结论 | 依据 |
| --- | --- | --- |
| `zhicore-id-generator` | 当前不迁移、不实现 HTTP API，也不接入其他 Go 服务；目录只是旧服务映射和未来集中发号能力落点。 | `services/zhicore-id-generator/README.md` 和 `api/http/README.md` 明确“当前不迁移 / 不提供 HTTP API”。 |

## 部分实现但仍需补齐的模块

这些模块已有真实 Go 实现，不归入“未实现服务模块”，但仍不能视为完整交付：

- `zhicore-file`：已有 HTTP handler、application 和 contract test；仍缺 metadata migration / repository、MinIO adapter、URL 签名策略、临时 / 未绑定 TTL、GC worker 和部分失败分支 contract test。
- `zhicore-content`：已有发布闭环 foundation 和多条已验证 endpoint；仍缺生产可运行 runtime、真实依赖打开、HTTP server listen / shutdown、真实 readiness、cleanup / repair / outbox worker、system HTTP test、剩余 API family、限流、resilience policy 和观测接入。
- `zhicore-auth`：已有注册、登录、refresh、session 等切片的 handler / application / migration；仍需闭合与 User 的注册初始化协议、Gateway access state fallback、更多安全操作和生产 runtime。
- `zhicore-user`：已有 profile、relationship、内部查询等切片的 handler / application / migration；仍需按模块设计补齐 Profile 状态、缓存 / File adapter / 事件发布和生产 runtime。
- `zhicore-comment`：已有首批评论创建、分页、详情、删除、点赞状态等 handler / application / migration；仍需补游标分页、增量补拉、编辑、管理端、outbox admin、计数 delta worker、缓存和生产 runtime。

## 退出条件

满足以下条件后可以关闭本技术债：

1. 对每个“当前未实现服务模块”，要么进入明确实现计划并落地最小可运行切片，要么在对应服务 README / HTTP schema 中登记“不迁移 / 不默认实现”的决策和恢复条件。
2. 对 `zhicore-ranking`，至少完成首个实现切片的 handler、application、domain / ports、migration contract test 和 handler contract test，且 `api/http/README.md` 不再标注 Go handler / contract test 待实现。
3. `docs/todos/debt/README.md` 的当前条目同步更新，避免后续服务盘点遗漏。

## 备注

本条目只记录服务模块级实现覆盖，不替代各服务自己的 implementation plan、review 证据或字段级 HTTP contract。后续实现单个服务前仍需按项目 `AGENTS.md` 读取对应服务设计、模块设计、migration workflow 和质量门禁。
