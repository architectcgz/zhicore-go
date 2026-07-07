# List Deliveries

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/notification-deliveries` |
| 兼容别名 | `/api/v1/notifications/deliveries` |
| Content-Type | 无 body |
| 鉴权 | 登录用户 |
| 幂等 | 查询接口，天然幂等 |

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `cursor` | string | 否 | 无 | 不透明游标。 |
| `size` | int | 否 | `20` | 最大 `50`。 |
| `channel` | string | 否 | 无 | `IN_APP`、`WEBSOCKET`、`EMAIL`、`SMS`。 |
| `status` | string | 否 | 无 | Delivery 状态。 |
## 权限

普通用户只能查询自己的 delivery；管理员后续可扩展 recipient 过滤。

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `items[].deliveryId` | string | 是 | Delivery `public_id`，不是内部 `BIGINT id`。 |
| `items[].notificationId` | string | 否 | 关联通知的 `notifications.public_id`。 |
| `items[].channel` | string | 是 | 投递通道。 |
| `items[].status` | string | 是 | Delivery 状态。 |
| `nextCursor` | string | 否 | 不透明 cursor；包含稳定翻页锚点，consumer 不解析。 |
