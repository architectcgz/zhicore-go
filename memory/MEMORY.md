# Memory Index

本目录包含跨 claude 和 codex 的共享记忆，以及各工具的专属记忆。

## 记忆分层规则

- **shared/**：claude 和 codex 都能读写的跨工具记忆
- **claude/**：只属于 claude 的交互偏好和工作模式
- **codex/**：只属于 codex 的编排流程和 worktree 习惯
- **项目级记忆**：继续保留在 `~/.claude/projects/.../memory/` 或 `~/.codex/projects/.../memory/`

## 作用域标注

每个记忆文件的 frontmatter 必须包含 `scope` 字段：
- `scope: [claude, codex]` — 共享记忆
- `scope: claude` — claude 专属
- `scope: codex` — codex 专属

## 提升机制

当某条反馈在两个工具中都反复命中时，手动提升到 `shared/` 并更新 `scope: [claude, codex]`。

---

## Shared（共享记忆）

### Reference

- [skills-directory-structure](shared/skills_directory_structure.md) — Skills 目录结构和软链接关系，~/.claude/skills 和 ~/.codex/skills 都是父目录级软链接
  - **关键词**：`skills`, `软链接`, `symlink`, `skill 创建`, `目录结构`
  - **核心事实**：`~/.claude/skills -> ~/.agents/skills` 和 `~/.codex/skills -> ~/.agents/skills`（双向父目录软链接），新 skill 无需任何链接操作
  - **优先级**：`high`（避免重复创建链接）

### Feedback

- [skill-creation-workflow](shared/feedback_skill_creation.md) — 创建 skill 前必须先使用 writing-skills skill 和检查现有工具 `[scope: claude, codex]`
  - **关键词**：`skill`, `writing-skills`, `create-skill`, `SKILL.md`, `helper script`
  - **快速规则**：创建 skill → `/writing-skills` → TDD 流程 → 验证
  - **优先级**：`high`（高频命中）

## Claude 专属

（暂无）

## Codex 专属

（暂无）
