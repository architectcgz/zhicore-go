# 取消定时发布

状态：已验证。本文固定作者取消定时发布入口，已由 application / repository / handler test 覆盖作者权限、非定时状态、取消调度记录和成功 envelope。

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
| 主路径 | `/api/v1/posts/{postId}/schedule` |
| 兼容别名 | 无 |
| Content-Type | 无 |
| 鉴权 | 作者，必须由 Gateway 注入 `X-User-Id` |
| 幂等 | 无业务幂等键；非 `SCHEDULED` 状态返回状态冲突 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `basePostVersion` | int | 否 | 无 | 取消确认时看到的 post 版本；传入时必须与当前版本一致。 |

## Body 字段

无。

## 成功响应 `LifecycleMutation`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |
| `postVersion` | int | 是 | 取消定时后的新 post 版本。 |
| `status` | string | 是 | 固定为 `DRAFT`。 |
| `updatedAt` | string | 是 | 服务端更新时间，RFC3339。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2006` | `401` | 请先登录 | 缺少可信 `X-User-Id`。 |
| `2008` | `403` | 无权访问该资源 | 当前用户不是作者。 |
| `4001` | `404` | 文章不存在 | post 不存在。 |
| `4003` | `409` | 文章未发布 | 文章不是 `SCHEDULED` 状态，无法取消定时发布。 |
| `4004` | `409` | 文章已删除 | 已删除文章不可取消定时发布。 |
| `4017` | `409` | 草稿冲突 | `basePostVersion` 与服务端当前版本不一致。 |
| `1001` | `400` | 参数校验失败 | `basePostVersion` query 不是正整数。 |
| `1004` | `503` | 服务暂时不可用 | PostgreSQL 或调度记录依赖不可用。 |

## 权限和可见性

- 只有作者可取消定时发布。
- 取消成功后文章回到 `DRAFT`，公开读接口仍不可见。

## 排序、分页和过滤

无。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `CancelSchedule` |
| 聚合 | Post lifecycle + scheduled publish record |
| 事务边界 | `posts.status`、版本号和 `scheduled_publish_event` 取消状态在同一 PostgreSQL 事务中提交。 |
| 事件 | 不产生公开可见事件。 |

## 测试要求

- Handler contract test：`services/zhicore-content/api/http/post_schedule_handler_test.go`。
- Application test：`services/zhicore-content/internal/content/application/post_schedule_test.go`。
- Repository test：`services/zhicore-content/internal/content/infrastructure/postgres/post_schedule_test.go`。
