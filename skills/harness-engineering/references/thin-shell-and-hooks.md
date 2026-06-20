# Thin Shell and Anti-Amnesia Hooks

Read this file when wiring a repo's entry files (`AGENTS.md` / `CODEX.md` / `.cursor`) and hooks so
that agent discipline survives long sessions, context compaction, and `/clear`. This is harness
wiring; for how to write the skills these route to, see the `authoring-project-skills` skill.

## 三层防失忆

长会话 + 多任务 + 多次 compact 下，自然语言指令（"去读 AGENTS.md / SKILL.md"）会被压缩丢掉。
靠三层结构化冗余兜底，每层都可能被压缩器丢掉，留给你下一层：

1. **入口文件的 Session Discipline / Auto-Triggers** —— 每个新任务重走路由（即使是同会话第 N 轮）。
2. **薄壳路由表** —— 结构化表格，压缩后比自然语言保留更多。
3. **SessionStart hook** —— `/clear`、compact 后从磁盘自动重注入入口/SKILL.md。

三者叠加才能扛住长会话 + 多任务 + 多次 compact。

## 薄壳(thin shell)

让规则在 Claude Code / Cursor / Codex / Gemini 多工具生效，不是把正文复制 N 份，而是在每个工具的
入口文件里内联一层薄壳（≤60 行）：把**最小可执行路由表**内联进去，压缩后表格仍活着。

薄壳三块（缺一不可）：
- **Quick Routing** —— `任务 | 必读文件 | workflow/skill` 三列，必须有兜底行 Other 和多子任务行。
  压缩后这是 Agent 找"该读哪些文件 / 用哪个 skill"的唯一线索。
- **Auto-Triggers** —— 事件→动作；最关键是"同会话新任务 → 重读入口、重走路由"。
- **Red Flags — STOP** —— 把"就这一次跳过"这类借口前置拦截（压缩后只剩薄壳时的最后防线）。

反例（soft-pointer-only）：只写 "Please read AGENTS.md/SKILL.md before starting"。长会话 compact 后
这句自然语言被摘要掉，Agent 没路由表可查，凭感觉动手，用户察觉不到。**用结构化表格，不要只靠一句指针。**

占位起步：项目刚 init 还没有真实薄壳时，生成**标准结构 + `<!-- FILL -->` 占位行**
（结构可复用、内容禁止预制），起步后逐行补充为真实必读文件与 skill。

## hook(机制级护栏，不靠 Agent 自觉)

- **SessionStart hook** —— 监听 startup / clear / compact，自动读项目入口薄壳或主 SKILL.md 注入 context（防遗忘）。
- **PreToolUse hook** —— 在 Edit 核心规则文件前拦一刀，非 0 退出码直接取消这次 Edit（防违规）。
- 注意：
  - hook schema 各工具不同（例如 Claude Code v2.1+ 的 PreToolUse 只认嵌套 `hooks:[{type,command}]`，
    flat 写法注册时不报错、debug 能看到名字，但 Edit 时静默不触发）。
  - hook 只能一定程度缓解；过多 hook 反而限制 Agent 发挥，复杂问题建议用 Sonnet 及以上模型。
  - 不要在全局 SessionStart 注入完整 skill 索引；description 负责触发，SKILL.md / references 按需加载。
- 本机 Codex 侧当前不启用默认 SessionStart 注入。若具体项目需要防遗忘 hook，只注入该项目的薄壳路由或
  `primary` 项目 SKILL.md 摘要，不回退展开全局 `~/.agents/skills`。

Codex 项目级可复用脚本：`~/.agents/harness/hooks/session-thin-shell.sh`。

项目 `.codex/hooks.json` 示例：

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup|resume|clear|compact",
        "hooks": [
          {
            "type": "command",
            "command": "bash ~/.agents/harness/hooks/session-thin-shell.sh",
            "statusMessage": "Reloading project routing"
          }
        ]
      }
    ]
  }
}
```

默认读取项目根 `AGENTS.md`，优先抽取以下精确标记块：

```markdown
<!-- codex-session-thin-shell:start -->
## Quick Routing
...
## Auto-Triggers
...
## Red Flags
...
<!-- codex-session-thin-shell:end -->
```

没有标记块时，脚本只抽取 `Quick Routing`、`Session Discipline`、`Auto-Triggers`、
`Red Flags` 等短路由章节；仍然找不到时才截取入口文件开头。需要改入口文件时，用
`CODEX_SESSION_THIN_SHELL_SOURCE=path/to/file.md`。

## 边界

本文件只管"把规则接进仓库入口与 hook、让它抗失忆"。怎么**写好**这些 skill 本身
（description 触发、SKILL.md 导航中心、渐进加载、录入标准、组合）属于 skill 手艺，见
`authoring-project-skills`。
