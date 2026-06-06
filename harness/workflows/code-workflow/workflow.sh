#!/usr/bin/env bash
set -euo pipefail

SCAFFOLD_VERSION="2026-06-06.5"

usage() {
  cat <<'EOF' >&2
Usage:
  bash ~/.agents/harness/workflow-installer.sh <repo-root> code-workflow [--dry-run]
  bash ~/.agents/harness/workflow-sync-check.sh <repo-root> code-workflow

Description:
  Install or verify the repo-local assets for the shared code-workflow package.
EOF
}

repo_root=""
dry_run=0
check_mode=0
check_fail=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run)
      dry_run=1
      shift
      ;;
    --check)
      check_mode=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --*)
      echo "FAIL: unknown argument: $1" >&2
      usage
      exit 1
      ;;
    *)
      if [[ -n "$repo_root" ]]; then
        echo "FAIL: repo root already set to $repo_root" >&2
        usage
        exit 1
      fi
      repo_root="$1"
      shift
      ;;
  esac
done

if [[ -z "$repo_root" ]]; then
  usage
  exit 1
fi

if [[ "$dry_run" -eq 1 && "$check_mode" -eq 1 ]]; then
  echo "FAIL: --dry-run and --check cannot be used together" >&2
  exit 1
fi

repo_root="$(cd "$repo_root" && pwd)"

if [[ ! -d "$repo_root/.git" ]] && ! git -C "$repo_root" rev-parse --show-toplevel >/dev/null 2>&1; then
  echo "FAIL: target is not a git repository: $repo_root" >&2
  exit 1
fi

repo_root="$(git -C "$repo_root" rev-parse --show-toplevel)"

with_managed_header() {
  local kind="$1"
  local content="$2"
  local marker="Managed by code-workflow package (version: $SCAFFOLD_VERSION)"
  local first_line=""
  local rest=""

  case "$kind" in
    shell|python)
      first_line="${content%%$'\n'*}"
      if [[ "$content" == *$'\n'* ]]; then
        rest="${content#*$'\n'}"
      fi
      printf '%s\n' "$first_line"
      printf '%s\n' "# $marker"
      if [[ -n "$rest" ]]; then
        printf '%s\n' "$rest"
      fi
      ;;
    markdown)
      printf '<!-- %s -->\n' "$marker"
      printf '%s\n' "$content"
      ;;
    *)
      echo "FAIL: unsupported managed header kind: $kind" >&2
      exit 1
      ;;
  esac
}

write_file() {
  local path="$1"
  local content="$2"
  local tmp_file=""

  if [[ "$check_mode" -eq 1 ]]; then
    if [[ ! -f "$path" ]]; then
      echo "FAIL: missing managed workflow file: $path" >&2
      check_fail=1
      return 0
    fi
    tmp_file="$(mktemp)"
    printf '%s\n' "$content" > "$tmp_file"
    if cmp -s "$path" "$tmp_file"; then
      echo "PASS: $path matches shared code-workflow baseline"
    else
      echo "FAIL: $path drifted from shared code-workflow baseline" >&2
      check_fail=1
    fi
    rm -f "$tmp_file"
    return 0
  fi

  if [[ "$dry_run" -eq 1 ]]; then
    echo "DRY RUN: would write $path"
    return 0
  fi
  mkdir -p "$(dirname "$path")"
  printf '%s\n' "$content" > "$path"
}

append_gitignore_line() {
  local line="$1"
  local path="$repo_root/.gitignore"
  if [[ "$check_mode" -eq 1 ]]; then
    if [[ -f "$path" ]] && grep -qxF "$line" "$path"; then
      echo "PASS: .gitignore contains $line"
    else
      echo "FAIL: .gitignore must contain $line" >&2
      check_fail=1
    fi
    return 0
  fi
  if [[ "$dry_run" -eq 1 ]]; then
    echo "DRY RUN: would ensure .gitignore contains $line"
    return 0
  fi
  touch "$path"
  if ! grep -qxF "$line" "$path"; then
    printf '%s\n' "$line" >> "$path"
  fi
}

read -r -d '' CHECK_TASK_INTAKE <<'EOF' || true
#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$script_dir/.."

if [[ -x "scripts/check-open-todos.sh" ]]; then
  bash scripts/check-open-todos.sh --quiet-if-empty
fi

echo "PASS: task intake reminder completed"
echo "- non-trivial or protected implementation should start with: bash scripts/start-implementation.sh <topic-or-slug>"
echo "- before finalizing the plan, run the intake analysis gate: relevant superpowers analysis pass first, then grill-with-docs"
EOF

read -r -d '' START_IMPLEMENTATION <<'EOF' || true
#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOH' >&2
Usage:
  bash scripts/start-implementation.sh <topic-or-slug> [--title <plan-title>] [--base <git-ref>] [--dry-run]
EOH
}

ROOT="$(git rev-parse --show-toplevel)"
REPO_NAME="$(basename "$ROOT")"
WORKSPACE_ROOT="$(dirname "$ROOT")"
WORKTREE_PARENT="${WORKTREE_PARENT:-$WORKSPACE_ROOT/.worktrees/$REPO_NAME}"
PLAN_TEMPLATE="$ROOT/harness/templates/implementation-plan-skeleton.md"

topic_or_slug=""
plan_title=""
base_ref="HEAD"
dry_run=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --title)
      [[ $# -ge 2 ]] || { echo "FAIL: --title requires a value" >&2; exit 1; }
      plan_title="$2"
      shift 2
      ;;
    --base)
      [[ $# -ge 2 ]] || { echo "FAIL: --base requires a git ref" >&2; exit 1; }
      base_ref="$2"
      shift 2
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --*)
      echo "FAIL: unknown arg: $1" >&2
      usage
      exit 1
      ;;
    *)
      if [[ -n "$topic_or_slug" ]]; then
        echo "FAIL: topic-or-slug already set to '$topic_or_slug'" >&2
        usage
        exit 1
      fi
      topic_or_slug="$1"
      shift
      ;;
  esac
done

if [[ -z "$topic_or_slug" ]]; then
  usage
  exit 1
fi

normalize_slug() {
  local raw="$1"
  local cleaned
  cleaned="$(
    printf '%s' "$raw" \
      | tr '[:upper:]' '[:lower:]' \
      | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//; s/-{2,}/-/g'
  )"
  printf '%s' "$cleaned"
}

topic_is_slug=0
if [[ "$topic_or_slug" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9]+(-[a-z0-9]+)*$ ]]; then
  task_slug="$topic_or_slug"
  topic_is_slug=1
else
  normalized_topic="$(normalize_slug "$topic_or_slug")"
  if [[ -z "$normalized_topic" ]]; then
    echo "FAIL: topic '$topic_or_slug' cannot be normalized into a valid slug" >&2
    exit 1
  fi
  task_slug="$(date +%F)-$normalized_topic"
fi

if [[ ! "$task_slug" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9]+(-[a-z0-9]+)*$ ]]; then
  echo "FAIL: task slug must match YYYY-MM-DD-topic format: $task_slug" >&2
  exit 1
fi

if [[ -z "$plan_title" ]]; then
  if [[ "$topic_is_slug" -eq 1 ]]; then
    plan_title="$task_slug"
  else
    plan_title="$topic_or_slug"
  fi
fi

branch_name="task/$task_slug"
worktree_path="$WORKTREE_PARENT/$task_slug"
plan_file_name="$task_slug-implementation-plan.md"
gate_dir=".harness/session-gates"
gate_file_name="$task_slug.json"
started_at="$(date -u +%FT%TZ)"

created_worktree=0
created_plan=0
created_gate=0
plan_path=""
gate_path=""
initial_plan_sha=""

cleanup() {
  local status=$?
  if [[ "$status" -eq 0 ]]; then
    return 0
  fi

  if [[ "$created_gate" -eq 1 && -n "$gate_path" && -f "$gate_path" ]]; then
    rm -f "$gate_path"
  fi

  if [[ "$created_plan" -eq 1 && -n "$plan_path" && -f "$plan_path" ]]; then
    local current_sha
    current_sha="$(sha256sum "$plan_path" | awk '{print $1}')"
    if [[ "$current_sha" == "$initial_plan_sha" ]]; then
      rm -f "$plan_path"
    fi
  fi

  if [[ "$created_worktree" -eq 1 && -d "$worktree_path" ]]; then
    if [[ -z "$(git -C "$worktree_path" status --porcelain 2>/dev/null || true)" ]]; then
      git -C "$ROOT" worktree remove --force "$worktree_path" >/dev/null 2>&1 || true
    fi
  fi

  return "$status"
}

trap cleanup EXIT

if [[ ! -f "$PLAN_TEMPLATE" ]]; then
  echo "FAIL: missing implementation plan template: $PLAN_TEMPLATE" >&2
  exit 1
fi

if git -C "$ROOT" rev-parse --verify --quiet "$branch_name" >/dev/null; then
  echo "FAIL: branch already exists: $branch_name" >&2
  exit 1
fi

if [[ -e "$worktree_path" ]]; then
  echo "FAIL: worktree path already exists: $worktree_path" >&2
  exit 1
fi

mkdir -p "$WORKTREE_PARENT"

bash "$ROOT/scripts/check-task-intake.sh"

git -C "$ROOT" rev-parse --verify --quiet "$base_ref" >/dev/null || {
  echo "FAIL: base ref does not exist: $base_ref" >&2
  exit 1
}

if [[ "$dry_run" -eq 1 ]]; then
  printf '%s\n' "DRY RUN: implementation workspace would be initialized"
  printf '%s\n' "- task slug: $task_slug"
  printf '%s\n' "- worktree: $worktree_path"
  printf '%s\n' "- branch: $branch_name"
  printf '%s\n' "- plan: docs/plan/impl-plan/$plan_file_name"
  printf '%s\n' "- gate: $gate_dir/$gate_file_name"
  exit 0
fi

git -C "$ROOT" worktree add -b "$branch_name" "$worktree_path" "$base_ref" >/dev/null
created_worktree=1

plan_path="$worktree_path/docs/plan/impl-plan/$plan_file_name"
gate_path="$worktree_path/$gate_dir/$gate_file_name"

mkdir -p "$(dirname "$plan_path")" "$(dirname "$gate_path")"

sed \
  -e "s#__TASK_TITLE__#$(printf '%s' "$plan_title" | sed 's/[&/]/\\&/g')#g" \
  -e "s#__TASK_SLUG__#$(printf '%s' "$task_slug" | sed 's/[&/]/\\&/g')#g" \
  -e "s#__STARTED_AT__#$(printf '%s' "$started_at" | sed 's/[&/]/\\&/g')#g" \
  -e "s#__WORKTREE_PATH__#$(printf '%s' "$worktree_path" | sed 's/[&/]/\\&/g')#g" \
  -e "s#__BRANCH_NAME__#$(printf '%s' "$branch_name" | sed 's/[&/]/\\&/g')#g" \
  "$PLAN_TEMPLATE" > "$plan_path"
created_plan=1
initial_plan_sha="$(sha256sum "$plan_path" | awk '{print $1}')"

cat > "$gate_path" <<EOG
{
  "task_slug": "$task_slug",
  "status": "active",
  "started_at": "$started_at",
  "repo_root": "$ROOT",
  "worktree_path": "$worktree_path",
  "branch": "$branch_name",
  "plan_path": "docs/plan/impl-plan/$plan_file_name"
}
EOG
created_gate=1

printf '%s\n' "PASS: implementation workspace initialized"
printf '%s\n' "- task slug: $task_slug"
printf '%s\n' "- worktree: $worktree_path"
printf '%s\n' "- branch: $branch_name"
printf '%s\n' "- plan: docs/plan/impl-plan/$plan_file_name"
printf '%s\n' "- gate: $gate_dir/$gate_file_name"
printf '\n'
printf '%s\n' "Next steps:"
printf '%s\n' "1. cd $worktree_path"
printf '%s\n' "2. Run the intake analysis gate: relevant superpowers analysis pass first, then grill-with-docs"
printf '%s\n' "3. Complete the plan via superpowers:writing-plans using that analysis output"
printf '%s\n' "4. Start implementation only after the plan is complete enough for the current slice"
EOF

read -r -d '' CHECK_STARTUP_GATE_SH <<'EOF' || true
#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$script_dir/.."

python3 harness/checks/check_startup_gate.py "$@"
EOF

read -r -d '' RUN_WORKFLOW_STAGE <<'EOF' || true
#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOH' >&2
Usage:
  bash scripts/run-workflow-stage.sh <stage>

Stages:
  pre-commit-quick
  completion-full
  workflow-governance
EOH
}

changed_files() {
  local staged
  staged="$(git diff --cached --name-only)"
  if [[ -n "$staged" ]]; then
    printf '%s\n' "$staged" | sort -u
    return
  fi

  git diff --name-only | sort -u
}

red() { printf '\033[31m%s\033[0m' "$1"; }
green() { printf '\033[32m%s\033[0m' "$1"; }

if [[ $# -lt 1 ]]; then
  usage
  exit 2
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
REQUESTED_STAGE="$1"
shift

case "$REQUESTED_STAGE" in
  review-governance)
    STAGE="workflow-governance"
    ;;
  *)
    STAGE="$REQUESTED_STAGE"
    ;;
esac

PLUGIN_DIR="$ROOT_DIR/harness/workflow-plugins/code-workflow/${STAGE}.d"

cd "$ROOT_DIR"

if [[ ! -d "$PLUGIN_DIR" ]]; then
  echo "[workflow-stage] $STAGE"
  echo "  $(green PASS) — no plugins registered"
  exit 0
fi

mapfile -t plugins < <(find "$PLUGIN_DIR" -maxdepth 1 -type f -name '*.sh' | sort)

if [[ "${#plugins[@]}" -eq 0 ]]; then
  echo "[workflow-stage] $STAGE"
  echo "  $(green PASS) — plugin directory is empty"
  exit 0
fi

export WORKFLOW_STAGE="$STAGE"
export WORKFLOW_REPO_ROOT="$ROOT_DIR"
export WORKFLOW_CHANGED_FILES="$(changed_files)"
export WORKFLOW_TASK_SLUG="$(
  if [[ -x "$ROOT_DIR/scripts/check-startup-gate.sh" ]]; then
    bash "$ROOT_DIR/scripts/check-startup-gate.sh" --print-active-slug 2>/dev/null || true
  fi
)"

fail=0

echo "[workflow-stage] $STAGE"
if [[ "$REQUESTED_STAGE" != "$STAGE" ]]; then
  echo "  [alias] $REQUESTED_STAGE -> $STAGE"
fi

for plugin in "${plugins[@]}"; do
  label="$(basename "$plugin")"
  if [[ ! -x "$plugin" ]]; then
    echo "  $(red FAIL) — $label is not executable"
    fail=1
    continue
  fi

  echo "  [plugin] $label"
  if "$plugin" "$@"; then
    echo "    $(green PASS) — $label"
  else
    echo "    $(red FAIL) — $label"
    fail=1
  fi
done

if [[ "$fail" -eq 0 ]]; then
  echo "$(green "✓ workflow stage passed: $STAGE")"
else
  echo "$(red "✗ workflow stage failed: $STAGE")"
fi

exit "$fail"
EOF

read -r -d '' ARCHIVE_TASK_ARTIFACTS <<'EOF' || true
#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOH' >&2
Usage:
  bash harness/workflow-plugins/code-workflow/archive_task_artifacts.sh [--task-slug <slug>] [--plan <path>] [--task <path> ...] [--dry-run]

Description:
  Archive the completed implementation plan and any matching docs/tasks artifacts for a task.
  If --task-slug is omitted, the script will try to use the current active startup gate.
EOH
}

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

task_slug=""
plan_path=""
dry_run=0
declare -a explicit_task_paths=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --task-slug)
      [[ $# -ge 2 ]] || { echo "FAIL: --task-slug requires a value" >&2; exit 1; }
      task_slug="$2"
      shift 2
      ;;
    --plan)
      [[ $# -ge 2 ]] || { echo "FAIL: --plan requires a value" >&2; exit 1; }
      plan_path="$2"
      shift 2
      ;;
    --task)
      [[ $# -ge 2 ]] || { echo "FAIL: --task requires a path" >&2; exit 1; }
      explicit_task_paths+=("$2")
      shift 2
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --*)
      echo "FAIL: unknown arg: $1" >&2
      usage
      exit 1
      ;;
    *)
      echo "FAIL: unexpected positional arg: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if [[ -z "$task_slug" && -x "scripts/check-startup-gate.sh" ]]; then
  task_slug="$(bash scripts/check-startup-gate.sh --print-active-slug 2>/dev/null || true)"
fi

if [[ -z "$task_slug" ]]; then
  echo "FAIL: unable to resolve task slug; pass --task-slug or run inside a worktree with an active startup gate" >&2
  exit 1
fi

if [[ ! "$task_slug" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9]+(-[a-z0-9]+)*$ ]]; then
  echo "FAIL: invalid task slug: $task_slug" >&2
  exit 1
fi

active_gate_path=""
if [[ -x "scripts/check-startup-gate.sh" ]]; then
  active_gate_path="$(bash scripts/check-startup-gate.sh --print-gate-path 2>/dev/null || true)"
fi

if [[ -z "$plan_path" ]]; then
  if [[ -n "$active_gate_path" && -f "$active_gate_path" ]]; then
    plan_path="$(
      python3 - "$active_gate_path" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
print(payload.get("plan_path", ""))
PY
    )"
  fi

  if [[ -z "$plan_path" ]]; then
    plan_path="docs/plan/impl-plan/${task_slug}-implementation-plan.md"
  fi
fi

if [[ ! -f "$plan_path" ]]; then
  echo "FAIL: plan file not found: $plan_path" >&2
  exit 1
fi

archive_month="$(printf '%s' "$task_slug" | cut -d- -f1,2)"
plan_archive_dir="docs/plan/archive/impl-plan/$archive_month"
plan_archive_path="$plan_archive_dir/$(basename "$plan_path")"

declare -a task_paths=()
declare -a task_archive_paths=()

if [[ "${#explicit_task_paths[@]}" -gt 0 ]]; then
  for path in "${explicit_task_paths[@]}"; do
    task_paths+=("$path")
  done
elif [[ -d "docs/tasks" ]]; then
  while IFS= read -r path; do
    [[ -z "$path" ]] && continue
    task_paths+=("$path")
  done < <(find "docs/tasks" \
    -path "docs/tasks/archive" -prune -o \
    -type f -name "*${task_slug}*.md" -print | sort)
fi

for path in "${task_paths[@]}"; do
  if [[ ! -f "$path" ]]; then
    echo "FAIL: task file not found: $path" >&2
    exit 1
  fi
  task_archive_paths+=("docs/tasks/archive/$archive_month/$(basename "$path")")
done

if [[ -e "$plan_archive_path" ]]; then
  echo "FAIL: archived plan already exists: $plan_archive_path" >&2
  exit 1
fi

for path in "${task_archive_paths[@]}"; do
  if [[ -e "$path" ]]; then
    echo "FAIL: archived task file already exists: $path" >&2
    exit 1
  fi
done

if [[ "$dry_run" -eq 1 ]]; then
  printf '%s\n' "DRY RUN: task artifacts would be archived"
  printf '%s\n' "- task slug: $task_slug"
  printf '%s\n' "- plan: $plan_path -> $plan_archive_path"
  if [[ "${#task_paths[@]}" -eq 0 ]]; then
    printf '%s\n' "- tasks: none"
  else
    for i in "${!task_paths[@]}"; do
      printf '%s\n' "- task: ${task_paths[$i]} -> ${task_archive_paths[$i]}"
    done
  fi
  exit 0
fi

mkdir -p "$plan_archive_dir"
mv "$plan_path" "$plan_archive_path"

if [[ "${#task_paths[@]}" -gt 0 ]]; then
  mkdir -p "docs/tasks/archive/$archive_month"
  for i in "${!task_paths[@]}"; do
    mv "${task_paths[$i]}" "${task_archive_paths[$i]}"
  done
fi

if [[ -n "$active_gate_path" && -f "$active_gate_path" ]]; then
  python3 - "$active_gate_path" "$task_slug" "$plan_archive_path" "${task_archive_paths[@]}" <<'PY'
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

gate_path = Path(sys.argv[1])
task_slug = sys.argv[2]
plan_archive_path = sys.argv[3]
task_archive_paths = sys.argv[4:]

payload = json.loads(gate_path.read_text(encoding="utf-8"))
if payload.get("task_slug") == task_slug and payload.get("status") == "active":
    payload["status"] = "archived"
    payload["archived_at"] = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    payload["archived_plan_path"] = plan_archive_path
    payload["archived_task_paths"] = task_archive_paths
    gate_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY
fi

printf '%s\n' "PASS: task artifacts archived"
printf '%s\n' "- task slug: $task_slug"
printf '%s\n' "- plan: $plan_archive_path"
if [[ "${#task_archive_paths[@]}" -eq 0 ]]; then
  printf '%s\n' "- tasks: none"
else
  for path in "${task_archive_paths[@]}"; do
    printf '%s\n' "- task: $path"
  done
fi
EOF

read -r -d '' CHECK_STARTUP_GATE_PY <<'EOF' || true
#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SESSION_GATES_DIR = ROOT / ".harness" / "session-gates"
TASK_SLUG_RE = re.compile(r"^[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9]+(?:-[a-z0-9]+)*$")
REQUIRED_PLAN_HEADINGS = (
    "## Task Metadata",
    "## Task Classification",
    "## Files",
    "## 复用与 Owner 决策",
    "## Intake Analysis Gate",
    "## Validation",
)
PLACEHOLDER_TOKENS = ("TODO", "待填写", "__TASK_", "__STARTED_AT__", "__WORKTREE_PATH__", "__BRANCH_NAME__")
LOW_RISK_PREFIXES = (
    "docs/",
    "README",
    ".gitignore",
)
LOW_RISK_SUFFIXES = (
    ".md",
    ".txt",
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="validate local startup gate state for non-trivial work")
    parser.add_argument("--print-active-slug", action="store_true")
    parser.add_argument("--print-gate-path", action="store_true")
    parser.add_argument("--quiet", action="store_true")
    parser.add_argument("--staged", action="store_true")
    parser.add_argument("--base")
    parser.add_argument("--head", default="HEAD")
    args = parser.parse_args()
    if args.staged and args.base:
        parser.error("--staged and --base cannot be used together")
    return args


def run_git(*args: str) -> str:
    result = subprocess.run(["git", *args], cwd=ROOT, check=True, capture_output=True, text=True)
    return result.stdout


def changed_paths(args: argparse.Namespace) -> list[str]:
    if args.base:
      output = run_git("diff", "--name-only", "--diff-filter=ACMR", f"{args.base}...{args.head}")
    else:
      output = run_git("diff", "--cached", "--name-only", "--diff-filter=ACMR")
    return [line.strip() for line in output.splitlines() if line.strip()]


def requires_gate(path: str) -> bool:
    if path.startswith(LOW_RISK_PREFIXES) or path.endswith(LOW_RISK_SUFFIXES):
        return False
    return True


def load_active_gates() -> list[tuple[Path, dict[str, object]]]:
    gates: list[tuple[Path, dict[str, object]]] = []
    if not SESSION_GATES_DIR.is_dir():
        return gates
    for path in sorted(SESSION_GATES_DIR.glob("*.json")):
        if not path.is_file():
            continue
        try:
            payload = json.loads(path.read_text(encoding="utf-8"))
        except (json.JSONDecodeError, OSError):
            raise SystemExit(f"FAIL: invalid startup gate file: {path.relative_to(ROOT).as_posix()}")
        if payload.get("status") == "active":
            gates.append((path, payload))
    return gates


def contains_placeholder(text: str) -> bool:
    return any(token in text for token in PLACEHOLDER_TOKENS)


def extract_section(plan_text: str, heading: str) -> str:
    pattern = re.compile(rf"^{re.escape(heading)}\s*$([\s\S]*?)(?=^## |\Z)", re.MULTILINE)
    match = pattern.search(plan_text)
    return match.group(1).strip() if match else ""


def validate_active_gate(path: Path, payload: dict[str, object], *, require_completed_plan: bool) -> list[str]:
    errors: list[str] = []

    task_slug = payload.get("task_slug")
    if not isinstance(task_slug, str) or not TASK_SLUG_RE.fullmatch(task_slug):
        errors.append("task_slug missing or invalid")

    branch = payload.get("branch")
    if not isinstance(branch, str) or not branch.startswith("task/"):
        errors.append("branch missing or invalid")

    plan_path_value = payload.get("plan_path")
    if not isinstance(plan_path_value, str):
        errors.append("plan_path missing")
        return errors

    plan_path = ROOT / plan_path_value
    if not plan_path.is_file():
        errors.append(f"plan file missing: {plan_path_value}")
        return errors

    plan_text = plan_path.read_text(encoding="utf-8")
    for heading in REQUIRED_PLAN_HEADINGS:
        if heading not in plan_text:
            errors.append(f"plan missing required heading: {heading}")

    if "**Goal:**" not in plan_text or "**Architecture:**" not in plan_text:
        errors.append("plan missing required summary fields")
    elif require_completed_plan:
        summary_lines = "\n".join(
            line for line in plan_text.splitlines() if line.startswith("**Goal:**") or line.startswith("**Architecture:**")
        )
        if contains_placeholder(summary_lines):
            errors.append("plan summary fields still contain placeholders")

    if require_completed_plan:
        for heading in ("## Task Classification", "## Files", "## 复用与 Owner 决策", "## Intake Analysis Gate", "## Validation"):
            section_text = extract_section(plan_text, heading)
            if not section_text:
                errors.append(f"plan section is empty: {heading}")
                continue
            if contains_placeholder(section_text):
                errors.append(f"plan section still contains placeholders: {heading}")

    return errors


def main() -> int:
    args = parse_args()
    gates = load_active_gates()

    if args.print_active_slug or args.print_gate_path:
        if not gates:
            return 1
        if len(gates) > 1:
            print("FAIL: multiple active startup gates in current worktree", file=sys.stderr)
            return 1
        gate_path, payload = gates[0]
        errors = validate_active_gate(gate_path, payload, require_completed_plan=False)
        if errors:
            print("FAIL: active startup gate is invalid", file=sys.stderr)
            for error in errors:
                print(f"- {error}", file=sys.stderr)
            return 1
        print(payload["task_slug"] if args.print_active_slug else gate_path.relative_to(ROOT).as_posix())
        return 0

    changed = changed_paths(args)
    gated = sorted(path for path in changed if requires_gate(path))

    if not gated:
        if not args.quiet:
            print("PASS: no startup-gated changes in diff")
        return 0

    if not gates:
        print("FAIL: startup-gated changes require an active task gate", file=sys.stderr)
        for path in gated:
            print(f"- {path}", file=sys.stderr)
        print("Use scripts/start-implementation.sh before continuing.", file=sys.stderr)
        return 1

    if len(gates) > 1:
        print("FAIL: multiple active startup gates in current worktree", file=sys.stderr)
        return 1

    gate_path, payload = gates[0]
    plan_path_value = payload.get("plan_path")
    requires_completed_plan = any(path != plan_path_value for path in gated)
    errors = validate_active_gate(gate_path, payload, require_completed_plan=requires_completed_plan)
    if errors:
        print("FAIL: active startup gate is invalid", file=sys.stderr)
        print(f"- gate: {gate_path.relative_to(ROOT).as_posix()}", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    if not args.quiet:
        print("PASS: startup gate covers current diff")
        print(f"- gate: {gate_path.relative_to(ROOT).as_posix()}")
        print(f"- task slug: {payload['task_slug']}")
        print(f"- plan: {payload['plan_path']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
EOF

read -r -d '' IMPLEMENTATION_PLAN_SKELETON <<'EOF' || true
# __TASK_TITLE__ Implementation Plan

**Goal:** TODO

**Architecture:** TODO

**Tech Stack:** TODO

---

## Task Metadata

- Task Slug: `__TASK_SLUG__`
- Started At: `__STARTED_AT__`
- Worktree: `__WORKTREE_PATH__`
- Branch: `__BRANCH_NAME__`

## Objective And Non-Goals

- Objective:
- Non-Goals:

## Inputs

- Source docs:
- Related architecture/contracts:
- Related prior work:

## Task Classification

- Classification: `非琐碎任务`
- Why:

## Files

- Create:
- Modify:
- Review:
- Test:

## 复用与 Owner 决策

- Existing patterns searched:
- Reuse / extend / split / create-new decision:
- Owner boundary:
- Why this is the narrowest safe surface:

## Intake Analysis Gate

- Relevant superpowers analysis pass:
- Why this pass fits:
- grill-with-docs findings:
- Plan adjustments after challenge:

## Validation

- Commands:
- Manual checks:
- Review focus:
EOF

CHECK_TASK_INTAKE="$(with_managed_header shell "$CHECK_TASK_INTAKE")"
START_IMPLEMENTATION="$(with_managed_header shell "$START_IMPLEMENTATION")"
CHECK_STARTUP_GATE_SH="$(with_managed_header shell "$CHECK_STARTUP_GATE_SH")"
RUN_WORKFLOW_STAGE="$(with_managed_header shell "$RUN_WORKFLOW_STAGE")"
ARCHIVE_TASK_ARTIFACTS="$(with_managed_header shell "$ARCHIVE_TASK_ARTIFACTS")"
CHECK_STARTUP_GATE_PY="$(with_managed_header python "$CHECK_STARTUP_GATE_PY")"
IMPLEMENTATION_PLAN_SKELETON="$(with_managed_header markdown "$IMPLEMENTATION_PLAN_SKELETON")"

write_file "$repo_root/scripts/check-task-intake.sh" "$CHECK_TASK_INTAKE"
write_file "$repo_root/scripts/start-implementation.sh" "$START_IMPLEMENTATION"
write_file "$repo_root/scripts/check-startup-gate.sh" "$CHECK_STARTUP_GATE_SH"
write_file "$repo_root/harness/workflow-plugins/code-workflow/run_workflow_stage.sh" "$RUN_WORKFLOW_STAGE"
write_file "$repo_root/harness/workflow-plugins/code-workflow/archive_task_artifacts.sh" "$ARCHIVE_TASK_ARTIFACTS"
write_file "$repo_root/harness/checks/check_startup_gate.py" "$CHECK_STARTUP_GATE_PY"
write_file "$repo_root/harness/templates/implementation-plan-skeleton.md" "$IMPLEMENTATION_PLAN_SKELETON"
append_gitignore_line "/.harness/session-gates/"

if [[ "$check_mode" -eq 1 ]]; then
  if [[ -f "$repo_root/scripts/archive-task-artifacts.sh" ]]; then
    echo "FAIL: legacy archive entry must be removed: $repo_root/scripts/archive-task-artifacts.sh" >&2
    check_fail=1
  fi
  if [[ -d "$repo_root/.harness/session-gates" ]]; then
    echo "PASS: .harness/session-gates directory exists"
  else
    echo "FAIL: missing .harness/session-gates directory" >&2
    check_fail=1
  fi

  if [[ "$check_fail" -eq 0 ]]; then
    echo "PASS: shared code-workflow package is in sync for $repo_root"
  else
    echo "FAIL: shared code-workflow package drift detected in $repo_root" >&2
  fi
  exit "$check_fail"
elif [[ "$dry_run" -eq 1 ]]; then
  echo "DRY RUN: code-workflow package install checked"
else
  chmod +x \
    "$repo_root/scripts/check-task-intake.sh" \
    "$repo_root/scripts/start-implementation.sh" \
    "$repo_root/scripts/check-startup-gate.sh" \
    "$repo_root/harness/workflow-plugins/code-workflow/run_workflow_stage.sh" \
    "$repo_root/harness/workflow-plugins/code-workflow/archive_task_artifacts.sh" \
    "$repo_root/harness/checks/check_startup_gate.py"
  rm -f "$repo_root/scripts/archive-task-artifacts.sh"
  mkdir -p "$repo_root/.harness/session-gates"
  echo "PASS: code-workflow package installed in $repo_root"
fi
