#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from common import filename_has_required_date, markdown_files_under, resolve_scope


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--cwd", default=".")
    args = parser.parse_args()

    scope = resolve_scope(Path(args.cwd).resolve())
    candidates = markdown_files_under(scope.live_dir) + markdown_files_under(scope.archive_dir)
    bad = [path for path in candidates if not filename_has_required_date(path)]
    if bad:
        print("FAIL: todo files must use YYYY-MM-DD-HHMM-slug.md naming")
        for path in bad:
            print(f"- {path}")
        return 1

    print("PASS: todo filenames carry the required date prefix")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
