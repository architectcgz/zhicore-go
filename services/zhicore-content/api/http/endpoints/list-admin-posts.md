# 管理端文章列表

状态：已验证。本文从 `content-api.md` 拆出管理端文章查询入口，已由 Go application / handler / repository test 验证管理员角色、分页过滤、响应字段和 PostgreSQL 查询。

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- Application 设计：`docs/architecture/services/content/application-and-ports.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`
- Go handler：`services/zhicore-content/api/http/admin_posts_handlers.go`
- Go contract test：`services/zhicore-content/api/http/admin_posts_handler_test.go`
- Application test：`services/zhicore-content/internal/content/application/admin_posts_test.go`
- Repository test：`services/zhicore-content/internal/content/infrastructure/postgres/admin_posts_test.go`
- 大草案：`services/zhicore-content/api/http/endpoints/content-api.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/admin/content/posts` |
| 兼容别名 | 无 |
| Content-Type | 无 |
| 鉴权 | 管理员，必须由 Gateway 注入 `X-User-Id` 和包含 `admin` 或 `ROLE_ADMIN` 的 `X-User-Roles` |
| 幂等 | 查询幂等 |

## Path 参数

无。

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `status` | string | 否 | 全部状态 | 支持 `draft`、`published`、`scheduled`、`deleted`，application 归一化为大写状态。 |
| `authorId` | int | 否 | 不过滤作者 | 文章作者内部 ID。 |
| `page` | int | 否 | `1` | 从 `1` 开始。 |
| `size` | int | 否 | `20` | application 限制最大值。 |

## 成功响应 `AdminPostListResp`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `items` | `AdminPostItem[]` | 是 | 当前页文章。 |
| `page` | int | 是 | 当前页码。 |
| `size` | int | 是 | 页大小。 |
| `total` | int | 是 | 符合条件的总数。 |

`AdminPostItem`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |
| `authorId` | string | 是 | 作者内部 ID，按字符串返回避免前端数字精度问题。 |
| `authorName` | string | 否 | Content 保存的作者快照。 |
| `authorAvatarFileId` | string | 否 | 作者头像文件 ID 快照。 |
| `title` | string | 是 | 已发布文章取 published 标题，否则取 draft 标题。 |
| `summary` | string | 否 | 已发布文章取 published 摘要，否则取 draft 摘要。 |
| `coverFileId` | string | 否 | 已发布文章取 published 封面，否则取 draft 封面。 |
| `status` | string | 是 | `DRAFT`、`PUBLISHED`、`SCHEDULED` 或 `DELETED`。 |
| `postVersion` | int | 是 | 当前文章版本。 |
| `publishedAt` | string | 否 | 发布时间，RFC3339。 |
| `createdAt` | string | 是 | 创建时间，RFC3339。 |
| `updatedAt` | string | 是 | 更新时间，RFC3339。 |
| `stats` | object | 是 | `viewCount`、`likeCount`、`favoriteCount`、`commentCount`。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2006` | `401` | 请先登录 | 缺少可信 `X-User-Id`。 |
| `2007` | `403` | 需要特定角色 | 缺少管理员角色。 |
| `1001` | `400` | 参数校验失败 | status、authorId、page 或 size 非法。 |
| `1004` | `503` | 服务暂时不可用 | PostgreSQL 等依赖不可用。 |

## 权限和可见性

- 只允许管理员查询所有状态的文章。
- Content 是文章事实 owner；Admin facade 如存在，也必须委托 Content 查询，不复制文章表。

## 排序、分页和过滤

- 使用 page 分页，排序固定为 `updated_at DESC, public_id DESC`。
- `status` 和 `authorId` 都是可选过滤；空过滤仍必须分页。

## 设计追踪

| 项 | 值 |
| --- | --- |
| Use case | `ListAdminPosts` |
| 聚合 | Post / PostStats read model |
| 事务边界 | 只读查询，不改变文章状态。 |
| 事件 | 不产生事件。 |

## 测试要求

- Handler contract test：`services/zhicore-content/api/http/admin_posts_handler_test.go`。
- Application test：`services/zhicore-content/internal/content/application/admin_posts_test.go`。
- Repository test：`services/zhicore-content/internal/content/infrastructure/postgres/admin_posts_test.go`。
