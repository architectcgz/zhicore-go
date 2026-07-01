# User 运行期 Resilience 设计

本文记录 `zhicore-user` 首次实现时必须落地的 timeout、retry、circuit breaker、max-in-flight 和降级策略。全局规则见 `docs/architecture/runtime-operations.md`；本文只补 User 的 provider / operation 级矩阵。

当前状态：本文是设计事实源，不表示 Go 代码已经实现。首次实现任一 endpoint、typed client、adapter、dispatcher 或 worker 前，必须把对应行的配置项、adapter 行为测试或 application 编排测试补齐。

## 总原则

- resilience policy 在 runtime wiring 声明，不能散落到 handler、application、repository 或 adapter。
- 统计维度使用 `provider + operation`。Profile 查询、关系查询、头像校验、URL 解析和 outbox 投递不能共用一个全局熔断开关。
- timeout、retry、breaker、max-in-flight、fallback 窗口和 worker 并发数必须配置化；默认值只作为本地开发和首轮压测基线。
- User 是 profile、relationship、block 和 check-in 事实源。PostgreSQL 不可用时不能把失败伪装成用户不存在、未关注或未拉黑。
- 缓存不是正确性依赖。Redis 缓存失败可以回源 PostgreSQL，但回源必须受限流和 max-in-flight 保护。
- 写路径默认 fail closed。头像校验、创建 profile、关注、拉黑、状态变更、Admin 资料命令和 check-in 写入在关键依赖不可确认时不得伪造成功。
- 降级策略由 application 选择；adapter 只返回语义错误、观测字段和 dependency status，不构造业务 DTO 伪装成功。
- 服务间 typed client 必须携带 `X-Caller-Service` 和 `X-Caller-Operation`；缺少可信 caller identity 的内部接口不能落到匿名公开配额。
- 熔断打开、降级执行、本机限流兜底、retry 最终失败和 URL 解析降级必须可观测，字段规则见 `docs/architecture/observability.md`。

## 下游依赖矩阵

| Provider | Operation | 调用方 / 场景 | Timeout 基线 | Retry | Circuit breaker key | Max in-flight | 降级策略 | 幂等 / 一致性 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `postgres` | `user.command_tx` | profile 初始化、资料更新、状态变更、Admin 删除 / 恢复 | `1s..3s` | 不在事务外盲重试；仅依靠 `accountId`、`nickname`、状态条件和 outbox 唯一键处理幂等 | `postgres.user.command_tx` | 按 DB pool 和写路径并发配置保护 | `fail-fast` -> 业务错误或 `SERVICE_DEGRADED` | 事务内更新 users 和 outbox；失败不能承诺成功。 |
| `postgres` | `user.query` | `GetMe`、公开 profile、UserSimple、availability、Admin 查询 | `1s..3s` | 对连接抖动最多 2 次总尝试 | `postgres.user.query` | 按查询族限制，保护公开 profile 热点 | DB 失败返回 `SERVICE_DEGRADED`；不得伪装成 404 / 空列表 | 缺失用户和依赖失败必须区分。 |
| `postgres` | `relationship.command_tx` | follow、unfollow、block、unblock、统计同步 | `1s..3s` | 不盲重试；依靠唯一约束和幂等删除处理重复请求 | `postgres.relationship.command_tx` | 关系写路径独立限并发 | `fail-fast` -> 业务错误或 `SERVICE_DEGRADED` | 同事务写关系、统计和 outbox。 |
| `postgres` | `relationship.query` | followers、following、blocked list、blocked pair、follow check、follower shard | `1s..3s` | 对连接抖动最多 2 次总尝试 | `postgres.relationship.query` | 列表和内部 fanout 查询分别限并发 | DB 失败返回 `SERVICE_DEGRADED`；不得伪装成未关注 / 未拉黑 | cursor 查询必须稳定排序。 |
| `postgres` | `check_in.command_tx` | 后续签到写入、连续签到统计 | `1s..3s` | 不盲重试；依靠日期唯一键幂等 | `postgres.check_in.command_tx` | check-in 写路径独立限并发 | Redis bitmap 不可用时仍以 PG 为准；PG 失败返回失败 | Check-in 后续实现时补测试。 |
| `redis` | `rate_limit.check` | User 业务限流 | `100ms..300ms` | 不重试放大延迟；按 `rate-limiting.md` 可短时本机 fallback | `redis.rate_limit.check` | 独立于缓存连接池 | 按 `rate-limiting.md` 决定 `ALLOW` / `1003` / `1004` | 高副作用路径不能 fail-open。 |
| `redis` | `user.cache` | profile、UserSimple、availability、`publicId -> userId` cache-aside | `100ms..300ms` | 不阻塞主查询重试 | `redis.user.cache` | cache 操作独立限并发 | cache miss / error 后受控回源 DB；写缓存失败 best-effort | cache 不是事实源。 |
| `redis` | `relationship.cache` | follow/block pair、列表和统计缓存 | `100ms..300ms` | 不阻塞主查询重试 | `redis.relationship.cache` | 热用户关系独立限并发 | cache error 后受控回源 DB；写路径权限检查首批优先查 DB | 权限类缓存延迟必须有失效测试。 |
| `redis` | `check_in_bitmap` | 后续月度签到图和连续签到读模型 | `100ms..300ms` | 不重试阻塞用户路径 | `redis.check_in_bitmap` | check-in 热读独立限并发 | Redis 不可用时可回源 PG 或返回 `SERVICE_DEGRADED`，由后续 schema 固定 | PG 仍是签到事实源。 |
| `upload-service` | `file.validate_avatar` | `UpdateProfile` 写入头像前校验文件可引用 | `2s..5s` | 只读校验最多 2 次总尝试 | `upload.file.validate_avatar` | Profile 更新路径限并发 | 业务拒绝映射 `USER_AVATAR_INVALID`；依赖不可用映射 `SERVICE_DEGRADED`，不写 profile | 不能把不可确认文件引用写入 User 事实。 |
| `upload-service` | `file.resolve_avatar_url` | `GetMe`、公开 profile、列表展示派生 `avatarUrl` | `1s..2s` | 可重试最多 2 次总尝试 | `upload.file.resolve_avatar_url` | 查询路径限并发，避免拖垮 profile 查询 | contract 允许时省略 / 置空 `avatarUrl` 并记录 degraded；profile 查询仍可成功 | URL 是派生展示字段，不落库、不进事件。 |
| `auth-service` | `account.verify_registration_source` | 可选：`CreateProfileForAccount` 校验 account 存在或调用来源 | `2s..5s` | 只读校验最多 2 次总尝试 | `auth.account.verify_registration_source` | Auth 编排路径限并发 | Auth 不可用时 profile 初始化失败并由 Auth 补偿，不创建孤儿 profile | User 不写 Auth 状态。 |
| `rabbitmq` | `outbox.publish` | outbox dispatcher 投递 User 集成事件 | publish confirm timeout `2s..5s` | 按 outbox retry + backoff + jitter | `rabbitmq.outbox.publish` | dispatcher worker 并发配置 | 失败更新 outbox retry metadata；超过阈值进入 dead / admin retry | `event_id` 唯一，consumer 按 event id 幂等。 |

## Provider Operation 语义

User 作为 provider 的 typed client endpoint 必须区分“业务不存在”和“User 或依赖不可用”。consumer 的 timeout、retry、breaker 由 consumer runtime policy 管，但 User contract 必须固定以下失败语义。

| Operation | Consumer | Provider 失败语义 | Consumer 降级规则 |
| --- | --- | --- | --- |
| `CreateProfileForAccount` | Auth | `accountId` 已存在返回已有 profile；DB / Auth 校验不可用返回 `SERVICE_DEGRADED` | Auth 不能向客户端承诺完整注册成功；进入补偿或返回注册失败。 |
| `DeactivateUserProfile` | Auth | 已停用返回成功；DB 不可用返回 `SERVICE_DEGRADED` | Auth 不应把 User 停用失败伪装成已完成注销。 |
| `BatchGetUserSimple` | Content、Comment、Notification、Ranking | 缺失用户进入 `missingUserIds`；DB 不可用返回 `SERVICE_DEGRADED` | 查询展示可由 consumer application 使用占位摘要；adapter 不得伪造 `UserSimple`。 |
| `BatchGetUserAvailability` | Comment、Message | 用户不存在或非 `ACTIVE` 返回不可用；DB 不可用返回 `SERVICE_DEGRADED` | 写路径 guard 必须 fail closed，不允许新增评论、私信等互动。 |
| `BatchCheckBlocked` | Comment、Message | 任一用户缺失可返回 `blocked=false`；DB 不可用返回 `SERVICE_DEGRADED` | 写路径 guard 在 degraded 时 fail closed，不能当成未拉黑。 |
| `CheckFollowing` | Message、Notification | 任一用户缺失返回 `false`；DB 不可用返回 `SERVICE_DEGRADED` | Message 可按业务拒绝陌生互动；Notification fanout 应 retry / DLQ 或跳过本次任务。 |
| `GetStrangerMessageSetting` | Message | 用户不存在或设置缺失返回 `false`；DB 不可用返回 `SERVICE_DEGRADED` | Message 在 degraded 时 fail closed，不允许把不可确认设置当成允许。 |
| `ListFollowerShard` | Notification | cursor 合法时返回稳定 shard；DB 不可用返回 `SERVICE_DEGRADED` | fanout job retry / DLQ；不能把空 shard 当成 fanout 成功。 |

服务间 endpoint 必须要求可信 `X-Caller-Service` / `X-Caller-Operation`。缺少 caller identity、operation 非白名单或 caller 超配额时，按内部认证 / 限流错误处理，不进入匿名公开查询分支。

## API 路径降级规则

| API / 能力 | 允许降级 | 禁止降级 |
| --- | --- | --- |
| `GetMe`、公开 profile、UserSimple 查询 | Redis cache 不可用时回源 DB；头像 URL 解析失败时按 schema 省略 / 置空 URL 并记录 degraded | 不得把 DB 查询失败伪装成 404、空 profile 或空 `missingUserIds`。 |
| `UpdateProfile` | 旧头像不自动删除；缓存删除失败 best-effort 并记录 | 头像校验不可确认时不得写入 `avatarFileId`；nickname 冲突不得 retry 成另一个昵称。 |
| Follow / Block 命令 | 重复关注、重复取关、重复拉黑、重复解除拉黑按幂等规则收敛 | DB、限流或关键 guard 不可确认时不得创建 / 删除关系事实。 |
| 关系列表和统计查询 | Redis cache 不可用时回源 DB；统计可由后续 repair 重建 | 不得把 DB 失败伪装成空列表、未关注、未拉黑或 0 统计。 |
| Admin 资料命令 | 缓存删除、事件投递走 outbox / retry | 限流、DB 或状态 guard 不可确认时不得返回管理命令成功。 |
| Check-in | Redis bitmap 失败可回源 PG 或返回 degraded，具体由后续 schema 固定 | PG 写入失败时不得返回签到成功。 |

## 配置准入

首次实现前，User runtime 配置至少覆盖：

- HTTP server read、write、idle、header 和 shutdown timeout。
- PostgreSQL query timeout、transaction timeout、pool size。
- Redis dial/read/write timeout、rate limit fallback window、本机 fallback 容量。
- File service base URL、timeout、retry、breaker、max-in-flight。
- 可选 Auth account verify client base URL、timeout、retry、breaker、max-in-flight。
- RabbitMQ publish confirm timeout、dispatcher concurrency、retry backoff、dead threshold。
- 每个 circuit breaker 的统计窗口、最小请求数、失败率阈值、连续失败阈值、打开时长和半开探测数。
- typed client provider 的 caller service 白名单、operation 白名单和内部调用配额。

配置加载、默认值和校验必须遵守 `docs/architecture/configuration.md`：handler、repository、adapter 和普通构造函数不得读取环境变量。

## 测试准入

首次实现相关切片时至少覆盖：

- 下游 timeout / circuit open 映射到 application 语义错误，不透出底层错误文本。
- 只读 client 的 retry 不超过配置次数，最终业务结果只计一次业务失败。
- 非幂等写路径不 retry；具备幂等保障的写路径重复执行不会制造重复事实。
- adapter 不构造业务 DTO 做降级，降级选择发生在 application。
- breaker key 按 `provider + operation` 分开，头像 URL 解析失败不熔断头像写入校验。
- 缓存不可用时查询受控回源 DB；DB 不可用时不伪装成空结果。
- typed client 中缺失用户和 `SERVICE_DEGRADED` 分别可测，写路径 guard 在 degraded 时 fail closed。
- Redis rate limit 不可用时，`rate-limiting.md` 中 fail-open / fail-closed / local fallback 分支分别可测。
