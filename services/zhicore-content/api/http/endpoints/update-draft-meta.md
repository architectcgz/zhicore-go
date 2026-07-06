# 更新草稿元数据

状态：已验证。本文固定草稿 metadata 更新入口第一阶段，已由 application / repository / handler test 覆盖可信 `X-User-Id`、作者权限、乐观锁、标题 / 摘要 / 封面更新和错误 envelope。

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `PATCH` |
| 主路径 | `/api/v1/posts/{postId}/draft/meta` |
| 鉴权 | 作者，必须由 Gateway 注入 `X-User-Id` |
| Content-Type | `application/json` |

## Body 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `basePostVersion` | int | 是 | 乐观锁版本。 |
| `title` | string | 否 | 最大 200。 |
| `summary` | string | 否 | 用户摘要。 |
| `coverFileId` | string | 否 | 置空表示移除封面。 |
| `topicId` | string | 否 | 任务 8 taxonomy 落地前传入返回 `1001`。 |
| `categoryId` | string | 否 | 任务 8 taxonomy 落地前传入返回 `1001`。 |
| `tags` | string[] | 否 | 任务 8 taxonomy 落地前传入返回 `1001`。 |

## 成功响应

`data` 为 `DraftMutation`，包含 `postId`、`postVersion`、`title`、`summary`、`coverFileId` 和 `updatedAt`。

## 错误响应

| code | HTTP status | 触发条件 |
| --- | --- | --- |
| `2006` | `401` | 缺少可信 `X-User-Id`。 |
| `2008` | `403` | 当前用户不是作者。 |
| `4001` | `404` | 文章不存在。 |
| `4004` | `409` | 文章已删除。 |
| `1001` | `400` | `basePostVersion` 非法，或本阶段传入 `topicId` / `categoryId` / `tags`。 |
| `4007` | `400` | 标题过长。 |
| `4017` | `409` | `basePostVersion` 冲突。 |
| `4023` | `400` | 封面文件不可用。 |

## 测试

- Handler contract test：`services/zhicore-content/api/http/author_workbench_handler_test.go`
- Application test：`services/zhicore-content/internal/content/application/author_workbench_test.go`
- Repository test：`services/zhicore-content/internal/content/infrastructure/postgres/author_workbench_test.go`
