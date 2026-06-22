# zhicore-go

`zhicore-go` 是 ZhiCore 后端从 Java 迁移到 Go 的微服务工作区。

当前 Java 后端仍在 `../zhicore-microservice`。本仓库用于承接 Go 版本服务，目标是先把每个目标服务的落点、边界、契约和验证方式固定下来，再按服务逐步替换 Java 实现，同时保持现有前端和网关依赖的外部 API 契约。

## 项目结构

- `go.work`：本地 workspace，用于串联所有服务模块和共享库模块。
- `services/zhicore-*`：独立可构建、可测试、可部署的 Go 服务。
- `services/zhicore-*/go.mod`：每个服务独立拥有自己的 Go module。
- `services/zhicore-*/internal`：服务私有的应用、领域、传输和基础设施代码。
- `libs/contracts`：跨服务 client 契约和事件契约。
- `libs/kit`：小型共享技术原语，例如响应封装、认证、配置、可观测性、数据库、缓存、RabbitMQ 客户端封装。
- `deploy/`：Docker 和 Kubernetes 部署资产。
- `docs/migration/`：迁移映射、迁移顺序和分阶段替换说明。

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
