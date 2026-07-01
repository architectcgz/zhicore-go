# User API 背后设计

本文只描述 API 背后的业务流程、权限、状态机和 use case 追踪；字段级 HTTP schema 放在 `services/zhicore-user/api/http/`。

## 鉴权上下文

| API 类型 | 鉴权 | 说明 |
| --- | --- | --- |
| 公开用户资料查询 | 匿名 / 登录用户 | 目标用户由 path `publicId` 定位；User 本地解析为内部 `userId`。 |
| 当前用户资料查询和更新 | 登录用户 | Gateway 注入内部 `X-User-Id`；handler 不从 request body 接收当前操作者。 |
| Follow / Block 命令 | 登录用户 | actor 使用内部 `X-User-Id`；目标用户使用 path `publicId`。 |
| 服务间 typed client | 服务间认证 | 使用内部 `userId`、`accountId`，不使用 `publicId` 做关系和权限键。 |
| Admin facade 调用 | 管理员 | Admin 负责权限和审计；User command 仍校验目标资料状态和幂等。 |

客户端伪造的 `X-User-*` header 必须由 Gateway 清理后重新注入。Gateway 不解析业务目标 `publicId`。

## 标识规则

| 标识 | Owner | 用途 |
| --- | --- | --- |
| `accountId` | Auth | 账号、凭证、角色、封禁、token 生命周期；User 只保存唯一引用。 |
| `userId` | User | User 内部主键；服务间写路径、关系表、Comment 作者引用和 Gateway `X-User-Id` 使用它。 |
| `publicId` | User | 外部公开用户 ID；前端 URL、HTTP response 和作者展示使用它。 |
| `nickname` | User | 全局唯一展示名；不替代 `publicId` 或 `userId`。 |

User 本地负责 `publicId -> userId` 解析。其他服务不能直连 User 数据库解析。

## 前端公开 API

| Endpoint | Use case | 主要副作用 |
| --- | --- | --- |
| `GET /api/v1/users/me` | `GetMyProfile` | 无业务写入；可运行时解析 `avatarUrl`。 |
| `PATCH /api/v1/users/me/profile` | `UpdateProfile` | 更新 nickname、avatarFileId、bio、strangerMessageAllowed；公开资料变化时递增 `profileVersion` 并写 outbox。 |
| `GET /api/v1/users/{publicId}` | `GetUserProfileByPublicId` | 无业务写入；`DELETED` 建议按 404 返回。 |
| `POST /api/v1/users/{publicId}/follow` | `FollowUser` | 写关注关系、统计和 `user.followed` outbox。 |
| `DELETE /api/v1/users/{publicId}/follow` | `UnfollowUser` | 删除关注关系、统计和 `user.unfollowed` outbox。 |
| `GET /api/v1/users/{publicId}/followers` | `ListFollowers` | 无业务写入；cursor 分页。 |
| `GET /api/v1/users/{publicId}/following` | `ListFollowing` | 无业务写入；cursor 分页。 |
| `POST /api/v1/users/{publicId}/block` | `BlockUser` | 写拉黑关系；同事务解除双方关注并写关系事件。 |
| `DELETE /api/v1/users/{publicId}/block` | `UnblockUser` | 删除拉黑关系并写 `user.unblocked` outbox。 |
| `GET /api/v1/users/me/blocked` | `ListBlockedUsers` | 无业务写入；cursor 分页。 |

前端 HTTP 响应应返回 `avatarUrl`，但 `avatarUrl` 是 File service 派生展示字段，不落库、不进 User 事件、不进默认 typed client。

## 服务间 / Internal API

| Contract | 用途 | Consumer | 约束 |
| --- | --- | --- | --- |
| `CreateProfileForAccount(accountId, username)` | Auth 注册后初始化 User profile | Auth | 按 `accountId` 幂等；默认 nickname 冲突时返回占用错误。 |
| `DeactivateUserProfile(accountId 或 userId)` | 普通用户注销时停用资料 | Auth | 幂等；只改 User profile 状态，不吊销 token。 |
| `BatchGetUserSimple(userIds)` | 批量用户摘要 | Content、Comment、Notification、Ranking 展示层 | 入参内部 `userId`；最多 100；缺失项省略并返回 `missingUserIds`。 |
| `BatchGetUserAvailability(userIds)` | 写路径用户可用性 guard | Comment、Message | 只表达 User profile 是否存在且 `ACTIVE`；不复制 Auth 账号状态。 |
| `BatchCheckBlocked(pairs)` | 批量拉黑关系判断 | Comment、Message | pair 使用内部 `userId`；任一用户缺失返回 `blocked=false` 并记录观测。 |
| `CheckFollowing(followerId, followingId)` | 关注关系判断 | Message、Notification | 任一用户缺失返回 `false`。 |
| `GetStrangerMessageSetting(userId)` | 陌生人私信设置 | Message | 正常新用户默认 `true`；用户不存在或设置缺失时返回 `false`。 |
| `ListFollowerShard(cursor, limit)` | Notification fanout 分片读取粉丝 | Notification | 内部 cursor 分页；依赖不可用返回 `SERVICE_DEGRADED`，不能把空 shard 当成功。 |

`BatchGetUserSimple` 只返回 `avatarFileId`，不返回 `avatarUrl`。需要展示 URL 的调用方批量走 File service 解析，或由面向前端的 owning service 自己解析。

## Admin 边界

- `GET /api/v1/admin/users`、用户资料修正、逻辑删除和恢复归 User 提供，可由 Admin facade 暴露和审计。
- 账号封禁、启用、角色调整、强制 token 失效归 Auth。
- 管理员封禁使用 Auth `BANNED`；User 不保存 banned 状态。
- 管理员资料删除使用 User `DELETED`，用于隐藏头像、昵称、简介、主页等资料，不等同封禁。

## Profile 初始化流程

```text
Auth RegisterAccount
-> Auth 本地 account / credential / role / outbox
-> 调用 User CreateProfileForAccount(accountId, username)
-> User 创建 users 行、publicId、nickname、profileVersion=0、user.profile.created
-> Auth 返回注册结果
-> 如 User 初始化失败，Auth 不向客户端承诺完整注册成功，并登记补偿
```

默认值：

- `nickname` 使用 Auth 传入的 `username`。
- `avatarFileId` 为空。
- `bio` 为空字符串。
- `strangerMessageAllowed` 默认 `true`。
- User 不保存 email。

如果默认 `nickname` 已被占用，User 返回昵称占用错误，不自动生成后缀。

## Profile 更新流程

`PATCH /api/v1/users/me/profile` 首批允许更新：

- `nickname`
- `avatarFileId`
- `bio`
- `strangerMessageAllowed`

不允许更新：

- `publicId`
- `accountId`
- `userId`
- `profileVersion`

公开资料字段变化时才递增 `profileVersion` 并发布 `user.profile.updated`。`strangerMessageAllowed` 更新不触发 `user.profile.updated`。

头像规则：

- `avatarFileId` 为空表示清除头像。
- 非空时，User 写入前同步调用 File service 校验文件存在、类型为图片且状态可引用。
- 校验失败不写 profile、不递增版本、不发事件。
- 旧头像首批不自动删除。

## 状态和关系写操作

User profile 状态：

- `ACTIVE`
- `DEACTIVATED`
- `DELETED`

关系写操作规则：

- actor 非 `ACTIVE`：禁止所有 User 写操作。
- 新增关注 / 拉黑要求目标用户 `ACTIVE`。
- 取关 / 解除拉黑允许清理历史关系，即使目标已注销或删除。
- 注销 / 删除不清理 `user_follows` 和 `user_blocks`。
- 列表查询返回非 `ACTIVE` 用户状态或占位摘要。

## 分页

关注、粉丝和拉黑列表使用 cursor 分页：

- 默认 `limit=20`。
- 最大 `limit=100`。
- `user_follows` 和 `user_blocks` 使用内部递增 `id` 作为 cursor 锚点。
- 列表按 `id DESC` 排序。
- cursor 不透明，HTTP 不暴露内部 SQL 列。

Notification fanout 的 `ListFollowerShard` 使用服务间内部 cursor，不复用前端 cursor。

## 错误语义

User 公开错误先固定 symbolic code 和 HTTP status，后续在 `docs/contracts/error-codes.md` 登记数字码。body `code` 不能退化成 HTTP status。

| 场景 | HTTP Status | Symbolic code |
| --- | --- | --- |
| 用户不存在 | 404 | `USER_NOT_FOUND` |
| 用户非 ACTIVE | 403 | `USER_NOT_ACTIVE` |
| nickname 为空、过长或含危险字符 | 400 | `USER_NICKNAME_INVALID` |
| nickname 已占用 | 409 | `USER_NICKNAME_TAKEN` |
| bio 超过 100 或含危险字符 | 400 | `USER_BIO_INVALID` |
| 头像文件不可引用 | 400 | `USER_AVATAR_INVALID` |
| 不能关注自己 | 400 | `USER_CANNOT_FOLLOW_SELF` |
| 不能拉黑自己 | 400 | `USER_CANNOT_BLOCK_SELF` |
| 任一方向拉黑导致不能关注/互动 | 403 | `USER_INTERACTION_BLOCKED` |
| cursor 非法 | 400 | `USER_CURSOR_INVALID` |
| File service/Auth 下游不可用 | 502 / 503 | 项目通用下游不可用错误 |

## 字段命名

字段统一使用 `nickname`：

- User HTTP / typed client：`nickname`
- User DB：`users.nickname`
- User 事件：`nickname`
- Comment 作者摘要：`author.nickname`
- Content 作者快照：`authorNickname` / `ownerNickname`

不继续扩散 `displayName`、`authorName`、`owner_name` 的新口径；既有旧文档后续同步修正。
