# User Domain 设计

## `User` 聚合

`User` 是用户资料聚合根，拥有 User 资料生命周期和公开展示字段。

- **内部标识**：`UserID`，User 服务拥有的内部业务用户 ID。
- **外部标识**：`PublicID`，前端和 HTTP 使用的公开用户 ID。
- **账号引用**：`AccountID`，Auth 账号引用，唯一但不作为 User 主键。
- **资料字段**：`Nickname`、`AvatarFileID`、`Bio`、`ProfileVersion`。
- **设置字段**：`StrangerMessageAllowed`。
- **状态字段**：`UserStatus`，包括 `ACTIVE`、`DEACTIVATED`、`DELETED`。
- **行为**：`CreateProfileForAccount`、`UpdateProfile`、`Deactivate`、`MarkDeleted`、`RestoreDeleted`、`UpdateStrangerMessageSetting`。
- **领域事件**：`UserProfileCreated`、`UserProfileUpdated`、`UserDeactivated`、`UserDeleted`、`UserRestored`。

## `User` 不变量

- `accountId` 必填，并在 User 本地唯一。
- `publicId` 必填，并在 User 本地唯一。
- `nickname` 必填，trim 后长度 `1..15` 个字符，全局唯一。
- `nickname` 唯一性大小写敏感：`Alice` 和 `alice` 可共存。
- `nickname` 禁止危险字符和控制字符，例如换行、`<`、`>`。
- 非 `ACTIVE` 用户继续占用 `nickname`。
- `bio` 最大 100，空值保存为空字符串。
- `bio` 允许少量换行，按纯文本展示；禁止危险字符和控制字符，不允许 raw HTML 或 Markdown 渲染语义。
- `profileVersion` 非负，只在公开展示资料变化时递增。
- `strangerMessageAllowed` 默认 `true`，更新它不触发 `user.profile.updated`。
- 普通用户不能修改 `publicId`、`accountId`、`userId`、`profileVersion`。
- 非 `ACTIVE` 用户禁止资料更新、关注、拉黑、签到等写操作。

## `UserStatus`

| 状态 | 含义 | 写路径 | 读路径 |
| --- | --- | --- | --- |
| `ACTIVE` | 正常用户资料 | 允许资料、关系和签到写操作 | 正常展示 |
| `DEACTIVATED` | 用户主动注销 / 停用资料 | 禁止新增写操作；允许取关和解除拉黑清理历史关系 | 返回最小摘要或“已注销用户”占位 |
| `DELETED` | 管理员资料治理 / 合规隐藏 | 禁止新增写操作；允许取关和解除拉黑清理历史关系 | 用户主页建议 404；历史内容可显示占位 |

管理员封禁不进入 `UserStatus`，归 Auth `BANNED`。

## `UserFollow` / `UserFollowStats`

`UserFollow` 是关注关系实体：

- **标识**：内部递增 `id`，仅作为 cursor 锚点，不对外暴露。
- **自然键**：`(FollowerID, FollowingID)`。
- **行为**：创建关注、取消关注。
- **事件**：`UserFollowed`、`UserUnfollowed`。

`UserFollowStats` 是可重建读模型：

- **标识**：`UserID`。
- **字段**：`FollowersCount`、`FollowingCount`。
- **不变量**：计数不能为负。
- **事实源**：`user_follows`。

关注规则：

- 不能关注自己。
- actor 和 target 必须都是 `ACTIVE`。
- 任一方向存在拉黑时不能关注。
- 重复关注幂等成功，不重复更新统计，不重复发事件。
- 重复取关幂等成功，不发事件。

## `UserBlock`

`UserBlock` 是拉黑关系实体：

- **标识**：内部递增 `id`，仅作为 cursor 锚点，不对外暴露。
- **自然键**：`(BlockerID, BlockedID)`。
- **行为**：拉黑、解除拉黑。
- **事件**：`UserBlocked`、`UserUnblocked`。

拉黑规则：

- 不能拉黑自己。
- 新增拉黑要求 actor 和 target 都是 `ACTIVE`。
- 重复拉黑幂等成功，不重复发事件。
- 重复解除拉黑幂等成功，不发事件。
- 拉黑时自动解除双方关注关系，并发布对应 `user.unfollowed(reason=BLOCKED)`。
- 解除拉黑不恢复历史关注。

## Check-in 子域

Check-in 属于 User 完整边界，但不进入首批实现或字段级 contract 细化。

后续对象：

- `UserCheckIn`：自然键 `(UserID, CheckInDate)`。
- `UserCheckInStats`：总签到天数、连续签到天数、最大连续天数、最近签到日期。

核心规则：

- 同一天只能签到一次。
- 连续签到按业务日期和业务时区计算。
- Redis bitmap 只能作为查询加速，PostgreSQL 是真相源。

## 值对象

| 值对象 | 含义 | 约束 |
| --- | --- | --- |
| `UserID` | User 内部标识 | 不进入前端 URL；服务间写路径和关系表使用 |
| `PublicID` | User 外部公开标识 | 前端 URL 和 HTTP response 使用；User 本地生成和持久化 |
| `AccountID` | Auth 账号引用 | User 本地唯一；不等同 `UserID` |
| `Nickname` | 唯一昵称 | trim；`1..15`；大小写敏感唯一；禁止危险字符 |
| `AvatarFileID` | 头像文件引用 | 可空；文件事实归 File service |
| `Bio` | 简介 | 最大 100；纯文本；允许少量换行；禁止危险字符 |
| `ProfileVersion` | 资料版本 | 公开资料变化时由 repository 原子递增 |
| `UserStatus` | User 资料生命周期状态 | `ACTIVE`、`DEACTIVATED`、`DELETED` |
| `UserPair` | 两个用户之间的关系键 | 用于关注、拉黑、锁和批量检查 |
| `FollowUnfollowReason` | 取关原因 | `USER_REQUEST`、`BLOCKED`、后续可扩展 |

## 领域服务

| 领域服务 | 职责 |
| --- | --- |
| `ProfileInitializationPolicy` | 校验 Auth 初始化 profile 的默认资料规则。 |
| `ProfilePolicy` | 校验 nickname、bio、头像引用输入和公开资料变化。 |
| `UserStatusPolicy` | 判断状态转换、写操作可用性和展示占位语义。 |
| `RelationshipPolicy` | 校验关注、取关、拉黑、解除拉黑的规则。 |
| `CursorPolicy` | 定义关系列表 cursor 锚点语义，不编码具体 Base64。 |

领域服务不依赖数据库、Redis、HTTP client、JWT、File SDK 或 RabbitMQ。

## 工厂

### `UserProfileFactory`

负责从 Auth 注册上下文创建 User profile：

```text
CreateProfileForAccount(accountId, username)
-> 生成 userId / publicId
-> nickname = trim(username)
-> avatarFileId = null
-> bio = ""
-> strangerMessageAllowed = true
-> userStatus = ACTIVE
-> profileVersion = 0
-> UserProfileCreated
```

如果 nickname 已被占用，返回 `USER_NICKNAME_TAKEN`，不自动生成后缀。

### `UserSimpleFactory`

负责把 User 聚合或查询行映射为展示摘要：

- `ACTIVE`：返回真实 nickname 和 avatarFileId。
- `DEACTIVATED`：返回状态和“已注销用户”占位。
- `DELETED`：返回状态和“已删除用户”占位；用户主页可映射为 404。

`avatarUrl` 不属于 domain，由前端 HTTP adapter/application 通过 FileURLResolver 派生。
