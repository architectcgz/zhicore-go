# Auth 模块架构

`auth` 模块对应 `zhicore-auth` 服务内的账号认证上下文。

## 模块职责

- 管理账号、登录标识、密码 hash、账号状态和角色事实。
- 签发 access / refresh token，维护 refresh session、token hash rotation、登出和强制失效。
- 为 Gateway 提供 access token 校验所需的 claims、黑名单 / token version 语义。
- 向 User 发布账号注册、账号禁用、角色变更等事件或提供同步 contract。

## 边界

模块拥有认证事实，不拥有用户公开资料：

- 昵称、头像、简介、资料版本归 User。
- 关注、拉黑、签到归 User。
- 文章、评论、私信、通知等资源权限归对应业务服务。
- Admin 只做 facade 和审计，不复制 Auth 状态变更逻辑。

## API Family

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/me`
- `GET /api/v1/auth/sessions`
- `DELETE /api/v1/auth/sessions/current`
- `DELETE /api/v1/auth/sessions/{sessionId}`
- `GET /api/v1/auth/security-operations/{operationId}`

后续再补密码修改、密码重置、账号禁用 / 启用、角色变更和账号级 token 全量失效。

## 文档拆分

| 文档 | 内容 |
| --- | --- |
| `api.md` | API 背后的业务流程、权限、状态机、副作用和 use case 追踪。 |
| `service.md` | Application service、事务边界、幂等、错误映射和运行机制。 |
| `domain.md` | 聚合、值对象、不变量和领域事件。 |
| `ports.md` | repository、token、cache、outbox、User client 等端口。 |
| `data-model.md` | PostgreSQL 表、字段、索引、约束、保留策略和 migration 切片建议。 |
| `data-events.md` | 数据归属、缓存、事件和跨服务一致性。 |
| `rate-limiting.md` | Gateway 粗限流、Auth 业务安全限流、Redis 故障限流降级和 API 限流矩阵。 |
| `redis-keys.md` | Auth/Gateway 协作使用的 Redis key、TTL、敏感信息边界和故障语义。 |
