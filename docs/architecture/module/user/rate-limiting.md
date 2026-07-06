# User 限流设计

本文是 `zhicore-user` 的限流和频控专题事实源。字段级 HTTP schema 只引用本文，不在每个 endpoint 重复完整矩阵。

当前状态：本文只固定设计和实现准入条件，不表示 Go 代码已经实现限流。首次实现任一 User endpoint 或 typed client provider endpoint 前，必须先把本文的 `RateLimiter` 决策语义、配置项和 contract test 落到对应切片。

## 目标

User 限流同时服务三个目标：

- 保护公开 profile、关注 / 粉丝列表、UserSimple 批量查询和 Notification fanout 这类高频读入口，避免单个 IP、用户、目标用户或服务调用方耗尽 PostgreSQL、Redis 和缓存回源能力。
- 保护资料更新、关注、拉黑、状态变更、Admin 资料命令和后续 check-in 写路径，避免重复提交、脚本刷写或管理误操作放大副作用。
- 在 Redis 短时不可用时明确哪些查询可以用本机限流或 Gateway 粗限流兜底，哪些写路径必须 fail closed 或返回服务暂时不可用。

## 两层限流

| 层级 | 归属 | 职责 |
| --- | --- | --- |
| Gateway 粗限流 | `zhicore-gateway` | 按 IP、route、method、基础突发流量限流，阻挡匿名洪水流量和明显扫描。 |
| User 业务限流 | `zhicore-user` | 按 actor、target user、service caller、operation、batch size 和高成本资源维度限流；保护关系写入、批量查询、follower fanout 和管理命令。 |

Gateway 不能替代 User 业务限流。Gateway 不知道目标 `publicId` 解析结果、关系写入幂等状态、内部服务调用方、批量 `userId` 数量和 Notification follower shard 成本。

User 限流 key 只能保存规范化值或 hash。不得在 Redis key、日志或 metrics label 中保存完整昵称、简介、raw `publicId` 列表、access token、cookie、Authorization header 或未规范化的用户输入文本。

## API 矩阵

| API / 能力 | Gateway 粗限流 | User 业务限流 | Redis 不可用时 |
| --- | --- | --- | --- |
| `GET /api/v1/users/{publicId}` | IP + route | IP / actor + route + target publicId hash；登录用户额外按 actor。 | 可短时依赖 Gateway 和本机限流兜底；DB 失败返回 `SERVICE_DEGRADED`，不伪装 404。 |
| `GET /api/v1/users/me` | IP + route | actor + route；避免单用户高频回源。 | 可本机限流兜底；头像 URL 解析降级按 `runtime-resilience.md`。 |
| `PATCH /api/v1/users/me/profile` | IP + route + body size | actor + operation；nickname 改名、头像更新和简介更新分别可有子限额。 | 资料更新是写路径；分布式限流不可确认时返回 `1004`，不 fail-open。 |
| `POST` / `DELETE /api/v1/users/{publicId}/follow` | IP + route | actor + target publicId hash + operation；重复请求仍计入频控。 | 关系写路径不能 fail-open；返回 `1004`，不执行 use case。 |
| `POST` / `DELETE /api/v1/users/{publicId}/block` | IP + route | actor + target publicId hash + operation；拉黑同时清理关注，限额应独立于 follow。 | 高副作用写路径不能 fail-open；返回 `1004`，不执行 use case。 |
| `GET /api/v1/users/{publicId}/followers`、`following` | IP + route | IP / actor + target publicId hash + route + cursor bucket；限制大范围翻页。 | 可本机限流兜底；DB 失败返回 `SERVICE_DEGRADED`，不返回空列表。 |
| `GET /api/v1/users/me/blocked` | IP + route | actor + route + cursor bucket。 | 可本机限流兜底；DB 失败返回 `SERVICE_DEGRADED`。 |
| `CreateProfileForAccount` | service route + `X-Caller-Service` | `X-Caller-Service=zhicore-auth` + `X-Caller-Operation` + account hash；限制注册补偿风暴。 | Auth 编排写路径不能 fail-open；返回 `SERVICE_DEGRADED`。 |
| `DeactivateUserProfile` | service route + `X-Caller-Service` | `zhicore-auth` + operation + account / user hash。 | 注销编排不能伪造成功；返回 `SERVICE_DEGRADED`。 |
| `BatchGetUserSimple` | service route + `X-Caller-Service` | caller service + operation + route + batch size bucket；`userIds` 数量上限先做参数校验。 | 内部批量查询不能落匿名配额；可短时本机按 caller 兜底，持续不可用后返回 `SERVICE_DEGRADED`。 |
| `BatchGetUserAvailability` | service route + `X-Caller-Service` | caller service + operation + batch size bucket；写路径 guard 独立配额。 | 写路径 guard fail closed，返回 `SERVICE_DEGRADED`。 |
| `BatchCheckBlocked`、`CheckFollowing`、`GetStrangerMessageSetting` | service route + `X-Caller-Service` | caller service + operation + target hash / batch size bucket。 | Message / Comment guard fail closed；不能把限流依赖不可用当成未拉黑或允许私信。 |
| `ListFollowerShard` | service route + `X-Caller-Service` | `zhicore-notification` + operation + `audienceClass` + shard cursor bucket；限制 fanout worker 并发和分页速率。`ALL` / backfill 配额必须严于 `HOT`。 | fanout job retry / DLQ；不能返回空 shard 冒充成功，不能从 `HOT` 自动 fallback 到 `ALL`。 |
| Admin profile 查询 | IP + route | admin actor + route + query bucket；限制大范围扫描和高频翻页。 | 可本机限流兜底；依赖不可用返回 `SERVICE_DEGRADED`。 |
| Admin 删除 / 恢复 / 修正 profile | IP + route | admin actor + target user hash + operation；重复请求仍限频。 | 高风险管理写路径不能 fail-open；返回 `SERVICE_DEGRADED`。 |
| Check-in 后续 API | IP + route | actor + date bucket + operation；签到写入和月度图查询分开配额。 | 签到写路径不能 fail-open；查询可按后续 schema 决定本机兜底或返回 degraded。 |

缺少可信 `X-Caller-Service` / `X-Caller-Operation` 的服务间-only 调用不能落到公开匿名配额；应按内部认证、caller identity 缺失或 `SERVICE_DEGRADED` 处理。

## 错误和响应

- 业务限流命中时返回 HTTP `429`，body `code` 使用 `1003 REQUEST_TOO_FREQUENT`。
- Gateway 粗限流命中时也可以返回 HTTP `429`；如果 Gateway 保留历史 `body.code=429`，必须在 Gateway contract 中登记为例外，User 不扩大该例外。
- User 不能把限流错误伪装成参数错误、权限错误、用户不存在、未关注或未拉黑。
- Redis / limiter 依赖不可用导致写路径不能确认配额时，返回 HTTP `503`，body `code` 使用 `1004 SERVICE_DEGRADED`。
- 查询路径允许本机兜底时必须记录 degraded 决策；一旦持续不可用或本机 fallback 超窗，返回 `1004`。

## Redis 故障原则

Redis 不可用时不能统一放行，也不能把所有 User API 直接打死。

- 公开 profile 查询、当前用户 profile 查询、followers / following / blocked list 和 Admin 只读查询可短时依赖 Gateway 与本机限流兜底，但必须记录 degraded metric，并继续受 DB max-in-flight 保护。
- 资料更新、关注、拉黑、Admin 资料命令、注册 profile 初始化、注销 profile、check-in 写入和内部写路径 guard 属于高副作用或权限关键路径；分布式限流不可确认时返回 `1004`，不 fail-open。
- `BatchGetUserSimple` 可按 caller 做短时本机兜底；`BatchGetUserAvailability`、`BatchCheckBlocked`、`CheckFollowing` 和 `GetStrangerMessageSetting` 用于写路径或消息权限 guard 时必须 fail closed。
- Notification `ListFollowerShard` 是高 fanout 读路径；Redis 限流不可用时优先让 job retry / DLQ，而不是无限回源 PostgreSQL。`HOT` 活跃受众查询失败时不能改查 `ALL`，否则会把限流或活跃读模型故障放大成全量粉丝写入。

## `RateLimiter` 端口决策语义

`RateLimiter` 不能只返回布尔 `allow / reject`。User 需要把 Redis 故障、本机兜底、caller identity 缺失和公开错误码清楚传给 application / handler。

建议端口语义：

```go
type RateLimitDecision struct {
    Outcome    RateLimitOutcome
    PublicCode int
    Reason     string
    LimitType  string
    RetryAfter time.Duration
    Fallback   RateLimitFallback
}
```

`Outcome` 使用稳定枚举：

| Outcome | HTTP 行为 | 使用场景 |
| --- | --- | --- |
| `ALLOW` | 继续执行 use case | 分布式限流或允许的本机兜底通过。 |
| `REJECT_TOO_FREQUENT` | HTTP `429` + code `1003` | 达到业务频控阈值。 |
| `DEGRADED_ALLOW_LOCAL` | 继续执行 use case，并记录 degraded metric | 公开 profile、当前用户查询、关系列表、Admin 只读查询等短时 Redis 不可用且允许本机兜底的路径。 |
| `DEGRADED_DENY_UNAVAILABLE` | HTTP `503` + code `1004` | 资料更新、关系写入、Admin 命令、注册 / 注销、内部 guard 和 follower shard 等不能 fail-open 的路径。 |

规则：

- `Reason` 必须是稳定机器码，例如 `actor_target_operation_limit`、`redis_unavailable_fail_closed`、`caller_identity_missing`，不能写入原始错误文本。
- `LimitType` 使用低基数枚举，例如 `public_profile_read`、`profile_write`、`relationship_write`、`relationship_list`、`internal_client`、`admin_command`、`check_in`。
- `Fallback` 区分 `none`、`local_memory`、`gateway_only`，便于 metrics 和日志聚合。
- `RetryAfter` 只在频控窗口可明确计算时返回；不能为了凑响应而写死。
- application 拥有限流结果到业务错误的映射；Redis adapter 只翻译依赖错误，不构造 HTTP response 或业务 DTO。

首次实现前必须补测试：

- `REJECT_TOO_FREQUENT` 映射为 `1003 / 429`。
- `DEGRADED_DENY_UNAVAILABLE` 映射为 `1004 / 503`，且不执行 use case。
- 允许本机兜底的查询 API 在 Redis 不可用时继续执行并记录 degraded 决策。
- 高副作用 API 在 Redis 不可用时不执行 use case。
- 缺少 caller identity 的内部接口不落到匿名公开配额。

## 配置和观测

所有阈值、窗口、burst、冷却时间、批量大小桶、内部服务调用配额、Notification fanout 并发、`audienceClass` 配额和 Redis 故障 fallback 时长必须配置化，不能写死在 handler 或 application 中。

每类限流至少记录：

- allow / reject 计数。
- `route`、`operation`、`limitType`、`reason`。
- Redis unavailable 和 local fallback 计数。
- caller service、caller operation 和 batch size bucket。
- actor 维度只记录是否登录、角色类型或 hash，不记录原始用户输入。
- high-cost operation 的目标类型，例如 `profile`、`relationship`、`follower_shard`、`admin_user`。

metrics label 不得包含原始昵称、简介、完整 `publicId`、完整 `userIds` 列表、IP、token、cookie、Authorization header 或完整 URL。
