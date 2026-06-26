# 文档规则

## 目的

本文件定义 `zhicore-go` 的文档归属、放置位置、路径登记和验证规则。

## 核心原则

- 文档是长期项目记忆，不是当前任务草稿。
- 每个文档只承担一个主要角色：当前事实、计划、review 证据、运维指南、外部参考或未解决工作。
- 入口文档负责路由读者，不重复完整事实源内容。
- 项目文档正文默认使用中文；代码标识、API 字段、协议名、命令、路径、错误文本和外部专有名词保持原文。
- 只有用户明确要求，或外部规范、上游模板、协议文档必须使用英文时，才为对应文档正文使用英文。
- 文档变更必须和代码、契约、脚本、测试、架构边界、迁移状态保持同步。

## 避免循环引用

- 本文件是文档规则事实源。
- `docs/README.md` 是文档导航索引。
- 项目 `AGENTS.md` 可以路由到这两个文件，但不要重复完整规则。

## 编辑前阅读

创建、移动、删除或编辑文档前：

1. 先读本文件，除非当前任务正在创建本文件。
2. 再读 `docs/README.md` 或最近的父级索引。
3. 读取被修改事实的当前来源，例如 Java 源码、Go 代码、contract、配置、测试或运维文档。
4. 新增、移动、重命名或删除路径时，搜索现有引用。

## 已登记路径

路径：`docs/README.md`
类型：导航索引
负责人：仓库文档
是否入口：是
允许内容：当前文档地图、阅读顺序、路径路由
禁止内容：长篇实现说明
编辑前阅读：本文件
验证命令：`bash scripts/check-structure.sh`

路径：`CONTEXT-MAP.md`
类型：限界上下文术语入口
负责人：仓库文档
是否入口：是
允许内容：各服务/上下文的 `CONTEXT.md` 路由和上下文关系
禁止内容：实现方案、架构决策正文、临时讨论记录
编辑前阅读：对应上下文 `CONTEXT.md`、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/migration/`
类型：迁移计划和映射
负责人：ZhiCore Java 到 Go 迁移
是否入口：是
允许内容：服务映射、迁移顺序、发布说明、兼容性说明
禁止内容：未验证的“某服务已迁移”结论
编辑前阅读：`../zhicore-microservice` Java 源码、Go 服务落点、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/architecture/`
类型：当前架构事实
负责人：ZhiCore Go 服务架构
是否入口：是
允许内容：仓库目录布局、服务边界、数据归属、依赖方向、contract 放置规则、长期技术约束
禁止内容：临时任务记录、未评审实现计划、review 证据
编辑前阅读：`../zhicore-microservice` Java 源码、Go 服务模块、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/architecture/repository-layout.md`
类型：当前架构事实
负责人：ZhiCore Go 服务架构
是否入口：否
允许内容：仓库目录结构、服务目录模板、`api/http` 与 `internal` 边界、脚本入口和机械检查分层、脚手架演进规则
禁止内容：服务内业务规则、单个服务实现计划、迁移过程临时日志
编辑前阅读：Go 服务目录、`scripts/check-structure.sh`、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/architecture/services/`
类型：当前架构事实
负责人：ZhiCore Go 服务架构
是否入口：是
允许内容：各服务职责、API 族、数据归属、事件、跨服务依赖、Go 落点、迁移风险、下一步实现切片和服务设计图索引
禁止内容：字段级 HTTP schema、完整 SQL migration、单次任务临时日志、未核对来源的“已实现”结论
编辑前阅读：`docs/architecture/service-boundaries.md`、`docs/architecture/repository-layout.md`、`docs/architecture/go-service-design.md`、对应 Java controller/schema/contract、本文档
验证命令：`bash scripts/check-structure.sh`

路径：`docs/architecture/services/<service>/README.md`
类型：当前架构事实
负责人：对应 Go 目标服务
是否入口：是
允许内容：单个服务的模块级设计、职责边界、DDD 目标设计、数据归属、事件、跨服务依赖、迁移风险和下一步实现切片
禁止内容：其他服务的完整设计、字段级 HTTP schema、完整 SQL migration、单次任务临时日志、未核对来源的“已实现”结论
编辑前阅读：`docs/architecture/services/README.md`、`docs/architecture/service-boundaries.md`、`docs/architecture/repository-layout.md`、`docs/architecture/go-service-design.md`、对应 Java controller/schema/contract、本文档
验证命令：`bash scripts/check-structure.sh`

路径：`docs/architecture/services/<service>/*.md`
类型：当前架构事实的服务专题文档
负责人：对应 Go 目标服务
是否入口：否
允许内容：对应服务 README 拆分出的长期专题事实，例如 `CONTEXT.md` 术语表、领域模型、应用流程、正文存储、数据事件契约、实现切片和迁移风险
禁止内容：其他服务完整设计、字段级 HTTP schema、完整 SQL migration、单次任务临时日志、未登记的新顶层文档入口
编辑前阅读：对应服务 `README.md`、`docs/architecture/services/README.md`、本文档；涉及跨服务边界时同时读 `docs/architecture/service-boundaries.md`
验证命令：`bash scripts/check-structure.sh`

路径：`docs/architecture/services/<service>/adr/`
类型：服务内架构决策记录
负责人：对应 Go 目标服务
是否入口：否
允许内容：该服务内难以逆转、容易被后续读者质疑、且有真实取舍的架构决策及其原因
禁止内容：跨服务全局决策、普通实现细节、临时讨论记录、没有取舍的事实说明、完整设计文档替代品
编辑前阅读：对应服务 `README.md`、对应服务 `CONTEXT.md`、相关专题文档、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/architecture/services/<service>/decision-log/`
类型：服务内设计讨论复盘记录
负责人：对应 Go 目标服务
是否入口：否
允许内容：从设计压测、grill-with-docs 或重要评审中重建的问题、结论、原因、修正和链接到 ADR / 专题文档的复盘记录
禁止内容：未沉淀结论的聊天流水账、临时任务执行日志、与当前设计文档冲突且未标注 superseded 的旧结论
编辑前阅读：对应服务 `README.md`、对应服务 `CONTEXT.md`、相关 ADR 和专题文档、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/architecture/services/<service>/service-design.<service>.png`
类型：当前架构事实的辅助渲染图
负责人：对应 Go 目标服务
是否入口：否
允许内容：对应服务设计图的导出图片
禁止内容：没有 reviewable 源文件的唯一事实源、其他服务图片、临时截图
编辑前阅读：`docs/architecture/services/README.md`、`docs/architecture/services/_overview/service-design.drawio`、对应服务文档、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/architecture/services/<service>/service-detail.drawio`
类型：当前架构事实的 reviewable 图表源
负责人：对应 Go 目标服务
是否入口：否
允许内容：对应服务的详细设计图源文件，覆盖入口、owner、guard、数据、副作用、运行依赖和禁止路径
禁止内容：跨服务总览、临时截图、未在服务文档中确认的服务/队列/缓存/中间件
编辑前阅读：`docs/architecture/services/README.md`、对应服务文档、本文件
验证命令：`bash scripts/check-structure.sh`；导出图片时用 draw.io CLI 验证 XML 可读

路径：`docs/architecture/services/<service>/service-detail.png`
类型：当前架构事实的辅助渲染图
负责人：对应 Go 目标服务
是否入口：否
允许内容：对应服务详细设计图的导出图片
禁止内容：没有 `service-detail.drawio` 源文件的唯一事实源、其他服务图片、临时截图
编辑前阅读：`docs/architecture/services/<service>/service-detail.drawio`、对应服务文档、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/architecture/services/_overview/`
类型：跨服务设计图源和总览导出
负责人：ZhiCore Go 服务架构
是否入口：否
允许内容：跨服务总览图、服务设计图集源文件和总览导出图片
禁止内容：单个服务正文、临时截图、没有服务归属的零散图片
编辑前阅读：`docs/architecture/services/README.md`、`docs/architecture/service-boundaries.md`、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/architecture/id-strategy.md`
类型：当前架构事实
负责人：ZhiCore Go 服务架构
是否入口：否
允许内容：内部主键、外部公开 ID、业务编号、发号服务定位和 ID 兼容约束
禁止内容：具体服务私有字段清单、临时算法实验、未验证性能结论
编辑前阅读：`docs/architecture/service-boundaries.md`、受影响的服务 schema、受影响的 contract、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/architecture/go-service-design.md`
类型：当前架构事实
负责人：ZhiCore Go 服务架构
是否入口：否
允许内容：Go 服务内分层、依赖方向、运行时依赖映射、命名和映射归属、migration、缓存、事件、事务和 API 兼容规则
禁止内容：单个服务的完整实现计划、临时迁移记录、未验证性能结论
编辑前阅读：`docs/architecture/repository-layout.md`、`docs/architecture/service-boundaries.md`、`docs/architecture/id-strategy.md`、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/architecture/configuration.md`
类型：当前架构事实
负责人：ZhiCore Go 配置治理
是否入口：否
允许内容：服务配置、环境变量命名、配置来源优先级、必填校验、配置模板、密钥脱敏、`libs/kit/config` 边界和配置验证要求
禁止内容：真实密钥、生产连接串、具体环境部署值、完整 Helm values、单个服务私有配置清单
编辑前阅读：`docs/architecture/runtime-operations.md`、`docs/architecture/go-service-design.md`、`libs/kit/config`、受影响服务 `configs/`、本文件
验证命令：`bash scripts/check-structure.sh`；涉及配置加载代码时运行最窄相关 `go test`

路径：`docs/architecture/observability.md`
类型：当前架构事实
负责人：ZhiCore Go 可观测性治理
是否入口：否
允许内容：结构化日志字段、日志级别、requestId / traceId 传播、operation 命名、metrics 标签、脱敏规则、上报边界和 `libs/kit/observability` 职责
禁止内容：真实日志样本中的敏感数据、具体日志平台部署配置、完整告警策略、服务私有业务审计日志 schema
编辑前阅读：`docs/architecture/error-handling.md`、`docs/architecture/runtime-operations.md`、`docs/architecture/configuration.md`、`libs/kit/observability`、本文件
验证命令：`bash scripts/check-structure.sh`；涉及可观测性代码时运行最窄相关 `go test`

路径：`docs/architecture/security.md`
类型：当前架构事实
负责人：ZhiCore Go 安全与权限治理
是否入口：否
允许内容：认证、授权、JWT、身份传播、角色和资源权限、Admin 审计、上传安全、外部 URL、敏感输入、`libs/kit/auth` 边界和安全测试要求
禁止内容：真实密钥、真实 token、生产账号、漏洞利用细节、完整审计日志 schema、服务私有权限清单
编辑前阅读：`docs/architecture/service-boundaries.md`、`docs/architecture/configuration.md`、`docs/architecture/observability.md`、`docs/contracts/errors.md`、`libs/kit/auth`、本文件
验证命令：`bash scripts/check-structure.sh`；涉及认证、授权或上传安全代码时运行最窄相关 `go test`

路径：`docs/architecture/migrations.md`
类型：当前架构事实
负责人：ZhiCore Go schema migration
是否入口：否
允许内容：`golang-migrate` 工具选择、migration 文件命名、up/down、事务、seed/reference data、GORM 边界、review 清单和执行命令
禁止内容：单个服务完整 SQL、一次性数据修复日志、生产数据库连接串、未核对服务归属的表设计
编辑前阅读：`docs/architecture/service-boundaries.md`、`docs/architecture/go-service-design.md`、`docs/architecture/id-strategy.md`、受影响服务 schema 来源、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/architecture/testing.md`
类型：当前架构事实
负责人：ZhiCore Go 测试策略
是否入口：否
允许内容：风险分级测试策略、test-first 触发条件、测试写法和规模控制、测试目录归属、改动类型测试要求、验证命令选择和不新增测试时的说明规则
禁止内容：单个任务的测试执行记录、具体服务完整测试清单、临时手动验证日志、与当前代码不符的覆盖率结论
编辑前阅读：`tests/README.md`、受影响服务测试、相关 contract / architecture 文档、本文件
验证命令：`make test-size`；路径或索引变化时再运行 `bash scripts/check-structure.sh`

路径：`docs/architecture/error-handling.md`
类型：当前架构事实
负责人：ZhiCore Go 服务架构
是否入口：否
允许内容：Go 服务内部错误分层、错误依赖方向、底层错误翻译、application 错误映射和错误处置分级
禁止内容：对外 HTTP 错误响应 schema、服务公开错误码清单、字段级 API 错误详情
编辑前阅读：`docs/architecture/go-service-design.md`、`docs/architecture/observability.md`、本文件；涉及对外错误响应时再读 `docs/contracts/errors.md`
验证命令：`bash scripts/check-structure.sh`

路径：`docs/architecture/runtime-operations.md`
类型：当前架构事实
负责人：ZhiCore Go 运行期架构
是否入口：否
允许内容：启动流程、构造函数外部副作用、context 传播、优雅停机、健康检查、HTTP server timeout、下游 client timeout、重试、熔断、幂等、goroutine / worker / consumer 停机和运行完成标准
禁止内容：配置命名细则、单个服务的临时部署记录、具体环境密钥、完整 Helm/Kubernetes manifest、一次性排障日志
编辑前阅读：`docs/architecture/go-service-design.md`、`docs/architecture/configuration.md`、`docs/architecture/error-handling.md`、本文件；涉及对外 HTTP contract 时再读 `docs/contracts/http.md`
验证命令：`bash scripts/check-structure.sh`

路径：`docs/migration/java-design-migration.md`
类型：迁移计划和映射
负责人：ZhiCore Java 到 Go 迁移
是否入口：否
允许内容：Java 设计事实来源、保留/改写/废弃决策、逐服务迁移分析、迁移风险和后续切片
禁止内容：Go 服务已实现结论、未核对源码的猜测、单次任务临时日志
编辑前阅读：`../zhicore-microservice` Java 源码和文档、`docs/architecture/service-boundaries.md`、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/migration/service-migration-workflow.md`
类型：迁移流程规则
负责人：ZhiCore Java 到 Go 迁移
是否入口：否
允许内容：单服务或服务内 API 族迁移顺序、Java 事实提取、HTTP contract、schema migration、测试策略、Go 实现顺序、完成标准和交付验证规则
禁止内容：单次任务执行日志、具体服务字段级 schema、完整 SQL migration、正式 review 证据、未核对来源的“已实现”结论
编辑前阅读：`docs/migration/README.md`、`docs/migration/java-design-migration.md`、`docs/architecture/testing.md`、`docs/contracts/http-schema-template.md`、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/contracts/`
类型：当前 contract 治理规则
负责人：ZhiCore Go 跨服务 contract
是否入口：是
允许内容：contract 归属、兼容性规则、版本策略、变更流程、发布约束
禁止内容：服务私有 DTO 细节、具体协议专题规则、临时迁移记录、review 证据
编辑前阅读：`docs/architecture/service-boundaries.md`、受影响的 `libs/contracts/...`、受影响的 `services/<service>/api/http`、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/contracts/http.md`
类型：当前 contract 治理规则
负责人：ZhiCore Go HTTP API contract
是否入口：否
允许内容：HTTP path、method、header、响应 envelope、版本化和服务级 HTTP schema 放置规则
禁止内容：错误码全集、服务字段级 schema、Go 内部 handler 实现、运行时路由配置
编辑前阅读：`docs/contracts/README.md`、对应 Java controller、受影响的 `services/<service>/api/http`、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/contracts/http-schema-template.md`
类型：当前 contract 治理规则
负责人：ZhiCore Go HTTP API contract
是否入口：否
允许内容：服务级 HTTP schema 的文件布局、endpoint 文档模板、字段记录要求、状态标记和提取流程
禁止内容：单个服务的完整字段级 schema、Go handler 实现、一次性提取记录、未核对来源的 endpoint 结论
编辑前阅读：`docs/contracts/http.md`、`docs/contracts/errors.md`、`docs/contracts/data-types.md`、本文件；涉及具体服务时再读对应 Java controller / DTO / 测试
验证命令：`bash scripts/check-structure.sh`

路径：`docs/contracts/errors.md`
类型：当前 contract 治理规则
负责人：ZhiCore Go 错误 contract
是否入口：否
允许内容：对外错误响应、公开错误码、HTTP status 映射、字段级校验错误形态和错误码归属
禁止内容：Go 内部错误分层实现、底层 driver 或 SDK 错误细节、服务私有 sentinel
编辑前阅读：Java `ResultCode` / `ApiResponse`、受影响的服务 HTTP contract、本文件；涉及 Go 内部映射边界时再读 `docs/architecture/error-handling.md`
验证命令：`bash scripts/check-structure.sh`

路径：`docs/contracts/error-codes.md`
类型：当前 contract 治理规则
负责人：ZhiCore Go 错误 contract
是否入口：否
允许内容：Go 对外 `body.code` 的错误码表、错误码范围归属、历史例外和内部错误标识映射
禁止内容：Go 内部错误类型实现、服务私有 sentinel、字段级 HTTP schema、一次性排障记录
编辑前阅读：Java `ResultCode` / `ApiResponse` / `GlobalExceptionHandler`、受影响服务的 Java exception handler、本文件；涉及响应形态时再读 `docs/contracts/errors.md`
验证命令：`bash scripts/check-structure.sh`

路径：`docs/contracts/data-types.md`
类型：当前 contract 治理规则
负责人：ZhiCore Go 通用数据类型 contract
是否入口：否
允许内容：时间、ID、枚举、空值、数字、布尔和 JSON 字段命名的序列化规则
禁止内容：具体服务数据库字段清单、单个 DTO 的完整字段级 schema、ID 算法实验
编辑前阅读：`docs/architecture/id-strategy.md`、受影响的 Java DTO、受影响的 contract、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/contracts/pagination.md`
类型：当前 contract 治理规则
负责人：ZhiCore Go 分页 contract
是否入口：否
允许内容：page/cursor 分页、排序、过滤、cursor 不透明性和返回形态
禁止内容：具体服务查询 SQL、索引设计细节、服务私有 repository filter
编辑前阅读：受影响的 Java controller / DTO、受影响的服务 HTTP contract、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/contracts/events.md`
类型：当前 contract 治理规则
负责人：ZhiCore Go 事件 contract
是否入口：否
允许内容：RabbitMQ exchange、routing key、事件归属、事件 envelope、outbox、幂等和事件兼容性
禁止内容：具体事件 payload 全量字段、consumer 私有处理策略、broker 部署运维细节
编辑前阅读：`docs/architecture/go-service-design.md`、受影响的 `libs/contracts/events/...`、受影响的服务文档、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`services/<service>/api/http/README.md`
类型：当前 HTTP contract schema
负责人：对应 Go 服务
是否入口：是
允许内容：服务级 HTTP schema 索引、Java/Go 来源、服务级公共规则、endpoint 索引和服务级公开错误码子集
禁止内容：Go handler 实现说明、服务内 application 设计、数据库字段清单、临时迁移日志
编辑前阅读：`docs/contracts/http-schema-template.md`、对应服务文档、对应 Java controller / DTO / 测试、本文件
验证命令：最窄相关 `go test`；脚手架或索引变化时运行 `bash scripts/check-structure.sh`

路径：`services/<service>/api/http/endpoints/`
类型：当前 HTTP contract schema
负责人：对应 Go 服务
是否入口：否
允许内容：单个 endpoint 的 path、method、request 字段、response `data` 字段、错误码、权限、分页、测试要求和兼容例外
禁止内容：多个无关 endpoint 混写、Go handler 内部流程、repository 查询细节、未核对来源的字段
编辑前阅读：`docs/contracts/http-schema-template.md`、对应服务 `api/http/README.md`、对应 Java controller / DTO / 测试、本文件
验证命令：最窄相关 `go test`；脚手架或索引变化时运行 `bash scripts/check-structure.sh`

路径：`docs/reviews/`
类型：review 证据
负责人：实现和架构 review 流程
是否入口：是
允许内容：review 分类目录、review 发现、review 轮次、验证记录
禁止内容：尚未提升到事实源文档的当前架构事实、未核对 diff 的泛泛评价
编辑前阅读：`docs/reviews/README.md`、被 review 的 diff 或 commit、相关代码、本文件
验证命令：路径和链接人工检查；路径变化时运行 `bash scripts/check-structure.sh`

路径：`docs/reviews/README.md`
类型：review 流程规则
负责人：实现和架构 review 流程
是否入口：是
允许内容：review 触发条件、记录位置、记录格式、finding 书写规则和事实提升规则
禁止内容：单次 review 发现、具体实现计划、尚未确认的架构事实
编辑前阅读：`docs/reviews/done-definition.md`、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/reviews/done-definition.md`
类型：完成标准
负责人：实现和架构 review 流程
是否入口：否
允许内容：交付完成门槛、正式 review 触发条件、finding 分级、验证证据要求、不可交付状态和技术债登记规则
禁止内容：单次任务验证记录、单个 review 发现、具体服务实现细节
编辑前阅读：`docs/architecture/testing.md`、`docs/reviews/README.md`、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/reviews/commit-message.md`
类型：提交规范
负责人：实现和架构 review 流程
是否入口：否
允许内容：提交信息格式、commit-msg hook、项目提交策略、Task 元数据和提交信息机械检查维护规则
禁止内容：单次提交记录、完整 pre-commit 测试门禁、具体任务 review 发现
编辑前阅读：`docs/reviews/quality-gates.md`、`harness/policies/commit-message.json`、`.githooks/README.md`、本文件
验证命令：`bash scripts/check-structure.sh`；修改策略后运行 `bash scripts/check-commit-message.sh <message-file>`

路径：`docs/reviews/quality-gates.md`
类型：质量门禁规则
负责人：实现和架构 review 流程
是否入口：否
允许内容：本地验证命令职责、命令选择规则、未来 CI / Git hook 最低要求和检查脚本维护规则
禁止内容：提交信息格式细则、单次任务验证记录、具体 CI 平台完整配置、服务私有测试清单、临时排障日志
编辑前阅读：`docs/reviews/README.md`、`docs/reviews/done-definition.md`、`docs/reviews/commit-message.md`、`docs/architecture/testing.md`、`Makefile`、相关检查脚本、本文件
验证命令：`make check`；仅路径索引调整时至少运行 `bash scripts/check-structure.sh`

路径：`docs/todos/debt/`
类型：未解决技术债跟踪
负责人：迁移技术债管理
是否入口：是
允许内容：包含负责人、影响和退出条件的未解决技术债
禁止内容：泛任务 backlog 或临时草稿
编辑前阅读：最近的 debt 索引，以及产生 debt 的源码或 review
验证命令：`bash scripts/check-structure.sh`

## 验证

当文档路径、索引或事实发生变化时：

- 运行 `bash scripts/check-structure.sh`。
- 搜索被重命名、移动或删除路径的陈旧引用。
- 从最近的父级索引检查链接是否仍然成立。
