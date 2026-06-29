# Auth Application Service

## 命令用例

| Use case | 职责 |
| --- | --- |
| `RegisterAccount` | 校验登录标识唯一性、hash 密码、创建账号、分配默认角色，并在同一事务内写 `auth.account.registered` outbox 事件。User profile 初始化由 User 服务异步消费该事件完成，不在 Auth 事务内同步调用。 |
| `Login` | 按登录标识加载账号，校验状态和密码，创建 PostgreSQL refresh session，签发 access / refresh token，并尽力写 Redis 缓存和限流状态。 |
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

**注册事务**：

```text
auth_accounts
+ auth_password_credentials
+ auth_account_roles
+ auth_outbox_events(auth.account.registered)
```

事务提交即代表注册成功，Auth 立即返回成功响应。User profile 初始化由 User 服务异步消费 `auth.account.registered` 事件完成：

- User consumer 消费事件，创建 profile，写入 `users` 表。
- consumer 幂等：若 profile 已存在（重复消费）则 no-op。
- consumer 失败：按 User 服务 consumer retry / DLQ 策略重试；最终 profile 必然被创建。
- profile 短暂不存在（事件尚未消费）属于已知最终一致窗口，通常 < 1s；首次登录查询用户资料前若需要确认 profile 已初始化，可以在查询侧容忍 profile not-found 并提示"资料加载中"。

**不再采用同步调用**：Auth 本地提交后立即调用 User 初始化是不正确的设计，因为 Auth 事务已提交后再同步调用失败，Auth 无法回滚，产生孤儿账号或需要复杂补偿。

**密码、状态和角色命令**：

```text
auth_accounts / auth_password_credentials / auth_account_roles
```

事务提交后清理本地缓存，并按命令类型吊销 refresh token 或发布事件。token 吊销失败需要告警或补偿，不应被普通资料更新逻辑吞掉。
