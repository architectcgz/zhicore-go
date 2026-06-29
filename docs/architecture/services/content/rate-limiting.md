# Content 限流设计

本文是 `zhicore-content` 的限流和频控专题事实源。字段级 HTTP schema 只引用本文，不在每个 endpoint 重复完整矩阵。

当前状态：本文只固定设计和实现准入条件，不表示 Go 代码已经实现限流。首次实现任一 Content endpoint 前，必须先把本文的 `RateLimiter` 决策语义、配置项和 contract test 落到对应切片。

## 目标

Content 限流同时服务三个目标：

- 保护公开阅读、标签搜索、正文读取、批量摘要和 reader presence 这类高频入口，避免单个 IP、用户、文章或服务调用方耗尽数据库、MongoDB、Redis 和缓存回源能力。
- 保护草稿保存、发布、定时发布、点赞、收藏、管理端删除和 outbox retry 等写路径，避免重复提交、自动保存风暴或管理误操作放大副作用。
- 在 Redis 短时不可用时明确哪些 API 可以用本机限流或 Gateway 粗限流兜底，哪些写路径必须 fail closed 或返回服务暂时不可用。

## 两层限流

| 层级 | 归属 | 职责 |
| --- | --- | --- |
| Gateway 粗限流 | `zhicore-gateway` | 按 IP、route、method、基础突发流量限流，阻挡匿名洪水流量和明显扫描。 |
| Content 业务限流 | `zhicore-content` | 按 actor、post、session、service caller、operation 和高成本资源维度限流；保护正文存储、发布事务、互动统计、presence 和管理命令。 |

Gateway 不能替代 Content 业务限流。Gateway 不知道文章 owner、`postId`、草稿保存节奏、发布幂等键、presence session、内部服务调用方和 outbox retry 目标。

Content 限流 key 只能保存规范化值或 hash。不得在 Redis key、日志或 metrics label 中保存完整请求 body、正文 blocks、raw title / summary、access token、cookie、Authorization header 或未规范化的用户输入文本。

## API 矩阵

| API / 能力 | Gateway 粗限流 | Content 业务限流 | Redis 不可用时 |
| --- | --- | --- | --- |
| `GET /api/v1/posts`、`GET /api/v1/tags/{slug}/posts` | IP + route + query 大类 | IP / 匿名指纹 + route；登录用户额外按 actor；`authorId`、`tag`、`categoryId` 只进入规范化低基数字段或 hash。 | 可继续依赖 Gateway 和更严格本机限流；DB / cache 回源失败按 `1004`。 |
| `GET /api/v1/posts/{postId}`、`GET /api/v1/posts/{postId}/body` | IP + route | IP / actor + `postId` + route；服务间调用按 `X-Caller-Service` + `X-Caller-Operation` + route + `postId`。 | 公开读可本机限流兜底；Search 等服务间调用应返回 `1004` 或让 consumer retry，不能无限回源 MongoDB。 |
| `POST /api/v1/posts/batch-get` | IP + route | IP / actor / `X-Caller-Service` + `X-Caller-Operation` + route；`postIds` 数量上限先做参数校验；超频返回 `1003`。 | 服务间批量摘要应 fail closed 返回 `1004`；公开调用可短时本机限流兜底。 |
| `POST /api/v1/posts` | IP + route | actor + route；可选 actor + normalized title hash 防止脚本批量建草稿。 | 短时本机限流兜底；持续不可用后停止创建草稿并返回 `1004`。 |
| `GET /api/v1/me/posts`、`GET /api/v1/me/drafts`、`GET /api/v1/posts/{postId}/draft` | IP + route | actor + route；作者草稿读取额外按 actor + `postId`。 | 可本机限流兜底；数据库可用时继续服务。 |
| `PATCH /api/v1/posts/{postId}/draft/meta` | IP + route | actor + `postId` + operation；保护乐观锁冲突风暴。 | 短时本机限流兜底；持续不可用后返回 `1004`。 |
| `PUT /api/v1/posts/{postId}/draft/body` | IP + route + body size | actor + `postId` + operation；按 autosave 场景允许短 burst，但限制持续 QPS 和单位时间 body 字节量。 | 不能无限写 MongoDB；短时本机限流兜底，持续不可用后返回 `1004`。 |
| `DELETE /api/v1/posts/{postId}/draft` | IP + route | actor + `postId` + operation；重复删除可返回当前状态或成功空响应，但仍计入重复提交频控。 | 可本机限流兜底；清理任务写入失败按业务错误处理。 |
| `POST /api/v1/posts/{postId}/publish`、`unpublish`、`schedule`、`restore`、`DELETE /api/v1/posts/{postId}` | IP + route | actor + `postId` + operation；有 `Idempotency-Key` 时同时按 key 限制重放；无 key 时更严格。 | 发布、定时发布、删除和恢复是高副作用写路径；分布式限流不可确认时返回 `1004`，不要 fail-open。 |
| `PUT /api/v1/posts/{postId}/tags`、`DELETE /api/v1/posts/{postId}/tags/{slug}` | IP + route | actor + `postId` + operation；标签集合大小先做参数校验。 | 短时本机限流兜底；持续不可用后返回 `1004`。 |
| `PUT` / `DELETE /api/v1/posts/{postId}/like`、`favorite` | IP + route | actor + `postId` + operation；actor 全局互动写频控；幂等重复请求可返回当前状态，但刷写仍受限。 | 可短时本机限流兜底；持续不可用后返回 `1004`，避免统计和 outbox 被刷爆。 |
| `GET /api/v1/posts/{postId}/engagement`、`POST /api/v1/posts/engagement/batch-status` | IP + route | IP / actor + route；batch 按 `postIds` 数量校验；Redis 故障 DB fallback 额外按本机预算和 max-in-flight 控制。 | 可本机限流兜底；缓存不可用时只允许受控批量 DB 回源，预算耗尽时返回 viewer/item unknown 或 `1004`，不得逐条 `EXISTS`。 |
| `PUT /api/v1/posts/{postId}/reader-sessions/{sessionId}` | IP + route | sessionId + `postId` + clientId / actor；短间隔 heartbeat 先合并为 no-op success，持续洪水才返回 `1003`。 | Presence 是附加能力；Redis 不可用时返回 HTTP `200` + 空 `ReaderPresence`，并标记 `degraded=true`。 |
| `DELETE /api/v1/posts/{postId}/reader-sessions/{sessionId}` | IP + route | sessionId + `postId`；离开动作是收敛动作，不应被普通限流阻断。 | Redis 不可用时返回 HTTP `200` + 空 `data`，记录 degraded metric；不返回 `1004`。 |
| `GET /api/v1/posts/{postId}/reader-presence` | IP + route | IP / actor + `postId`；保护 Redis 热 key。 | Redis 不可用时返回空 presence 摘要并记录 degraded metric，不能影响正文读取。 |
| `GET /api/v1/tags`、`search`、`hot` | IP + route | IP / actor + route；`keyword` 规范化后只进入 hash；限制 keyword 长度和 limit。 | 可本机限流兜底；缓存不可用时控制 DB 回源。 |
| `GET /api/v1/admin/content/posts` | IP + route | admin actor + route + query 大类；限制大范围扫描和高频翻页。 | 管理查询可本机限流兜底；依赖不可用返回 `1004`。 |
| `DELETE /api/v1/admin/content/posts/{postId}` | IP + route | admin actor + `postId` + operation；写 Admin 审计；重复请求可返回当前状态但仍限频。 | 高风险管理写路径不能 fail-open；限流不可确认时返回 `1004`。 |
| `GET /api/v1/admin/content/outbox-events` | IP + route | admin actor + route；按状态和 eventType 低基数字段限流。 | 可本机限流兜底。 |
| `POST /api/v1/admin/content/outbox-events/{eventId}/retry` | IP + route | admin actor + `eventId` + operation；同一 event retry 必须有冷却窗口。 | 高副作用运维命令不能 fail-open；限流不可确认时返回 `1004`。 |
| Content typed client / 内部服务调用 | service route + `X-Caller-Service` | `X-Caller-Service` + `X-Caller-Operation` + target；Search 拉正文、Ranking / Notification 读摘要分别独立配额。 | consumer 应 retry / DLQ；缺少可信 caller identity 的服务间-only 调用不能落到匿名公开配额。 |

## 错误和响应

- 业务限流命中时返回 HTTP `429`，body `code` 使用 `1003 REQUEST_TOO_FREQUENT`。
- Gateway 粗限流命中时也可以返回 HTTP `429`；如果 Gateway 保留历史 `body.code=429`，必须在 Gateway contract 中登记为例外，Content 不扩大该例外。
- Content 不能把限流错误伪装成参数错误、权限错误或资源不存在。
- Presence heartbeat 的短间隔重复请求可以在业务层合并为 no-op success；只有持续洪水或超过配置窗口时才返回 `1003`。
- Redis / limiter 依赖不可用导致写路径不能确认配额时，返回 HTTP `503`，body `code` 使用 `1004 SERVICE_DEGRADED`。
- Reader presence 是附加能力，Redis 不可用时不返回 `1004`：`PUT` 和 `GET` 返回空 `ReaderPresence` 并标记 `degraded=true`，`DELETE` 返回空成功。

## Redis 故障原则

Redis 不可用时不能统一放行，也不能把所有 Content API 直接打死。

- 公开读、标签查询、作者工作台只读 API 可短时依赖 Gateway 和本机限流兜底，但必须记录 degraded metric。
- 保存正文、发布、定时发布、管理删除、outbox retry 和内部拉取正文属于高成本或高副作用路径；分布式限流不可确认时返回 `1004`，不 fail-open。
- 点赞 / 收藏虽然幂等，但会改统计和 outbox；可短时本机限流兜底，持续不可用后返回 `1004`。
- Engagement 读路径可短时 DB fallback，但必须通过本机 fallback limiter、DB breaker 和 max-in-flight 保护；当前用户状态不可确认时使用 `null + degraded=true` 表达 unknown，不能当成未点赞 / 未收藏。
- Reader presence 是附加能力；Redis 不可用不能影响文章详情、正文读取和公开列表。

## `RateLimiter` 端口决策语义

`RateLimiter` 不能只返回布尔 `allow / reject`。Content 需要把 Redis 故障、本机兜底、presence no-op 和公开错误码清楚传给 application / handler。

建议端口语义：

```go
type RateLimitDecision struct {
    Outcome     RateLimitOutcome
    PublicCode  int
    Reason      string
    LimitType   string
    RetryAfter  time.Duration
    Fallback    RateLimitFallback
}
```

`Outcome` 使用稳定枚举：

| Outcome | HTTP 行为 | 使用场景 |
| --- | --- | --- |
| `ALLOW` | 继续执行 use case | 分布式限流或允许的本机兜底通过。 |
| `REJECT_TOO_FREQUENT` | HTTP `429` + code `1003` | 达到业务频控阈值。 |
| `DEGRADED_ALLOW_LOCAL` | 继续执行 use case，并记录 degraded metric | 公开读、标签、作者只读、部分互动在 Redis 短时不可用且允许本机兜底。 |
| `DEGRADED_DENY_UNAVAILABLE` | HTTP `503` + code `1004` | 发布、删除、管理命令、内部高成本读取等不能 fail-open 的路径。 |
| `NOOP_SUCCESS` | HTTP `200`，不执行写入副作用 | reader presence 短间隔 heartbeat 合并、重复 leave 等收敛动作。 |

规则：

- `Reason` 必须是稳定机器码，例如 `actor_post_operation_limit`、`redis_unavailable_fail_closed`、`presence_heartbeat_coalesced`，不能写入原始错误文本。
- `LimitType` 使用低基数枚举，例如 `public_read`、`draft_write`、`publish_lifecycle`、`engagement_write`、`presence`、`admin_command`、`internal_client`。
- Engagement 读路径使用独立 `LimitType`，例如 `engagement_read` 和 `engagement_db_fallback`，避免和互动写路径共享同一预算。
- `Fallback` 区分 `none`、`local_memory`、`gateway_only`、`presence_empty`，便于 metrics 和日志聚合。
- `RetryAfter` 只在频控窗口可明确计算时返回；不能为了凑响应而写死。
- application 拥有限流结果到业务错误的映射；Redis adapter 只翻译依赖错误，不构造 HTTP response 或业务 DTO。

首次实现前必须补测试：

- `REJECT_TOO_FREQUENT` 映射为 `1003 / 429`。
- `DEGRADED_DENY_UNAVAILABLE` 映射为 `1004 / 503`。
- 允许本机兜底的 API 在 Redis 不可用时继续执行并记录 degraded 决策。
- 高副作用 API 在 Redis 不可用时不执行 use case。
- Presence heartbeat 的 `NOOP_SUCCESS` 不写 Redis / DB，不返回错误。

## 配置和观测

所有阈值、窗口、burst、冷却时间、单位时间 body 字节量、内部服务调用配额和 Redis 故障 fallback 时长必须配置化，不能写死在 handler 或 application 中。

每类限流至少记录：

- allow / reject 计数
- `route`、`operation`、`limitType`、`reason`
- Redis unavailable 和 local fallback 计数
- actor 维度只记录是否登录、角色类型或 hash，不记录原始用户输入
- high-cost operation 的目标类型，例如 `post`、`body`、`outbox_event`

metrics label 不得包含原始标题、摘要、正文、tag keyword、IP、token、cookie、Authorization header 或完整 URL。
