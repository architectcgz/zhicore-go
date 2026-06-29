# Gateway Redis 降级策略

本文定义 `zhicore-gateway` 在 Redis 不可用时的行为决策。认证架构和身份传播规则见 `docs/architecture/security.md`。

## 背景

Gateway 依赖 Redis 做两件事：

1. **Access token 黑名单查询**：检查已吊销的 `jti` 或账号 token version。
2. **Token 校验结果缓存**：缓存已校验通过的 token 解析结果，减少重复验签。

Redis 不可用时，这两个能力同时失效，产生安全和可用性的二选一问题。

## 决策

**不使用全局 fail-open（不能因为 Redis 不可用就让所有请求跳过黑名单检查）。**

采用**按路由风险分级的有限兜底 + fail-closed 升级**策略。具体 high-risk / normal 分类见 `route-risk-policy.md`：

| 阶段 | 条件 | 行为 |
|------|------|------|
| 短时降级（≤ 60s） | Redis 连续失败，仍在容忍窗口内 | normal 读请求仅在 L1 短 TTL 命中且未过期时可放行；L1 miss/过期则回源 Auth，回源失败即 fail-closed。high-risk 请求和未归类写请求必须 Redis 可用或回源 Auth 成功，否则 fail-closed。记录 `gw_redis_degraded` metric；通知告警 |
| 长时降级（> 60s） | Redis 持续不可用 | fail-closed：所有需要认证状态确认的请求返回 `503 SERVICE_UNAVAILABLE`；仅保留匿名公开请求和 `/health/*` 可访问 |
| 恢复 | Redis 重新可用 | 自动恢复，清除降级状态；记录恢复事件 |

**60s 兜底窗口的安全含义：**

- Access token TTL 通常为 15-30 分钟。
- 在 60s 内，只有已经被本 Gateway 实例短 TTL L1 缓存确认过的 normal 读请求可以继续使用；它无法感知 Redis 故障期间其他实例或 Auth 刚写入的新撤销。
- high-risk 请求、Admin 路由、未归类写请求和需要立即收敛权限的路径不接受这个风险。
- 超过 60s 仍不可用，继续放行会使"立即禁用账号"等安全操作完全失效，必须 fail-closed。

## 例外路径（始终 fail-closed，不进入 blind window）

以下操作无论 Redis 是否可用，都必须可靠执行或失败：

- `POST /auth/refresh`：需要查 Redis session 投影或回源 Auth 确认 refresh 是否有效；状态不可确认时必须返回 `503`，不能 blind 放行。
- `POST /auth/logout` 等账号安全操作：这些操作本身会写 Redis 投影；Gateway 在降级时转发这些请求到 Auth，但 Auth 会因为 Redis 不可用而返回 `202 PROCESSING`。

## 配置项

| 配置 | 默认值 | 说明 |
|------|--------|------|
| `GATEWAY_REDIS_FALLBACK_WINDOW_SECONDS` | `60` | Redis 不可用后允许 normal 读请求短时兜底的时间窗口 |
| `GATEWAY_REDIS_FAIL_CLOSED_AFTER_SECONDS` | `60` | 超过此时间强制 fail-closed |

## 观测

| metric | 说明 |
|--------|------|
| `gw_redis_degraded_requests_total` | 兜底窗口内处理的请求数 |
| `gw_redis_fail_closed_requests_total` | fail-closed 后拒绝的请求数 |
| `gw_redis_degraded_duration_seconds` | 本次降级持续时长 |

降级开始和结束都必须输出 `ERROR` 级别结构化日志，字段包含 `reason`、`degradedAt`、`failClosedAt`（如适用）。

## 与 Auth 的协调

- Auth 侧安全吊销（禁用账号、吊销 token）在 Redis 不可用时返回 `202 PROCESSING`（详见 `auth/rate-limiting.md`）。
- 这意味着在 Redis 故障期间，安全操作可能处于"提交到 DB 但未投影到 Gateway 可见的 Redis"的中间状态。
- 这是已知的最终一致性窗口，不是设计漏洞；Recovery 后 Auth 会补全 Redis 投影，Gateway 再次查黑名单时会生效。
