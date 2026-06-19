# 全局协作规范

## 0. AGENTS 层级归属
- `~/.agents/AGENTS.md` 是共享全局入口，只放跨项目、跨工作区、跨 agent 都必须生效的总则和路由。
- `~/.codex/AGENTS.md`、`~/.claude/AGENTS.md` 默认只作为 agent 入口层，应回链到 `~/.agents/AGENTS.md`，不要各自维护一份正文。
- `~/.codex/CLAUDE.md`、`~/.claude/CLAUDE.md` 默认保持 `CLAUDE.md -> AGENTS.md` 软链接，作为各入口目录内的发现别名。
- `~/AGENTS.md` 只用于在 home 目录本身工作时的轻量说明，不承载全局规则，也不复制本文件内容。
- `~/workspace/projects/AGENTS.md` 只约束 workspace 根目录和非 Git 子目录；进入具体 Git 项目后，项目根 `AGENTS.md` 才是项目规则入口。
- 项目内 `AGENTS.md` 只写该项目的事实、命令、架构边界、文档归属和覆盖规则；不要重复全局沟通、Git、安全和输出风格规则。
- 更深层目录的 `AGENTS.md` 只补充目录局部规则；不要回写与上层相同的通用约束。
- 共享 skill、共享 harness、共享 agents 的主体默认放在 `~/.agents/` 下维护，例如 `~/.agents/skills/`、`~/.agents/harness/`、`~/.agents/codex-agents/`、`~/.agents/claude-agents/`。
- `~/.codex/skills/`、`~/.codex/agents/`、`~/.claude/skills/`、`~/.claude/agents/` 默认只作为入口或软链接层；除非某项内容明确只属于单一 agent，否则不要在这些入口层分别维护一份正文。
- 当共享 skill / harness / agent 需要修改时，默认直接修改 `~/.agents/` 下的主体，再检查各入口软链接是否仍然正确；不要先改 `.codex` 再回填。

## 1. 全局总则
- 默认使用中文沟通；代码、命令、路径、报错、协议字段保持原样。
- 回答先给结论，再给依据和关键细节，避免空泛铺垫。
- 先读代码、配置、文档和上下文，再下结论；不要基于猜测直接实现。
- 先服从项目既有风格和模式，再判断最小可行改动（minimal diff）；但是不要把“少改当前文件”误当成 minimal diff。
- 开发阶段默认不为旧数据、旧接口、旧参数、旧状态做向后兼容；只有用户明确要求兼容，或当前任务的完成标准明确要求兼容时，才添加兼容逻辑。
- 完成修改后优先做与本次改动直接相关的最小充分验证；没有执行就不要声称“已验证”或“已通过”。
- 停下来问用户只有一种合法情况：存在真正的歧义，继续工作会产出与用户意图相反的成果。
- 高风险操作默认先说明影响、风险和回退方式，再等待确认。
- 每个任务开始时必须主动检查是否存在适用的 skill 或专门 agent 流程；若没有更具体入口，默认先从 `using-superpowers` 开始。
- 以下专题文档是本文件的延伸正文；命中对应场景时，必须继续读取对应文档，而不是只看本文件摘要。

## 2. 专题路由
- 协作基线、工作哲学、提问边界：
  [docs/agent-rules/collaboration-basics.md](/home/azhi/.agents/docs/agent-rules/collaboration-basics.md)
- 验证、运行安全、后台进程和大范围扫描：
  [docs/agent-rules/runtime-safety.md](/home/azhi/.agents/docs/agent-rules/runtime-safety.md)
- 本机固定工具路径与环境事实：
  [docs/agent-rules/local-environment.md](/home/azhi/.agents/docs/agent-rules/local-environment.md)
- Git、worktree、提交、推送和 shared workflow 同步：
  [docs/agent-rules/git-workspace.md](/home/azhi/.agents/docs/agent-rules/git-workspace.md)
- 文档归属、专题规则沉淀和结构性改动流程：
  [docs/agent-rules/documentation-boundaries.md](/home/azhi/.agents/docs/agent-rules/documentation-boundaries.md)
- 高风险操作确认细则：
  [docs/agent-rules/risky-operations.md](/home/azhi/.agents/docs/agent-rules/risky-operations.md)
- 默认输出风格：
  [docs/agent-rules/output-style.md](/home/azhi/.agents/docs/agent-rules/output-style.md)

## 3. 记忆系统
- 记忆系统入口和索引：
  [memory/MEMORY.md](/home/azhi/.agents/memory/MEMORY.md)
- 记忆系统设计和作用域规则：
  [memory/README.md](/home/azhi/.agents/memory/README.md)
- 每次任务开始前，根据当前工具加载对应作用域的记忆：
  - Claude: `memory/shared/` + `memory/claude/`
  - Codex: `memory/shared/` + `memory/codex/`

## 4. 文档入口
- `docs/documentation-rules.md` 是 `~/.agents/docs/` 的归属规则源：
  [docs/documentation-rules.md](/home/azhi/.agents/docs/documentation-rules.md)
- `docs/README.md` 是 `~/.agents/docs/` 的导航索引：
  [docs/README.md](/home/azhi/.agents/docs/README.md)
