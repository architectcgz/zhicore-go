# 技术债：Auth 记住我 refresh TTL 目标契约待实现

状态：部分处理
优先级：中
负责人：未分配
来源：Auth 文档更新；`rememberMe=false` 使用 7 天 refresh TTL，`rememberMe=true` 使用 30 天 refresh TTL。

## 影响

Auth 目标文档已经固定 `login.rememberMe`、`auth_refresh_sessions.persistence_policy` 和 refresh rotation 沿用原始持久化策略的规则。当前已完成 login handler / application command / ports / migration 的基础支持；真实 runtime env config、repository 持久化实现、refresh rotation use case、Redis refresh cache TTL、注册成功自动登录 application 行为和相关 contract test 仍待实现。

如果不补实现，前端勾选“记住我”无法真正影响 refresh session 生命周期，且文档中的 7/30 天策略不会被测试保护。

## 退出条件

- `auth_refresh_sessions` repository 实现持久化原始策略字段。
- Auth 运行时配置加载拆分 `AUTH_ACCESS_TOKEN_TTL`、`AUTH_REFRESH_STANDARD_TTL`、`AUTH_REFRESH_REMEMBERED_TTL` 和 `AUTH_REFRESH_TOKEN_PEPPER`，启动期校验 TTL 关系和 secret 必填。
- `RefreshToken` rotation 沿用 session 中保存的策略滚动 `expires_at`，并同步刷新 cookie 与 Redis cache TTL。
- `RegisterAccount` 或后续注册自动登录 use case 真正创建标准 7 天 refresh session；HTTP register handler 当前只以 service 返回值写 cookie，endpoint contract 仍保持“草案”。
- Handler / application / repository 测试覆盖注册自动登录默认标准 7 天、refresh 过期后重新登录、refresh 不允许修改原始策略和 Redis refresh cache TTL 对齐。

## 备注

相关事实源：

- `docs/architecture/module/auth/decision-log.md` #22、#46。
- `docs/architecture/module/auth/service.md`。
- `docs/architecture/module/auth/data-model.md`。
- `services/zhicore-auth/api/http/endpoints/login.md`。
- `services/zhicore-auth/api/http/endpoints/refresh.md`。
