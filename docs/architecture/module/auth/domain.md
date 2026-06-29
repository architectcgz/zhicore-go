# Auth Domain

## AccountID 与 UserID 的关系

当前阶段 Auth 的 `AccountID` 和 User 的 `UserID` 数值相同（User 用 Auth `accountId` 初始化 profile），但它们是**不同语义的标识符**，归属不同服务：

| 标识符 | 归属 | 语义 |
|--------|------|------|
| `AccountID` | Auth | 认证身份，用于登录、token 签发、角色、账号状态 |
| `UserID` | User | 用户资料，用于昵称、头像、关注、签到 |

使用规则：

- Gateway 注入的 `X-User-Id` 传递的是 Auth 的 `AccountID`（当前与 UserID 相同）。
- 下游服务调用 User contract 时，传入的 ID 在当前阶段即为 `AccountID`，User 服务接受它作为 `UserID`。
- 代码和文档中不能混用 `accountId` 和 `userId` 字段名：Auth 侧用 `accountId`，User 侧用 `userId`，哪怕当前值相同。
- 如果将来 User 需要支持多账号绑定（一个 User 对应多个 Auth 账号），两者必然分离；届时再建立显式映射关系，现有引用不需要提前兼容。

## 聚合

### `Account`

`Account` 是账号认证聚合根，负责维护登录身份、账号状态和角色关系：

- **标识**：`AccountID`，第一阶段可与 User profile 使用同一个公开用户 ID，但 owner 归 Auth。
- **登录标识**：`Username`、`Email`，后续可扩展 phone 或第三方账号绑定。
- **状态**：`AccountStatus`（`Active`、`Disabled`、`Locked`、`PendingVerification`）。
- **角色**：`RoleName` 集合，例如 `ROLE_USER`、`ROLE_ADMIN`。
- **行为**：`Disable`、`Enable`、`Lock`、`Unlock`、`AssignRole`、`RemoveRole`。
- **领域事件**：`AccountRegistered`、`AccountDisabled`、`AccountEnabled`、`AccountRoleChanged`。

`Account` 不保存昵称、头像、简介、关注关系或签到统计。

### `Credential`

`Credential` 表示账号的登录凭证：

- **标识**：`AccountID`。
- **密码**：`PasswordHash`，不保存明文密码。
- **行为**：`ChangePasswordHash`。
- **领域事件**：`PasswordChanged`。

密码强度和 hash 参数由 application 通过 policy 和 port 编排；domain 不依赖具体 hash 库。

### `Role`

第一阶段 `Role` 作为受控参考数据处理：

- `RoleName` 全局唯一。
- 注册账号默认分配 `ROLE_USER`。
- 管理员角色变更属于 Auth command。

如果未来引入复杂权限模板、资源权限或审计策略，再把权限模型单独设计，不塞进 Gateway 或 User。

## 值对象

| 值对象 | 含义 |
| --- | --- |
| `AccountID` | 账号内部标识 |
| `Username` | 唯一用户名 |
| `Email` | 登录邮箱 |
| `PasswordHash` | 已 hash 的密码 |
| `AccountStatus` | 账号生命周期状态 |
| `RoleName` | 角色名称 |
| `TokenID` | refresh token 或 access token 的唯一标识 |

## 核心不变量

- 登录标识不能为空，唯一性由 repository 约束和 application 校验共同保证。
- 密码只以 hash 形式持久化，不进入日志、事件或错误响应。
- 禁用账号不能登录、refresh、修改密码或执行需要登录态的 Auth command。
- 角色事实只能由 Auth command 修改，Gateway、User、Admin 不直接写角色表。
- token 是安全凭证，不作为业务资源 ID 使用。

## 领域服务

| 领域服务 | 职责 |
| --- | --- |
| `RegistrationPolicy` | 校验注册输入格式和默认账号规则。 |
| `CredentialPolicy` | 校验密码复杂度和凭证变更条件。 |
| `AccountStatusPolicy` | 判断账号状态是否允许登录、refresh 或凭证变更。 |
| `RolePolicy` | 校验默认角色、管理员角色和角色变更规则。 |

JWT 签发、refresh session 持久化与 rotation、Redis 黑名单和密码 hash / verify 都不是领域服务职责，由 application 通过 ports 编排。
