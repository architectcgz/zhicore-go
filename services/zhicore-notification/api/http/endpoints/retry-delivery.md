# Retry Delivery

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `POST` |
| 主路径 | `/api/v1/notification-deliveries/{deliveryId}/retry` |
| 兼容别名 | `/api/v1/notifications/deliveries/{deliveryId}/retry` |
| Content-Type | 无 body |
| 鉴权 | 登录用户 |
| 幂等 | 非幂等；每次成功 retry 会更新 attempt 状态 |

## 规则

- 普通用户只能 retry 自己的 delivery。
- 管理员可 retry 任意 delivery。
- Provider 未配置只影响 delivery 状态，不影响站内通知事实。
- 第一阶段不实际启用 SMS provider。

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `deliveryId` | string | 是 | Delivery 短公开 ID，不暴露 `notification_delivery.id`。 |
| `status` | string | 是 | retry 后状态。 |
| `retried` | bool | 是 | 是否触发 retry。 |
