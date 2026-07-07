# Auth Ports

Ports 放在 `services/zhicore-auth/internal/auth/ports`，按能力和用例族定义 consumer-side interface。

## 核心端口

| Port | 职责 |
| --- | --- |
| `AccountRepository` | Account 聚合加载、保存、状态更新和登录标识唯一性检查。 |
| `CredentialRepository` | 密码 hash 读取和更新。 |
| `RoleRepository` | 默认角色查询、账号角色关系写入和角色查询。 |
| `AccountQueryRepository` | 当前主体、账号主体和管理端账号查询。 |
| `AccessStateQuery` | Gateway 回源校验 access token claims 对应的账号、session 和 principal 状态。 |

## 安全机制端口

| Port | 职责 |
| --- | --- |
| `PasswordHasher` | 密码 hash 和 verify。 |
| `RefreshTokenMaterialIssuer` | 按 application 传入的 refresh TTL / session 持久化策略生成 refresh token material：`sessionId/tokenId/plaintext/hash/expiresAt`；application 必须先校验这些字段非空，且 `expiresAt` 等于 application 按策略计算的 `now + TTL`，再持久化 refresh session metadata。 |
| `RefreshSessionStore` | Refresh session 真相源；持久化已确定的 `sessionId/currentTokenId/currentTokenHash/persistencePolicy/expiresAt`，refresh rotation 时按已保存的 `persistencePolicy` 计算新过期时间，不负责回传 refresh token 明文。 |
| `TokenIssuer` | 只负责 access token/JWT claims 的签发、解析和校验；不能先发明 refresh session 真相身份再让 repository 追认。 |
| `AccessTokenBlacklist` | Access token 黑名单或 token version 失效机制；task 1 先保留端口定义，不强制 application 构造期接线；后续 `logout`、单 session revoke、改密码和账号禁用接入时，它是对应能力的 required owner。 |
| `AuthCacheStore` | 账号主体、角色和 token 校验缓存；task 1 先保留端口定义，不强制 application 构造期接线；后续 `me`、session revoke 和 Gateway principal/session projection 接入时，它是对应能力的 required owner。 |
| `RateLimiter` | 认证相关限流；返回 typed outcome（如 allow / reject / degraded / unavailable），避免后续 HTTP 无法区分拒绝与下游降级；register/login 作为首批安全热路径，构造期必须提供。 |

## 事务和集成端口

| Port | 职责 |
| --- | --- |
| `TransactionRunner` | Auth 本地显式事务边界。 |
| `OutboxPublisher` | 事务内追加 Auth 集成事件。 |
| `UserProfileClient` | 注册链路同步调用 User `CreateProfileForAccount` 闭合 profile，并返回 `userId` 给 Auth 激活账号；返回零值 `userId` 视为 contract violation。 |
| `Clock` | 时间源和 token 过期计算。 |

## 注册 retry 端口语义

- `AccountRepository.CreateOrLoadPendingForRegister` 负责“新建 pending”或“复用未过期 pending 并更新 nickname”。
- `CredentialRepository.SaveForPendingAccount` 负责当前 pending account 的密码凭证写入；重试时允许覆盖同账号的 pending credential。
- 这样 `RegisterAccount` 在事务 B 或 User 调用失败后，下一次重试可以沿用同一个 `accountId` 继续闭合 User 幂等初始化和本地激活事务。

端口不能暴露 `*gorm.DB`、`*redis.Client`、Gin context、HTTP DTO、ORM sentinel 或外部 SDK 类型。底层 not-found、重复键、Redis nil 等错误由 infrastructure adapter 翻译为 module-local 语义。
