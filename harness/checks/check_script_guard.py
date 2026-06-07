#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="guardrail for harness/operator script growth")
    parser.add_argument("--cwd", default=".", help="repository root to evaluate")
    parser.add_argument("--policy", default=None, help="path to script guard JSON policy")
    return parser.parse_args()


def load_policy(policy_path: Path) -> dict[str, object]:
    try:
        return json.loads(policy_path.read_text(encoding="utf-8"))
    except FileNotFoundError as exc:
        raise SystemExit(f"FAIL: missing script guard policy: {policy_path}") from exc
    except json.JSONDecodeError as exc:
        raise SystemExit(f"FAIL: invalid script guard policy JSON: {policy_path}: {exc}") from exc


def rel(root: Path, path: Path) -> str:
    return path.relative_to(root).as_posix()


def matches_any(path: str, patterns: list[str]) -> bool:
    target = Path(path)
    return any(target.match(pattern) for pattern in patterns)


def collect_files(root: Path, include_patterns: list[str], exclude_patterns: list[str]) -> list[Path]:
    selected: dict[str, Path] = {}
    for pattern in include_patterns:
        for path in root.glob(pattern):
            if not path.is_file():
                continue
            relative = rel(root, path)
            if matches_any(relative, exclude_patterns):
                continue
            selected[relative] = path
    return [selected[key] for key in sorted(selected)]


def line_count(path: Path) -> int:
    with path.open("r", encoding="utf-8", errors="ignore") as handle:
        return sum(1 for _ in handle)


def threshold_for(path: str, default_limit: int, overrides: dict[str, int]) -> int:
    matched_limit = default_limit
    matched_pattern_length = -1
    target = Path(path)
    for pattern, limit in overrides.items():
        if target.match(pattern) and len(pattern) > matched_pattern_length:
            matched_limit = limit
            matched_pattern_length = len(pattern)
    return matched_limit


def main() -> int:
    args = parse_args()
    root = Path(args.cwd).resolve()
    policy_path = Path(args.policy).resolve() if args.policy else root / "harness" / "policies" / "script-guard.json"
    policy = load_policy(policy_path)

    include_patterns = [str(item) for item in policy.get("include", [])]
    exclude_patterns = [str(item) for item in policy.get("exclude", [])]
    default_limit = int(policy.get("max_lines", 260))
    overrides = {str(key): int(value) for key, value in dict(policy.get("max_lines_by_glob", {})).items()}
    advice = str(policy.get("advice", "")).strip()

    if not include_patterns:
        raise SystemExit(f"FAIL: script guard policy has no include patterns: {policy_path}")

    files = collect_files(root, include_patterns, exclude_patterns)
    failures: list[str] = []

    for path in files:
        relative = rel(root, path)
        lines = line_count(path)
        limit = threshold_for(relative, default_limit, overrides)
        if lines > limit:
            failures.append(f"{relative}: {lines} lines > {limit}")

    if failures:
        print("FAIL: script guard violations detected", file=sys.stderr)
        print(f"- policy: {policy_path}", file=sys.stderr)
        for item in failures:
            print(f"- {item}", file=sys.stderr)
        if advice:
            print(f"- advice: {advice}", file=sys.stderr)
        return 1

    print("PASS: script guard passed")
    print(f"- policy: {policy_path}")
    print(f"- checked files: {len(files)}")
    print(f"- default max lines: {default_limit}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
