# Auth 数据和事件

## 数据归属

Auth 拥有账号认证事实：

| 表 / 存储 | 归属 | 说明 |
| --- | --- | --- |
| `accounts` | Auth | 账号 ID、登录标识、账号状态、创建时间和更新时间。 |
| `account_credentials` | Auth | password hash、凭证版本和更新时间。 |
| `roles` | Auth | 角色参考数据。 |
| `account_roles` | Auth | 账号角色关系。 |
| `outbox_events` | Auth | Auth 集成事件 outbox。 |
| Redis refresh token 白名单 | Auth | `(accountId, tokenId)` TTL 记录。 |
| Redis token blacklist / version | Auth / Gateway 协作 | Auth 产生失效语义，Gateway 用于入口校验。 |

User 不保存 Auth 表的副本。User 可以保存 `accountId` 作为资料归属引用，并维护自己的 profile、关系和签到表。

## 集成事件

| 事件 | 触发用例 | 主要 payload | 当前 / 目标 consumer | outbox 要求 |
| --- | --- | --- | --- | --- |
| `auth.account.registered` | `RegisterAccount` | `accountId`、`username`、`email`、`occurredAt` | User 初始化 profile；Notification 或运营读模型如需要欢迎事件再消费 | 关键事件，使用 producer outbox |
| `auth.account.disabled` | `DisableAccount` | `accountId`、`occurredAt`、`reason` | Gateway 清理认证缓存；Admin 记录审计；User 可限制资料更新 | 关键事件，使用 producer outbox |
| `auth.account.enabled` | `EnableAccount` | `accountId`、`occurredAt` | Gateway 清理认证缓存；Admin 记录审计 | 关键事件，使用 producer outbox |
| `auth.role.changed` | `AssignRole` / `RemoveRole` | `accountId`、`roles`、`occurredAt` | Gateway 清理角色缓存；Admin 记录审计 | 关键事件，使用 producer outbox |
| `auth.password.changed` | `ChangePassword` | `accountId`、`occurredAt` | 安全审计；默认不广播给业务服务 | 可选，按审计 owner 决定 |

事件 payload 不包含 password hash、JWT、refresh token、Authorization header 或完整请求 body。

## 与 User 的一致性

注册需要同时形成账号和用户资料。第一阶段推荐同步初始化 User profile，并登记补偿：

- Auth 本地事务提交失败：不调用 User。
- Auth 本地事务提交成功但 User 初始化失败：返回注册失败或待补偿状态，写补偿任务或 outbox，不向客户端承诺完整可用账号。
- User profile 已存在：按幂等成功或冲突处理，具体语义必须在 User profile 初始化 contract 中登记。

如果改成纯事件驱动，必须定义 `PendingProfile` 或等价状态，避免登录后 `me` 成功但用户资料接口 404。
