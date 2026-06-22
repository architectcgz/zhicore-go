#!/usr/bin/env python3
"""Check test file size limits."""

from __future__ import annotations

import argparse
from pathlib import Path
import subprocess
import sys


DEFAULT_MAX_LINES = 800
DEFAULT_SCAN_DIRS = ("services", "libs", "tests")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Fail when a *_test.go file is too large to review safely.",
    )
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument(
        "--working-tree",
        action="store_true",
        help="inspect changed and untracked *_test.go files",
    )
    mode.add_argument(
        "--staged",
        action="store_true",
        help="inspect staged *_test.go files",
    )
    mode.add_argument(
        "--files",
        nargs="+",
        help="inspect explicit files or directories",
    )
    parser.add_argument(
        "--root",
        type=Path,
        default=Path(__file__).resolve().parents[1],
        help="repository root; defaults to the parent of scripts/",
    )
    parser.add_argument(
        "--max-lines",
        type=int,
        default=DEFAULT_MAX_LINES,
        help=f"maximum allowed lines per *_test.go file; default {DEFAULT_MAX_LINES}",
    )
    return parser.parse_args()


def run_git(root: Path, *args: str) -> str:
    result = subprocess.run(
        ["git", *args],
        cwd=root,
        check=True,
        capture_output=True,
        text=True,
    )
    return result.stdout


def count_lines(path: Path) -> int:
    with path.open("rb") as file:
        return sum(1 for _ in file)


def is_test_file(path: Path) -> bool:
    return path.name.endswith("_test.go")


def is_in_scan_dirs(root: Path, path: Path) -> bool:
    try:
        rel_path = path.resolve().relative_to(root)
    except ValueError:
        return False
    return bool(rel_path.parts) and rel_path.parts[0] in DEFAULT_SCAN_DIRS


def all_test_files(root: Path) -> list[Path]:
    files: list[Path] = []
    for directory in DEFAULT_SCAN_DIRS:
        scan_root = root / directory
        if scan_root.exists():
            files.extend(scan_root.rglob("*_test.go"))
    return sorted(files)


def changed_working_tree_files(root: Path) -> list[Path]:
    tracked = run_git(root, "diff", "--name-only", "--diff-filter=ACMR", "HEAD")
    untracked = run_git(root, "ls-files", "--others", "--exclude-standard")
    return paths_from_git(root, [*tracked.splitlines(), *untracked.splitlines()])


def changed_staged_files(root: Path) -> list[Path]:
    output = run_git(root, "diff", "--cached", "--name-only", "--diff-filter=ACMR")
    return paths_from_git(root, output.splitlines())


def paths_from_git(root: Path, paths: list[str]) -> list[Path]:
    files: list[Path] = []
    for raw_path in paths:
        if not raw_path:
            continue
        path = root / raw_path
        if path.is_file() and is_test_file(path) and is_in_scan_dirs(root, path):
            files.append(path)
    return sorted(set(files))


def explicit_test_files(root: Path, paths: list[str]) -> list[Path]:
    files: list[Path] = []
    for raw_path in paths:
        path = Path(raw_path)
        if not path.is_absolute():
            path = root / path

        if path.is_dir():
            files.extend(
                candidate
                for candidate in path.rglob("*_test.go")
                if is_in_scan_dirs(root, candidate)
            )
            continue

        if path.is_file() and is_test_file(path) and is_in_scan_dirs(root, path):
            files.append(path)

    return sorted(set(files))


def test_files_for_mode(args: argparse.Namespace, root: Path) -> tuple[str, list[Path]]:
    if args.working_tree:
        return "working-tree", changed_working_tree_files(root)
    if args.staged:
        return "staged", changed_staged_files(root)
    if args.files:
        return "files", explicit_test_files(root, args.files)
    return "all", all_test_files(root)


def main() -> int:
    args = parse_args()
    root = args.root.resolve()
    mode, test_files = test_files_for_mode(args, root)

    oversized: list[tuple[Path, int]] = []
    for test_file in test_files:
        line_count = count_lines(test_file)
        if line_count > args.max_lines:
            oversized.append((test_file.relative_to(root), line_count))

    if not test_files:
        print(f"[test-size] no *_test.go files for mode: {mode}")
        return 0

    for rel_path, line_count in oversized:
        print(
            f"test file too large: {rel_path} has {line_count} lines; "
            f"max is {args.max_lines}",
            file=sys.stderr,
        )

    if oversized:
        print(
            "split tests by behavior, endpoint, use case, repository query, "
            "or worker scenario",
            file=sys.stderr,
        )
        return 1

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
