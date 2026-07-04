# 获取粉丝列表

状态：已验证。本文固定 `GET /api/v1/users/{publicId}/followers` 字段级 contract，已由 Go handler / contract test 验证。

## 来源

- 模块 API 设计：`docs/architecture/module/user/api.md`
- 模块 service 设计：`docs/architecture/module/user/service.md`
- 当前 API schema：`services/zhicore-user/api/http/README.md`
- Go handler：`services/zhicore-user/api/http/handler.go`
- Go contract test：`services/zhicore-user/api/http/relationship_handler_test.go`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/users/{publicId}/followers` |
| 兼容别名 | 无 |
| Content-Type | 无 body |
| 鉴权 | 匿名 / 登录用户 |
| 幂等 | 查询接口，天然幂等 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `publicId` | string | 是 | 被查看粉丝的目标用户公开 ID。 |

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `cursor` | string | 否 | 空 | 不透明游标。 |
| `limit` | int | 否 | `20` | 每页数量，最大 `100`。 |

Body 无。

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `items` | `UserProfileResp[]` | 是 | 粉丝用户摘要。 |
| `nextCursor` | string | 否 | 下一页不透明游标；无下一页时省略或为空。 |
| `hasMore` | boolean | 是 | 是否还有下一页。 |

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `publicId`、`cursor` 或 `limit` 非法。 |
| `3001` | `404` | 用户不存在 | 目标用户不存在或已删除不可见。 |
| `1004` | `503` | 服务暂时不可用 | User DB 不可用。 |

## 权限和可见性

- 匿名可查看公开粉丝列表。
- 列表可返回非 `ACTIVE` 用户的摘要；展示占位由 application/query owner 决定。

## 排序、分页和过滤

- Cursor 分页，按 `user_follows.id DESC` 稳定排序。
- `cursor` 对外不透明，不能要求前端解析内部关系 ID。

## 测试要求

- Handler contract test：`services/zhicore-user/api/http/relationship_handler_test.go`。
- System HTTP test：待补。
