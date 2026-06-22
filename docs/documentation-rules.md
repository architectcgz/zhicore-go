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
允许内容：仓库目录结构、服务目录模板、`api/http` 与 `internal` 边界、脚手架演进规则
禁止内容：服务内业务规则、单个服务实现计划、迁移过程临时日志
编辑前阅读：Go 服务目录、`scripts/check-structure.sh`、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/architecture/services/`
类型：当前架构事实
负责人：ZhiCore Go 服务架构
是否入口：是
允许内容：各服务职责、API 族、数据归属、事件、跨服务依赖、Go 落点、迁移风险和下一步实现切片
禁止内容：字段级 HTTP schema、完整 SQL migration、单次任务临时日志、未核对来源的“已实现”结论
编辑前阅读：`docs/architecture/service-boundaries.md`、`docs/architecture/repository-layout.md`、`docs/architecture/go-service-design.md`、对应 Java controller/schema/contract、本文档
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
允许内容：Go 服务内分层、依赖方向、运行时依赖映射、migration、缓存、事件、事务和 API 兼容规则
禁止内容：单个服务的完整实现计划、临时迁移记录、未验证性能结论
编辑前阅读：`docs/architecture/repository-layout.md`、`docs/architecture/service-boundaries.md`、`docs/architecture/id-strategy.md`、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/migration/java-design-migration.md`
类型：迁移计划和映射
负责人：ZhiCore Java 到 Go 迁移
是否入口：否
允许内容：Java 设计事实来源、保留/改写/废弃决策、逐服务迁移分析、迁移风险和后续切片
禁止内容：Go 服务已实现结论、未核对源码的猜测、单次任务临时日志
编辑前阅读：`../zhicore-microservice` Java 源码和文档、`docs/architecture/service-boundaries.md`、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/contracts/`
类型：当前 contract 治理规则
负责人：ZhiCore Go 跨服务 contract
是否入口：是
允许内容：contract 归属、兼容性规则、版本策略、变更流程、发布约束
禁止内容：服务私有 DTO 细节、临时迁移记录、review 证据
编辑前阅读：`docs/architecture/service-boundaries.md`、受影响的 `libs/contracts/...`、受影响的 `services/<service>/api/http`、本文件
验证命令：`bash scripts/check-structure.sh`

路径：`docs/reviews/`
类型：review 证据
负责人：实现和架构 review 流程
是否入口：是
允许内容：review 发现、review 轮次、验证记录
禁止内容：尚未提升到事实源文档的当前架构事实
编辑前阅读：被 review 的 diff 或 commit、相关代码、本文件
验证命令：路径和链接人工检查；路径变化时运行 `bash scripts/check-structure.sh`

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
