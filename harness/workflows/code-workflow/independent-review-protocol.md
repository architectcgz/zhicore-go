# Independent Review Protocol

`code-workflow` 的实现完成后，不能直接在原实现上下文里宣布完成。

对 `非琐碎任务`，默认顺序是：

1. 当前实现上下文完成本地实现与最小充分验证
2. 运行项目本地 `completion-full` 检查
3. 交给独立 `code-reviewer` agent 做 gate review
4. 若有 material findings，回到实现上下文修复并重跑受影响验证
5. review 通过后，再进入 `workflow-governance` / doctor / 收尾

## 为什么不在原上下文 review

- 原实现上下文只能算 self-check，不算独立 gate
- 独立 reviewer 更容易发现 owner 漂移、结构债、验证缺口和回归风险
- `agent orchestration` 与 `mechanical enforcement` 必须分离；独立 review 属于前者

## Reviewer 输入最小集合

发给独立 reviewer 的上下文应尽量收敛，而不是整段实现对话原样转发。至少包含：

- 仓库根路径
- `task-slug`
- implementation plan 路径
- review 目标：commit range、当前 diff，或明确的文件列表
- 当前实现已执行的验证命令与结果
- 这次变更的高风险点与 review focus
- 相关架构文档路径、契约文档路径、已知结构债路径

## Reviewer 必做项

- 使用 `code-reviewer` skill
- 以项目内架构文档、契约文档、AGENTS 规则为 review 基准
- 明确区分：
  - blocker findings
  - suggestions
  - missing validation
- 给出 gate verdict：`pass` / `pass with minor issues` / `blocked`

## 项目本地架构检查的使用方式

如果仓库暴露了项目本地的架构或 workflow 检查入口，reviewer 应把它们作为 review 输入，并在必要时自己重跑最小相关集合，例如：

- `scripts/check-workflow-complete.sh`
- `scripts/check-backend-architecture.sh`
- `scripts/check-frontend-architecture.sh`
- 项目自定义的 review / contract / policy checks

这里的原则不是“机械地全跑一遍”，而是：

- 先读取当前实现上下文已经执行过什么
- reviewer 识别哪些结果足以直接复用
- 对高风险或证据不足的部分，再独立补跑最小充分检查

## 推荐的 reviewer handoff 结构

```text
Repository: <repo-root>
Task Slug: <task-slug>
Plan: <plan-path>
Review Target: <commit-range | diff | file-list>
Validation Evidence:
- <command 1>: <result>
- <command 2>: <result>
Architecture Inputs:
- <doc path 1>
- <doc path 2>
Known Risks / Review Focus:
- <risk 1>
- <risk 2>
Project-local Checks To Consider:
- <command/path 1>
- <command/path 2>
```

## 子 agent 使用约定

- 默认使用独立 `code-reviewer` agent
- 默认 `fork_context=false`
- 不把完整实现对话原样灌给 reviewer；只传收敛后的 handoff
- 进入 `code-workflow` 就等价于用户已经对这一个“最小必要”的 reviewer delegation 给出显式授权；不要再额外等待一轮“是否允许起 reviewer subagent”的确认
- 如果工具策略或用户要求不允许启动子 agent，必须明确说明：
  - 独立 review gate 尚未满足
  - 当前只能做 self-check，不能当作最终 completion review
