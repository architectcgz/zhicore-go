#!/usr/bin/env python3
from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parent


def run(script: str, cwd: str) -> int:
    return subprocess.run([sys.executable, str(SCRIPT_DIR / script), "--cwd", cwd], check=False).returncode


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--cwd", default=".")
    args = parser.parse_args()

    failures = 0
    for script in ("check_todo_paths.py", "check_todo_filenames.py", "check_todo_archive_state.py"):
        failures |= run(script, args.cwd)
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
