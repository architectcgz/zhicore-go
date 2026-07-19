# Skill Validation Checks

Read this file when adding mechanical validation for skills in a repo's harness — the "质检" element
that catches forgetting-type errors a human won't. This is harness (enforcement), not skill craft;
for how to *write* the skill being validated, see `authoring-project-skills`.

80% 的 skill 失败来自遗忘而非误解，脚本能抓住，应接进 harness 的检查链路(consistency check / hooks / CI)。

## smoke-test(防漏项)

把 SKILL.md 当**唯一数据源**自检，不另维护测试清单：
- Common Tasks / Reference Map 引用的文件是否都存在、有无孤儿。
- 行数是否超标(导航中心 ≤100 行等)。
- 占位符残留(`<!-- FILL -->` 是否未替换)、各工具入口(`.claude` / `.cursor` / `.codex`)description 是否一致、薄壳一致性。
- 你在 Common Tasks 加一条引用 `workflows/deploy.md` 的新任务，脚本应自动发现这个文件还不存在。

## test-trigger(防纸面合规)

从 Common Tasks / Quick Routing 生成用户可能的真实说法，测对应 skill 的 `description` 命中率：
- 单独读一遍 SKILL.md 觉得没问题，跑 test-trigger 才发现一半触发短语命不中。
- 本机实现:`harness/test-trigger-rate.py`(initializer 生成 `scripts/test-trigger-rate.sh`)。
- 占位/FILL 行豁免、不计入门禁;只对**项目自有** skill 把关，指向全局 skill 的标准行只报告不 gate。
- 连字符 skill 名（如 `backend-engineer`）按反引号整体提取，不可按 `-` 拆分。

## 跑的时机
初次写完、改完 SKILL.md 或薄壳后、从上游模板升级后、宣布"完成"前。

## 边界
脚本抓不到语义问题(description 是否够准、路由是否合理);那部分靠人，或 `writing-skills` 的
subagent 检索/压力测试。结构性拦截(Red Flags / Rationalizations / SessionStart hook)见
`references/thin-shell-and-hooks.md`。
