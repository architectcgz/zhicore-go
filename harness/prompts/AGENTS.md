# ~/.agents/harness/prompts

本目录是跨项目共享 harness prompt 的正文 owner。

## 角色

- 这里保存可跨项目复用的 prompt 正文。
- 项目内 `harness/prompts/*.md` 可以保留稳定入口、局部参数、项目特化补充和仓库内交叉引用。
- 只对单一仓库有效、离开该仓库就失真的 prompt，继续留在项目内。
- 明显属于某个 skill 方法论的正文，例如 review 方法、架构分析方法、测试方法，优先放到对应 skill 下，而不是继续放在 harness prompt 里。

## 当前条目

- `architecture-diagram-generation.md`：根据事实源生成架构图输入包的共享模板。

## 共享 prompt 写作原则

- Prompt / 规则 / 反馈正文默认先抽象问题类别，再用本次事故或代码片段作反例；不要把一次现象直接写成只适用于本次的指令。
- 写作顺序：现象 → 本质问题类别 → 可复用规则 → 反例 / 正例 → 检查句。缺少“本质问题类别”时，先不要沉淀为共享 prompt。
- 反例只用于帮助识别模式，不作为规则主语。规则主语应覆盖同类问题，例如“构造期隐藏外部依赖读取”，而不是“runtime-agent hostname 读取”。
- 如果规则只能套用到一个仓库、一个文件或一次 review，它应留在项目 feedback / plan / review 中，不进入共享 prompt 正文。
