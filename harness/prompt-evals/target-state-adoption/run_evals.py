#!/usr/bin/env python3
"""通过任意 stdin/stdout CLI agent 运行目标状态迁移行为评测。"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from pathlib import Path
from typing import Any


BASE = Path(__file__).resolve().parent
RULE_FILES = [
    BASE.parents[2] / "docs/agent-rules/collaboration-basics.md",
    BASE.parents[2] / "skills/architect-agent/SKILL.md",
    BASE.parents[2] / "skills/superpowers/brainstorming/SKILL.md",
    BASE.parents[2] / "skills/superpowers/writing-plans/SKILL.md",
]


def load_cases() -> list[dict[str, Any]]:
    return json.loads((BASE / "cases.json").read_text(encoding="utf-8"))


def build_prompt(case: dict[str, Any]) -> str:
    rules = "\n\n".join(
        f"--- {path.name} ---\n{path.read_text(encoding='utf-8')}" for path in RULE_FILES
    )
    return f"""你是迁移计划决策 agent。以下共享规则必须执行：

{rules}

IMPORTANT：这是实际决策，必须选择，不要把问题当学术问答。

场景：
{case['scenario']}

只输出一个 JSON object，不要 Markdown：
{{
  "choice": "A|B|C",
  "final_state": "最终生产路径",
  "legacy_removals": ["完成后必须消失的旧路径"],
  "retained_boundaries": ["明确保留的稳定边界或临时边界及退出条件"],
  "reason": "简短理由"
}}
"""


def extract_json(output: str) -> dict[str, Any]:
    start = output.find("{")
    end = output.rfind("}")
    if start < 0 or end < start:
        raise ValueError("response does not contain a JSON object")
    return json.loads(output[start : end + 1])


def score(case: dict[str, Any], response: dict[str, Any]) -> list[str]:
    failures: list[str] = []
    if response.get("choice") != case["expected_choice"]:
        failures.append(
            f"choice={response.get('choice')!r}, expected={case['expected_choice']!r}"
        )

    combined = json.dumps(response, ensure_ascii=False).lower()
    removals = json.dumps(response.get("legacy_removals", []), ensure_ascii=False).lower()
    for term in case.get("required_terms", []):
        if term.lower() not in combined:
            failures.append(f"missing required term: {term}")
    for term in case.get("required_removal_terms", []):
        if term.lower() not in removals:
            failures.append(f"legacy_removals missing: {term}")
    for term in case.get("forbidden_terms", []):
        if term.lower() in combined:
            failures.append(f"contains forbidden rationalization: {term}")
    return failures


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "command",
        nargs=argparse.REMAINDER,
        help="在 -- 后提供从 stdin 读取 prompt、向 stdout 输出结果的命令",
    )
    args = parser.parse_args()
    command = args.command
    if command and command[0] == "--":
        command = command[1:]
    if not command:
        parser.error("missing agent command after --")

    failed = False
    for case in load_cases():
        try:
            completed = subprocess.run(
                command,
                input=build_prompt(case),
                text=True,
                capture_output=True,
                timeout=180,
                check=False,
            )
            if completed.returncode != 0:
                raise RuntimeError(
                    f"agent command exited {completed.returncode}: {completed.stderr.strip()}"
                )
            response = extract_json(completed.stdout)
            failures = score(case, response)
        except Exception as exc:  # noqa: BLE001 - eval runner must report each case.
            failures = [str(exc)]

        if failures:
            failed = True
            print(f"FAIL {case['id']}")
            for failure in failures:
                print(f"  - {failure}")
        else:
            print(f"PASS {case['id']}")

    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
