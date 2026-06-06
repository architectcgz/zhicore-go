#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from common import markdown_files_under, resolve_scope, todo_is_completed, update_status_line


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--cwd", default=".")
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()

    scope = resolve_scope(Path(args.cwd).resolve())
    live_files = [path for path in markdown_files_under(scope.live_dir) if "archive" not in path.parts]
    archived = []

    for path in live_files:
        text = path.read_text(encoding="utf-8")
        if not todo_is_completed(text):
            continue
        target = scope.archive_dir / path.name
        if target.exists():
            raise SystemExit(f"archive target already exists: {target}")
        archived.append((path, target, update_status_line(text, "Archived")))

    if args.dry_run:
        if not archived:
            print("PASS: no completed todo files need archiving")
            return 0
        print("DRY RUN: completed todo files would be archived")
        for source, target, _ in archived:
            print(f"- {source} -> {target}")
        return 0

    scope.archive_dir.mkdir(parents=True, exist_ok=True)
    for source, target, content in archived:
        target.write_text(content, encoding="utf-8")
        source.unlink()

    if not archived:
        print("PASS: no completed todo files needed archiving")
    else:
        print("PASS: completed todo files archived")
        for _, target, _ in archived:
            print(f"- {target}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
