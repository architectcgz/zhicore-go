# ~/.agents/harness/prompts

本目录是跨项目共享 harness prompt 的正文 owner。

## 角色

- 这里保存可跨项目复用的 prompt 正文。
- 项目内 `harness/prompts/*.md` 可以保留稳定入口、局部参数、项目特化补充和仓库内交叉引用。
- 只对单一仓库有效、离开该仓库就失真的 prompt，继续留在项目内。
- 明显属于某个 skill 方法论的正文，例如 review 方法、架构分析方法、测试方法，优先放到对应 skill 下，而不是继续放在 harness prompt 里。

## 当前条目

- `architecture-diagram-generation.md`：根据事实源生成架构图输入包的共享模板。
- `coding-agent-system-prompt.md`：reuse-first coding agent 的共享模板。
