#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOH' >&2
Usage:
  bash harness/workflow-plugins/code-workflow/cleanup_task_worktree.sh [--task-slug <slug>] [--branch <branch>] [--worktree <path>] [--merged-into <git-ref>] [--delete-branch] [--dry-run]

Description:
  Safely close a dedicated task worktree after the task has already been archived
  and merged. By default, the script requires the task startup gate to be in
  ready_to_merge status, the worktree to be clean, and the task HEAD to already
  be merged into the target ref (default: HEAD).

  When the task was executed directly in the main worktree, this script will not
  remove the current repository root. It only updates the local startup gate to
  archived so the task no longer appears as an effective gate.
EOH
}

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

task_slug=""
branch_name=""
worktree_path=""
merged_into_ref="HEAD"
delete_branch=0
dry_run=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --task-slug)
      [[ $# -ge 2 ]] || { echo "FAIL: --task-slug requires a value" >&2; exit 1; }
      task_slug="$2"
      shift 2
      ;;
    --branch)
      [[ $# -ge 2 ]] || { echo "FAIL: --branch requires a value" >&2; exit 1; }
      branch_name="$2"
      shift 2
      ;;
    --worktree)
      [[ $# -ge 2 ]] || { echo "FAIL: --worktree requires a path" >&2; exit 1; }
      worktree_path="$2"
      shift 2
      ;;
    --merged-into)
      [[ $# -ge 2 ]] || { echo "FAIL: --merged-into requires a git ref" >&2; exit 1; }
      merged_into_ref="$2"
      shift 2
      ;;
    --delete-branch)
      delete_branch=1
      shift
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

if [[ -z "$branch_name" && -n "$task_slug" ]]; then
  branch_name="task/$task_slug"
fi

if [[ -z "$task_slug" && -n "$branch_name" && "$branch_name" =~ ^task/(.+)$ ]]; then
  task_slug="${BASH_REMATCH[1]}"
fi

if [[ -z "$task_slug" ]]; then
  echo "FAIL: unable to resolve task slug; pass --task-slug or run inside the task worktree" >&2
  exit 1
fi

if [[ ! "$task_slug" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9]+(-[a-z0-9]+)*$ ]]; then
  echo "FAIL: invalid task slug: $task_slug" >&2
  exit 1
fi

if [[ -z "$branch_name" ]]; then
  branch_name="task/$task_slug"
fi

git rev-parse --verify --quiet "$merged_into_ref" >/dev/null || {
  echo "FAIL: merged target ref does not exist: $merged_into_ref" >&2
  exit 1
}

resolve_worktree_path() {
  local branch="$1"
  python3 - "$branch" <<'PY'
import subprocess
import sys

branch = sys.argv[1]
output = subprocess.run(
    ["git", "worktree", "list", "--porcelain"],
    check=True,
    capture_output=True,
    text=True,
).stdout.splitlines()

entries = []
current = {}
for line in output:
    if not line:
        if current:
            entries.append(current)
            current = {}
        continue
    key, _, value = line.partition(" ")
    current[key] = value
if current:
    entries.append(current)

for entry in entries:
    if entry.get("branch") == f"refs/heads/{branch}":
        print(entry.get("worktree", ""))
        break
PY
}

if [[ -z "$worktree_path" ]]; then
  worktree_path="$(resolve_worktree_path "$branch_name")"
fi

if [[ -z "$worktree_path" ]]; then
  echo "FAIL: unable to resolve worktree path for branch: $branch_name" >&2
  exit 1
fi

if [[ ! -d "$worktree_path" ]]; then
  echo "FAIL: worktree path not found: $worktree_path" >&2
  exit 1
fi

worktree_path="$(cd "$worktree_path" && pwd)"
gate_path="$worktree_path/.harness/session-gates/$task_slug.json"

if [[ ! -f "$gate_path" ]]; then
  echo "FAIL: task gate not found: $gate_path" >&2
  exit 1
fi

gate_status=""
gate_branch=""
gate_worktree_path=""
task_head=""
readarray -t gate_fields < <(
  python3 - "$gate_path" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
print(payload.get("status", ""))
print(payload.get("branch", ""))
print(payload.get("worktree_path", ""))
PY
)
gate_status="${gate_fields[0]:-}"
gate_branch="${gate_fields[1]:-}"
gate_worktree_path="${gate_fields[2]:-}"

if [[ "$gate_status" != "ready_to_merge" ]]; then
  echo "FAIL: task gate must be ready_to_merge before cleanup: $gate_status" >&2
  echo "Hint: archive task artifacts first with harness/workflow-plugins/code-workflow/archive_task_artifacts.sh" >&2
  exit 1
fi

if [[ -n "$gate_branch" && "$gate_branch" != "$branch_name" ]]; then
  echo "FAIL: gate branch mismatch: expected $branch_name, got $gate_branch" >&2
  exit 1
fi

if [[ -n "$gate_worktree_path" ]]; then
  normalized_gate_worktree_path="$(cd "$gate_worktree_path" && pwd)"
  if [[ "$normalized_gate_worktree_path" != "$worktree_path" ]]; then
    echo "FAIL: gate worktree path mismatch: expected $worktree_path, got $normalized_gate_worktree_path" >&2
    exit 1
  fi
fi

if [[ -n "$(git -C "$worktree_path" status --porcelain)" ]]; then
  echo "FAIL: worktree has uncommitted changes: $worktree_path" >&2
  echo "Hint: commit, move, or discard those changes before cleanup." >&2
  exit 1
fi

task_head="$(git -C "$worktree_path" rev-parse HEAD)"
if ! git merge-base --is-ancestor "$task_head" "$merged_into_ref"; then
  echo "FAIL: task head is not merged into $merged_into_ref" >&2
  exit 1
fi

main_worktree=0
if [[ "$worktree_path" == "$ROOT" ]]; then
  main_worktree=1
fi

if [[ "$dry_run" -eq 1 ]]; then
  printf '%s\n' "DRY RUN: task worktree cleanup would proceed"
  printf '%s\n' "- task slug: $task_slug"
  printf '%s\n' "- branch: $branch_name"
  printf '%s\n' "- worktree: $worktree_path"
  printf '%s\n' "- merged into: $merged_into_ref"
  if [[ "$main_worktree" -eq 1 ]]; then
    printf '%s\n' "- action: mark startup gate archived in current worktree"
  else
    printf '%s\n' "- action: remove dedicated worktree"
  fi
  if [[ "$delete_branch" -eq 1 ]]; then
    printf '%s\n' "- action: delete branch after cleanup"
  fi
  exit 0
fi

mark_gate_archived() {
  local path="$1"
  local merged_ref="$2"
  local removed_worktree="$3"
  python3 - "$path" "$merged_ref" "$removed_worktree" <<'PY'
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

gate_path = Path(sys.argv[1])
merged_ref = sys.argv[2]
removed_worktree = sys.argv[3] == "1"
payload = json.loads(gate_path.read_text(encoding="utf-8"))
payload["status"] = "archived"
payload["closed_at"] = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
payload["merged_into"] = merged_ref
payload["removed_worktree"] = removed_worktree
gate_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY
}

if [[ "$main_worktree" -eq 1 ]]; then
  mark_gate_archived "$gate_path" "$merged_into_ref" 0
else
  mark_gate_archived "$gate_path" "$merged_into_ref" 1
  git -C "$ROOT" worktree remove "$worktree_path"
fi

if [[ "$delete_branch" -eq 1 ]] && git rev-parse --verify --quiet "$branch_name" >/dev/null; then
  git -C "$ROOT" branch -d "$branch_name"
fi

printf '%s\n' "PASS: task worktree cleanup complete"
printf '%s\n' "- task slug: $task_slug"
printf '%s\n' "- branch: $branch_name"
printf '%s\n' "- merged into: $merged_into_ref"
if [[ "$main_worktree" -eq 1 ]]; then
  printf '%s\n' "- startup gate archived in current worktree"
else
  printf '%s\n' "- removed worktree: $worktree_path"
fi
if [[ "$delete_branch" -eq 1 ]]; then
  printf '%s\n' "- deleted branch: $branch_name"
fi
