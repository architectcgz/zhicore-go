# Auth API 背后设计

本文只描述 API 背后的业务流程和 use case 追踪；字段级 HTTP schema 放在 `services/zhicore-auth/api/http/`。

## 鉴权上下文

| API | 鉴权 | 说明 |
| --- | --- | --- |
| `POST /api/v1/auth/register` | 匿名 | 创建账号凭证和默认角色，并触发 User profile 初始化。 |
| `POST /api/v1/auth/login` | 匿名 | 校验登录标识、账号状态和密码，签发 token。 |
| `POST /api/v1/auth/refresh` | 匿名 + refresh token | 校验 refresh token，执行 rotation。 |
| `POST /api/v1/auth/logout` | 登录用户或 refresh token | 吊销当前 refresh token；access token 黑名单按 contract 决定。 |
| `GET /api/v1/auth/me` | 登录用户 | 返回当前认证主体、账号状态和角色；用户资料由 User 查询。 |

## Use Case 追踪

| Endpoint | Use case | 主要副作用 |
| --- | --- | --- |
| `POST /api/v1/auth/register` | `RegisterAccount` | 写入账号、凭证、默认角色、outbox；初始化 User profile 或发布事件。 |
| `POST /api/v1/auth/login` | `Login` | 写入 refresh token 白名单，更新登录安全审计。 |
| `POST /api/v1/auth/refresh` | `RefreshToken` | 吊销旧 refresh token，写入新 refresh token；重放时吊销该账号全部 refresh token。 |
| `POST /api/v1/auth/logout` | `Logout` | 吊销当前 refresh token，必要时写 access token 黑名单。 |
| `GET /api/v1/auth/me` | `GetCurrentPrincipal` | 无业务写入。 |

## 注册流程

第一阶段推荐使用同步 profile 初始化，避免注册成功但用户资料缺失：

```text
Auth RegisterAccount
-> Auth 本地事务创建 account / credential / role / outbox
-> 调用 User CreateProfileForAccount
-> 成功后返回 token 或账号摘要
```

如果后续改成事件驱动，必须定义 pending profile 状态、补偿任务和前端可见错误语义。

## `me` 的返回边界

`GET /api/v1/auth/me` 只返回认证主体事实，例如 `accountId`、`username`、`roles`、`status`。昵称、头像、简介和用户展示摘要由 User contract 提供，避免 Auth 复制 profile DTO。
