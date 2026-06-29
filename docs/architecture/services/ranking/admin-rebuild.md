# Ranking Admin Rebuild 设计

本文固定 `zhicore-ranking` 管理员 `rebuild-from-ledger` 的权限、审计、互斥锁、状态机和 HTTP contract 关系。rebuild 执行流程见 [application-and-ports.md](application-and-ports.md)，运行期依赖故障见 [runtime-resilience.md](runtime-resilience.md)。

当前状态：设计事实源，不表示 Go handler / worker 已实现。

## 结论

- `rebuild-from-ledger` 是高风险管理员操作，Ranking 必须自己做管理员权限校验；Admin facade 只能委托，不能替代 Ranking 的校验。
- 请求成功只表示任务受理，HTTP `POST` 返回 `ACCEPTED`；执行进度和最终结果通过 `operationId` 查询。
- Ranking 保存 `ranking_rebuild_operation` 作为操作状态和审计事实；Admin 服务可以额外保存管理端审计，但不能成为 Ranking rebuild 状态事实源。
- rebuild 必须持有 Redis lock 或配置允许的 PostgreSQL advisory lock；无锁时拒绝启动。
- 首期 `force=true` 不允许，返回 `1008`；未来允许覆盖 stale lock 前必须先补更强审计和人工确认策略。

## 权限

HTTP endpoint：

- `POST /api/v1/ranking/admin/rebuild-from-ledger`
- `GET /api/v1/ranking/admin/rebuild-operations/{operationId}`

鉴权要求：

| 条件 | 失败码 |
| --- | --- |
| 缺少 `X-User-Id` | `2006` / `401` |
| `X-User-Roles` 不包含管理员角色 | `2007` / `403` |
| 角色格式非法或被业务策略拒绝 | `2008` / `403` |

handler 只把可信 header 映射成 `AdminActor`。application 负责判断角色、状态、锁、dry run 和 force 策略。

## 请求校验

| 字段 | 规则 |
| --- | --- |
| `dryRun` | 缺失为 `false`；`true` 时只校验权限、依赖和锁可获得性，不执行重建。 |
| `reason` | 可选，最长 200；建议管理端必填，但服务首期不强制。 |
| `force` | 缺失为 `false`；首期传 `true` 返回 `1008`。 |

`reason` 是审计摘要，不允许保存完整请求 body、token、Authorization、原始事件 payload 或生产连接信息。

## 互斥锁和 barrier

锁策略：

| 阶段 | 规则 |
| --- | --- |
| lock key | `ranking:lock:rebuild`。 |
| lock owner | `operationId`。 |
| TTL | 默认 `30m`，执行中必须续期。 |
| Redis 可用 | 使用 Redis lock。 |
| Redis 不可用 | 配置允许时使用 PostgreSQL advisory lock；否则返回 `1004`。 |
| 已有有效锁 | 返回 `1008`，不启动第二个 rebuild。 |

barrier：

- 获取锁后设置 `ranking:replay:active=true` 或等价状态。
- consumer 看到 barrier 后停止拉取、暂停写业务表，或 nack / requeue 等待 broker 重投。
- rebuild worker 等待 in-flight drain；超过 `drain_timeout` 标记 `FAILED`，释放 barrier 和锁。
- 任何失败路径都必须尽力释放 barrier；锁 TTL 是最后兜底。

## 状态机

| 状态 | 含义 | 允许转移 |
| --- | --- | --- |
| `ACCEPTED` | 请求校验通过，任务已创建。 | `RUNNING`、`FAILED`、`CANCELED` |
| `RUNNING` | 已持有锁并开始执行。 | `SUCCEEDED`、`PARTIAL_FAILED`、`FAILED`、`CANCELED` |
| `SUCCEEDED` | PG state、period、Redis snapshot 和候选集全部完成。 | 终态 |
| `PARTIAL_FAILED` | PG rebuild 成功，但 Redis refresh、candidate refresh 或后置阶段失败。 | 终态 |
| `FAILED` | 未完成核心 rebuild 或执行中失败。 | 终态 |
| `CANCELED` | 预留状态，首期不暴露取消 endpoint。 | 终态 |

`PARTIAL_FAILED` 不回滚 PostgreSQL 权威状态；后续 snapshot / candidate refresh 可修复 Redis 投影。

## 审计字段

`ranking_rebuild_operation` 至少保存：

| 字段 | 说明 |
| --- | --- |
| `operation_id` | 对外查询 ID。 |
| `requested_by` | 管理员 User 内部 ID。 |
| `reason` | 管理员触发原因摘要。 |
| `dry_run` / `force` | 请求策略。 |
| `status` | 当前状态。 |
| `lock_key` / `lock_owner` / `lock_ttl_seconds` | 锁信息。 |
| `failed_stage` / `error_code` / `error_message` | 失败分类；不暴露底层 SQL、Redis、堆栈或密钥。 |
| `replayed_events` / `rebuilt_posts` / `refreshed_snapshots` / `refreshed_candidates` | 执行结果摘要。 |
| `request_id` / `trace_id` | 观测关联。 |
| `accepted_at` / `started_at` / `completed_at` / `duration_ms` | 时间和耗时。 |

高风险失败也要写状态记录；不能只写普通日志。

## HTTP Contract

字段级 schema 见 `services/zhicore-ranking/api/http/endpoints/ranking-api.md`：

- `RebuildAccepted`
- `RebuildOperationStatus`
- `POST /api/v1/ranking/admin/rebuild-from-ledger`
- `GET /api/v1/ranking/admin/rebuild-operations/{operationId}`

状态查询只允许管理员访问。`operationId` 不作为权限凭证；即使知道 ID，没有管理员角色也不能查询。

## 测试准入

- Handler contract test：未登录、非管理员、非法 body、`force=true`、已有锁、lock 不可用、accepted response、status not found。
- Application test：状态转移、dry run 不执行 rebuild、无锁拒绝、已有运行任务拒绝、partial failedStage 记录。
- Worker test：barrier 开启、in-flight drain timeout、PG rebuild 成功但 Redis refresh 失败时进入 `PARTIAL_FAILED`。
- Audit test：失败和成功都写 `ranking_rebuild_operation`，且错误信息脱敏。
