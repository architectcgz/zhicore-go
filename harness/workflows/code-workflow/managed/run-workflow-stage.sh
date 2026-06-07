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
