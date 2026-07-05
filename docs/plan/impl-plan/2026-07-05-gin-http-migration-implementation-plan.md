# Gin HTTP 入站层迁移计划

> **给 agentic workers：** 本计划用于把已有 Go HTTP handler 从标准库 `http.ServeMux` 迁移到 Gin。Gin 只能停留在 `services/<service>/api/http` 和 runtime 挂载层；不得把 `*gin.Context` 传入 application、domain、ports 或 infrastructure。

**目标：** 所有已有 HTTP handler 模块统一使用 `github.com/gin-gonic/gin v1.12.0` 注册路由，为后续限流、恢复、日志、认证投影和观测 middleware 预留统一入口。

**非目标：**

- 不为只有 `api/http` 占位目录、尚无 handler 的服务提前添加未使用 Gin 依赖。
- 不改变 HTTP path、method、envelope、错误码、字段语义和 application command/query。
- 不引入 Gin binding tag 作为 contract 事实源；字段级 contract 仍归 `services/<service>/api/http/endpoints/*.md` 和 handler test。

## 范围

当前已有 HTTP handler 的服务：

- `services/zhicore-auth`
- `services/zhicore-user`
- `services/zhicore-file`
- `services/zhicore-comment`

占位但暂无 handler 的服务保持不变；后续新增 handler 时按 Gin 模板实现。

## 任务 1：Gin 依赖和架构约束

- [x] 为 4 个服务模块添加 `github.com/gin-gonic/gin v1.12.0`。
- [x] 更新 `docs/architecture/go-service-design.md`，固定 Gin 只属于 HTTP 入站层，application 只接收 `context.Context` 和显式 DTO。

验证：

```bash
rg -n "gin.Context|github.com/gin-gonic/gin" services libs
```

## 任务 2：迁移 Auth / User handler

- [x] Auth `api/http` 使用 `gin.Engine` / `gin.RouterGroup` 注册路由，保持 `NewHandler(...) http.Handler` 对外形态。
- [x] User `api/http` 使用 Gin 注册路由，保持头像 URL 降级和关系路由语义。
- [x] 跑 Auth / User handler 测试。

验证：

```bash
cd services/zhicore-auth && go test ./api/http
cd services/zhicore-user && go test ./api/http
```

## 任务 3：迁移 File handler

- [x] File `api/http` 使用 Gin 注册路由，保留 multipart 限制、临时文件清理、`MaxBytesReader` 和现有错误映射。
- [x] 跑 File handler 测试。

验证：

```bash
cd services/zhicore-file && go test ./api/http
```

## 任务 4：迁移 Comment handler

- [x] Comment `api/http` 使用 Gin 注册已有路由。
- [x] 新增业务路由沿用同一 Gin wrapper 接入，不把 `*gin.Context` 传入 application。
- [x] Comment 计划中的 handler 步骤在业务测试通过后同步勾选。

验证：

```bash
cd services/zhicore-comment && go test ./api/http
```

## 任务 5：集成验证

- [x] 运行受影响服务 HTTP 测试。
- [x] 运行结构检查。
- [x] 检查 `*gin.Context` 未越过 `api/http`。

验证：

```bash
bash scripts/check-structure.sh
rg -n "gin.Context|\\*gin.Context" services libs
```

## 任务 6：去除 Gin 迁移中间状态

- [x] Auth / User / Comment runtime 不再用 `http.NewServeMux` 聚合健康检查和业务 handler，统一由 `*gin.Engine` 承载。
- [x] Auth / Comment / File handler 不再保留 `ginHTTPHandler`、`SetPathValue` 和 `PathValue` 参数桥接，路由参数直接由 Gin 读取。
- [x] `NewHandler(...)` 对已有 HTTP handler 服务直接返回 `*gin.Engine`，runtime 的 `Module.HTTPHandler` 也收窄为 `*gin.Engine`。
- [x] Runtime 不再额外暴露 `LiveHandler` / `ReadyHandler` 字段，健康检查只通过同一个 Gin engine 注册。

验证：

```bash
cd services/zhicore-auth && go test ./api/http ./internal/auth/runtime
cd services/zhicore-user && go test ./api/http ./internal/user/runtime
cd services/zhicore-comment && go test ./api/http ./internal/comment/runtime
cd services/zhicore-file && go test ./api/http
rg -n "ginHTTPHandler|SetPathValue|PathValue\\(|http\\.NewServeMux|root\\.Handle\\(|http\\.Handle" services -g '*.go'
```
