`~/.agents/skills/` 现在作为跨 agent 共享 skill 的主体目录。

约定：

- 通用 skill 的主位置放在 `~/.agents/skills/`
- `Claude`、`Codex` 或其他 agent 需要兼容时，各自的私有 skill 目录应回链到这里
- 还没迁完的历史 skill 可以暂时继续保留在各 agent 私有目录，但新改动优先收口到这里

## 重要：软链接结构

**关键设计**：`~/.claude/skills` 和 `~/.codex/skills` 都是指向 `~/.agents/skills` 的**父目录级软链接**。

```bash
~/.agents/skills/                     # 主体（实体目录）
~/.claude/skills -> ~/.agents/skills  # 父目录软链接
~/.codex/skills -> ~/.agents/skills   # 父目录软链接
```

**创建新 skill 的步骤**：

1. 在主体创建：`mkdir -p ~/.agents/skills/<skill-name>`
2. **无需任何额外操作** — Claude 和 Codex 自动可用

**容器 skill（如 superpowers）**：
- 只需在 `~/.agents/skills/superpowers/` 创建容器目录
- 所有子 skills 放在容器内
- Claude 和 Codex 通过父目录软链接自动访问

**特殊目录**：
- `~/.codex/skills/.system/` — Codex 系统级 skills（如 skill-creator），保留在实体目录中
- 项目专属 skills — 放在项目的 `.agents/skills/` 目录中作为 wrapper/补充

**废弃的工具脚本**：
- `bash ~/.agents/scripts/setup-skill-links.sh` — 已不再需要（仅保留用于手动修复）

当前处于迁移期：

- 一部分目录已经是实体目录
- 一部分目录仍然是指向 `~/.codex/skills/<name>` 的历史软链
- 已迁成主体目录的后端/安全类 skill 包括：
  - `backend-engineer`
  - `go-backend`
  - `onion-clean-architecture`
  - `security-vulnerability-scan`
  - `code-workflow`

共享 workflow / harness 入口：

- harness 初始化入口
  - `bash ~/.agents/harness/init-project.sh <repo-root>`
  - 这个入口会先初始化 harness，再默认安装 `code-workflow`，最后跑项目本地 consistency check。
  - 低层命令 `harness-initializer.py` 和 `workflow-installer.sh` 继续保留给修复、调试和局部安装场景。

- `project-template`
  - 负责集中维护可复用的项目代码模板资产，以及项目级 `AGENTS.md` starter。
  - 不负责决定是否安装 harness 或 workflow package。

- `code-workflow`
  - 负责跨项目的非琐碎任务工作流模型：`琐碎任务 / 非琐碎任务` 判定、`writing-plans` 前置、每任务独立 worktree、`task-slug`、implementation plan、startup gate、review/doctor 分层。
  - 对 `非琐碎任务`，`completion-full` 只算实现上下文自检；完成前还需要独立 `code-reviewer` agent gate，再进入 `workflow-governance`。
  - 共享 package：`~/.agents/harness/workflows/code-workflow/`
  - 独立 review handoff 协议：`~/.agents/harness/workflows/code-workflow/independent-review-protocol.md`
  - 共享脚本入口：`bash ~/workspace/projects/scripts/start-workflow.sh <topic-or-slug>`
  - 共享安装器：`bash ~/.agents/harness/workflow-installer.sh <repo-root> code-workflow`
  - 共享同步入口：`bash ~/.agents/harness/workflow-sync.sh <repo-root> code-workflow`
  - 如果当前任务改了 shared `code-workflow` package，本轮结束前必须对目标仓库显式跑一次同步，不假设自动传播
  - 共享基线校验：`bash ~/.agents/harness/workflow-sync-check.sh <repo-root> code-workflow`
  - 完成归档入口：`bash harness/workflow-plugins/code-workflow/archive_task_artifacts.sh [--task-slug <slug>]`

- `workflow-package-manager`
  - 负责共享 workflow package 的选择、安装、升级和漂移校验。
  - 入口：`bash ~/.agents/harness/workflow-installer.sh <repo-root> <workflow-name>`
  - 校验：`bash ~/.agents/harness/workflow-sync-check.sh <repo-root> <workflow-name>`

全局守卫：

- `bash ~/.agents/scripts/check-project-skill-wrapper-shape.sh <project-root>`
- 用于检查项目里的 `.agents/skills/*/SKILL.md`：
  - 如果声明了 `通用主体：\`~/.agents/skills/...\``
  - 就必须保持“项目入口 / 项目补充”的薄包装形态，不能再把整份通用 skill 正文复制回项目里

如果后续需要新增或修改通用 skill，优先操作 `~/.agents/skills/` 下的对应目录，再视需要同步或回链到各 agent 私有目录。
