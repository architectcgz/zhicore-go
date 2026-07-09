# Content Rate Limiting 后续实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；本计划步骤使用 checkbox 追踪。提交前必须先使用 @committing-changes。

**目标：** 将 `docs/architecture/services/content/rate-limiting.md` 中尚未落地的限流、降级、观测和 resilience 工作拆成可独立验证、可独立提交的小任务。

**架构：** `cmd/server` 只负责加载和校验配置，运行期装配归 `internal/content/runtime`，业务决策归 `internal/content/application`，Redis / PostgreSQL / cache adapter 只返回语义结果。所有新增限流字段、fallback 窗口、resilience policy 和观测标签都必须保持低基数、可配置、可测试，不把 Redis 故障期间的本机预算伪装成分布式配额。

**技术栈：** Go 1.22、`services/zhicore-content` Go module、`libs/kit/observability`、Redis fixed-window adapter、PostgreSQL engagement repository、Content HTTP handler tests、`make check`。

---

## 计划性质

这是探索性拆分计划。执行前如果要绑定正式任务、review gate 或分支策略，应提升到 `docs/plan/impl-plan/` 并补齐任务 slug、执行人和正式 review 入口。

## 背景依据

| 文档 / 代码 | 需要遵守的事实 |
| --- | --- |
| `docs/architecture/services/content/rate-limiting.md` | 当前只落地 `limit/window/fallback/fallbackWindow/failClosed`；尚缺 burst、冷却窗口、单位时间 body 字节量、presence empty fallback、完整 engagement fallback、真实 metrics exporter 和全量 resilience policy。 |
| `docs/architecture/services/content/runtime-resilience.md` | provider + operation 维度声明 timeout、retry、breaker key、max-in-flight 和 fallback；高副作用路径不能 fail-open。 |
| `docs/architecture/services/content/engagement-design.md` | Redis 故障时 engagement 查询只能走受控 DB fallback；unknown 必须表达为 `null + degraded=true`，不能当成 `false`。 |
| `docs/architecture/observability.md` | metrics label 只能使用低基数字段；禁止 token、cookie、Authorization、IP、完整 URL、原始标题、摘要、正文和用户输入文本。 |
| `docs/architecture/configuration.md` | handler、repository、adapter 和普通构造函数不得读取环境变量；配置加载、默认值和校验必须在启动路径完成。 |
| `docs/architecture/testing.md` | 限流、降级、并发、worker、contract 和 bugfix 属于 R3/R4，优先写 focused test 或回归测试。 |

## 当前状态

| 能力 | 状态 |
| --- | --- |
| `RateLimiter` / `ContentObserver` 端口 | 已落地。 |
| Redis fixed-window limiter | 已落地，支持 `fallbackWindow` 内 `local_memory` / `gateway_only` 短时降级。 |
| `cmd/server` 限流环境变量 | 已落地 7 类规则的 `LIMIT`、`WINDOW`、`FALLBACK`、`FALLBACK_WINDOW`、`FAIL_CLOSED`。 |
| HTTP `1003 / 429` 与 `1004 / 503` 映射基础 | 已有通用映射，但各 API 家族覆盖仍需补齐。 |
| 真实 metrics exporter / observer adapter | 未落地，目前只有 noop observer 和测试 fake。 |
| burst、cooldown、body bytes budget | 未落地。 |
| engagement 受控 DB fallback | 部分落地：有 cache miss 后批量 DB 查询和 unknown 表达，但缺 fallback limiter、max-in-flight、breaker / timeout policy。 |
| presence no-op / empty fallback | 当前没有明确 API owner，不应在无 owner 情况下实现隐藏兼容。 |

## 任务总览

| 顺序 | 任务 | 类型 | 建议提交边界 |
| --- | --- | --- | --- |
| 1 | 限流观测 adapter 和低基数 metrics 原语 | TDD，R3 | `feat(content): 接入限流观测适配器` |
| 2 | runtime resilience policy 配置骨架 | TDD，R3/R4 | `feat(content): 增加运行韧性策略配置` |
| 3 | API 家族限流调用覆盖 | TDD，R4 | `fix(content): 补齐内容接口限流守卫` |
| 4 | draft body 的 burst、cooldown 和 body bytes budget | TDD，R4 | `feat(content): 增强草稿正文限流预算` |
| 5 | engagement 受控 DB fallback 收口 | TDD，R4 | `feat(content): 收口互动查询降级预算` |
| 6 | admin outbox retry 冷却窗口 | TDD，R4 | `feat(content): 增加管理重试冷却限流` |
| 7 | presence empty fallback 明确延期并清理状态漂移 | 文档 / 小清理，R0/R1 | `docs(content): 明确 presence 限流延期边界` |

## 文件职责图

| 路径 | 职责 |
| --- | --- |
| `services/zhicore-content/internal/content/ports/rate_limit.go` | 限流请求、决策、fallback 和 observer 端口契约。 |
| `services/zhicore-content/internal/content/application/rate_limit.go` | 限流 decision 到 application 错误语义的映射；不得构造 HTTP response。 |
| `services/zhicore-content/internal/content/application/*` | 各 use case 在执行业务副作用前选择正确 `RateLimitType`、resource、operation 和扩展预算输入。 |
| `services/zhicore-content/internal/content/infrastructure/redis/rate_limiter.go` | Redis / local fallback 的限流 adapter；只返回 typed decision，不读 env，不做 HTTP 映射。 |
| `services/zhicore-content/internal/content/runtime/*` | runtime-owned 配置类型、默认值、Redis dependency、observer adapter 和 resilience policy 组装。 |
| `services/zhicore-content/cmd/server/config*.go` | 环境变量加载、默认值覆盖、格式校验和脱敏摘要；只依赖 runtime 暴露类型。 |
| `services/zhicore-content/configs/local.example.env` | 本地非敏感配置模板，必须列出新增限流和 resilience 配置。 |
| `libs/kit/observability/*` | 跨服务稳定 metrics recorder / no-op recorder / 低基数标签原语；不得包含 Content 私有 operation。 |

## 任务 1：限流观测 adapter 和低基数 metrics 原语

**测试立场：** TDD - observer 字段、metrics label 和脱敏边界属于运行期契约，先写 focused test。

**文件：**

- 创建：`libs/kit/observability/metrics.go`
- 创建：`libs/kit/observability/metrics_test.go`
- 创建：`services/zhicore-content/internal/content/runtime/observer.go`
- 创建：`services/zhicore-content/internal/content/runtime/observer_test.go`
- 修改：`services/zhicore-content/internal/content/runtime/rate_limit.go`
- 修改：`services/zhicore-content/cmd/server/runtime_deps.go`

**验收清单：**

- [ ] `ContentObserver` 每次限流 decision 记录 `operation`、`route` 或 use case operation、`limitType`、`reason`、`outcome`、`fallback`、`status`，字段值全部为稳定枚举或模板。
- [ ] metrics label 不包含原始 `postId`、`userId`、IP、完整 URL、raw title、summary、正文、token、cookie、Authorization header 或错误文本。
- [ ] Redis unavailable、本机 fallback allow、fallback window exceeded、fixed-window reject 都有可聚合计数。
- [ ] observer 失败或 no-op recorder 不改变业务控制流。
- [ ] `libs/kit/observability` 只提供 recorder 原语和 label 校验，不登记 Content 私有 operation。

- [ ] **步骤 1：写 `libs/kit/observability` metrics recorder 测试**

  运行：`cd libs/kit && go test ./observability -run TestMetrics -count=1`

  预期：新增测试先失败，失败点指向缺少 recorder 或低基数 label 校验。

- [ ] **步骤 2：实现 recorder interface、no-op recorder 和低基数 label 校验**

  保持 API 小而稳定；不引入 Prometheus 具体 exporter，先固定跨服务语义。

- [ ] **步骤 3：写 Content runtime observer 测试**

  运行：`cd services/zhicore-content && go test ./internal/content/runtime -run TestContentObserver -count=1`

  预期：先失败，证明 observer 尚未把 decision 转成低基数字段。

- [ ] **步骤 4：实现 Content observer adapter 并替换 runtime noop 默认装配**

  `cmd/server/runtime_deps.go` 只能调用 runtime factory，不能直接 import application ports。

- [ ] **步骤 5：运行窄验证**

  运行：`cd libs/kit && go test ./observability -count=1`

  预期：通过。

  运行：`cd services/zhicore-content && go test ./internal/content/runtime ./internal/content/application -count=1`

  预期：通过。

## 任务 2：runtime resilience policy 配置骨架

**测试立场：** TDD - 配置加载、默认值、非法值和 owner 边界属于 R3；限流和 DB fallback 的 max-in-flight / timeout 属于 R4。

**文件：**

- 创建：`services/zhicore-content/internal/content/runtime/resilience.go`
- 创建：`services/zhicore-content/internal/content/runtime/resilience_test.go`
- 修改：`services/zhicore-content/cmd/server/config.go`
- 修改：`services/zhicore-content/cmd/server/config_defaults.go`
- 修改：`services/zhicore-content/cmd/server/config_loader.go`
- 修改：`services/zhicore-content/cmd/server/config_validation.go`
- 修改：`services/zhicore-content/cmd/server/config_test.go`
- 修改：`services/zhicore-content/configs/local.example.env`
- 修改：`docs/architecture/services/content/runtime-resilience.md`

**验收清单：**

- [ ] policy key 使用 `provider + operation`，首批至少覆盖 `redis.rate_limit.check`、`postgres.engagement.query`、`redis.engagement.cache`、`mongo.body.write_draft`、`rabbitmq.outbox.publish`。
- [ ] 每个 policy 显式包含 `timeout`、`retry max attempts`、`breaker key`、`max-in-flight`、`degradeStrategy` 和 `idempotency`。
- [ ] handler、application、repository、adapter 和普通构造函数不读取环境变量。
- [ ] duration 必须是 Go duration 字符串，布尔必须是 `true` / `false`，非正数和 overflow fail fast。
- [ ] 生产密钥或 DSN 不进入配置摘要；新增摘要只输出非敏感 policy 值。
- [ ] 不把 `redis.rate_limit.check` 的 fallback window 从 rate limit rule 里搬走；policy 只管理调用超时、breaker 和并发保护。

- [ ] **步骤 1：写 runtime policy 默认值和校验测试**

  运行：`cd services/zhicore-content && go test ./internal/content/runtime -run TestResiliencePolicy -count=1`

  预期：先失败，缺少 policy 类型或默认值。

- [ ] **步骤 2：实现 runtime-owned policy 类型和默认值**

  不把 policy 类型放进 `cmd/server`，避免进程根拥有业务运行规则。

- [ ] **步骤 3：写 `cmd/server` 配置加载测试**

  覆盖合法 env、非法 duration、非法 int、缺少必填 policy 时的错误信息。

- [ ] **步骤 4：实现 env overlay、校验和 `local.example.env` 示例**

  新增 env 名使用 `ZHICORE_CONTENT_RESILIENCE_<PROVIDER>_<OPERATION>_<FIELD>` 形态，`OPERATION` 使用大写下划线，例如 `RATE_LIMIT_CHECK`。

- [ ] **步骤 5：同步 runtime-resilience 当前状态**

  只把本任务真实落地的 policy config 标为已落地；不要宣称 adapter 已实际使用所有 policy。

- [ ] **步骤 6：运行窄验证**

  运行：`cd services/zhicore-content && go test ./cmd/server ./internal/content/runtime -count=1`

  预期：通过。

## 任务 3：API 家族限流调用覆盖

**测试立场：** TDD - 高副作用写路径 fail-closed 和公开错误映射属于 R4，先补 application / handler 回归测试。

**文件：**

- 修改：`services/zhicore-content/internal/content/application/engagement.go`
- 修改：`services/zhicore-content/internal/content/application/engagement_test.go`
- 修改：`services/zhicore-content/internal/content/application/published_body.go`
- 修改：`services/zhicore-content/internal/content/application/get_published_body_test.go`
- 修改：`services/zhicore-content/internal/content/application/admin_outbox_test.go`
- 修改：`services/zhicore-content/api/http/engagement_handler_test.go`
- 修改：`services/zhicore-content/api/http/get_post_body_handler_test.go`
- 修改：`services/zhicore-content/api/http/admin_outbox_handler_test.go`

**验收清单：**

- [ ] 点赞、取消点赞、收藏、取消收藏在写事务前调用 `RateLimitTypeEngagementWrite`，resource 使用 `postId`，operation 使用稳定枚举。
- [ ] `GetPostEngagement` 和 `BatchGetEngagementStatus` 在查询 viewer 状态前调用 `RateLimitTypeEngagementRead`。
- [ ] Search / Ranking 等可信内部拉正文路径使用 `RateLimitTypeInternalClient`；缺少可信 caller identity 的服务间-only 调用不能落到匿名公开配额。
- [ ] 发布、定时发布、删除、恢复、admin 删除和 outbox retry 在 `DEGRADED_DENY_UNAVAILABLE` 时不执行 use case 副作用。
- [ ] `REJECT_TOO_FREQUENT` 映射为 HTTP `429` + body `1003`，`DEGRADED_DENY_UNAVAILABLE` 映射为 HTTP `503` + body `1004`。
- [ ] observer 仍能收到每一次限流 decision。

- [ ] **步骤 1：写 engagement 写路径限流测试**

  运行：`cd services/zhicore-content && go test ./internal/content/application -run TestEngagement.*RateLimit -count=1`

  预期：先失败，当前 engagement 写路径未调用 limiter。

- [ ] **步骤 2：实现 engagement 写路径限流**

  确保 limiter 拒绝时不调用 `MutateEngagement`、outbox 或 stats task。

- [ ] **步骤 3：写 engagement 读路径限流测试**

  覆盖单篇和批量状态；`429` 不应被降级成 unknown。

- [ ] **步骤 4：实现 engagement 读路径限流**

  限流发生在 cache / DB 查询前；参数非法和未登录仍按既有校验顺序返回。

- [ ] **步骤 5：写内部 body read 限流测试**

  覆盖可信 caller 和缺少 caller identity 两个分支。

- [ ] **步骤 6：实现内部 body read 的 `internal_client` 配额选择**

  不改变公开读 API 的默认行为；只在已有输入或 handler 能可靠识别 caller 时切换。

- [ ] **步骤 7：补 handler 错误映射回归测试**

  运行：`cd services/zhicore-content && go test ./api/http -run 'Test(Engagement|GetPostBody|AdminOutbox).*RateLimit' -count=1`

  预期：通过。

## 任务 4：draft body 的 burst、cooldown 和 body bytes budget

**测试立场：** TDD - autosave 风暴和 MongoDB 写入保护属于 R4，先写限流 adapter 和 application 回归测试。

**文件：**

- 修改：`services/zhicore-content/internal/content/ports/rate_limit.go`
- 修改：`services/zhicore-content/internal/content/application/rate_limit.go`
- 修改：`services/zhicore-content/internal/content/application/save_draft_body.go`
- 修改：`services/zhicore-content/internal/content/application/save_draft_body_test.go`
- 修改：`services/zhicore-content/internal/content/infrastructure/redis/rate_limiter.go`
- 修改：`services/zhicore-content/internal/content/infrastructure/redis/rate_limiter_test.go`
- 修改：`services/zhicore-content/internal/content/runtime/rate_limit.go`
- 修改：`services/zhicore-content/internal/content/runtime/rate_limit_test.go`
- 修改：`services/zhicore-content/cmd/server/config_loader.go`
- 修改：`services/zhicore-content/cmd/server/config_validation.go`
- 修改：`services/zhicore-content/cmd/server/config_test.go`
- 修改：`services/zhicore-content/configs/local.example.env`
- 修改：`docs/architecture/services/content/rate-limiting.md`

**验收清单：**

- [ ] 现有 `LIMIT`、`WINDOW`、`FALLBACK`、`FALLBACK_WINDOW`、`FAIL_CLOSED` env 名和语义不变。
- [ ] 新增 burst、cooldown 和 body bytes 配置必须可选；未配置时保持现有 fixed-window 行为。
- [ ] `PUT /draft/body` 的 limiter request 携带 body size 或等价成本输入；成本不来自 raw body 文本。
- [ ] body bytes budget 超限返回 `REJECT_TOO_FREQUENT` / `1003`，不会写 MongoDB。
- [ ] Redis 不可用时仍受 `fallbackWindow` 限制；多实例下本机预算不被描述为全局配额。
- [ ] Redis key 和 metrics label 只包含 hash / 低基数字段，不包含正文 blocks。
- [ ] cooldown 适用于同一 actor + post + operation 的重复 autosave 风暴；正常编辑 burst 不被单次保存永久锁死。

- [ ] **步骤 1：写 Redis adapter burst / cooldown / bytes budget 测试**

  运行：`cd services/zhicore-content && go test ./internal/content/infrastructure/redis -run TestFixedWindowRateLimiter.*Budget -count=1`

  预期：先失败，当前 adapter 没有这些预算。

- [ ] **步骤 2：扩展 rule config 和 adapter**

  保持默认 rule 完全兼容；新增字段只在配置大于零时生效。

- [ ] **步骤 3：写 `cmd/server` 新 env 覆盖和非法值测试**

  建议 env 后缀：`_BURST_LIMIT`、`_BURST_WINDOW`、`_COOLDOWN`、`_BODY_BYTES_LIMIT`、`_BODY_BYTES_WINDOW`。

- [ ] **步骤 4：实现配置加载、runtime 映射和模板示例**

  新增字段由 runtime 暴露类型拥有，`cmd/server` 不 import `ports`。

- [ ] **步骤 5：写 `SaveDraftBody` application 测试**

  覆盖 body bytes 超限、cooldown 命中、Redis fail-closed 时不写 body store。

- [ ] **步骤 6：实现 `SaveDraftBody` 的成本输入和文档同步**

  只同步已落地字段，不把 engagement fallback 或 presence 写成已完成。

- [ ] **步骤 7：运行窄验证**

  运行：`cd services/zhicore-content && go test ./cmd/server ./internal/content/application ./internal/content/infrastructure/redis ./internal/content/runtime -count=1`

  预期：通过。

## 任务 5：engagement 受控 DB fallback 收口

**测试立场：** TDD - unknown 语义、max-in-flight、DB fallback 和批量查询属于 R4，必须先补回归测试。

**文件：**

- 修改：`services/zhicore-content/internal/content/application/engagement.go`
- 修改：`services/zhicore-content/internal/content/application/engagement_test.go`
- 修改：`services/zhicore-content/internal/content/ports/engagement.go`
- 修改：`services/zhicore-content/internal/content/infrastructure/postgres/engagement.go`
- 修改：`services/zhicore-content/internal/content/infrastructure/postgres/engagement_test.go`
- 修改：`services/zhicore-content/internal/content/infrastructure/redis/engagement_cache.go`
- 修改：`services/zhicore-content/internal/content/infrastructure/redis/engagement_cache_test.go`
- 修改：`services/zhicore-content/internal/content/runtime/module.go`
- 修改：`services/zhicore-content/internal/content/runtime/module_test.go`
- 修改：`docs/architecture/services/content/engagement-design.md`
- 修改：`docs/architecture/services/content/rate-limiting.md`

**验收清单：**

- [ ] Redis engagement cache error 不再无条件触发 DB fallback；必须同时满足本机 fallback limiter、`postgres.engagement.query` policy、max-in-flight 和 request context。
- [ ] 单篇 viewer 状态不可确认返回 `viewer.liked=null`、`viewer.favorited=null`、`viewer.degraded=true`，文章 stats 可继续返回。
- [ ] 批量状态部分不可确认时只标记对应 item `degraded=true`；整体 DB breaker open 或 max-in-flight 耗尽时返回 `1004`。
- [ ] batch repository 只允许一次批量 SQL；禁止循环逐条 `EXISTS(user_id, post_id)`。
- [ ] Redis miss 的确定状态可以回填 cache；unknown / degraded 不写入 Redis。
- [ ] DB fallback 指标区分 `success`、`partial_unknown`、`budget_exhausted`、`breaker_open`、`timeout`。

- [ ] **步骤 1：写 application fallback gate 测试**

  运行：`cd services/zhicore-content && go test ./internal/content/application -run TestEngagement.*Fallback -count=1`

  预期：先失败，当前 `readViewerStatus` 没有 fallback limiter / max-in-flight gate。

- [ ] **步骤 2：引入 fallback gate 端口或 runtime 注入对象**

  gate 只表达是否允许 fallback 和观测字段；不在 repository 里做业务降级判断。

- [ ] **步骤 3：写 max-in-flight 和 context deadline 测试**

  使用 fake gate 或可控 semaphore，不用 `time.Sleep` 做同步。

- [ ] **步骤 4：实现 application 受控 DB fallback**

  保持 unknown 三值语义；不要让 `false` 代表未知。

- [ ] **步骤 5：写 repository 批量 SQL 回归测试**

  验证 `BatchGetViewerStatus` 是批量查询；如果测试只能在 fake DB 层证明，就固定一次调用和 `postIDs` 参数。

- [ ] **步骤 6：同步 docs 状态**

  `engagement-design.md` 和 `rate-limiting.md` 只标记已实现的 fallback gate，不提前声明完整 breaker exporter。

- [ ] **步骤 7：运行窄验证**

  运行：`cd services/zhicore-content && go test ./internal/content/application ./internal/content/infrastructure/postgres ./internal/content/infrastructure/redis ./internal/content/runtime -count=1`

  预期：通过。

## 任务 6：admin outbox retry 冷却窗口

**测试立场：** TDD - 管理命令高副作用、审计和重复 retry 防护属于 R4，先写 application 和 repository 回归测试。

**文件：**

- 修改：`services/zhicore-content/internal/content/application/admin_outbox.go`
- 修改：`services/zhicore-content/internal/content/application/admin_outbox_test.go`
- 修改：`services/zhicore-content/internal/content/ports/outbox.go`
- 修改：`services/zhicore-content/internal/content/infrastructure/postgres/outbox_admin.go`
- 修改：`services/zhicore-content/internal/content/infrastructure/postgres/outbox_admin_test.go`
- 修改：`services/zhicore-content/internal/content/infrastructure/postgres/sql/retry_admin_outbox_event.sql`
- 修改：`services/zhicore-content/internal/content/runtime/rate_limit.go`
- 修改：`services/zhicore-content/cmd/server/config_test.go`
- 修改：`services/zhicore-content/configs/local.example.env`
- 修改：`docs/architecture/services/content/rate-limiting.md`

**验收清单：**

- [ ] 同一 admin actor + `eventId` + `retry_admin_outbox_event` 在冷却窗口内不能重复放行。
- [ ] Redis 限流不可确认时 admin retry fail closed，返回 `1004`，不更新 outbox 状态，不写 retry 审计。
- [ ] 冷却命中返回 `1003`，不是参数错误、权限错误或资源不存在。
- [ ] repository 层仍用状态条件更新，只有 `FAILED` / `DEAD` 等可 retry 状态能回到 `PENDING`。
- [ ] retry reason 继续必填并保持现有脱敏规则；不得把 RabbitMQ URL / DSN 之类明文写入响应。

- [ ] **步骤 1：写 admin retry cooldown application 测试**

  运行：`cd services/zhicore-content && go test ./internal/content/application -run TestAdminOutbox.*Cooldown -count=1`

  预期：先失败，当前只有普通 `admin_command` fixed-window，没有同一 event 冷却语义。

- [ ] **步骤 2：扩展 retry command 限流请求或 cooldown 配置**

  保持 admin list 查询不受 retry cooldown 影响。

- [ ] **步骤 3：写 repository 状态条件测试**

  覆盖非 retry 状态、缺失 event、重复 retry。

- [ ] **步骤 4：实现 SQL 和 repository 行为**

  只在状态条件满足时更新，错误映射仍保留 `ErrOutboxEventNotFound` 或明确的不可 retry 错误。

- [ ] **步骤 5：补 handler 错误映射测试**

  运行：`cd services/zhicore-content && go test ./api/http -run TestAdminOutbox.*Retry -count=1`

  预期：通过。

## 任务 7：presence empty fallback 明确延期并清理状态漂移

**测试立场：** 文档 / 小清理。若只改文档属于 R0；若删除无 owner 枚举或死代码属于 R1，需要跑受影响 Go 测试。

**文件：**

- 修改：`docs/architecture/services/content/rate-limiting.md`
- 修改：`docs/architecture/services/content/README.md`
- 可选修改：`services/zhicore-content/internal/content/ports/rate_limit.go`
- 可选修改：`services/zhicore-content/internal/content/application/rate_limit.go`
- 可选修改：`services/zhicore-content/internal/content/application/rate_limit_test.go`

**验收清单：**

- [ ] 若 Content 当前没有 presence API owner，`presence_empty` 只能登记为延期目标，不新增环境变量、不新增隐式 fallback、不在生产路径返回 no-op success。
- [ ] 如果 `RateLimitOutcomeNoopSuccess` 没有生产 owner，删除或降级为明确 future note；不能保留一个看似支持但所有 use case 都不能安全处理的 outcome。
- [ ] `rate-limiting.md` 的“当前状态”必须和 Go 代码一致，不把延期能力写成已实现。
- [ ] 若未来重启 presence API，必须先补 HTTP contract、application owner、空响应语义和 handler tests，再接限流 fallback。

- [ ] **步骤 1：搜索 presence / noop 生产 owner**

  运行：`rg -n "presence|NoopSuccess|presence_empty|NOOP_SUCCESS" docs services/zhicore-content`

  预期：明确是否只有文档和测试引用。

- [ ] **步骤 2：按搜索结果更新文档或清理无 owner outcome**

  如果删除 Go 枚举，必须同步测试；如果只延期文档，不改生产代码。

- [ ] **步骤 3：运行对应验证**

  只改文档时运行：`bash scripts/check-structure.sh`

  改 Go 代码时额外运行：`cd services/zhicore-content && go test ./internal/content/application ./internal/content/ports -count=1`

  预期：通过。

## 每个切片的共同完成条件

- [ ] 对应 source docs 中的“已落地 / 未落地”状态同步完成。
- [ ] `services/zhicore-content/configs/local.example.env` 只包含非敏感示例值。
- [ ] 新增 env 不破坏既有 rate limit env 名。
- [ ] 运行最窄相关 `go test`，路径或索引变化时运行 `bash scripts/check-structure.sh`。
- [ ] Go 源码分层变化时运行 `python3 tests/architecture/check_boundaries.py --root .`。
- [ ] 交付前按风险决定是否运行 `make check`；跨 runtime / config / shared kit 的切片建议运行。
- [ ] 每个提交前使用 @committing-changes，提交信息使用“标题 + 正文”两段结构，不加无依据的 `Co-Authored-By`。

## 架构适配评估

- 本拆分保持 Content 分层：`cmd/server` 加载配置，`runtime` 组装依赖，`application` 持有降级选择，`infrastructure` 只实现 adapter 语义。
- 小任务按可独立验证的风险面拆分：观测、policy 配置、API guard、draft body 预算、engagement fallback、admin cooldown 和 presence 状态清理互相解耦。
- 结构性收敛没有被静默推迟：真实 metrics、resilience policy、fallback gate 和状态文档同步都拥有单独任务和验收标准。
- 当前不把所有 provider 的完整熔断实现一次性塞进一个大任务；先落 policy 类型和关键调用点，再逐步让 adapter 使用 policy。
- 多实例 `local_memory` 的语义保持明确：它只是每进程短降级预算，必须受 `fallbackWindow` 限制，不作为全局分布式限流。

## 主要风险

- 如果任务 2 的 policy 类型设计过宽，会提前把服务私有策略提升成共享抽象；实现时优先保持 Content runtime 本地类型，只有跨服务重复后再抽到 `libs/kit`。
- 如果任务 4 和任务 5 同时改 `RateLimitRequest`，容易产生冲突；建议先完成任务 3，再分别推进 draft body 预算和 engagement fallback。
- Engagement fallback 不能为了“用户体验平滑”把 unknown 变成 `false`；这会制造错误业务事实。
- Admin retry cooldown 同时涉及限流和 repository 状态条件，不能只做 rate limiter 而忽略 SQL 条件更新。
- presence 没有 API owner 时不要实现隐藏 fallback，否则后续会出现无法审查的兼容语义。
