# ZhiCore Go Agent 规则

## 项目概览

- 本仓库是 ZhiCore 后端从 Java 迁移到 Go 的工作区。
- 当前 Java 实现仍保留在 `../zhicore-microservice`，Java 代码只作为接口和行为事实源；迁移目标不规划 Java/Go 运行时并存。
- 迁移必须按服务增量推进。除非用户明确要求批量迁移，否则不要一次重写多个服务。

## 常用命令

- `make check`：本地统一交付门禁，运行脚手架检查、测试文件规模检查和所有 Go 模块测试。
- `make test`：在每个 Go workspace 模块内运行 `go test ./...`。
- `make test-size`：全量运行 `scripts/check-test-size.py`，检查 `*_test.go` 文件规模；局部模式见 `docs/architecture/testing.md`。
- `bash scripts/check-structure.sh`：检查服务入口、模块目录、文档入口和 agent 入口是否齐全。
- `bash scripts/check-commit-message.sh <message-file>`：按 `harness/policies/commit-message.json` 检查提交信息。
- `bash scripts/install-githooks.sh`：安装本仓库版本化 Git hooks，当前只启用 `commit-msg` 检查。

当前还没有 CI。CI 建立前以本地质量门禁为准；提交信息由可安装的 `commit-msg` hook 检查，汇报脚手架或代码改动完成前按 `docs/reviews/quality-gates.md` 选择并记录验证命令。

## 架构边界

- `services/<service>` 是独立可部署、可测试、可构建的服务单元。
- 每个服务拥有自己的 `go.mod`；不要添加根应用模块。
- 修改仓库目录布局、服务目录模板、`api/http` / `internal` 落点、脚本入口或机械检查分层前，先读 `docs/architecture/repository-layout.md`。
- `services/<service>/cmd/server` 只放进程入口和运行时装配。
- `services/<service>/api/http` 放 HTTP 入站层和外部 API 兼容代码。
- `services/<service>/internal` 是服务私有代码，其他服务不得导入。
- `libs/kit` 只放小而稳定的跨服务技术原语，不放服务特定业务规则。
- `libs/contracts/events` 放跨服务事件 payload 契约。
- `libs/contracts/clients` 放服务间同步调用的 typed client 契约。
- 修改跨服务数据归属、同步调用、facade 路由或 contract 放置前，先读 `docs/architecture/service-boundaries.md`。
- 修改单个服务职责、API 族、数据归属、事件、依赖或迁移风险前，先读 `docs/architecture/services/README.md` 和对应服务文档。
- 修改服务内分层、运行时依赖、数据库列命名、Go 内部命名、显式 mapper/tag、缓存、RabbitMQ 事件或事务边界前，先读 `docs/architecture/go-service-design.md`。
- 修改 schema migration、`golang-migrate` 命令、migration 文件命名、GORM schema 边界或数据修复规则前，先读 `docs/architecture/migrations.md`。
- 修改测试策略、测试目录归属、测试分层、验证命令或 test-first 要求前，先读 `docs/architecture/testing.md`。
- 修改服务配置、环境变量、配置模板、`libs/kit/config`、密钥处理或配置校验前，先读 `docs/architecture/configuration.md`。
- 修改启动流程、构造函数外部副作用、context 传播、健康检查、优雅停机、HTTP server timeout、下游 client timeout、重试、熔断、幂等、goroutine / worker / consumer 停机或运行期完成标准前，先读 `docs/architecture/runtime-operations.md`；涉及配置时同时读 `docs/architecture/configuration.md`。
- 修改日志、metrics、trace、`requestId` / `traceId`、operation 命名、脱敏、上报边界或 `libs/kit/observability` 前，先读 `docs/architecture/observability.md`；涉及错误处置时同时读 `docs/architecture/error-handling.md`。
- 修改认证、授权、JWT、身份 header、角色、资源权限、Admin 审计、上传安全、外部 URL、敏感输入或 `libs/kit/auth` 前，先读 `docs/architecture/security.md`；涉及密钥时同时读 `docs/architecture/configuration.md`，涉及日志脱敏时同时读 `docs/architecture/observability.md`。
- 修改内部主键、外部公开 ID、业务编号或发号服务定位前，先读 `docs/architecture/id-strategy.md`。
- 修改同步 client contract、事件 payload 或对外 API schema 前，先读 `docs/contracts/README.md`。
- 修改 HTTP path、method、header、响应 envelope、版本化或服务级 HTTP schema 前，先读 `docs/contracts/http.md` 和 `docs/contracts/http-schema-template.md`。
- 修改对外错误响应、公开错误码、HTTP status 映射或字段级校验错误前，先读 `docs/contracts/errors.md` 和 `docs/contracts/error-codes.md`。
- 修改 Go 服务内部错误分层、底层错误翻译、application 错误映射或错误处置规则前，先读 `docs/architecture/error-handling.md`；涉及日志、trace 或上报字段时同时读 `docs/architecture/observability.md`。
- 修改 contract 中的时间、ID、枚举、空值、数字、布尔或 JSON 字段命名前，先读 `docs/contracts/data-types.md`；涉及 ID 策略时同时读 `docs/architecture/id-strategy.md`。
- 修改分页、排序、过滤或 cursor 语义前，先读 `docs/contracts/pagination.md`。
- 修改 RabbitMQ 事件 contract、事件 envelope、routing key、outbox 或幂等规则前，先读 `docs/contracts/events.md`。
- 修改 `Makefile`、检查脚本、本地质量门禁、CI / Git hook、验证命令选择或交付前校验组合前，先读 `docs/reviews/quality-gates.md`；涉及测试策略时同时读 `docs/architecture/testing.md`。
- 修改 review 流程、完成标准、验证证据、finding 分级或技术债登记规则前，先读 `docs/reviews/README.md`、`docs/reviews/done-definition.md` 和 `docs/reviews/quality-gates.md`。
- 修改提交信息格式、commit-msg hook、`harness/policies/commit-message.json`、`scripts/check-commit-message.sh` 或 `scripts/install-githooks.sh` 前，先读 `docs/reviews/commit-message.md`。
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
- 交付完成门槛和 review 触发条件见 `docs/reviews/done-definition.md`。
- 本地质量门禁、验证命令选择和未来 CI 最低要求见 `docs/reviews/quality-gates.md`。
- 提交信息格式、commit-msg hook 和机械检查策略见 `docs/reviews/commit-message.md`。
- 未解决技术债放在 `docs/todos/debt/`。

## 提交规则

- 提交前必须先走全局 `committing-changes` skill，并叠加本仓库 `docs/reviews/commit-message.md`。
- 提交信息必须使用“标题 + 正文”两段结构；标题使用英文 type 和中文描述，例如 `docs(配置): 确立环境变量规范`。
- 普通提交正文至少两行有效内容，说明改动点、原因、影响或验证中的关键信息。
- 提交信息检查策略由 `harness/policies/commit-message.json` 维护；安装 hook 后由 `.githooks/commit-msg` 自动调用。

## 测试规则

- 本项目不强制所有代码改动采用严格 TDD；测试要求按 `docs/architecture/testing.md` 的风险分级执行。
- 所有改动都必须有验证证据；行为变更必须有测试或明确的手动验证方式。
- Bugfix、contract、权限、分页、事务、幂等、并发、worker / consumer 和 migration 属于高风险面，应优先先补失败用例或回归测试。
- 服务内 `*_test.go` 用于验证服务本地 handler、service、repository、worker、adapter 等行为。
- `libs/*` 下的测试只验证共享 contract 或 kit 原语。
- `tests/architecture` 用于源码级架构边界检查。
- `tests/system/http` 用于黑盒 HTTP 场景。
- `tests/runtime` 用于需要真实服务、容器、端口或外部依赖的测试。
- `tests/testkit` 用于可复用的黑盒测试 fixture 和断言。
- 测试文件按行为、endpoint、use case、repository query 或 worker 场景拆分；不要把多个不相关场景堆进一个超大 `*_test.go`。
- 测试失败时先查 owner、contract、分层或实现语义，不要通过放宽断言、fixture 或 mock 迁就错误实现。
- 测试是行为规格和回归保护。只有在信号重复、过时、过度耦合实现，或被迁移到更合适归属处时，才删除或合并。

改动代码或测试后，先运行最窄相关的 `go test` 命令；当脚手架或共享边界发生变化时，交付前再按 `docs/reviews/quality-gates.md` 运行本地质量门禁。
