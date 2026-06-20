# Skill Isolation and Composition

Read this file for multi-skill craft: keeping skills focused, and composing one skill from another.
Two harness concerns are NOT here: making skills survive long sessions (thin shell, SessionStart /
PreToolUse) and mechanically validating skills (smoke-test / test-trigger) both belong to the
`harness-engineering` skill.

## 一个 skill 干一件事(隔离)

- 不要从 GitHub 拉一堆同类 skill —— 冲突必现，命中率下降。少而精，优先自动触发而非主动引用。
- 多 skill 项目：每个 skill 独立 SKILL.md；跨 skill 通用约定放 `shared/`；
  用 frontmatter `primary: true` 标默认 skill；不同领域保持独立，合并只会让 description 变"万金油"。
- 每个 skill 用 **Do Not Use** 明确边界，把不属于自己的任务指向正确的 skill。
- 该裂成多 skill 的信号：两领域 Common Tasks 完全不相交；description 要列 10+ 跨领域触发短语；
  gotchas 自然分成两半。

## skill 组合(composition)

一个 skill 的 workflow 可以"外包"一段工作给另一个 skill。三种模式：
- **A 嵌入调用**：workflow 某步显式 `Read skills/<other>/SKILL.md`，跟完其某条路由再返回。
- **B 直接路由**：Common Tasks 某类任务直接指向另一个 skill 的 workflow，不写自己的包装。
- **C 子 Agent 委派**：开干净子 Agent 整个隔离执行，只回结构化结果。

反模式：
- 隐式传递依赖（调的 skill 下游不存在 → 静默失败）→ 要么 vendor 一份，要么加"缺 skill 就停下问用户"。
- 循环组合（A 调 B 调 A）。
- 匿名调用（不写具体路径 → 失去复现性）→ 永远写具体 skill 路径。
- 组合当偷懒借口；调完别人就跳过自己的 AAR。
