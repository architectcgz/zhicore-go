# zhicore-auth

`zhicore-auth` 是账号认证服务的 Go 目标服务模块。

服务职责：

- 拥有账号身份、登录标识、登录凭证、账号状态、角色事实和 JWT 签发 / refresh 行为。
- 管理 refresh token 白名单、token rotation、登出、强制失效和高风险凭证变更。
- 向 Gateway 提供 access token 校验所需的签名、claims 规则和失效语义；Gateway 仍负责入口校验和可信身份 header 注入。
- 向 User 提供账号创建后的用户资料初始化所需事实，例如 `accountId`、`username`、默认昵称来源。

数据归属：

- `accounts`
- `account_credentials`
- `roles`
- `account_roles`
- Auth 服务自己的 `outbox_events`
- Redis refresh token 白名单、token 黑名单或 token 版本缓存

Go 设计注意点：

- Auth 不拥有用户公开资料、关注、拉黑、签到或用户资料摘要。
- User 不保存密码 hash，不签发 token，不维护角色事实。
- 账号状态和角色是认证事实；用户资料是否可展示、关系是否可互动由对应业务服务 application 判断。
