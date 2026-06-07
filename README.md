# my-agents

这份仓库是本机共享 agent / harness / workflow 的主体目录。

主体内容放在：

- `~/.agents/AGENTS.md`
- `~/.agents/skills/`
- `~/.agents/harness/`
- `~/.agents/claude-agents/`
- `~/.agents/codex-agents/`

`~/.claude/`、`~/.codex/` 默认只作为入口层，不在里面维护共享正文。

## 新机器安装

1. 克隆仓库到 `~/.agents`

```bash
git clone git@github.com:architectcgz/my-agents.git ~/.agents
```

2. 执行机器级 bootstrap

```bash
bash ~/.agents/harness/bootstrap-agent-home.sh
```

这个脚本会补齐并校验：

- `~/.agents/CLAUDE.md -> AGENTS.md`
- `~/.claude/AGENTS.md -> ~/.agents/AGENTS.md`
- `~/.claude/CLAUDE.md -> AGENTS.md`
- `~/.claude/agents -> ~/.agents/claude-agents`
- `~/.claude/skills -> ~/.agents/skills`
- `~/.codex/AGENTS.md -> ~/.agents/AGENTS.md`
- `~/.codex/CLAUDE.md -> AGENTS.md`
- `~/.codex/agents -> ~/.agents/codex-agents`
- `~/.codex/skills`

其中 `~/.codex/skills` 的规则是：

- 如果缺失，就创建为 `~/.agents/skills` 的软链接
- 如果已经是目录入口，就保留，不强行替换
- 如果被普通文件或错误软链接占住，脚本会直接失败，不静默覆盖

3. 单独跑一次校验

```bash
bash ~/.agents/harness/check-agent-home.sh
```

## 项目初始化

进入目标仓库后，按需初始化项目 harness 和 workflow：

```bash
bash ~/.agents/harness/init-project.sh "$PWD"
```

如果项目已经有 harness，只需要安装共享 workflow：

```bash
bash ~/.agents/harness/workflow-installer.sh "$PWD" code-workflow
```

## 共享 workflow 变更后的同步

如果当前任务修改了 `~/.agents/harness/workflows/<workflow-name>/`，结束前要显式同步到目标仓库：

```bash
bash ~/.agents/harness/workflow-sync.sh <repo-root> <workflow-name>
```

## 常见失败点

- `~/.claude/AGENTS.md` 或 `~/.codex/AGENTS.md` 已经是普通文件，不是软链接
- `~/.claude/skills`、`~/.claude/agents`、`~/.codex/agents` 被现有目录占住
- `~/.codex/skills` 已存在，但不是目录也不是正确软链接
- 仓库没有被克隆到 `~/.agents`

遇到这些情况时，先手动清理冲突，再重新执行 bootstrap。
