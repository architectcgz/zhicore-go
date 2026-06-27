# Auth HTTP Schema

本目录记录 `zhicore-auth` 的服务级 HTTP contract。当前只固定服务边界和首批 endpoint 方向；字段级 schema 后续按 `docs/contracts/http-schema-template.md` 拆到 `endpoints/`。

## 服务级规则

- 服务拥有 `/api/v1/auth` API family。
- `register`、`login`、`refresh` 直接处理客户端凭证；普通业务服务不得解析 `Authorization` 作为身份来源。
- `me` 读取 Gateway 注入的可信身份上下文，或在 Auth 服务自身入口中由 Auth middleware 校验 access token 后构造等价上下文。
- 成功和失败响应使用 `docs/contracts/http.md` 定义的 ZhiCore envelope。

## Endpoint 索引

| Endpoint | Use case | 设计文档 | Contract 状态 |
| --- | --- | --- | --- |
| `POST /api/v1/auth/register` | `RegisterAccount` | `docs/architecture/module/auth/service.md` | 待提取 |
| `POST /api/v1/auth/login` | `Login` | `docs/architecture/module/auth/service.md` | 待提取 |
| `POST /api/v1/auth/refresh` | `RefreshToken` | `docs/architecture/module/auth/service.md` | 待提取 |
| `POST /api/v1/auth/logout` | `Logout` | `docs/architecture/module/auth/service.md` | 待提取 |
| `GET /api/v1/auth/me` | `GetCurrentPrincipal` | `docs/architecture/module/auth/service.md` | 待提取 |

## 待确认

- 是否保留 Java 既有 `GET /api/v1/auth/me` 的完整响应字段，还是仅返回认证主体并让前端按需调用 User 资料接口。
- `register` 是否同步创建 User profile，或由 Auth 发布 `auth.account.registered` 后由 User 消费初始化。
