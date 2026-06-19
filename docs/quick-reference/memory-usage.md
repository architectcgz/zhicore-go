# 记忆系统使用快速参考

## 新增记忆时

### 1. 判断作用域

| 问题 | 作用域 | 存放位置 |
|------|--------|----------|
| Claude 和 Codex 都需要遵守？ | `[claude, codex]` | `~/.agents/memory/shared/` |
| 只属于 Claude 的交互偏好？ | `claude` | `~/.agents/memory/claude/` |
| 只属于 Codex 的编排流程？ | `codex` | `~/.agents/memory/codex/` |
| 只在当前项目有效的状态？ | `project` | `~/.claude/projects/.../memory/` |

### 2. 创建记忆文件

```markdown
---
name: kebab-case-name
description: 一行摘要，用于决定是否加载
metadata:
  type: user | feedback | project | reference
  scope: [claude, codex] | claude | codex
  priority: high | medium | low
  keywords: [关键词1, 关键词2, ...]
---

记忆正文...

**Why:** 原因/背景/触发事件

**How to apply:** 具体应用规则

**Related:**
- 相关文件或记忆 [[other-memory]]
```

**优先级选择指南**：
- `high`：高频命中（如通用协作规则、核心偏好）→ 启动时自动加载
- `medium`：专题反馈（如 Git 流程、测试策略）→ 命中关键词时加载
- `low`：历史决策、边缘场景 → 手动引用时加载

### 3. 更新索引

在 `~/.agents/memory/MEMORY.md` 对应章节添加：

```markdown
- [显示名称](shared/feedback_xxx.md) — 一行描述 `[scope: claude, codex]`
  - **关键词**：`keyword1`, `keyword2`, `keyword3`
  - **快速规则**：核心规则的一句话摘要（约 50 tokens）
  - **优先级**：`high/medium/low`
```

**快速规则**用于启动时快速判断，避免立即加载完整文件。

## 提升记忆作用域

当某条记忆满足以下条件时，从单工具提升到 `shared/`：

1. 在两个工具的对话中都被命中过 ≥2 次
2. 反馈内容对两个工具都适用
3. 用户明确表示"这是通用规则"

**提升流程**：

```bash
# 1. 移动文件
mv ~/.agents/memory/claude/feedback_xxx.md ~/.agents/memory/shared/

# 2. 更新 frontmatter
# scope: claude → scope: [claude, codex]

# 3. 更新索引
# 在 MEMORY.md 中从 "Claude 专属" 移到 "Shared"
```

## 检查记忆加载

```bash
# 测试高优先级加载（启动时默认）
~/.agents/scripts/load-memory.sh claude

# 测试中优先级加载
~/.agents/scripts/load-memory.sh claude medium

# 测试全量加载
~/.agents/scripts/load-memory.sh claude all

# Codex 侧
~/.agents/scripts/load-memory.sh codex
```

**预期输出**：
```
📚 Loading claude memory (priority: high)...
  ✓ Index loaded: ~/.agents/memory/MEMORY.md
  ✓ Shared: feedback_skill_creation.md [high]

Summary: 1 shared + 0 tool-specific = 1 files loaded
```

## 当前记忆索引

所有记忆列表见：`~/.agents/memory/MEMORY.md`

## 三大禁忌

1. ❌ **不要在 `shared/` 放单工具偏好**  
   会导致 over-share：Claude 的简洁偏好污染 Codex 的详细输出

2. ❌ **不要在两个工具目录各存一份相同反馈**  
   会导致 under-share：维护两份，容易不同步

3. ❌ **不要省略 frontmatter `scope` 字段**  
   无法在数据层隔离，只能依赖提示层（不可靠）
