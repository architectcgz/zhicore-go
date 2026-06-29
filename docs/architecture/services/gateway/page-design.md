# Gateway 页面设计

本文记录 Gateway 相关管理 / 运维页面的前端草稿、页面分区、加载状态和降级规则。Gateway 没有面向普通用户的产品页面；本文只定义内部路由、认证、限流和依赖诊断台的初设计。

当前状态：本文固定页面初设计和加载逻辑，不表示前端已经实现。

## 设计原则

- Gateway 页面是内部运维工具，不是业务前台页面。
- Gateway 不拥有业务数据，不展示用户、文章、评论等业务详情，只展示入口路由、认证、限流和下游健康状态。
- 诊断页面必须脱敏：不展示 JWT、refresh token、Authorization header、cookie、完整 IP 或敏感 header。
- 路由配置查看和诊断优先只读；任何运行时开关都需要权限、原因、审计和回滚路径。
- Redis / Auth 降级状态必须明确区分 L1 cache 命中、Auth fallback、fail closed 和高风险路由阻断。

## 页面范围

本文覆盖：

- Gateway 状态总览。
- 路由表查看。
- 认证链路诊断。
- 限流和路由风险策略查看。
- 下游服务健康检查。

本文不覆盖：

- 普通用户登录页，这些归 Auth 页面设计。
- 业务服务页面和管理动作，这些归对应服务或 Admin 页面设计。

## Gateway 状态总览

### 页面草稿

```text
┌────────────────────────────────────────────┐
│ Gateway ops                                │
├────────────────────────────────────────────┤
│ Status: routing · auth · redis · rate limit│
├────────────────────────────────────────────┤
│ Traffic summary                            │
│ request rate · errors · p95 · degraded     │
├────────────────────────────────────────────┤
│ Alerts / recent failures                   │
└────────────────────────────────────────────┘
```

### 加载逻辑

1. 进入页面确认管理员或运维权限。
2. 并行加载 Gateway 自身 health、Redis/auth 投影状态、路由摘要和限流摘要。
3. 某个依赖诊断失败时只降级对应面板。
4. Gateway 自身不可达时由外层运维入口显示服务不可用。

## 路由表查看

### 页面草稿

```text
┌────────────────────────────────────────────┐
│ Routes                                     │
├────────────────────────────────────────────┤
│ Filters: service · method · auth required  │
├────────────────────────────────────────────┤
│ Table                                      │
│ method · path · target · auth · risk       │
├────────────────────────────────────────────┤
│ Route detail · timeouts · headers          │
└────────────────────────────────────────────┘
```

规则：

- 展示 path、method、目标服务、是否匿名、风险等级和 timeout。
- 内部身份 header 展示为名称列表，不展示真实值。
- 路由缺失或冲突是配置错误，页面显示告警，不在 UI 中临时修补业务 path。

## 认证链路诊断

### 页面草稿

```text
┌────────────────────────────────────────────┐
│ Auth diagnostics                           │
├────────────────────────────────────────────┤
│ JWT verify · blacklist · principal cache   │
│ auth fallback · high-risk fail closed      │
├────────────────────────────────────────────┤
│ Recent auth failures                       │
│ code · route · risk · request id           │
└────────────────────────────────────────────┘
```

页面只能展示聚合诊断：

- token 校验成功 / 失败计数。
- Redis blacklist / principal cache 可用性。
- Auth fallback 调用成功率、超时和熔断状态。
- 高风险路由 fail closed 次数。

禁止展示：

- JWT 原文、claims 全量、refresh token、cookie、Authorization header。
- 可重放的 session ID 明文或 token `jti` 原值。

## 限流和风险策略

### 页面草稿

```text
┌────────────────────────────────────────────┐
│ Rate limit / route risk                    │
├────────────────────────────────────────────┤
│ Risk groups: anonymous · normal · high     │
├────────────────────────────────────────────┤
│ Counters                                   │
│ key pattern · current · limited · fallback │
└────────────────────────────────────────────┘
```

规则：

- Redis 不可用时展示入口级限流是否降级、哪些 route fail open / fail closed。
- 高风险 route 的降级策略以 `route-risk-policy.md` 为准。
- 限流 key 只能展示脱敏后的 pattern，不展示完整用户 ID、IP 或 token。

## 下游健康检查

### 页面草稿

```text
┌────────────────────────────────────────────┐
│ Downstream services                        │
├────────────────────────────────────────────┤
│ service · base url · health · p95 · errors │
│ auth · user · content · comment · ...      │
├────────────────────────────────────────────┤
│ Detail · timeout · circuit state           │
└────────────────────────────────────────────┘
```

加载逻辑：

1. Gateway 聚合下游健康和最近转发错误。
2. 单个服务不可用时当前行显示 degraded。
3. 点击详情展示 timeout、熔断、最近错误码和 requestId。
4. 不展示下游业务响应 body 中的敏感字段。

## 跨服务页面约定

- Gateway 诊断页面不能成为业务聚合页面。
- Gateway 不转换业务响应，不在页面里配置业务 DTO 映射。
- 任何运行时开关都应走 Ops 或 Admin 的审计流程；Gateway 页面首期只读。
