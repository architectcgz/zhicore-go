#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF' >&2
Usage:
  bash ~/.agents/harness/init-project.sh <repo-root> [--project-name <name>] [--mode <default|strict-reference>] [--workflow <name>] [--skip-workflow]

Description:
  Initialize project-local harness scaffolding, then optionally install a shared workflow package.

Defaults:
  - mode: default
  - workflow: code-workflow
EOF
}

if [[ $# -lt 1 ]]; then
  usage
  exit 1
fi

repo_root=""
project_name=""
mode="default"
workflow_name="code-workflow"
skip_workflow=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --project-name)
      [[ $# -ge 2 ]] || { echo "FAIL: --project-name requires a value" >&2; exit 1; }
      project_name="$2"
      shift 2
      ;;
    --mode)
      [[ $# -ge 2 ]] || { echo "FAIL: --mode requires a value" >&2; exit 1; }
      mode="$2"
      shift 2
      ;;
    --workflow)
      [[ $# -ge 2 ]] || { echo "FAIL: --workflow requires a value" >&2; exit 1; }
      workflow_name="$2"
      shift 2
      ;;
    --skip-workflow)
      skip_workflow=1
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

repo_root="$(cd "$repo_root" && pwd)"
if [[ ! -d "$repo_root/.git" ]] && ! git -C "$repo_root" rev-parse --show-toplevel >/dev/null 2>&1; then
  echo "FAIL: target is not a git repository: $repo_root" >&2
  exit 1
fi

repo_root="$(git -C "$repo_root" rev-parse --show-toplevel)"
if [[ -z "$project_name" ]]; then
  project_name="$(basename "$repo_root")"
fi

case "$mode" in
  default|strict-reference)
    ;;
  *)
    echo "FAIL: unsupported mode: $mode" >&2
    exit 1
    ;;
esac

echo "[init-project] initialize harness"
python3 ~/.agents/harness/harness-initializer.py \
  --repo "$repo_root" \
  --project-name "$project_name" \
  --mode "$mode"

if [[ "$skip_workflow" -eq 0 ]]; then
  echo "[init-project] install workflow package: $workflow_name"
  bash ~/.agents/harness/workflow-installer.sh "$repo_root" "$workflow_name"
else
  echo "[init-project] skip workflow installation"
fi

if [[ -x "$repo_root/scripts/check-harness-consistency.sh" ]]; then
  echo "[init-project] run project harness consistency check"
  bash "$repo_root/scripts/check-harness-consistency.sh"
fi

echo "[init-project] done"
echo "- repo: $repo_root"
echo "- mode: $mode"
if [[ "$skip_workflow" -eq 0 ]]; then
  echo "- workflow: $workflow_name"
else
  echo "- workflow: skipped"
fi
