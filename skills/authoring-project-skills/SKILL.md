---
name: authoring-project-skills
description: >
  Use when creating, restructuring, or reviewing a project-level skill that
  lives in a repo's `.agents/skills/` and routes across multiple rule, reference,
  or workflow files — especially when a SKILL.md has grown into a long inline
  wall, when descriptions fail to trigger, when traps keep getting missed, or
  when the skill should self-evolve from captured lessons. Also use when deciding
  whether a skill should stay a single file or become a folder, and how its
  SKILL.md should route. For the TDD process of testing a skill with subagents,
  use `writing-skills`; for repo-wide AGENTS.md / hooks harness, use
  `harness-engineering`.
---

# Authoring Project Skills

把一个项目 skill 写好的**结构与演化范式**导航中心。本文件只路由:判断该读哪个
reference,再按需读。详细规范在 `references/`,不在此内联。

来源:LINUX DO《如何写一个好的 skill》+ 开源项目 `WoJiSama/skill-based-architecture`。

三句核心:
- **结构服务于内容** —— 按需引入目录层级,不用结构撑完整性。
- **激活优于存储** —— 经验必须出现在 Agent 的任务路径上,只躺在 references 里不算捕获。
- **结构可复用,内容禁止预制** —— 脚手架可复制,项目特定内容必须留空待填。

## Use When
- 新建项目 skill,或一个 SKILL.md 已长成长篇内联规则墙需要重构为导航中心。
- description 命不中、该触发时没触发。
- 同类坑反复踩,但 skill 里没有让 Agent 在任务路径上读到它的地方。
- 需要让 skill 随项目演化(录入新经验、清退过时规则)。
- 判断一个 skill 该单文件还是文件夹,SKILL.md 该怎么路由。

## Do Not Use
- 测试 skill 是否真的生效(subagent 压力/检索测试、RED-GREEN-REFACTOR)→ `writing-skills`。
- 仓库级 harness(AGENTS.md 导航、SessionStart/PreToolUse hook、机械检查、CI 护栏)→ `harness-engineering`。
- 可复用项目代码模板 / 项目 `AGENTS.md` 脚手架 → `project-template`。
- 把跨项目通用方法误塞进项目 skill —— 通用的应进对应全局 skill,不在项目层长期复制正文。

## Reference Map
| 任务涉及 | 读这个 |
|---|---|
| 单文件 vs 文件夹、SKILL.md 导航中心解剖、rules/references/workflows 三分、文件大小信号 | `references/skill-structure.md` |
| 触发式 description、命名、关键词覆盖、各工具入口一致 | `references/description-and-triggers.md` |
| 原则+检验句(✓Check)、约束而非硬编码、不写显而易见、一个好例子 | `references/writing-rules.md` |
| 三级渐进加载、激活优于存储、坑点一句话+锚点、2/3 录入标准、泛化、录入位置、防日记本、规则清退 | `references/progressive-disclosure-and-evolution.md` |
| 一个 skill 一件事、primary/Do Not Use 边界、skill 组合三模式 | `references/isolation-and-composition.md` |
| 薄壳、SessionStart/PreToolUse hook、三层防失忆、skill 校验(smoke-test/test-trigger)（harness 层，非 skill 手艺） | 读本项目实际的薄壳/hook/路由 → 当前项目 `AGENTS.md` + `harness/` + 项目 hook 配置（`.codex/hooks.json` 等）；学"怎么搭/怎么校验" → `harness-engineering` skill |

## Common Tasks
- 新建项目 skill → 读 `skill-structure.md`(先单文件,命中文件夹化信号再拆)+ `description-and-triggers.md`。
- 重构超长 SKILL.md → 读 `skill-structure.md`(导航中心解剖),把规则下沉到 rules/references,SKILL.md 只留路由。
- description 命不中 → 读 `description-and-triggers.md`,改成触发条件式。
- 录入新经验 / 清退旧规则 → 读 `progressive-disclosure-and-evolution.md`(2/3 标准 + 录入位置表)。
- 保持 skill 聚焦 / 组合 → 读 `isolation-and-composition.md`。
- 让规则跨工具抗压缩、机械校验 skill(薄壳 + hook + smoke-test/test-trigger)→ harness 层:**读本项目**看当前 `AGENTS.md` + `harness/` + 项目 hook;**学怎么搭/校验**见 `harness-engineering`。
- 不在列表内 → 先读本文件 Reference Map,再按主题匹配 reference。

## Known Gotchas(命中即停)
- SKILL.md 当百科:把全部规则内联 → Agent 每次读完整本书。导航中心 ≤100 行,规则下沉。见 `skill-structure.md`。
- description 写成摘要/功能说明 → 命不中,且 Agent 可能照 description 抄近路跳过正文。见 `description-and-triggers.md`。
- 坑点只躺 references 里 → 未来 Agent 走任务路径读不到 = 没生效。见 `progressive-disclosure-and-evolution.md`。
- 把会话日志/调试过程当经验写进 references → skill 变日记本。见 `progressive-disclosure-and-evolution.md`。
- 从 GitHub 拉一堆同类 skill → 触发冲突、命中率下降。少而精,一个 skill 一件事。见 `isolation-and-composition.md`。

## ✓Check(收尾自查)
- SKILL.md 是否 ≤100 行且只讲"读什么/何时读"?
- description 是否只写触发条件、覆盖用户多种说法、不含流程摘要?
- 每条规则是否带可执行的 ✓Check(命令或可自问的具体问题)?
- 高代价坑点是否同时出现在任务路径上(SKILL.md Known Gotchas / workflow 完成检查)?
- Reference Map 引用的文件是否都真实存在、无孤儿?
- 是否已按 `writing-skills` 做最小检索/触发验证?
