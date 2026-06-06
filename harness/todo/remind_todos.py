#!/usr/bin/env python3
from __future__ import annotations

import argparse
import re
from pathlib import Path

from common import is_archived_path, markdown_files_under, resolve_scope, todo_is_completed


OPEN_ITEM_RE = re.compile(r"^- \[ \] (.+)$")
IGNORED_FILENAMES = {"README.md", "active.md"}
MAX_LINES = 20


def render_relative(path: Path, base: Path) -> str:
    try:
        return str(path.relative_to(base))
    except ValueError:
        return str(path)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--cwd", default=".")
    parser.add_argument("--strict", action="store_true")
    parser.add_argument("--quiet-if-empty", action="store_true")
    args = parser.parse_args()

    scope = resolve_scope(Path(args.cwd).resolve())
    todo_dir = scope.live_dir

    if not todo_dir.exists():
        if not args.quiet_if_empty:
            print(f"[todo] no todo directory found at {todo_dir}")
        return 0

    open_items: list[tuple[Path, int, str]] = []
    completed_files: list[Path] = []

    for path in markdown_files_under(todo_dir):
        if path.name in IGNORED_FILENAMES or is_archived_path(path, scope):
            continue
        text = path.read_text(encoding="utf-8")
        for lineno, line in enumerate(text.splitlines(), start=1):
            match = OPEN_ITEM_RE.match(line)
            if match:
                open_items.append((path, lineno, match.group(1)))
        if todo_is_completed(text):
            completed_files.append(path)

    if not open_items and not completed_files:
        if not args.quiet_if_empty:
            print(f"[todo] no open todo items in {todo_dir}")
        return 0

    if open_items:
        unique_files = {path for path, _, _ in open_items}
        print(f"[todo] reminder: {len(open_items)} open items across {len(unique_files)} files in {todo_dir}")
        for path, lineno, text in open_items[:MAX_LINES]:
            print(f"  - {render_relative(path, scope.root)}:{lineno} {text}")
        if len(open_items) > MAX_LINES:
            print(f"  - ... and {len(open_items) - MAX_LINES} more open items")

    if completed_files:
        print(f"[todo] reminder: {len(completed_files)} completed todo files still need archive in {todo_dir}")
        for path in completed_files[:MAX_LINES]:
            print(f"  - {render_relative(path, scope.root)}")
        if len(completed_files) > MAX_LINES:
            print(f"  - ... and {len(completed_files) - MAX_LINES} more completed files")

    if args.strict:
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
