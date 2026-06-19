# Skill 组织规范

## 核心原则

**Claude Code 会自动从 `~/.agents/skills/` 发现全局 skills，不需要任何软链接或安装步骤。**

## 正确的归属规则

### 全局共享 skill

- **位置**：`~/.agents/skills/<skill-name>/`
- **命名**：不带项目前缀（如 `code-reviewer`、`frontend-engineer`、`go-backend`）
- **用途**：跨项目通用能力
- **如何使用**：直接调用，Claude Code 自动发现

### 项目专属 skill

- **位置**：`<project-root>/.agents/skills/<project>-<skill-name>/`
- **命名**：带项目前缀（如 `ctf-dark-surface-alignment`、`ctf-backend-patterns`）
- **用途**：只属于该项目的专门 skill
- **如何使用**：直接调用，Claude Code 自动发现

## 不需要的操作

❌ **不需要**创建软链接：
- 不需要 `<project>/.claude/skills -> ../.agents/skills`
- 不需要 `~/.codex/skills/<project>-<skill>`
- 不需要任何"安装"或"注册"步骤

❌ **不需要**在项目内保留全局 skill 的副本：
- 全局 skill 只维护一份，在 `~/.agents/skills/`
- 项目内不要复制或软链接全局 skill

## 已废弃的脚本

以下脚本基于错误的设计，已标记为废弃：
- `install-project-skills.sh.deprecated`
- `uninstall-project-skills.sh.deprecated`

这些脚本试图通过软链接"安装" skill，但 Claude Code 的 skill 发现机制不需要这些操作。

## 迁移指南

如果现有项目按旧逻辑组织了 skills：

1. 将全局通用 skill 移到 `~/.agents/skills/`
2. 将项目专属 skill 保留在 `<project>/.agents/skills/`（带项目前缀）
3. 删除所有 skill 相关的软链接
4. 删除项目内全局 skill 的过时副本

## 检查清单

新建或整理项目时：

- [ ] 项目专属 skill 在 `<project>/.agents/skills/` 下，带项目前缀
- [ ] 全局共享 skill 在 `~/.agents/skills/` 下，不带项目前缀
- [ ] 没有 skill 相关的软链接
- [ ] 没有全局 skill 在项目内的副本
