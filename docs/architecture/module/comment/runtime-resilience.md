# Comment 运行韧性设计

本文固定 `zhicore-comment` 在微服务化后的 timeout、retry、熔断、降级、限流和依赖故障语义。通用运行期规则见 `docs/architecture/runtime-operations.md`；本文只记录 Comment 模块自己的业务取舍。

## 核心原则

- PostgreSQL 是评论、点赞、统计、rank 和 outbox 的真相源；PostgreSQL 不可用时 Comment 业务不可用。
- Redis 只用于缓存、限流和分布式锁 / claim 加速，不保存不可重建的评论事实。
- RabbitMQ 不是请求路径真相源；Comment 在本地事务内写 outbox，RabbitMQ 发布失败由 dispatcher 重试。
- 写路径前置 guard 不能降级放行：Content、User、User relation、File 校验不可确认时，不写入本地评论事实。
- 查询路径允许展示增强降级：作者摘要失败可返回占位作者，File URL 解析失败可省略 URL；但 `publicId` 只能来自 User 或本地已确认快照，文章可见性、评论存在性和登录态不能靠降级伪造。
- client adapter 只负责 timeout、retry、熔断和错误翻译；是否降级由 application use case 按本文矩阵决定。

## 请求时间预算

Comment handler 必须使用请求 `context.Context` 贯穿 application、repository、cache、MQ 和下游 client。

| 场景 | 建议总预算 | 说明 |
| --- | --- | --- |
| 创建 / 更新评论 | `2s` 到 `3s` | 外部 guard 必须在进入本地事务前完成；本地事务应尽量短。 |
| 删除评论 / 点赞 / 取消点赞 | `1s` 到 `2s` | 主要依赖 PostgreSQL，缓存失效和实时 publish 不阻塞主语义。 |
| 评论列表 / 详情查询 | `1s` 到 `2s` | PostgreSQL 查询和批量作者摘要 / URL 解析共享预算。 |
| Admin outbox retry / summary | `2s` 到 `5s` | 管理端可稍长，但必须分页、限批次。 |
| 后台 worker 单批次 | `5s` 到 `15s` | 按批次 claim，超时后释放或标记可重试。 |

下游 client 的单次 timeout 不得超过上游剩余 deadline。没有明确上游 deadline 时，runtime 必须为 HTTP handler 设置默认 request timeout。

## 下游依赖矩阵

| 依赖 / operation | 调用路径 | timeout | retry | 熔断 | 降级策略 |
| --- | --- | --- | --- | --- | --- |
| Content `CheckPostCommentable(postId)` | 创建 / 更新 / 公开读取前校验文章可见性；写路径返回 `internalId` 和 `postAuthorId` | `500ms` 到 `1s` | 只读 guard 可最多 2 次总尝试，带 jitter | 开启 | 写路径和可见性校验 fail closed，返回 `1004`；不基于旧页面状态放行。 |
| User `GetInteractivePrincipal` / 状态校验 | 创建 / 更新 / 点赞等新增或修改事实的写路径 | `500ms` 到 `1s` | 最多 2 次总尝试 | 开启 | fail closed，返回 `1004` 或认证/权限类错误；不能让禁用、封禁或不可互动用户继续写入。 |
| User relation `BatchCheckBlocked` | 创建根评论 / 回复 / 更新 / 点赞 | `500ms` 到 `1s` | 最多 2 次总尝试 | 开启 | fail closed，返回 `1004`；不能因为 relation 不可用绕过拉黑边界。 |
| User `BatchGetUserSimple` | 列表 / 详情作者摘要 | `500ms` 到 `1s` | 最多 2 次总尝试 | 开启 | 可降级为 `displayName="Unknown user"`、`unavailable=true`；`publicId` 有 User 返回或本地快照时必须保留，缺失时省略且不得伪造。 |
| File `ValidateFileReferences` | 创建 / 更新评论媒体 guard | `500ms` 到 `1s` | 最多 2 次总尝试 | 开启 | fail closed，返回 `1004` 或媒体校验错误；不能保存未确认文件引用。 |
| File `BatchResolveFileURLs` | 列表 / 详情展示 URL | `300ms` 到 `800ms` | 最多 2 次总尝试 | 开启 | 可省略 `imageUrls` / `voiceUrl`，保留 `imageFileIds` / `voiceFileId`。 |
| Ranking `GetHotPostCandidates` | 首页评论缓存 / 热门候选同步 job | `1s` 到 `2s` | job 内按退避重试 | 开启 | 不影响普通评论写读；使用上一批候选或跳过本轮。 |
| PostgreSQL | 所有本地事实读写 | 查询 `1s` 到 `3s` | 请求路径不盲重试写事务 | 不适用 | 不可降级；返回 `1004` / `503`，readiness 非 ready。 |
| Redis cache | 列表、详情、点赞状态、首页缓存 | `50ms` 到 `200ms` | 不重试或最多一次快速重试 | 开启 / 记录状态 | 读路径 miss / Redis 不可用时回源 PostgreSQL；写后缓存失效失败不回滚 DB，记录补偿和短 TTL 兜底。 |
| Redis rate limit / lock | 评论创建、互动、rank decay claim | `50ms` 到 `200ms` | 不重试或最多一次快速重试 | 开启 | 短时本机严格限流兜底；超过降级窗口后写路径返回 `1004` 或 `429`。 |
| RabbitMQ publish confirm | outbox dispatcher | `1s` 到 `3s` | 按 outbox 退避重试 | 开启 | 请求路径不依赖实时 publish；outbox backlog 增长时告警，事件延迟但业务事实已落库。 |

## API 降级矩阵

| API / use case | 关键依赖不可用 | 策略 |
| --- | --- | --- |
| `CreateComment` / `CreateReply` | Content、User、User relation、File service 或 PostgreSQL 不可用 | 失败 `1004`，不创建评论、不写 outbox。 |
| `UpdateComment` | Content、User、User relation、File service 或 PostgreSQL 不可用 | 失败 `1004`；不能在文章、权限和媒体事实不可确认时改内容。 |
| `DeleteComment` / `AdminDeleteComment` | PostgreSQL 不可用 | 失败 `1004`；Redis 缓存失效失败不回滚删除，记录补偿并依赖 TTL 收敛。 |
| `LikeComment` | User relation 或 PostgreSQL 不可用 | relation 不可确认时失败 `1004`；PostgreSQL 不可用失败；Redis 点赞状态缓存不可用时回源 DB。 |
| `UnlikeComment` | PostgreSQL 不可用 | 失败 `1004`；取消点赞不调用 User relation guard，允许用户撤销自己的历史点赞；Redis 点赞状态缓存不可用时回源 DB。 |
| `ListTopLevelComments` / `GetCommentDetail` | PostgreSQL 不可用 | 失败 `1004`。 |
| `ListTopLevelComments` / `GetCommentDetail` | User 作者摘要不可用 | 返回评论本体，作者摘要降级为占位；`publicId` 缺失时仅在 `unavailable=true` 下省略；记录 degraded metric。 |
| `ListTopLevelComments` / `GetCommentDetail` | File URL 解析不可用 | 返回文件 ID，省略 URL 字段；记录 degraded metric。 |
| `ListTopLevelComments` / `GetCommentDetail` | Content 可见性不可确认 | 首期 fail closed 返回 `1004`；后续若有 Content 可见性本地投影，再单独登记短 TTL 降级策略。 |
| `GetLikeStatus` / `BatchGetLikeStatus` | Redis 不可用 | 回源 PostgreSQL；不能用异步 `likeCount` 推断 `viewer.liked`。 |
| `ApplyCommentCounterDeltas` | PostgreSQL 不可用 | 本轮失败，delta 保持可重试；不丢弃。 |
| `DecayRecommendedRank` | Redis 分布式锁不可用 | 使用 PostgreSQL `FOR UPDATE SKIP LOCKED` claim 或跳过本轮；不能多实例重复重算同一批。 |
| Outbox dispatcher | RabbitMQ 不可用 | outbox 事件保持 `PENDING/FAILED`，退避重试；请求路径已提交的业务事实不回滚。 |

## 限流和 Redis 故障

Comment 限流分 Gateway 粗限流和 Comment 业务限流两层：

- Gateway 按 IP、用户、route 和 body size 做入口粗限流。
- Comment 按 `actorUserId + postId`、actor 全局、同内容短时间重复、点赞/取消点赞频率等业务维度限流。
- Redis 正常时，业务限流优先使用 Redis 分布式计数。
- Redis 短时不可用时，Comment 可以启用本机内存限流兜底，阈值必须比 Redis 分布式限流更严格，并记录 degraded metric。
- 本机限流不跨实例共享，只允许短窗口兜底；超过配置窗口后，评论创建、更新和高频互动返回 `1004` 或 `429`，避免反垃圾防线长期退化。
- 降低风险或幂等的操作，例如重复取消点赞、Admin 重试 outbox，不应被普通防刷限流完全阻断；可以限制批次和频率。

建议配置项：

| 配置 | 默认建议 | 说明 |
| --- | --- | --- |
| `COMMENT_RATE_LIMIT_REDIS_DEGRADED_WINDOW` | `60s` | Redis 限流不可用时允许本机兜底窗口。 |
| `COMMENT_CREATE_PER_USER_POST_WINDOW` | 配置化 | `actorUserId + postId` 维度创建评论频控。 |
| `COMMENT_CREATE_PER_USER_GLOBAL_WINDOW` | 配置化 | 单用户全局评论频控。 |
| `COMMENT_INTERACTION_PER_USER_WINDOW` | 配置化 | 点赞 / 取消点赞频控。 |

具体阈值首期保持配置化，不在 contract 中硬编码。

## 缓存策略

- Redis 缓存不可用不能阻止普通查询回源 PostgreSQL。
- 缓存值必须可重建；缓存 key 不能成为 API contract，也不能泄漏到 application 层。
- 列表、详情、点赞状态和首页评论缓存必须有 TTL；写操作提交后执行 best-effort 失效。
- 缓存失效失败不回滚已经提交的 Comment DB 事务；必须记录 `cache_invalidation_failed` metric，并通过短 TTL、异步补偿或下一次写入继续收敛。
- 对需要强一致的 `viewer.liked`，Redis 只能作为加速；缓存缺失或不可用时必须查 `comment_likes`。
- 对最终一致的 `likeCount`、HOT 和 RECOMMENDED，允许短暂延迟，但 worker lag 必须可观测。

## RabbitMQ / Outbox

Comment 生产跨服务事件必须使用 transactional outbox：

- 业务表和 `outbox_events` 在同一 PostgreSQL 事务中写入。
- 请求线程可以 best-effort 触发实时 publish，但不得把 RabbitMQ publish 成功作为业务成功条件。
- RabbitMQ 不可用时，outbox dispatcher 退避重试并更新 `retry_count`、`next_retry_at`、`last_error`。
- outbox backlog、oldest pending age、dead count 必须有 metrics 和告警。
- Admin `retry-dead` 只改变 outbox 状态，不重写业务事实。

## 熔断策略

每个外部 client adapter 在 runtime wiring 时声明 resilience policy：

| provider | 操作 | 熔断打开时 application 行为 |
| --- | --- | --- |
| `zhicore-content` | `CheckPostCommentable` | 写路径和可见性读取 fail closed，返回 `1004`。 |
| `zhicore-user` | 状态 / relation guard | 创建、更新和点赞 fail closed，返回 `1004` 或权限错误；取消点赞不依赖 relation guard。 |
| `zhicore-user` | 作者摘要批量查询 | 查询路径降级为占位作者。 |
| `zhicore-file` | 文件引用校验 | 写路径 fail closed。 |
| `zhicore-file` | URL 解析 | 查询路径省略 URL。 |
| `zhicore-ranking` | 热门候选同步 | job 跳过本轮或使用上一批候选。 |

熔断状态必须进入 metrics 和健康详情，但 adapter 不直接返回 HTTP response；application 负责映射为 `1004`、占位作者或省略 URL。

## 健康检查

`/health/live` 只检查进程存活。

`/health/ready` 至少检查：

- PostgreSQL 连接和轻量 ping。
- 必要配置已加载，例如 Content/User/File client base URL、RabbitMQ/Redis 配置格式。

Redis 和 RabbitMQ 首期不建议作为 Comment HTTP readiness 的硬阻断项：

- Redis 失败会让缓存和分布式限流进入 degraded，但普通读可以回源 DB，短窗口写可以本机限流兜底。
- RabbitMQ 失败会造成 outbox backlog，但请求路径事实已在 PostgreSQL 落库。

如果部署策略要求 Redis/RabbitMQ 故障时摘除 Comment 流量，应作为环境配置显式开启，不作为默认行为。

## Metrics 和告警

最低 metrics：

| metric | 标签 | 说明 |
| --- | --- | --- |
| `comment_downstream_requests_total` | `provider,operation,result` | 下游调用总数，result 包含 `success/timeout/circuit_open/error/degraded`。 |
| `comment_downstream_duration_ms` | `provider,operation` | 下游调用耗时。 |
| `comment_degraded_total` | `operation,reason` | 降级次数，例如 `author_summary_unavailable`、`upload_url_unavailable`、`rate_limiter_local_fallback`。 |
| `comment_rate_limited_total` | `dimension,operation` | 业务限流命中。 |
| `comment_cache_errors_total` | `operation,cache` | Redis cache 读写/失效失败。 |
| `comment_outbox_pending_total` | `eventType` | outbox 待发送数量。 |
| `comment_outbox_oldest_pending_seconds` | `eventType` | 最老 pending 事件年龄。 |
| `comment_counter_delta_lag_seconds` | `worker` | 点赞 delta 到统计读模型的滞后。 |
| `comment_rank_decay_lag_seconds` | `worker` | 推荐排序衰减任务滞后。 |

日志必须带 `requestId` / `traceId`、`operation`、`provider`、`durationMs`、`degradedReason`，不得记录评论全文、Authorization、cookie、完整 User-Agent 或下游敏感错误堆栈。

## 测试要求

- Application test 覆盖写路径下游不可用 fail closed：Content、User relation、File 校验失败时不写 comment、不写 outbox；更新评论必须覆盖 Content guard 不可用失败。
- Application test 覆盖查询降级：作者摘要不可用返回占位作者，`publicId` 缺失时仅在 `unavailable=true` 下省略，File URL 解析失败时保留 file ID 并省略 URL。
- Application test 覆盖互动降级：点赞在 relation 不可用时失败，取消点赞不因 relation 不可用失败。
- Adapter test 覆盖 timeout、熔断打开和 HTTP status 到 module-local 错误的翻译。
- Worker test 覆盖 outbox publish 失败退避重试、DEAD retry、delta worker 失败可重试和 rank decay claim 并发。
- Handler contract test 覆盖 `1004` / `503` envelope、`Retry-After` 可选 header 和 degraded 响应字段不泄露底层依赖。
