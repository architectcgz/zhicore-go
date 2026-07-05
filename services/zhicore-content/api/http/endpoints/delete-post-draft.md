# 删除文章草稿

状态：草案。本文从 `content-api.md` 拆出草稿删除入口，Go handler 尚未实现。

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `DELETE` |
| 主路径 | `/api/v1/posts/{postId}/draft` |
| 鉴权 | 作者，必须由 Gateway 注入 `X-User-Id` |

## 成功响应

`data` 可省略。已发布文章删除草稿不影响线上 published body；Content application 必须为旧 draft body 创建 cleanup task。

## 错误响应

| code | HTTP status | 触发条件 |
| --- | --- | --- |
| `2006` | `401` | 缺少可信 `X-User-Id`。 |
| `2008` | `403` | 当前用户不是作者。 |
| `4001` | `404` | 文章不存在。 |
| `4004` | `409` | 文章已删除。 |
| `1004` | `503` | PostgreSQL 等依赖不可用。 |
