# 管理端删除文章

状态：已验证。本文从 `content-api.md` 拆出管理端删除文章入口，已由 Go application / handler / repository test 验证管理员角色、body actor 不可信、软删除、审计上下文、定时发布取消和 visibility outbox。

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- Application 设计：`docs/architecture/services/content/application-and-ports.md`
- 事件契约：`libs/contracts/events/content/post-events.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`
- Go handler：`services/zhicore-content/api/http/admin_posts_handlers.go`
- Go contract test：`services/zhicore-content/api/http/admin_posts_handler_test.go`
- Application test：`services/zhicore-content/internal/content/application/admin_posts_test.go`
- Repository test：`services/zhicore-content/internal/content/infrastructure/postgres/admin_posts_test.go`
- 大草案：`services/zhicore-content/api/http/endpoints/content-api.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `DELETE` |
| 主路径 | `/api/v1/admin/content/posts/{postId}` |
| 兼容别名 | 无 |
| Content-Type | `application/json`，空 body 也允许 |
| 鉴权 | 管理员，必须由 Gateway 注入 `X-User-Id` 和包含 `admin` 或 `ROLE_ADMIN` 的 `X-User-Roles` |
| 幂等 | 已删除文章再次删除返回 `4004` 状态冲突 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |

## Query 参数

无。

## Body 字段 `AdminPostDeleteReq`

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `reason` | string | 否 | 空值由 application 使用默认原因 | 管理员删除原因，写入 Content 本地审计表。 |

请求 body 中的 `userId`、`actor`、`adminUserId` 等字段一律不作为操作者来源；操作者只来自可信 `X-User-Id`。

## 成功响应 `AdminPostDeleteResp`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |
| `status` | string | 是 | 固定为 `DELETED`。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2006` | `401` | 请先登录 | 缺少可信 `X-User-Id`。 |
| `2007` | `403` | 需要特定角色 | 缺少管理员角色。 |
| `4001` | `404` | 文章不存在 | post 不存在。 |
| `4004` | `409` | 文章已删除 | 重复删除。 |
| `1001` | `400` | 参数校验失败 | postId 缺失或 JSON body 非法。 |
| `1004` | `503` | 服务暂时不可用 | PostgreSQL、outbox 或高副作用路径依赖不可用。 |

## 权限和可见性

- 管理端删除不做作者 owner 校验，但必须由 Content application 校验管理员角色和文章状态。
- 删除是软删除，不清理 draft / published body pointer；正文清理由 Content cleanup 机制负责。
- 删除成功后必须写 `content.post.visibility_changed`，payload `reason` 固定为 `ADMIN_DELETED`，`publicVisible=false`。
- 管理员 ID、删除原因、前后状态和发生时间必须写入 Content 本地审计表。

## 排序、分页和过滤

无。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `DeleteAdminPost` |
| 聚合 | Post lifecycle |
| 事务边界 | `posts` 软删除、`admin_post_audit`、定时发布取消和 visibility outbox event 必须在同一个 PostgreSQL 事务中提交。 |
| 事件 | `content.post.visibility_changed`。 |

## 测试要求

- Handler contract test：`services/zhicore-content/api/http/admin_posts_handler_test.go`。
- Application test：`services/zhicore-content/internal/content/application/admin_posts_test.go`。
- Repository test：`services/zhicore-content/internal/content/infrastructure/postgres/admin_posts_test.go`。
