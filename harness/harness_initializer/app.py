#!/usr/bin/env python3
"""Harness initializer orchestration."""

from __future__ import annotations

import argparse
from pathlib import Path

from .profile_default import configure_current
from .profile_strict import configure_strict_reference
from .scaffold import (
    add_gitignore_exceptions,
    ensure_claude_symlink,
    insert_commit_msg_hook,
    insert_hook,
    insert_or_replace,
    run_agent_entrypoint_check,
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", default=".", help="Target repository root")
    parser.add_argument("--project-name", default=None)
    parser.add_argument("--profile", default="generic")
    parser.add_argument("--mode", default="default", choices=["default", "strict-reference"])
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    repo = Path(args.repo).resolve()
    project_name = args.project_name or repo.name
    if args.mode == "strict-reference":
        message, hook_docs = configure_strict_reference(repo, project_name, args.profile)
    else:
        message, hook_docs = configure_current(repo, project_name, args.profile)
    insert_hook(repo / ".githooks/pre-commit")
    insert_commit_msg_hook(repo / ".githooks/commit-msg")
    insert_or_replace(repo / ".githooks/README.md", "hook-docs", hook_docs)
    ensure_claude_symlink(repo)
    run_agent_entrypoint_check(repo)
    add_gitignore_exceptions(repo)
    print(f"{message} for {project_name} at {repo}")
