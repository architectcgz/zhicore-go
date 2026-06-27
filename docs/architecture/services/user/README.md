# User 服务设计

## 事实来源

- Java `zhicore-user` controller：`UserCommandController`、`UserQueryController`、`Follow*Controller`、`Block*Controller`、`CheckIn*Controller`、`AdminUser*Controller`。原 `AuthController` 的认证职责已抽离到 `docs/architecture/services/auth/README.md`。
- Java `database/init-all-databases.sql` 和 `docker/postgres-init/02-init-tables.sql` 的 user 表。
- `zhicore-client` 中 User 相关 Feign client 和 DTO。

## 职责边界

`zhicore-user` 拥有用户公开资料、头像引用、陌生人消息设置、关注、拉黑、签到和用户资料摘要查询。

User 不拥有账号凭证、密码 hash、角色事实、JWT 签发、refresh token、文章、评论、通知或私信。认证账号由 `zhicore-auth` 拥有；当前不提供用户文章 facade，用户主页里“某用户发表的文章”直接调用 Content 作者过滤接口。

## DDD 目标设计

User 是独立限界上下文。统一语言以“用户资料、公开资料、头像引用、陌生人消息设置、关注、粉丝、拉黑、签到、资料版本、用户摘要”为核心，不把 Auth、Content、Comment、Message、Notification 或 Upload 的模型引入 User 领域层。

DDD 设计用于指导 Go 目标实现，不表示当前 Go 代码已经完成。Java 侧已有领域模型、命令服务、outbox 和 Redis 适配器是事实来源之一，但 Go 实现按本仓库 `api/http -> application -> domain/ports -> infrastructure` 的依赖方向重新落点。

### 限界上下文与子域

User 上下文内按职责拆成以下子域：

| 子域 | 职责 | 主要存储 |
| --- | --- | --- |
| Profile | 用户公开资料、头像引用、陌生人消息设置和资料版本 | `users` |
| Relationship | 关注、取消关注、粉丝/关注列表、关系判断和关注统计 | `user_follows`、`user_follow_stats` |
| Block | 拉黑、取消拉黑、拉黑列表和拉黑判断 | `user_blocks` |
| Check-in | 签到记录、连续签到统计和月度签到图 | `user_check_ins`、`user_check_in_stats`、Redis bitmap |
| Integration | User 生产的跨服务事件、outbox 派发、缓存失效和事务后补偿 | `outbox_events` |

### 聚合

#### `User` 聚合

`User` 是用户资料聚合根，负责维护公开资料、头像引用、资料版本和消息设置：

- **标识**：内部 `UserID`。第一阶段可以与 Auth `AccountID` 保持一一映射；User 不生成或签发认证主体。
- **账号引用**：`AccountID`，引用 Auth 拥有的账号事实。
- **资料字段**：`NickName`、`AvatarFileID`、`Bio`、`ProfileVersion`。
- **设置字段**：`StrangerMessageAllowed`。
- **行为**：`CreateProfileForAccount`、`UpdateProfile`、`UpdateStrangerMessageSetting`。
- **领域事件**：`UserProfileCreated`、`UserProfileUpdated`。

`ProfileVersion` 用于保证 User 资料事件的单调顺序。Go 实现中版本号必须由 repository 在 SQL 中原子递增并返回新值，例如 `profile_version = profile_version + 1 RETURNING profile_version`；application 不能手动把旧版本加一后写回。

`User` 的核心不变量：

- User profile 必须引用一个 Auth account。
- 昵称和简介长度必须符合 User 服务规则。
- 资料更新事件必须携带更新后的 `profileVersion`，供 Content 作者快照抵御乱序事件。
- 账号禁用、角色变更和登录状态由 Auth 决定；User 可通过 Auth contract 或事件获知账号状态，但不写 Auth 状态。

#### `UserFollow` / `UserFollowStats`

`UserFollow` 是以 `(FollowerID, FollowingID)` 为自然唯一键的关注关系实体，不作为独立聚合根暴露生命周期。它的生命周期由关注/取消关注用例拥有。

`UserFollowStats` 是独立统计聚合根，负责维护粉丝数和关注数：

- **标识**：`UserID`
- **统计字段**：`FollowersCount`、`FollowingCount`
- **行为**：`IncrementFollowers`、`DecrementFollowers`、`IncrementFollowing`、`DecrementFollowing`
- **不变量**：计数不能为负数

关注和取消关注写入时由 application 在同一 PostgreSQL 事务内完成：

```text
user_follows 插入/删除
+ user_follow_stats 原子增减
+ outbox_events 写入 user.followed / user.unfollowed
```

Redis 关注统计缓存只在事务提交后 best-effort 更新，失败不回滚业务事务。

#### `UserBlock`

`UserBlock` 是以 `(BlockerID, BlockedID)` 为自然唯一键的拉黑关系实体，不作为独立聚合根暴露生命周期。它的生命周期由拉黑/取消拉黑用例拥有。

拉黑行为必须保证：

- 不能拉黑自己。
- 被拉黑用户必须存在。
- 新增拉黑关系时，同一事务内删除双方已有关注关系并修正关注统计。
- 取消拉黑只删除拉黑关系，不自动恢复历史关注。

拉黑操作涉及多条关系写入，application 应使用稳定顺序的分布式锁或数据库唯一约束保护并发，避免双向关注统计漂移。

#### `UserCheckIn` / `UserCheckInStats`

`UserCheckIn` 是以 `(UserID, CheckInDate)` 为自然唯一键的签到记录实体，不作为独立聚合根暴露生命周期。它的生命周期由签到用例拥有。

`UserCheckInStats` 是签到统计聚合根：

- **标识**：`UserID`
- **统计字段**：`TotalDays`、`ContinuousDays`、`MaxContinuousDays`、`LastCheckInDate`
- **行为**：`RecordCheckIn`
- **不变量**：同一天只能签到一次；连续签到按业务日期计算。

月度签到图可以由 Redis bitmap 或查询模型加速，但 PostgreSQL 的 `user_check_ins` 和 `user_check_in_stats` 是签到真相源。

#### 聚合和对象分类

| 对象 | 分类 | 说明 |
| --- | --- | --- |
| `User` | 聚合根 | 拥有公开资料、头像引用、资料版本和消息设置的强一致生命周期 |
| `UserFollowStats` | 聚合根 | 维护关注统计，独立于 `User` 避免热点聚合 |
| `UserCheckInStats` | 聚合根 | 维护签到统计，按 `UserID` 独立更新 |
| `UserFollow` | 关系实体 | 自然键为 `(FollowerID, FollowingID)` |
| `UserBlock` | 关系实体 | 自然键为 `(BlockerID, BlockedID)` |
| `UserCheckIn` | 记录实体 | 自然键为 `(UserID, CheckInDate)` |
| `AccountID`、`NickName` 等 | 值对象 | 无独立生命周期，通过构造和 policy 保障约束 |

#### 非领域聚合

以下对象不建成领域聚合：

- 账号、密码、角色、JWT access / refresh token、Refresh Token 白名单：属于 Auth 服务。
- Redis lock、cache key、缓存 TTL：属于 infrastructure。
- `outbox_events`：跨服务可靠消息机制，不是领域聚合。
- 头像旧文件删除：事务后补偿副作用，不进入 `User` 聚合。

### 值对象

User 领域层优先用值对象表达有业务含义的基础值，避免在业务规则中到处传裸 `string` / `int64`。

| 值对象 | 含义 |
| --- | --- |
| `UserID` | 用户内部标识 |
| `AccountID` | Auth 账号引用 |
| `NickName` | 用户展示昵称 |
| `AvatarFileID` | Upload / File Service 拥有的头像文件引用 |
| `Bio` | 个人简介 |
| `ProfileVersion` | 资料版本，用于跨服务事件顺序判断 |
| `UserPair` | 两个用户之间的关系键，例如关注和拉黑 |
| `CheckInDate` | 按业务时区计算的签到日期 |

**核心约束**：

| 值对象 | 约束 |
| --- | --- |
| `AccountID` | 必填，引用 Auth 账号；唯一性由 User profile 初始化 contract 保证 |
| `NickName` | 默认取 Auth `username` 或前端提交昵称；非空更新时最大长度 50 |
| `Bio` | 可空或空字符串；最大长度 500 |
| `AvatarFileID` | 可空；仅保存 Upload / File Service 的文件引用，不保存文件事实 |
| `ProfileVersion` | 非负，资料更新时由 repository 原子递增 |
| `CheckInDate` | 按业务时区生成，不直接使用客户端传入时间判断“今天” |

### 领域服务

领域服务只承载纯业务规则，不依赖数据库、Redis、HTTP client、JWT 库或 MQ。

| 领域服务 | 职责 |
| --- | --- |
| `ProfileInitializationPolicy` | 校验 Auth account 初始化 profile 时的默认资料规则 |
| `ProfilePolicy` | 校验昵称、简介、头像引用等资料字段 |
| `RelationshipPolicy` | 校验不能关注自己、不能拉黑自己、被对方拉黑时不能关注 |
| `CheckInPolicy` | 判断同日重复签到和连续签到计算规则 |

账号状态、密码 hash / verify、JWT 签发、Refresh Token 白名单、登录标识唯一性和角色事实都不是 User 领域服务职责。它们由 Auth 拥有。

### Application 用例

User application 层按命令、查询分层组织 use case。application 拥有事务边界、权限上下文、幂等、端口调用、缓存失效和错误映射。

**命令用例（Commands）**：

- `CreateProfileForAccount`：由 Auth 注册流程同步调用或消费 `auth.account.registered` 事件，创建 User profile，写 `user.profile.created` outbox。
- `UpdateProfile`：更新用户资料，repository 原子递增 `profileVersion`，写 `user.profile.updated` outbox，事务后清理缓存和旧头像。
- `UpdateStrangerMessageSetting`：维护陌生人消息设置。
- `FollowUser` / `UnfollowUser`：维护关注关系、关注统计和关注事件。
- `BlockUser` / `UnblockUser`：维护拉黑关系，拉黑时解除双方关注并修正统计。
- `CheckIn`：插入签到记录、更新签到统计和月度 bitmap。

**查询用例（Queries）**：

- `GetMyProfile`、`GetUserDetail`、`GetUserSimple`、`BatchGetUserSimple`。
- `ListFollowers`、`ListFollowerShard`、`ListFollowings`、`GetFollowStats`、`CheckFollowing`。
- `ListBlockedUsers`、`CheckBlocked`。
- `GetStrangerMessageSetting`。
- `GetCheckInStats`、`GetMonthlyCheckInBitmap`。
- `ListAdminUsers`。
- 用户文章列表不属于 User 查询用例；调用方直接使用 Content 作者过滤接口。

**命令和查询分离**：

- 命令用例修改状态，返回简单成功/失败或用户 ID。
- 查询用例只读取，返回 DTO 或视图模型。
- 复杂列表和管理端查询不进入领域层；跨服务聚合查询优先由数据归属服务提供。

### 错误映射

User 服务典型错误及 HTTP 映射：

| 场景 | Domain/Ports 错误 | HTTP Status | 公开错误码 | 说明 |
| --- | --- | --- | --- | --- |
| 用户不存在 | `ErrUserNotFound` | 404 | `USER_NOT_FOUND` | 查询或操作不存在的用户 |
| 账号资料已存在 | `ErrProfileAlreadyExists` | 409 | `USER_ALREADY_EXISTS` | Auth account 重复初始化 User profile |
| 账号不可用 | `ErrAccountUnavailable` | 403 | `ACCOUNT_DISABLED` | Auth 判断账号禁用或不可互动 |
| 不能关注自己 | `ErrCannotFollowSelf` | 400 | `USER_CANNOT_FOLLOW_SELF` | 业务规则校验失败 |
| 不能拉黑自己 | `ErrCannotBlockSelf` | 400 | `USER_CANNOT_BLOCK_SELF` | 业务规则校验失败 |
| 已经关注 | `ErrAlreadyFollowing` | 409 | `USER_ALREADY_FOLLOWING` | 重复关注同一用户 |
| 未关注无法取消 | `ErrNotFollowing` | 409 | `USER_NOT_FOLLOWING` | 取消关注不存在的关系 |
| 今日已签到 | `ErrAlreadyCheckedIn` | 409 | `USER_ALREADY_CHECKED_IN` | 同一天重复签到 |
| 操作过于频繁 | `ErrTooManyRequests` | 429 | `RATE_LIMIT_EXCEEDED` | 并发控制锁超时或频率限制 |

详细错误响应格式见 `docs/contracts/errors.md`。Infrastructure 层的 `sql.ErrNoRows`、Redis nil、外部 SDK 错误必须由 adapter 翻译为上述 domain/ports 语义，再由 application 映射为公开错误码和 HTTP status。

### 运行机制策略

**Auth 账号状态依赖**：

- User 写操作使用 Gateway 注入的可信身份上下文作为 actor，不解析客户端 JWT。
- 如用例需要确认账号是否仍可互动，application 通过 Auth contract 查询账号状态；不能跨库读取 Auth 表。
- User 消费 `auth.account.disabled`、`auth.account.enabled` 或 `auth.role.changed` 事件时，只能用于本地缓存失效或查询加速，不改变 Auth 事实。

**关注和拉黑并发控制**：

- 关注 / 取消关注以 `(followerId, followingId)` 作为唯一业务键，数据库唯一约束是最终幂等保障。
- 写关注关系前可以使用 `LockManager` 对该关系键加短租约锁，降低重复请求下的唯一约束冲突和统计重试。
- 拉黑需要同时处理双向关注关系，锁 key 必须使用稳定顺序：先锁 `block:<blockerId>:<blockedId>`，再按 `(minUserId, maxUserId)` 顺序锁双向 follow key。
- 拿不到锁时返回“操作过于频繁”，不进入事务。
- 即使使用 Redis lock，repository 仍必须依赖唯一约束、UPSERT 和原子增减保证最终正确性；Redis lock 不是唯一正确性来源。

### Ports

Ports 放在 `services/zhicore-user/internal/user/ports`，按能力和用例族定义 consumer-side interface。

**核心端口**：

| Port | 职责 |
| --- | --- |
| `UserRepository` | User 聚合加载、保存、profile 初始化和资料乐观锁更新 |
| `UserQueryRepository` | 用户详情、简单资料、批量简单资料、管理端查询 |
| `FollowRepository` | 关注关系和关注统计的事务内写入 |
| `FollowQueryRepository` | 粉丝、关注列表、分片查询和关系判断 |
| `BlockRepository` | 拉黑关系写入和查询 |
| `CheckInRepository` | 签到记录和签到统计写入 |
| `CheckInQueryRepository` | 签到统计和月度记录查询 |

**基础设施机制端口**：

| Port | 职责 |
| --- | --- |
| `TransactionRunner` | 显式事务边界 |
| `OutboxPublisher` | 业务事务内追加 User 集成事件 |
| `LockManager` | 关注和拉黑并发控制 |
| `Clock` | 时间源和业务日期 |

**缓存和外部服务端口**：

| Port | 职责 |
| --- | --- |
| `UserCacheStore` | 用户详情、简单资料、陌生人消息设置缓存 |
| `FollowStatsCacheStore` | 关注统计缓存和失效 |
| `CheckInBitmapStore` | 月度签到 bitmap |
| `AuthAccountClient` | 查询 Auth 账号状态、初始化来源和必要的账号可用性 |
| `ContentPostClient` | User 需要读取 Content 事实时调用 Content contract |
| `FileDeletionClient` | 事务提交后删除旧头像文件引用 |
| `FileURLResolver` | 如查询响应需要头像 URL，由 adapter 调用 Upload 解析文件 URL |

端口不能暴露 `*gorm.DB`、`*redis.Client`、Gin context、HTTP DTO、ORM sentinel 或外部 SDK 类型。底层 not-found、重复键、Redis nil 等错误由 infrastructure adapter 翻译为 module-local 语义，再由 application 映射为公开错误。

### 一致性与事务边界

**Profile 初始化事务**：

```text
users
+ outbox_events(user.profile.created)
```

Auth 注册流程可以同步调用 `CreateProfileForAccount`，也可以发布 `auth.account.registered` 后由 User 消费初始化。无论采用哪种方式，User 本地只维护 profile 事实；关注统计和签到统计不要求初始化时预创建。`user_follow_stats` 和 `user_check_in_stats` 可以在首次关注、被关注或签到时通过 UPSERT 惰性创建；查询缺失统计时返回零值视图。

**资料更新事务**：

```text
users(profile_version = profile_version + 1)
+ outbox_events(user.profile.updated)
```

事务提交后：

- 删除用户详情、简单资料和陌生人消息设置缓存。
- best-effort 删除旧头像文件；失败进入补偿或日志告警，不回滚资料更新。

**关注事务**：

```text
user_follows 插入/删除
+ user_follow_stats 原子增减
+ outbox_events(user.followed / user.unfollowed)
```

事务提交后 best-effort 更新 Redis 关注统计缓存。

**拉黑事务**：

```text
user_blocks 插入
+ 删除 blocker -> blocked 关注关系（如果存在）
+ 删除 blocked -> blocker 关注关系（如果存在）
+ user_follow_stats 原子修正
```

取消拉黑只删除 `user_blocks`，不恢复历史关注。

**签到事务**：

```text
user_check_ins 插入（user_id, check_in_date 唯一）
+ user_check_in_stats 更新连续签到统计
```

Redis bitmap 在事务提交后更新，失败不影响 PostgreSQL 真相源。

**缓存失效策略**：

| 命令 | 缓存处理 | 失败语义 |
| --- | --- | --- |
| `CreateProfileForAccount` | 新 profile 默认无需失效；如写入欢迎缓存，必须事务后执行 | best-effort |
| `UpdateProfile` | 删除用户详情、简单资料、陌生人消息设置缓存 | best-effort，失败记录日志或补偿 |
| `UpdateStrangerMessageSetting` | 删除陌生人消息设置、用户详情和简单资料缓存 | best-effort |
| `FollowUser` / `UnfollowUser` | 更新或删除双方关注统计缓存、关注列表缓存 | best-effort，PostgreSQL 为真相源 |
| `BlockUser` / `UnblockUser` | 删除拉黑关系缓存、双方关注统计和关注列表缓存 | best-effort |
| `CheckIn` | 更新月度 bitmap 和签到统计缓存 | best-effort，PostgreSQL 为真相源 |

**Auth 相关缓存说明**：

- User 不维护角色缓存。角色和账号状态缓存由 Auth / Gateway 自己负责。
- 如果 User 为交互权限缓存了账号可用性，只能作为 Auth contract 的短 TTL 派生缓存；Auth 事件到达时删除派生缓存，不改变账号事实。

**热点用户保护**：

对于粉丝数超过阈值（例如 10 万）的热点用户，资料和关注统计缓存更新建议：

- 资料更新后，主动写回新缓存，而不是仅删除，避免缓存击穿。
- 关注统计更新后，使用 `INCR`/`DECR` 更新缓存值，而不是删除。
- 如缓存更新失败，降级为删除缓存，允许短时间内读穿到数据库。

第一阶段如无明确热点用户压力，可先采用简单的删除策略，后续根据监控调整。

### Go 包落点

目标目录：

```text
services/zhicore-user/
  api/http/              # HTTP 入站适配器
  internal/user/
    application/
      commands/          # 命令用例
      queries/           # 查询用例
    domain/
      user/              # User profile 聚合和值对象
      relationship/      # 关注、拉黑关系和值对象
      checkin/           # 签到记录和统计
      shared/            # 跨子域值对象和纯策略
      events/            # 领域事件定义
    ports/               # 端口接口定义
    infrastructure/
      postgres/          # PostgreSQL repository 和 mapper
      redis/             # 缓存、bitmap、lock
      rabbitmq/
        publishers/      # outbox publisher / dispatcher
      clients/           # Auth、Content、Upload client adapter
      jobs/              # outbox dispatcher、补偿任务
    runtime/
      module.go          # 依赖注入和模块装配
```

**分层依赖方向**：

```text
api/http -> application -> domain
                  \-> ports <- infrastructure
runtime -> api/http/application/infrastructure
```

第一版可以不机械拆出所有子包；如果代码量较小，`domain` 下可先保留少量文件。拆包标准是职责和依赖边界，而不是为了看起来像 DDD。

### 推荐首个实现切片

User 第一轮建议选“profile 初始化、更新资料和资料更新事件”，用于支撑 Auth 注册链路和 Content 作者快照链路，并覆盖核心 DDD 模式：

1. **Domain 层**：
   - 建 `User` profile 聚合和值对象：`UserID`、`AccountID`、`NickName`、`AvatarFileID`、`Bio`、`ProfileVersion`
   - 建 `UserProfileFactory`
   - 建 `ProfileInitializationPolicy`、`ProfilePolicy`
   - 定义 `UserProfileCreated`、`UserProfileUpdated` 领域事件

2. **Ports 层**：
   - 定义 `UserRepository`、`UserQueryRepository`、`AuthAccountClient`、`OutboxPublisher`、`TransactionRunner`、`Clock`

3. **Domain 测试**：
   - 测试 profile 初始化必须绑定 Auth account
   - 测试资料字段长度和空值规则
   - 测试 `UserProfileFactory` 创建默认昵称和初始 `profileVersion`

4. **Application 层**：
   - 建 `CreateProfileForAccount`、`UpdateProfile`、`GetMyProfile`、`BatchGetUserSimple`
   - 用内存 fake 实现端口，测试 profile 幂等、Auth account 不可用、资料版本事件写入

5. **Infrastructure 层**：
   - 实现 PostgreSQL `UserRepository`
   - 实现 PostgreSQL `UserQueryRepository` 的我的资料和批量简单资料查询
   - 实现 outbox writer
   - 实现 Auth typed client adapter

6. **HTTP 层**：
   - 实现 Auth 调用的 `POST /api/v1/internal/users/profile`
   - 实现 `GET /api/v1/users/me`
   - 实现 `PUT /api/v1/users/{userId}/profile`
   - 实现 `POST /api/v1/users/batch/simple`

这个切片能覆盖聚合根、值对象、工厂、领域事件、端口、application 编排、事务边界、outbox、Auth 依赖和最小查询 contract，同时不会一开始把关注、拉黑、签到、Admin 全部卷入。

## API 保留范围

必须保留以下 API 族：

- `/api/v1/users`：用户资料查询、简单资料批量查询、资料更新、私信设置。
- `/api/v1/internal/users/profile`：Auth 注册流程初始化 User profile；是否暴露为 HTTP 还是 typed client 由实现切片固定。
- `/api/v1/users/{userId}/followers`、`following`、`follow-stats`、关注检查。
- `/api/v1/users/{userId}/blocking`：拉黑、取消拉黑、拉黑检查。
- `/api/v1/users/{userId}/check-in`：签到、统计、月度记录。
- `/api/v1/admin/users`：管理端用户资料查询和关系信息 facade；账号禁用、启用、token 失效委托 Auth。

字段级 request/response 需要后续按目标 Go schema 固定到 `services/zhicore-user/api/http`；需要核对已发布行为时再参考既有 DTO。

## 数据归属

User 拥有：

- `users`
- `user_follows`
- `user_follow_stats`
- `user_blocks`
- `user_check_ins`
- `user_check_in_stats`
- User 服务自己的 `outbox_events`

`outbox_events` 是 User 服务私有表。仓库允许不同服务数据库中存在同名 outbox 表，例如 User 和 Comment 都可以有 `outbox_events`；表归所在服务数据库拥有，不使用跨服务共享 outbox 表。Content 当前使用 `outbox_event` 单数表名是服务私有历史选择，不要求 User 跟随。

内部主键使用 PostgreSQL sequence / identity。对外公开 ID 如需要隐藏数量增长，按 `docs/architecture/id-strategy.md` 单独设计。

## 事件

### 领域事件与集成事件

领域事件只存在于 User 领域和 application 内部，用于表达聚合内发生的业务事实。集成事件是跨服务 contract，必须有明确 consumer 或产品语义后才发布到 RabbitMQ。

**User 跨服务集成事件**：

| 事件 | 触发用例 | 主要 payload | 当前/目标 consumer | outbox 要求 |
| --- | --- | --- | --- | --- |
| `user.profile.created` | `CreateProfileForAccount` | `userId`、`accountId`、`nickname`、`avatarFileId`、`profileVersion`、`occurredAt` | Content、Search 或运营读模型如需要用户资料创建事实时消费 | 关键事件，使用 producer outbox |
| `user.profile.updated` | `UpdateProfile` | `userId`、`accountId`、`nickname`、`avatarFileId`、`bio`、`profileVersion`、`occurredAt` | Content 刷新作者快照；Search 如展示用户资料索引可消费或调用 User contract | 关键事件，使用 producer outbox |
| `user.followed` | `FollowUser` | `followerId`、`followingId`、`occurredAt` | Notification 创建关注通知；其他服务如需关系 read model 再登记 | 关键事件，使用 producer outbox |
| `user.unfollowed` | `UnfollowUser` | `followerId`、`followingId`、`occurredAt` | 关系 read model、通知聚合或运营分析如需关注关系撤销事实时消费 | 关键事件，使用 producer outbox |

**User 领域事件或内部事件**：

| 事件 | 默认处理 |
| --- | --- |
| `UserProfileCreated` | 默认用于 profile 初始化后的缓存和 outbox 编排 |
| `UserProfileUpdated` | 默认用于 profile 版本递增、缓存失效和 outbox 编排 |

**`user.profile.updated` payload 草案**：

```json
{
  "eventId": "evt_01HY...",
  "occurredAt": "2026-06-23T10:30:00Z",
  "userId": 12345,
  "accountId": 12345,
  "nickname": "Alice",
  "avatarFileId": "file_abc123",
  "bio": "Go backend engineer",
  "profileVersion": 7
}
```

**`user.followed` payload 草案**：

```json
{
  "eventId": "evt_01HY...",
  "occurredAt": "2026-06-23T10:31:00Z",
  "followerId": 12345,
  "followingId": 67890
}
```

以上是简化的 payload 示例。实际发布到 RabbitMQ 时应包含完整 envelope，包含 `eventType`（例如 `user.profile.updated`）、`payloadVersion`、`producer`（`zhicore-user`）、`aggregateType`、`aggregateId`、`requestId`、`traceId` 等字段，详见 `docs/contracts/events.md`。

关键跨服务事件必须用 producer outbox，不能在业务提交后直接发 RabbitMQ。没有明确 consumer 的领域事件不要为了“完整”提前发布为集成事件。

User 消费其他服务事件不是第一阶段重点；如果要维护本地快照，必须在对应服务文档记录用途和失效方式。

## User 对外 Contract

同步调用 contract 由 User 作为 provider 拥有，放在：

```text
libs/contracts/clients/user/
```

目标能力：

| Contract | 用途 | 典型 consumer | 约束 |
| --- | --- | --- | --- |
| `GetUserSimple(userId)` | 查询单个用户简单资料 | Comment、Message、Notification、Admin | 用户不存在返回 not-found 语义 |
| `BatchGetUserSimple(userIds)` | 批量查询用户简单资料，避免 N+1 | Content、Comment、Notification、Ranking 展示层 | 单次最多 100 个 ID；不存在的 ID 在结果中省略或返回 null，不报错 |
| `CheckBlocked(blockerId, blockedId)` | 判断拉黑关系 | Message 私信权限、Comment 互动权限 | 返回布尔值；任一用户不存在视为未拉黑 |
| `CheckFollowing(followerId, followingId)` | 判断关注关系 | Message 陌生人私信权限、Notification 偏好判断 | 返回布尔值；任一用户不存在视为未关注 |
| `GetStrangerMessageSetting(userId)` | 查询是否允许陌生人私信 | Message | 用户不存在或设置缺失时默认返回 false（不允许） |
| `ListFollowerShard(userId, cursorFollowerId, size)` | 粉丝 fanout 分片 | Notification 广播或作者订阅 fanout | 单页最多 1000 条；用 cursor 分页避免深分页 |
| `AdminQueryUsers(filter)` | 管理端用户资料查询 | Admin facade | 管理员权限由 Admin / Auth 校验；User 只返回资料事实 |

Admin 服务不直接导入 User `internal` 包，也不复制资料变更逻辑。账号禁用、启用或 token 失效通过 Auth command contract 完成；涉及用户资料的管理查询和资料修正才委托 User。

事件 consumer 归属：

- Content：消费 `user.profile.updated` 刷新作者快照。
- Notification：消费 `user.followed` 创建关注通知；如需要注册欢迎通知，优先消费 Auth 的 `auth.account.registered` 或 User 的 `user.profile.created`。
- Search：当前主要消费 Content 事件；如果未来建立用户搜索索引，再消费 `user.profile.updated` 或调用 User contract。
- Ranking：不拥有用户事实；排行榜展示需要用户资料时优先批量调用 User contract 或使用上游事件携带的稳定快照。
- Comment / Message / Admin：默认通过同步 User contract 获取用户摘要、关系判断或管理命令，不消费 User 事件作为事实源。

## 跨服务依赖

- Upload：头像、图片资源只保存 `file_id` 或公开 URL 引用，文件事实仍归 Upload/File Service。
- Auth：账号、凭证、角色、账号状态和 token 生命周期归 Auth；User 通过 contract 或事件获取必要派生事实。
- Content：用户文章列表 facade 通过 Content contract 委托。
- Message：私信权限可以查询 User 的拉黑、私信设置和关注关系。

## Go 目标落点

- HTTP：`services/zhicore-user/api/http`
- Application：`services/zhicore-user/internal/user/application`
- Domain：`services/zhicore-user/internal/user/domain`
- Ports：`services/zhicore-user/internal/user/ports`
- Infrastructure：`services/zhicore-user/internal/user/infrastructure`
- Runtime：`services/zhicore-user/internal/user/runtime/module.go`

## 实现风险

- Java 中部分 ID 依赖 `IdGeneratorFeignClient`；Go 目标不默认迁移该依赖。
- 用户资料更新会影响 Content 作者快照，必须通过事件和版本号处理，不允许 Content 直接读 User 数据库。
- Admin 账号操作必须走 Auth 的 command contract；User 只处理资料、关系和签到事实。

## 下一步

- 按 `docs/contracts/http-schema-template.md` 提取 User HTTP 字段级 contract。
- 生成 User migration 草案，重点核对：
  - `users` 表：`account_id` 唯一引用、`profile_version` 默认值 0 和非空约束、昵称、头像、简介和陌生人消息设置；登录标识、密码、角色和账号状态迁移到 Auth。
  - `user_follows`：`(follower_id, following_id)` 联合唯一索引、防止自我关注的 `CHECK (follower_id != following_id)` 约束
  - `user_blocks`：`(blocker_id, blocked_id)` 联合唯一索引、防止自我拉黑的 `CHECK (blocker_id != blocked_id)` 约束
  - `user_check_ins`：`(user_id, check_in_date)` 联合唯一索引
  - `user_follow_stats` 和 `user_check_in_stats`：计数字段的非负 `CHECK` 约束（例如 `CHECK (followers_count >= 0)`）
  - `outbox_events`：`event_id` 唯一索引、`status` 字段（pending/published/failed）和 `created_at`/`updated_at` 索引用于 dispatcher claim
- 先实现”profile 初始化、更新资料和资料更新事件”核心切片。
- 先写 domain 层测试，验证 `User` 聚合不变量和资料事件；测试覆盖率和测试文件规模按 `docs/architecture/testing.md` 执行。
- 再写 application 层测试，验证 profile 初始化、Auth account 不可用和资料更新 outbox 编排。
- 最后接入 PostgreSQL / Redis / HTTP，完成端到端验证。

## DDD 设计总结

本文档按 DDD 战术模式重新设计 User 服务，关键点：

1. **User 聚合只维护资料和消息设置**：账号、凭证、角色和 token 生命周期已抽离到 Auth；关注统计、签到统计不混入 User 热点聚合。
2. **资料版本显式建模**：`profileVersion` 由数据库原子递增，保证 `user.profile.updated` 能被 Content 按版本顺序应用。
3. **Auth 运行机制归 Auth 服务**：JWT、Refresh Token 白名单和 Redis 失败策略不进入 User。
4. **关系写模型有明确事务边界**：关注、拉黑和统计修正必须在本地事务内收口。
5. **User 不复制跨服务数据**：用户文章列表直接走 Content contract，当前不提供 User facade。
6. **端口按能力分组**：避免把 repository、cache、lock、client 混成宽泛 `UserService`。
7. **分层依赖方向符合 Onion Architecture**：domain 不依赖 HTTP、Redis、PostgreSQL、JWT、Auth client 或 RabbitMQ。
