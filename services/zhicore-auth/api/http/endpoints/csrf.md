# Get CSRF Token

## 来源

- 服务总览：`docs/architecture/services/auth/README.md`
- 模块 API 设计：`docs/architecture/module/auth/api.md`
- CSRF 决策：`docs/architecture/module/auth/decision-log.md`
- 限流设计：`docs/architecture/module/auth/rate-limiting.md`
- 当前 API schema：`services/zhicore-auth/api/http/README.md`
- Go handler：待实现
- Go contract test：待补

## 请求

| 项 | 值 |
| --- | --- |
| 方法 | `GET` |
| 主路径 | `/api/v1/auth/csrf` |
| 兼容别名 | 无 |
| Content-Type | 无 body |
| 鉴权 | 匿名 |
| 幂等 | 可重复获取；每次可签发新的 CSRF token 并覆盖 cookie。 |

## Path 参数

无。

## Query 参数

无。

## Body 字段

无。

## 成功响应 `data`

响应必须设置非 HttpOnly `csrf_token` cookie，并在 body 返回同值，供前端后续提交 `X-CSRF-Token`。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `csrfToken` | string | 是 | 新签发的 CSRF token。 |

`GET /api/v1/auth/csrf` 不签发 access token，不签发或轮换 refresh token，不改变 refresh session，也不要求已有登录态。

## 错误响应

| code | HTTP status | message 语义 | 触发条件 |
| --- | --- | --- | --- |
| `2015` | `429` | 请求过于频繁 | 触发 IP 或 session 维度的 CSRF endpoint 限流。 |
| `1004` | `503` | 服务暂时不可用 | CSRF token 签发依赖或配置不可用。 |

## 权限和可见性

- 匿名可调用，用于多标签页旧 CSRF token 失效后的恢复路径。
- CSRF token 防跨站伪造，不防本站 XSS；不得把它当作登录身份或 refresh 凭证。
- 响应不暴露 refresh token、sessionId、accountId、userId 或 Redis key。

## 排序、分页和过滤

无。

## 测试要求

- Handler contract test：待补，覆盖匿名获取、覆盖旧 `csrf_token` cookie、不签发 refresh/access token、限流。
- System HTTP test：待补。
