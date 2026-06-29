# User 服务设计

## 事实来源

- User 模块设计：`docs/architecture/module/user/README.md`
- User 运行期 resilience：`docs/architecture/module/user/runtime-resilience.md`
- User 业务限流：`docs/architecture/module/user/rate-limiting.md`
- User 模块决策日志：`docs/architecture/module/user/decision-log.md`
- 服务边界：`docs/architecture/service-boundaries.md`
- Java `zhicore-user` controller、User 相关 Feign client 和既有 DTO 仅作为能力参考，不作为 Go 字段级 contract 事实源。

## 职责边界

`zhicore-user` 拥有用户资料、公开 `publicId`、唯一 `nickname`、头像文件引用、简介、陌生人私信设置、User 业务状态、关注、粉丝、拉黑、用户摘要和用户可用性查询。

User 不拥有账号凭证、密码 hash、角色事实、管理员封禁、JWT 签发、refresh token、文章、评论、通知或私信。认证账号由 `zhicore-auth` 拥有；管理员封禁使用 Auth `BANNED`。用户资料治理和逻辑删除由 User 拥有，通常通过 Admin facade 调用。

Gateway 只负责认证、清理客户端伪造 header、注入当前操作者内部 `X-User-Id` 和路由，不解析业务目标 `publicId`。

## 模块设计

User 的模块内部设计已迁移到 `docs/architecture/module/user/`：

| 文档 | 内容 |
| --- | --- |
| `docs/architecture/module/user/README.md` | 模块职责、边界、API family、实现切片、关联服务和当前状态。 |
| `docs/architecture/module/user/api.md` | API 背后的业务流程、权限、状态机、副作用和 use case 追踪。 |
| `docs/architecture/module/user/service.md` | Application service、事务边界、幂等、错误映射、缓存失效和实现切片。 |
| `docs/architecture/module/user/domain.md` | 聚合、实体、值对象、不变量、领域服务和工厂。 |
| `docs/architecture/module/user/ports.md` | repository、cache、client、event publisher、outbox 和 external adapter 端口归属。 |
| `docs/architecture/module/user/data-events.md` | 数据归属、目标 schema 草案、缓存 key、事件 payload 和跨服务一致性。 |
| `docs/architecture/module/user/runtime-resilience.md` | timeout、retry、熔断、降级、max-in-flight 和 provider operation 语义。 |
| `docs/architecture/module/user/rate-limiting.md` | 业务限流矩阵、Redis 故障策略和 `RateLimiter` 决策语义。 |
| `docs/architecture/module/user/decision-log.md` | 设计压测中已确认的决策、原因和后续依赖。 |
| [page-design.md](page-design.md) | User 页面初设计、前端草稿、加载逻辑和降级规则。 |

## API 范围

字段级 HTTP schema 后续固定到 `services/zhicore-user/api/http/`。

| API family | 范围 | 状态 |
| --- | --- | --- |
| Profile | 当前用户资料、公开资料查询、资料更新、头像 URL 派生 | 待提取 |
| Profile 状态 | Auth 编排的主动注销、Admin facade 调用的资料逻辑删除和恢复 | 待提取 |
| Block | 拉黑、解除拉黑、拉黑列表、批量拉黑检查 | 待提取 |
| Follow | 关注、取关、粉丝列表、关注列表、关注检查和统计 | 待提取 |
| Typed client | UserSimple、availability、block、follow、陌生人私信设置 | 待提取到 `libs/contracts/clients/user/` |
| Check-in | 签到、签到统计和月度签到图 | 后续切片 |
| Admin | 用户资料管理查询、资料修正、逻辑删除和恢复 | 后续字段级 schema |

前端公开 API 使用 `publicId`；服务间 typed client 使用内部 `userId` 和 `accountId`。

## 数据归属

User 首批拥有：

- `users`
- `user_follows`
- `user_follow_stats`
- `user_blocks`
- User 本地 `outbox_events`

User 本地表之间不建数据库 FK，跨服务也不建 FK。关系完整性由 application/repository 事务、唯一约束、计数非负保护和测试维护。

Check-in 属于 User 完整边界，但不进入首批 schema 细化。

## 跨服务依赖

| 依赖 | 用途 |
| --- | --- |
| Auth | 注册同步初始化 profile；主动注销由 Auth 编排；管理员封禁和账号状态归 Auth。 |
| Upload / File Service | 写头像前校验 `avatarFileId`；前端 HTTP 查询时解析 `avatarUrl`。 |
| Content | 消费 `user.profile.updated` 刷新作者快照；用户主页文章列表直接走 Content 作者过滤接口。 |
| Comment | 写路径调用 User availability 和 block check；查询路径批量获取 UserSimple 作者摘要。 |
| Message | 查询 block/follow 和 `strangerMessageAllowed`。 |
| Notification | 消费 `user.followed` 创建关注通知；其他 User 事件按产品需要登记。 |
| Admin | 作为管理端 facade 调 User 资料管理 contract 和 Auth 账号治理 contract。 |

## Go 目标落点

- HTTP：`services/zhicore-user/api/http`
- Application：`services/zhicore-user/internal/user/application`
- Domain：`services/zhicore-user/internal/user/domain`
- Ports：`services/zhicore-user/internal/user/ports`
- Infrastructure：`services/zhicore-user/internal/user/infrastructure`
- Runtime：`services/zhicore-user/internal/user/runtime/module.go`
- Typed client contract：`libs/contracts/clients/user/`

## 实现风险

- `nickname` 是全局唯一且大小写敏感，非 ACTIVE 用户继续占用；默认 nickname 冲突会让 Auth 注册链路失败并进入补偿。
- User 前端 HTTP 可返回 `avatarUrl`，但 URL 不落库、不进事件、不进默认 typed client。
- `profileVersion` 只在公开资料变化时递增；陌生人私信设置、关注、拉黑和签到不触发 `user.profile.updated`。
- `DEACTIVATED` / `DELETED` 不清理关注和拉黑关系；写路径必须用 `user_status=ACTIVE` guard。
- Admin 封禁不能写 User 状态；封禁归 Auth `BANNED`。
- 关注统计是可重建读模型，事实源是 `user_follows`。
- User 缓存不是正确性依赖；Redis 缓存故障可受控回源 DB，但限流 Redis 故障必须按 User 业务矩阵 fail-open / fail-closed。
- 服务间 typed client 必须携带 `X-Caller-Service` 和 `X-Caller-Operation`，缺少 caller identity 的内部接口不能落到匿名公开配额。
- `BatchGetUserSimple` 中缺失用户和 `SERVICE_DEGRADED` 必须区分；写路径 guard 在 User degraded 时 fail closed。

## 下一步

1. 按 `docs/architecture/module/user/` 生成 `services/zhicore-user/api/http/README.md` 和首批 endpoint schema。
2. 提取 `libs/contracts/clients/user/` typed client contract。
3. 落地 User runtime 配置、resilience policy 和 `RateLimiter` 决策语义。
4. 生成 User migration 草案，覆盖 `users`、`user_follows`、`user_follow_stats`、`user_blocks` 和 outbox。
5. 按首批切片顺序实现：Profile 基础、Profile 状态、Block、Follow。
