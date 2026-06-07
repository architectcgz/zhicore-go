#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOH' >&2
Usage:
  bash harness/workflow-plugins/code-workflow/archive_task_artifacts.sh [--task-slug <slug>] [--plan <path>] [--task <path> ...] [--dry-run]

Description:
  Archive the completed implementation plan and any matching docs/tasks artifacts for a task.
  If --task-slug is omitted, the script will try to use the current active startup gate.
  When the current worktree owns that gate, archiving moves it from active to ready_to_merge.
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
    payload["status"] = "ready_to_merge"
    payload["archived_at"] = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    payload["plan_path"] = plan_archive_path
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
