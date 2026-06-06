#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from common import is_allowed_todo_path, markdown_files_under, resolve_scope


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--cwd", default=".")
    args = parser.parse_args()

    scope = resolve_scope(Path(args.cwd).resolve())
    candidates: list[Path] = []
    for directory in scope.candidate_dirs:
        candidates.extend(markdown_files_under(directory))

    bad = [path for path in candidates if not is_allowed_todo_path(path, scope)]
    if bad:
        print("FAIL: todo files outside allowed todo paths")
        for path in bad:
            print(f"- {path}")
        return 1

    print(f"PASS: todo paths stay under {scope.live_dir} and {scope.archive_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
