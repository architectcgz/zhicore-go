#!/usr/bin/env python3
"""Shared scaffold helpers for harness initializer."""

from __future__ import annotations

from pathlib import Path
import subprocess


START = "BEGIN HARNESS ENGINEERING"
END = "END HARNESS ENGINEERING"
WORKSPACE_AGENT_ENTRYPOINT_CHECK = Path.home() / "workspace" / "projects" / "scripts" / "check-agent-entrypoints.sh"
AGENTS_SKILLS_DIR = Path.home() / ".agents" / "skills"
CODEX_SKILLS_DIR = Path.home() / ".codex" / "skills"
STANDARD_DOC_DIRS = [
    "requirements",
    "contracts",
    "spec",
    "design",
    "todo",
    "architecture",
    "plan",
    "operations",
    "reviews",
    "reports",
    "improvements",
    "refs",
]
IMPROVEMENT_STATUS_DIRS = [
    "not-impl",
    "implemented",
    "agent-recorded",
    "rejected",
    "archived",
]


def resolve_skill_dir(skill_name: str) -> Path:
    for base in (AGENTS_SKILLS_DIR, CODEX_SKILLS_DIR):
        candidate = base / skill_name
        if candidate.exists():
            return candidate.resolve()
    raise RuntimeError(f"unable to resolve skill directory for {skill_name}")


DOCS_ASSETS = resolve_skill_dir("documentation-architecture") / "assets" / "docs"


def write(path: Path, content: str, executable: bool = False) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content.rstrip() + "\n", encoding="utf-8")
    if executable:
        path.chmod(path.stat().st_mode | 0o111)


def write_if_missing(path: Path, content: str, executable: bool = False) -> None:
    if path.exists():
        return
    write(path, content, executable=executable)


def read_asset(relative: str) -> str:
    return (DOCS_ASSETS / relative).read_text(encoding="utf-8")


def ensure_documentation_scaffold(repo: Path) -> None:
    for directory in STANDARD_DOC_DIRS:
        (repo / "docs" / directory).mkdir(parents=True, exist_ok=True)
    for directory in IMPROVEMENT_STATUS_DIRS:
        (repo / "docs" / "improvements" / directory).mkdir(parents=True, exist_ok=True)
    write(repo / "docs" / "documentation-rules.md", read_asset("documentation-rules.md"))
    write(repo / "docs" / "README.md", read_asset("README.md"))
    write(repo / "docs" / "improvements" / "README.md", read_asset("improvements/README.md"))
    write_if_missing(
        repo / "docs" / "architecture" / "README.md",
        """# Architecture Index

This directory stores current system design, module boundaries, data flow, integration points, and long-lived technical constraints.

## How To Use

- Keep only current architecture facts here.
- When a structural change lands, update this index or the nearest affected architecture document in the same change.
- If the project has not written stable architecture facts yet, record that state explicitly instead of leaving the directory unindexed.

## Current Entries

- Add links to active architecture documents here as they appear.
""",
    )


def ensure_claude_symlink(repo: Path) -> None:
    agents_path = repo / "AGENTS.md"
    claude_path = repo / "CLAUDE.md"
    if not agents_path.exists():
        raise RuntimeError(f"missing {agents_path} before creating CLAUDE.md symlink")
    if claude_path.is_symlink():
        if claude_path.resolve() == agents_path.resolve():
            return
        raise RuntimeError(
            f"{claude_path} already points to {claude_path.resolve()}, expected {agents_path.resolve()}"
        )
    if claude_path.exists():
        raise RuntimeError(
            f"{claude_path} already exists and is not a symlink; replace it manually with CLAUDE.md -> AGENTS.md"
        )
    claude_path.symlink_to("AGENTS.md")


def run_agent_entrypoint_check(repo: Path) -> None:
    if WORKSPACE_AGENT_ENTRYPOINT_CHECK.is_file():
        subprocess.run(["bash", str(WORKSPACE_AGENT_ENTRYPOINT_CHECK), str(repo)], check=True)


def managed_block(kind: str, body: str) -> str:
    return f"<!-- {START}: {kind} -->\n{body.rstrip()}\n<!-- {END}: {kind} -->"


def insert_or_replace(path: Path, kind: str, body: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    block = managed_block(kind, body)
    start = f"<!-- {START}: {kind} -->"
    end = f"<!-- {END}: {kind} -->"
    text = path.read_text(encoding="utf-8") if path.exists() else ""
    if start in text and end in text:
        before, rest = text.split(start, 1)
        _, after = rest.split(end, 1)
        path.write_text(before.rstrip() + "\n\n" + block + after, encoding="utf-8")
        return
    sep = "\n\n" if text.rstrip() else ""
    path.write_text(text.rstrip() + sep + block + "\n", encoding="utf-8")


def insert_hook(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    text = path.read_text(encoding="utf-8") if path.exists() else "#!/usr/bin/env bash\nset -euo pipefail\n"
    start = f"# {START}: pre-commit"
    end = f"# {END}: pre-commit"
    body = f"""{start}
if [[ -x scripts/check-consistency.sh ]]; then
  bash scripts/check-consistency.sh
fi
if [[ -x scripts/check-skill-sync-reminder.sh ]]; then
  bash scripts/check-skill-sync-reminder.sh --staged
fi
# {END}: pre-commit"""
    if start in text and end in text:
        before, rest = text.split(start, 1)
        _, after = rest.split(end, 1)
        text = (before.rstrip() + "\n" + after.lstrip()).rstrip() + "\n"
    if 'if [[ "$needs_sync" -eq 0 ]]; then' in text:
        text = text.replace('if [[ "$needs_sync" -eq 0 ]]; then', body + '\n\nif [[ "$needs_sync" -eq 0 ]]; then', 1)
    else:
        text = text.rstrip() + "\n\n" + body + "\n"
    path.write_text(text, encoding="utf-8")
    path.chmod(path.stat().st_mode | 0o111)


def insert_commit_msg_hook(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    text = path.read_text(encoding="utf-8") if path.exists() else "#!/usr/bin/env bash\nset -euo pipefail\n"
    start = f"# {START}: commit-msg"
    end = f"# {END}: commit-msg"
    body = f"""{start}
cd "$(git rev-parse --show-toplevel)"

bash scripts/check-commit-message.sh "$1"
# {END}: commit-msg"""
    if start in text and end in text:
        before, rest = text.split(start, 1)
        _, after = rest.split(end, 1)
        text = (before.rstrip() + "\n" + after.lstrip()).rstrip() + "\n"
    text = text.rstrip() + "\n\n" + body + "\n"
    path.write_text(text, encoding="utf-8")
    path.chmod(path.stat().st_mode | 0o111)


def upsert_gitignore_block(text: str, kind: str, lines: list[str]) -> str:
    start = f"# {START}: {kind}"
    end = f"# {END}: {kind}"
    body = "\n".join([start, *lines, end])
    if start in text and end in text:
        before, rest = text.split(start, 1)
        _, after = rest.split(end, 1)
        return (before.rstrip() + "\n" + body + "\n" + after.lstrip()).rstrip() + "\n"
    if not text.rstrip():
        return body + "\n"
    return text.rstrip() + "\n\n" + body + "\n"


def add_gitignore_exceptions(repo: Path) -> None:
    path = repo / ".gitignore"
    text = path.read_text(encoding="utf-8") if path.exists() else ""
    local_defaults = [
        "/.claude/",
        "/.playwright-cli/",
        "/.tmp/",
        "/.harness/reuse-index/",
        "/.harness/runtime-runs/",
        "/backups/",
        "/tmp/",
        "/TODO/",
    ]
    additions = [
        "!.harness/",
        "!.harness/*.md",
        "!harness/",
        "!harness/**",
        "!concepts/",
        "!concepts/*.md",
        "!thinking/",
        "!thinking/*.md",
        "!practice/",
        "!practice/**",
        "!feedback/",
        "!feedback/*.md",
        "!works/",
        "!works/*.md",
        "!prompts/",
        "!prompts/*.md",
        "!references/",
        "!references/*.md",
        "!docs/reviews/general/",
        "!docs/reviews/general/*.md",
    ]
    text = upsert_gitignore_block(text, "local-artifacts", local_defaults)
    text = upsert_gitignore_block(text, "allowlist", additions)
    path.write_text(text, encoding="utf-8")
