# Update Author Subscription

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `PUT` |
| 主路径 | `/api/v1/author-subscriptions/{authorId}` |
| 兼容别名 | `/api/v1/notifications/author-subscriptions/{authorId}` |
| Content-Type | `application/json` |
| 鉴权 | 登录用户 |
| 幂等 | 同一 payload 重复提交结果一致 |

## Request Body

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `level` | string | 是 | `ALL`、`DIGEST_ONLY`、`MUTED`。 |
| `inAppEnabled` | bool | 是 | `ALL` 时按提交值保存。 |
| `websocketEnabled` | bool | 是 | `DIGEST_ONLY` 和 `MUTED` 会归一化为 `false`。 |
| `emailEnabled` | bool | 是 | `DIGEST_ONLY` 和 `MUTED` 会归一化为 `false`。 |
| `digestEnabled` | bool | 是 | `DIGEST_ONLY` 会归一化为 `true`，`MUTED` 为 `false`。 |

## 规则

- `DIGEST_ONLY` 只允许摘要投递。
- `MUTED` 禁用所有主动和摘要通道。
- 用户通知偏好是全局 gate；`EMAIL` 默认为关闭，只有用户在 `notification-preferences` 显式开启对应通知类型的 `EMAIL`，作者订阅的 `DIGEST_ONLY` 才会生成摘要投递。
- 提交成功后失效 `notification:{userId}:author:{authorId}:subscription`。
