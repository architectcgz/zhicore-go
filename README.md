# zhicore-go

`zhicore-go` 是 ZhiCore 后端从 Java 迁移到 Go 的微服务工作区。

当前 Java 后端仍在 `../zhicore-microservice`，但只作为接口和行为事实源。迁移目标不规划 Java/Go 运行时并存，本仓库先固定每个目标服务的落点、边界、契约和验证方式，再按服务逐步用 Go 实现替换 Java 实现，同时保持现有前端和网关依赖的外部 API 契约。

## 项目结构

- `go.work`：本地 workspace，用于串联所有服务模块和共享库模块。
- `services/zhicore-*`：独立可构建、可测试、可部署的 Go 服务。
- `services/zhicore-*/go.mod`：每个服务独立拥有自己的 Go module。
- `services/zhicore-*/api/http`：服务的 HTTP 入站层和外部 API 兼容代码。
- `services/zhicore-*/internal`：服务私有的应用、领域、端口、运行时组装和基础设施代码。
- `libs/contracts`：跨服务 client 契约和事件契约。
- `libs/kit`：小型共享技术原语，例如响应封装、认证、配置、可观测性、数据库、缓存、RabbitMQ 客户端封装。
- `deploy/`：Docker 和 Kubernetes 部署资产。
- `docs/migration/`：迁移映射、迁移顺序和分阶段替换说明。
- `docs/migration/java-design-migration.md`：Java 侧设计迁移盘点。
- `docs/architecture/repository-layout.md`：仓库目录、服务目录模板和 `api` / `internal` 边界。
- `docs/architecture/go-service-design.md`：Go 服务分层、运行时依赖、migration、缓存和事件规则。
- `docs/architecture/id-strategy.md`：内部主键、外部公开 ID、业务编号和发号服务定位。

## 目标服务

- `zhicore-gateway`
- `zhicore-user`
- `zhicore-content`
- `zhicore-comment`
- `zhicore-message`
- `zhicore-notification`
- `zhicore-search`
- `zhicore-ranking`
- `zhicore-admin`
- `zhicore-upload`
- `zhicore-id-generator`
- `zhicore-ops`

## 常用命令

```bash
make check
make test
```

`make check` 会检查项目脚手架并运行 Go 测试。当前大部分服务目录仍是迁移占位，后续实现应按服务逐个推进。
