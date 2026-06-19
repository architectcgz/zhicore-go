# 记忆自动加载配置

## 当前状态

截至 2026-06-19，记忆系统已迁移到 `~/.agents/memory/` 三层结构，但 **自动加载机制尚未完全配置**。

## 记忆加载规则

### Claude Code 侧

当前 Claude Code 使用 **auto memory** 机制（system prompt 内置），默认加载：
```
~/.claude/projects/<project-hash>/memory/
```

**新增需求**：需要让 Claude Code 同时加载：
- `~/.agents/memory/shared/*.md`（共享记忆）
- `~/.agents/memory/claude/*.md`（claude 专属记忆）

### Codex 侧

Codex 当前没有内置 auto memory，需要手动配置加载机制。

**需要加载**：
- `~/.agents/memory/shared/*.md`（共享记忆）
- `~/.agents/memory/codex/*.md`（codex 专属记忆）

## 技术方案

### 方案 A：通过 AGENTS.md 引用（当前临时方案）

在 `~/.agents/AGENTS.md` 添加：

```markdown
## 记忆系统

- 共享记忆索引：[~/.agents/memory/MEMORY.md](/home/azhi/.agents/memory/MEMORY.md)
- 记忆系统说明：[~/.agents/memory/README.md](/home/azhi/.agents/memory/README.md)

每次任务开始前，根据当前工具加载对应作用域的记忆：
- Claude: `shared/` + `claude/`
- Codex: `shared/` + `codex/`
```

**优点**：立即生效，无需修改配置文件  
**缺点**：依赖 agent 主动读取，不是真正的"自动加载"

### 方案 B：通过 hooks 配置（推荐但需工具支持）

**Claude Code (`~/.claude/settings.json`)**:

```json
{
  "hooks": {
    "user-prompt-submit": "~/.agents/scripts/load-memory.sh claude"
  }
}
```

**Codex (`~/.codex/config.toml`)**:

```toml
[hooks]
user_prompt_submit = "~/.agents/scripts/load-memory.sh codex"
```

**优点**：每次用户提交 prompt 前自动注入记忆路径  
**缺点**：需要工具支持 hooks 机制

### 方案 C：扩展 claudeMd 上下文（Claude Code 专用）

利用 Claude Code 的 `claudeMd` 上下文加载机制，在项目级 `CLAUDE.md` 引用全局记忆：

```markdown
# 全局记忆

{{read ~/.agents/memory/MEMORY.md}}
{{read ~/.agents/memory/shared/*.md}}
{{read ~/.agents/memory/claude/*.md}}
```

**优点**：利用现有机制  
**缺点**：需要确认 Claude Code 是否支持动态文件引用

## 当前实施方案

**阶段 1（已完成）**：目录结构迁移
- ✅ 创建 `~/.agents/memory/{shared,claude,codex}/`
- ✅ 迁移现有记忆并标注 scope
- ✅ 建立 README 和索引

**阶段 2（当前临时方案）**：通过 AGENTS.md 引用
- ⏳ 在 `~/.agents/AGENTS.md` 添加记忆索引引用
- ⏳ 依赖 agent 在任务开始前主动读取

**阶段 3（待实施）**：配置真正的自动加载
- ⬜ 测试 Claude Code 和 Codex 的 hooks 支持
- ⬜ 根据测试结果选择方案 B 或 C
- ⬜ 编写自动化脚本验证加载成功

## 辅助工具

**记忆加载测试脚本**：`~/.agents/scripts/load-memory.sh`

```bash
# 测试 claude 记忆加载
~/.agents/scripts/load-memory.sh claude

# 测试 codex 记忆加载
~/.agents/scripts/load-memory.sh codex
```

## 待解决问题

1. Claude Code 的 auto memory 机制能否配置多个加载路径？
2. Codex 是否支持类似 Claude Code 的 auto memory？
3. hooks 机制的输出如何注入到 system prompt？

## 相关文件

- 记忆系统设计：`~/.agents/memory/README.md`
- 记忆索引：`~/.agents/memory/MEMORY.md`
- 全局规则入口：`~/.agents/AGENTS.md`
