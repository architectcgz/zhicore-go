# Auth Ports

Ports 放在 `services/zhicore-auth/internal/auth/ports`，按能力和用例族定义 consumer-side interface。

## 核心端口

| Port | 职责 |
| --- | --- |
| `AccountRepository` | Account 聚合加载、保存、状态更新和登录标识唯一性检查。 |
| `CredentialRepository` | 密码 hash 读取和更新。 |
| `RoleRepository` | 默认角色查询、账号角色关系写入和角色查询。 |
| `AccountQueryRepository` | 当前主体、账号主体和管理端账号查询。 |

## 安全机制端口

| Port | 职责 |
| --- | --- |
| `PasswordHasher` | 密码 hash 和 verify。 |
| `TokenIssuer` | JWT access / refresh token 签发、解析、claims 校验和 token ID 提取。 |
| `RefreshTokenStore` | Refresh token 白名单、吊销、rotation 和账号级全量吊销。 |
| `AccessTokenBlacklist` | Access token 黑名单或 token version 失效机制。 |
| `AuthCacheStore` | 账号主体、角色和 token 校验缓存。 |

## 事务和集成端口

| Port | 职责 |
| --- | --- |
| `TransactionRunner` | Auth 本地显式事务边界。 |
| `OutboxPublisher` | 事务内追加 Auth 集成事件。 |
| `UserProfileClient` | 注册后初始化 User profile，或查询必要的用户资料存在性。 |
| `Clock` | 时间源和 token 过期计算。 |

端口不能暴露 `*gorm.DB`、`*redis.Client`、Gin context、HTTP DTO、ORM sentinel 或外部 SDK 类型。底层 not-found、重复键、Redis nil 等错误由 infrastructure adapter 翻译为 module-local 语义。
