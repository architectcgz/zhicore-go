# User Application Service 设计

Application 层拥有事务边界、端口调用、幂等、缓存失效和错误映射。Domain 只表达规则，不直接访问数据库、Redis、HTTP client 或 MQ。

## 命令用例

| Use case | 职责 |
| --- | --- |
| `CreateProfileForAccount` | Auth 注册后同步调用；创建 User profile、生成 `publicId`、写 `user.profile.created` outbox。 |
| `UpdateProfile` | 更新 nickname、avatarFileId、bio、strangerMessageAllowed；公开资料变化时递增 `profileVersion` 并写 `user.profile.updated`。 |
| `DeactivateUserProfile` | Auth 主动注销编排中调用；将 User profile 置为 `DEACTIVATED` 并写 `user.deactivated`。 |
| `MarkUserDeleted` | Admin facade 调用；将 User profile 置为 `DELETED` 并写 `user.deleted`。 |
| `RestoreDeletedUserProfile` | Admin facade 调用；将 `DELETED` 恢复为 `ACTIVE` 并写 `user.restored`。 |
| `FollowUser` | 解析 target publicId，校验双方 `ACTIVE` 和拉黑关系，写关注关系、统计和 `user.followed`。 |
| `UnfollowUser` | 删除关注关系、修正统计和写 `user.unfollowed(reason=USER_REQUEST)`。 |
| `BlockUser` | 写拉黑关系，同事务解除双方关注并写 block/unfollow 事件。 |
| `UnblockUser` | 删除拉黑关系并写 `user.unblocked`。 |

## 查询用例

| Use case | 职责 |
| --- | --- |
| `GetMyProfile` | 返回当前用户资料；前端 HTTP 可解析 `avatarUrl`。 |
| `GetUserProfileByPublicId` | 按 `publicId` 查询公开资料；`DELETED` 用户主页建议 404。 |
| `BatchGetUserSimple` | 按内部 `userId` 批量返回摘要；缺失项省略并返回 `missingUserIds`。 |
| `BatchGetUserAvailability` | 写路径 guard；返回 User profile 是否存在且 `ACTIVE`。 |
| `ListFollowers` / `ListFollowing` | cursor 分页返回用户摘要和关系创建顺序。 |
| `ListBlockedUsers` | cursor 分页返回当前用户拉黑列表。 |
| `BatchCheckBlocked` | 批量判断 `(blockerUserId, blockedUserId)` 是否存在拉黑关系。 |
| `CheckFollowing` | 判断关注关系是否存在。 |
| `GetStrangerMessageSetting` | 给 Message 查询陌生人私信设置；用户不存在返回 false。 |
| `ListFollowerShard` | 给 Notification fanout 按 `audienceClass` / `activeSince` 分片读取活跃粉丝；依赖不可用返回 degraded，不能以空 shard 冒充成功，也不能从 `HOT` 静默 fallback 到 `ALL`。 |

## 事务边界

### Profile 初始化

```text
事务内：
  users 插入
  outbox_events(user.profile.created)

事务外：
  Auth 接收成功结果
  缓存可选回填
```

幂等规则：

- 同一 `accountId` 已有 profile 时返回已有 profile。
- 重复初始化不覆盖资料、不重新生成 `publicId`、不重置 `profileVersion`、不重放 `user.profile.created`。
- 默认 nickname 被占用时返回 `USER_NICKNAME_TAKEN`，Auth 注册链路按 User 初始化失败处理并补偿。

### Profile 更新

```text
事务前：
  如 avatarFileId 非空，调用 File service 校验文件存在、图片类型、状态可引用

事务内：
  users 更新
  如公开资料变化：profile_version = profile_version + 1 RETURNING profile_version
  如公开资料变化：outbox_events(user.profile.updated)

事务后：
  删除 UserSimple、profile、publicId 相关缓存
  前端 HTTP 查询时按需解析 avatarUrl
```

公开资料字段：

- `nickname`
- `avatarFileId`
- `bio`

设置字段：

- `strangerMessageAllowed`

只更新 `strangerMessageAllowed` 不递增 `profileVersion`，不发布 `user.profile.updated`。

### Profile 状态

`DeactivateUserProfile`：

```text
事务内：
  users.user_status: ACTIVE -> DEACTIVATED
  outbox_events(user.deactivated)
```

- 由 Auth 编排主动注销时调用。
- 幂等：已 `DEACTIVATED` 返回成功，不重复发事件。
- 不吊销 token，不禁用账号；Auth 负责账号和 token。

`MarkUserDeleted`：

```text
事务内：
  users.user_status: ACTIVE/DEACTIVATED -> DELETED
  记录 operatorId、reason、deletedAt
  outbox_events(user.deleted)
```

- 由 Admin facade 调用。
- 用于资料治理和合规隐藏，不等同 Auth 封禁。
- 不物理删除 User，也不清理关注和拉黑关系。

`RestoreDeletedUserProfile`：

```text
事务内：
  users.user_status: DELETED -> ACTIVE
  记录 operatorId、reason、restoredAt
  outbox_events(user.restored)
```

- 不恢复 Auth 状态。
- 用户主动 `DEACTIVATED` 不由管理员直接恢复。

### Follow

`FollowUser`：

```text
事务前：
  解析 target publicId -> targetUserId
  校验 actor 和 target 都是 ACTIVE
  校验 actor != target
  批量检查任一方向拉黑

事务内：
  插入 user_follows(follower_id, following_id)
  UPSERT / 更新 user_follow_stats
  outbox_events(user.followed)
```

幂等：

- 已关注再次调用：成功，不重复统计，不重复事件。
- 任一方向拉黑：返回 `USER_INTERACTION_BLOCKED`。

`UnfollowUser`：

```text
事务内：
  删除 user_follows
  如真实删除：修正 user_follow_stats
  如真实删除：outbox_events(user.unfollowed reason=USER_REQUEST)
```

- 未关注再次调用：幂等成功，不发事件。
- 目标非 ACTIVE 时仍允许清理历史关系。

### Block

`BlockUser`：

```text
事务前：
  解析 target publicId -> targetUserId
  校验 actor 和 target 都是 ACTIVE
  校验 actor != target

事务内：
  插入 user_blocks(blocker_id, blocked_id)
  删除 actor -> target 关注关系（如果存在）
  删除 target -> actor 关注关系（如果存在）
  修正 user_follow_stats
  outbox_events(user.blocked)
  如删除关注：outbox_events(user.unfollowed reason=BLOCKED)
```

幂等：

- 已拉黑再次调用：成功，不重复统计，不重复事件。
- 解除拉黑不恢复历史关注。

`UnblockUser`：

```text
事务内：
  删除 user_blocks
  如真实删除：outbox_events(user.unblocked)
```

- 未拉黑再次调用：幂等成功，不发事件。
- 目标非 ACTIVE 时仍允许清理历史关系。

## 缓存失效

缓存不是首批正确性依赖；PostgreSQL 是 Profile、关系和统计真相源。

| 命令 | 缓存处理 |
| --- | --- |
| `CreateProfileForAccount` | 可写入 `publicId -> userId` 和 UserSimple 缓存；失败不影响事务。 |
| `UpdateProfile` | 删除 UserSimple、profile、publicId 相关缓存；URL 不缓存进 profile 事实。 |
| `DeactivateUserProfile` / `MarkUserDeleted` / `RestoreDeletedUserProfile` | 删除 UserSimple、availability、profile、关系列表展示缓存。 |
| `FollowUser` / `UnfollowUser` | 删除双方 follow stats、followers/following 列表缓存。 |
| `BlockUser` / `UnblockUser` | 删除 block pair、blocked list、双方 follow stats 和关系列表缓存。 |

关系权限类检查优先直接查 PostgreSQL；后续如加缓存，必须补失效测试和权限延迟说明。

## 错误映射

| 语义 | HTTP Status | Symbolic code |
| --- | --- | --- |
| 用户不存在 | 404 | `USER_NOT_FOUND` |
| 用户非 ACTIVE | 403 | `USER_NOT_ACTIVE` |
| nickname 无效 | 400 | `USER_NICKNAME_INVALID` |
| nickname 已占用 | 409 | `USER_NICKNAME_TAKEN` |
| bio 无效 | 400 | `USER_BIO_INVALID` |
| 头像不可引用 | 400 | `USER_AVATAR_INVALID` |
| 不能关注自己 | 400 | `USER_CANNOT_FOLLOW_SELF` |
| 不能拉黑自己 | 400 | `USER_CANNOT_BLOCK_SELF` |
| 互动被拉黑阻止 | 403 | `USER_INTERACTION_BLOCKED` |
| cursor 非法 | 400 | `USER_CURSOR_INVALID` |
| 下游 File service/Auth 不可用 | 502 / 503 | 项目通用下游不可用错误 |

Infrastructure adapter 必须把 SQL not-found、唯一约束冲突、HTTP 下游错误和 Redis nil 翻译为 module-local 错误，再由 application 映射为公开错误。

## 实现切片测试重点

1. Profile 基础：
   - `accountId` 幂等初始化。
   - nickname 唯一冲突。
   - nickname trim、大小写敏感、长度 15 和危险字符。
   - bio 长度 100、纯文本和危险字符。
   - avatarFileId 校验失败不写入。
   - `profileVersion` 只在公开资料变化时递增。

2. Profile 状态：
   - 非 ACTIVE 禁止新增写操作。
   - `DEACTIVATED` 幂等。
   - `DELETED` 和恢复幂等。
   - 状态变更不清理关系。

3. Block：
   - 拉黑自己失败。
   - 拉黑解除双方关注和统计。
   - 重复拉黑不重复事件。
   - 非 ACTIVE 目标不能新增拉黑。

4. Follow：
   - 关注自己失败。
   - 任一方向拉黑时关注失败。
   - 重复关注 / 取关幂等。
   - cursor 分页按内部关系 `id DESC`。
