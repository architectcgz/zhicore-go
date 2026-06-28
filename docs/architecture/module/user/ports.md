# User Ports 设计

Ports 放在 `services/zhicore-user/internal/user/ports`，按能力和用例族定义 consumer-side interface。端口不能暴露 `*gorm.DB`、`*redis.Client`、Gin context、HTTP DTO、ORM sentinel 或外部 SDK 类型。

## 核心端口

| Port | 职责 |
| --- | --- |
| `UserCommandRepository` | profile 初始化、资料更新、状态变更、按 `accountId` / `userId` 加载并保存 User 聚合。 |
| `UserQueryRepository` | `GetMe`、按 `publicId` 查询、批量 UserSimple、availability、管理端资料查询。 |
| `UserIdentityRepository` | 生成和解析 `publicId`，查询 `publicId -> userId`，检查 nickname 唯一性。可合并进 User repository，但语义要清楚。 |
| `FollowRepository` | 关注 / 取关写入、关系存在性检查、统计同步更新。 |
| `FollowQueryRepository` | followers、following、follow stats、`CheckFollowing`、`ListFollowerShard`。 |
| `BlockRepository` | 拉黑 / 解除拉黑写入、批量拉黑检查、拉黑列表查询。 |
| `FollowStatsRepairRepository` | 关注统计对账和重建；首批可只定义，不实现入口。 |

## 基础设施机制端口

| Port | 职责 |
| --- | --- |
| `TransactionRunner` | 显式事务边界，application 不直接持有数据库连接。 |
| `OutboxPublisher` | 业务事务内追加 User 集成事件。 |
| `Clock` | UTC 当前时间和审计时间。 |
| `CursorCodec` | 关系列表 cursor 编码/解码；使用内部关系 `id`，HTTP 不暴露 SQL 锚点。 |
| `UserPublicIDCodec` | 根据内部 `userId` 生成和解析 User `publicId`；封装前缀、版本和 secret。 |
| `RateLimiter` | User 业务限流决策；返回稳定 `Outcome`、公开错误码、原因和 fallback 类型，不只返回 bool。 |
| `MetricsRecorder` | 低基数指标记录；不能影响业务控制流。 |

`publicId` 生成不依赖 `zhicore-id-generator`。推荐使用 User 本地短公开 ID 规则，例如 `u` 前缀 + 版本 + `Base58/Base62(permute64(userId, secret))`。

## 外部服务端口

| Port | 职责 |
| --- | --- |
| `AuthAccountClient` | 主动注销由 Auth 编排；User 如需校验初始化来源或账号存在性，可调用 Auth contract。User 不写 Auth 状态。 |
| `FileReferenceClient` | 写入头像前校验 `avatarFileId` 存在、类型为图片且状态可引用。 |
| `FileURLResolver` | 前端 HTTP 查询时把 `avatarFileId` 批量解析为 `avatarUrl`。解析失败不让 profile 查询整体失败。 |
| `AdminAuditClient` | 可选；通常 Admin facade 自己记录审计，User 只保存最小 operator/reason 字段。 |

## 缓存端口

缓存不是首批正确性依赖。

| Port | 职责 |
| --- | --- |
| `UserSimpleCacheStore` | `user:{userId}:simple` cache-aside。 |
| `UserProfileCacheStore` | `user:{userId}:profile`、`user:public:{publicId}:id` 缓存。 |
| `UserAvailabilityCacheStore` | `user:{userId}:availability` 短 TTL 缓存；写路径可直接查 DB。 |
| `RelationshipCacheStore` | block/follow pair 和列表缓存；首批可不启用。 |
| `FollowStatsCacheStore` | `user:{userId}:follow_stats` 缓存。 |

`avatarUrl` 不缓存进 User profile 缓存。若需要缓存 URL，只能在 FileURLResolver 或 Upload/File Service adapter 内遵守 File Service TTL。

## Typed Client Provider Contract

User 作为 provider 拥有 `libs/contracts/clients/user/`。Go contract 后续应拆出 interface、DTO、HTTP adapter 和 contract tests。

目标 interface 草案：

```go
type Client interface {
    CreateProfileForAccount(ctx context.Context, input CreateProfileForAccountInput) (UserProfile, error)
    DeactivateUserProfile(ctx context.Context, input DeactivateUserProfileInput) error
    BatchGetUserSimple(ctx context.Context, userIDs []int64) (BatchUserSimpleResult, error)
    BatchGetUserAvailability(ctx context.Context, userIDs []int64) (map[int64]UserAvailability, error)
    BatchCheckBlocked(ctx context.Context, pairs []UserPair) (map[UserPair]bool, error)
    CheckFollowing(ctx context.Context, followerID, followingID int64) (bool, error)
    GetStrangerMessageSetting(ctx context.Context, userID int64) (bool, error)
    ListFollowerShard(ctx context.Context, input ListFollowerShardInput) (ListFollowerShardResult, error)
}
```

DTO 规则：

- `UserSimple` 入参使用内部 `userId`。
- `UserSimple` 返回 `userId`、`publicId`、`nickname`、`avatarFileId`、`profileVersion`、`status`。
- 默认 typed client 不返回 `avatarUrl`。
- 缺失用户在批量结果中省略，并返回 `missingUserIds`。
- 服务间 HTTP adapter 必须从 consumer 配置和调用点常量写入 `X-Caller-Service` / `X-Caller-Operation`，不能透传客户端输入。

## 端口约束

- repository 返回 module-local 语义错误，例如 `UserNotFound`、`NicknameTaken`、`DuplicateFollow`。
- 外部 HTTP / SDK adapter 负责把 status、超时、熔断和 payload 错误翻译为 module-local 错误；application 拥有降级选择，adapter 不构造业务 DTO 伪装成功。
- `OutboxPublisher` 只负责事务内追加事件；dispatcher 的 claim、retry、dead 状态更新属于 infrastructure job。
- `FileReferenceClient` 用于写路径校验，失败时不写 User 资料。
- `FileURLResolver` 用于读路径展示，失败时省略 `avatarUrl` 并记录观测。
- `RateLimiter` 按 [rate-limiting.md](rate-limiting.md) 返回 `ALLOW`、`REJECT_TOO_FREQUENT`、`DEGRADED_ALLOW_LOCAL` 或 `DEGRADED_DENY_UNAVAILABLE`；高副作用写路径不能在 Redis 限流不可确认时 fail-open。
- runtime wiring 必须按 [runtime-resilience.md](runtime-resilience.md) 为每个下游 `provider + operation` 声明 timeout、retry、circuit breaker、max-in-flight 和 degrade strategy。
- `BatchGetUserSimple` 不可承担写路径资格判断；写路径使用 `BatchGetUserAvailability` 和 `BatchCheckBlocked`。
- `BatchCheckBlocked` / `CheckFollowing` 中“用户缺失返回 false”和“User 依赖不可用返回 `SERVICE_DEGRADED`”必须区分，consumer 写路径在 degraded 时 fail closed。

## Go 包落点

```text
services/zhicore-user/
  api/http/
  internal/user/
    application/
      commands/
      queries/
    domain/
      profile/
      relationship/
      block/
      shared/
      events/
    ports/
    infrastructure/
      postgres/
      redis/
      rabbitmq/
        publishers/
        jobs/
      clients/
      cursor/
      publicid/
    runtime/
      module.go
```

第一版可以按实际代码量合并子包；拆包标准是职责和依赖边界，而不是为了看起来像 DDD。
