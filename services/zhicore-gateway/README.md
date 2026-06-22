# zhicore-gateway

`zhicore-gateway` 是网关服务的 Go 迁移模块。

服务职责：

- 处理边缘路由、认证拦截、CORS 和网关级过滤器。
- 校验 JWT，并维护网关使用的 token 校验缓存和黑名单缓存。
- 在迁移开发期间承接 Go 服务目标路由配置和回滚策略。

数据归属：

- 网关路由配置
- token 黑名单和校验缓存
- 路由切换状态

迁移注意点：

- Gateway 不拥有用户身份、角色、登录凭证或业务 API schema。
- 网关只做入口控制和路由，不直接实现业务逻辑。
- 当前开发阶段不做灰度，Gateway 不实现用户灰度判断。
- 服务发现优先适配 Kubernetes Service/DNS；是否兼容 Nacos 只作为过渡策略处理。
