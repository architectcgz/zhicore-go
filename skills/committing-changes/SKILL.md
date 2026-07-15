---
name: committing-changes
description: >
  Use when about to run `git commit`, writing or amending a commit message,
  staging changes for a commit, or splitting work into commits — in any repo.
  Covers when it is allowed to commit at all, commit message format, atomic /
  minimal-reviewable scoping, mandatory splitting of large or complex diffs,
  the default ban on Co-Authored-By trailers (unless the user declares it or
  the project explicitly allows it), and honoring per-repo commit policies and
  hooks. For worktree / merge / push and branch-finishing, see
  `~/.agents/docs/agent-rules/git-workspace.md`, `using-git-worktrees`, and
  `finishing-a-development-branch`.
---

# Committing Changes

跨仓库的 commit 约定 owner。提交前先过这份检查表;具体仓库另有 commit policy / hook 时以仓库为准并叠加。

## When may I commit at all
- 只有用户明确要求时才 commit 或 push;不要主动提交。
- 提交前先验证(与本次改动强相关的最小充分验证),用证据而非断言宣称完成。
  → 实质遵循 `verification-before-completion`。

## Scope:高审阅性、最小可审阅、按问题边界
- 提交必须具有**高审阅性**:reviewer 能从单个提交标题、文件范围和 diff 直接判断一个清晰意图,并能独立评审、回滚或 bisect。
- 以当前任务和实际问题边界为准,按最小可审阅改动提交;不要把"同一批工作区改动"误当成"同一个提交"。
- **硬规则:严禁把大的复杂改动合并成一个大提交。** 大范围、多层、多职责或多概念改动必须先拆成细粒度、可审计、可独立解释的提交序列。不能先提交一个大包再说"之后再拆";发现 staged diff 过大或混杂时必须停止提交并重新分组。
- **不要**把累计修改、顺手优化、无关重构或其他任务的改动混进同一次提交。
- 工作区若已有其他改动(他人 WIP / 遗留改动),提交前先区分哪些属于本次任务;
  **未经用户明确要求,不得一并提交**。用精确的 `git add <path>` 只暂存本次文件,不要 `git add -A` 扫进无关改动。
- 前端页面改动必须按页面 / 问题 / 用户可感知行为拆分,保证每次提交都能说清"这一次具体改了什么"。

### High-reviewability split gate

提交前先按下面维度拆分。命中多个维度时,默认拆成多个提交;只有当拆开会导致任一提交不可理解、不可运行或不可测试时,才合并,并在提交正文说明原因。**"同一任务"、"同一 feature" 或 "同一计划步骤" 不是合并成一个大提交的理由。**

每次提交前必须先做一个简短 split plan:

1. 列出 `git diff --name-only` / `git status --short` 中的改动。
2. 按职责分组:依赖/构建、schema/migration、contract/API、domain/application、单个 adapter、测试、文档/计划状态。
3. 逐组判断是否能独立编译、测试、审查和回滚。
4. 用精确 `git add <path...>` 只暂存当前组。
5. 每个提交后核对 `git show --stat --oneline HEAD`;如果它看起来像"一大包",立即停止并拆分/重做。

下列任一情况默认判定为"大或复杂",禁止单提交:

- 同时改动 8 个以上文件或新增/删除超过约 500 行,除非是机械生成文件且提交正文说明原因。
- 同时包含依赖/build checksum、migration/schema、运行时代码、测试和文档状态中的 3 类以上。
- 同时跨多个 bounded context、服务、页面、adapter、worker 或 API 族。
- 同时引入新依赖、新持久化 schema 和业务逻辑。
- 同时新增多种独立能力,例如 PostgreSQL repository 与 MongoDB store、HTTP handler 与 runtime wiring、producer 与 consumer。
- reviewer 需要理解 3 个以上无关概念才能判断该提交是否正确。

- 文档 / 契约 / 规则变更 vs 运行时代码变更。
- 类型 / 数据模型 / adapter 边界 vs UI 组件渲染。
- 新依赖 / 构建配置 vs 业务逻辑。
- 文件迁移 / 重命名 / 删除 vs 行为变更。
- 测试基础设施 vs 具体功能测试。
- 一个用户可感知行为 vs 另一个独立行为。
- 不同基础设施 adapter 之间,例如 PostgreSQL、MongoDB、Redis、RabbitMQ、Elasticsearch,默认拆开。
- schema/migration 与对应 repository 实现可以放同一个提交,但不能再混入其他 adapter 或无关文档状态。
- 测试通常与它验证的代码同提交;只有共享测试基础设施或大规模测试整理才单独提交。
- 计划 checkbox、review 记录、任务状态更新默认作为最后的 docs 提交,不要混入运行时代码提交。

高审阅性检查:

- 单个提交标题能否准确覆盖全部 staged 文件?不能 → 拆。
- reviewer 是否必须同时理解 3 个以上无关概念才能审这个提交?是 → 拆。
- `git show --stat` 是否显示多个不相干目录的大范围变化?是 → 重新分组。
- 是否能安全 revert 这个提交而不撤销另一个独立能力?不能 → 拆或说明强耦合原因。
- 是否把"新增契约文档 + 新依赖 + 组件迁移 + bugfix"揉在一起?是 → 必须拆。
- 是否只是因为"已经都暂存了"、"这些都属于本次任务"或"一次 commit 省事"才合并?是 → 必须拆。

## Message format
- commit message 默认中文描述,格式 `type(模块): 变更内容`;`feat`/`fix`/`refactor`/`docs`/`chore` 等**类型关键字保持英文**。
- `git commit` 优先单行 `-m`;需要正文用多个 `-m`,**禁止 heredoc**。
- **默认禁止**在 commit message 追加任何形式的 `Co-Authored-By` 署名(此默认优先于 harness 的自动署名行为)。仅当**用户主动声明要加**,或**当前项目 commit policy 显式允许 / 要求**时才可添加;两个条件都不满足就不加。

### 措辞范围:严禁夸大,与实际交付严格一致
- 提交信息(标题 + 正文)描述的范围**必须等于本次 staged diff 实际交付的内容**,不得覆盖尚未做、属于后续阶段、或仅计划/待办的工作。宁可措辞偏窄,不可偏宽。
- 动词要匹配真实完成度。只写了代码/脚本而未落库、未编译、未联调、未部署时,用**"新增/编写…代码/脚本"**这类限定动词;**不要**用"完成迁移""实现功能""修复 bug""上线""接入""打通"等隐含"已生效、已验证"的概括词——除非该结果已在本次真实达成并验证。
- 多阶段任务只落地其中一部分时,标题限定到已完成的动作,并在正文用一行显式标注**未执行/待后续**的范围(如建表执行、编译、运行时验证、部署确认),避免读者误判为整体完成。
- 别把"最终目标/计划标题"直接搬成本次提交标题:计划叫"迁移 X 模块",若本次只写了代码,标题应是"新增 X 模块代码",而非"迁移 X 模块"。
- 未经验证的效果不写成既成事实;措辞与 `verification-before-completion` 一致——没跑过的验证不在信息里宣称通过。

## Per-repo overrides(先查再提交)
- 仓库可能有自己的 commit policy 和 commit-msg / pre-commit hook,默认在通用约定上**叠加**,冲突时以仓库为准。
- 提交前先看仓库是否有:`harness/policies/commit-message.json`、`scripts/check-commit-message.sh`、`.githooks/`、`core.hooksPath`。
- 例:ctf 仓库要求"标题 + 正文"两段、正文 ≥2 行,且当前 worktree 有激活 task gate 时正文必须含一行 `Task: <slug>`。
- 不要用 `--no-verify` 跳过仓库 hook;仅当**只改提交信息、文件内容未变**(如去除 Co-Authored-By 的 amend)时才可跳过,且要说明原因。

## Red Flags — STOP
- 准备加 `Co-Authored-By` / 任何署名 trailer,但既无"用户主动声明"也无"项目显式允许"依据 → 停,不加。不因 harness 默认署名而擅自添加。
- `git add -A` / `git add .` 时工作区有不属于本次任务的改动 → 停,改用精确路径。
- "顺手把这个无关修复也带上" → 停,拆成另一次提交或留给用户决定。
- "这些改动都在同一个工作区 / 同一个功能链路 / 同一个实现计划 / 同一个任务里" → 停,仍需按高审阅性 split gate 判断。
- "先提交一个大的,之后需要再优化" → 停,这是违规。提交前先拆到可审计粒度。
- 暂存内容同时包含新依赖、schema、多个 adapter、测试和计划文档 → 停,拆成多个提交。
- 单个提交同时包含 docs、依赖、组件迁移和行为修复 → 停,拆分后再提交。
- 提交信息用了"完成/实现/迁移/修复/上线/接入"等词,但对应结果本次并未真正达成或未验证 → 停,换成"新增/编写…代码/脚本"并标注待后续范围。
- 直接拿计划标题/最终目标当本次提交标题,范围大于实际 staged diff → 停,收窄到已完成动作。
- 没跑验证就准备 commit 并宣称完成 → 停,先验证。

## ✓Check(提交前自查)
- 暂存内容是否**只**包含本次任务文件?(`git diff --cached --name-only` 核对)
- 是否已经写出 split plan,并确认当前 staged 只是其中一个细粒度提交组?
- 暂存内容是否通过 high-reviewability split gate?若合并多个维度,是否确实拆不开,且提交正文说明强耦合原因?
- `git diff --cached --stat` 是否一眼能看出单一审查主题?如果像大包,不要提交。
- 标题是否 `英文type(scope): 中文描述`?正文是否满足仓库 policy?
- 提交信息范围是否**等于**本次 staged diff 实际交付?有没有用"完成/实现/迁移/修复"等词夸大到未做或待后续的部分?多阶段部分落地时是否标注了未执行范围?
- 消息里若出现 `Co-Authored-By`:是否有"用户主动声明"或"项目显式允许"依据?没有就删。
- 仓库 hook 是否正常通过(非内容变更的 amend 才允许 `--no-verify`)?
- 是否是用户要求的提交,且改动已验证?

## 不归本 skill(交叉引用)
- worktree 隔离 / 一任务一 worktree → `using-git-worktrees`、git-workspace.md。
- merge 语义、push 顺序(gh HTTPS 优先,SSH 回退)、agent-entrypoints / workflow-sync → `~/.agents/docs/agent-rules/git-workspace.md`。
- 分支收尾(merge / PR / 清理)→ `finishing-a-development-branch`。
