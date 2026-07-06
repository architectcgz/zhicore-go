# 撤回文章

状态：已验证。本文固定作者撤回已发布文章入口，已由 application / repository / handler test 覆盖作者权限、状态冲突、版本冲突、visibility event 和成功 envelope。

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- 领域模型：`docs/architecture/services/content/domain-model.md`
- Application 设计：`docs/architecture/services/content/application-and-ports.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`
- 大草案：`services/zhicore-content/api/http/endpoints/content-api.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/posts/{postId}/unpublish` |
| 兼容别名 | 无 |
| Content-Type | `application/json` |
| 鉴权 | 作者，必须由 Gateway 注入 `X-User-Id` |
| 幂等 | 无业务幂等键；重复撤回已是未发布状态时返回状态冲突 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |

## Query 参数

无。

## Body 字段

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `basePostVersion` | int | 是 | 不允许为空 | 撤回确认时看到的 post 版本，用于乐观锁。 |

## 成功响应 `LifecycleMutation`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |
| `postVersion` | int | 是 | 撤回后的新 post 版本。 |
| `status` | string | 是 | 固定为 `DRAFT`。 |
| `updatedAt` | string | 是 | 服务端更新时间，RFC3339。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2006` | `401` | 请先登录 | 缺少可信 `X-User-Id`。 |
| `2008` | `403` | 无权访问该资源 | 当前用户不是作者。 |
| `4001` | `404` | 文章不存在 | post 不存在。 |
| `4003` | `409` | 文章未发布 | 文章不是 `PUBLISHED` 状态。 |
| `4004` | `409` | 文章已删除 | 已删除文章不可撤回。 |
| `4017` | `409` | 草稿冲突 | `basePostVersion` 与服务端当前版本不一致。 |
| `1004` | `503` | 服务暂时不可用 | PostgreSQL、outbox 或高副作用路径依赖不可用。 |

## 权限和可见性

- 只有作者可撤回。
- 撤回成功后公开列表、详情和正文读取都不可再看到该文章。
- 撤回不删除 draft / published body pointer；正文资源清理是否发生由 application 状态机和后续 cleanup 策略决定，不由 handler 决定。

## 排序、分页和过滤

无。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `UnpublishPost` |
| 聚合 | Post lifecycle |
| 事务边界 | `posts.status`、版本号和 visibility outbox event 必须在同一个 PostgreSQL 事务中提交。 |
| 事件 | 成功后写入 `content.post.visibility_changed`。 |

## 测试要求

- Handler contract test：`services/zhicore-content/api/http/post_lifecycle_handler_test.go`。
- Application test：`services/zhicore-content/internal/content/application/post_lifecycle_test.go`。
- Repository test：`services/zhicore-content/internal/content/infrastructure/postgres/post_lifecycle_test.go`。
