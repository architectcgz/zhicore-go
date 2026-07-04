# 更新当前用户资料

状态：草案。本文固定当前用户资料更新 contract，尚未由 Go handler / contract test 验证。

## 来源

- 模块 API 设计：`docs/architecture/module/user/api.md`
- 当前 API schema：`services/zhicore-user/api/http/README.md`

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `PATCH` |
| 主路径 | `/api/v1/users/me/profile` |
| 兼容别名 | 无 |
| Content-Type | `application/json` |
| 鉴权 | 登录用户，`X-User-Id` 必填 |
| 幂等 | 同一字段值重复提交应返回相同最终资料 |

Path / Query 均无。

## Body 字段 `UpdateProfileReq`

| 字段 | 类型 | 必填 | 空值语义 | 说明 |
| --- | --- | --- | --- | --- |
| `nickname` | string | 否 | 缺失表示不修改 | 全局唯一展示名。 |
| `avatarFileId` | string/null | 否 | `null` 或空字符串表示清除头像 | File 文件引用；非空时必须是可引用图片。 |
| `bio` | string | 否 | 缺失表示不修改 | 个人简介。 |
| `strangerMessageAllowed` | boolean | 否 | 缺失表示不修改 | 是否允许陌生人私信。 |

不允许更新 `publicId`、`accountId`、`userId` 或 `profileVersion`。

## 成功响应 `UserProfileResp`

字段见服务级 README 的 `UserProfileResp`。公开资料字段变化时必须递增 `profileVersion`；仅 `strangerMessageAllowed` 变化不触发 `user.profile.updated` 事件。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2006` | `401` | 请先登录 | 缺少可信 `X-User-Id`。 |
| `6001` | `404` | 用户不存在 | 当前用户 profile 不存在或已删除。 |
| `6003` | `400` | nickname 非法 | 昵称为空、过长或含危险字符。 |
| `6004` | `409` | nickname 已占用 | 昵称唯一约束冲突。 |
| `6005` | `400` | bio 非法 | 简介过长或含危险字符。 |
| `6006` | `400` | 头像文件不可引用 | File 校验失败或文件不是图片。 |
| `1004` | `503` | 服务暂时不可用 | User DB、File service 或 outbox 不可用。 |

## 权限和可见性

- 只能更新当前登录用户自己的资料。
- request body 中的操作者字段必须拒绝或忽略，不能替代 `X-User-Id`。

## 排序、分页和过滤

无。

## 测试要求

- Handler contract test：待补，覆盖成功、缺登录态、nickname 冲突、头像校验失败、bio 校验和 `profileVersion` 递增。
- System HTTP test：待补。
