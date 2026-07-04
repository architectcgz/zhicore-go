# zhicore-ranking HTTP Schema

本目录记录 `zhicore-ranking` 的对外 HTTP contract。Go handler、contract test、typed client 和 Gateway 路由必须以这里记录的字段级 schema 为准。

## 来源

- 服务总览：`docs/architecture/services/ranking/README.md`
- 领域模型：`docs/architecture/services/ranking/domain-model.md`
- Application / Ports：`docs/architecture/services/ranking/application-and-ports.md`
- 查询与物化：`docs/architecture/services/ranking/query-materialization.md`
- 运行期 resilience：`docs/architecture/services/ranking/runtime-resilience.md`
- 决策日志：`docs/architecture/services/ranking/decision-log/2026-06-29-ranking-design-decisions.md`
- 当前 API schema：`services/zhicore-ranking/api/http/endpoints/ranking-api.md`
- Go handler：待实现
- Go contract test：待实现

## 定位

Ranking API 暴露文章榜、创作者榜、话题榜、热门候选集和管理员 rebuild 入口。Ranking 拥有分数、排名和榜单物化，不拥有文章详情、用户资料或话题源事实。

对外 `postId` 使用 Content 的公开 ID。Ranking 内部 `post_id BIGINT` 不出现在外部 HTTP response。服务间 typed client 候选集可以额外返回 `internalId`，但外部 HTTP 只返回公开文章 ID 字段。

## 公共规则

- 响应 envelope：见 `docs/contracts/http.md`。
- 错误码：见 `docs/contracts/error-codes.md`。
- 时间、ID、枚举、空值和 JSON 字段：见 `docs/contracts/data-types.md`。
- 分页、排序和过滤：见 `docs/contracts/pagination.md`。
- 运行期 timeout、retry、熔断、降级和观测：见 `docs/architecture/services/ranking/runtime-resilience.md`。
- 字段级 endpoint schema：见 [endpoints/ranking-api.md](endpoints/ranking-api.md)。本轮前端 provider adapter 先固定公开热榜 `GET /api/v1/ranking/posts/hot` 和公开热榜分数 `GET /api/v1/ranking/posts/hot/scores`。

## 鉴权上下文

| 鉴权类型 | Header | 说明 |
| --- | --- | --- |
| 匿名 | 无需 `X-User-Id` | 可读取公开榜单、分数和热门候选集。 |
| 服务间调用 | `X-Caller-Service` + `X-Caller-Operation` | 下游服务读取候选集或内部榜单时必填，用于限流、审计和观测。 |
| 管理员 | `X-User-Id` + `X-User-Roles` | `rebuild-from-ledger` 需要管理员角色。 |

客户端伪造的 `X-User-*` 和 `X-Caller-*` header 必须由 Gateway 清理后重新注入。Ranking handler 不从 request body 接收当前操作者 `userId`。

## Endpoint 索引

| 方法 | 路径 | 文档 | 状态 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/ranking/posts/hot` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/posts/hot/details` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/posts/hot/scores` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/posts/hot/candidates` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/posts/daily` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/posts/weekly` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/posts/monthly` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/posts/daily/scores` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/posts/weekly/scores` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/posts/monthly/scores` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/posts/{postId}/rank` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/posts/{postId}/score` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/creators/hot` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/creators/hot/scores` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/creators/{userId}/rank` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/creators/{userId}/score` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/topics/hot` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/topics/hot/scores` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/topics/{topicId}/rank` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/topics/{topicId}/score` | `endpoints/ranking-api.md` | 草案 |
| `POST` | `/api/v1/ranking/admin/rebuild-from-ledger` | `endpoints/ranking-api.md` | 草案 |
| `GET` | `/api/v1/ranking/admin/rebuild-operations/{operationId}` | `endpoints/ranking-api.md` | 草案 |

## 服务级公开错误码

| code | HTTP status | 含义 | 适用场景 |
| --- | --- | --- | --- |
| `1001` | `400` | 参数校验失败 | path/query/body 字段非法、分页参数越界、period 参数非法。 |
| `1003` | `429` | 请求过于频繁 | Ranking 业务限流命中。 |
| `1004` | `503` | 服务暂时不可用 | PostgreSQL、Redis 回源、Content 解析、MongoDB archive 或 rebuild lock 等依赖不可用。 |
| `1005` | `404` | 数据不存在 | 目标文章、创作者、话题或冷历史榜单不存在。 |
| `1008` | `409` | 操作不允许 | rebuild 已在运行、状态不允许或请求与当前任务冲突。 |
| `2006` | `401` | 请先登录 | 管理端 endpoint 缺少 Gateway 注入身份。 |
| `2007` | `403` | 需要特定角色 | 管理端 endpoint 缺少管理员角色。 |
| `2008` | `403` | 无权访问该资源 | 已登录但无权执行管理操作。 |

## 测试要求

- 每个 endpoint 实现前必须补 handler contract test，覆盖 path、method、query/body、鉴权 header、envelope 和错误码。
- 列表 endpoint 必须覆盖默认分页、最大 `size`、非法参数、稳定排序、空榜和 Redis miss 回源。
- 详情 endpoint 必须覆盖 Content 批量详情不可用时返回 `1004`，不能把只有 score 的结果伪装成详情。
- 管理 endpoint 必须覆盖缺少登录态、非管理员、rebuild 已运行、lock 不可用、accepted response 和 status 查询。
- 仅更新本文档和 endpoint schema 时运行 `bash scripts/check-structure.sh` 与 `git diff --check`。
