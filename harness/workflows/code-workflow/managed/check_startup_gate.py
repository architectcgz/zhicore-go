#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SESSION_GATES_DIR = ROOT / ".harness" / "session-gates"
TASK_SLUG_RE = re.compile(r"^[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9]+(?:-[a-z0-9]+)*$")
EFFECTIVE_GATE_STATUSES = {"active", "ready_to_merge"}
REQUIRED_PLAN_HEADINGS = (
    "## Task Metadata",
    "## Task Classification",
    "## Files",
    "## 复用与 Owner 决策",
    "## Intake Analysis Gate",
    "## Validation",
)
PLACEHOLDER_TOKENS = ("TODO", "待填写", "__TASK_", "__STARTED_AT__", "__WORKTREE_PATH__", "__BRANCH_NAME__")
LOW_RISK_PREFIXES = (
    "docs/",
    "README",
    ".gitignore",
)
LOW_RISK_SUFFIXES = (
    ".md",
    ".txt",
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="validate local startup gate state for non-trivial work")
    parser.add_argument("--print-active-slug", action="store_true")
    parser.add_argument("--print-gate-path", action="store_true")
    parser.add_argument("--quiet", action="store_true")
    parser.add_argument("--staged", action="store_true")
    parser.add_argument("--base")
    parser.add_argument("--head", default="HEAD")
    args = parser.parse_args()
    if args.staged and args.base:
        parser.error("--staged and --base cannot be used together")
    return args


def run_git(*args: str) -> str:
    result = subprocess.run(["git", *args], cwd=ROOT, check=True, capture_output=True, text=True)
    return result.stdout


def changed_paths(args: argparse.Namespace) -> list[str]:
    if args.base:
        output = run_git("diff", "--name-only", "--diff-filter=ACMR", f"{args.base}...{args.head}")
    else:
        output = run_git("diff", "--cached", "--name-only", "--diff-filter=ACMR")
    return [line.strip() for line in output.splitlines() if line.strip()]


def requires_gate(path: str) -> bool:
    if path.startswith(LOW_RISK_PREFIXES) or path.endswith(LOW_RISK_SUFFIXES):
        return False
    return True


def load_effective_gates() -> list[tuple[Path, dict[str, object]]]:
    gates: list[tuple[Path, dict[str, object]]] = []
    if not SESSION_GATES_DIR.is_dir():
        return gates
    for path in sorted(SESSION_GATES_DIR.glob("*.json")):
        if not path.is_file():
            continue
        try:
            payload = json.loads(path.read_text(encoding="utf-8"))
        except (json.JSONDecodeError, OSError):
            raise SystemExit(f"FAIL: invalid startup gate file: {path.relative_to(ROOT).as_posix()}")
        if payload.get("status") in EFFECTIVE_GATE_STATUSES:
            gates.append((path, payload))
    return gates


def contains_placeholder(text: str) -> bool:
    return any(token in text for token in PLACEHOLDER_TOKENS)


def extract_section(plan_text: str, heading: str) -> str:
    pattern = re.compile(rf"^{re.escape(heading)}\s*$([\s\S]*?)(?=^## |\Z)", re.MULTILINE)
    match = pattern.search(plan_text)
    return match.group(1).strip() if match else ""


def validate_effective_gate(path: Path, payload: dict[str, object], *, require_completed_plan: bool) -> list[str]:
    errors: list[str] = []

    task_slug = payload.get("task_slug")
    if not isinstance(task_slug, str) or not TASK_SLUG_RE.fullmatch(task_slug):
        errors.append("task_slug missing or invalid")

    status = payload.get("status")
    if status not in EFFECTIVE_GATE_STATUSES:
        errors.append("status missing or invalid")

    branch = payload.get("branch")
    if not isinstance(branch, str) or not branch.startswith("task/"):
        errors.append("branch missing or invalid")

    plan_path_value = payload.get("plan_path")
    if not isinstance(plan_path_value, str):
        errors.append("plan_path missing")
        return errors

    plan_path = ROOT / plan_path_value
    if not plan_path.is_file():
        errors.append(f"plan file missing: {plan_path_value}")
        return errors

    plan_text = plan_path.read_text(encoding="utf-8")
    for heading in REQUIRED_PLAN_HEADINGS:
        if heading not in plan_text:
            errors.append(f"plan missing required heading: {heading}")

    if "**Goal:**" not in plan_text or "**Architecture:**" not in plan_text:
        errors.append("plan missing required summary fields")
    elif require_completed_plan:
        summary_lines = "\n".join(
            line for line in plan_text.splitlines() if line.startswith("**Goal:**") or line.startswith("**Architecture:**")
        )
        if contains_placeholder(summary_lines):
            errors.append("plan summary fields still contain placeholders")

    if require_completed_plan:
        for heading in ("## Task Classification", "## Files", "## 复用与 Owner 决策", "## Intake Analysis Gate", "## Validation"):
            section_text = extract_section(plan_text, heading)
            if not section_text:
                errors.append(f"plan section is empty: {heading}")
                continue
            if contains_placeholder(section_text):
                errors.append(f"plan section still contains placeholders: {heading}")

    return errors


def main() -> int:
    args = parse_args()
    gates = load_effective_gates()

    if args.print_active_slug or args.print_gate_path:
        if not gates:
            return 1
        if len(gates) > 1:
            print("FAIL: multiple effective startup gates in current worktree", file=sys.stderr)
            return 1
        gate_path, payload = gates[0]
        errors = validate_effective_gate(gate_path, payload, require_completed_plan=False)
        if errors:
            print("FAIL: effective startup gate is invalid", file=sys.stderr)
            for error in errors:
                print(f"- {error}", file=sys.stderr)
            return 1
        print(payload["task_slug"] if args.print_active_slug else gate_path.relative_to(ROOT).as_posix())
        return 0

    changed = changed_paths(args)
    gated = sorted(path for path in changed if requires_gate(path))

    if not gated:
        if not args.quiet:
            print("PASS: no startup-gated changes in diff")
        return 0

    if not gates:
        print("FAIL: startup-gated changes require an effective task gate", file=sys.stderr)
        for path in gated:
            print(f"- {path}", file=sys.stderr)
        print("Use scripts/start-implementation.sh before continuing.", file=sys.stderr)
        return 1

    if len(gates) > 1:
        print("FAIL: multiple effective startup gates in current worktree", file=sys.stderr)
        return 1

    gate_path, payload = gates[0]
    plan_path_value = payload.get("plan_path")
    requires_completed_plan = any(path != plan_path_value for path in gated)
    errors = validate_effective_gate(gate_path, payload, require_completed_plan=requires_completed_plan)
    if errors:
        print("FAIL: effective startup gate is invalid", file=sys.stderr)
        print(f"- gate: {gate_path.relative_to(ROOT).as_posix()}", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    if not args.quiet:
        print("PASS: startup gate covers current diff")
        print(f"- gate: {gate_path.relative_to(ROOT).as_posix()}")
        print(f"- status: {payload['status']}")
        print(f"- task slug: {payload['task_slug']}")
        print(f"- plan: {payload['plan_path']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
