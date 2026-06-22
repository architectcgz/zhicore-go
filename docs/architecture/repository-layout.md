# 仓库目录布局

本文件记录 `zhicore-go` 的仓库目录、服务目录模板和 `api` / `internal` 的边界。

## 基本原则

- `services/<service>` 是独立可构建、可测试、可部署的服务单元。
- 每个服务拥有自己的 `go.mod`，仓库根目录只使用 `go.work` 串联模块。
- 仓库根目录不放 `cmd/`、`internal/` 或 `go.mod`。
- `services/<service>/api/http` 是服务的 HTTP 入站层，放在服务根目录下，不放进 `internal/<domain>`。
- `services/<service>/internal/<domain>` 是服务私有业务代码和技术实现，其他服务不得导入。
- 服务之间通过外部 HTTP API、`libs/contracts/clients` 或 `libs/contracts/events` 交互，不通过共享 repository、共享数据库模型或互相导入 `internal` 交互。

## 服务目录骨架

```text
services/<service>/
├── go.mod
├── cmd/
│   └── server/
├── api/
│   └── http/
├── internal/
│   └── <domain>/
├── migrations/
└── configs/
```

说明：

- `<service>` 使用部署服务名，例如 `zhicore-content`。
- `<domain>` 使用服务内业务名，例如 `content`、`user`、`comment`。
- `api/http` 可以导入本服务的 `internal/<domain>/application`，但不能直接访问数据库、Redis、RabbitMQ 或外部 SDK。
- `api/http` 包名可以使用 `httpapi`，避免和标准库 `net/http` 的 `http` 包名混淆。
- 如果某个服务未来确实包含多个业务域，可以在 `internal/` 下增加多个 `<domain>`；不要提前为所有服务引入多域结构。

## 服务实现展开

服务进入实际迁移实现时，`internal/<domain>` 按以下结构展开：

```text
internal/<domain>/
├── application/
├── domain/
├── ports/
├── infrastructure/
│   ├── postgres/
│   ├── redis/
│   ├── rabbitmq/
│   ├── mongo/
│   ├── es/
│   └── clients/
└── runtime/
    └── module.go
```

`internal/<domain>/runtime/module.go` 是服务内部组装入口，负责把 infrastructure、application 和 `api/http` 连接起来。

## 顶层目录

```text
zhicore-go/
├── services/
├── libs/
│   ├── contracts/
│   │   ├── clients/
│   │   └── events/
│   └── kit/
├── docs/
├── deploy/
├── scripts/
└── tests/
```

目录职责：

- `services/`：所有可部署服务。
- `libs/contracts/clients`：provider 拥有的同步调用 contract。
- `libs/contracts/events`：provider 拥有的事件 payload contract。
- `libs/kit`：小而稳定的跨服务技术原语，不放业务规则。
- `docs/`：长期架构、契约、迁移和 review 事实源。
- `deploy/`：Docker、Kubernetes 等部署资产。
- `scripts/`：仓库级稳定命令入口，例如结构检查、测试规模检查和本地维护命令。
- `tests/`：跨服务架构检查、黑盒 HTTP 场景、运行时测试和测试夹具。

## `api` 和 `internal` 的边界

`api/http` 代表服务对外 HTTP 边界。这里可以放：

- 路由注册。
- handler。
- 请求 DTO。
- 响应 DTO。
- HTTP 参数校验。
- 认证上下文到 application input 的映射。
- 兼容当前前端的 path、method、query、body 和响应封装。

`api/http` 不放：

- SQL、ORM row、数据库事务细节。
- Redis key 拼接和缓存策略。
- RabbitMQ publish / consume 细节。
- 领域状态机和业务规则。
- 调用其他服务的 fallback、重试、熔断策略。

`internal/<domain>` 代表服务私有实现。这里放：

- `domain`：实体、值对象、领域规则、领域事件和领域错误。
- `application`：use case、事务编排、权限上下文、幂等和端口调用。
- `ports`：application 需要的 consumer-side interface。
- `infrastructure`：PostgreSQL、Redis、RabbitMQ、MongoDB、Elasticsearch、对象存储和同步 HTTP client 实现。
- `runtime`：服务内部依赖组装。

## 脚手架演进规则

- 新增服务时必须先创建服务目录骨架，再补 `go.mod`、README 和 migration 占位。
- 迁移某个服务的业务实现前，再补齐该服务的 `internal/<domain>` 分层目录。
- 目录变更必须同步更新 `scripts/check-structure.sh`。
- 如果目录变更影响服务内依赖方向，必须同步更新 `docs/architecture/go-service-design.md`。

## 脚本和机械检查分层

`scripts/` 是稳定入口层，不是所有检查逻辑的堆放点。

规则：

- `scripts/check-structure.sh` 只检查仓库结构、固定入口和必备路径，不承载测试规模、源码扫描、契约语义或运行时规则的具体实现。
- 每类机械检查应有按职责命名的独立入口，例如 `scripts/check-test-size.py`；`make check` 负责组合这些入口。
- 简单路径存在性检查可以继续用 shell；涉及递归扫描、解析、聚合错误、白名单、差异模式或后续扩展的检查优先使用 Python。
- 新增检查脚本时同步更新 `Makefile`、`AGENTS.md` 和相关事实源文档；如果检查守住某个文档规则，文档要写明对应命令。
- 不为“少一个文件”把多个不相关防线塞进同一个脚本；检查边界应按被守护的规则 owner 拆分。
