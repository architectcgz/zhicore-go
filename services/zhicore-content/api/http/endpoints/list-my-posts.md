# 我的文章列表

状态：草案。本文从 `content-api.md` 拆出作者工作台列表入口，Go handler 尚未实现。

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/me/posts` |
| 鉴权 | 登录用户，必须由 Gateway 注入 `X-User-Id` |

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `status` | string | 否 | `all` | `all`、`draft`、`published`、`scheduled`、`deleted`。 |
| `cursor` | string | 否 | 无 | Opaque cursor。 |
| `limit` | int | 否 | `20` | `1..100`。 |

## 成功响应

`data` 为 `CursorPage<PostSummary>`，排序固定为 `updated_at DESC, public_id DESC`。

## 错误响应

| code | HTTP status | 触发条件 |
| --- | --- | --- |
| `2006` | `401` | 缺少可信 `X-User-Id`。 |
| `1001` | `400` | status、cursor 或 limit 非法。 |
| `1004` | `503` | PostgreSQL 等依赖不可用。 |
