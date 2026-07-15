# 文档规则

## 目的

本文件定义本项目文档的归属、放置位置、编辑前阅读、路径登记和验证规则。

## 核心原则

- 文档是长期项目记忆，不是当前任务草稿。
- 每个文档应只有一个主要角色：当前事实、草案 / 设计、实施计划、review 证据、运维指南、外部参考、agent 反馈或 harness 资产。
- 入口文档负责路由读者，不重复长篇规则或事实源内容。
- 文档正文默认使用中文；代码标识、命令、路径、包名、API / 协议字段和外部专有名词保持原文。
- 文档变更必须和代码、contract、脚本、测试、架构边界和机械护栏保持同步。
- 如果稳定结论替代旧文档，应将旧文档标记为已废弃，或从活跃索引中移除。

## 避免循环引用 (No Circular References)

- 本文件是文档规则事实源。
- `docs/README.md` 是文档导航索引。
- 项目 `AGENTS.md` 可以路由到本文件和 `docs/README.md`，但不要重复完整规则。
- 不要让两个文档在编辑前互相要求读取，导致循环依赖。
- 编辑本文件时，先读当前文件，再按影响范围检查索引和引用。
- 编辑 `docs/README.md` 时，先读本文件，再检查链接和受影响目标文档；不要把 `docs/README.md` 当成编辑它自身的前置条件。

## 编辑前阅读协议 (Pre-Edit Reading Protocol)

创建、移动、删除或编辑文档前：

1. 先读本文件；如果当前任务正在创建本文件且文件尚不存在，可以跳过。
2. 读取目标区域最近的现有索引。仓库级导航优先使用 `docs/README.md`；如果不存在，则检查最近父目录和项目 `AGENTS.md`。
3. 读取被修改事实的当前事实源，例如代码、contract、配置、运维文档或 review 记录。
4. 搜索被新增、移动、重命名或删除路径的现有引用。

写入前先判断本次变更属于当前事实、草案 / 设计、实施计划、review 证据、运维指南、外部参考、agent 反馈还是 harness 资产。然后确认事实源 owner、最近索引、陈旧引用和必要机械检查。

写入前回答：

- 这个文档是什么类型？
- 当前事实源在哪里？
- 本次是在更新事实源、增加导航、归档历史，还是记录过程证据？
- 本次是否需要机械检查、脚本、hook、CI 或文档化的人工护栏？

## 新路径登记 (New Path Registration)

新增长期文档目录、活跃入口或文档类别时，在同一次变更中登记：

- 本文件。
- `docs/README.md` 或最近父级索引。
- agent 路由变化时的项目 `AGENTS.md`。
- 当路径必须保持稳定时，对应的机械检查，例如 `scripts/check-consistency.sh`、`scripts/check-docs-consistency.py`、CI 或等价 guard。

登记格式：

```md
路径：`...`
类型：当前事实 / 草案 / 计划 / review 证据 / 运维 / 外部参考 / 反馈 / harness 资产
负责人：...
是否活跃入口：是 / 否
允许内容：
禁止内容：
编辑前阅读：
验证方式：
```

不要为当前任务草稿创建长期文档路径。若项目已有 scratch 或 harness 位置，先放在那里；任务结束后删除草稿，或把长期知识提升到对应事实源。

如果文档规则形成稳定护栏，并且仓库已有对应脚本、hook、CI 或检查入口，同一次变更中应更新机械检查或明确人工检查。不要在已有 guardrail 路径的项目里只留下纯文本规则。

## 标准路径

- `docs/README.md`：文档入口、阅读顺序、当前事实源地图和陈旧文档规则。
- `docs/requirements/`：产品需求、范围定义、验收标准、用户故事、约束和需求差距分析。
- `docs/contracts/`：API contract、DTO / event schema、协议规格、payload 示例和兼容性说明。
- `docs/spec/`：实施计划前的可执行功能规格。
- `docs/design/`：产品设计、UI / UX 设计、设计系统说明、交互流程，以及尚未成为当前架构事实的视觉决策。
- `docs/todo/`：可执行任务列表、backlog 拆分、清理队列和未解决事项。
- `docs/architecture/`：当前系统设计、模块边界、数据流、依赖决策、ADR 风格记录、长期技术约束。
- `docs/plan/`：实施计划、迁移计划、发布计划、阶段性重构计划和临时执行计划。
- `docs/operations/`：runbook、本地运维、部署说明、故障处理、维护命令和运维验证记录。
- `docs/reviews/`：代码 review、架构 review、UI / UX review、审计快照和 review 发现。
- `docs/reports/`：状态报告、差距报告、实施总结、调查报告和限时分析输出。
- `docs/improvements/`：agent 发现的改进项和提升状态，不作为通用任务 backlog。
- `docs/refs/`：外部参考、研究笔记、源材料、供应商文档摘要、论文，以及不应直接视为项目决策的复制上下文。

## Review 证据规则

正式 review 证据必须明确被 review 的对象：

- 使用明确 commit hash、commit range、merge commit、PR 或等价不可变 artifact。
- 将 live worktree diff 视为草案或预检查输入；除非之后绑定到 commit 或 commit range，否则不要作为最终正式 review 对象。
- 多轮正式 review 应保存为独立文件，并带 `round-<n>` 后缀，例如 `2026-06-13-runtime-node-health-round-1.md` 和 `2026-06-13-runtime-node-health-round-2.md`。
- 不要用后续发现或修复覆盖早期 review 轮次。summary 或 index 可以链接所有轮次，但每一轮本身应作为不可变过程记录保留。
- 如果 blocker 修复改变了被 review 的代码，记录修复 commit 或 range，并对更新后的对象执行下一轮 review。

## 事实源规则

- 只有当 `docs/architecture/` 和 `docs/contracts/` 与代码和测试匹配时，才把它们视为当前技术事实。
- `docs/spec/` 是计划输入。
- `docs/plan/`、`docs/reviews/` 和 `docs/reports/` 是过程历史，除非稳定结论被提升到 architecture、contracts 或 requirements。
- 过程历史不会因为最新就自动成为当前事实。
- `docs/README.md` 负责说明哪些文档是当前事实，哪些是历史。
- 草案设计被采纳后，把稳定结论移动到 `docs/architecture/` 或 `docs/contracts/`。
- 文档被替代后，标记为 superseded，或从活跃索引移除。

## 验证

当文档路径、索引或事实发生变化时：

- 运行项目文档一致性检查（如果存在）。
- 搜索被重命名、移动或删除路径的陈旧引用。
- 从最近父级索引验证新增链接。
- 若新稳定路径没有机械检查，要么补充检查，要么明确说明为什么不需要。
