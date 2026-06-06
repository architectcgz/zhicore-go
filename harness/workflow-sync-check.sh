#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF' >&2
Usage:
  bash ~/.agents/harness/workflow-sync-check.sh <repo-root> <workflow-name>

Description:
  Check whether a repository still matches a shared workflow package baseline.
EOF
}

if [[ $# -ne 2 ]]; then
  usage
  exit 1
fi

repo_root="$1"
workflow_name="$2"

workflow_root="/home/azhi/.agents/harness/workflows/$workflow_name"
workflow_script="$workflow_root/workflow.sh"

if [[ ! -d "$workflow_root" ]]; then
  echo "FAIL: workflow package not found: $workflow_root" >&2
  exit 1
fi

if [[ ! -x "$workflow_script" ]]; then
  echo "FAIL: workflow entrypoint is missing or not executable: $workflow_script" >&2
  exit 1
fi

exec bash "$workflow_script" "$repo_root" --check
