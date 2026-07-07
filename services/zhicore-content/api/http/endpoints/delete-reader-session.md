# delete-reader-session

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
| 方法 | `DELETE` |
| 主路径 | `/api/v1/posts/{postId}/reader-sessions/{sessionId}` |
| 兼容别名 | 无 |
| Content-Type | 无 |
| 鉴权 | 匿名 / 登录用户 |
| 幂等 | 是；重复 leave 或不存在的 session 返回成功。 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `postId` | string | 是 | Content 公开文章 ID。 |
| `sessionId` | string | 是 | 客户端生成的阅读 session ID。 |

## 成功响应 `data`

空对象 `{}`。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `postId` 或 `sessionId` 为空或格式非法。 |
| `4001` | `404` | 文章不存在 | 文章不存在、已删除或不可见。 |

离开动作是收敛动作，不应被普通限流阻断。Redis 不可用时仍返回 HTTP `200` 和空对象，记录 degraded metric，不返回 `1004`。

## 副作用

- 登录用户 leave 删除 Redis presence session。
- 匿名 leave 不写 Redis，直接空成功。

## 测试要求

- Application test：重复 leave 幂等成功。
- Application / Redis adapter test：Redis delete 失败返回空成功。
- Handler contract test：空成功、参数错误和文章不存在映射。
