#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANAGED_DIR="$SCRIPT_DIR/managed"
SCAFFOLD_VERSION="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["version"])' "$SCRIPT_DIR/manifest.json")"

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

read_managed_source() {
  local relative_path="$1"
  local path="$MANAGED_DIR/$relative_path"
  if [[ ! -f "$path" ]]; then
    echo "FAIL: missing managed workflow source: $path" >&2
    exit 1
  fi
  cat "$path"
}

CHECK_TASK_INTAKE="$(with_managed_header shell "$(read_managed_source "check-task-intake.sh")")"
START_IMPLEMENTATION="$(with_managed_header shell "$(read_managed_source "start-implementation.sh")")"
CHECK_STARTUP_GATE_SH="$(with_managed_header shell "$(read_managed_source "check-startup-gate.sh")")"
CHECK_EPIC_DEPENDENCIES="$(with_managed_header shell "$(read_managed_source "check-epic-dependencies.sh")")"
RUN_WORKFLOW_STAGE="$(with_managed_header shell "$(read_managed_source "run-workflow-stage.sh")")"
ARCHIVE_TASK_ARTIFACTS="$(with_managed_header shell "$(read_managed_source "archive-task-artifacts.sh")")"
CLEANUP_TASK_WORKTREE="$(with_managed_header shell "$(read_managed_source "cleanup-task-worktree.sh")")"
CHECK_STARTUP_GATE_PY="$(with_managed_header python "$(read_managed_source "check_startup_gate.py")")"
IMPLEMENTATION_PLAN_SKELETON="$(with_managed_header markdown "$(read_managed_source "implementation-plan-skeleton.md")")"
EPIC_INDEX_SKELETON="$(with_managed_header markdown "$(read_managed_source "epic-index-skeleton.md")")"

write_file "$repo_root/scripts/check-task-intake.sh" "$CHECK_TASK_INTAKE"
write_file "$repo_root/scripts/start-implementation.sh" "$START_IMPLEMENTATION"
write_file "$repo_root/scripts/check-startup-gate.sh" "$CHECK_STARTUP_GATE_SH"
write_file "$repo_root/scripts/check-epic-dependencies.sh" "$CHECK_EPIC_DEPENDENCIES"
write_file "$repo_root/harness/workflow-plugins/code-workflow/run_workflow_stage.sh" "$RUN_WORKFLOW_STAGE"
write_file "$repo_root/harness/workflow-plugins/code-workflow/archive_task_artifacts.sh" "$ARCHIVE_TASK_ARTIFACTS"
write_file "$repo_root/harness/workflow-plugins/code-workflow/cleanup_task_worktree.sh" "$CLEANUP_TASK_WORKTREE"
write_file "$repo_root/harness/checks/check_startup_gate.py" "$CHECK_STARTUP_GATE_PY"
write_file "$repo_root/harness/templates/implementation-plan-skeleton.md" "$IMPLEMENTATION_PLAN_SKELETON"
write_file "$repo_root/harness/templates/epic-index-skeleton.md" "$EPIC_INDEX_SKELETON"
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
    "$repo_root/scripts/check-epic-dependencies.sh" \
    "$repo_root/harness/workflow-plugins/code-workflow/run_workflow_stage.sh" \
    "$repo_root/harness/workflow-plugins/code-workflow/archive_task_artifacts.sh" \
    "$repo_root/harness/workflow-plugins/code-workflow/cleanup_task_worktree.sh" \
    "$repo_root/harness/checks/check_startup_gate.py"
  rm -f "$repo_root/scripts/archive-task-artifacts.sh"
  mkdir -p "$repo_root/.harness/session-gates"
  echo "PASS: code-workflow package installed in $repo_root"
fi
