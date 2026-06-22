# ZhiCore Go Agent 规则

## 项目概览

- 本仓库是 ZhiCore 后端从 Java 迁移到 Go 的工作区。
- 当前 Java 实现仍保留在 `../zhicore-microservice`，Java 代码只作为接口和行为事实源；迁移目标不规划 Java/Go 运行时并存。
- 迁移必须按服务增量推进。除非用户明确要求批量迁移，否则不要一次重写多个服务。

## 常用命令

- `make check`：运行脚手架检查和所有 Go 模块测试。
- `make test`：在每个 Go workspace 模块内运行 `go test ./...`。
- `bash scripts/check-structure.sh`：检查服务入口、模块目录、文档入口和 agent 入口是否齐全。

当前还没有 CI 或 Git hook 强制校验。汇报脚手架或代码改动完成前，必须手动运行 `make check`。

## 架构边界

- `services/<service>` 是独立可部署、可测试、可构建的服务单元。
- 每个服务拥有自己的 `go.mod`；不要添加根应用模块。
- 修改仓库目录布局、服务目录模板、`api/http` 和 `internal` 落点前，先读 `docs/architecture/repository-layout.md`。
- `services/<service>/cmd/server` 只放进程入口和运行时装配。
- `services/<service>/api/http` 放 HTTP 入站层和外部 API 兼容代码。
- `services/<service>/internal` 是服务私有代码，其他服务不得导入。
- `libs/kit` 只放小而稳定的跨服务技术原语，不放服务特定业务规则。
- `libs/contracts/events` 放跨服务事件 payload 契约。
- `libs/contracts/clients` 放服务间同步调用的 typed client 契约。
- 修改跨服务数据归属、同步调用、facade 路由或 contract 放置前，先读 `docs/architecture/service-boundaries.md`。
- 修改服务内分层、运行时依赖、migration、缓存、RabbitMQ 事件或事务边界前，先读 `docs/architecture/go-service-design.md`。
- 修改内部主键、外部公开 ID、业务编号或发号服务定位前，先读 `docs/architecture/id-strategy.md`。
- 修改同步 client contract、事件 payload 或对外 API schema 前，先读 `docs/contracts/README.md`。
- 共享库必须保持朴素、明确。对于不稳定的服务本地代码，优先保留重复，不要过早提升到 `libs`。
- 数据库 schema 演进必须显式、可审查。不要在服务启动路径里添加运行时自动迁移。
- 保留现有 Java 外部 API 形态；前端暂时不修改，当前开发阶段不做灰度，Gateway 只能做路由或环境切换，不能把 API 形态变化传递给前端。

## 服务落点

- `zhicore-gateway` -> `services/zhicore-gateway`
- `zhicore-user` -> `services/zhicore-user`
- `zhicore-content` -> `services/zhicore-content`
- `zhicore-comment` -> `services/zhicore-comment`
- `zhicore-message` -> `services/zhicore-message`
- `zhicore-notification` -> `services/zhicore-notification`
- `zhicore-search` -> `services/zhicore-search`
- `zhicore-ranking` -> `services/zhicore-ranking`
- `zhicore-admin` -> `services/zhicore-admin`
- `zhicore-upload` -> `services/zhicore-upload`
- `zhicore-id-generator` -> `services/zhicore-id-generator`
- `zhicore-ops` -> `services/zhicore-ops`
- Java `zhicore-common`、`zhicore-client`、`zhicore-integration` 映射到 `libs/kit` 和 `libs/contracts`，默认不是可部署的 Go 服务。

## 文档

- 创建、移动或编辑长期文档前，先读 `docs/documentation-rules.md`。
- 使用 `docs/README.md` 作为文档索引。
- 新建或初始化 README、docs、部署说明和 agent 规则时，正文默认使用中文；代码标识、包名、协议字段、命令、路径和错误文本保持原文。
- 只有用户明确要求，或外部规范、上游模板、协议文档必须使用英文时，才为对应文档正文使用英文。
- 迁移计划放在 `docs/migration/`。
- 正式 review 证据放在 `docs/reviews/`。
- 未解决技术债放在 `docs/todos/debt/`。

## 测试规则

- 后端行为变更需要 TDD：先写边界测试，确认按预期失败，再实现。
- 服务内 `*_test.go` 用于验证服务本地 handler、service、repository、worker、adapter 等行为。
- `libs/*` 下的测试只验证共享 contract 或 kit 原语。
- `tests/architecture` 用于源码级架构边界检查。
- `tests/system/http` 用于黑盒 HTTP 场景。
- `tests/runtime` 用于需要真实服务、容器、端口或外部依赖的测试。
- `tests/testkit` 用于可复用的黑盒测试 fixture 和断言。
- TDD 测试是行为规格和回归保护。只有在信号重复、过时、过度耦合实现，或被迁移到更合适归属处时，才删除或合并。

改动代码或测试后，先运行最窄相关的 `go test` 命令；当脚手架或共享边界发生变化时，交付前再运行 `make check`。
