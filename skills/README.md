`~/.agents/skills/` 现在作为跨 agent 共享 skill 的主体目录。

约定：

- 通用 skill 的主位置放在 `~/.agents/skills/`
- `Claude`、`Codex` 或其他 agent 需要兼容时，各自的私有 skill 目录应回链到这里
- 还没迁完的历史 skill 可以暂时继续保留在各 agent 私有目录，但新改动优先收口到这里

当前处于迁移期：

- 一部分目录已经是实体目录
- 一部分目录仍然是指向 `~/.codex/skills/<name>` 的历史软链
- 已迁成主体目录的后端/安全类 skill 包括：
  - `backend-engineer`
  - `go-backend`
  - `onion-clean-architecture`
  - `security-vulnerability-scan`
  - `code-workflow`

共享 workflow 入口：

- `code-workflow`
  - 负责跨项目的非琐碎任务工作流模型：`琐碎任务 / 非琐碎任务` 判定、`writing-plans` 前置、每任务独立 worktree、`task-slug`、implementation plan、startup gate、review/doctor 分层。
  - 共享 package：`~/.agents/harness/workflows/code-workflow/`
  - 共享脚本入口：`bash ~/workspace/projects/scripts/start-workflow.sh <topic-or-slug>`
  - 共享安装器：`bash ~/.agents/harness/workflow-installer.sh <repo-root> code-workflow`
  - 共享基线校验：`bash ~/.agents/harness/workflow-sync-check.sh <repo-root> code-workflow`
  - 完成归档入口：`bash scripts/archive-task-artifacts.sh [--task-slug <slug>]`

- `harness-workflow`
  - 负责共享 workflow package 的选择、安装、升级和漂移校验。
  - 入口：`bash ~/.agents/harness/workflow-installer.sh <repo-root> <workflow-name>`
  - 校验：`bash ~/.agents/harness/workflow-sync-check.sh <repo-root> <workflow-name>`

全局守卫：

- `bash ~/.agents/scripts/check-project-skill-wrapper-shape.sh <project-root>`
- 用于检查项目里的 `.agents/skills/*/SKILL.md`：
  - 如果声明了 `通用主体：\`~/.agents/skills/...\``
  - 就必须保持“项目入口 / 项目补充”的薄包装形态，不能再把整份通用 skill 正文复制回项目里

如果后续需要新增或修改通用 skill，优先操作 `~/.agents/skills/` 下的对应目录，再视需要同步或回链到各 agent 私有目录。
