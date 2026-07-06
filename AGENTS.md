# 全局协作规范

## 0. AGENTS 层级归属
- `~/.agents/AGENTS.md` 是共享全局入口，只放跨项目、跨工作区、跨 agent 都必须生效的总则和路由。
- `~/.codex/AGENTS.md`、`~/.claude/AGENTS.md` 默认只作为 agent 入口层，应回链到 `~/.agents/AGENTS.md`，不要各自维护一份正文。
- `~/.codex/CLAUDE.md`、`~/.claude/CLAUDE.md` 默认保持 `CLAUDE.md -> AGENTS.md` 软链接，作为各入口目录内的发现别名。
- `~/.gemini/AGENTS.md` 是 Antigravity / Gemini 入口层，应回链到 `~/.agents/AGENTS.md`；`~/.gemini/GEMINI.md` 如存在，只作为 `AGENTS.md` 的兼容发现别名，不承载独立正文。
- `~/AGENTS.md` 只用于在 home 目录本身工作时的轻量说明，不承载全局规则，也不复制本文件内容。
- `~/workspace/projects/AGENTS.md` 只约束 workspace 根目录和非 Git 子目录；进入具体 Git 项目后，项目根 `AGENTS.md` 才是项目规则入口。
- 项目内 `AGENTS.md` 只写该项目的事实、命令、架构边界、文档归属和覆盖规则；不要重复全局沟通、Git、安全和输出风格规则。
- 更深层目录的 `AGENTS.md` 只补充目录局部规则；不要回写与上层相同的通用约束。
- 共享 skill、共享 harness、共享 agents 的主体默认放在 `~/.agents/` 下维护，例如 `~/.agents/skills/`、`~/.agents/harness/`、`~/.agents/codex-agents/`、`~/.agents/claude-agents/`。
- `~/.codex/skills/`、`~/.codex/agents/`、`~/.claude/skills/`、`~/.claude/agents/`、`~/.gemini/skills/` 默认只作为入口或软链接层；除非某项内容明确只属于单一 agent，否则不要在这些入口层分别维护一份正文。
- 当共享 skill / harness / agent 需要修改时，默认直接修改 `~/.agents/` 下的主体，再检查各入口软链接是否仍然正确；不要先改 `.codex` 再回填。

## 1. 全局总则
- 默认使用中文沟通；代码、命令、路径、报错、协议字段保持原样。
- 新项目初始化、README、AGENTS.md、docs、部署说明和长期 Markdown 正文默认使用中文；代码标识、命令、路径、包名、API / 协议字段和外部专有名词保持原文。
- 回答先给结论，再给依据和关键细节，避免空泛铺垫。
- 先读代码、配置、文档和上下文，再下结论；不要基于猜测直接实现。
- 先服从项目既有风格和模式，再判断最小可行改动（minimal diff）；但是不要把“少改当前文件”误当成 minimal diff。
- 新增或修改代码时必须为关键业务逻辑补充注释，方便人类维护者理解意图；注释应解释业务规则、领域不变量、状态转换、权限判断、幂等/补偿、跨服务副作用或非显然取舍，避免只复述语法或实现步骤。
- 开发阶段默认不为旧数据、旧接口、旧参数、旧状态做向后兼容；只有用户明确要求兼容，或当前任务的完成标准明确要求兼容时，才添加兼容逻辑。
- TDD 只对行为、状态、数据流、权限、校验、异步流程、算法、接口契约或可复现 bug 等逻辑承载改动默认适用；简单 UI / 展示层改动（布局、间距、颜色、排版、静态文案、控件位置迁移且不改变状态语义）不默认使用 TDD，改用最小充分的类型、构建、组件渲染、截图或人工验证。
- 完成修改后优先做与本次改动直接相关的最小充分验证；没有执行就不要声称“已验证”或“已通过”。
- 前端任务不要为了“给一个预览地址”默认启动 dev server；只有用户要求、需要浏览器/截图/真实交互验证、或测试与类型检查无法覆盖关键 UI 风险时才启动。
- 调用前端实现、UI/UX 设计、页面原型、视觉 polish 类 subagent 时，必须使用 `gpt-5.5`；不要选择工具说明中模型被固定为 `gpt-5.4` 且不可覆盖的内置角色。若当前工具无法确认或无法满足 `gpt-5.5`，先停止调度并向用户说明限制。
- 停下来问用户只有一种合法情况：存在真正的歧义，继续工作会产出与用户意图相反的成果。
- 高风险操作默认先说明影响、风险和回退方式，再等待确认。
- 每个任务开始时必须主动检查是否存在适用的 skill 或专门 agent 流程；只有用户点名或任务明确命中 description 时才读取对应 skill，不设 `superpowers` / `using-superpowers` 默认兜底。
- 创建 `git worktree` 时，worktree 路径一律放到对应项目目录下；不要放到 `/tmp`、`~/.codex`、`~/.agents` 或其他脱离项目归属的位置。
- 任何 `git commit` 前必须先走 `committing-changes` skill 核对并组织提交信息（类型 + 中文描述、最小可审阅拆分、默认禁止 `Co-Authored-By`、按仓库 commit policy/hook 叠加），不得跳过；该 skill 是提交约定的唯一事实源。
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
