# Gateway 路由风险策略

本文是 Gateway 在 Redis/Auth 降级时区分 normal 和 high-risk 请求的专题事实源。Auth 模块拥有认证事实；Gateway 只按路由风险决定校验失败或依赖不可用时是否允许短时兜底。

## 核心规则

Gateway 使用规则集匹配高风险路由，而不是要求每个 endpoint 手工标记。规则按 `method`、`path pattern`、`route group`、`required role` 和服务归属组合判断。

未命中高风险规则的 `GET/HEAD/OPTIONS` 可归为 normal。未命中规则的写请求默认按 high-risk 处理，或禁止 fail-open。

Redis 不可用时：

- high-risk 请求不使用 L1 stale cache 放行，必须 Redis 可用或 `ValidateAccessState(claims)` 回源 Auth 成功，否则 fail closed。
- normal 读请求可在 L1 短 TTL 命中且未过期时短时放行；L1 miss/过期时回源 Auth，Auth 不可用则 fail closed。
- 匿名公开请求不需要 access token，不受认证状态 fallback 影响，但仍受基础限流和路由规则约束。

## High-Risk 规则集

首批 high-risk 包括：

| 类别 | 规则示例 | 原因 |
| --- | --- | --- |
| Auth 安全操作 | `/api/v1/auth/logout`、`/api/v1/auth/sessions/**`、`/api/v1/auth/password/**`、`/api/v1/auth/account/deactivate`、`/api/v1/auth/security-operations/**` 的写路径 | 影响 access token 立即失效、会话撤销或账号安全状态。 |
| Admin 路由 | `/api/v1/admin/**` 或要求 `ROLE_ADMIN` 的 route group | 具备跨账号管理能力，不能在状态不可确认时放行。 |
| 账号/角色/封禁命令 | ban/unban、role grant/revoke、force logout、disable/enable account | 直接改变认证、权限或全站访问状态。 |
| 敏感资料写 | 改邮箱、改密码、绑定/解绑安全凭据、修改可用于账号恢复的资料 | 影响账号归属或找回能力。 |
| 支付/资金预留 | payment、refund、withdraw、subscription 管理等 route group | 一旦引入资金能力，默认 high-risk。 |
| 未归类写请求 | 未命中明确 normal 规则的 `POST/PUT/PATCH/DELETE` | 保守默认，避免新增写接口漏标后 fail-open。 |

`GET /api/v1/auth/me`、`GET /api/v1/auth/sessions` 和 `GET /api/v1/auth/security-operations/{operationId}` 是读请求，但属于安全域查询，应至少要求认证状态可确认；Redis 不可用时可回源 Auth，不依赖长期 L1 stale cache。

## Gateway 校验流程

```text
request
-> match route auth requirement and risk level
-> if anonymous: route without user principal
-> verify JWT signature, kid, exp, type=access
-> check jti blacklist / session revoked / version / principal cache
-> if Redis ok and state matched: inject principal
-> if Redis unavailable or cache miss:
   -> high-risk: call Auth ValidateAccessState or fail closed
   -> normal read: allow only with unexpired L1 hit, otherwise call Auth
-> Auth unavailable when state is required: fail closed
```

Gateway 传给 Auth 的 `ValidateAccessState` 只能是已验签 claims，不传 raw access token。Auth 以 PostgreSQL account/session 真相源判断 `ALLOW/DENY`。

## L1 Cache 约束

L1 是 Gateway 实例内短 TTL 缓存，只用于减少 Redis 压力和短时兜底：

- 默认 TTL 3-10 秒，配置化。
- 只缓存已知校验结果、version、principal snapshot 或 negative revoke 查询结果。
- 不跨实例共享，不能替代 Redis 作为撤销/版本事实投影。
- Redis 不可用期间，L1 不能感知其他实例或 Auth 刚发生的新撤销。

## 降级响应和观测

Gateway fail closed 时返回认证/权限不可确认的 401/403/503 映射，用户文案不要暴露 Redis、L1 或内部拓扑，只表达“登录状态暂时无法确认/系统繁忙/稍后重试”。

至少记录：

- `routeRisk=high|normal|anonymous`
- `authDecision=allow|deny|fail_closed|fallback`
- `fallbackReason=redis_unavailable|l1_miss|principal_stale|validate_timeout`
- `ValidateAccessState` latency / timeout / circuit open count
- Redis unavailable count
- L1 hit/miss/stale count

metrics label 不得包含 token、email、IP、cookie、Authorization header 或用户输入文本。
