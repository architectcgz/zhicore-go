# 解除拉黑用户

状态：已验证。本文固定 `DELETE /api/v1/users/{publicId}/block` 字段级 contract，已由 Go handler / contract test 验证。

## 来源

- 模块 API 设计：`docs/architecture/module/user/api.md`
- 模块 service 设计：`docs/architecture/module/user/service.md`
- 当前 API schema：`services/zhicore-user/api/http/README.md`
- Go handler：`services/zhicore-user/api/http/handler.go`
- Go contract test：`services/zhicore-user/api/http/relationship_handler_test.go`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `DELETE` |
| 主路径 | `/api/v1/users/{publicId}/block` |
| 兼容别名 | 无 |
| Content-Type | 无 body |
| 鉴权 | 登录用户，`X-User-Id` 必填 |
| 幂等 | 未拉黑时重复解除返回成功，不发事件 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `publicId` | string | 是 | 解除拉黑目标用户的公开 ID。 |

Query / Body 均无。

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `blocked` | boolean | 是 | 固定为 `false`。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2006` | `401` | 请先登录 | 缺少可信 `X-User-Id`。 |
| `1001` | `400` | 参数校验失败 | `publicId` 格式非法。 |
| `3001` | `404` | 用户不存在 | 目标用户不存在。 |
| `1004` | `503` | 服务暂时不可用 | User DB 或 outbox 不可用。 |

## 权限和可见性

- 当前操作者只能来自 Gateway 注入的 `X-User-Id`。
- 解除拉黑允许清理历史关系，即使目标用户已非 `ACTIVE`。
- 解除拉黑不恢复历史关注关系。

## 排序、分页和过滤

无。

## 测试要求

- Handler contract test：`services/zhicore-user/api/http/relationship_handler_test.go`。
- System HTTP test：待补。
