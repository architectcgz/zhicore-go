# zhicore-gateway HTTP Schema

本目录记录 `zhicore-gateway` 自有 HTTP contract。Gateway 是薄入口，不定义 Content / User / Comment 等业务 DTO。

## Provider Owner

Gateway 拥有路由、认证上下文注入、CORS、入口级限流、请求 ID / trace ID 和自身错误 envelope。Gateway 不拥有业务数据，不做下游响应形态转换，不判断资源归属权限。

## Gateway 自有 endpoint 候选

| 方法 | 路径 | 用途 | 状态 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/gateway/health` | Gateway 自身健康检查 | API 族已识别 |
| `GET` | `/api/v1/gateway/routes` | 路由诊断，仅内部/管理员 | API 族已识别 |
| `GET` | `/api/v1/gateway/auth/diagnostics` | 认证链路诊断，仅内部/管理员 | API 族已识别 |

## 自有错误 envelope

Gateway 可以定义并返回以下入口级错误 envelope：

- 认证失败：access token 缺失、无效、过期或撤销状态不可确认。
- 权限失败：路由需要登录或角色，但 Gateway 无法构造可信身份上下文。
- 限流失败：入口级限流命中。
- 路由失败：下游不可达或路由未配置。

这些错误只描述入口控制结果，不包含 provider 业务 DTO。

## 禁止规则

- 不定义 Content / User / Comment / Auth / File / Ranking 的业务 request / response。
- 不把 Gateway 做成 API 形态转换层。
- 转发前必须清理客户端伪造的内部身份 header，再注入可信 `X-Account-Id`、`X-User-Id`、`X-Session-Id` 等。
- 暂不创建前端 `src/api/gateway.ts`，除非诊断 endpoint 达到 `Contract 草案`。
