#!/usr/bin/env python3
"""Check test file size limits."""

from __future__ import annotations

import argparse
from pathlib import Path
import sys


DEFAULT_MAX_LINES = 800
DEFAULT_SCAN_DIRS = ("services", "libs", "tests")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Fail when a *_test.go file is too large to review safely.",
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


def count_lines(path: Path) -> int:
    with path.open("rb") as file:
        return sum(1 for _ in file)


def iter_test_files(root: Path) -> list[Path]:
    files: list[Path] = []
    for directory in DEFAULT_SCAN_DIRS:
        scan_root = root / directory
        if scan_root.exists():
            files.extend(scan_root.rglob("*_test.go"))
    return sorted(files)


def main() -> int:
    args = parse_args()
    root = args.root.resolve()

    oversized: list[tuple[Path, int]] = []
    for test_file in iter_test_files(root):
        line_count = count_lines(test_file)
        if line_count > args.max_lines:
            oversized.append((test_file.relative_to(root), line_count))

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
