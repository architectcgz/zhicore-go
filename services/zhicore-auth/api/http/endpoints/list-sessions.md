# List Sessions

## 来源

- 服务总览：`docs/architecture/services/auth/README.md`
- 模块 API 设计：`docs/architecture/module/auth/api.md`
- 模块 service 设计：`docs/architecture/module/auth/service.md`
- 数据模型：`docs/architecture/module/auth/data-model.md`
- 当前 API schema：`services/zhicore-auth/api/http/README.md`
- Go handler：`services/zhicore-auth/api/http/handler.go`
- Go contract test：`services/zhicore-auth/api/http/auth_handler_test.go`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/auth/sessions` |
| 兼容别名 | 无 |
| Content-Type | 无 body |
| 鉴权 | 登录用户 |
| 幂等 | 查询接口，天然幂等 |

## Path 参数

无。

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `page` | int | 否 | `1` | 页码，从 `1` 开始。 |
| `size` | int | 否 | `20` | 每页数量，最大 `50`。 |

## Body 字段

无。

## 成功响应 `data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `items` | array | 是 | 当前账号的 refresh session 摘要列表，只返回调用者自己的 session。 |
| `page` | int | 是 | 当前页码。 |
| `size` | int | 是 | 当前页大小。 |
| `total` | int | 是 | 当前账号符合条件的 session 总数。 |

`items[]`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `sessionId` | string | 是 | 登录会话 ID。 |
| `createdAt` | string | 是 | RFC3339 创建时间。 |
| `lastSeenAt` | string/null | 是 | RFC3339 最近使用时间；从未 refresh 或使用时为 `null`。 |
| `expiresAt` | string | 是 | RFC3339 过期时间。 |
| `deviceLabel` | string/null | 是 | 设备展示名，例如 `Chrome on macOS`；无法解析时为 `null`。 |
| `current` | boolean | 是 | 是否为当前请求所属 session。 |

不返回 refresh token hash、refresh token 明文、access token `jti`、完整 IP、完整 User-Agent、Redis key 或撤销投影细节。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | `page` / `size` 非法。 |
| `2006` | `401` | 请先登录 | 缺少可信登录身份上下文。 |
| `2015` | `429` | 请求过于频繁 | 触发 Auth 读限流。 |
| `1004` | `503` | 服务暂时不可用 | Auth DB 不可用或查询降级失败。 |

## 权限和可见性

- 用户只能查看自己账号下的 session。
- `current` 由 Gateway 注入的可信 `X-Session-Id` 或等价 Auth 上下文判断。
- 查询结果不暴露其他账号是否存在指定 session。

## 排序、分页和过滤

- 使用 page 分页。
- 默认按 `lastSeenAt DESC NULLS LAST, createdAt DESC, sessionId DESC` 稳定排序。
- 首期只返回未过期且未撤销的 active sessions；历史 revoked / expired session 不在本接口返回。

## 测试要求

- Handler contract test：已验证，覆盖分页默认值、当前 session 标记和敏感字段不返回。
- System HTTP test：待补。
