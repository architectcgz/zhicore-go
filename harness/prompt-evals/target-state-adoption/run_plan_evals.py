#!/usr/bin/env python3
"""让任意 stdin/stdout agent 生成 implementation plan，并机械检查目标状态完整性。"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from datetime import datetime
from pathlib import Path
from typing import Any


BASE = Path(__file__).resolve().parent
AGENT_HOME = BASE.parents[2]
RULE_FILES = [
    AGENT_HOME / "AGENTS.md",
    AGENT_HOME / "docs/agent-rules/collaboration-basics.md",
    AGENT_HOME / "skills/architect-agent/SKILL.md",
    AGENT_HOME / "skills/superpowers/brainstorming/SKILL.md",
    AGENT_HOME / "skills/superpowers/writing-plans/SKILL.md",
]


def load_cases() -> list[dict[str, Any]]:
    return json.loads((BASE / "plan_cases.json").read_text(encoding="utf-8"))


def build_prompt(case: dict[str, Any]) -> str:
    rules = "\n\n".join(
        f"--- {path} ---\n{path.read_text(encoding='utf-8')}" for path in RULE_FILES
    )
    return f"""你是负责落地迁移的架构与实施计划 agent。遵守以下共享规则：

{rules}

任务场景：
{case['scenario']}

输出一份完整、可执行的中文 Markdown implementation plan。计划必须基于场景定义最终生产路径或明确的阶段边界，列出运行时 owner、逐文件/逐模块改动、旧路径删除或临时保留、验收命令、失败恢复/回退。不要输出 JSON，不要讨论本评测。
"""


def contains(pattern: str, text: str) -> bool:
    return re.search(pattern, text, flags=re.IGNORECASE | re.MULTILINE) is not None


def score(case: dict[str, Any], plan: str) -> list[str]:
    failures: list[str] = []
    for section in case.get("required_sections", []):
        if not contains(rf"^#{{1,6}}\s+.*{re.escape(section)}", plan):
            failures.append(f"missing section: {section}")
    for pattern in case.get("required_patterns", []):
        if not contains(pattern, plan):
            failures.append(f"missing required pattern: {pattern}")
    for pattern in case.get("required_removal_patterns", []):
        removal_scope = re.findall(
            r"(?ims)^#{1,6}\s+.*(?:删除|移除|清理|退役).*?(?=^#{1,6}\s+|\Z)", plan
        )
        if not any(contains(pattern, block) for block in removal_scope):
            failures.append(f"removal section missing pattern: {pattern}")
    for pattern in case.get("forbidden_patterns", []):
        if contains(pattern, plan):
            failures.append(f"contains forbidden pattern: {pattern}")
    if case.get("requires_exit_criteria") and not contains(
        r"退出条件|完成条件|移除条件", plan
    ):
        failures.append("missing staged exit criteria")
    if not contains(r"`[^`]*(?:test|check|build|lint|vet|pytest|npm|pnpm|go )[^`]*`", plan):
        failures.append("missing exact verification command")
    return failures


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--results-dir", type=Path)
    parser.add_argument(
        "command", nargs=argparse.REMAINDER,
        help="在 -- 后提供从 stdin 读取 prompt、向 stdout 输出 Markdown 的命令",
    )
    args = parser.parse_args()
    command = args.command[1:] if args.command[:1] == ["--"] else args.command
    if not command:
        parser.error("missing agent command after --")

    result_dir = args.results_dir or BASE / "results" / datetime.now().strftime("%Y%m%d-%H%M%S")
    result_dir.mkdir(parents=True, exist_ok=True)
    failed = False
    summary: list[str] = []
    for case in load_cases():
        try:
            completed = subprocess.run(
                command, input=build_prompt(case), text=True, capture_output=True,
                timeout=300, check=False,
            )
            if completed.returncode != 0:
                raise RuntimeError(
                    f"agent command exited {completed.returncode}: {completed.stderr.strip()}"
                )
            plan = completed.stdout.strip()
            if not plan:
                raise ValueError("agent returned an empty plan")
            (result_dir / f"{case['id']}.md").write_text(plan + "\n", encoding="utf-8")
            failures = score(case, plan)
        except Exception as exc:  # noqa: BLE001 - eval runner reports all cases.
            failures = [str(exc)]

        status = "FAIL" if failures else "PASS"
        failed = failed or bool(failures)
        summary.append(f"{status} {case['id']}")
        print(summary[-1])
        for failure in failures:
            print(f"  - {failure}")

    (result_dir / "SUMMARY.txt").write_text("\n".join(summary) + "\n", encoding="utf-8")
    print(f"results: {result_dir}")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
