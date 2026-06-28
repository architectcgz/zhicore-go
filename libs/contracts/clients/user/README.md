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

## Caller Identity

User typed client 调用必须携带服务间 caller 身份，供 User 做内部调用限流、审计和观测。caller 身份不是用户身份，不能替代 `Actor` / `AuthContext` 或资源权限校验。

| Header | 必填 | 来源 | 说明 |
| --- | --- | --- | --- |
| `X-Caller-Service` | 是 | consumer 服务静态配置 | 稳定服务名，例如 `zhicore-auth`、`zhicore-content`、`zhicore-comment`、`zhicore-message`、`zhicore-notification`、`zhicore-ranking`、`zhicore-admin`。 |
| `X-Caller-Operation` | 是 | typed client 调用点常量 | 稳定低基数字符串，例如 `auth.create_profile`、`comment.check_user_availability`、`message.check_blocked`、`notification.list_follower_shard`。不得包含用户输入、`userId`、`publicId`、cursor 或错误文本。 |
| `X-Request-Id` | 否 | 上游请求或任务 metadata | 用于单次请求关联。 |
| `X-Trace-Id` | 否 | 上游请求或任务 metadata | 用于跨服务链路关联。 |

规则：

- typed client adapter 负责写入 `X-Caller-Service` / `X-Caller-Operation`，业务代码不手写 header。
- User 对内部高成本接口按 `callerService + operation + target` 限流；未知 caller 或缺少 caller header 的服务间-only endpoint 默认按未认证内部调用处理，返回 `SERVICE_DEGRADED` 或权限类错误，而不是落到匿名公开配额。
- 如果某个 consumer 需要代表当前用户调用 User，必须在 User HTTP schema 中显式登记允许的用户身份 header；普通 typed client 查询默认只使用服务身份。

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
    ListFollowerShard(ctx context.Context, input ListFollowerShardInput) (ListFollowerShardResult, error)
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

缺失用户和依赖不可用必须区分：

- `BatchGetUserSimple` 的缺失用户进入 `missingUserIds`，不代表 User 降级。
- `BatchCheckBlocked` / `CheckFollowing` 中任一用户缺失可返回 `false`，但 User DB、限流或 circuit open 必须返回 `SERVICE_DEGRADED`。
- `GetStrangerMessageSetting` 中用户不存在或设置缺失返回 `false`，依赖不可用返回 `SERVICE_DEGRADED`。

## Resilience

- HTTP client policy 必须按 `docs/architecture/runtime-operations.md` 和 `docs/architecture/module/user/runtime-resilience.md` 配置 timeout、retry、circuit breaker、max-in-flight 和观测字段。
- 查询类调用可以在配置内重试；`CreateProfileForAccount`、`DeactivateUserProfile` 等写路径只允许依靠 `accountId`、状态条件、唯一约束或明确幂等键收敛，不做盲重试。
- typed client adapter 不得伪造 `UserSimple`、availability、blocked、following 或 stranger message setting 作为降级结果；降级选择由 consumer application 决定。
- 写路径 guard 调用 User degraded 时必须 fail closed。Comment、Message 等 consumer 不得把 `SERVICE_DEGRADED` 当成“用户可用”“未拉黑”或“允许陌生人消息”。
- Notification fanout 的 `ListFollowerShard` degraded 时应 retry / DLQ；不能把空 shard 当成 fanout 成功。

## 约束

- Consumer 不得导入 `services/zhicore-user/internal`。
- Consumer 不得直连 User 数据库解析 `publicId` 或读取关系表。
- 写路径 guard 不得复用 `BatchGetUserSimple`；应调用 availability / block / follow contract。
- User 事件默认使用内部 `userId`，不把 `publicId` 作为关系和权限事实。
