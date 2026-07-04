# 文档规则

## 目的

本文件定义 `zhicore-go` 的文档归属、放置位置、路径登记和验证规则。

## 核心原则

- 文档是长期项目记忆，不是当前任务草稿。
- 每个文档只承担一个主要角色：当前事实、计划、review 证据、运维指南、外部参考或未解决工作。
- 入口文档负责路由读者，不重复完整事实源内容。
- 项目文档正文默认使用中文；代码标识、API 字段、协议名、命令、路径、错误文本和外部专有名词保持原文。
- 只有用户明确要求，或外部规范、上游模板、协议文档必须使用英文时，才为对应文档正文使用英文。
- 文档变更必须和代码、契约、脚本、测试、架构边界、服务交付状态保持同步。

## 避免循环引用

- 本文件是文档规则事实源。
- `docs/README.md` 是文档导航索引。
- 项目 `AGENTS.md` 可以路由到这两个文件，但不要重复完整规则。

## 编辑前阅读

创建、移动、删除或编辑文档前：

1. 先读本文件，除非当前任务正在创建本文件。
2. 再读 `docs/README.md` 或最近的父级索引。
3. 读取被修改事实的当前来源，优先核对 Go 设计、Go 代码、contract、配置、测试或运维文档；需要确认既有行为时再参考 Java 源码。
4. 新增、移动、重命名或删除路径时，搜索现有引用。

## 已登记路径

| 路径 | 类型 | Owner | 入口 | 允许内容 | 禁止内容 | 编辑前阅读 | 验证命令 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `docs/README.md` | 导航索引 | 仓库文档 | 是 | 当前文档地图、阅读顺序、路径路由 | 长篇实现说明 | 本文件 | `bash scripts/check-structure.sh` |
| `CONTEXT-MAP.md` | 限界上下文术语入口 | 仓库文档 | 是 | 各服务/上下文的 `CONTEXT.md` 路由和上下文关系 | 实现方案、架构决策正文、临时讨论记录 | 对应上下文 `CONTEXT.md`、本文件 | `bash scripts/check-structure.sh` |
| `docs/migration/` | 既有实现参考和服务替换计划 | ZhiCore Go 服务交付 | 是 | 既有服务映射、服务替换顺序、发布说明、已发布 contract 约束 | 长期 Java 设计盘点、未验证的“某服务已完成”结论 | Go 服务落点、本文件；需要核对既有行为时再读 `../zhicore-microservice` | `bash scripts/check-structure.sh` |
| `docs/architecture/` | 当前架构事实 | ZhiCore Go 服务架构 | 是 | 仓库目录布局、服务边界、数据归属、依赖方向、contract 放置规则、长期技术约束 | 临时任务记录、未评审实现计划、review 证据 | Go 服务模块、本文件；需要核对既有行为时再读 `../zhicore-microservice` | `bash scripts/check-structure.sh` |
| `docs/architecture/repository-layout.md` | 当前架构事实 | ZhiCore Go 服务架构 | 否 | 仓库目录结构、服务目录模板、`api/http` 与 `internal` 边界、脚本入口和机械检查分层、脚手架演进规则 | 服务内业务规则、单个服务实现计划、迁移过程临时日志 | Go 服务目录、`scripts/check-structure.sh`、本文件 | `bash scripts/check-structure.sh` |
| `docs/architecture/services/` | 当前架构事实 | ZhiCore Go 服务架构 | 是 | 各服务职责、API 族总览、数据归属总览、跨服务依赖、Go 落点、实现风险、下一步实现切片和服务设计图索引 | 字段级 HTTP schema、完整模块内部 service/domain/ports 设计、完整 SQL migration、单次任务临时日志、未核对来源的“已实现”结论 | `docs/architecture/service-boundaries.md`、`docs/architecture/repository-layout.md`、`docs/architecture/go-service-design.md`、对应 Go contract、本文档；需要核对既有行为时再读 Java controller/schema/contract | `bash scripts/check-structure.sh` |
| `docs/architecture/services/<service>/README.md` | 当前架构事实 | 对应 Go 目标服务 | 是 | 单个服务的职责边界、API 族总览、数据归属总览、事件总览、跨服务依赖、实现风险、下一步实现切片和模块设计链接 | 其他服务的完整设计、字段级 HTTP schema、完整模块内部 service/domain/ports 设计、完整 SQL migration、单次任务临时日志、未核对来源的“已实现”结论 | `docs/architecture/services/README.md`、`docs/architecture/service-boundaries.md`、`docs/architecture/repository-layout.md`、`docs/architecture/go-service-design.md`、对应 Go contract、本文档；需要核对既有行为时再读 Java controller/schema/contract | `bash scripts/check-structure.sh` |
| `docs/architecture/services/<service>/*.md` | 当前架构事实的既有服务专题文档 | 对应 Go 目标服务 | 否 | 已存在的服务专题事实；后续新增或重写 API / service / domain / ports 设计时优先落到 `docs/architecture/module/<module>/` | 其他服务完整设计、字段级 HTTP schema、完整 SQL migration、单次任务临时日志、未登记的新顶层文档入口 | 对应服务 `README.md`、`docs/architecture/services/README.md`、`docs/architecture/module/README.md`、本文档；涉及跨服务边界时同时读 `docs/architecture/service-boundaries.md` | `bash scripts/check-structure.sh` |
| `docs/architecture/module/` | 当前架构事实 | ZhiCore Go 模块架构 | 是 | 模块内部 API 背后设计、application service、domain、ports、数据和事件文档索引 | 字段级 HTTP schema、服务级总览、跨服务全局图、临时任务记录 | `docs/architecture/services/README.md`、`docs/architecture/service-boundaries.md`、本文档 | `bash scripts/check-structure.sh` |
| `docs/architecture/module/<module>/README.md` | 当前架构事实 | 对应 Go 模块 | 是 | 模块职责、边界、API family、实现切片、关联服务和当前状态 | 字段级 HTTP schema、完整 SQL migration、单次任务临时日志 | `docs/architecture/module/README.md`、对应服务 `docs/architecture/services/<service>/README.md`、本文档 | `bash scripts/check-structure.sh` |
| `docs/architecture/module/<module>/*.md` | 当前架构事实的模块专题文档 | 对应 Go 模块 | 否 | `api.md`、`service.md`、`domain.md`、`ports.md`、`data-events.md` 等模块内部长期设计 | 多个无关模块混写、字段级 HTTP schema、handler 实现细节、repository 查询细节 | 对应模块 `README.md`、`docs/architecture/module/README.md`、本文档；涉及跨服务边界时同时读 `docs/architecture/service-boundaries.md` | `bash scripts/check-structure.sh` |
| `docs/architecture/services/<service>/adr/` | 服务内架构决策记录 | 对应 Go 目标服务 | 否 | 该服务内难以逆转、容易被后续读者质疑、且有真实取舍的架构决策及其原因 | 跨服务全局决策、普通实现细节、临时讨论记录、没有取舍的事实说明、完整设计文档替代品 | 对应服务 `README.md`、对应服务 `CONTEXT.md`、相关专题文档、本文件 | `bash scripts/check-structure.sh` |
| `docs/architecture/services/<service>/decision-log/` | 服务内设计讨论复盘记录 | 对应 Go 目标服务 | 否 | 从设计压测、grill-with-docs 或重要评审中重建的问题、结论、原因、修正和链接到 ADR / 专题文档的复盘记录 | 未沉淀结论的聊天流水账、临时任务执行日志、与当前设计文档冲突且未标注 superseded 的旧结论 | 对应服务 `README.md`、对应服务 `CONTEXT.md`、相关 ADR 和专题文档、本文件 | `bash scripts/check-structure.sh` |
| `docs/architecture/services/<service>/service-design.<service>.png` | 当前架构事实的辅助渲染图 | 对应 Go 目标服务 | 否 | 对应服务设计图的导出图片 | 没有 reviewable 源文件的唯一事实源、其他服务图片、临时截图 | `docs/architecture/services/README.md`、`docs/architecture/services/_overview/service-design.drawio`、对应服务文档、本文件 | `bash scripts/check-structure.sh` |
| `docs/architecture/services/<service>/service-detail.drawio` | 当前架构事实的 reviewable 图表源 | 对应 Go 目标服务 | 否 | 对应服务的详细设计图源文件，覆盖入口、owner、guard、数据、副作用、运行依赖和禁止路径 | 跨服务总览、临时截图、未在服务文档中确认的服务/队列/缓存/中间件 | `docs/architecture/services/README.md`、对应服务文档、本文件 | `bash scripts/check-structure.sh`；导出图片时用 draw.io CLI 验证 XML 可读 |
| `docs/architecture/services/<service>/service-detail.png` | 当前架构事实的辅助渲染图 | 对应 Go 目标服务 | 否 | 对应服务详细设计图的导出图片 | 没有 `service-detail.drawio` 源文件的唯一事实源、其他服务图片、临时截图 | `docs/architecture/services/<service>/service-detail.drawio`、对应服务文档、本文件 | `bash scripts/check-structure.sh` |
| `docs/architecture/services/_overview/` | 跨服务设计图源和总览导出 | ZhiCore Go 服务架构 | 否 | 跨服务总览图、服务设计图集源文件和总览导出图片 | 单个服务正文、临时截图、没有服务归属的零散图片 | `docs/architecture/services/README.md`、`docs/architecture/service-boundaries.md`、本文件 | `bash scripts/check-structure.sh` |
| `docs/architecture/id-strategy.md` | 当前架构事实 | ZhiCore Go 服务架构 | 否 | 内部主键、外部公开 ID、业务编号、发号服务定位和已发布 ID contract 约束 | 具体服务私有字段清单、临时算法实验、未验证性能结论 | `docs/architecture/service-boundaries.md`、受影响的服务 schema、受影响的 contract、本文件 | `bash scripts/check-structure.sh` |
| `docs/architecture/go-service-design.md` | 当前架构事实 | ZhiCore Go 服务架构 | 否 | Go 服务内分层、依赖方向、运行时依赖映射、命名和映射归属、migration、缓存、事件、事务和 API contract 规则 | 单个服务的完整实现计划、临时服务替换记录、未验证性能结论 | `docs/architecture/repository-layout.md`、`docs/architecture/service-boundaries.md`、`docs/architecture/id-strategy.md`、本文件 | `bash scripts/check-structure.sh` |
| `docs/architecture/configuration.md` | 当前架构事实 | ZhiCore Go 配置治理 | 否 | 服务配置、环境变量命名、配置来源优先级、必填校验、配置模板、密钥脱敏、`libs/kit/config` 边界和配置验证要求 | 真实密钥、生产连接串、具体环境部署值、完整 Helm values、单个服务私有配置清单 | `docs/architecture/runtime-operations.md`、`docs/architecture/go-service-design.md`、`libs/kit/config`、受影响服务 `configs/`、本文件 | `bash scripts/check-structure.sh`；涉及配置加载代码时运行最窄相关 `go test` |
| `docs/architecture/observability.md` | 当前架构事实 | ZhiCore Go 可观测性治理 | 否 | 结构化日志字段、日志级别、requestId / traceId 传播、operation 命名、metrics 标签、脱敏规则、上报边界和 `libs/kit/observability` 职责 | 真实日志样本中的敏感数据、具体日志平台部署配置、完整告警策略、服务私有业务审计日志 schema | `docs/architecture/error-handling.md`、`docs/architecture/runtime-operations.md`、`docs/architecture/configuration.md`、`libs/kit/observability`、本文件 | `bash scripts/check-structure.sh`；涉及可观测性代码时运行最窄相关 `go test` |
| `docs/architecture/security.md` | 当前架构事实 | ZhiCore Go 安全与权限治理 | 否 | 认证、授权、JWT、身份传播、角色和资源权限、Admin 审计、上传安全、外部 URL、敏感输入、`libs/kit/auth` 边界和安全测试要求 | 真实密钥、真实 token、生产账号、漏洞利用细节、完整审计日志 schema、服务私有权限清单 | `docs/architecture/service-boundaries.md`、`docs/architecture/configuration.md`、`docs/architecture/observability.md`、`docs/contracts/errors.md`、`libs/kit/auth`、本文件 | `bash scripts/check-structure.sh`；涉及认证、授权或上传安全代码时运行最窄相关 `go test` |
| `docs/architecture/migrations.md` | 当前架构事实 | ZhiCore Go schema migration | 否 | `golang-migrate` 工具选择、migration 文件命名、up/down、事务、seed/reference data、GORM 边界、review 清单和执行命令 | 单个服务完整 SQL、一次性数据修复日志、生产数据库连接串、未核对服务归属的表设计 | `docs/architecture/service-boundaries.md`、`docs/architecture/go-service-design.md`、`docs/architecture/id-strategy.md`、受影响服务 schema 来源、本文件 | `bash scripts/check-structure.sh` |
| `docs/architecture/testing.md` | 当前架构事实 | ZhiCore Go 测试策略 | 否 | 风险分级测试策略、test-first 触发条件、测试写法和规模控制、测试目录归属、改动类型测试要求、验证命令选择和不新增测试时的说明规则 | 单个任务的测试执行记录、具体服务完整测试清单、临时手动验证日志、与当前代码不符的覆盖率结论 | `tests/README.md`、受影响服务测试、相关 contract / architecture 文档、本文件 | `make test-size`；路径或索引变化时再运行 `bash scripts/check-structure.sh` |
| `docs/architecture/error-handling.md` | 当前架构事实 | ZhiCore Go 服务架构 | 否 | Go 服务内部错误分层、错误依赖方向、底层错误翻译、application 错误映射和错误处置分级 | 对外 HTTP 错误响应 schema、服务公开错误码清单、字段级 API 错误详情 | `docs/architecture/go-service-design.md`、`docs/architecture/observability.md`、本文件；涉及对外错误响应时再读 `docs/contracts/errors.md` | `bash scripts/check-structure.sh` |
| `docs/architecture/runtime-operations.md` | 当前架构事实 | ZhiCore Go 运行期架构 | 否 | 启动流程、构造函数外部副作用、context 传播、优雅停机、健康检查、HTTP server timeout、下游 client timeout、重试、熔断、幂等、goroutine / worker / consumer 停机和运行完成标准 | 配置命名细则、单个服务的临时部署记录、具体环境密钥、完整 Helm/Kubernetes manifest、一次性排障日志 | `docs/architecture/go-service-design.md`、`docs/architecture/configuration.md`、`docs/architecture/error-handling.md`、本文件；涉及对外 HTTP contract 时再读 `docs/contracts/http.md` | `bash scripts/check-structure.sh` |
| `docs/migration/service-migration-workflow.md` | 服务替换流程规则 | ZhiCore Go 服务交付 | 否 | 单服务或服务内 API 族实现顺序、既有事实核对、HTTP contract、schema migration、测试策略、Go 实现顺序、完成标准和交付验证规则 | 单次任务执行日志、具体服务字段级 schema、完整 SQL migration、正式 review 证据、未核对来源的“已实现”结论、长期 Java 设计盘点 | `docs/migration/README.md`、`docs/architecture/testing.md`、`docs/contracts/http-schema-template.md`、本文件；需要核对既有行为时再读 `../zhicore-microservice` | `bash scripts/check-structure.sh` |
| `docs/plan/` | 实施计划目录 | ZhiCore Go 服务交付 | 是 | 跨服务、跨仓或结构性任务的计划目录 | 当前架构事实、字段级 HTTP schema、review 证据、完成结论 | `docs/README.md`、本文件 | `bash scripts/check-structure.sh` |
| `docs/plan/README.md` | 实施计划索引 | ZhiCore Go 服务交付 | 是 | 实施计划目录说明、当前计划索引和状态 | 计划正文、当前架构事实、review 证据 | `docs/README.md`、本文件 | `bash scripts/check-structure.sh` |
| `docs/plan/impl-plan/` | 正式实现计划 | ZhiCore Go 服务交付 | 否 | 需要按任务执行、review 和验证的正式实现计划，包含文件落点、步骤、测试和风险 | 已完成事实源、一次性聊天记录、未验证的“已完成”结论 | `docs/plan/README.md` 或最近父级索引、相关架构 / contract 文档、本文件 | `bash scripts/check-structure.sh` |
| `docs/contracts/` | 当前 contract 治理规则 | ZhiCore Go 跨服务 contract | 是 | contract 归属、兼容性规则、版本策略、变更流程、发布约束 | 服务私有 DTO 细节、具体协议专题规则、临时服务替换记录、review 证据 | `docs/architecture/service-boundaries.md`、受影响的 `libs/contracts/...`、受影响的 `services/<service>/api/http`、本文件 | `bash scripts/check-structure.sh` |
| `docs/contracts/http.md` | 当前 contract 治理规则 | ZhiCore Go HTTP API contract | 否 | HTTP path、method、header、响应 envelope、版本化和服务级 HTTP schema 放置规则 | 错误码全集、服务字段级 schema、Go 内部 handler 实现、运行时路由配置 | `docs/contracts/README.md`、受影响的 `services/<service>/api/http`、本文件；需要核对既有行为时再读对应 Java controller | `bash scripts/check-structure.sh` |
| `docs/contracts/api-design-documentation.md` | API 设计文档规范 | ZhiCore Go HTTP API contract | 否 | API 背后设计、HTTP contract、endpoint 文档和实现追踪的分层结构、状态判定和落地优先级 | 单个服务字段级 schema、Go handler 实现、一次性执行日志、未核对来源的 endpoint 结论 | `docs/contracts/README.md`、`docs/contracts/http.md`、`docs/contracts/http-schema-template.md`、本文件；涉及具体服务时先读对应服务设计和服务级 HTTP schema | `bash scripts/check-structure.sh` |
| `docs/contracts/http-schema-template.md` | 当前 contract 治理规则 | ZhiCore Go HTTP API contract | 否 | 服务级 HTTP schema 的文件布局、endpoint 文档模板、字段记录要求、状态标记和提取流程 | 单个服务的完整字段级 schema、Go handler 实现、一次性提取记录、未核对来源的 endpoint 结论 | `docs/contracts/http.md`、`docs/contracts/api-design-documentation.md`、`docs/contracts/errors.md`、`docs/contracts/data-types.md`、本文件；涉及具体服务时先读服务级 Go schema，需要核对既有行为时再读对应 Java controller / DTO / 测试 | `bash scripts/check-structure.sh` |
| `docs/contracts/errors.md` | 当前 contract 治理规则 | ZhiCore Go 错误 contract | 否 | 对外错误响应、公开错误码、HTTP status 映射、字段级校验错误形态和错误码归属 | Go 内部错误分层实现、底层 driver 或 SDK 错误细节、服务私有 sentinel | 受影响的服务 HTTP contract、本文件；涉及 Go 内部映射边界时再读 `docs/architecture/error-handling.md`；需要核对既有错误行为时再参考 Java `ResultCode` / `ApiResponse` | `bash scripts/check-structure.sh` |
| `docs/contracts/error-codes.md` | 当前 contract 治理规则 | ZhiCore Go 错误 contract | 否 | Go 对外 `body.code` 的错误码表、错误码范围归属、历史例外和内部错误标识映射 | Go 内部错误类型实现、服务私有 sentinel、字段级 HTTP schema、一次性排障记录 | 受影响服务的 HTTP contract、本文件；涉及响应形态时再读 `docs/contracts/errors.md`；需要核对既有错误行为时再参考 Java `ResultCode` / `ApiResponse` / `GlobalExceptionHandler` | `bash scripts/check-structure.sh` |
| `docs/contracts/data-types.md` | 当前 contract 治理规则 | ZhiCore Go 通用数据类型 contract | 否 | 时间、ID、枚举、空值、数字、布尔和 JSON 字段命名的序列化规则 | 具体服务数据库字段清单、单个 DTO 的完整字段级 schema、ID 算法实验 | `docs/architecture/id-strategy.md`、受影响的 Go contract、本文件；需要核对既有行为时再读受影响的 Java DTO | `bash scripts/check-structure.sh` |
| `docs/contracts/pagination.md` | 当前 contract 治理规则 | ZhiCore Go 分页 contract | 否 | page/cursor 分页、排序、过滤、cursor 不透明性和返回形态 | 具体服务查询 SQL、索引设计细节、服务私有 repository filter | 受影响的服务 HTTP contract、本文件；需要核对既有行为时再读受影响的 Java controller / DTO | `bash scripts/check-structure.sh` |
| `docs/contracts/events.md` | 当前 contract 治理规则 | ZhiCore Go 事件 contract | 否 | RabbitMQ exchange、routing key、事件归属、事件 envelope、outbox、幂等和事件兼容性 | 具体事件 payload 全量字段、consumer 私有处理策略、broker 部署运维细节 | `docs/architecture/go-service-design.md`、受影响的 `libs/contracts/events/...`、受影响的服务文档、本文件 | `bash scripts/check-structure.sh` |
| `services/<service>/api/http/README.md` | 当前 HTTP contract schema | 对应 Go 服务 | 是 | 服务级 HTTP schema 索引、Go 来源、必要的既有行为参考来源、服务级公共规则、endpoint 索引和服务级公开错误码子集 | Go handler 实现说明、服务内 application 设计、数据库字段清单、临时服务替换日志 | `docs/contracts/http-schema-template.md`、对应服务文档、本文件；需要核对既有行为时再读对应 Java controller / DTO / 测试 | 最窄相关 `go test`；脚手架或索引变化时运行 `bash scripts/check-structure.sh` |
| `services/<service>/api/http/endpoints/` | 当前 HTTP contract schema | 对应 Go 服务 | 否 | 单个 endpoint 的 path、method、request 字段、response `data` 字段、错误码、权限、分页、测试要求和兼容例外 | 多个无关 endpoint 混写、Go handler 内部流程、repository 查询细节、未核对来源的字段 | `docs/contracts/http-schema-template.md`、对应服务 `api/http/README.md`、本文件；需要核对既有行为时再读对应 Java controller / DTO / 测试 | 最窄相关 `go test`；脚手架或索引变化时运行 `bash scripts/check-structure.sh` |
| `docs/reviews/` | review 证据 | 实现和架构 review 流程 | 是 | review 分类目录、review 发现、review 轮次、验证记录 | 尚未提升到事实源文档的当前架构事实、未核对 diff 的泛泛评价 | `docs/reviews/README.md`、被 review 的 diff 或 commit、相关代码、本文件 | 路径和链接人工检查；路径变化时运行 `bash scripts/check-structure.sh` |
| `docs/reviews/README.md` | review 流程规则 | 实现和架构 review 流程 | 是 | review 触发条件、记录位置、记录格式、finding 书写规则和事实提升规则 | 单次 review 发现、具体实现计划、尚未确认的架构事实 | `docs/reviews/done-definition.md`、本文件 | `bash scripts/check-structure.sh` |
| `docs/reviews/done-definition.md` | 完成标准 | 实现和架构 review 流程 | 否 | 交付完成门槛、正式 review 触发条件、finding 分级、验证证据要求、不可交付状态和技术债登记规则 | 单次任务验证记录、单个 review 发现、具体服务实现细节 | `docs/architecture/testing.md`、`docs/reviews/README.md`、本文件 | `bash scripts/check-structure.sh` |
| `docs/reviews/commit-message.md` | 提交规范 | 实现和架构 review 流程 | 否 | 提交信息格式、commit-msg hook、项目提交策略、Task 元数据和提交信息机械检查维护规则 | 单次提交记录、完整 pre-commit 测试门禁、具体任务 review 发现 | `docs/reviews/quality-gates.md`、`harness/policies/commit-message.json`、`.githooks/README.md`、本文件 | `bash scripts/check-structure.sh`；修改策略后运行 `bash scripts/check-commit-message.sh <message-file>` |
| `docs/reviews/quality-gates.md` | 质量门禁规则 | 实现和架构 review 流程 | 否 | 本地验证命令职责、命令选择规则、未来 CI / Git hook 最低要求和检查脚本维护规则 | 提交信息格式细则、单次任务验证记录、具体 CI 平台完整配置、服务私有测试清单、临时排障日志 | `docs/reviews/README.md`、`docs/reviews/done-definition.md`、`docs/reviews/commit-message.md`、`docs/architecture/testing.md`、`Makefile`、相关检查脚本、本文件 | `make check`；仅路径索引调整时至少运行 `bash scripts/check-structure.sh` |
| `docs/todos/debt/` | 未解决技术债跟踪 | 迁移技术债管理 | 是 | 包含负责人、影响和退出条件的未解决技术债 | 泛任务 backlog 或临时草稿 | 最近的 debt 索引，以及产生 debt 的源码或 review | `bash scripts/check-structure.sh` |

## 验证

当文档路径、索引或事实发生变化时：

- 运行 `bash scripts/check-structure.sh`。
- 搜索被重命名、移动或删除路径的陈旧引用。
- 从最近的父级索引检查链接是否仍然成立。
