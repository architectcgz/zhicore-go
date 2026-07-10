# 共享后台 worker 循环原语收敛实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；涉及 `libs/kit` 新原语、Content / Notification / Comment runtime 迁移和生命周期语义的步骤按 @test-driven-development 执行。每个 checkbox 达到预期后立即更新；如需提交，提交前必须先使用 @committing-changes，且必须按最小可审阅切片拆分提交，禁止把新原语、三个服务迁移和文档合成一个大提交。

**目标：** 把当前分散在三个服务、行为语义不一致的后台 worker 循环骨架收敛为 `libs/kit` 中单一、已验证、panic 安全且能自愈的共享原语，消灭 Content / Comment 的静默永久退出缺陷和 readiness 盲区。

**架构：** 新增 `libs/kit/worker`,提供轮询循环层(`Loop`)+ 生命周期编排层(`Supervisor`)两个正交能力,与已有 `libs/kit/taskworker`(claim-process-ack 任务处理层)分工清晰:一个 `Loop` 的每轮执行体(`Tick`)内部可以调用 `taskworker.Runner.RunUntilIdle`。原语只负责"轮询循环 + panic 恢复 + 遇错自愈 + 生命周期 + readiness 状态 + 可选每轮观测",不承载任何业务概念(限流、outbox、campaign 等留在各服务)。三个服务的 runtime 改为装配该原语,删除各自的 `pollingWorker` / `LoopWorker` / `OutboxDispatcher.Run` 循环层。

**技术栈：** Go 1.26、`libs/kit/observability`、`libs/kit/worker`(新增)、`go.work` 多 module、`make check`。

---

## 背景依据

- `docs/reviews/backend/2026-07-10-content-module-completion.md`(Content worker 静默退出 Blocker 的独立 review 证据)
- `docs/todos/debt/content-worker-lifecycle-no-restart.md`
- `docs/architecture/runtime-operations.md`
- `docs/architecture/go-service-design.md`
- `docs/architecture/observability.md`
- `docs/architecture/testing.md`
- `docs/reviews/quality-gates.md`
- 现有正确样板:`services/zhicore-notification/internal/notification/runtime/module.go`(`LoopWorker` + `workerSupervisor`)

## 当前基线(三套并存,行为不一致)

| 服务 | 循环类型 | 非 context 错误 | panic recovery | readiness 跟踪 | 是否已接线运行 |
| --- | --- | --- | --- | --- | --- |
| Content | `pollingWorker.Run`(`internal/content/runtime/module.go:316`) | `return`(worker 死) ❌ | 无 ❌ | 恒绿 `healthyWorkerChecker`(`module.go:375`) ❌ | 是,cmd/server 完整接线 |
| Notification | `LoopWorker.Run` + `workerSupervisor`(`internal/notification/runtime/module.go:182`) | 继续下一轮 ✅ | `runOnceSafely` recover ✅ | `workerSupervisor` 跟踪 ✅ | 是 |
| Comment | `OutboxDispatcher.Run`(`internal/comment/application/outbox_worker.go:127`)+ `outboxWorker`(`runtime/module.go:118`) | `return`(worker 死,且只排除 `Canceled` 未排除 `DeadlineExceeded`) ❌ | 无 ❌ | 完全无跟踪(dependency-free `healthHandler`) ❌ | 否,`cmd/server/main.go` 传空 `Deps{}`,outbox worker 从未运行 |

- 全仓库确认仅此三套 worker 循环骨架,无第四套(其余 `ctx.Done` 均为 HTTP server 优雅关机)。
- `libs/kit` 已有共享原语惯例(`taskworker`、`postgres/outbox`、`observability` 等),但缺 worker loop 原语。
- Content 的 `ObserveWorkerResult` 每轮上报 `zhicore_content_worker_jobs_total` 计数(`internal/content/runtime/observer.go:39`);Notification `LoopWorker` 无观测;Comment 无观测。

## 已确认的设计决策(执行时不得擅自更改)

- **落点**:`libs/kit/worker`,与 `taskworker` 并列。
- **命名**:panic 恢复方法命名为 `runOnceRecovered`(不用 `runOnceSafely`),对齐 Go `recover()` 术语。
- **panic 是公开契约**:`Tick` 类型文档必须显式声明"每轮 panic 会被 recover 并转成 error,视为该轮失败——worker 不崩溃、不静默退出、继续下一轮";不得把 recover 藏成私有实现细节。
- **遇错自愈**:非 context 错误 → 记录/上报后继续下一轮,绝不 return;仅 `ctx` 取消/超时才退出。此为原语核心契约。
- **观测走方案 A**:原语内置**可选**的每轮结果回调(成功/失败/panic + 耗时),复用现有 `libs/kit/observability.MetricsRecorder`,不趁机换 Prometheus/otel(观测后端选型是独立决策,原语与其解耦)。回调契约固化两条安全约束:(1) 标签只用稳定 errorClass,禁止原始错误 / DSN / broker URL 进标签;(2) 观测失败不影响循环业务流。
- **只搬 worker-result 观测**:Content 的 `ObserveRateLimitDecision` 是 HTTP 请求路径概念,与 worker 循环无关,留在 Content,不进原语。
- **Supervisor 一起提取**:暴露 `Descriptors()`(Name/Ready),供 `/health/ready` 消费,解决 Content 恒绿 + Comment 无跟踪。

## 不可并行修改文件

- `libs/kit/worker/*`:由任务 1 先固定原语 API 和契约;任务 2/3/4 的三个服务迁移都依赖它合并后才能开始,不得在任务 1 未合并前预先引用。
- 三个服务的 runtime 迁移(任务 2/3/4)彼此独立,原语合并后可并行执行。

---

## 任务 1:新增 `libs/kit/worker` 共享原语

**测试立场：** TDD - 循环自愈、panic 恢复、生命周期 start/stop、readiness 状态转换和观测回调都是行为承载,先写失败测试。

**文件：**
- 新增：`libs/kit/worker/doc.go`
- 新增：`libs/kit/worker/loop.go`(`Tick` 类型 + `Loop` + `runOnceRecovered`)
- 新增：`libs/kit/worker/loop_test.go`
- 新增：`libs/kit/worker/supervisor.go`(`Supervisor` + `Descriptor` + `Start`/`Stop`/`Descriptors`)
- 新增：`libs/kit/worker/supervisor_test.go`
- 新增：`libs/kit/worker/observer.go`(可选每轮结果回调类型 + 稳定 errorClass 归一化)
- 新增：`libs/kit/worker/observer_test.go`
- 修改:`libs/kit/README.md` 或对应 kit 索引(若存在),登记新原语

**验收清单：**

- [ ] `Tick func(context.Context) error` 类型有公开文档,显式声明 panic→error 契约和"遇错继续、仅 context 退出"语义。
- [ ] `Loop.Run` 对非 context 错误:调用可选观测回调后继续下一轮,绝不 return;仅 `ctx.Err() != nil` 时返回 `ctx.Err()`。
- [ ] `runOnceRecovered` 用 `defer recover()` 把每轮 panic 转成 error,worker 不崩溃、继续下一轮;命名为 `runOnceRecovered`。
- [ ] 每轮结束(成功 / 失败 / panic)都产出一个 `Result`(至少含 Name、Status、ErrorClass、Duration);观测回调可选,为 nil 时零成本跳过。
- [ ] errorClass 归一化到稳定有限类别(至少 `canceled` / `deadline_exceeded` / `panic` / `error`),禁止把原始 error 文本、DSN、broker URL 放进结果标签。
- [ ] 观测回调 panic 或耗时不得影响循环;回调失败被吞掉不冒泡。
- [ ] `Supervisor.Start` 为每个 worker 起独立 goroutine,`Stop` 通过 cancel + WaitGroup 有界等待,`ctx` 超时返回 `ctx.Err()`。
- [ ] `Supervisor` 在 worker 运行时 `Ready=true`,worker 退出(含 panic / ctx 取消)后翻转为 `Ready=false`;`Descriptors()` 反映实时状态。
- [ ] worker 内 panic 被 supervisor 边界二次兜底(defer recover),使 readiness 翻转且进程能继续优雅关机。
- [ ] 空 worker 列表时 `Supervisor` 为 no-op(Start/Stop 返回 nil),不 panic。

- [ ] **步骤 1:写 loop / supervisor / observer 失败测试**

  至少覆盖:非 context 错误后继续下一轮、context 取消退出、每轮 panic 被恢复后继续、观测回调收到 success/failed/panic 三态、errorClass 稳定化、supervisor readiness 随 worker 生死翻转、Stop 有界等待。

- [ ] **步骤 2:实现 `Loop`、`runOnceRecovered`、`Supervisor` 和观测回调**

  行为对齐 Notification 已验证的 `LoopWorker` + `workerSupervisor` 语义,叠加可选观测回调。

- [ ] **步骤 3:运行原语验证**

  运行:`cd libs/kit && go test ./worker/... -race -count=1`

  预期:通过。

- [ ] **步骤 4:提交原语切片**

  原语独立提交,不和任何服务迁移混在一起。

## 任务 2:Notification 迁移到共享原语(样板对齐,最低风险)

**测试立场：** TDD - 先让现有 `health_test.go` 的 worker 行为测试改指向共享原语并保持通过。

**文件：**
- 修改：`services/zhicore-notification/internal/notification/runtime/module.go`(删除本地 `LoopWorker` / `runOnceSafely` / `workerSupervisor`,改用 `libs/kit/worker`)
- 修改：`services/zhicore-notification/internal/notification/runtime/default_build.go`(`BuildCampaignShardWorkers` 改造 `NewLoopWorker` 调用)
- 修改：`services/zhicore-notification/internal/notification/runtime/health_test.go`(worker 相关测试对齐新原语)
- 视需要修改:`services/zhicore-notification/cmd/server/*`(readiness 消费点)

**验收清单：**

- [ ] Notification 不再自有 worker 循环 / supervisor 实现,全部委托 `libs/kit/worker`。
- [ ] campaign shard worker 的 workerID lease 语义(shard 租约令牌)不变,迁移不改变 lease 正确性。
- [ ] `/health/ready` 的 worker 状态仍由 supervisor 实时提供,`WorkerDescriptor` 对外 JSON 字段(name/enabled/ready)不变。
- [ ] `TestLoopWorkerContinuesAfterRecoverableRunOnceError` 等价行为在新原语下仍被测试覆盖(可迁移到 `libs/kit/worker` 或保留服务级集成测试)。

- [ ] **步骤 1:改测试指向共享原语**
- [ ] **步骤 2:删除本地循环 / supervisor,接入 `libs/kit/worker`**
- [ ] **步骤 3:运行验证**

  运行:`cd services/zhicore-notification && go test ./... -race -count=1`

- [ ] **步骤 4:提交 Notification 迁移切片**

## 任务 3:Content 迁移并修复静默退出 + readiness 假绿(Blocker 修复)

**测试立场：** TDD - 这是修复 `content-worker-lifecycle-no-restart.md` 记录的 Blocker,先写"瞬时错误后 worker 继续、readiness 反映真实状态"的失败测试。

**文件：**
- 修改：`services/zhicore-content/internal/content/runtime/module.go`(删除 `pollingWorker`、`healthyWorkerChecker` 恒绿逻辑,`WorkerDescriptor` readiness 改由 supervisor 提供)
- 修改：`services/zhicore-content/cmd/server/runtime_deps.go`(`contentWorkerLifecycle` 改为委托 `libs/kit/worker.Supervisor`,删除裸 goroutine + done channel 无人消费的接线)
- 修改：`services/zhicore-content/cmd/server/server.go`(如需要:让运行期能感知 worker 退出)
- 修改：`services/zhicore-content/internal/content/runtime/observer.go`(`ObserveWorkerResult` 改为适配 `libs/kit/worker` 的观测回调;`ObserveRateLimitDecision` 保留不动)
- 修改：`services/zhicore-content/internal/content/runtime/module_test.go`、`workers_test.go`、`observer_test.go`、`health_test.go`
- 视需要修改:`internal/content/ports/rate_limit.go`(`WorkerResult` / `ContentObserver` 拆分,worker-result 部分对齐原语)

**验收清单：**

- [ ] Content 不再自有 `pollingWorker` 循环;worker 遇瞬时(非 context)错误后继续下一轮,不再永久退出。
- [ ] `/health/ready` 的 worker 项由 supervisor 实时状态驱动,worker 退出后 readiness 变为 not ready,不再恒绿。
- [ ] worker 内 panic 被恢复,不再掀翻整个 Content 进程。
- [ ] 现有 `zhicore_content_worker_jobs_total` 指标语义(worker/operation/status/errorClass 标签,稳定 errorClass)保持不变,通过原语观测回调产出。
- [ ] `ObserveRateLimitDecision` 及其 `zhicore_content_rate_limit_decisions_total` 指标不受影响,仍在 Content HTTP 路径。
- [ ] 新增测试证明:注入一次瞬时错误后 worker 仍在下一轮被调用(锁死回归)。

- [ ] **步骤 1:写静默退出 + readiness 回归失败测试**
- [ ] **步骤 2:迁移 runtime 到 `libs/kit/worker`,拆分 observer**
- [ ] **步骤 3:运行验证**

  运行:`cd services/zhicore-content && go test ./... -race -count=1`

- [ ] **步骤 4:更新 Blocker debt 状态并提交**

  更新 `docs/todos/debt/content-worker-lifecycle-no-restart.md` 为已处理,迁移与 debt 状态分开提交。

## 任务 4:Comment 迁移并修复同款静默退出

**测试立场：** TDD - Comment outbox worker 尚未接线运行,但循环缺陷已在代码中;先写行为测试再迁移。

**文件：**
- 修改：`services/zhicore-comment/internal/comment/application/outbox_worker.go`(删除自有 `Run(ctx, interval)` 循环层,`DispatchOnce` 保留为 `Tick`)
- 修改：`services/zhicore-comment/internal/comment/runtime/module.go`(`outboxWorker` 改用 `libs/kit/worker`,readiness 接 supervisor,`healthHandler` 补 worker 状态)
- 视需要修改:`services/zhicore-comment/internal/comment/application/outbox_worker_test.go` 及 runtime 测试
- 视需要修改:`services/zhicore-comment/cmd/server/main.go`(仅在不扩大接线范围前提下调整;完整接线不在本计划范围)

**验收清单：**

- [ ] Comment 不再自有 worker 循环;`DispatchOnce` 作为 `Tick` 被共享 `Loop` 驱动。
- [ ] 非 context 错误(含 `DeadlineExceeded`,原实现漏排除)后 worker 继续下一轮,不再静默退出。
- [ ] Comment outbox worker 若被启用,readiness 能反映其状态(不再 dependency-free 恒绿)。
- [ ] 不扩大 Comment 服务的整体接线范围:`cmd/server/main.go` 的 fail-fast 骨架语义除必要调整外保持;完整接线留给 Comment 自己的后续计划。

- [ ] **步骤 1:写 outbox worker 循环行为失败测试**
- [ ] **步骤 2:迁移到 `libs/kit/worker`**
- [ ] **步骤 3:运行验证**

  运行:`cd services/zhicore-comment && go test ./... -race -count=1`

- [ ] **步骤 4:提交 Comment 迁移切片**

## 任务 5:文档收口、debt 关闭与最终验证

**测试立场：** 验证门禁 + 文档切片。

**文件：**
- 修改：`docs/todos/debt/content-worker-lifecycle-no-restart.md`(标记已处理,或改为跨服务并关闭)
- 修改：`docs/architecture/runtime-operations.md` 或 `observability.md`(记录共享 worker 原语的循环 / panic / readiness / 观测契约为事实源)
- 新增：`docs/reviews/backend/<date>-shared-worker-loop-primitive.md`(review 证据)
- 视需要修改:各服务 runtime 相关 README

**验收清单：**

- [ ] `content-worker-lifecycle-no-restart.md` 的退出条件已满足并标记已处理。
- [ ] 架构文档写清共享原语的四条契约(遇错自愈 / panic→error / readiness 实时 / 可选稳定观测)为长期事实源,不再散落在三个服务。
- [ ] 三套骨架已消除,全仓库只剩 `libs/kit/worker` 一处 worker 循环实现。

- [ ] **步骤 1:更新架构文档和 debt 状态**
- [ ] **步骤 2:请求独立 review**

  对完整 diff、原语契约、三个服务迁移和回归测试做 review;有 finding 先用 @receiving-code-review 判断有效性再最小修复。

- [ ] **步骤 3:记录 review 证据**

- [ ] **步骤 4:最终提交 review 证据**

## 集成验证

- [ ] 运行 `cd libs/kit && go test ./worker/... -race -count=1`。
- [ ] 运行 `cd services/zhicore-notification && go test ./... -race -count=1`。
- [ ] 运行 `cd services/zhicore-content && go test ./... -race -count=1`。
- [ ] 运行 `cd services/zhicore-comment && go test ./... -race -count=1`。
- [ ] 运行 `bash scripts/check-structure.sh`。
- [ ] 运行 `git diff --check`。
- [ ] 触达 `libs/kit` 共享原语和多个服务 runtime,交付前运行 `make check`。

## 架构适配评估

- 新原语 `libs/kit/worker` 只承载"后台轮询循环 + 生命周期 + readiness + 可选观测"的通用控制流,不知道任何业务概念,与 `libs/kit/taskworker`(任务处理)正交。
- 观测通过依赖倒置接 `libs/kit/observability.MetricsRecorder`,真实 exporter(Prometheus / OpenTelemetry)选型作为独立后续决策,不与本次重构耦合;换端时本原语零改动。
- Content 的 `ObserveRateLimitDecision` 属于 HTTP 请求路径,明确留在 Content,不进原语,避免原语沾染限流业务概念。
