#!/usr/bin/env python3
"""Strict-reference harness initializer profile."""

from __future__ import annotations

from pathlib import Path

from .consistency_content import check_script
from .docs_content import strict_docs
from .profile_common import quick_routing_shell, write_common_scaffold
from .scaffold import HARNESS_ROOT, ensure_documentation_scaffold, harness_dir, insert_or_replace, write


def configure_strict_reference(repo: Path, project_name: str, profile: str) -> tuple[str, str]:
    root = harness_dir(repo)
    ensure_documentation_scaffold(repo)
    for relative, content in strict_docs(project_name, profile).items():
        write(root / relative, content)
    write_common_scaffold(repo, profile, check_script())
    insert_or_replace(
        repo / "AGENTS.md",
        "root-navigation",
        f"""## Harness Engineering 学习档案

严格参考 `deusyu/harness-engineering` 的顶层结构：

| 目录 | 内容 | 说明 |
|------|------|------|
| `{HARNESS_ROOT}/concepts/` | AGENTS 补充 | 补充项目 `AGENTS.md`，记录长期概念、原则和 harness 定义 |
| `{HARNESS_ROOT}/thinking/` | 独立思考 | 对项目 harness 边界和取舍的判断 |
| `{HARNESS_ROOT}/practice/` | 动手实践 | 初始化和后续实验记录 |
| `{HARNESS_ROOT}/feedback/` | 反馈记录 | 踩坑、修正和可复用经验 |
| `{HARNESS_ROOT}/works/` | 作品输出 | 可展示模板、报告和说明 |
| `{HARNESS_ROOT}/prompts/` | 提示词积累 | 已验证提示词和工作流 |
| `{HARNESS_ROOT}/references/` | 外部资源 | 文章、仓库和工具索引 |
| `{HARNESS_ROOT}/docs/architecture/` | 架构事实 | 当前系统设计、边界和长期技术约束 |

项目根保持 `CLAUDE.md -> AGENTS.md`，让 Claude / Codex 使用同一份入口规则。

机械化检查：`bash {HARNESS_ROOT}/scripts/check-harness-consistency.sh`。
架构守卫入口：`bash {HARNESS_ROOT}/scripts/check-architecture.sh`。""",
    )
    insert_or_replace(
        repo / "AGENTS.md",
        "quick-routing",
        quick_routing_shell(),
    )
    insert_or_replace(
        repo / "AGENTS.md",
        "todo-reminder",
        f"""## Todo Reminder

开始新任务前，先运行 `bash {HARNESS_ROOT}/scripts/check-open-todos.sh --quiet-if-empty`，先过一遍 `{HARNESS_ROOT}/docs/todo/` 里的未完成事项；如果命中当前主题，首条回复先提醒。已完成但还没归档的 todo 也会在这里提示。""",
    )
    insert_or_replace(
        repo / "AGENTS.md",
        "test-workflow",
        f"""## Test Workflow

如果仓库存在自动化测试或明显测试面，先写或更新最小相关测试，再进入实现。

- Write or update the narrowest relevant tests first.
- After changing tests, run the smallest relevant test command that covers the touched surface.
- After the test command, run the relevant script check such as `bash {HARNESS_ROOT}/scripts/check-test-workflow.sh` or `bash {HARNESS_ROOT}/scripts/check-harness-consistency.sh` before claiming completion.
- 如果当前仓库已经有 `{HARNESS_ROOT}/scripts/check-harness-consistency.sh`、git hooks 或 CI guardrail，测试相关脚本检查必须接入这些实际检查链路，不能只停留在提示词里。""",
    )
    insert_or_replace(
        repo / "README.md",
        "readme-harness",
        f"""## Harness Engineering

本项目按 `deusyu/harness-engineering` 建立顶层 harness 结构：

- `{HARNESS_ROOT}/concepts/`：项目 `AGENTS.md` 的补充，记录长期概念与原则
- `{HARNESS_ROOT}/thinking/`：独立思考
- `{HARNESS_ROOT}/practice/`：实践记录
- `{HARNESS_ROOT}/feedback/`：反馈闭环
- `{HARNESS_ROOT}/works/`：作品输出
- `{HARNESS_ROOT}/prompts/`：提示词积累
- `{HARNESS_ROOT}/references/`：外部资料

一致性检查：

```bash
bash {HARNESS_ROOT}/scripts/check-harness-consistency.sh
```

最小架构守卫：

```bash
bash {HARNESS_ROOT}/scripts/check-architecture.sh
```""",
    )
    hook_docs = f"""## Harness 检查

- `pre-commit`：运行 `{HARNESS_ROOT}/scripts/check-harness-consistency.sh`，其中会继续执行 `{HARNESS_ROOT}/scripts/check-architecture.sh` 与 `{HARNESS_ROOT}/scripts/check-test-workflow.sh`，检查严格参考 harness 的顶层目录、导航、最小架构守卫和测试工作流约束。
- `pre-commit`：非阻塞运行 `{HARNESS_ROOT}/scripts/check-skill-sync-reminder.sh --staged`，提醒把跨项目规则上收全局 skill 或 shared harness。
- `commit-msg`：运行 `{HARNESS_ROOT}/scripts/check-commit-message.sh`，由共享检查器读取 `{HARNESS_ROOT}/harness/policies/commit-message.json` 校验标题、正文和激活任务的 `Task:` 绑定。
- 原有 API 合同同步逻辑继续保留。"""
    return "Initialized strict-reference harness", hook_docs
