# Generic Draw.io Business Diagram Prompt

Use this prompt when asking an agent to generate reviewable diagrams.net / draw.io XML from a plan, architecture doc, migration design, business workflow, or review fix.

````text
你是资深架构师和技术图表设计者。请基于本地仓库与给定文档，生成一份 diagrams.net / draw.io 可导入的 XML 图，用图梳理当前任务的业务关系、owner 边界、主流程、决策分支、数据副作用、阻断路径和非目标。

## 输入上下文

- 仓库根目录：`<repo_root>`
- 任务名称：`<task_name>`
- 任务类型：`<implementation-plan | architecture-design | api-contract | migration | review-fix | operations-flow>`
- 服务范围：`<service_name_or_service_dir>`（必须具体到 service / module；只有跨服务关系需要解释时才补充总览）
- 核心用例：`<api_or_job_and_use_case>`（例如 `POST /comments -> CreateComment -> comment.created -> Notification consumer`；详细图必须围绕具名用例展开）
- 项目产物规则：`<project_defined_artifact_rule | 未定义>`（项目未定义时，默认建议 `docs/architecture/services/<service>/`；跨服务总览默认建议 `docs/architecture/services/_overview/`）
- 核心文档：
  - `<primary_plan_or_design_doc>`
  - `<related_architecture_doc_1>`
  - `<related_contract_doc_1>`
- 关键代码路径：
  - `<code_path_1>`
  - `<code_path_2>`
- 需要重点解释的问题：
  - `<question_or_risk_1>`
  - `<question_or_risk_2>`
- 目标读者：`<reviewer | implementer | maintainer | operator>`

如果某些输入不存在，不要虚构。请在图中省略对应页面或节点，并用 callout 标注“未在输入中确认”。

## 读取顺序

生成图前必须按顺序读取：

1. `<repo_root>/AGENTS.md`，了解项目规则、文档归属和验证要求。
2. 项目文档规范中关于架构图、路径引用、计划文档或契约文档的章节。
3. 项目的 service / module 目录结构，确认 `<service_name_or_service_dir>` 的 owner、入口、依赖、数据表、任务和边界。
4. “输入上下文”里的核心文档。
5. 与该 service 和 `<api_or_job_and_use_case>` 直接相关的 handler / consumer、application use case、repository / adapter、migration、table、event contract、测试或脚本路径。
6. 当前 skill 的 `references/drawio-styles.md`，按其中的节点和边样式生成 XML。

事实来源优先级：

1. 当前代码、测试、配置、脚本和契约。
2. 当前架构文档或当前实施计划。
3. review 记录只能作为历史证据，不能覆盖当前事实。
4. draft / exploratory 文档只能作为候选方案，图中必须标注 `Draft` 或“待落地”。

如果图展示的是目标设计而非已落地事实，必须在标题或 Legend 中标注 `target design / planned`，不要画成当前架构事实。

## 粒度与产物位置

默认生成 service 级架构图，不要只画仓库、平台或系统总览。除非用户明确只要 overview，否则必须先选定一个或多个具体 service，并分别生成图。

- 每个 service 至少有独立页面或独立 XML，详细图默认画一个具名业务用例流程，页面内容围绕该用例的入口、owner、数据、副作用、运行链路和验证门禁展开。
- 详细图不得退化为静态模块结构图；只画 `handler -> application -> domain -> repository -> DB` 不合格。
- 跨 service 图只能解释依赖、调用方向、数据所有权和边界，不替代 service 内部架构图。
- 产物放置由项目文档规则负责；生成前必须读取项目的文档归属、架构图、服务文档或 docs 索引规则。
- 如果项目已有服务文档或图表目录，必须服从项目规则，不要用本 skill 覆盖项目归属。
- 如果项目没有产物规则，默认建议 `docs/architecture/services/<service>/` 放服务图相关产物，`docs/architecture/services/_overview/` 放跨服务总览；这只是缺省建议，不是跨项目硬规则。
- 如果只输出 XML 而不落文件，在回答或交付说明中给出建议落点即可。

## 用例流程硬门禁

service detail diagram 必须画具体业务用例的端到端流程，而不是概括 service 分层。必须满足：

- 标题、Legend 或 callout 明确写出具名核心用例，例如 `POST /comments 发布评论流程`。
- 从真实入口开始：API / route / handler / consumer / job / command，不从抽象的“业务层”开始。
- 展开 handler / consumer 的解析、鉴权、幂等、参数校验或上下文构造。
- 展开 application use case 的编排动作、读取 facts、guard、分支判断和失败返回。
- 写清 repository / adapter 调用和具体 DB 表、缓存键、对象存储、外部 API 或消息主题。
- 写清事务边界、写入顺序、锁 / 并发 guard、回滚或不写入保证。
- 如果存在异步链路，必须画出 outbox / dispatcher / broker / consumer / 幂等 / 最终副作用。
- 成功路径必须落到用户或系统可见结果；失败路径必须落到 blocker、错误码、重试、补偿或 no-op。

禁止反例：

```text
Handler -> Application -> Domain -> Repository -> DB
```

上面的图缺少具名用例、具体判断、表、事件、事务、副作用和可见结果，不能作为 service detail diagram 交付。

合格示例（评论发布）：

```text
POST /comments
-> CommentHandler 解析 postId / parentId / content / media
-> CreateComment application 校验 Content / User facts
-> parentId 是否为空？
-> 根评论：分配 floor + insert comments / comment_stats + outbox comment.created
-> 回复：加载 root / parent + 校验同文章/未删除 + root.reply_count +1 + outbox comment.created
-> Outbox dispatcher -> RabbitMQ -> Notification consumer
-> parentId != null ? 通知被回复者 : 根评论通知策略
```

## 输出要求

只输出一个可导入 draw.io 的 XML 文档：

- 根节点必须是 `<mxfile>`。
- XML 必须是未压缩的 draw.io XML，方便 review。
- 不要输出 Mermaid、PlantUML、PNG、SVG 或解释性长文。
- 图中所有文字使用中文为主；代码路径、API、表名、字段名、状态码、错误码、blocker code 保持原文。
- 每个页面左上角必须有简短标题，标题说明 service 和具名用例 / 视角，例如“comment-service 发布评论流程”“billing-service 支付确认决策流程”“user-service 注册数据与副作用”。
- 每个页面左上或右上必须有 Legend，说明颜色、线型和特殊符号。
- Legend 或标题必须标明当前图的 service scope；跨 service overview 必须明确标注 `overview`，不能伪装成具体 service 内部架构。

建议至少生成 3 个页面；如果输入内容不足，可以减少，但不能少于 2 个页面：

1. `Use Case Workflow`：具名核心用例的入口、handler / consumer、application 编排、判断节点、写入、返回和 blocker。
2. `Async Events And Side Effects`：outbox、事件、消息主题、dispatcher、consumer、幂等、通知、搜索索引、计数器等最终副作用。仅在任务涉及异步链路时生成。
3. `Data And Transaction Boundaries`：该用例读写的数据表、状态、不变量、事务边界、锁、回滚或补偿。
4. `Service Business Relationship Map`：该 service 内的业务对象、角色、owner、基数关系、禁止关系。仅作为边界补充，不替代用例流程。
5. `Validation / Guardrails`：测试、脚本、hook、review gate、人工检查。仅在验证策略复杂时生成。
6. `Cross-Service Overview`：仅当多个 service 互相依赖时生成，只画服务间调用、数据所有权、事实提供方和禁止跨界写入。

## 视觉规范

请使用一致的图例：

- 蓝色：用户、管理员、UI、API 或外部入口。
- 绿色：当前任务的主要业务 owner 或领域模块。
- 橙色：外部事实提供方、基础设施、运行时、第三方系统或资源池。
- 紫色：消费方、下游流程、运行入口或异步任务。
- 灰色：DB 表、持久化对象、migration、历史状态、归档数据。
- 红色：blocker、禁止路径、已废弃路径、不支持行为、失败分支。
- 黄色：需要人工确认、二次确认、审批或高风险操作。
- 实线箭头：必须发生的调用、写入、状态转换或业务依赖。
- 虚线箭头：只读查询、说明性依赖、观测数据或审计信息。
- 红色 stop / T 形标识：被移除、禁止或不可达的旧路径。

节点形状必须表达语义：

- 动作 / API / 命令：圆角矩形。
- 业务对象 / 模块 owner：圆角矩形或泳道。
- 数据表 / 持久化对象：圆柱或灰色矩形。
- 状态：矩形或带状态名的圆角矩形。
- 判断 / 选择 / 条件分支：必须使用棱形节点，样式见 `references/drawio-styles.md`。
- 不要用普通矩形表达 `if / switch / 是否 / 选择 / 分支` 语义。
- 棱形节点的出边必须带条件标签，例如 `yes / no`、`same target / different target`、`pass / fail`、`allowed / blocked`。

布局要求：

- 优先使用从左到右或从上到下的主流程，避免交叉线。
- 主流程必须连续，blocker 和异常分支从主流程侧边分出。
- 每个节点文字控制在 1-4 行；复杂约束放 callout。
- 图要展示业务关系，不能只是把文档段落搬进多个文本框。
- 每条边都必须代表业务关系、调用、数据流、状态转换、读写、副作用、阻断或依赖。
- 不要为了图好看新增不存在的队列、网关、服务、缓存、中间件或领域层。

## Page 1: Use Case Workflow

画指定 service 的具名核心用例流程，而不是概念说明图、全系统鸟瞰图或静态分层图。必须包含：

- 真实入口：用户动作 / API / route / handler / consumer / job / command。
- handler / consumer 解析：请求字段、上下文、鉴权、幂等键、trace、tenant 或 actor。
- application 编排：读取 facts、加载聚合 / 当前状态、调用 domain policy、选择执行路径。
- 具体 guard：权限、状态、容量、依赖、并发、版本、幂等、重复提交、软删除等。
- 具体写入：repository / adapter 调用、表名、字段组、事务边界、锁定对象。
- 具体事件：outbox event、消息 topic、payload 关键字段、消费方和最终副作用。
- 成功返回：DTO、状态、用户可见结果、后续异步结果。
- blocker 返回：错误码、blocking reason、用户提示、重试 / 不重试、回滚或不写入保证。

必须满足：

- 所有“是否”“选择”“条件判断”“类型分派”“guard 通过/失败”节点都用棱形。
- 每个棱形节点至少有两条带标签出边。
- 动作节点只写动作，不混入选择条件。
- blocker 节点必须写清用户可见结果或系统副作用。
- 成功路径必须写清最终状态或交付物。
- 节点名必须尽量使用项目里的真实 handler、use case、repository、table、event、consumer 名称；不允许只写抽象分层名。

推荐流程结构：

1. 入口动作：该 service 的用户 / API / job / handler 触发。
2. 读取当前事实：DB、配置、外部系统、已有状态。
3. 第一个关键判断：是否已有对象、是否满足前置条件、是否命中锁或幂等。
4. 业务校验：权限、状态、容量、依赖、并发 guard、expected 值。
5. 写入或副作用：创建、更新、释放、调用、发送事件、启动任务。
6. 成功返回：返回 DTO、状态、可见结果。
7. blocker 返回：错误码、blocking reason、用户提示、回滚或不写入保证。

如果有二次确认或高风险动作：

- 普通动作先返回黄色确认上下文或红色 blocker。
- 用户显式确认后才进入单独动作节点。
- 确认动作必须有 guard 棱形节点，例如 expected 值、版本号、当前状态、并发锁、是否已被消费。
- guard 失败必须走 blocker，不做部分写入。

## Page 2: Async Events And Side Effects（可选）

如果用例涉及 outbox、消息队列、异步消费、通知、索引、计数器、审计、后台任务或外部调用，生成本页。

必须包含：

- 事务内写入的 outbox / event / job 记录。
- dispatcher / scheduler / relay 的触发方式、重试方式和可观测状态。
- broker / queue / topic / routing key / payload 关键字段。
- consumer 名称、幂等键、去重表或幂等策略。
- consumer 读取 facts 或调用外部 service 的依赖边界。
- 最终副作用，例如通知被回复者、刷新搜索索引、更新榜单、写审计、发送邮件。
- 失败分支和用户 / 操作者可见结果。
- 哪些 bypass、force、fallback 或 legacy path 不能绕过 gate。

如果没有异步链路，不要虚构；在 `Use Case Workflow` 页用 callout 标注“输入未确认异步副作用”即可。

## Page 3: Data And Transaction Boundaries

根据该 service 的任务内容画数据模型、状态机、副作用和回滚关系。必须包含适用项：

- 新增 / 修改 / 删除的数据表或字段。
- 状态枚举、允许状态、禁止状态。
- active / current / latest 这类唯一性不变量。
- 事务边界和锁定对象。
- 写入顺序、回滚、补偿或人工恢复点。
- migration up / down 的限制。
- 审计字段、操作者字段、时间字段、快照字段的语义。
- 哪些字段只是解释性快照，不能被画成强一致锁或强资源保留。

如果任务没有数据库变更，则将本页改为 `State And Side Effects`：

- 展示内存状态、缓存、文件、外部 API、容器、消息、任务调度或用户可见副作用。
- 明确失败后哪些副作用必须回滚，哪些需要补偿，哪些保持只读。

## Page 4: Service Business Relationship Map（可选）

仅当 service 的 owner 边界、跨 service 依赖或禁止写入关系需要补充解释时，生成本页。它只能作为用例流程的边界补充，不能替代 `Use Case Workflow`。

必须包含：

- 该 service 的关键角色、API / handler / job 入口或外部系统。
- 该 service 拥有或编排的主要业务对象。
- 每个业务对象的 owner module / application service / aggregate / table。
- 关键基数关系，例如 `1:1`、`0..1`、`1:N`、“最多一个 active”。
- 该 service 内部的数据事实与运行事实边界。
- 哪些模块只提供 facts，哪些模块拥有业务决策。
- 哪些外部 service 或下游只能消费已确认结果，不能反向创建或修改 owner 数据。
- 明确禁止或废弃的旧路径，特别是跨 service 直接写 owner 数据的路径。

每个核心节点至少回答：

- 它负责什么。
- 它不负责什么。
- 它和上游 / 下游的业务关系是什么。

## Page 5: Validation / Guardrails（可选）

如果验证策略复杂，生成本页。

必须包含：

- 哪些测试覆盖主路径。
- 哪些测试覆盖 blocker / guard / 并发 / migration。
- 哪些脚本或 hook 是完成门禁。
- 哪些 review 重点必须人工检查。
- 哪些非目标必须防止误实现。

## 非目标与禁止误画

从输入文档中提取非目标，用红色 callout 展示。不要新增输入中不存在的非目标。

通用禁止项：

- 不得把 implementation plan 画成当前已落地事实，除非代码已经证明落地。
- 不得把 draft 设计画成 current 架构。
- 不得把 service detail diagram 画成抽象分层图或模块总览图。
- 不得只画 `handler -> application -> domain -> repository -> DB`，必须补齐具名用例、具体判断、表、事件、事务、副作用和可见结果。
- 不得把说明性 snapshot 画成强一致资源锁。
- 不得把只读 facts provider 画成业务 owner。
- 不得把消费方画成 owner。
- 不得省略 blocker、失败分支、回滚限制和旧路径删除。
- 不得只画“系统会统一处理”这类无 owner 表达；必须落到具体模块、API、表、任务或脚本。

## 自检清单

输出 XML 前逐项检查：

- 是否明确标出了当前 service scope，而不是只画全局 overview。
- 是否明确标出了具名核心用例，而不是只画 service 的静态模块结构。
- 是否能从 API / handler / consumer / job 入口追踪到 application 编排、guard、DB transaction、outbox / event、async consumer 和可见结果。
- 如果涉及多个 service，是否每个 service 都有独立视角，overview 只作为补充。
- 是否遵循项目文档产物规则；若项目没有规则，是否给出了默认建议落点。
- 是否一眼能看出核心业务 owner。
- 是否一眼能看出 facts provider、业务 owner、消费方、数据表之间的边界。
- 是否展示了业务对象之间的基数关系和不变量。
- 是否展示了主流程、成功路径、失败路径和 blocker。
- 是否展示了 handler / consumer 解析、application 编排、repository / adapter 调用和具体表 / topic / consumer 名称。
- 是否所有判断 / 选择 / 条件分支都使用棱形节点。
- 是否所有棱形节点出边都有条件标签。
- 是否没有用普通矩形表达“是否 / 选择 / if / switch”。
- 是否展示了二次确认或高风险动作的确认上下文与 guard。
- 是否展示了数据副作用、事务边界、回滚或补偿。
- 如果有异步链路，是否展示了 outbox / dispatcher / broker / consumer / 幂等 / 最终副作用。
- 是否明确画出禁止旧路径、不可绕过 gate 或废弃 fallback。
- 是否把 target design / planned 与 current fact 区分清楚。
- 是否没有新增不存在的服务、队列、缓存、中间件、事件或外部系统。
- 是否没有把文字说明堆成图，而是真的画出业务关系。
````
