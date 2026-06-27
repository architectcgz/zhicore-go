# Auth 服务设计

## 事实来源

- Java `zhicore-user` 中的 `AuthController`、登录注册 DTO、token 工具和用户凭证逻辑。
- `docs/architecture/services/user/README.md` 中原本混在 User 内的 `Identity / Profile`、`Auth Token`、角色和凭证设计。
- `docs/architecture/security.md`、`docs/contracts/http.md` 中关于 Gateway、User 和 `libs/kit/auth` 的认证边界。

## 职责边界

`zhicore-auth` 拥有账号认证上下文：账号、登录标识、凭证、账号状态、角色事实、JWT 签发、refresh token 生命周期和 token 失效语义。

Auth 不拥有用户公开资料、关注、拉黑、签到、用户资料摘要、文章、评论、私信或通知。用户资料由 `zhicore-user` 拥有，业务资源权限由资源归属服务 application 判断。

## 与 Gateway / User 的分工

| 服务 | 职责 |
| --- | --- |
| `zhicore-auth` | 签发 access / refresh token，校验 refresh token，维护账号状态、角色、密码 hash、token rotation 和强制失效。 |
| `zhicore-gateway` | 校验外部 access token，查询黑名单或认证缓存，清理客户端伪造身份 header，向下游注入可信 `X-User-*`。 |
| `zhicore-user` | 维护用户公开资料、头像引用、资料版本、关注、拉黑、签到和用户摘要查询。 |
| `libs/kit/auth` | JWT、claims、密码 hash 等技术原语，不拥有账号或权限业务事实。 |

## 数据归属

Auth 拥有：

- `accounts`
- `account_credentials`
- `roles`
- `account_roles`
- Auth 服务自己的 `outbox_events`
- Redis refresh token 白名单、token 黑名单或 token 版本缓存

User 可以把 Auth 的 `accountId` / `userId` 作为用户资料主键或外部引用，但不复制密码、角色事实或 token 状态。

## API 族

- `/api/v1/auth`：注册、登录、refresh、登出、当前认证主体、高风险凭证变更。
- 管理端账号禁用、启用、强制 token 失效后续由 Admin facade 委托 Auth command contract。

字段级 request/response 后续固定到 `services/zhicore-auth/api/http/README.md` 和 `services/zhicore-auth/api/http/endpoints/`。

## 模块设计链接

- 模块入口：`docs/architecture/module/auth/README.md`
- API 背后设计：`docs/architecture/module/auth/api.md`
- Application service：`docs/architecture/module/auth/service.md`
- Domain：`docs/architecture/module/auth/domain.md`
- Ports：`docs/architecture/module/auth/ports.md`
- 数据和事件：`docs/architecture/module/auth/data-events.md`

## 实现风险

- 如果 Auth 拆出后仍从 User 读取密码、状态和角色，会形成两个服务共同拥有身份事实；必须避免。
- 注册流程跨 Auth 和 User，不能用跨服务数据库事务。第一阶段应明确同步创建 profile 还是事件驱动初始化，并登记失败补偿。
- Gateway 的 access token 缓存不能绕过 Auth 的禁用、角色变更和 token 失效语义；需要 TTL、黑名单或 token version 机制闭合。

## 下一步

- 固定 `services/zhicore-auth/api/http` 字段级 contract。
- 设计 Auth 与 User 的注册初始化协议。
- 更新 User migration 设计，把密码、角色和账号状态从 User 表设计中剥离。
