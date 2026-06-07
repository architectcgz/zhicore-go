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
