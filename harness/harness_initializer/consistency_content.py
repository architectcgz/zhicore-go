#!/usr/bin/env python3
"""Consistency script templates for harness initializer."""

from __future__ import annotations


def ctf_current_check_script() -> str:
    return r"""#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$script_dir/.."

fail=0

red() { printf '\033[31m%s\033[0m' "$1"; }
green() { printf '\033[32m%s\033[0m' "$1"; }

check_file() {
  if [[ -f "$1" ]]; then
    echo "  $(green PASS) — $1"
  else
    echo "  $(red FAIL) — missing $1"
    fail=1
  fi
}

check_dir() {
  if [[ -d "$1" ]]; then
    echo "  $(green PASS) — $1"
  else
    echo "  $(red FAIL) — missing $1"
    fail=1
  fi
}

check_contains() {
  local file="$1" pattern="$2" label="$3"
  if [[ ! -f "$file" ]]; then
    echo "  $(red FAIL) — $label: missing $file"
    fail=1
  elif grep -qE "$pattern" "$file"; then
    echo "  $(green PASS) — $label"
  else
    echo "  $(red FAIL) — $label"
    fail=1
  fi
}

echo "[C1] current-task and durable harness directories exist"
check_dir ".harness"
check_dir ".harness/reuse-decisions"
check_file ".harness/reuse-decisions/.gitkeep"
check_dir "harness"
check_dir "harness/policies"
check_dir "harness/templates"
check_dir "harness/prompts"
check_dir "harness/checks"
check_dir "feedback"

echo "[C2] local private reuse index is wired"
if grep -qx '/.harness/reuse-index/' ".gitignore"; then
  echo "  $(green PASS) — .gitignore reserves /.harness/reuse-index/"
else
  echo "  $(red FAIL) — .gitignore must ignore /.harness/reuse-index/"
  fail=1
fi
if [[ -d ".harness/reuse-index" ]]; then
  echo "  $(green PASS) — .harness/reuse-index exists"
else
  echo "  $(green PASS) — .harness/reuse-index is optional and currently absent"
fi

echo "[C3] project harness assets exist"
check_file "harness/policies/reuse-first.yaml"
check_file "harness/policies/project-patterns.yaml"
check_file "harness/templates/reuse-decision.md"
check_file "harness/prompts/AGENTS.md"
check_file "harness/prompts/harness-router.md"
check_file "harness/checks/common.py"
check_file "feedback/AGENTS.md"
check_file "docs/documentation-rules.md"
check_file "docs/README.md"
check_file "docs/improvements/README.md"
check_file "scripts/check-open-todos.sh"
check_file "scripts/check-todo-governance.sh"
check_file "scripts/check-skill-sync-reminder.sh"
for dir in requirements contracts spec design todo architecture plan operations reviews reports improvements refs; do
  check_dir "docs/$dir"
done
for dir in not-impl implemented agent-recorded rejected archived; do
  check_dir "docs/improvements/$dir"
done

echo "[C4] root navigation references current harness shape"
check_contains "AGENTS.md" '\.harness/' "AGENTS references current-task harness"
check_contains "AGENTS.md" '\.harness/reuse-index/' "AGENTS references local private reuse index"
check_contains "AGENTS.md" 'harness/policies/' "AGENTS references harness policies"
check_contains "AGENTS.md" 'harness/prompts/' "AGENTS references harness prompts"
check_contains "AGENTS.md" 'harness/checks/' "AGENTS references harness checks"
check_contains "AGENTS.md" 'feedback/' "AGENTS references feedback"
check_contains "AGENTS.md" 'docs/documentation-rules\.md' "AGENTS references documentation rules"
check_contains "AGENTS.md" 'docs/README\.md' "AGENTS references documentation index"
check_contains "AGENTS.md" 'scripts/check-open-todos\.sh' "AGENTS references todo reminder"
check_contains "docs/documentation-rules.md" 'Pre-Edit Reading Protocol' "documentation rules define pre-edit reading"
check_contains "docs/documentation-rules.md" 'New Path Registration' "documentation rules define new path registration"
check_contains "docs/documentation-rules.md" 'No Circular References' "documentation rules forbid circular references"

echo "[C4a] project agent entrypoints stay aligned"
check_file "scripts/check-agent-entrypoints.sh"
if [[ -x "scripts/check-agent-entrypoints.sh" ]]; then
  bash scripts/check-agent-entrypoints.sh
else
  echo "  $(red FAIL) — scripts/check-agent-entrypoints.sh is not executable"
  fail=1
fi

echo "[C5] hooks and commit message guard are wired"
check_file "scripts/check-commit-message.sh"
check_file "scripts/check-architecture.sh"
check_file "scripts/check-test-workflow.sh"
check_file "scripts/check-script-guard.sh"
check_file "harness/policies/script-guard.json"
if [[ -f ".githooks/pre-commit" ]]; then
  check_contains ".githooks/pre-commit" 'scripts/check-consistency\.sh' "pre-commit runs scripts/check-consistency.sh"
  check_contains ".githooks/pre-commit" 'scripts/check-skill-sync-reminder\.sh --staged' "pre-commit runs scripts/check-skill-sync-reminder.sh"
else
  echo "  $(red FAIL) — missing .githooks/pre-commit"
  fail=1
fi
if [[ -f ".githooks/commit-msg" ]]; then
  check_contains ".githooks/commit-msg" 'scripts/check-commit-message\.sh' "commit-msg runs scripts/check-commit-message.sh"
else
  echo "  $(red FAIL) — missing .githooks/commit-msg"
  fail=1
fi

echo "[C6] architecture guard is surfaced to the operator"
if [[ -x "scripts/check-architecture.sh" ]]; then
  bash scripts/check-architecture.sh
else
  echo "  $(red FAIL) — scripts/check-architecture.sh is not executable"
  fail=1
fi

echo "[C7] test workflow guard is surfaced to the operator"
if [[ -x "scripts/check-test-workflow.sh" ]]; then
  bash scripts/check-test-workflow.sh
else
  echo "  $(red FAIL) — scripts/check-test-workflow.sh is not executable"
  fail=1
fi

echo "[C8] open todos are surfaced to the operator"
if [[ -x "scripts/check-open-todos.sh" ]]; then
  bash scripts/check-open-todos.sh --quiet-if-empty
else
  echo "  $(red FAIL) — scripts/check-open-todos.sh is not executable"
  fail=1
fi

echo "[C9] script guard stays consistent"
if [[ -x "scripts/check-script-guard.sh" ]]; then
  bash scripts/check-script-guard.sh
else
  echo "  $(red FAIL) — scripts/check-script-guard.sh is not executable"
  fail=1
fi

echo "[C10] todo governance stays consistent"
if [[ -x "scripts/check-todo-governance.sh" ]]; then
  bash scripts/check-todo-governance.sh
else
  echo "  $(red FAIL) — scripts/check-todo-governance.sh is not executable"
  fail=1
fi

if [[ "$fail" -eq 0 ]]; then
  echo "$(green '✓ all harness consistency checks passed')"
else
  echo "$(red '✗ harness consistency checks failed')"
fi

exit "$fail"
"""


def check_script() -> str:
    return r"""#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$script_dir/.."

fail=0

red() { printf '\033[31m%s\033[0m' "$1"; }
green() { printf '\033[32m%s\033[0m' "$1"; }

check_file() {
  if [[ -f "$1" ]]; then
    echo "  $(green PASS) — $1"
  else
    echo "  $(red FAIL) — missing $1"
    fail=1
  fi
}

check_dir() {
  if [[ -d "$1" ]]; then
    echo "  $(green PASS) — $1"
  else
    echo "  $(red FAIL) — missing $1"
    fail=1
  fi
}

check_contains() {
  local file="$1" pattern="$2" label="$3"
  if [[ ! -f "$file" ]]; then
    echo "  $(red FAIL) — $label: missing $file"
    fail=1
  elif grep -qE "$pattern" "$file"; then
    echo "  $(green PASS) — $label"
  else
    echo "  $(red FAIL) — $label"
    fail=1
  fi
}

echo "[C1] strict harness directories exist"
for dir in concepts thinking practice feedback works prompts references; do
  check_dir "$dir"
  check_file "$dir/AGENTS.md"
done

echo "[C2] root navigation references strict harness"
check_contains "AGENTS.md" 'concepts/' "AGENTS references concepts"
check_contains "AGENTS.md" 'thinking/' "AGENTS references thinking"
check_contains "AGENTS.md" 'practice/' "AGENTS references practice"
check_contains "AGENTS.md" 'feedback/' "AGENTS references feedback"
check_contains "AGENTS.md" 'works/' "AGENTS references works"
check_contains "AGENTS.md" 'prompts/' "AGENTS references prompts"
check_contains "AGENTS.md" 'references/' "AGENTS references references"

echo "[C2a] project agent entrypoints stay aligned"
check_file "scripts/check-agent-entrypoints.sh"
if [[ -x "scripts/check-agent-entrypoints.sh" ]]; then
  bash scripts/check-agent-entrypoints.sh
else
  echo "  $(red FAIL) — scripts/check-agent-entrypoints.sh is not executable"
  fail=1
fi

echo "[C3] articles.md numbering is contiguous 1..N"
nums=$(grep -nE '^### [0-9]+\.' references/articles.md | sed -E 's/^[0-9]+:### ([0-9]+)\..*/\1/' || true)
count=$(echo "$nums" | sed '/^$/d' | wc -l | tr -d ' ')
if [[ "$count" -eq 0 ]]; then
  echo "  $(red FAIL) — references/articles.md has no numbered entries"
  fail=1
else
  sorted=$(echo "$nums" | sort -n)
  expected=$(seq 1 "$count")
  if [[ "$sorted" = "$expected" ]]; then
    echo "  $(green PASS) — $count contiguous entries"
  else
    echo "  $(red FAIL) — article numbering is not contiguous"
    fail=1
  fi
fi

echo "[C4] article count claim matches numbered entries"
claim=$(grep -oE '权威计数：[0-9]+ 篇' references/articles.md | head -1 | grep -oE '[0-9]+' || true)
if [[ -z "$claim" || "$claim" != "$count" ]]; then
  echo "  $(red FAIL) — references/articles.md claims ${claim:-none}, actual $count"
  fail=1
else
  echo "  $(green PASS) — count claim $claim"
fi

echo "[C5] hooks and commit message guard are wired"
check_file "scripts/check-commit-message.sh"
check_file "scripts/check-architecture.sh"
check_file "scripts/check-test-workflow.sh"
check_file "scripts/check-script-guard.sh"
check_file "harness/policies/script-guard.json"
if [[ -f ".githooks/pre-commit" ]]; then
  check_contains ".githooks/pre-commit" 'scripts/check-consistency\.sh' "pre-commit runs scripts/check-consistency.sh"
  check_contains ".githooks/pre-commit" 'scripts/check-skill-sync-reminder\.sh --staged' "pre-commit runs scripts/check-skill-sync-reminder.sh"
else
  echo "  $(red FAIL) — missing .githooks/pre-commit"
  fail=1
fi
if [[ -f ".githooks/commit-msg" ]]; then
  check_contains ".githooks/commit-msg" 'scripts/check-commit-message\.sh' "commit-msg runs scripts/check-commit-message.sh"
else
  echo "  $(red FAIL) — missing .githooks/commit-msg"
  fail=1
fi

echo "[C6] documentation architecture exists"
check_file "docs/documentation-rules.md"
check_file "docs/README.md"
check_file "scripts/check-open-todos.sh"
check_file "scripts/check-todo-governance.sh"
check_file "scripts/check-skill-sync-reminder.sh"
check_contains "docs/documentation-rules.md" 'No Circular References' "documentation rules forbid circular references"
check_contains "AGENTS.md" 'scripts/check-open-todos\.sh' "AGENTS references todo reminder"

echo "[C7] architecture guard is surfaced to the operator"
if [[ -x "scripts/check-architecture.sh" ]]; then
  bash scripts/check-architecture.sh
else
  echo "  $(red FAIL) — scripts/check-architecture.sh is not executable"
  fail=1
fi

echo "[C8] test workflow guard is surfaced to the operator"
if [[ -x "scripts/check-test-workflow.sh" ]]; then
  bash scripts/check-test-workflow.sh
else
  echo "  $(red FAIL) — scripts/check-test-workflow.sh is not executable"
  fail=1
fi

echo "[C9] open todos are surfaced to the operator"
if [[ -x "scripts/check-open-todos.sh" ]]; then
  bash scripts/check-open-todos.sh --quiet-if-empty
else
  echo "  $(red FAIL) — scripts/check-open-todos.sh is not executable"
  fail=1
fi

echo "[C10] script guard stays consistent"
if [[ -x "scripts/check-script-guard.sh" ]]; then
  bash scripts/check-script-guard.sh
else
  echo "  $(red FAIL) — scripts/check-script-guard.sh is not executable"
  fail=1
fi

echo "[C11] todo governance stays consistent"
if [[ -x "scripts/check-todo-governance.sh" ]]; then
  bash scripts/check-todo-governance.sh
else
  echo "  $(red FAIL) — scripts/check-todo-governance.sh is not executable"
  fail=1
fi

if [[ "$fail" -eq 0 ]]; then
  echo "$(green '✓ all harness consistency checks passed')"
else
  echo "$(red '✗ harness consistency checks failed')"
fi

exit "$fail"
"""
