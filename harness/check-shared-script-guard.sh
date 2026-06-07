#!/usr/bin/env bash
set -euo pipefail

python3 ~/.agents/harness/checks/check_script_guard.py \
  --cwd ~/.agents \
  --policy ~/.agents/harness/policies/script-guard.json
