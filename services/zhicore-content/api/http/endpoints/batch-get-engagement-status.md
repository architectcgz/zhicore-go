# batch-get-engagement-status

状态：已验证（handler contract test）。

## 来源

- 服务总览：`docs/architecture/services/content/README.md`
- Engagement 设计：`docs/architecture/services/content/engagement-design.md`
- 运行期 resilience：`docs/architecture/services/content/runtime-resilience.md`
- 当前 API schema：`services/zhicore-content/api/http/README.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/posts/engagement/batch-status` |
| 兼容别名 | 无 |
| Content-Type | `application/json` |
| 鉴权 | 登录用户 |
| 幂等 | 无副作用。 |

## Body 字段

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `postIds` | string[] | 是 | 不允许为空 | 最多 100 个；重复 ID 按首次出现位置去重。 |

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `items` | `EngagementStatusItem[]` | 是 | 顺序与请求去重后的 `postIds` 一致。 |

`EngagementStatusItem`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | 文章公开 ID。 |
| `liked` | boolean \| null | 是 | `null` 表示当前无法确认，不等于未点赞。 |
| `favorited` | boolean \| null | 是 | `null` 表示当前无法确认，不等于未收藏。 |
| `degraded` | bool | 是 | 当前 item 是否因 Redis / DB fallback 降级而状态不可确认。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | body 非法、`postIds` 为空或超过上限。 |
| `1004` | `503` | 服务暂时不可用 | 批量状态服务整体不可用。 |
| `2006` | `401` | 请先登录 | 缺少可信 `X-User-Id`。 |

## 降级语义

- Redis 正常时使用批量读取，禁止逐条网络往返。
- Redis miss 的子集可批量回源 PostgreSQL。
- Redis 不可用时只允许在 fallback budget 内执行一次批量 SQL，禁止循环逐条 `EXISTS`。
- 部分 item 不可确认时，仅对应 item 返回 `liked=null`、`favorited=null`、`degraded=true`。

## 测试要求

- Application test：重复 postId 去重并保持首次顺序。
- Application / repository test：批量状态查询使用批量方法，不逐条回源。
- Handler contract test：登录态、参数上限、确定状态和 degraded item。
