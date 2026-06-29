# Content 运行期 Resilience 设计

本文记录 `zhicore-content` 首次实现时必须落地的 timeout、retry、circuit breaker、max-in-flight 和降级策略。全局规则见 `docs/architecture/runtime-operations.md`；本文只补 Content 的 provider / operation 级矩阵。

当前状态：本文是设计事实源，不表示 Go 代码已经实现。首次实现任一依赖调用或 worker 前，必须为对应行补配置项、adapter 行为测试或 application 编排测试。

## 总原则

- resilience policy 在 runtime wiring 声明，不能散落到 handler、application 或 repository。
- 统计维度使用 `provider + operation`。除非依赖同一容量和失败域，否则不使用 provider 级全局熔断。
- timeout、retry、breaker、max-in-flight、fallback 窗口和并发数必须配置化，默认值只能作为本地开发和首轮压测基线。
- 写路径只有具备幂等保障时才允许 retry；没有幂等键、唯一约束或状态机条件更新时不重试。
- 降级策略由 application 选择；adapter 只返回语义错误、观测字段和 dependency status，不构造业务 DTO 伪装成功。
- 熔断打开、降级执行、Redis 本机兜底和 retry 最终失败必须可观测，字段规则见 `docs/architecture/observability.md`。

## Provider / Operation 矩阵

| Provider | Operation | 调用方 / 场景 | Timeout 基线 | Retry | Circuit breaker key | Max in-flight | 降级策略 | 幂等 / 一致性 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `postgres` | `post.command_tx` | 创建草稿、保存元数据、发布、删除、恢复、标签维护 | `1s..3s` | 不在事务外盲重试 | `postgres.post.command_tx` | 按 DB pool 配置保护 | `fail-fast` -> `1004` 或业务错误 | 事务内用乐观锁、状态条件和唯一约束保证幂等 / 冲突语义。 |
| `postgres` | `post.query` | 公开列表、详情元数据、作者工作台、管理查询 | `1s..3s` | 可对连接抖动最多 2 次总尝试 | `postgres.post.query` | 按查询族限制 | 查询失败返回 `1004`，不伪装为空列表 | cursor / page 查询必须稳定排序。 |
| `postgres` | `engagement.query` | Redis miss / Redis 故障后的点赞收藏状态批量回源 | `300ms..1s` | 不对单个请求盲重试；可由上层重新发起完整查询 | `postgres.engagement.query` | 独立小并发池，不能占满 post query | 单篇 viewer 状态不可确认时返回 unknown；整体不可用返回 `1004` | 必须批量查询 `(user_id, post_ids)`，禁止循环逐条 `EXISTS`。 |
| `mongo` | `body.write_draft` | 保存草稿正文 copy-on-write | `3s..5s` | 仅当 body id / content hash 已稳定且写入幂等时最多 2 次 | `mongo.body.write_draft` | 限制 autosave 写入并发 | 失败返回 `1004`，PG 指针不变 | 新 body 未被 PG 引用时由 cleanup task 清理。 |
| `mongo` | `body.write_snapshot` | 发布前写 published snapshot | `3s..5s` | 仅当 snapshot body id 稳定且写入幂等时最多 2 次 | `mongo.body.write_snapshot` | 发布路径单独限并发 | 失败返回 `1004`，线上 `published_*` 不变 | PG commit 前 snapshot 只是候选正文。 |
| `mongo` | `body.read_published` | 详情页、Search 拉正文 | `2s..5s` | 可重试最多 2 次总尝试 | `mongo.body.read_published` | 按公开读 / 内部调用分别限并发 | 普通查询返回 `CONTENT_BODY_UNAVAILABLE` / `1004`；Search consumer retry / DLQ | miss 创建 repair task，不读 draft 冒充 published。 |
| `redis` | `rate_limit.check` | Content 业务限流 | `100ms..300ms` | 不重试放大延迟；可短本机 fallback | `redis.rate_limit.check` | 独立于缓存连接池 | 按 `rate-limiting.md` 决定 `ALLOW` / `1003` / `1004` / no-op | 高副作用路径不能 fail-open。 |
| `redis` | `post.cache` | 文章详情、列表、标签缓存 | `100ms..300ms` | 不阻塞主查询重试 | `redis.post.cache` | cache 操作独立限并发 | cache miss / cache error 后按 DB / Mongo 回源限流；写缓存失败 best-effort | cache 不是事实源。 |
| `redis` | `engagement.cache` | 点赞 / 收藏状态和计数缓存 | `100ms..300ms` | 不阻塞主查询重试 | `redis.engagement.cache` | 与 post cache 独立限并发 | 读失败后按 `engagement-design.md` 进入受控 DB fallback；写失败 best-effort | Redis 不是事实源，unknown 不能伪装成 false。 |
| `redis` | `reader_presence` | heartbeat、leave、presence 查询 | `100ms..300ms` | 不重试阻塞用户路径 | `redis.reader_presence` | 热 post 独立限并发 | Redis 不可用时返回空 presence / 空成功并记录 degraded | presence 是附加能力，不能影响正文读取。 |
| `user-service` | `profile.get_summary` | 创建草稿作者快照、作者快照修复 | `2s..5s` | 只读调用最多 2 次总尝试 | `user.profile.get_summary` | 按 actor / worker 分别限并发 | 创建草稿不能伪造作者信息；返回 `1004` 或进入补偿任务 | 作者快照不是 User 事实源，刷新按版本比较。 |
| `upload-service` | `file.validate_ref` | 保存 / 发布封面和正文媒体引用校验 | `2s..5s` | 只读校验最多 2 次总尝试 | `upload.file.validate_ref` | 按发布 / 保存路径限并发 | 发布时校验不可用返回 `1004`；保存草稿可按实现切片选择暂存待校验状态 | 不能把不可确认文件引用发布成线上事实。 |
| `upload-service` | `file.resolve_url` | 详情 / 列表派生封面、头像展示 URL | `1s..2s` | 可重试最多 2 次总尝试 | `upload.file.resolve_url` | 查询路径限并发 | contract 允许时可留空 URL，并记录 degraded；不得影响文章事实字段 | URL 是派生展示字段，不作为 Content 持久化事实。 |
| `rabbitmq` | `outbox.publish` | outbox dispatcher 投递 Content 集成事件 | publish confirm timeout `2s..5s` | 按 outbox retry + backoff + jitter | `rabbitmq.outbox.publish` | dispatcher worker 并发配置 | 失败更新 outbox retry metadata；超过阈值进入 dead / admin retry | `event_id` 唯一，consumer 按 event id 幂等。 |

## API 路径降级规则

| API / 能力 | 允许降级 | 禁止降级 |
| --- | --- | --- |
| 公开列表、标签、详情元数据 | 可返回 `1004`；cache 不可用可回源，但必须受限流保护 | 不得把 DB / Content 查询失败伪装为空列表。 |
| published body 读取 | Search consumer 可 retry / DLQ；普通查询可返回 `CONTENT_BODY_UNAVAILABLE` 或 `1004` | 不得读取 draft、旧 snapshot 或空 body 冒充 published body。 |
| 草稿保存 | Mongo / PG 任一失败返回失败；PG 未切指针时旧草稿保持有效 | 不得在 PG 失败后认为新 Mongo body 已成为当前草稿。 |
| 发布生命周期 | 核心依赖失败返回失败；outbox 写入必须在 PG 事务内完成 | 不得在正文不可读、文件引用不可确认或限流不可确认时发布成功。 |
| 点赞 / 收藏 | Redis 缓存失败可 best-effort 跳过；PostgreSQL 事务失败返回失败 | 不得只写缓存不写事实表；不得因幂等就无限放行刷写。 |
| Engagement 状态查询 | Redis 不可用时可短时受控 DB fallback；状态不可确认时返回 unknown / degraded | 不得无界 DB 回源，不得把查询失败伪装成未点赞或未收藏。 |
| Reader presence | Redis 不可用时返回空 presence / 空成功 | 不得阻断文章详情、正文读取或公开列表。 |
| 管理 outbox retry | RabbitMQ 不可用时记录 retry 失败或保持 dead / failed | 不得在限流不可确认时放行高频 retry。 |

## 配置准入

首次实现前，Content runtime 配置至少覆盖：

- `postgres` query timeout、transaction timeout、pool size。
- `mongo` read / write timeout、pool size、body write max-in-flight。
- `redis` dial/read/write timeout、rate limit fallback window、本机 fallback 容量。
- Engagement cache timeout、DB fallback timeout、fallback max-in-flight、fallback 本机预算和 batch `postIds` 上限。
- `user-service`、`upload-service` base URL、timeout、retry、breaker、max-in-flight。
- `rabbitmq` publish confirm timeout、dispatcher concurrency、retry backoff、dead threshold。
- 每个 circuit breaker 的统计窗口、最小请求数、失败率阈值、连续失败阈值、打开时长和半开探测数。

配置加载、默认值和校验必须遵守 `docs/architecture/configuration.md`：handler、repository、adapter 和普通构造函数不得读取环境变量。

## 测试准入

首次实现相关切片时至少覆盖：

- 下游 timeout / circuit open 映射到 application 语义错误，不透出底层错误文本。
- 只读 client 的 retry 不超过配置次数，最终业务结果只计一次业务失败。
- 非幂等写路径不 retry；具备幂等保障的写路径重复执行不会制造重复事实。
- adapter 不构造业务 DTO 做降级，降级选择发生在 application。
- breaker key 按 `provider + operation` 分开，Search 拉正文失败不熔断普通摘要查询。
- Redis rate limit 不可用时，`rate-limiting.md` 中 fail-open / fail-closed / no-op 分支分别可测。
