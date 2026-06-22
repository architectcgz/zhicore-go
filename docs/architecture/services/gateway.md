# Gateway 服务设计

## 事实来源

- Java `zhicore-gateway` 模块。
- Java 架构文档 `02-microservices-list.md`。
- 当前迁移约束：前端暂时不修改，外部 API 形态保持稳定。

## 职责边界

`zhicore-gateway` 是统一外部入口，负责路由、认证上下文、CORS、限流和基础观测。

Gateway 不拥有业务数据，不定义用户、文章、评论、通知、搜索或排行的业务模型。

## API 保留范围

Go Gateway 必须保持前端当前访问路径可用。它可以把请求路由到 Go 服务、Ingress 或本地开发地址，但不能把后端迁移造成的 path、method、响应封装变化暴露给前端。

## 数据归属

Gateway 不拥有业务表。可使用 Redis 保存：

- token 黑名单。
- 鉴权缓存。
- 限流计数。

这些数据只服务入口控制，不是业务事实源。

## Go 目标落点

- HTTP 入口：`services/zhicore-gateway/api/http`
- 私有实现：`services/zhicore-gateway/internal/gateway`
- 配置：`services/zhicore-gateway/configs`

Gateway 的 `runtime/module.go` 负责装配路由、认证 middleware、反向代理和观测组件。

## 运行时依赖

- 服务发现：Kubernetes Service DNS 或本地配置。
- 配置注入：env、ConfigMap、Secret。
- 认证：JWT 校验和 token 黑名单。
- 限流：Go middleware 和 Redis。

不迁移 Nacos、Spring Cloud Gateway、Sentinel 的技术形态。

## 迁移风险

- Gateway 很容易被做成 API 形态转换层。当前约束是不改变前端契约，不能把服务内部重构泄漏到 Gateway。
- 灰度相关逻辑不迁移；当前开发阶段不做 Java/Go 并存。

## 下一步

- 从前端实际调用和 Java Gateway 路由配置提取路由清单。
- 固定认证失败、权限失败、限流失败的响应格式。
- 设计 Go Gateway 的最小 middleware 链。
