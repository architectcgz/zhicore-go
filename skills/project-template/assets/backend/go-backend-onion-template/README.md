# go-backend-onion-template

基于 `ctf/code/backend` 提炼的 Go 后端架构模板。

目标：

- 保留 CTF 当前后端最稳定的启动方式与组合边界
- 提供可复用的 Onion/Clean Architecture 目录骨架
- 给新项目一个“够像真实项目”的起步结构，而不是空目录

这套模板强调的边界：

- `cmd/` 只放程序入口
- `internal/bootstrap/` 负责进程启动、资源初始化、优雅关闭
- `internal/app/` 负责 HTTP server、router 与 runtime composition
- `internal/module/<domain>/` 负责领域模块本身
- `internal/infrastructure/` 放数据库、缓存、日志等外部适配
- `internal/shared/` 放跨模块共享内核
- `migrations/`、`configs/`、`tests/` 与运行时脚本保持独立

测试放置规则：

- 包内 `*_test.go`：模块内业务语义、未导出实现、application service / command / query / repository 的局部契约；需要访问包私有符号时默认留在源码旁边。
- `internal/testutil/*`：需要接近内部实现、会被多个包内或系统测试复用的测试工具；不要放只服务单个测试文件的一次性封装。
- `tests/architecture`：源码级架构 guardrail，只检查边界、目录和迁移约束，不跑业务语义回归。
- `tests/system/http/<scenario>`：黑盒 HTTP / router 级长场景，只表达请求、角色、状态和响应断言；环境搭建、seed 和通用断言优先复用 testutil / testkit。
- `tests/runtime`：需要 PostgreSQL、runtime agent、容器端口、外部进程或真实 migration 参与的集成测试。
- `tests/testkit`：跨 `tests/*` 复用的场景 builder、fixture、assert helper 和测试数据工厂；不访问未导出实现。
- TDD 写出的测试默认是行为规格和回归护栏，不因为对应功能已经实现就删除；只在行为信号重复、实现细节锁定、迁移 guard 到期，或目标行为明确废弃时合并或删除。

当前模板现在更接近“可继续直接开工的 starter”：

- `tree.txt`：推荐目录树
- `starter-files/`：起步文件与代码片段
- `manifest.json`：模板元信息与占位变量说明

这次补上的最小闭环：

- `internal/config/`：`viper` 配置加载与基础配置模型
- `configs/config*.yaml`：base/dev/prod 样例
- `internal/infrastructure/{logger,postgres,redis}/`：基础外部适配
- `internal/app/router_runtime.go`：`gin` router 与模块挂载骨架
- `internal/module/__DOMAIN_NAME__/`：包含 entity/domain/contracts/commands/queries/http/runtime/infrastructure 的完整示例模块
- `TxRunner`：保留 command 用例的事务边界样例，避免 service 直接耦合 GORM 事务
- `redis.enabled`：默认关闭 Redis，降低 starter 首次启动门槛

当前边界也要明确：

- 示例模块里的 `Repository` 仍是内存实现，目的是演示分层、依赖方向和 runtime wiring
- 新项目落地时，应优先将该仓储替换为真实 persistence，例如 GORM/Postgres、Redis cache、外部 service adapter 等
- 不建议把这份内存仓储直接保留到生产代码里

生成后建议先做：

- 把 `internal/module/__DOMAIN_NAME__/infrastructure/repository.go` 换成真实持久化实现
- 决定 Redis 是否为启动必需项；模板默认是可选依赖，需要时再打开 `redis.enabled`
- 按项目需要补 `tests/architecture`、`tests/system` 或模块 `testsupport`

关键占位符：

- `__GO_MODULE__`：Go module path，例如 `github.com/acme/example-service`
- `__SERVICE_NAME__`：服务名，例如 `example-service`
- `__DOMAIN_NAME__`：示例模块名，例如 `example`

来源特征：

- 启动入口参考 `ctf/code/backend/cmd/api/main.go`
- bootstrap 参考 `ctf/code/backend/internal/bootstrap/run.go`
- composition root 参考 `ctf/code/backend/internal/app/composition/root.go`
- 模块分层参考 `ctf/code/backend/internal/module/{challenge,practice,runtime}`
