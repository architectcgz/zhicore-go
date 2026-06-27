# Auth Application Service

## 命令用例

| Use case | 职责 |
| --- | --- |
| `RegisterAccount` | 校验登录标识唯一性、hash 密码、创建账号、分配默认角色、写 outbox，并初始化 User profile。 |
| `Login` | 按登录标识加载账号，校验状态和密码，签发 access / refresh token，写入 refresh token 白名单。 |
| `RefreshToken` | 校验 refresh token、检查白名单、执行 token rotation；疑似重放时吊销账号全部 refresh token。 |
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

- Refresh token 必须包含可识别的 `tokenId`、`accountId`、token type 和过期时间。
- `Login` 签发 refresh token 后，将 `(accountId, tokenId)` 写入 Redis 白名单，并设置 TTL。
- `RefreshToken` 先校验签名和 token type，再从 token 中解析 `accountId` 与 `tokenId`。
- 如果签名有效但 Redis 白名单不存在 `(accountId, tokenId)`，按疑似重放处理：吊销该账号全部 refresh token，并返回登录态失效错误。
- 如果签名无效、过期、token 类型错误或无法解析账号 ID，按普通无效 token 返回，不触发全量吊销。
- rotation 成功时，先吊销旧 `tokenId`，再写入新 `tokenId`。写入新白名单失败时不得返回新 token。

## 事务边界

**注册事务**：

```text
accounts
+ account_credentials
+ account_roles
+ outbox_events(auth.account.registered)
```

User profile 初始化不和 Auth 本地事务共享数据库事务。第一阶段如采用同步调用，Auth 本地提交成功但 User 初始化失败时，必须返回明确失败并登记补偿策略，不能静默返回完整注册成功。

**密码、状态和角色命令**：

```text
accounts / account_credentials / account_roles
```

事务提交后清理本地缓存，并按命令类型吊销 refresh token 或发布事件。token 吊销失败需要告警或补偿，不应被普通资料更新逻辑吞掉。
