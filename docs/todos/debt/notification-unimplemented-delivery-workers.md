# 技术债：Notification digest 投递与 consumed_events 清理 worker 未实现

状态：未处理
优先级：中
负责人：未分配
来源：`docs/reviews/backend/2026-07-07-notification-module-foundation-review.md` 之后的实现盘点（核对 `2026-07-06-notification-module-foundation-implementation-plan.md`）

## 影响

Notification 模块基础实现计划的 checkbox 已全部达成，但计划文件清单中的两个 worker 只落地了写入侧，缺消费侧，属于计划范围内但未强制 checkbox 的实现缺口：

- **digest 投递 job（`process_pending_digest`）未实现。** campaign fanout 对 `DIGEST_ONLY` 订阅者只在 `notification_delivery` 写入 `DIGEST_PENDING` 状态（`infrastructure/postgres/campaign_store.go:162`），全库没有消费该状态、聚合成 digest 并投递、再落 `DIGEST_DELIVERED` 的 job。`DIGEST_PENDING` 目前只被 `retry_delivery.sql` 引用。净效果：`DIGEST_ONLY` 用户永远收不到 digest，delivery 记录无限停留在 pending。

- **`consumed_events` 清理 worker 未实现。** `runtime/default_build.go:101` 把 `cleanup_consumed_events` worker 声明为 `Enabled:false, Ready:false`，全库没有对应的清理 SQL 或 job；`consumed_events` 只有 insert/mark，没有按保留期回收过期行的路径。净效果：双层幂等表随事件量单调增长，无回收。

## 退出条件

- 实现 digest 投递 job：消费 `DIGEST_PENDING` delivery、按接收者聚合、投递后落 `DIGEST_DELIVERED`（或失败落 `FAILED`），纳入 runtime start/stop、panic recovery、readiness descriptor 和配置校验。
- 实现 `consumed_events` 清理 job：按 `consumer.consumed_events_retention` 配置回收过期行，接入 runtime worker lifecycle，并把 `cleanup_consumed_events` descriptor 置为 enabled/ready。
- 补对应失败测试与验证命令；触达 runtime / migration 后运行 `make check`。

## 备注

两项均为该计划“文件清单项”而非验收 checkbox，故不影响计划完成判定；但 digest 与幂等表回收是可运维性的真实缺口，独立登记以免随计划归档而丢失。相关 review：[[content-worker-lifecycle-no-restart]] 记录的是 Content 侧同类 worker lifecycle 隐患，可一并作为“共享 worker 生命周期基线”的参考。
