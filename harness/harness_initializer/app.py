#!/usr/bin/env python3
"""Harness initializer orchestration."""

from __future__ import annotations

import argparse
from pathlib import Path

from .consistency_content import check_script, ctf_current_check_script
from .content import (
    agent_entrypoints_check_script,
    architecture_guard_commands_policy,
    architecture_guard_paths_policy,
    architecture_guard_script,
    commit_message_check_script,
    commit_message_policy_content,
    script_guard_check_script,
    script_guard_policy_content,
    skill_sync_reminder_script,
    test_workflow_check_script,
    todo_governance_check_script,
    todo_reminder_script,
)
from .docs_content import ctf_current_docs, strict_docs
from .scaffold import (
    add_gitignore_exceptions,
    ensure_claude_symlink,
    ensure_documentation_scaffold,
    insert_commit_msg_hook,
    insert_hook,
    insert_or_replace,
    run_agent_entrypoint_check,
    write,
    write_if_missing,
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", default=".", help="Target repository root")
    parser.add_argument("--project-name", default=None)
    parser.add_argument("--profile", default="generic")
    parser.add_argument("--mode", default="ctf-current", choices=["ctf-current", "strict-reference"])
    return parser.parse_args()


def write_common_scaffold(repo: Path, profile: str, consistency_script: str) -> None:
    write(repo / "scripts/check-harness-consistency.sh", consistency_script, executable=True)
    write(repo / "scripts/check-agent-entrypoints.sh", agent_entrypoints_check_script(), executable=True)
    write(repo / "scripts/check-architecture.sh", architecture_guard_script(), executable=True)
    write(repo / "scripts/check-test-workflow.sh", test_workflow_check_script(), executable=True)
    write(repo / "scripts/check-script-guard.sh", script_guard_check_script(), executable=True)
    write(repo / "scripts/check-open-todos.sh", todo_reminder_script(), executable=True)
    write(repo / "scripts/check-todo-governance.sh", todo_governance_check_script(), executable=True)
    write(repo / "scripts/check-skill-sync-reminder.sh", skill_sync_reminder_script(), executable=True)
    write(repo / "scripts/check-commit-message.sh", commit_message_check_script(), executable=True)
    write_if_missing(repo / "harness/policies/architecture-guard-paths.txt", architecture_guard_paths_policy())
    write_if_missing(repo / "harness/policies/architecture-guard-commands.txt", architecture_guard_commands_policy())
    write(repo / "harness/policies/commit-message.json", commit_message_policy_content(profile))
    write_if_missing(repo / "harness/policies/script-guard.json", script_guard_policy_content())


def configure_strict_reference(repo: Path, project_name: str, profile: str) -> tuple[str, str]:
    ensure_documentation_scaffold(repo)
    for relative, content in strict_docs(project_name, profile).items():
        write(repo / relative, content)
    write_common_scaffold(repo, profile, check_script())
    insert_or_replace(
        repo / "AGENTS.md",
        "root-navigation",
        """## Harness Engineering 学习档案

严格参考 `deusyu/harness-engineering` 的顶层结构：

| 目录 | 内容 | 说明 |
|------|------|------|
| `concepts/` | AGENTS 补充 | 补充项目 `AGENTS.md`，记录长期概念、原则和 harness 定义 |
| `thinking/` | 独立思考 | 对项目 harness 边界和取舍的判断 |
| `practice/` | 动手实践 | 初始化和后续实验记录 |
| `feedback/` | 反馈记录 | 踩坑、修正和可复用经验 |
| `works/` | 作品输出 | 可展示模板、报告和说明 |
| `prompts/` | 提示词积累 | 已验证提示词和工作流 |
| `references/` | 外部资源 | 文章、仓库和工具索引 |
| `docs/architecture/` | 架构事实 | 当前系统设计、边界和长期技术约束 |

项目根保持 `CLAUDE.md -> AGENTS.md`，让 Claude / Codex 使用同一份入口规则。

机械化检查：`bash scripts/check-harness-consistency.sh`。
架构守卫入口：`bash scripts/check-architecture.sh`。""",
    )
    insert_or_replace(
        repo / "AGENTS.md",
        "todo-reminder",
        """## Todo Reminder

开始新任务前，先运行 `bash scripts/check-open-todos.sh --quiet-if-empty`，先过一遍 `docs/todo/` 里的未完成事项；如果命中当前主题，首条回复先提醒。已完成但还没归档的 todo 也会在这里提示。""",
    )
    insert_or_replace(
        repo / "AGENTS.md",
        "test-workflow",
        """## Test Workflow

如果仓库存在自动化测试或明显测试面，先写或更新最小相关测试，再进入实现。

- Write or update the narrowest relevant tests first.
- After changing tests, run the smallest relevant test command that covers the touched surface.
- After the test command, run the relevant script check such as `bash scripts/check-test-workflow.sh` or `bash scripts/check-harness-consistency.sh` before claiming completion.
- 如果当前仓库已经有 `scripts/check-harness-consistency.sh`、git hooks 或 CI guardrail，测试相关脚本检查必须接入这些实际检查链路，不能只停留在提示词里。""",
    )
    insert_or_replace(
        repo / "README.md",
        "readme-harness",
        """## Harness Engineering

本项目按 `deusyu/harness-engineering` 建立顶层 harness 结构：

- `concepts/`：项目 `AGENTS.md` 的补充，记录长期概念与原则
- `thinking/`：独立思考
- `practice/`：实践记录
- `feedback/`：反馈闭环
- `works/`：作品输出
- `prompts/`：提示词积累
- `references/`：外部资料

一致性检查：

```bash
bash scripts/check-harness-consistency.sh
```

最小架构守卫：

```bash
bash scripts/check-architecture.sh
```""",
    )
    hook_docs = """## Harness 检查

- `pre-commit`：运行 `scripts/check-harness-consistency.sh`，其中会继续执行 `scripts/check-architecture.sh` 与 `scripts/check-test-workflow.sh`，检查严格参考 harness 的顶层目录、导航、最小架构守卫和测试工作流约束。
- `pre-commit`：非阻塞运行 `scripts/check-skill-sync-reminder.sh --staged`，提醒把跨项目规则上收全局 skill 或 shared harness。
- `commit-msg`：运行 `scripts/check-commit-message.sh`，由共享检查器读取 `harness/policies/commit-message.json` 校验标题、正文和激活任务的 `Task:` 绑定。
- 原有 API 合同同步逻辑继续保留。"""
    return "Initialized strict-reference harness", hook_docs


def configure_ctf_current(repo: Path, project_name: str, profile: str) -> tuple[str, str]:
    ensure_documentation_scaffold(repo)
    for relative, content in ctf_current_docs(project_name, profile).items():
        write(repo / relative, content)
    write_common_scaffold(repo, profile, ctf_current_check_script())
    insert_or_replace(
        repo / "AGENTS.md",
        "root-navigation",
        """## Harness Engineering

当前默认采用 CTF 探索版 harness 形态，并保留 `deusyu/harness-engineering` 的核心原则作为重要参考。

| 路径 | 内容 | 说明 |
|------|------|------|
| `.harness/` | 当前任务状态 | 只保存短期执行证据和当前 reuse 决策 |
| `.harness/reuse-index/` | 本地私有索引 | 用户自用的长期复用线索，默认 gitignore，`index.yaml` + 镜像 `README.md` |
| `harness/policies/` | 项目策略 | 可被检查脚本读取的本地规则 |
| `harness/templates/` | 模板 | 当前项目重复使用的决策或记录模板 |
| `harness/prompts/` | Prompt 入口 | 仓库内稳定入口、局部补充，以及仍然项目专属的 prompt |
| `harness/checks/` | 检查脚本 | 机械化一致性和规则检查 |
| `feedback/` | 反馈记录 | 踩坑、修正和可复用流程经验 |
| `docs/documentation-rules.md` | 文档规范 | 改文档前置读取与新增路径登记 |
| `docs/README.md` | 文档索引 | 当前事实源地图和文档阅读顺序 |
| `docs/architecture/` | 架构事实 | 当前系统设计、边界和长期技术约束 |

项目根保持 `CLAUDE.md -> AGENTS.md`，让 Claude / Codex 使用同一份入口规则。

机械化检查：`bash scripts/check-harness-consistency.sh`。
架构守卫入口：`bash scripts/check-architecture.sh`。

开发过程中，如果某个模块第一次形成稳定复用模式，主动补 `.harness/reuse-index/<source-path>/README.md`；如果模块内部也已经分出稳定层次，再继续补该子路径下的镜像 `README.md`。这是本地提醒，不作为 pre-commit 阻塞项。

如果用户明确要求严格参考 `deusyu/harness-engineering` 的目录形态，再使用 strict reference 模式。""",
    )
    insert_or_replace(
        repo / "AGENTS.md",
        "todo-reminder",
        """## Todo Reminder

开始新任务前，先运行 `bash scripts/check-open-todos.sh --quiet-if-empty`，先过一遍 `docs/todo/` 里的未完成事项；如果命中当前主题，首条回复先提醒。已完成但还没归档的 todo 也会在这里提示。""",
    )
    insert_or_replace(
        repo / "AGENTS.md",
        "test-workflow",
        """## Test Workflow

如果仓库存在自动化测试或明显测试面，先写或更新最小相关测试，再进入实现。

- Write or update the narrowest relevant tests first.
- After changing tests, run the smallest relevant test command that covers the touched surface.
- After the test command, run the relevant script check such as `bash scripts/check-test-workflow.sh` or `bash scripts/check-harness-consistency.sh` before claiming completion.
- 如果当前仓库已经有 `scripts/check-harness-consistency.sh`、git hooks 或 CI guardrail，测试相关脚本检查必须接入这些实际检查链路，不能只停留在提示词里。""",
    )
    insert_or_replace(
        repo / "README.md",
        "readme-harness",
        """## Harness Engineering

本项目采用 CTF 探索版 harness 形态：

- `.harness/`：当前任务状态
- `.harness/reuse-index/`：用户本地私有的长期复用索引，默认 gitignore
- `harness/policies/`：项目策略
- `harness/templates/`：模板
- `harness/prompts/`：Prompt 入口
- `harness/checks/`：检查脚本
- `feedback/`：反馈记录
- `docs/documentation-rules.md`：文档修改前置读取与新增路径登记
- `docs/README.md`：文档索引和当前事实源地图

一致性检查：

```bash
bash scripts/check-harness-consistency.sh
```

最小架构守卫：

```bash
bash scripts/check-architecture.sh
```""",
    )
    hook_docs = """## Harness 检查

- `pre-commit`：运行 `scripts/check-harness-consistency.sh`，其中会继续执行 `scripts/check-architecture.sh` 与 `scripts/check-test-workflow.sh`，检查 CTF 探索版 harness 的目录、导航、本地私有 reuse 索引归属、最小架构守卫和测试工作流约束。
- `pre-commit`：非阻塞运行 `scripts/check-skill-sync-reminder.sh --staged`，提醒把跨项目规则上收全局 skill 或 shared harness。
- `commit-msg`：运行 `scripts/check-commit-message.sh`，由共享检查器读取 `harness/policies/commit-message.json` 校验标题、正文和激活任务的 `Task:` 绑定。
- 原有项目 hook 逻辑继续保留。"""
    return "Initialized CTF-current harness", hook_docs


def main() -> None:
    args = parse_args()
    repo = Path(args.repo).resolve()
    project_name = args.project_name or repo.name
    if args.mode == "strict-reference":
        message, hook_docs = configure_strict_reference(repo, project_name, args.profile)
    else:
        message, hook_docs = configure_ctf_current(repo, project_name, args.profile)
    insert_hook(repo / ".githooks/pre-commit")
    insert_commit_msg_hook(repo / ".githooks/commit-msg")
    insert_or_replace(repo / ".githooks/README.md", "hook-docs", hook_docs)
    ensure_claude_symlink(repo)
    run_agent_entrypoint_check(repo)
    add_gitignore_exceptions(repo)
    print(f"{message} for {project_name} at {repo}")
