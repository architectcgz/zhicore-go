#!/usr/bin/env python3
from __future__ import annotations

import argparse
import datetime as dt
from pathlib import Path

from common import build_filename, is_allowed_todo_path, parse_items, resolve_scope


def resolve_explicit_path(raw_path: str, scope_root: Path, title: str, created_at: dt.datetime) -> Path:
    explicit = Path(raw_path)
    if not explicit.is_absolute():
        explicit = (scope_root / explicit).resolve()
    else:
        explicit = explicit.resolve()
    if explicit.suffix == ".md":
        return explicit
    return explicit / build_filename(title, created_at)


def build_content(title: str, scope_root: Path, created_at: dt.datetime, context: str, notes: str) -> str:
    items = parse_items(title, notes)
    lines = [
        f"# {title}",
        "",
        f"- Project: `{scope_root}`",
        f"- Created: `{created_at.isoformat(timespec='minutes')}`",
        "- Status: `Open`",
        "",
    ]
    if context.strip():
        lines.extend(["## Context", "", context.strip(), ""])
    lines.extend(["## Open Items", ""])
    for item in items:
        lines.append(f"- [ ] {item}")
    lines.append("")
    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("title")
    parser.add_argument("--notes", default="")
    parser.add_argument("--context", default="")
    parser.add_argument("--cwd", default=".")
    parser.add_argument("--path", default="")
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()

    cwd = Path(args.cwd).resolve()
    scope = resolve_scope(cwd)
    created_at = dt.datetime.now().astimezone()

    if args.path:
        todo_path = resolve_explicit_path(args.path, scope.root, args.title, created_at)
        if not is_allowed_todo_path(todo_path.parent, scope):
            raise SystemExit(
                f"explicit path must stay under {scope.live_dir} or {scope.archive_dir}: {todo_path}"
            )
    else:
        todo_path = scope.live_dir / build_filename(args.title, created_at)

    if args.dry_run:
        print(todo_path)
        return 0

    todo_path.parent.mkdir(parents=True, exist_ok=True)
    todo_path.write_text(
        build_content(args.title, scope.root, created_at, args.context, args.notes),
        encoding="utf-8",
    )
    print(todo_path)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
