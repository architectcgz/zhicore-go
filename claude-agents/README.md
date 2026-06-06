`~/.agents/claude-agents/` 是 Claude agent 定义的主体目录。

入口约定：

- `~/.claude/agents -> ~/.agents/claude-agents`

说明：

- 这里保留 Claude agent 的原生 Markdown 定义
- 不要与 Codex agent 原生定义目录直接合并
- 如果后续需要共享语义，应通过生成或桥接同步，而不是共用同一份原生文件
