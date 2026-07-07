# list-tags

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- 应用与端口：`docs/architecture/services/content/application-and-ports.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`
- Go handler：`services/zhicore-content/api/http/taxonomy_handlers.go`
- Go contract test：`services/zhicore-content/api/http/taxonomy_handler_test.go`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/tags` |
| 兼容别名 | 无 |
| Content-Type | 无 |
| 鉴权 | 匿名 |
| 幂等 | 无 |

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `cursor` | string | 否 | 无 | 不透明 cursor。 |
| `limit` | int | 否 | `20` | 最大 `100`。 |

## 成功响应 `data`

`CursorPage<Tag>`。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `items` | `Tag[]` | 是 | 标签列表。 |
| `nextCursor` | string | 否 | 下一页 cursor。 |
| `hasMore` | bool | 是 | 是否还有下一页。 |
| `limit` | int | 是 | 实际 limit。 |

`Tag`：`tagId`、`name`、`slug`、`postCount`。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `cursor` 或 `limit` 非法。 |
| `1004` | `503` | 服务暂时不可用 | PostgreSQL 不可用。 |

## 排序、分页和过滤

- Cursor 分页，排序为 `slug ASC, id ASC`。
- 空列表返回 `items=[]`、`hasMore=false`。

## 测试要求

- Handler contract test：覆盖默认分页、非法 limit 和成功 envelope。

状态：已验证。
