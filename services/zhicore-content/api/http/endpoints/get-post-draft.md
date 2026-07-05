# 读取文章草稿

状态：草案。本文从 `content-api.md` 拆出作者读取草稿入口，Go handler 尚未实现。

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

`data` 为 `Draft`，包含 `postId`、`postVersion`、`meta`、可选 `draftBody`、`draftBodyHash` 和 `savedAt`。

## 错误响应

| code | HTTP status | 触发条件 |
| --- | --- | --- |
| `2006` | `401` | 缺少可信 `X-User-Id`。 |
| `2008` | `403` | 当前用户不是作者。 |
| `4001` | `404` | 文章不存在。 |
| `4004` | `409` | 文章已删除。 |
| `1004` | `503` | PostgreSQL / MongoDB 等依赖不可用。 |
