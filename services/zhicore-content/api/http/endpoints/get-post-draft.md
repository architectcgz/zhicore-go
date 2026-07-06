# 读取文章草稿

状态：已验证。本文固定作者读取草稿入口，已由 application / handler test 覆盖可信 `X-User-Id`、作者权限、草稿 metadata、可选 draft body 和错误 envelope。

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/posts/{postId}/draft` |
| 鉴权 | 作者，必须由 Gateway 注入 `X-User-Id` |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |

## 成功响应

`data` 为 `Draft`，包含 `postId`、`postVersion`、`title`、`summary`、`coverFileId`、`status`、可选 `body`、`draftBodyId`、`draftBodyHash`、`createdAt` 和 `updatedAt`。

## 错误响应

| code | HTTP status | 触发条件 |
| --- | --- | --- |
| `2006` | `401` | 缺少可信 `X-User-Id`。 |
| `2008` | `403` | 当前用户不是作者。 |
| `4001` | `404` | 文章不存在。 |
| `4004` | `409` | 文章已删除。 |
| `4018` | `500` | draft body 指针存在但 body 缺失，需要 repair。 |
| `4019` | `409` | draft body hash 校验失败。 |
| `1004` | `503` | PostgreSQL / MongoDB 等依赖不可用。 |

## 测试

- Handler contract test：`services/zhicore-content/api/http/author_workbench_handler_test.go`
- Application test：`services/zhicore-content/internal/content/application/author_workbench_test.go`
- Repository test：`services/zhicore-content/internal/content/infrastructure/postgres/author_workbench_test.go`
