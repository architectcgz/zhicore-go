#!/usr/bin/env python3
"""CTF-current harness initializer profile."""

from __future__ import annotations

from pathlib import Path

from .consistency_content import ctf_current_check_script
from .docs_content import ctf_current_docs
from .profile_common import quick_routing_shell, write_common_scaffold
from .scaffold import HARNESS_ROOT, ensure_documentation_scaffold, harness_dir, insert_or_replace, write


def configure_ctf_current(repo: Path, project_name: str, profile: str) -> tuple[str, str]:
    root = harness_dir(repo)
    ensure_documentation_scaffold(repo)
    for relative, content in ctf_current_docs(project_name, profile).items():
        write(root / relative, content)
    write_common_scaffold(repo, profile, ctf_current_check_script())
    insert_or_replace(
        repo / "AGENTS.md",
        "root-navigation",
        f"""## Harness Engineering

当前默认采用 CTF 探索版 harness 形态，并保留 `deusyu/harness-engineering` 的核心原则作为重要参考。

| 路径 | 内容 | 说明 |
|------|------|------|
| `{HARNESS_ROOT}/state/` | 当前任务状态 | 只保存短期执行证据和当前 reuse 决策 |
| `{HARNESS_ROOT}/state/reuse-index/` | 本地私有索引 | 用户自用的长期复用线索，默认 gitignore，`index.yaml` + 镜像 `README.md` |
| `{HARNESS_ROOT}/harness/policies/` | 项目策略 | 可被检查脚本读取的本地规则 |
| `{HARNESS_ROOT}/harness/templates/` | 模板 | 当前项目重复使用的决策或记录模板 |
| `{HARNESS_ROOT}/harness/prompts/` | Prompt 入口 | 仓库内稳定入口、局部补充，以及仍然项目专属的 prompt |
| `{HARNESS_ROOT}/harness/checks/` | 检查脚本 | 机械化一致性和规则检查 |
| `{HARNESS_ROOT}/feedback/` | 反馈记录 | 踩坑、修正和可复用流程经验 |
| `{HARNESS_ROOT}/docs/documentation-rules.md` | 文档规范 | 改文档前置读取与新增路径登记 |
| `{HARNESS_ROOT}/docs/README.md` | 文档索引 | 当前事实源地图和文档阅读顺序 |
| `{HARNESS_ROOT}/docs/architecture/` | 架构事实 | 当前系统设计、边界和长期技术约束 |

项目根保持 `CLAUDE.md -> AGENTS.md`，让 Claude / Codex 使用同一份入口规则。

机械化检查：`bash {HARNESS_ROOT}/scripts/check-harness-consistency.sh`。
架构守卫入口：`bash {HARNESS_ROOT}/scripts/check-architecture.sh`。

开发过程中，如果某个模块第一次形成稳定复用模式，主动补 `{HARNESS_ROOT}/state/reuse-index/<source-path>/README.md`；如果模块内部也已经分出稳定层次，再继续补该子路径下的镜像 `README.md`。这是本地提醒，不作为 pre-commit 阻塞项。

如果用户明确要求严格参考 `deusyu/harness-engineering` 的目录形态，再使用 strict reference 模式。""",
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
    hook_docs = f"""## Harness 检查

- `pre-commit`：运行 `{HARNESS_ROOT}/scripts/check-harness-consistency.sh`，其中会继续执行 `{HARNESS_ROOT}/scripts/check-architecture.sh` 与 `{HARNESS_ROOT}/scripts/check-test-workflow.sh`，检查 CTF 探索版 harness 的目录、导航、本地私有 reuse 索引归属、最小架构守卫和测试工作流约束。
- `pre-commit`：非阻塞运行 `{HARNESS_ROOT}/scripts/check-skill-sync-reminder.sh --staged`，提醒把跨项目规则上收全局 skill 或 shared harness。
- `commit-msg`：运行 `{HARNESS_ROOT}/scripts/check-commit-message.sh`，由共享检查器读取 `{HARNESS_ROOT}/harness/policies/commit-message.json` 校验标题、正文和激活任务的 `Task:` 绑定。
- 原有项目 hook 逻辑继续保留。"""
    return "Initialized CTF-current harness", hook_docs
