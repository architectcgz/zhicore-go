# User Client Contract

本目录放 `zhicore-user` 作为 provider 拥有的同步 typed client contract。

当前状态为设计草案，尚未生成 Go client 代码。实现时应把本文件拆成稳定 DTO、client interface、HTTP adapter 和 contract tests。

## 使用场景

- Auth 注册后同步初始化 User profile。
- Auth 主动注销时调用 User 停用 profile。
- Comment 写路径校验作者 availability 和拉黑关系。
- Comment 查询路径批量获取作者摘要。
- Message 查询拉黑、关注和陌生人私信设置。
- Notification fanout 查询粉丝分片。
- Content 刷新或展示作者快照时获取 User 摘要。

## Client interface 草案

```go
type Client interface {
    CreateProfileForAccount(ctx context.Context, input CreateProfileForAccountInput) (UserProfile, error)
    DeactivateUserProfile(ctx context.Context, input DeactivateUserProfileInput) error
    BatchGetUserSimple(ctx context.Context, userIDs []int64) (BatchUserSimpleResult, error)
    BatchGetUserAvailability(ctx context.Context, userIDs []int64) (map[int64]UserAvailability, error)
    BatchCheckBlocked(ctx context.Context, pairs []UserPair) (map[UserPair]bool, error)
    CheckFollowing(ctx context.Context, followerID, followingID int64) (bool, error)
    GetStrangerMessageSetting(ctx context.Context, userID int64) (bool, error)
}
```

## DTO 规则

`BatchGetUserSimple` 入参使用内部 `userId`。返回 DTO 至少包含：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `userId` | int64 | User 内部标识，用于调用方映射原请求。 |
| `publicId` | string | 前端公开用户 ID。 |
| `nickname` | string | 唯一昵称。非 ACTIVE 用户可返回占位昵称。 |
| `avatarFileId` | string | Upload / File Service 文件引用，可空。 |
| `profileVersion` | int64 | 公开资料版本。 |
| `status` | string | `ACTIVE`、`DEACTIVATED`、`DELETED`。 |

默认 typed client 不返回 `avatarUrl`。需要展示 URL 的前端聚合接口由 owning service 调 Upload/File Service 批量解析。

批量查询中缺失用户不让整体失败，结果省略缺失项并返回 `missingUserIds`。

## 错误语义

| 错误 | 语义 | Consumer 处理 |
| --- | --- | --- |
| `USER_NOT_FOUND` | 单用户查询目标不存在 | 查询路径可展示占位；写路径按业务失败。 |
| `USER_NOT_ACTIVE` | User profile 非 `ACTIVE` | 写路径禁止新增互动。 |
| `USER_NICKNAME_TAKEN` | 初始化或改名时昵称已占用 | Auth 注册或资料更新返回明确错误。 |
| `USER_INTERACTION_BLOCKED` | 拉黑关系阻止互动 | Comment/Message 返回权限类错误。 |
| `SERVICE_DEGRADED` | User 或依赖不可用 | 写路径失败；查询路径按调用方降级策略处理。 |

## 约束

- Consumer 不得导入 `services/zhicore-user/internal`。
- Consumer 不得直连 User 数据库解析 `publicId` 或读取关系表。
- 写路径 guard 不得复用 `BatchGetUserSimple`；应调用 availability / block / follow contract。
- User 事件默认使用内部 `userId`，不把 `publicId` 作为关系和权限事实。
