# 项目协作规则

## 快速路由

| 任务类型 | 必读内容 | 工作流 / Skill |
| --- | --- | --- |
| 后端功能（API / Service / Repository） | `[后端模式文档]` + `tests/README.md` + `harness/policies/reuse-first.yaml` | `backend-engineer` skill -> `code-workflow` |
| 前端功能（Page / Component） | `[前端模式文档]` + 前端本地规则 | `frontend-engineer` skill -> `code-workflow` |
| Review | `docs/documentation-rules.md` 的 review 相关规则 | `reviewer` skill |
| 后端 bugfix | `tests/README.md` + 后端模式文档 | `systematic-debugging` -> `backend-engineer` |
| 前端 bugfix | 前端本地规则 | `systematic-debugging` -> `frontend-engineer` |
| 新增或修改测试 | `tests/README.md` | `test-driven-development` skill |
| 架构变更 | `docs/architecture/` + `brainstorming` + `writing-plans` | 先写计划，再进入 `code-workflow` |
| 文档更新 | `docs/documentation-rules.md` + 最近的父级索引 | 直接编辑；若属于实现任务的一部分，跟随实现工作流 |
| 新的非平凡任务 | `[项目任务入口脚本或工作流]` | `brainstorming` -> `grill-with-docs` -> `writing-plans` |
| 其他 | 完整读取本文件，再判断是否需要澄清 | 从 `harness-router` skill 开始 |

## 会话纪律

- 新任务进入同一会话时，重新读取本文件和相关 skill 的 `SKILL.md`。
- 上下文压缩或清空后，按项目入口重新加载当前规则。
- 修改 `AGENTS.md`、`docs/documentation-rules.md`、harness 策略或受保护脚本前，先确认对应规则和机械检查。
- 非平凡任务完成前，运行项目要求的最小充分验证，并在发现可复用经验时更新对应规则或反馈位置。
- 提交前遵守全局 Git 规则和本仓库提交约束。

## 停止信号

出现下面想法时，先停下来重读相关规则：

| 想法 | 实际要求 |
| --- | --- |
| “这次先跳过 reuse-first” | 没有例外，先搜索既有模式。 |
| “时间紧，逻辑改动先不写测试” | 行为、状态、数据流、权限、校验、异步流程、算法、接口契约或可复现 bug 改动需要最窄相关测试。 |
| “改动很小，不用看规则” | 判断标准是是否触达受保护边界，不是改动行数。 |
| “我记得规则怎么写” | 规则会演化，必须读当前版本。 |
| “先违反一次架构约束” | 先记录影响、风险和退出条件，再决策。 |
| “删几个测试没关系” | 删除测试需要明确的移除条件。 |

## 1. 作用范围和继承

- 本文件定义 agent 在本仓库工作时必须遵守的项目级规则。
- 本文件服从更高优先级的系统、用户和全局规则。
- 当 Claude / Codex 自动发现入口属于本地工作流时，仓库根目录应保持 `CLAUDE.md -> AGENTS.md` 软链接。
- 全局 `AGENTS.md` 是默认策略来源；本文件只记录仓库特有的约束、命令、架构边界、文档入口和路由规则。
- 决策前优先读取项目事实：源码、配置、构建脚本、测试、文档和已有实现，而不是凭经验猜测。
- 新建 README、docs、初始化说明和 agent 规则时，正文默认使用中文；代码标识、命令、路径、包名、API / 协议字段和外部专有名词保持原文。

## 2. 项目概览

- 项目类型：`[填写：frontend / backend / full-stack / library / CLI / service / other]`
- 主要语言：`[填写]`
- 框架：`[填写]`
- 包管理器 / 构建工具：`[填写]`
- 运行时要求：`[填写]`
- 主要入口：`[填写]`

## 3. 初始化和常用命令

使用项目已有包管理器和脚本，不要发明仓库里不存在的命令。

```bash
# 安装依赖
[填写]

# 启动开发服务或本地进程
[填写]

# 运行测试
[填写]

# 运行 lint
[填写]

# 运行 typecheck
[填写]

# 构建
[填写]
```

如果命令不可用或依赖缺失，汇报被阻塞的确切命令和原因。

- 如果仓库标准化使用 Trellis 等 repo-scoped workflow layer，在这里或最近的 setup 文档记录准确安装和初始化命令；否则删除本条，不要暗示 Trellis 已经存在。

## 4. 架构和变更边界

- 遵循现有模块布局、命名约定、分层、错误处理、日志风格和测试模式。
- 在这里记录仓库真实代码边界，例如前后端归属、服务层边界、生成文件策略、迁移流程、API 兼容规则，或通常不应编辑的目录。
- 不要引入新抽象、依赖、框架或架构层，除非当前任务确实需要。
- 生成文件、vendored code、构建产物和 lockfile 必须符合仓库既有策略。
- 不要把实现笔记、TODO 标记、设计解释或内部评论渲染到用户可见 UI 或 API 响应中。

## 4.5. 代码质量核对

完成代码变更后，用下面的问题做自检：

### 手术式改动

- [ ] 每一行改动都能追溯到用户请求吗？
- [ ] 是否引入了用户没有要求的功能或优化？
- [ ] 如果用户要求撤销最后一个功能，是否能干净删除？

### 避免过早抽象

- [ ] 这个抽象服务于几个真实用例？少于 3 个时，为什么现在需要？
- [ ] 如果只有一个用例，接口是否比问题本身更复杂？
- [ ] 这个抽象是否降低了实际复杂度，而不只是移动了复杂度？

### 测试质量

- [ ] 删除这个测试后，还能检测到同类回归吗？
- [ ] 这个测试验证的是行为，还是实现细节？
- [ ] 测试失败时，错误信息是否能说明失败原因？

### 命名和可读性

- [ ] 函数或变量名是否说明职责和副作用？
- [ ] 删除注释后，6 个月后的维护者是否仍能理解代码？
- [ ] 魔法数字是否有业务含义？如果有，是否应提升为常量？

### 依赖和耦合

- [ ] 改动这个模块会影响几个其他模块？
- [ ] 模块依赖的是接口还是具体实现？
- [ ] 添加新功能时是否必须修改大量现有代码？

完整参考：`~/.agents/harness/docs/verification-questions-guide.md`。

## 5. 项目特定验证

先遵守全局验证规则，再列出本项目最小充分验证命令。

```bash
[填写项目特定验证命令]
```

仓库存在测试时，在这里保留明确测试规则：

- 行为、状态、数据流、权限、校验、异步流程、算法、接口契约或可复现 bug 改动，先编写或更新最窄相关测试。
- 简单 UI / 展示层改动（布局、间距、颜色、排版、静态文案、控件位置迁移且不改变状态语义）不默认使用 TDD，改用最小充分的类型、构建、组件渲染、截图或人工验证。
- TDD 测试是长期行为规格和回归保护。只有在行为信号重复、过时、过度耦合实现，或被迁移到更合适归属处时，才删除或合并。
- 每个测试放在能证明对应 contract 的 owner / layer：
  - 后端：包内测试验证模块语义和未导出细节；internal test utilities 支持需要私有访问的 helper；system / API 测试验证黑盒路由或传输行为；runtime / integration 测试覆盖真实数据库、外部进程或容器；architecture 测试验证边界护栏；testkit 放稳定 fixture、builder 和断言。
  - 前端：共享、feature、page、store、API、router、runtime、config 或 utility 的测试靠近归属代码；根级 `src/__tests__` 只放架构、设计系统或跨切面护栏；共享测试 setup 和 helper 放在项目测试支持目录。
- 不要在多个层重复证明同一个行为信号。若多个层都需要覆盖，每层必须证明不同 contract。
- 写完测试后，运行覆盖改动面的最窄测试命令。
- 如果仓库有 `scripts/check-*.sh`、`scripts/check-*.py` 或其他 guard 命令，测试后继续运行相关脚本检查。
- 如果仓库已有 `scripts/check-consistency.sh`、git hooks、CI 或其他机械护栏，把相关测试脚本接入实际 enforcement path，不要只停留在提示文本。
- 如果还没有机械 enforcement path，明确说明它仍未建立。

在这里补充仓库特定约束，例如测试顺序、snapshot 更新策略、浏览器检查、fixture 刷新步骤，或已知昂贵命令。

## 6. Git 和交付说明

遵守全局 Git / worktree 规则，只在这里记录仓库特有交付约束。

- 分支或提交命名约定：`[如与全局默认不同则填写]`
- 必须纳入或排除的生成文件 / lockfile：`[填写]`
- 会影响变更分组的发布、迁移或部署耦合：`[填写]`

## 7. 文档

- 文档规则由 `docs/documentation-rules.md` 维护；文档导航由 `docs/README.md` 维护。
- 创建、移动、删除或编辑文档前，先读 `docs/documentation-rules.md`，再读目标区域最近的索引。
- 新建 README、docs、部署说明和 agent 规则时，正文默认使用中文；代码标识、命令、路径、包名、API / 协议字段和外部专有名词保持原文。
- 当代码行为、API、配置、数据库形态、初始化流程或用户可见行为变化时，先确定文档 owner 和目标路径。
- 如果项目采用常态化交付后 review 治理，review 报告放在 `docs/reviews/`，未解决技术债放在 `docs/todos/debt/`。
- 技术债应拆成独立 debt 文件并由 `docs/todos/debt/` 索引管理，不要让根目录 `DEBT.md` 无限增长。
- 不要在本文件重复完整文档分类表。新增长期文档路径若影响 agent 路由，只在这里添加路由，详细规则放到文档 owner 文件。
