# zhicore-user HTTP Schema

本目录记录 `zhicore-user` 的对外 HTTP contract。Go handler、contract test、typed client 和 Gateway 路由必须以这里记录的字段级 schema 为准。

## 来源

- 模块设计：`docs/architecture/module/user/README.md`
- 模块 API 设计：`docs/architecture/module/user/api.md`
- 模块 service 设计：`docs/architecture/module/user/service.md`
- Go handler：待实现
- Go contract test：待补

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
| `GET` | `/api/v1/users/me` | `endpoints/get-me.md` | 草案 |
| `GET` | `/api/v1/users/{publicId}` | `endpoints/get-profile.md` | 草案 |
| `PATCH` | `/api/v1/users/me/profile` | `endpoints/update-profile.md` | 草案 |

## 服务级公开错误码

| code | HTTP status | 含义 | 适用场景 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | path、body 字段非法。 |
| `1004` | `503` | 服务暂时不可用 | PostgreSQL、File service 或下游依赖不可用。 |
| `2006` | `401` | 请先登录 | 登录态 endpoint 缺少 Gateway 注入身份。 |
| `6001` | `404` | 用户不存在 | 目标用户不存在、已删除或不可见。 |
| `6002` | `403` | 用户不可用 | 目标用户非 `ACTIVE`，且当前场景不允许展示。 |
| `6003` | `400` | nickname 非法 | 昵称为空、过长或含危险字符。 |
| `6004` | `409` | nickname 已占用 | 昵称唯一约束冲突。 |
| `6005` | `400` | bio 非法 | 简介过长或含危险字符。 |
| `6006` | `400` | 头像文件不可引用 | File 校验失败或文件不是图片。 |

## 测试要求

- 每个 endpoint 实现前必须补 handler contract test，覆盖 path、method、鉴权 header、envelope 和错误码。
- 资料更新必须覆盖 nickname、avatarFileId、bio、strangerMessageAllowed、头像 File 校验和 profileVersion 递增。
- 仅更新本文档和 endpoint schema 时运行 `bash scripts/check-structure.sh` 与 `git diff --check`。
