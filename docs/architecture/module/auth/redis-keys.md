# Auth Redis Key 设计

本文是 Auth 模块 Redis key、TTL 和故障语义的专题事实源。`decision-log.md` 记录决策结论；具体 key 命名和读写规则以本文为准。

## 定位

PostgreSQL 是账号、refresh session、安全 operation 和审计的真相源。Redis 是 Gateway 可见的撤销、版本、principal cache、refresh session cache 和限流投影。

Redis 不可用不能让 DB 中的账号/session 事实失效，但会影响 Gateway 及时感知已签发 access token 的撤销和权限收敛。因此影响 access token 立即失效的操作必须按安全 operation / degraded 规则处理。

## Key 清单

| Key | Owner | Value | TTL | 用途 |
| --- | --- | --- | --- | --- |
| `auth:jti:blacklist:{jti}` | Auth 写，Gateway 读 | `1` 或撤销原因摘要 | `access token` 剩余有效期 + `clockSkew` | 精确吊销单张 access token。 |
| `auth:session:revoked:{sessionId}` | Auth 写，Gateway 读 | `1` 或撤销原因摘要 | refresh session 剩余有效期或配置保留窗口 | 吊销某个登录设备/session 下的 access token。 |
| `auth:account:session_version:{accountId}` | Auth 写，Gateway 读 | number | 不设 TTL 或长 TTL，配置化 | 账号级强制失效版本；旧 access token 直接 401。 |
| `auth:account:principal_version:{accountId}` | Auth 写，Gateway 读 | number | 不设 TTL 或长 TTL，配置化 | 认证主体快照版本；落后时 Gateway 回源 Auth 刷新 principal。 |
| `auth:principal:{accountId}` | Auth 写，Gateway 读 | principal JSON | 默认 1-5 分钟，配置化 | Gateway 注入身份上下文的短 TTL 快照。 |
| `auth:refresh:session:{sessionId}` | Auth 写读 | refresh session 校验摘要 | refresh cookie 剩余有效期 | refresh 校验加速缓存；DB 仍是真相源。 |
| `auth:login:fail:email:{emailHash}` | Auth 写读 | 失败计数 | 默认 15 分钟，配置化 | email 维度登录失败限流和临时锁定辅助。 |
| `auth:login:fail:ip:{ipHash}` | Auth 写读 | 失败计数 | 默认 10 分钟，配置化 | IP 维度登录失败限流。 |

`sessionVersion` 和 `principalVersion` 首期拆成独立 version key，不再只并入 `auth:principal:{accountId}`。principal cache 仍可冗余携带两个版本字段，但不能作为唯一版本事实投影。

## 敏感信息边界

Redis key 不得包含以下明文：

- email、手机号、IP、User-Agent。
- access token、refresh token、cookie、Authorization header。
- password、password hash、reset token、验证码。

需要按输入维度限流时，使用规范化值的 hash，例如 `emailHash`、`ipHash`。metrics label 同样不得使用原始敏感值。

## 写入时机

| 场景 | 必须写入 |
| --- | --- |
| `logout current` | 当前 access token 的 `auth:jti:blacklist:{jti}`；当前 session 的 revoked 投影。 |
| `DELETE /auth/sessions/{sessionId}` | `auth:session:revoked:{sessionId}`；如已知目标 session 当前 access `jti`，同时写 jti blacklist。 |
| `logout all` / 改密码 / 找回密码成功 / 主动注销 / 封禁 | 递增 DB `session_version`，吊销 refresh sessions，更新 `auth:account:session_version:{accountId}`。 |
| 角色变化 / 权限标签变化 / email 展示变化 / 解封 | 递增 DB `principal_version`，更新 `auth:account:principal_version:{accountId}`，删除或覆盖 `auth:principal:{accountId}`。 |
| login / register 成功 | 创建 DB refresh session 后，尽力写 `auth:refresh:session:{sessionId}` 和 `auth:principal:{accountId}`；短时 Redis 不可用可按登录降级策略成功。 |
| refresh 成功 | DB rotation 提交后更新 refresh session cache；Redis 不可用时按 refresh 降级矩阵判断是否允许签新 token。 |

安全撤销类操作的 Redis 成功标准至少是写命令同步返回 OK 且 TTL / value 正确；高风险操作可读回校验，必要时配置 Redis `WAIT` 等待副本确认。

## Cache Miss 和故障语义

- 黑名单/撤销 key miss 不等于一定未撤销；当 Redis 不可用或 Gateway L1 过期时，Gateway 必须按路由风险策略决定回源 Auth、fail closed 或短时放行。
- version key miss 时，Gateway 可用 `ValidateAccessState(claims)` 回源 Auth 获取当前状态；Auth 不可用时需要认证的请求 fail closed。
- principal cache miss 时，Gateway 回源 Auth 拉取 principal；不能把 access token claims 中的 `roles/accountStatus` 当授权事实源。
- refresh session cache miss 时，Auth 回 PostgreSQL 校验；Redis 不是 refresh session 真相源。

## 配置和观测

所有 TTL、key prefix、clock skew、读回校验、Redis 写超时、Gateway L1 TTL 和降级窗口必须配置化，不能写死在 handler 或 middleware 中。

至少记录以下指标：

- Redis write/read error count。
- blacklist / revoked session / version projection write latency。
- projection write degraded count。
- Gateway fallback to `ValidateAccessState` count。
- Redis key TTL invalid / readback failed count。
- L1 hit/miss/stale decision count。
