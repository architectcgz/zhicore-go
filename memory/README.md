# 跨工具记忆系统

## 设计原则

基于 2026 年多 agent 系统的工业实践（mem0.ai、vectorize.io、mindstudio.ai），采用 **多作用域分层共享** 架构。

## 目录结构

```
~/.agents/memory/
├─ shared/          # 跨工具共享：用户角色、通用反馈、跨项目参考
├─ claude/          # claude 专属：交互偏好、UI/UX 倾向
├─ codex/           # codex 专属：编排流程、worktree 习惯
├─ MEMORY.md        # 总索引
└─ README.md        # 本文件
```

## 作用域规则

### 写入时强制标注

每个记忆文件的 frontmatter 必须包含：

```yaml
metadata:
  type: user | feedback | project | reference
  scope: [claude, codex] | claude | codex
```

### 检索时组合加载

- claude 加载：`shared/` + `claude/`
- codex 加载：`shared/` + `codex/`

### 跨作用域提升

当某条反馈满足以下条件时，从单工具层提升到 `shared/`：

1. 在 claude 和 codex 对话中都被命中过 ≥2 次
2. 反馈内容对两个工具都适用（如"别 mock 数据库"）
3. 用户明确表示"这是通用规则"

提升流程：
1. 将文件从 `claude/` 或 `codex/` 移到 `shared/`
2. 更新 frontmatter：`scope: [claude, codex]`
3. 更新 `MEMORY.md` 索引标注

## 项目级记忆

不迁移到本目录的内容：

- 项目特定进度、技术债、临时决策
- 仅在单个项目/工作区有效的状态

这些继续保留在：
- `~/.claude/projects/<project-hash>/memory/`
- `~/.codex/projects/<project-hash>/memory/`

## 按需加载机制

为避免上下文浪费，记忆采用分级加载：

### 优先级标注

frontmatter 增加 `priority` 和 `keywords` 字段：
```yaml
metadata:
  priority: high | medium | low
  keywords: [关键词1, 关键词2, ...]
```

### 加载策略

| 优先级 | 加载时机 | 上下文开销 |
|--------|---------|-----------|
| `high` | 启动时自动加载 | 每个 ~200-600 tokens |
| `medium` | 用户 prompt 命中关键词时加载 | 按需 |
| `low` | 手动引用或明确需要时加载 | 按需 |

### MEMORY.md 索引格式

每条记忆附加快速规则和关键词：
```markdown
- [name](path.md) — 描述 `[scope: ...]`
  - **关键词**：`keyword1`, `keyword2`
  - **快速规则**：简短规则摘要（~50 tokens）
  - **优先级**：`high/medium/low`
```

启动时只读索引和高优先级记忆；中低优先级按需加载。

## 隔离机制

**在数据层隔离，不在提示层**

- ✅ 用文件路径（`shared/` vs `claude/`）和 frontmatter `scope` 字段强制隔离
- ❌ 不依赖"claude 你别读 codex 的记忆"这种提示

## 三大风险与应对

| 风险 | 表现 | 应对 |
|------|------|------|
| **Over-share** | claude 的简洁偏好污染 codex 的详细输出 | 严格遵守 scope 标注，单工具偏好不放 shared |
| **Under-share** | 重复学习用户习惯，反馈维护两份 | 通用反馈放 shared，定期检查可提升项 |
| **作用域蔓延** | scope 维度过多（>4个）导致混乱 | 当前只用 3 维：shared / claude / codex |

## 迁移记录

- 2026-06-19：从 `~/.claude/projects/-home-azhi-workspace-projects/memory/` 迁移首个共享反馈
- 初始记忆：`shared/feedback_skill_creation.md`

## 参考资料

- [Multi-Agent Memory Systems - mem0.ai](https://mem0.ai/blog/multi-agent-memory-systems)
- [Shared vs Private AI Agent Memory - MindStudio](https://www.mindstudio.ai/blog/shared-vs-private-ai-agent-memory-team-access-control/)
- [Single Brain Multi-Agent Systems - Vectorize](https://vectorize.io/articles/single-brain-multi-agent-systems)
