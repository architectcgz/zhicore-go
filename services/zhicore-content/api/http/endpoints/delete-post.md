# 删除文章

状态：已验证。本文固定作者软删除文章入口，已由 application / repository / handler test 覆盖作者权限、重复删除、软删除不清正文指针、visibility event 和成功 envelope。

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- 领域模型：`docs/architecture/services/content/domain-model.md`
- Application 设计：`docs/architecture/services/content/application-and-ports.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`
- 大草案：`services/zhicore-content/api/http/endpoints/content-api.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `DELETE` |
| 主路径 | `/api/v1/posts/{postId}` |
| 兼容别名 | 无 |
| Content-Type | 无 |
| 鉴权 | 作者，必须由 Gateway 注入 `X-User-Id` |
| 幂等 | 重复删除已删除文章返回 `4004` 状态冲突 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `basePostVersion` | int | 否 | 无 | 删除确认时看到的 post 版本；传入时必须与当前版本一致。 |

## Body 字段

无。

## 成功响应 `LifecycleMutation`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |
| `postVersion` | int | 是 | 删除后的新 post 版本。 |
| `status` | string | 是 | 固定为 `DELETED`。 |
| `updatedAt` | string | 是 | 服务端更新时间，RFC3339。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2006` | `401` | 请先登录 | 缺少可信 `X-User-Id`。 |
| `2008` | `403` | 无权访问该资源 | 当前用户不是作者。 |
| `4001` | `404` | 文章不存在 | post 不存在。 |
| `4004` | `409` | 文章已删除 | 重复删除。 |
| `4017` | `409` | 草稿冲突 | `basePostVersion` 与服务端当前版本不一致。 |
| `1001` | `400` | 参数校验失败 | `basePostVersion` query 不是正整数。 |
| `1004` | `503` | 服务暂时不可用 | PostgreSQL、outbox 或高副作用路径依赖不可用。 |

## 权限和可见性

- 只有作者可删除自己的文章。
- 删除成功后公开列表、详情和正文读取都不可再看到该文章。
- 删除是软删除，不立即删除 draft / published body pointer；后续清理由 Content cleanup 机制负责，不能由 handler 直接删除 MongoDB body。

## 排序、分页和过滤

无。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `DeletePost` |
| 聚合 | Post lifecycle |
| 事务边界 | `posts.status`、版本号和 visibility outbox event 必须在同一个 PostgreSQL 事务中提交。 |
| 事件 | 成功后写入 `content.post.deleted` 或 `content.post.visibility_changed`。 |

## 测试要求

- Handler contract test：`services/zhicore-content/api/http/post_lifecycle_handler_test.go`。
- Application test：`services/zhicore-content/internal/content/application/post_lifecycle_test.go`。
- Repository test：`services/zhicore-content/internal/content/infrastructure/postgres/post_lifecycle_test.go`。
