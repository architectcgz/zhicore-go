# Auth 数据和事件

## 数据归属

Auth 拥有账号认证事实：

| 表 / 存储 | 归属 | 说明 |
| --- | --- | --- |
| `auth_accounts` | Auth | 账号 ID、登录标识、账号状态、User 映射、版本号和登录元数据。 |
| `auth_password_credentials` | Auth | password hash、凭证版本和更新时间。 |
| `auth_account_roles` | Auth | 账号角色授予、撤销和审计来源。 |
| `auth_refresh_sessions` | Auth | Refresh session、当前 refresh token hash、原始持久化策略、过期和撤销状态；PostgreSQL 是真相源。 |
| `auth_used_refresh_tokens` | Auth | 已 rotation 使用过的 refresh token 记录，用于 replay 检测。 |
| `auth_email_verifications` | Auth | 邮箱验证码发送、校验、尝试次数和发送状态。 |
| `auth_verification_tokens` | Auth | 注册、找回密码等短期一次性不透明 token hash。 |
| `auth_security_operations` | Auth | Redis 投影写失败或安全同步处理中时的可查询 operation。 |
| `auth_audit_logs` | Auth | Auth 本地安全审计日志。 |
| `auth_outbox_events` | Auth | Auth 集成事件 transactional outbox。 |
| Redis refresh session cache | Auth | `sessionId` 维度的 refresh 校验缓存；Redis 不作为 refresh session 真相源。 |
| Redis token blacklist / version | Auth / Gateway 协作 | Auth 产生失效语义，Gateway 用于入口校验。 |

User 不保存 Auth 表的副本。User 可以保存 `accountId` 作为资料归属引用，并维护自己的 profile、关系和签到表。

PostgreSQL 表、字段、约束、索引和保留策略见 `data-model.md`。正式可执行 schema 后续落到 `services/zhicore-auth/migrations/`，不要新增服务级 `schema/` 目录作为第二事实源。

Redis key、TTL、敏感信息边界和 Gateway cache miss 语义见 `redis-keys.md`。`sessionVersion`、`principalVersion` 首期使用独立 Redis version key，同时可冗余进入 `auth:principal:{accountId}` 短 TTL 快照。

## 集成事件

| 事件 | 触发用例 | 主要 payload | 当前 / 目标 consumer | outbox 要求 |
| --- | --- | --- | --- | --- |
| `auth.account.registered` | `RegisterAccount` 激活事务 B | `accountId`、`userId`、`email`、`occurredAt` | Notification、运营读模型或权限投影等已激活账号 consumer；不再用于驱动 User 初始化 profile | 关键事件，使用 producer outbox |
| `auth.account.disabled` | `DisableAccount` | `accountId`、`occurredAt`、`reason` | Gateway 清理认证缓存；Admin 记录审计；User 可限制资料更新 | 关键事件，使用 producer outbox |
| `auth.account.enabled` | `EnableAccount` | `accountId`、`occurredAt` | Gateway 清理认证缓存；Admin 记录审计 | 关键事件，使用 producer outbox |
| `auth.role.changed` | `AssignRole` / `RemoveRole` | `accountId`、`roles`、`occurredAt` | Gateway 清理角色缓存；Admin 记录审计 | 关键事件，使用 producer outbox |
| `auth.password.changed` | `ChangePassword` | `accountId`、`occurredAt` | 安全审计；默认不广播给业务服务 | 可选，按审计 owner 决定 |

事件 payload 不包含 password hash、JWT、refresh token、Authorization header、nickname 或完整请求 body。

## 与 User 的一致性

注册需要同时形成账号和用户资料。当前采用同步初始化 User profile：

- 事务 A：Auth 创建 `PENDING_PROFILE` account / credential，不授予默认 `ROLE_USER`，不写 `auth.account.registered`。
- 同步调用：Auth 调 User `CreateProfileForAccount(accountId, nickname)`，User 按 `accountId` 幂等返回 `userId`。
- 事务 B：Auth 写 `auth_accounts.user_id`、切 `ACTIVE`、授予默认 `ROLE_USER`、写 `auth.account.registered` outbox。
- Auth 本地事务 A 失败：不调用 User。
- User 初始化失败：返回错误，不写 `auth.account.registered`，也不把 pending 账号伪装成注册成功。
- 事务 B 失败：返回错误，前端可重试；User 端因 `accountId` 幂等会返回同一个 `userId`，Auth 再补齐本地激活事务。

`auth.account.registered` 只在账号已经 `ACTIVE` 且存在 `userId` 时发布，避免下游消费到“注册成功但 profile 尚未闭合”的半成品事实。
