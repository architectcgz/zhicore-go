# Get Notification Unread Breakdown

## 来源

- 服务总览：`docs/architecture/services/notification/README.md`
- 当前 API schema：`services/zhicore-notification/api/http/README.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/notifications/unread/breakdown` |
| 兼容别名 | 无 |
| Content-Type | 无 body |
| 鉴权 | 登录用户 |
| 幂等 | 查询接口，天然幂等。 |

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `total` | int | 是 | 当前用户全部未读数。 |
| `interaction` | int | 是 | `INTERACTION` 未读数。 |
| `content` | int | 是 | `CONTENT` 未读数。 |
| `social` | int | 是 | `SOCIAL` 未读数。 |
| `system` | int | 是 | `SYSTEM` 未读数。 |
| `security` | int | 是 | `SECURITY` 未读数。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2006` | `401` | 请先登录 | 缺少可信登录身份上下文。 |
| `1004` | `503` | 服务暂时不可用 | Notification DB 或 Redis 回源不可用。 |

## 权限和可见性

只返回当前登录用户自己的分类未读数。分类来自 Notification 本地 `category`，不从源服务实时计算。

## 测试要求

- Handler contract test：任务 2 补齐。
- Application / repository test：任务 2 补齐分类统计和 cache-aside。
