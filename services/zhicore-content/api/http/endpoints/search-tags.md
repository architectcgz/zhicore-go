# search-tags

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
| 主路径 | `/api/v1/tags/search` |
| 兼容别名 | 无 |
| Content-Type | 无 |
| 鉴权 | 匿名 |
| 幂等 | 无 |

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `q` | string | 是 | 无 | 按 slug / name 前缀搜索，trim 后至少 1 个字符。 |
| `limit` | int | 否 | `10` | 最大 `20`。 |

## 成功响应 `data`

`Tag[]`，每项包含 `tagId`、`name`、`slug`、`postCount`。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `q` 为空或 `limit` 非法。 |
| `1004` | `503` | 服务暂时不可用 | PostgreSQL 不可用。 |

## 排序、分页和过滤

- 排序为前缀匹配优先、`postCount DESC, slug ASC`。
- 空结果返回空数组。

## 测试要求

- Handler contract test：覆盖 q 必填、limit 上限和成功数组响应。

状态：已验证。
