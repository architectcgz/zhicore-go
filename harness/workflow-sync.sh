#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF' >&2
Usage:
  bash ~/.agents/harness/workflow-sync.sh <repo-root> <workflow-name> [--dry-run]
  bash ~/.agents/harness/workflow-sync.sh <repo-root> <workflow-name> --check

Description:
  Sync a repository to the latest shared workflow package baseline.

Behavior:
  - default: reinstall the shared workflow package into the repository
  - --check: only verify drift, do not write files
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -lt 2 ]]; then
  usage
  exit 1
fi

repo_root="$1"
workflow_name="$2"
shift 2

check_mode=0
extra_args=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --check)
      check_mode=1
      shift
      ;;
    *)
      extra_args+=("$1")
      shift
      ;;
  esac
done

if [[ "$check_mode" -eq 1 ]]; then
  exec bash "$HOME/.agents/harness/workflow-sync-check.sh" "$repo_root" "$workflow_name"
fi

exec bash "$HOME/.agents/harness/workflow-installer.sh" "$repo_root" "$workflow_name" "${extra_args[@]}"
