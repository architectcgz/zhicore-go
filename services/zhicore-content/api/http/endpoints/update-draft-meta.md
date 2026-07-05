# 更新草稿元数据

状态：草案。本文从 `content-api.md` 拆出草稿 metadata 更新入口，Go handler 尚未实现。

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
| `topicId` | string | 否 | 置空表示移除话题。 |
| `categoryId` | string | 否 | 分类引用。 |
| `tags` | string[] | 否 | 最多 10 个。 |

## 成功响应

`data` 为 `Draft`。

## 错误响应

| code | HTTP status | 触发条件 |
| --- | --- | --- |
| `2006` | `401` | 缺少可信 `X-User-Id`。 |
| `2008` | `403` | 当前用户不是作者。 |
| `4001` | `404` | 文章不存在。 |
| `4004` | `409` | 文章已删除。 |
| `4007` | `400` | 标题过长。 |
| `4012` | `404` | 分类、话题或标签引用不存在。 |
| `4017` | `409` | `basePostVersion` 冲突。 |
| `4021` | `400` | File 媒体引用非法。 |
