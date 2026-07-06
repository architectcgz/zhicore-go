# 设置定时发布

状态：已验证。本文固定作者创建或更新定时发布入口，已由 migration / application / repository / handler test 覆盖草稿状态、正文校验、File 引用校验、调度记录和成功 envelope。

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
| 主路径 | `/api/v1/posts/{postId}/schedule` |
| 兼容别名 | 无 |
| Content-Type | `application/json` |
| 鉴权 | 作者，必须由 Gateway 注入 `X-User-Id` |
| 幂等 | 无业务幂等键；同一版本重复设置按状态 / 版本冲突处理 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |

## Query 参数

无。

## Body 字段

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `basePostVersion` | int | 是 | 不允许为空 | 定时发布确认时看到的 post 版本。 |
| `draftBodyId` | string | 是 | 不允许为空 | 定时发布时要上线的草稿 body。 |
| `draftBodyHash` | string | 是 | 不允许为空 | 定时发布时要上线的草稿 body hash，格式 `sha256:<hex>`。 |
| `scheduledAt` | string | 是 | 不允许为空 | RFC3339，必须是未来时间。 |

## 成功响应 `SchedulePostResp`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |
| `postVersion` | int | 是 | 设置定时发布后的新 post 版本。 |
| `status` | string | 是 | 固定为 `SCHEDULED`。 |
| `scheduledAt` | string | 是 | 计划发布时间，RFC3339 UTC。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2006` | `401` | 请先登录 | 缺少可信 `X-User-Id`。 |
| `2008` | `403` | 无权访问该资源 | 当前用户不是作者。 |
| `4001` | `404` | 文章不存在 | post 不存在。 |
| `4004` | `409` | 文章已删除 | 已删除文章不可设置定时发布。 |
| `4016` | `400` | 正文有效文本不足 | 定时发布校验时正文有效 rune 数低于最小要求。 |
| `4017` | `409` | 草稿冲突 | `basePostVersion`、`draftBodyId` 或 `draftBodyHash` 与服务端当前草稿不一致。 |
| `4019` | `409` | 正文 hash 冲突 | `draftBodyHash` 不匹配。 |
| `4021` | `400` | 媒体引用非法 | File 引用不满足发布要求。 |
| `4023` | `400` | 封面不可用 | 草稿封面引用已经不可用或不可发布。 |
| `1001` | `400` | 参数校验失败 | `scheduledAt` 非 RFC3339 或不是未来时间。 |
| `1004` | `503` | 服务暂时不可用 | MongoDB、File、PostgreSQL、outbox 或高副作用路径依赖不可用。 |

## 权限和可见性

- 只有作者可设置定时发布。
- 只有 `DRAFT` 状态文章可以设置定时发布；已发布文章必须先撤回后再重新设置。
- 定时发布成功后公开读接口仍不可见，直到调度执行发布并切换为 `PUBLISHED`。

## 排序、分页和过滤

无。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `SchedulePost` |
| 聚合 | Post lifecycle + scheduled publish record |
| 事务边界 | `posts.status`、版本号、`scheduled_publish_event` 和必要 outbox / internal task 在同一 PostgreSQL 事务中提交。 |
| 事件 | 设置定时发布本身不产生公开可见事件；实际执行发布时产生 `content.post.published` / visibility event。 |

## 测试要求

- Handler contract test：`services/zhicore-content/api/http/post_schedule_handler_test.go`。
- Application test：`services/zhicore-content/internal/content/application/post_schedule_test.go`。
- Repository test：`services/zhicore-content/internal/content/infrastructure/postgres/post_schedule_test.go`。
- Migration contract test：`services/zhicore-content/migrations/migration_contract_test.go`。
