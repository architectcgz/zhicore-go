# 文档索引

本目录是 `zhicore-go` 的文档入口。

## 阅读顺序

1. 先读 `docs/documentation-rules.md`，确认文档归属和放置规则。
2. 再通过本索引找到相关事实源。
3. 修改当前事实前，用 Java 源码、Go 代码、contract、配置、测试或运维记录做核对。

## 当前事实源

- `docs/architecture/`：当前服务边界和数据归属决策。
  - `docs/architecture/repository-layout.md`：仓库目录、服务目录模板和 `api` / `internal` 边界。
  - `docs/architecture/service-boundaries.md`：服务边界、数据归属、依赖方向和 contract 放置规则。
  - `docs/architecture/services/`：各 Go 目标服务的模块级设计、API 族、数据归属、事件和迁移风险。
  - `docs/architecture/go-service-design.md`：Go 服务内分层、运行时依赖映射、命名和映射归属、migration、缓存和事件规则。
  - `docs/architecture/migrations.md`：`golang-migrate`、SQL migration 文件命名、事务、down migration、seed 和 GORM 边界。
  - `docs/architecture/testing.md`：风险分级测试策略、test-first 触发条件、测试写法、规模控制、测试放置和验证命令选择。
  - `docs/architecture/runtime-operations.md`：配置、启动、健康检查、优雅停机、超时、重试、熔断、幂等和运行完成标准。
  - `docs/architecture/error-handling.md`：Go 服务内部错误分层和对外错误映射边界。
  - `docs/architecture/id-strategy.md`：内部主键、外部公开 ID、业务编号和 `zhicore-id-generator` 定位。
- `docs/contracts/`：跨服务 contract 归属、兼容性、版本和变更流程。
  - `docs/contracts/http.md`：HTTP path、method、header、envelope、版本化和服务级 HTTP schema 放置规则。
  - `docs/contracts/http-schema-template.md`：`services/<service>/api/http/` 下服务级 schema 和 endpoint 文档格式。
  - `docs/contracts/errors.md`：对外错误响应、公开错误码、HTTP status 映射和校验错误形态。
  - `docs/contracts/error-codes.md`：Go 对外 `body.code` 的项目级错误码表。
  - `docs/contracts/data-types.md`：时间、ID、枚举、空值、数字、布尔和 JSON 字段命名规则。
  - `docs/contracts/pagination.md`：page/cursor 分页、排序、过滤和返回形态。
  - `docs/contracts/events.md`：RabbitMQ 事件 contract、envelope、outbox 和兼容性规则。
- `docs/migration/`：Java 到 Go 的服务迁移映射、迁移顺序和发布说明。
  - `docs/migration/java-design-migration.md`：Java 侧设计的保留、改写、废弃和服务迁移盘点。

## 流程和历史

- `docs/reviews/`：review 证据和发现。
- `docs/todos/debt/`：迁移过程中不能丢失的未解决技术债。

## 部署说明

部署资产放在 `deploy/`：

- `deploy/docker/`
- `deploy/k8s/`
