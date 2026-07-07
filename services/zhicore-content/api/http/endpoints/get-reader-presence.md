# get-reader-presence

状态：草案。

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- 领域模型：`docs/architecture/services/content/domain-model.md`
- 限流设计：`docs/architecture/services/content/rate-limiting.md`
- 运行期 resilience：`docs/architecture/services/content/runtime-resilience.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/posts/{postId}/reader-presence` |
| 兼容别名 | 无 |
| Content-Type | 无 |
| 鉴权 | 匿名 |
| 幂等 | 无副作用。 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | 文章公开 ID。 |
| `onlineCount` | int | 是 | 当前可确认的在线读者数量。 |
| `degraded` | bool | 是 | `true` 表示 Redis 不可用或 fallback 预算耗尽，`onlineCount=0` 不能解释为真实无人在线。 |
| `ttlSeconds` | int | 是 | 当前 presence session TTL 秒数。 |

响应不得包含 `userId`、session 列表、IP、设备信息或其他可识别读者身份的数据。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `postId` 为空或格式非法。 |
| `1003` | `429` | 请求过于频繁 | presence 查询限流拒绝。 |
| `4001` | `404` | 文章不存在 | 文章不存在、已删除或不可见。 |

Redis 不可用时不返回 `1004`；返回 HTTP `200`、`onlineCount=0`、`degraded=true`。

## 测试要求

- Application / Redis adapter test：只返回聚合人数，不返回身份列表。
- Handler contract test：正常聚合、Redis 降级空摘要、参数错误和文章不存在映射。
