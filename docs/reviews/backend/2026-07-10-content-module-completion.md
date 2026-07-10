# Content 模块补全 Review

## Review 对象

- 计划：`docs/plan/impl-plan/2026-07-05-content-module-completion-implementation-plan.md`（任务 0–12）
- Diff 范围：`af13254..HEAD`（baseline `af13254` = 发布闭环交付证据）
- 补全范围：runtime/config、四类 polling worker（body cleanup / repair、engagement stats projection、outbox dispatch）、错误契约、管理端文章 HTTP + 审计、taxonomy、engagement（点赞/收藏）、reader-presence 移除、Redis fixed-window 限流、resilience 配置矩阵、observability 适配器；触达 `libs/kit/taskworker`（新 runner）、`libs/kit/rabbitmq`、`tests/system/http`、`tests/testkit`。
- 规模：content 服务 241 文件、约 +23600 行；另 `libs/kit/taskworker` 新增 runner + 测试。
- Review 类型：任务 12 步骤 6 独立 review 门禁。

## 分类判断

高副作用写路径（发布 copy-on-write、outbox dispatch、engagement stats projection、限流 fail-open/closed）按 TDD 立场逐一核对；worker lifecycle 和 readiness 作为“可运行、可运维”目标的核心属性重点审查。

## Findings

### Blocker（未修复）

**worker 遇瞬时错误永久静默退出，且 `/health/ready` 仍报绿。**

链条：

1. `libs/kit/taskworker/runner.go:91` — `RunUntilIdle` 在 `Claim` / `MarkSucceeded` / `MarkFailed` 返回错误时直接 `return err`，runner 本身无 panic recovery。
2. `services/zhicore-content/internal/content/runtime/module.go:325-330` — `pollingWorker.Run` 对任何非 `context.Canceled` / `context.DeadlineExceeded` 错误直接 `return`，不做退避重试。
3. `services/zhicore-content/cmd/server/runtime_deps.go:224-231` — `contentWorkerLifecycle.Start` 的 goroutine 把 `Run` 的返回值塞进带缓冲的 `done` channel 后退出；`Wait()`（同文件:239-255）只在关机路径被调用时才读 `done`。
4. `services/zhicore-content/cmd/server/server.go:86-94` — `runContentServer` 的 select 只监听 `serveErr` / `signals` / `ctx.Done()`，**运行期从不消费 worker 退出信号**，进程不会因 worker 死亡而崩溃或重启。
5. `services/zhicore-content/internal/content/runtime/module.go:305-306,373-377` — enabled worker 的 readiness checker 是恒返回 `nil` 的 `healthyWorkerChecker`，worker 退出后 readiness 不感知。

**失败场景：** outbox dispatcher（或 body cleanup / repair / engagement stats）在一次瞬时 PostgreSQL 抖动中 `Claim` 报错 → 对应 worker goroutine 永久退出 → 发布事件不再投递、正文孤儿不再清理、互动统计不再收敛，而 `GET /health/ready` 持续返回 ready，运维和编排器无法察觉。

**对照：** Notification review（`docs/reviews/backend/2026-07-07-notification-module-foundation-review.md` finding #3）已修过同类问题——`LoopWorker` 将单轮 recoverable error / panic 限定为本轮失败并继续轮询，只有 context cancel 才退出。Content 的 `pollingWorker` 未采用同一约定。

**附带风险（Note 级）：** handler 内 panic 会穿透 `RunUntilIdle`（无 recover）和 `pollingWorker.Run`（无 recover）到 `contentWorkerLifecycle.Start` 的裸 goroutine，导致整个 server 进程崩溃。单条畸形任务即可拉垮进程。

### 已核实安全的路径

- **限流 fail-open / fail-closed**（`internal/content/application/rate_limit.go:27-37`）：写路径 `DEGRADED_DENY_UNAVAILABLE` → `ErrDependencyUnavailable` fail-closed；读路径 `DEGRADED_ALLOW_LOCAL` → 放行；`REJECT_TOO_FREQUENT` → `ErrRateLimited`；未知 outcome → fail-closed。无“限流器不可用时静默成功”的写路径。
- **engagement stats 幂等**（`internal/content/infrastructure/postgres/tasks.go:207-235`）：claim + apply delta + mark done 在单事务内；`!claimed` 返回 `ErrTaskClaimLost` 且 worker handle 将其视为 no-op（`engagement_stats_worker.go:59-66`），瞬时 apply 失败走正常 retry / dead-letter，不重复计数。
- **限流 observer 脱敏**（`internal/content/runtime/observer.go`）：worker result 只记录稳定 `errorClass`，不把原始错误、DSN、broker URL 写入 label。

### Note

- 计划已声明的待补项属实、非缺陷：resilience 的 breaker / max-in-flight 目前是配置事实，真实执行器待后续切片；真实 metrics exporter 待接入。
- Notification（本次同批核对的另一模块，非本 review 范围）存在 `cleanup_consumed_events` worker 声明为 disabled 且无实现、`process_pending_digest` 仅落 `DIGEST_PENDING` 无消费 job 两处非 checkbox 缺口，已另行记录。

## 验证证据

任务 12 步骤 1–5 已执行，结果如下（本机，无真实外部依赖）：

```bash
cd services/zhicore-content && go test ./...           # 全部 ok（含 cached）
go test ./tests/system/http -run TestContent           # ok（无真实依赖时 skip 相关子用例）
python3 scripts/check-test-size.py --files services/zhicore-content tests/system/http tests/testkit  # exit 0
bash scripts/check-structure.sh                        # structure ok
git diff --check                                       # 无 whitespace error
```

## Required Re-validation

修改 Content runtime、worker lifecycle、HTTP handler、application 语义、PostgreSQL/MongoDB/Redis/RabbitMQ adapters 或 migration 时，至少重跑：

```bash
cd services/zhicore-content && go test ./...
python3 scripts/check-test-size.py --files services/zhicore-content tests/system/http tests/testkit
bash scripts/check-structure.sh
```

触达 `libs/kit/taskworker`、`libs/kit/rabbitmq` 等共享边界后，交付前再跑 `make check`。

## Residual Risk

- 系统测试在无真实 PostgreSQL / MongoDB / Redis / RabbitMQ 时跳过端到端断言；当前证据层级为服务内 Go test + handler contract test。
- migration 未用真实目标库执行 `up -> down 1 -> up`。
- 上述 Blocker 未修复：worker 永久退出 + readiness 假绿在生产运行期会造成事件投递、正文修复、互动统计静默停摆。

## 技术债状态

- 新增待处理 Blocker（见 Findings）：worker lifecycle 无退避重试 / 无 panic recovery / readiness 不反映 worker 死亡。修复方向与 owner 待用户确认后落地或登记 `docs/todos/debt/`。
