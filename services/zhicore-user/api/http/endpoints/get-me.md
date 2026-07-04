# 获取当前用户资料

状态：已验证。本文固定 `GET /api/v1/users/me` 字段级 contract，已由 Go handler / contract test 验证。

## 来源

- 模块 API 设计：`docs/architecture/module/user/api.md`
- 当前 API schema：`services/zhicore-user/api/http/README.md`
- Go handler：`services/zhicore-user/api/http/handler.go`
- Go contract test：`services/zhicore-user/api/http/profile_handler_test.go`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/users/me` |
| 兼容别名 | 无 |
| Content-Type | 无 body |
| 鉴权 | 登录用户，`X-User-Id` 必填 |
| 幂等 | 查询接口，天然幂等 |

Path / Query / Body 均无。

## 成功响应 `UserProfileResp`

字段见服务级 README 的 `UserProfileResp`。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2006` | `401` | 请先登录 | 缺少可信 `X-User-Id`。 |
| `3001` | `404` | 用户不存在 | 当前登录用户 profile 尚未创建、已删除或不可见。 |
| `1004` | `503` | 服务暂时不可用 | User DB 或核心依赖不可用。 |

## 权限和可见性

- 只能读取当前登录用户自己的资料。
- Auth principal 字段不在本接口重复返回。
- `avatarUrl` 解析失败时省略该字段，不影响整个 profile 响应。

## 排序、分页和过滤

无。

## 测试要求

- Handler contract test：`services/zhicore-user/api/http/profile_handler_test.go`，覆盖成功、缺 `X-User-Id`、profile 缺失和 avatarUrl 派生。
- System HTTP test：待补。
