# 获取公开用户资料

状态：已验证。本文固定公开用户资料查询 contract，已由 Go handler / contract test 验证。

## 来源

- 模块 API 设计：`docs/architecture/module/user/api.md`
- 当前 API schema：`services/zhicore-user/api/http/README.md`
- Go handler：`services/zhicore-user/api/http/handler.go`
- Go contract test：`services/zhicore-user/api/http/profile_handler_test.go`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/users/{publicId}` |
| 兼容别名 | 无 |
| Content-Type | 无 body |
| 鉴权 | 匿名 / 登录用户 |
| 幂等 | 查询接口，天然幂等 |

## Path 参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `publicId` | string | 是 | User 对外公开用户 ID。 |

Query / Body 均无。

## 成功响应 `UserProfileResp`

字段见服务级 README 的 `UserProfileResp`。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `publicId` 格式非法。 |
| `3001` | `404` | 用户不存在 | 目标用户不存在、已删除或不可见。 |
| `1004` | `503` | 服务暂时不可用 | User DB 或核心依赖不可用。 |

## 权限和可见性

- 匿名可读取公开资料。
- `DELETED` 建议按 404 返回。
- 头像 URL 解析失败时可省略 `avatarUrl`，但不得伪造。

## 排序、分页和过滤

无。

## 测试要求

- Handler contract test：`services/zhicore-user/api/http/profile_handler_test.go`，覆盖成功、publicId 非法、用户不存在和 avatarUrl 降级。
- System HTTP test：待补。
