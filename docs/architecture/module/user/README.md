# User 模块架构

`user` 模块对应 `zhicore-user` 服务内的用户资料、关系和用户状态上下文。Go 实现按 `api/http -> application -> domain/ports -> infrastructure` 的依赖方向落点；本文档描述目标设计，不表示当前 Go 代码已经完成。

## 模块职责

- 管理用户资料、公开 `publicId`、唯一 `nickname`、头像文件引用、简介、陌生人私信设置和资料版本。
- 管理 User 自己的业务状态：`ACTIVE`、`DEACTIVATED`、`DELETED`。
- 管理关注、粉丝、拉黑、关系查询和关注统计。
- 为 Content、Comment、Message、Notification 等服务提供用户摘要、用户可用性、拉黑和关注关系 typed client contract。
- 生产 `user.profile.*`、`user.followed`、`user.unfollowed`、`user.blocked`、`user.unblocked`、`user.deactivated`、`user.deleted`、`user.restored` 等 User 集成事件。

## 边界

User 不拥有账号、凭证、角色、封禁、JWT 或 token 生命周期：

- `accountId` 是 Auth 拥有的账号标识；User 只在 `users.account_id` 保存唯一引用。
- `userId` 是 User 拥有的业务用户内部标识；服务间写路径、关系表、Comment 作者引用和 Gateway `X-User-Id` 使用它。
- `publicId` 是 User 生成和持久化的外部公开标识；前端 URL、HTTP 响应和作者展示用它。
- 管理员封禁归 Auth，用 `BANNED` 表达；User 不保存 `banned` 状态。
- 普通用户主动注销由 Auth 编排：Auth 调 User `DeactivateUserProfile` 后失效账号和 token。
- 管理员资料逻辑删除归 User，通常由 Admin facade 调 User `MarkUserDeleted`；它用于资料治理和合规隐藏，不等同 Auth 封禁。
- 头像文件事实归 Upload / File Service；User 只保存 `avatarFileId`，前端 HTTP 响应中的 `avatarUrl` 是运行时派生字段。
- 用户文章列表不属于 User facade；用户主页文章列表直接调用 Content 作者过滤接口。

Gateway 保持薄入口：校验 JWT、清理客户端伪造身份 header、注入当前操作者内部 `X-User-Id` 并路由请求。Gateway 不解析业务目标 `publicId`。

## 子域

| 子域 | 职责 | 主要存储 |
| --- | --- | --- |
| Profile | 用户资料、`publicId`、唯一 `nickname`、头像引用、简介、陌生人私信设置、资料版本和 User 业务状态 | `users` |
| Relationship | 关注、取关、粉丝/关注列表、关注关系判断和关注统计 | `user_follows`、`user_follow_stats` |
| Block | 拉黑、解除拉黑、拉黑列表和批量拉黑检查 | `user_blocks` |
| Check-in | 签到记录、连续签到统计和月度签到图 | 后续 `user_check_ins`、`user_check_in_stats`、Redis bitmap |
| Integration | User outbox、跨服务事件、缓存失效和补偿 | `outbox_events` |

## API Family

字段级 HTTP schema 后续固定到 `services/zhicore-user/api/http/`。模块级 API 背后设计见 [api.md](api.md)。

### 前端公开 API

前端公开 API 使用 `publicId` 作为目标用户定位符，不暴露内部 `userId`：

- `GET /api/v1/users/me`
- `PATCH /api/v1/users/me/profile`
- `GET /api/v1/users/{publicId}`
- `POST /api/v1/users/{publicId}/follow`
- `DELETE /api/v1/users/{publicId}/follow`
- `GET /api/v1/users/{publicId}/followers`
- `GET /api/v1/users/{publicId}/following`
- `POST /api/v1/users/{publicId}/block`
- `DELETE /api/v1/users/{publicId}/block`
- `GET /api/v1/users/me/blocked`

### 服务间 / Internal API

服务间 API 和 typed client 使用内部 `userId`、`accountId`，不使用 `publicId` 做权限和关系键：

- `CreateProfileForAccount(accountId, username)`
- `DeactivateUserProfile(accountId 或 userId)`
- `BatchGetUserSimple(userIds)`
- `BatchGetUserAvailability(userIds)`
- `BatchCheckBlocked(pairs)`
- `CheckFollowing(followerId, followingId)`
- `GetStrangerMessageSetting(userId)`

### Admin API

本轮只固定边界，不提取字段级 endpoint：

- User 拥有资料相关管理查询、资料修正、逻辑删除和恢复。
- Auth 拥有账号封禁、启用、角色调整和 token 失效。
- Admin facade 后续分别委托 User 和 Auth contract，并记录审计。

## 当前状态

- 已确认：身份模型、Auth/User 分工、`publicId` 解析边界、昵称唯一规则、头像 URL 边界、Profile 状态、Block/Follow 关系、事件、缓存和核心 schema。
- 已固定：User 运行期 resilience、业务限流和服务间 caller identity 约束。
- 已记录：设计压测决策见 [decision-log.md](decision-log.md)。
- 待提取：User HTTP 字段级 schema、typed client Go DTO、migration SQL 和行为测试清单。

## 首批实现切片

首批顺序按依赖关系组织：

1. **Profile 基础**
   - `CreateProfileForAccount`
   - `GetMe`
   - `GetUserByPublicId`
   - `UpdateProfile`
   - `BatchGetUserSimple`
   - `BatchGetUserAvailability`

2. **Profile 状态**
   - `DeactivateUserProfile`
   - `MarkUserDeleted`
   - `RestoreDeletedUserProfile`

3. **Block**
   - `BlockUser`
   - `UnblockUser`
   - `ListBlockedUsers`
   - `BatchCheckBlocked`

4. **Follow**
   - `FollowUser`
   - `UnfollowUser`
   - `ListFollowers`
   - `ListFollowing`
   - `CheckFollowing`
   - `FollowStats`

5. **后续切片**
   - Check-in。
   - Admin 查询字段级 contract。
   - 关注统计对账 / 重建。
   - 用户资料合规修正细节。

Profile 状态必须早于 Block/Follow，因为关系写操作需要 `user_status=ACTIVE` guard。

## 文档拆分

| 文档 | 内容 |
| --- | --- |
| [api.md](api.md) | API 背后的业务流程、权限、状态机、副作用和 use case 追踪。 |
| [service.md](service.md) | Application service、事务边界、幂等、错误映射、缓存失效和实现切片。 |
| [domain.md](domain.md) | 聚合、实体、值对象、不变量、领域服务和工厂。 |
| [ports.md](ports.md) | repository、cache、client、event publisher、outbox 和 external adapter 端口归属。 |
| [data-events.md](data-events.md) | 数据归属、目标 schema 草案、缓存 key、事件 payload 和跨服务一致性。 |
| [runtime-resilience.md](runtime-resilience.md) | User timeout、retry、熔断、降级、max-in-flight 和 provider operation 语义。 |
| [rate-limiting.md](rate-limiting.md) | User 业务限流矩阵、Redis 故障 fail-open / fail-closed 规则和 `RateLimiter` 决策语义。 |
| [decision-log.md](decision-log.md) | 设计压测中已确认的决策、原因和后续依赖。 |
