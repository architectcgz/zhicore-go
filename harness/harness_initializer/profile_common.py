#!/usr/bin/env python3
"""Common harness initializer profile helpers."""

from __future__ import annotations

from pathlib import Path

from .content import (
    aar_directory_readme,
    aar_example,
    aar_hook_readme,
    agent_entrypoints_check_script,
    architecture_guard_commands_policy,
    architecture_guard_paths_policy,
    architecture_guard_script,
    commit_message_check_script,
    commit_message_policy_content,
    known_antipatterns_examples,
    known_antipatterns_readme,
    post_tooluse_aar_hook_script,
    script_guard_check_script,
    script_guard_policy_content,
    skill_sync_reminder_script,
    test_trigger_rate_readme,
    test_trigger_rate_script,
    test_workflow_check_script,
    todo_governance_check_script,
    todo_reminder_script,
)
from .scaffold import HARNESS_ROOT, harness_dir, write, write_if_missing


def write_common_scaffold(repo: Path, profile: str, consistency_script: str) -> None:
    root = harness_dir(repo)
    write(root / "scripts/check-harness-consistency.sh", consistency_script, executable=True)
    write(root / "scripts/check-agent-entrypoints.sh", agent_entrypoints_check_script(), executable=True)
    write(root / "scripts/check-architecture.sh", architecture_guard_script(), executable=True)
    write(root / "scripts/check-test-workflow.sh", test_workflow_check_script(), executable=True)
    write(root / "scripts/check-script-guard.sh", script_guard_check_script(), executable=True)
    write(root / "scripts/check-open-todos.sh", todo_reminder_script(), executable=True)
    write(root / "scripts/check-todo-governance.sh", todo_governance_check_script(), executable=True)
    write(root / "scripts/check-skill-sync-reminder.sh", skill_sync_reminder_script(), executable=True)
    write(root / "scripts/check-commit-message.sh", commit_message_check_script(), executable=True)
    write(root / "scripts/test-trigger-rate.sh", test_trigger_rate_script(), executable=True)
    write_if_missing(root / "scripts/README-trigger-rate.md", test_trigger_rate_readme())
    write_if_missing(root / "harness/policies/architecture-guard-paths.txt", architecture_guard_paths_policy())
    write_if_missing(root / "harness/policies/architecture-guard-commands.txt", architecture_guard_commands_policy())
    write(root / "harness/policies/commit-message.json", commit_message_policy_content(profile))
    write_if_missing(root / "harness/policies/script-guard.json", script_guard_policy_content())

    # PostToolUse AAR Hook for after action review
    write(root / "harness/hooks/post-tooluse-aar.sh", post_tooluse_aar_hook_script(), executable=True)
    write_if_missing(root / "harness/hooks/README-AAR.md", aar_hook_readme())

    # AAR feedback directory
    write_if_missing(root / "feedback/aar/README.md", aar_directory_readme())
    write_if_missing(root / "feedback/aar/example-aar.md", aar_example())

    # Known Antipatterns directory
    write_if_missing(root / "harness/known-antipatterns/README.md", known_antipatterns_readme())
    write_if_missing(root / "harness/known-antipatterns/EXAMPLES.md", known_antipatterns_examples())

def quick_routing_shell() -> str:
    """生成新项目 AGENTS.md 的标准占位薄壳（Quick Routing + Auto-Triggers）。

    结构可复用、内容禁止预制：通用行用现成全局 skill，项目特定行留 `<!-- FILL -->`
    占位，项目起步后补充。两节放同一个 managed block，保证 test-trigger 解析锚点稳定。
    """
    return f"""## Quick Routing

项目刚起步时这是占位薄壳：通用行可直接用，项目特定行用 `<!-- FILL -->` 占位，起步后替换为真实文件与 skill。压缩后这张表仍是 Agent 查"该读哪些文件 / 用哪个 skill"的线索。

| Task type | Required reads | Workflow / Skill |
|-----------|---------------|------------------|
| Backend feature | <!-- FILL: backend patterns + tests/README --> | <!-- FILL: backend skill --> + `code-workflow` |
| Frontend feature | <!-- FILL: frontend rules --> | <!-- FILL: frontend skill --> + `code-workflow` |
| Review | <!-- FILL: review 规范 --> | `reviewer` |
| Bug fix | <!-- FILL: 相关 rules/tests --> | `systematic-debugging` |
| Add/Edit test | <!-- FILL: tests/README --> | `test-driven-development` |
| Architecture change | `{HARNESS_ROOT}/docs/architecture/` | `brainstorming` then `writing-plans` |
| Documentation update | <!-- FILL: 文档规范 --> | Direct edit |
| Commit changes | — | `committing-changes` |
| New non-trivial task | `bash {HARNESS_ROOT}/scripts/check-open-todos.sh --quiet-if-empty` | `harness-router` then `writing-plans` |
| Other | Read this AGENTS.md fully, then ask user | `harness-router` |

## Auto-Triggers

- New task in same session → re-read this AGENTS.md + the relevant skill SKILL.md（"我已经读过了"不算，上下文会压缩）。
- Context compact / clear → SessionStart hook 重新注入 skill bootstrap（若已配置）。
- Before any `git commit` → 先走 `committing-changes` skill 组织提交信息（默认禁止 Co-Authored-By）。
- Task complete (non-trivial) → 跑验证门禁，再做 AAR，若有新模式更新 `{HARNESS_ROOT}/feedback/`。
- 起步后把 `<!-- FILL -->` 行替换为本项目真实的必读文件与 skill，替换完运行 `bash {HARNESS_ROOT}/scripts/test-trigger-rate.sh` 检查 description 触发率。"""
