# Gateway 路由与认证基础实现计划

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；涉及认证、路由、限流、header 清理、runtime 和系统测试的步骤按 @test-driven-development 执行。每个 checkbox 达到预期后立即更新；如需提交，提交前必须先使用 @committing-changes。

**目标：** 把 `zhicore-gateway` 从薄入口占位推进到能承载路由清单、JWT 校验、身份 header 注入、Auth fallback、诊断 endpoint 和最小系统验证的 Gateway foundation。

**架构：** Gateway 不拥有业务 DTO、不转换下游响应、不判断资源归属权限；它只做入口认证、路由、CORS、限流、观测和可信身份 header 注入。Java Gateway 只作为外部 path 覆盖面参考，Go upstream owner 必须按 Go 服务边界重排。

**技术栈：** Go 1.26、Gin、`httputil.ReverseProxy`、JWT、Redis/L1 cache、Auth typed client、route manifest、Gateway HTTP schema。

---

## 背景依据

- `docs/architecture/services/gateway/README.md`
- `docs/architecture/services/gateway/route-risk-policy.md`
- `docs/architecture/services/gateway/redis-degradation.md`
- `services/zhicore-gateway/api/http/README.md`
- `docs/architecture/security.md`
- `docs/architecture/runtime-operations.md`
- `docs/contracts/http.md`
- 需要核对既有入口时读取 `../zhicore-microservice/zhicore-gateway/src/main/resources/application.yml`

## 当前基线

- 生产 Go 源码只有 `services/zhicore-gateway/internal/gateway/doc.go`。
- 自有 HTTP schema 只有 3 个候选 endpoint，没有 `endpoints/`。
- Auth fallback typed contract、route manifest、JWT verifier、Redis/L1 cache 和 reverse proxy runtime 都未落地。

## 不可并行修改文件

- `libs/contracts/clients/auth/contract.go`：由本计划任务 2 首次固定 `ValidateAccessState`；Admin 计划必须等待本任务合并后再追加管理端账号操作 contract。

## 任务 1：路由清单与 HTTP schema 固化

**测试立场：** R0 文档 / 配置切片。

**文件：**
- 修改：`docs/architecture/services/gateway/README.md`
- 修改：`services/zhicore-gateway/README.md`
- 修改：`services/zhicore-gateway/api/http/README.md`
- 新增：`services/zhicore-gateway/api/http/endpoints/health.md`
- 新增：`services/zhicore-gateway/api/http/endpoints/routes-diagnostics.md`
- 新增：`services/zhicore-gateway/api/http/endpoints/auth-diagnostics.md`
- 新增：`services/zhicore-gateway/configs/routes.example.yaml`

**验收清单：**
- [ ] 外部 path 至少登记 `/api/v1/auth/**`、`/api/v1/users/**`、`/api/v1/posts/**`、`/api/v1/comments/**`、`/api/v1/files/**`、`/api/v1/admin/**`、`/ws/message/**`、`/ws/notification/**`、`/api/v1/gateway/*`。
- [ ] 每条 route 标明 upstream Go owner、anonymous / normal / high-risk、`stripPrefix`、`requiredRole`、`preserveHost`、timeout 和 body size 策略。
- [ ] `/api/v1/admin/**` 明确可按配置 strip 到 Admin 下游 `/admin/**`。
- [ ] Gateway 自有诊断 endpoint 是 Go-first 新增，只读且仅内部 / 管理员可见。
- [ ] Java 路由只作为外部 path 覆盖面参考，不作为 Go upstream service 名事实源。

- [ ] **步骤 1：核对 Java application.yml 中现有路由和白名单**
- [ ] **步骤 2：补 route manifest 示例和 endpoint schema**
- [ ] **步骤 3：运行文档验证**

运行：`bash scripts/check-structure.sh && git diff --check`

预期：通过。

## 任务 2：Auth fallback contract 与 Gateway 配置模型

**测试立场：** TDD - 认证 fallback 和路由策略属于 R4。

**文件：**
- 新增：`libs/contracts/clients/auth/contract.go`
- 新增：`libs/contracts/clients/auth/contract_test.go`
- 修改：`libs/contracts/clients/auth/README.md`
- 新增：`services/zhicore-gateway/cmd/server/config.go`
- 新增：`services/zhicore-gateway/cmd/server/config_loader.go`
- 新增：`services/zhicore-gateway/cmd/server/config_defaults.go`
- 新增：`services/zhicore-gateway/cmd/server/config_validation.go`
- 新增：`services/zhicore-gateway/cmd/server/config_test.go`
- 新增：`services/zhicore-gateway/internal/gateway/ports/auth.go`
- 新增：`services/zhicore-gateway/internal/gateway/ports/route_config.go`
- 新增：`services/zhicore-gateway/internal/gateway/ports/rate_limit.go`
- 新增：`services/zhicore-gateway/internal/gateway/application/claims.go`
- 新增：`services/zhicore-gateway/internal/gateway/application/route_policy.go`
- 新增：`services/zhicore-gateway/internal/gateway/application/*_test.go`

**验收清单：**
- [ ] `ValidateAccessState` 只接收已验签 claims，不接 raw token。
- [ ] 本任务只固定 Gateway fallback 需要的 Auth contract；不要混入 Admin disable / enable 语义。
- [ ] Auth fallback response 至少包含 `decision`、`denyReason`、`principal`、`principalRefreshed`、`cacheTtlSeconds`。
- [ ] 路由配置模型支持 `stripPrefix`、`requiredRole`、`riskLevel`、`preserveHost`、`extraHeaders`。
- [ ] runtime 配置必须覆盖 HTTP `ReadHeaderTimeout`、`ReadTimeout`、`WriteTimeout`、`IdleTimeout`、shutdown timeout、每 route timeout、body size、CORS allow origins / methods / headers、Redis fallback window、Auth client timeout 和限流参数。
- [ ] config test 覆盖 bool 严格解析、duration / size 单位、非正数 / overflow 拒绝、secret / DSN / token redaction、缺必填 upstream baseURL 失败。
- [ ] high-risk route 在 Redis / Auth 不可确认时 fail closed。
- [ ] normal 读路径只允许短 TTL L1 兜底，且必须记录 degraded metric。

- [ ] **步骤 1：写 Auth contract 和 route policy 失败测试**
- [ ] **步骤 2：实现 typed contract、配置模型和 route policy**
- [ ] **步骤 3：运行验证**

运行：`cd libs/contracts && go test ./clients/auth -count=1 && cd ../../services/zhicore-gateway && go test ./cmd/server ./internal/gateway/application -count=1`

预期：通过。

## 任务 3：匿名 / 公开路由代理骨架

**测试立场：** TDD - reverse proxy、header 清理和 envelope 属于 R4。

**文件：**
- 新增：`services/zhicore-gateway/api/http/handler.go`
- 新增：`services/zhicore-gateway/api/http/proxy_handlers.go`
- 新增：`services/zhicore-gateway/api/http/errors.go`
- 新增：`services/zhicore-gateway/api/http/request_helpers.go`
- 新增：`services/zhicore-gateway/api/http/rate_limit_middleware.go`
- 新增：`services/zhicore-gateway/api/http/rate_limit_middleware_test.go`
- 新增：`services/zhicore-gateway/api/http/proxy_handler_test.go`
- 新增：`services/zhicore-gateway/internal/gateway/runtime/module.go`
- 新增：`services/zhicore-gateway/internal/gateway/runtime/route_table.go`
- 新增：`services/zhicore-gateway/internal/gateway/runtime/*_test.go`
- 新增：`services/zhicore-gateway/internal/gateway/infrastructure/ratelimit/redis_limiter.go`
- 新增：`services/zhicore-gateway/internal/gateway/infrastructure/ratelimit/l1_limiter.go`
- 新增：`services/zhicore-gateway/internal/gateway/infrastructure/ratelimit/*_test.go`
- 新增：`services/zhicore-gateway/cmd/server/main.go`
- 新增：`services/zhicore-gateway/configs/local.example.env`

**验收清单：**
- [ ] 公开白名单 path 可匿名转发。
- [ ] 受保护 path 缺 token 返回 ZhiCore envelope，不返回裸 Java 风格错误。
- [ ] 转发前清理客户端伪造的 `X-Account-Id`、`X-User-*`、`X-Session-*`、`X-Caller-*`。
- [ ] `X-Request-Id` / `X-Trace-Id` 校验后透传或生成。
- [ ] `/api/v1/admin/**` 可按配置 strip 到下游 `/admin/**`。
- [ ] 限流支持 route / principal / IP 维度；超过限制返回 ZhiCore envelope、HTTP 429、公开错误码和 `Retry-After`，不返回裸 Redis 错误。
- [ ] Redis 限流不可用时 high-risk route fail closed；normal route 只允许配置窗口内 L1 兜底，并记录 degraded detail。

- [ ] **步骤 1：写 proxy handler 失败测试**
- [ ] **步骤 2：实现路由表、header 清理和 reverse proxy**
- [ ] **步骤 3：运行代理测试**

运行：`cd services/zhicore-gateway && go test ./api/http ./internal/gateway/runtime ./internal/gateway/infrastructure/ratelimit -count=1`

预期：通过。

## 任务 4：JWT 校验、Redis/L1 和 Auth fallback middleware

**测试立场：** TDD - 认证、缓存降级、权限错误属于 R4。

**文件：**
- 新增：`services/zhicore-gateway/internal/gateway/infrastructure/jwt/verifier.go`
- 新增：`services/zhicore-gateway/internal/gateway/infrastructure/jwt/verifier_test.go`
- 新增：`services/zhicore-gateway/internal/gateway/infrastructure/cache/l1.go`
- 新增：`services/zhicore-gateway/internal/gateway/infrastructure/cache/l1_test.go`
- 新增：`services/zhicore-gateway/internal/gateway/infrastructure/clients/auth_client.go`
- 新增：`services/zhicore-gateway/internal/gateway/infrastructure/clients/auth_client_test.go`
- 新增：`services/zhicore-gateway/api/http/auth_middleware_test.go`

**验收清单：**
- [ ] JWT 校验签名、`kid`、`iss`、`aud`、`exp`、token type、`jti`、session version、principal version。
- [ ] `jti` blacklist 命中、session revoked、session version 落后、principal version 落后、账号 disabled / banned 都必须 deny，并映射到已登记错误码。
- [ ] 成功认证后注入 `X-Account-Id`、`X-User-Id`、`X-User-Name`、`X-User-Roles`、`X-Session-Id`、`X-Session-Version`、`X-Principal-Version`。
- [ ] Redis miss、blacklist cache miss、principal/session version 落后或账号状态缺失时才调用 `ValidateAccessState`。
- [ ] Auth fallback 设置短超时、singleflight、并发上限和熔断。
- [ ] 错误码区分缺登录 `2006`、缺角色 `2007`、token 状态不可确认 `2016`、下游不可用 `1004`。

- [ ] **步骤 1：写 middleware / cache / JWT 失败测试**
- [ ] **步骤 2：实现 verifier、cache、Auth client 和 middleware**
- [ ] **步骤 3：运行认证测试**

运行：`cd services/zhicore-gateway && go test ./api/http ./internal/gateway/infrastructure/... -count=1`

预期：通过。

## 任务 5：诊断 endpoint、readiness 和系统验证

**测试立场：** TDD - 诊断权限、health 和黑盒路由属于 R3/R4。

**文件：**
- 新增：`services/zhicore-gateway/api/http/diagnostics_handlers.go`
- 新增：`services/zhicore-gateway/api/http/health_handlers.go`
- 新增：`services/zhicore-gateway/api/http/diagnostics_handler_test.go`
- 新增：`services/zhicore-gateway/internal/gateway/runtime/health_test.go`
- 新增：`tests/system/http/gateway_routing_test.go`

**验收清单：**
- [ ] `GET /api/v1/gateway/health`、`GET /api/v1/gateway/routes`、`GET /api/v1/gateway/auth/diagnostics` 只允许管理员 / 内部访问。
- [ ] 内部访问机制必须可测试：只接受 mTLS 标记的可信反代 header 或内网 allowlist 二选一；测试覆盖伪造 header 被拒绝、管理员 token 允许、普通用户拒绝。
- [ ] `/health/live` 不查依赖。
- [ ] `/health/ready` 输出依赖矩阵：路由配置和必填 upstream baseURL 是硬依赖；Redis、Auth client、每个下游 health 按 route 风险和配置标记 ready / degraded / failed。
- [ ] system test 至少覆盖 CORS preflight、公开白名单、admin strip-prefix、file route/header、websocket info 路由、protected route 401。
- [ ] 诊断输出不得泄露 secret、raw token、完整下游 URL userinfo。

- [ ] **步骤 1：写诊断和 system 失败测试**
- [ ] **步骤 2：实现 health、diagnostics 和 system fixture**
- [ ] **步骤 3：运行验证**

运行：`cd services/zhicore-gateway && go test ./... -count=1 && cd ../.. && go test ./tests/system/http -run TestGatewayRouting -count=1`

预期：通过。

## 集成验证

- [ ] 运行 `cd libs/contracts && go test ./clients/auth -count=1`。
- [ ] 运行 `cd services/zhicore-gateway && go test ./... -count=1`。
- [ ] 运行 `make test-size`。
- [ ] 运行 `bash scripts/check-structure.sh`。
- [ ] 完整执行本计划或触达共享 contract / runtime / 系统测试后，交付前运行 `make check`。
