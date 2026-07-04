# Auth Application Service

## 命令用例

| Use case | 职责 |
| --- | --- |
| `RegisterAccount` | 校验登录标识唯一性、hash 密码；事务 A 创建或复用未过期 `PENDING_PROFILE` account / credential；同步调用 User `CreateProfileForAccount` 获取非零 `userId`；事务 B 用真正激活时刻写 `auth_accounts.user_id`、切 `ACTIVE`、授予默认 `ROLE_USER` 并写 `auth.account.registered` outbox。 |
| `Login` | 按登录标识加载账号，校验状态和密码；`ACTIVE` 账号必须已有非零 `userId`；先生成并校验 refresh token material（`sessionId/tokenId/plaintext/hash/expiresAt`），再持久化 PostgreSQL refresh session metadata，最后基于已确定的 session 真相签发 access token，并把 refresh plaintext 返回给上层。 |
| `RefreshToken` | 以 PostgreSQL refresh session 和 token hash 为真相源校验 refresh token，执行 token rotation；疑似重放时吊销当前 session 或升级账号级处置。 |
| `Logout` | 吊销当前 refresh token，按需要写 access token 黑名单。 |
| `ChangePassword` | 校验旧凭证，更新 password hash，事务后吊销账号全部 refresh token。 |
| `DisableAccount` / `EnableAccount` | 维护账号状态，禁用后吊销 refresh token 并让 Gateway 缓存自然过期或失效。 |
| `AssignRole` / `RemoveRole` | 维护账号角色事实，事务后清理角色缓存并发布角色变更事件。 |

## 查询用例

| Use case | 职责 |
| --- | --- |
| `GetCurrentPrincipal` | 返回当前账号认证主体、状态和角色。 |
| `GetAccountPrincipal` | 给 Gateway、Admin 或服务间 contract 查询账号主体事实。 |
| `BatchGetAccountStatus` | 批量查询账号状态，用于管理端或安全扫描场景。 |

## 错误映射

| 场景 | Domain/Ports 错误 | HTTP Status | 公开错误码 |
| --- | --- | --- | --- |
| 登录标识已存在 | `ErrLoginIdentifierExists` | 409 | `USER_EMAIL_EXISTS` 或后续 Auth 专属码 |
| 账号不存在或密码错误 | `ErrInvalidCredentials` | 401 | `AUTH_INVALID_CREDENTIALS` |
| 账号禁用 | `ErrAccountDisabled` | 403 | `USER_DISABLED` 或后续 Auth 专属码 |
| Token 无效 | `ErrInvalidToken` | 401 | `AUTH_INVALID_TOKEN` |
| Token 过期 | `ErrTokenExpired` | 401 | `AUTH_TOKEN_EXPIRED` |
| Token 重放 | `ErrTokenReplayed` | 401 | `AUTH_TOKEN_REPLAYED` |
| 需要特定角色 | `ErrRoleRequired` | 403 | `ROLE_REQUIRED` |

错误码第一阶段可沿用既有 `2xxx` 认证授权范围；是否新增 Auth 专属错误码需在 `docs/contracts/error-codes.md` 登记。

## Refresh Token Rotation

- Refresh token 使用高熵 opaque token，服务端只保存 `sessionId`、`currentTokenId` 和 `currentTokenHash`，不保存明文 token。
- `Login` 签发 refresh token 后，先在 PostgreSQL 创建 refresh session；Redis 只保存 refresh session 校验缓存，不作为真相源。
- `RefreshToken` 从 refresh token 中解析 `sessionId/tokenId`，读取 PostgreSQL session 并校验未过期、未撤销、token hash 匹配。
- 如果旧 `tokenId` 或旧 token hash 再次出现，按疑似重放处理：首批吊销当前 session；同账号短时间多次、跨 IP/UA 异常或多个 session 重放时再升级为账号级处置。
- rotation 成功时，在同一数据库事务内把 `currentTokenId/currentTokenHash` 更新为新值；事务提交后再更新 Redis 缓存和 Gateway 可见的安全投影。
- Redis 不可用时，refresh 是否允许降级取决于 Gateway 是否能回源 Auth 校验 access state；如果 Gateway 不能回源，或账号/session 存在未完成安全 operation，则不得签发新 access token。
- Auth Redis key 和 TTL 以 `redis-keys.md` 为准；安全撤销类写入必须能让 Gateway 看到 `jti`、session revoked 或 account version 投影，不能只更新 DB 后静默返回成功。

## 事务边界

**注册事务 A（pending account）**：

```text
auth_accounts
+ auth_password_credentials
```

Auth 先落本地 pending 账号和凭证，不授予默认角色，也不写 `auth.account.registered`：

- pending 账号不进入有效 principal，也不允许登录。
- 默认 `ROLE_USER` 不在 pending 阶段生效。
- Auth 本地事务失败时，不调用 User。
- 当 email 已存在未过期 pending 时，事务 A 复用原 `accountId`，更新 pending nickname / credential，让下一次重试能继续闭合 User 幂等初始化和事务 B。

**同步 User 初始化**：

```text
事务 A 提交
-> User CreateProfileForAccount(accountId, nickname)
```

- User 按 `accountId` 幂等创建 profile 并返回内部 `userId`。
- 只有拿到 `userId` 后，Auth 才能把账号切成可用状态。
- 这里必须同步闭合，避免“注册成功但无法登录 / `users/me` 404”的裂缝。

**注册事务 B（激活账号）**：

```text
auth_accounts(user_id, status=ACTIVE, clear pending marker)
+ auth_account_roles(default ROLE_USER)
+ auth_outbox_events(auth.account.registered)
```

- `auth.account.registered` 只表达已经 `ACTIVE` 且已有 `userId` 的账号事实。
- 如果 User 初始化失败或事务 B 失败，Auth 不向客户端承诺注册成功，也不写 registered 事件。

**密码、状态和角色命令**：

```text
auth_accounts / auth_password_credentials / auth_account_roles
```

事务提交后清理本地缓存，并按命令类型吊销 refresh token 或发布事件。token 吊销失败需要告警或补偿，不应被普通资料更新逻辑吞掉。
