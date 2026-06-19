# Skill Creation Helper

## Quick Start

创建新 skill 并自动设置软链接：

```bash
~/.agents/scripts/create-skill.sh <skill-name>
```

## What It Does

1. ✓ 在 `~/.agents/skills/<skill-name>/` 创建目录
2. ✓ 生成 `SKILL.md` 模板
3. ✓ 自动创建 `~/.claude/skills/<skill-name>` 软链接
4. ✓ 自动创建 `~/.codex/skills/<skill-name>` 软链接
5. ✓ 避免循环软链接问题

## Example

```bash
# Create a new skill
~/.agents/scripts/create-skill.sh my-awesome-skill

# Output:
# ✓ Created skill directory: /home/azhi/.agents/skills/my-awesome-skill
# ✓ Created SKILL.md template
# ✓ Created symlink for Claude
# ✓ Created symlink for Codex
# ✅ Skill 'my-awesome-skill' created successfully!
```

## After Creation

1. **Edit the skill**:
   ```bash
   vim ~/.agents/skills/my-awesome-skill/SKILL.md
   ```

2. **Commit to git**:
   ```bash
   cd ~/.agents
   git add skills/my-awesome-skill/
   git commit -m "Add: my-awesome-skill"
   git push
   ```

3. **Test the skill**:
   - Claude: 说 "use my-awesome-skill" 或相关触发条件
   - Codex: 同样触发方式

## Skill Template Structure

生成的模板包含：

```markdown
---
name: skill-name
description: Brief description
---

# Skill Name

## When to Use
- Condition 1
- Condition 2

## Workflow
### Step 1
### Step 2

## Example
## Related Skills
```

## Troubleshooting

### 问题：skill 已存在
**错误**：`Error: Skill 'xxx' already exists`

**解决**：
```bash
# 删除旧 skill
rm -rf ~/.agents/skills/xxx
rm ~/.claude/skills/xxx
rm ~/.codex/skills/xxx

# 重新创建
~/.agents/scripts/create-skill.sh xxx
```

### 问题：软链接断开
**症状**：`Too many levels of symbolic links`

**解决**：
```bash
# 检查是否循环链接
stat ~/.agents/skills/xxx
stat ~/.claude/skills/xxx

# 如果是循环链接，删除并重建
unlink ~/.agents/skills/xxx  # 删除循环链接
mkdir ~/.agents/skills/xxx   # 创建真实目录
~/.agents/scripts/create-skill.sh xxx  # 会失败，但可以手动创建软链接
```

## Directory Structure

```
~/.agents/
├── skills/
│   └── my-skill/           # 真实目录
│       └── SKILL.md        # Skill 定义
└── scripts/
    └── create-skill.sh     # 创建脚本

~/.claude/skills/my-skill   → ~/.agents/skills/my-skill (软链接)
~/.codex/skills/my-skill    → ~/.agents/skills/my-skill (软链接)
```

## Best Practices

1. **命名规范**：使用 `kebab-case`（小写 + 连字符）
2. **描述清晰**：在 `description` 明确说明触发条件
3. **分步骤**：Workflow 部分结构化，方便 agent 执行
4. **提供示例**：包含真实可运行的命令示例
5. **关联其他 skill**：建立 skill 之间的引用关系

## Related

- `~/.agents/skills/README.md` - Skills 目录说明
- `~/.agents/skills/feynman-note/` - 实际案例参考
