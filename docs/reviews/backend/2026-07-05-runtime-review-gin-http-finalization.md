# Gin HTTP 入站层收口 Review

## Review 对象

- Diff source：当前工作区未提交改动。
- 范围：
  - `services/zhicore-auth/api/http/handler.go`
  - `services/zhicore-auth/internal/auth/runtime/module.go`
  - `services/zhicore-user/api/http/handler.go`
  - `services/zhicore-user/internal/user/runtime/module.go`
  - `services/zhicore-comment/api/http/handler.go`
  - `services/zhicore-comment/internal/comment/runtime/module.go`
  - `services/zhicore-file/api/http/handler.go`
  - 相关 runtime 测试
  - `docs/plan/impl-plan/2026-07-05-gin-http-migration-implementation-plan.md`
- 独立 reviewer：`code-reviewer` subagent `019f30c4-d54d-7581-bd89-3f69c979f688`。

## 分类判断

- 分类：非琐碎 runtime / HTTP 入站层重构。
- 触发原因：一次 diff 跨 Auth、User、Comment、File 多个服务，修改 runtime 健康检查挂载和 HTTP handler 路由框架边界。
- Gate verdict：`pass`。

## Findings

### Blocker

未发现 blocker finding。

### Major

未发现 major finding。

### Minor

- `LiveHandler` / `ReadyHandler` 在 Gin engine 承载健康检查后只剩测试引用，存在第二套无生产 owner 的健康检查挂载面。
  - 状态：已修复。
  - 修复：Auth / User / Comment runtime 只保留 `HTTPHandler *gin.Engine`，健康检查只通过同一个 Gin engine 注册。

### Note

- `http.HandlerFunc` 残留扫描只命中 Comment HTTP client 测试中的 `httptest.NewServer` fake server，不属于服务入站路由中间状态。
- 既有未提交的 `docs/architecture/go-service-design.md` 是本次任务前已存在改动，不纳入本次 review finding。

## Material Findings

无需要阻塞交付的 material finding。

## 验证证据

已执行并通过：

```bash
cd services/zhicore-auth && go test ./api/http ./internal/auth/runtime
cd services/zhicore-user && go test ./api/http ./internal/user/runtime
cd services/zhicore-comment && go test ./api/http ./internal/comment/runtime
cd services/zhicore-file && go test ./api/http
cd services/zhicore-auth && go test ./...
cd services/zhicore-user && go test ./...
cd services/zhicore-comment && go test ./...
cd services/zhicore-file && go test ./...
bash scripts/check-structure.sh
python3 tests/architecture/check_boundaries.py --root .
make check
```

残留扫描：

```bash
rg -n "LiveHandler|ReadyHandler|ginHTTPHandler|SetPathValue|PathValue\\(|http\\.NewServeMux|root\\.Handle\\(|http\\.Handle" services -g '*.go'
```

结果：只剩 `services/zhicore-comment/internal/comment/infrastructure/clients/http_test.go` 中的 `httptest.NewServer(http.HandlerFunc(...))` fake server。

## Required Re-validation

如后续继续修改 runtime 或 handler 路由挂载，至少重新执行：

```bash
cd services/zhicore-auth && go test ./api/http ./internal/auth/runtime
cd services/zhicore-user && go test ./api/http ./internal/user/runtime
cd services/zhicore-comment && go test ./api/http ./internal/comment/runtime
cd services/zhicore-file && go test ./api/http
make check
```

## Residual Risk

- 健康检查 contract 按当前文档只覆盖 `GET /health/live` 和 `GET /health/ready`；Gin 对 `HEAD`、错误 method 和尾斜杠的默认行为未作为本次 contract 变更处理。
- 本次只收口已有 HTTP handler 服务。尚无 handler 的占位服务不提前添加 Gin 依赖，后续新增 HTTP 入站层时按本计划约束实现。

## 技术债状态

未登记新的技术债。review 发现的无 owner 健康检查字段已在本次 touched surface 内收口。
