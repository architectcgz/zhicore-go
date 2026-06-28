# Auth Client Contract

本目录放 `zhicore-auth` 作为 provider 拥有的同步 typed client contract。

第一阶段能力：

- `ValidateAccessState`：Gateway 已完成 JWT 签名和时间校验后，回源 Auth 判断 access token claims 对应的账号、session 和 principal 状态是否仍可用。
- `GetAccountPrincipal`：查询当前或指定账号认证主体。
- 查询账号状态和角色。
- 管理端账号禁用、启用和 token 全量失效命令。

这里不放 User profile DTO。昵称、头像、简介和用户摘要归 `libs/contracts/clients/user/`。

## ValidateAccessState

使用场景：

- Redis 不可用且 Gateway L1 miss。
- Redis 不可用时 refresh 降级成功后，新 access token 首次进入 Gateway。
- Gateway 发现 `principalVersion` 可能过期，需要 Auth 回源刷新 principal。
- 高风险路由需要确认最新撤销状态，且不能只依赖本地缓存。

Gateway 不把 raw access token 传给 Auth。Gateway 先本地校验 JWT 签名、`exp`、`kid`、`type=access`，再把解析出的 claims 传给 Auth：

| 字段 | 说明 |
| --- | --- |
| `accountId` | Access token 中的账号 ID。 |
| `userId` | Access token 中的用户资料 ID。 |
| `sessionId` | 登录 session / 设备标识。 |
| `jti` | 当前 access token ID。 |
| `sessionVersion` | token 内登录态版本。 |
| `principalVersion` | token 内认证主体版本。 |
| `issuedAt` | token `iat`。 |
| `expiresAt` | token `exp`。 |
| `requestId` / `traceId` | 链路关联字段。 |

Auth 校验 PostgreSQL 真相源：

- account 存在，`userId` 绑定一致，且账号状态允许访问。
- session 存在，属于该 account，未 revoked，未 expired。
- `sessionVersion` 与 DB 当前值一致。
- `principalVersion` 落后时不直接拒绝，而是返回最新 principal 供 Gateway 更新注入上下文。

响应语义：

| 字段 | 说明 |
| --- | --- |
| `decision` | `ALLOW` 或 `DENY`。 |
| `denyReason` | `SESSION_REVOKED`、`SESSION_EXPIRED`、`SESSION_VERSION_STALE`、`ACCOUNT_DISABLED`、`ACCOUNT_BANNED`、`ACCOUNT_NOT_FOUND`、`USER_MISMATCH` 等。 |
| `principal` | `ALLOW` 时返回最新认证主体：`accountId`、`userId`、`email`、`roles`、`accountStatus`、`sessionVersion`、`principalVersion`。 |
| `principalRefreshed` | token 中的 `principalVersion` 落后但 session 仍有效时为 `true`。 |
| `cacheTtlSeconds` | Gateway 可写入 L1 的建议 TTL。高风险场景可返回 `0`。 |

传输错误、Auth 超时或 Auth 不可用不是 `DENY`，Gateway 必须按 fail closed 处理为认证状态不可确认。
