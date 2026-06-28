# Gateway 服务设计

## 事实来源

- Java `zhicore-gateway` 模块。
- Java 架构文档 `02-microservices-list.md`。
- 当前迁移约束：前端暂时不修改，外部 API 形态保持稳定。

## 职责边界

`zhicore-gateway` 是统一外部入口，负责路由、认证上下文、CORS、限流和基础观测。

Gateway 不拥有业务数据，不定义用户、文章、评论、通知、搜索或排行的业务模型。

## 当前决策

当前项目按个人项目和早期部署复杂度处理，先采用薄 Go Gateway 作为应用入口服务。

默认部署形态：

```text
Nginx 或本地反向代理
-> zhicore-gateway
-> 各 Go 后端服务
```

规则：

- Nginx 只承担 TLS、静态资源、基础反向代理、body size、超时和基础访问日志。
- `zhicore-gateway` 负责统一认证、token blacklist、请求 ID、入口级限流、CORS、路由转发和网关自身错误 envelope。
- `zhicore-gateway` 是普通业务 HTTP 请求的唯一 JWT 校验点；下游业务服务不做 JWT 解析 fallback。
- 暂不引入 Kubernetes Ingress 作为当前默认入口；只有部署进入 Kubernetes 后，再用 Ingress 放在 Go Gateway 前面。
- 暂不引入 APISIX、Kong、Envoy 等专业 API Gateway；只有动态路由、复杂限流、灰度、多租户或插件化治理成为真实需求时再评估。
- Gateway 必须保持薄入口定位，不做业务聚合、不定义业务 DTO、不做下游响应转换、不判断资源归属权限。

## 认证和分流

Gateway 的认证链路：

```text
外部请求 Authorization: Bearer <access-token>
-> Gateway 校验 JWT、过期时间、issuer / audience、黑名单和认证缓存
-> Gateway 清理客户端传入的内部身份 header
-> Gateway 写入 `X-Account-Id` / `X-User-Id` / `X-Session-Id` / `X-User-Roles` 等可信身份 header
-> Gateway 按路由规则分流到下游服务
```

规则：

- 匿名 endpoint 和登录/注册/refresh endpoint 由 Gateway 白名单或路由规则显式放行。
- 登录态业务 endpoint 必须先在 Gateway 完成认证；认证失败由 Gateway 返回兼容错误 envelope。
- Gateway 可以按 path、method、host、部署配置和认证结果分流请求，但不按文章作者、评论作者、关注关系等业务事实分流。
- Gateway 转发前必须清理或覆盖客户端伪造的 `X-Account-Id`、`X-User-Id`、`X-User-Name`、`X-User-Roles`、`X-Session-Id`、`X-Session-Version` 和 `X-Principal-Version`；身份 header 以 Gateway 注入值为准。
- `X-Request-Id` 和 `X-Trace-Id` 按观测规范校验、生成或透传，不作为身份事实。
- 下游服务只读取 Gateway 注入的可信身份上下文，不解析客户端 JWT。
- Redis 不可用时，Gateway 可在短 TTL L1 命中且非高风险路由时短时放行；L1 miss、principal 版本需要刷新或 refresh 降级后的新 access token 校验，应调用 Auth typed client `ValidateAccessState` 回源校验 access state。Auth 不可用或高风险路由无法确认最新撤销状态时 fail closed。
- `ValidateAccessState` fallback 只能作为 Redis 故障或版本刷新路径使用，不能变成每请求常规 introspection。Gateway 必须设置短超时、singleflight、并发上限和熔断；连续失败时快速 fail closed，并记录 degraded metric。
- high-risk / normal 路由分类、L1 cache 使用窗口和 fail-closed 规则见 `route-risk-policy.md`。

## API 保留范围

Go Gateway 必须保持前端当前访问路径可用。它可以把请求路由到 Go 服务、Ingress 或本地开发地址，但不能把后端迁移造成的 path、method、响应封装变化暴露给前端。

## 数据归属

Gateway 不拥有业务表。可使用 Redis 保存：

- token 黑名单。
- 鉴权缓存。
- 限流计数。

这些数据只服务入口控制，不是业务事实源。

Auth 相关 Redis key 由 Auth 模块定义，见 `docs/architecture/module/auth/redis-keys.md`；Gateway 只消费撤销、版本和 principal cache 投影。

## Go 目标落点

- HTTP 入口：`services/zhicore-gateway/api/http`
- 私有实现：`services/zhicore-gateway/internal/gateway`
- 配置：`services/zhicore-gateway/configs`

Gateway 的 `runtime/module.go` 负责装配路由、认证 middleware、反向代理和观测组件。

## 运行时依赖

- 服务发现：当前阶段使用本地配置或环境变量；进入 Kubernetes 后再切换为 Kubernetes Service DNS。
- 配置注入：当前阶段使用 env 和本地配置模板；进入 Kubernetes 后再映射为 ConfigMap、Secret。
- 认证：JWT 校验和 token 黑名单。
- 限流：Go middleware 和 Redis。

不迁移 Nacos、Spring Cloud Gateway、Sentinel 的技术形态。

## 实现风险

- Gateway 很容易被做成 API 形态转换层。当前约束是不改变前端契约，不能把服务内部重构泄漏到 Gateway。
- 灰度相关逻辑不迁移；当前开发阶段不做 Java/Go 并存。

## 下一步

- 从前端实际调用和 Java Gateway 路由配置提取路由清单。
- 固定认证失败、权限失败、限流失败的响应格式。
- 设计 Go Gateway 的最小 middleware 链。
- 固定 Gateway 注入给下游服务的身份 header：`X-Request-Id`、`X-Trace-Id`、`X-Account-Id`、`X-User-Id`、`X-User-Name`、`X-User-Roles`、`X-Session-Id`、`X-Session-Version`、`X-Principal-Version`。
