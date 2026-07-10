# Notification module foundation review

## 范围

- Worktree: `task/2026-07-06-notification-module-foundation`
- 计划: `docs/plan/archive/impl-plan/2026-07-06-notification-module-foundation-implementation-plan.md`
- 重点: campaign shard fanout、lease 安全、runtime worker lifecycle、migration 和质量门禁。

## 独立 review 结论

初次独立 review 判定为 blocked，发现 3 个阻塞问题：

1. campaign shard 拉取 follower 后直接 `CompleteCampaignShard`，没有物化 inbox / delivery。
2. `CompleteCampaignShard` / `FailCampaignShard` 只按 shard id 更新，旧 worker 过期后仍可能覆盖新 claim。
3. `LoopWorker` 单轮 recoverable error 会让 worker 永久退出，导致 retry shard 无人处理。

## 修复

- `CampaignShardExecutor` 改为先调用 `MaterializeCampaignFollowers`，再按实际 `ProcessedCount` / `SuccessCount` / `SkippedCount` / `FailedCount` 更新 shard 进度。
- PostgreSQL store 在 materialization 事务内为 follower 写入 `notifications`、`notification_group_state`、`notification_stats` 和 `notification_delivery`。
- materialization 增加 recipient channel plan，读取通知偏好、DND 和作者订阅；`MUTED` 写 `SKIPPED` 不物化未读，`DIGEST_ONLY` 写 `DIGEST_PENDING` 不物化 inbox，关闭或命中 DND 的 `WEBSOCKET` 不创建 `WEBSOCKET_PENDING`。
- shard complete / fail 增加 `WorkerID` + `ClaimDeadlineAt` lease token，SQL 限定 `status='PROCESSING'`、`claimed_by` 和 `claim_deadline_at`；0 行更新返回 `ErrShardLeaseLost`。
- `LoopWorker` 将单轮 error / panic 限定为本轮失败并继续轮询；只有 context cancel 才退出。
- HTTP server `serveErr` 分支补充 `StopRuntime`，避免监听失败后 worker 继续使用已关闭依赖。
- User follower client 拆出 `user_service.timeout` 配置，runtime wiring 不再复用 HTTP server read timeout。
- RabbitMQ consumer handler 增加真实 broker 集成测试入口，验证 handler 对坏消息发布 DLQ 后再 ack 原消息；默认无 `ZHICORE_NOTIFICATION_RABBITMQ_INTEGRATION_URL` 时跳过。

## 验证

- `cd libs/contracts && go test ./clients/user -count=1`
- `cd services/zhicore-notification && go test ./... -count=1`
- `cd services/zhicore-notification && go test -race ./internal/notification/... -run 'Consumer|Campaign|Unread|ReadAll' -count=1`
- `cd services/zhicore-notification && go test -race ./cmd/server ./internal/notification/runtime ./internal/notification/application -run 'Campaign|Worker|Module|Runtime' -count=1`
- `cd services/zhicore-notification && ZHICORE_NOTIFICATION_PG_INTEGRATION_DSN='postgres://postgres:postgres123456@localhost:5432/zhicore_notification_agent_test?sslmode=disable' go test ./internal/notification/infrastructure/postgres -run TestPostgresMaterializeCampaignFollowersFanoutIntegration -count=1 -v`
- `cd services/zhicore-notification && ZHICORE_NOTIFICATION_PG_INTEGRATION_DSN='postgres://postgres:postgres123456@localhost:5432/zhicore_notification_agent_test?sslmode=disable' go test ./internal/notification/infrastructure/postgres -run TestPostgresClaimCampaignShardMultiWorkerIntegration -count=1 -v`
- `cd services/zhicore-notification && go test ./internal/notification/infrastructure/rabbitmq -run TestRabbitMQConsumerHandlerDeadLettersAndAcksIntegration -count=1 -v`：默认跳过；设置 `ZHICORE_NOTIFICATION_RABBITMQ_INTEGRATION_URL` 后运行真实 broker 测试。
- `make test-size`
- `bash scripts/check-structure.sh`
- `make check`
- 使用 `shared-postgres:5432` 的 `zhicore_notification_agent_test` 执行 notification migration `up -> down 1 -> up`。
- 独立 reviewer 复查确认 lease 和 worker blocker 已关闭；第二轮指出 preference / DND / author subscription 绕过问题，已按上方修复并补测试。
- 第三轮独立 reviewer 复查结论为 `pass_with_remaining_risks`，未发现仍然阻塞的 blocker。

## 残余风险

- `Campaign.MaxConcurrentShardJobs` 已接入 runtime，会展开为多个带唯一 `WorkerID` 的 campaign shard loop，并补充 worker readiness / race 验证。
- User follower client timeout 已拆出独立配置；后续仍可按 provider 增加 retry / circuit breaker policy。
- `MaterializeCampaignFollowers` 已补真实 PostgreSQL fanout 集成测试，覆盖 muted / digest / DND / 全局偏好分支对 `notifications`、`notification_stats` 和 `notification_delivery` 的实际写入影响。
- `ClaimCampaignShard` 已补真实 PostgreSQL 多 worker 集成测试，覆盖同批 pending shard 在并发 worker 下不会重复 claim，并用 lease token 完成分片。
- campaign follower materialization 当前在单事务内逐 follower 决策和写入；默认 batch size 200 可接受，后续提高 batch 或接入更多 channel 时需关注事务时长和锁放大。
