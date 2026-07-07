# upsert-reader-session

状态：已验证（handler contract test）。

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- 领域模型：`docs/architecture/services/content/domain-model.md`
- 限流设计：`docs/architecture/services/content/rate-limiting.md`
- 运行期 resilience：`docs/architecture/services/content/runtime-resilience.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `PUT` |
| 主路径 | `/api/v1/posts/{postId}/reader-sessions/{sessionId}` |
| 兼容别名 | 无 |
| Content-Type | 无 |
| 鉴权 | 匿名 / 登录用户 |
| 幂等 | 是；同一 session heartbeat 刷新 TTL，短间隔 heartbeat 可合并为 no-op success。 |

匿名请求不写入 reader presence，只返回 no-op success。登录用户请求由 Gateway 注入可信 `X-User-Id` 后才可写入 Redis presence 状态。

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |
| `sessionId` | string | 是 | 客户端生成的阅读 session ID；只用于当前文章 presence，不在响应中暴露用户身份。 |

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | 文章公开 ID。 |
| `onlineCount` | int | 是 | 当前可确认的在线读者数量。匿名 no-op 或 Redis 不可用时为 `0`。 |
| `degraded` | bool | 是 | `true` 表示 presence 降级或匿名 no-op，不能代表真实无人在线。 |
| `ttlSeconds` | int | 是 | 服务建议的 heartbeat TTL 秒数；默认语义为 30 秒。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `postId` 或 `sessionId` 为空或格式非法。 |
| `1003` | `429` | 请求过于频繁 | presence 业务限流拒绝，持续 heartbeat 洪水。 |
| `4001` | `404` | 文章不存在 | 文章不存在、已删除或不可见。 |

Redis 不可用时不返回 `1004`；返回 HTTP `200`、`degraded=true` 和空 presence 摘要。

## 副作用

- 登录用户 heartbeat 写入或刷新 Redis presence session TTL。
- 匿名 heartbeat 不写 Redis，不进入在线计数。
- Redis 写失败不影响文章详情、正文读取或公开列表。

## 测试要求

- Application test：匿名请求 no-op success，不写 Redis。
- Application / Redis adapter test：登录用户 heartbeat 刷新 TTL。
- Handler contract test：匿名 no-op、登录成功、参数错误和 Redis 降级 `degraded=true`。
