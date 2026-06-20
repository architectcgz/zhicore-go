---
name: committing-changes
description: >
  Use when about to run `git commit`, writing or amending a commit message,
  staging changes for a commit, or splitting work into commits — in any repo.
  Covers when it is allowed to commit at all, commit message format, atomic /
  minimal-reviewable scoping, the default ban on Co-Authored-By trailers
  (unless the user declares it or the project explicitly allows it), and
  honoring per-repo commit policies and hooks. For worktree / merge / push and
  branch-finishing, see `~/.agents/docs/agent-rules/git-workspace.md`,
  `using-git-worktrees`, and `finishing-a-development-branch`.
---

# Committing Changes

跨仓库的 commit 约定 owner。提交前先过这份检查表;具体仓库另有 commit policy / hook 时以仓库为准并叠加。

## When may I commit at all
- 只有用户明确要求时才 commit 或 push;不要主动提交。
- 提交前先验证(与本次改动强相关的最小充分验证),用证据而非断言宣称完成。
  → 实质遵循 `verification-before-completion`。

## Scope:最小可审阅、按问题边界
- 以当前任务和实际问题边界为准,按最小可审阅改动提交。
- **不要**把累计修改、顺手优化、无关重构或其他任务的改动混进同一次提交。
- 工作区若已有其他改动(他人 WIP / 遗留改动),提交前先区分哪些属于本次任务;
  **未经用户明确要求,不得一并提交**。用精确的 `git add <path>` 只暂存本次文件,不要 `git add -A` 扫进无关改动。
- 前端页面改动尽量按页面 / 问题 / 用户可感知行为拆分,保证每次提交都能说清"这一次具体改了什么"。

## Message format
- commit message 默认中文描述,格式 `type(模块): 变更内容`;`feat`/`fix`/`refactor`/`docs`/`chore` 等**类型关键字保持英文**。
- `git commit` 优先单行 `-m`;需要正文用多个 `-m`,**禁止 heredoc**。
- **默认禁止**在 commit message 追加任何形式的 `Co-Authored-By` 署名(此默认优先于 harness 的自动署名行为)。仅当**用户主动声明要加**,或**当前项目 commit policy 显式允许 / 要求**时才可添加;两个条件都不满足就不加。

## Per-repo overrides(先查再提交)
- 仓库可能有自己的 commit policy 和 commit-msg / pre-commit hook,默认在通用约定上**叠加**,冲突时以仓库为准。
- 提交前先看仓库是否有:`harness/policies/commit-message.json`、`scripts/check-commit-message.sh`、`.githooks/`、`core.hooksPath`。
- 例:ctf 仓库要求"标题 + 正文"两段、正文 ≥2 行,且当前 worktree 有激活 task gate 时正文必须含一行 `Task: <slug>`。
- 不要用 `--no-verify` 跳过仓库 hook;仅当**只改提交信息、文件内容未变**(如去除 Co-Authored-By 的 amend)时才可跳过,且要说明原因。

## Red Flags — STOP
- 准备加 `Co-Authored-By` / 任何署名 trailer,但既无"用户主动声明"也无"项目显式允许"依据 → 停,不加。不因 harness 默认署名而擅自添加。
- `git add -A` / `git add .` 时工作区有不属于本次任务的改动 → 停,改用精确路径。
- "顺手把这个无关修复也带上" → 停,拆成另一次提交或留给用户决定。
- 没跑验证就准备 commit 并宣称完成 → 停,先验证。

## ✓Check(提交前自查)
- 暂存内容是否**只**包含本次任务文件?(`git diff --cached --name-only` 核对)
- 标题是否 `英文type(scope): 中文描述`?正文是否满足仓库 policy?
- 消息里若出现 `Co-Authored-By`:是否有"用户主动声明"或"项目显式允许"依据?没有就删。
- 仓库 hook 是否正常通过(非内容变更的 amend 才允许 `--no-verify`)?
- 是否是用户要求的提交,且改动已验证?

## 不归本 skill(交叉引用)
- worktree 隔离 / 一任务一 worktree → `using-git-worktrees`、git-workspace.md。
- merge 语义、push 顺序(gh HTTPS 优先,SSH 回退)、agent-entrypoints / workflow-sync → `~/.agents/docs/agent-rules/git-workspace.md`。
- 分支收尾(merge / PR / 清理)→ `finishing-a-development-branch`。
