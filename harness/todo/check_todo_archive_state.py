#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from common import is_archived_path, markdown_files_under, resolve_scope, todo_has_open_items, todo_is_completed


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--cwd", default=".")
    args = parser.parse_args()

    scope = resolve_scope(Path(args.cwd).resolve())
    live_files = [path for path in markdown_files_under(scope.live_dir) if not is_archived_path(path, scope)]
    archived_files = markdown_files_under(scope.archive_dir)

    bad_live: list[Path] = []
    bad_archived: list[Path] = []

    for path in live_files:
        text = path.read_text(encoding="utf-8")
        if todo_is_completed(text):
            bad_live.append(path)

    for path in archived_files:
        text = path.read_text(encoding="utf-8")
        if todo_has_open_items(text):
            bad_archived.append(path)

    if bad_live or bad_archived:
        print("FAIL: todo archive state is inconsistent")
        for path in bad_live:
            print(f"- completed todo still in live dir: {path}")
        for path in bad_archived:
            print(f"- archived todo still has open items: {path}")
        return 1

    print("PASS: completed todos are archived and archived todos have no open items")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
