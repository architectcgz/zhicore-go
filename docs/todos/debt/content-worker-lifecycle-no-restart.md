# 技术债：Content 后台 worker 瞬时错误后永久退出且 readiness 假绿

状态：未处理
优先级：高
负责人：未分配
来源：`docs/reviews/backend/2026-07-10-content-module-completion.md`（Content 模块补全独立 review Blocker）

## 影响

Content 的四个轮询 worker（`content-body-cleanup`、`content-body-repair`、`content-engagement-stats`、`content-outbox-dispatcher`）在遇到任何非 context 的瞬时错误（例如一次 Postgres 连接抖动、claim 查询临时失败）后会永久退出，且进程不会重启它们，`/health/ready` 仍持续报绿：

- `libs/kit/taskworker` 的 `Runner.RunUntilIdle`（`runner.go:91`）在 `Claim` / `MarkSucceeded` / `MarkFailed` 出错时直接 `return err`，无 panic recovery。
- `pollingWorker.Run`（`services/zhicore-content/internal/content/runtime/module.go:330`）对任何非 `context.Canceled` / `context.DeadlineExceeded` 错误直接 `return`，不重试、不退避、不重新进入轮询循环。
- `contentWorkerLifecycle.Start`（`services/zhicore-content/cmd/server/runtime_deps.go:226`）的 goroutine 把 worker 的返回值写入 `done` channel 后就退出；`Wait()`（同文件 `239`）只在关机路径消费 `done`。
- `runContentServer` 的主 select（`services/zhicore-content/cmd/server/server.go:86-94`）只监听 `serveErr` / `signals` / `ctx.Done()`，运行期从不消费 worker 退出信号，也没有 watchdog / 重启。
- readiness 使用永远返回 `nil` 的 `healthyWorkerChecker`（`module.go:375`），worker 退出后 readiness 探针无法反映。

净效果：outbox 事件停止投递、body cleanup / repair 停摆、engagement 统计不再收敛，而编排系统认为服务健康、不会重调度或告警。这与 Notification review 已修复的 finding #3（`LoopWorker` 单轮 error 永久退出）属同一类问题，但 Content 的 `pollingWorker` 未采用同样的“仅 context cancel 才退出、其余错误继续轮询”的修法。

## 退出条件

- `pollingWorker`（或其调度层）对瞬时错误改为记录 + 退避后继续轮询，只有 context cancel / 关机才终止，参考 Notification `LoopWorker` 的修法。
- 补 panic recovery，使单个 task handler panic 不会杀死 worker goroutine。
- worker 永久退出（若仍保留该终态）能通过 `/health/ready` 的 worker checker 反映，不再恒为绿。
- 覆盖以下场景的测试：单轮瞬时错误后 worker 继续存活并在下一轮恢复；handler panic 被 recover；worker 终止后 readiness 转为 not ready。

## 备注

其余高副作用路径本次 review 已核实安全：限流写路径 `DEGRADED_DENY_UNAVAILABLE` fail-closed、读路径 `DEGRADED_ALLOW_LOCAL` 放行（`internal/content/application/rate_limit.go:27-37`）；engagement stats 的 claim + apply + mark 在单事务内且丢失 claim 时幂等返回 `ErrTaskClaimLost`（`internal/content/infrastructure/postgres/tasks.go:207-235`）。
