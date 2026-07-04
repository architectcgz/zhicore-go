# zhicore-user HTTP Schema

本目录记录 `zhicore-user` 的对外 HTTP contract。Go handler、contract test、typed client 和 Gateway 路由必须以这里记录的字段级 schema 为准。

## 来源

- 模块设计：`docs/architecture/module/user/README.md`
- 模块 API 设计：`docs/architecture/module/user/api.md`
- 模块 service 设计：`docs/architecture/module/user/service.md`
- Go handler：`services/zhicore-user/api/http/handler.go`
- Go contract test：`services/zhicore-user/api/http/profile_handler_test.go`、`services/zhicore-user/api/http/relationship_handler_test.go`

## 定位

User 拥有用户资料、公开用户 ID、昵称、头像文件引用、简介、陌生人私信设置、关注和拉黑关系。Auth 只拥有账号、凭证、角色和登录态；`auth/me` 不复制 User profile DTO。

## 公共规则

- 响应 envelope：见 `docs/contracts/http.md`。
- 错误码：见 `docs/contracts/error-codes.md`。
- 时间、ID、枚举、空值和 JSON 字段：见 `docs/contracts/data-types.md`。
- 分页、排序和过滤：见 `docs/contracts/pagination.md`。
- `avatarFileId` 是 User 持久化事实；`avatarUrl` 是运行时派生展示字段，不落库、不进事件。
- 当前操作者只来自 Gateway 注入的 `X-User-Id`，不从 request body 接收。

## 通用对象 `UserProfileResp`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `publicId` | string | 是 | User 对外公开用户 ID。 |
| `nickname` | string | 是 | 全局唯一展示名。 |
| `avatarFileId` | string | 否 | File 文件引用。 |
| `avatarUrl` | string | 否 | 头像展示 URL，运行时派生。 |
| `bio` | string | 否 | 个人简介。 |
| `strangerMessageAllowed` | boolean | 是 | 是否允许陌生人私信。 |
| `profileVersion` | int | 是 | 资料版本，用于缓存和事件收敛。 |

## Endpoint 索引

| 方法 | 路径 | 文档 | 状态 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/users/me` | `endpoints/get-me.md` | 已验证 |
| `GET` | `/api/v1/users/{publicId}` | `endpoints/get-profile.md` | 已验证 |
| `PATCH` | `/api/v1/users/me/profile` | `endpoints/update-profile.md` | 已验证 |
| `POST` | `/api/v1/users/{publicId}/block` | `endpoints/block-user.md` | 已验证 |
| `DELETE` | `/api/v1/users/{publicId}/block` | `endpoints/unblock-user.md` | 已验证 |
| `GET` | `/api/v1/users/me/blocked` | `endpoints/list-blocked-users.md` | 已验证 |
| `POST` | `/api/v1/users/{publicId}/follow` | `endpoints/follow-user.md` | 已验证 |
| `DELETE` | `/api/v1/users/{publicId}/follow` | `endpoints/unfollow-user.md` | 已验证 |
| `GET` | `/api/v1/users/{publicId}/followers` | `endpoints/list-followers.md` | 已验证 |
| `GET` | `/api/v1/users/{publicId}/following` | `endpoints/list-following.md` | 已验证 |

## 服务级公开错误码

| code | HTTP status | 含义 | 适用场景 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | path、body 字段非法。 |
| `1004` | `503` | 服务暂时不可用 | PostgreSQL、File service 或下游依赖不可用。 |
| `2006` | `401` | 请先登录 | 登录态 endpoint 缺少 Gateway 注入身份。 |
| `3001` | `404` | 用户不存在 | 目标用户不存在、已删除或不可见。 |
| `3005` | `409` | 昵称已被使用 | 昵称唯一约束冲突。 |
| `3006` | `403` | 用户不可用 | 当前用户或目标用户非 `ACTIVE`。 |
| `3013` | `400` | 昵称不合法 | 昵称为空、过长或含危险字符。 |
| `3014` | `400` | 简介不合法 | 简介过长或含危险字符。 |
| `3015` | `400` | 头像文件不可引用 | File 校验失败或文件不是图片。 |
| `3007` | `400` | 不能关注自己 | 当前用户和目标用户相同。 |
| `3010` | `403` | 互动被拉黑阻止 | 任一方向存在拉黑关系。 |
| `3011` | `400` | 不能拉黑自己 | 当前用户和目标用户相同。 |

## 测试要求

- 每个 endpoint 实现前必须补 handler contract test，覆盖 path、method、鉴权 header、envelope 和错误码。
- 资料更新必须覆盖 nickname、avatarFileId、bio、strangerMessageAllowed、头像 File 校验和 profileVersion 递增。
- 当前 contract test：`services/zhicore-user/api/http/profile_handler_test.go`
- 仅更新本文档和 endpoint schema 时运行 `bash scripts/check-structure.sh` 与 `git diff --check`。
