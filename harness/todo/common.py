from __future__ import annotations

import datetime as dt
import re
import subprocess
from dataclasses import dataclass
from pathlib import Path


WORKSPACE_PROJECTS_ROOT = (Path.home() / "workspace" / "projects").resolve()
PROJECT_TODO_DIR = Path("docs") / "todo"
FILENAME_RE = re.compile(r"^\d{4}-\d{2}-\d{2}-\d{4}-[a-z0-9]+(?:-[a-z0-9]+)*\.md$")


@dataclass(frozen=True)
class TodoScope:
    root: Path
    in_project: bool
    live_dir: Path
    archive_dir: Path
    candidate_dirs: tuple[Path, ...]


def find_project_root(start: Path) -> Path | None:
    current = start.resolve()
    for candidate in [current, *current.parents]:
        if (candidate / ".git").exists():
            return candidate
    try:
        out = subprocess.check_output(
            ["git", "rev-parse", "--show-toplevel"],
            cwd=current,
            text=True,
            stderr=subprocess.DEVNULL,
        ).strip()
        if out:
            return Path(out).resolve()
    except Exception:
        pass
    return None


def resolve_scope(start: Path) -> TodoScope:
    project_root = find_project_root(start)
    if project_root is not None:
        live_dir = choose_project_todo_dir(project_root)
        return TodoScope(
            root=project_root,
            in_project=True,
            live_dir=live_dir,
            archive_dir=live_dir / "archive",
            candidate_dirs=(
                live_dir,
                project_root / "docs" / "todos",
                project_root / "todo",
            ),
        )

    live_dir = WORKSPACE_PROJECTS_ROOT / "todo"
    return TodoScope(
        root=WORKSPACE_PROJECTS_ROOT,
        in_project=False,
        live_dir=live_dir,
        archive_dir=live_dir / "archive",
        candidate_dirs=(live_dir,),
    )


def choose_project_todo_dir(project_root: Path) -> Path:
    return project_root / PROJECT_TODO_DIR


def slugify(text: str) -> str:
    slug = text.strip().lower()
    slug = re.sub(r"[^a-z0-9]+", "-", slug)
    slug = re.sub(r"-{2,}", "-", slug).strip("-")
    return slug or "todo"


def build_filename(title: str, created_at: dt.datetime) -> str:
    stamp = created_at.strftime("%Y-%m-%d-%H%M")
    return f"{stamp}-{slugify(title)}.md"


def parse_items(title: str, notes: str) -> list[str]:
    raw = [line.strip() for line in notes.splitlines() if line.strip()]
    if not raw:
        return [title]
    items: list[str] = []
    for line in raw:
        line = re.sub(r"^- \[(?: |x|X)\]\s+", "", line)
        line = re.sub(r"^[-*]\s+", "", line)
        line = re.sub(r"^\d+\.\s+", "", line)
        if line:
            items.append(line)
    return items or [title]


def is_allowed_todo_path(path: Path, scope: TodoScope) -> bool:
    resolved = path.resolve()
    if resolved == scope.live_dir.resolve() or resolved == scope.archive_dir.resolve():
        return True
    return resolved.is_relative_to(scope.live_dir.resolve()) or resolved.is_relative_to(scope.archive_dir.resolve())


def is_archived_path(path: Path, scope: TodoScope) -> bool:
    resolved = path.resolve()
    return resolved.is_relative_to(scope.archive_dir.resolve())


def filename_has_required_date(path: Path) -> bool:
    return FILENAME_RE.fullmatch(path.name) is not None


def todo_has_open_items(text: str) -> bool:
    return bool(re.search(r"^- \[ \] ", text, re.MULTILINE))


def todo_is_completed(text: str) -> bool:
    return not todo_has_open_items(text) and bool(re.search(r"^- \[[xX]\] ", text, re.MULTILINE))


def update_status_line(text: str, status: str) -> str:
    pattern = re.compile(r"^- Status: `[^`]+`$", re.MULTILINE)
    if pattern.search(text):
        return pattern.sub(f"- Status: `{status}`", text, count=1)
    lines = text.splitlines()
    insert_at = 1 if lines and lines[0].startswith("# ") else 0
    lines[insert_at:insert_at] = ["", f"- Status: `{status}`"]
    return "\n".join(lines).strip() + "\n"


def markdown_files_under(directory: Path) -> list[Path]:
    if not directory.exists():
        return []
    return sorted(path for path in directory.rglob("*.md") if path.is_file())
