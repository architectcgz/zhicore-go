#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF' >&2
Usage:
  bash ~/.agents/scripts/check-project-skill-wrapper-shape.sh [project-root]

Checks project-local wrapper skills that declare a global source:
- scans <project-root>/.agents/skills/*/SKILL.md
- only targets files containing: 通用主体：`~/.agents/skills/...`
- requires thin-wrapper structure instead of copied full generic skill bodies
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

root="${1:-$(pwd)}"
root="$(readlink -f "$root")"
skills_dir="$root/.agents/skills"

if [[ ! -d "$skills_dir" ]]; then
  echo "PASS: no project .agents/skills directory under $root"
  exit 0
fi

fail=0
checked=0

required_patterns=(
  '^## 主体来源$'
  '^## 在 .+使用方式$'
  '^## 本地补充关注点$'
)

forbidden_patterns=(
  '^## Overview$'
  '^## When To Use$'
  '^## Use When$'
  '^## Do Not Use$'
  '^## Core Guardrails$'
  '^## Workflow$'
  '^## Output Expectations$'
  '^## Output Protocol$'
  '^## Common Mistakes$'
  '^## Source Basis$'
  '^## Default Scope$'
  '^## Role$'
  '^## Decomposition Routing$'
  '^## Implementation Rules$'
  '^## Quick Checklist$'
  '^## Review Archive$'
  '^## Review Evidence Location$'
  '^## Stage Execution Rules$'
  '^## Workflow Overview$'
)

while IFS= read -r skill_file; do
  if ! rg -q '通用主体：`~/.agents/skills/' "$skill_file"; then
    continue
  fi

  checked=$((checked + 1))
  echo "[wrapper-check] $skill_file"

  line_count="$(wc -l < "$skill_file" | tr -d ' ')"
  if (( line_count > 80 )); then
    echo "FAIL: wrapper too long ($line_count lines > 80)" >&2
    fail=1
  else
    echo "PASS: wrapper length $line_count lines"
  fi

  for pattern in "${required_patterns[@]}"; do
    if rg -q "$pattern" "$skill_file"; then
      echo "PASS: required section present: $pattern"
    else
      echo "FAIL: missing required section matching $pattern" >&2
      fail=1
    fi
  done

  for pattern in "${forbidden_patterns[@]}"; do
    if rg -q "$pattern" "$skill_file"; then
      echo "FAIL: copied generic section detected: $pattern" >&2
      fail=1
    fi
  done
done < <(find "$skills_dir" -mindepth 2 -maxdepth 2 -name SKILL.md | sort)

if (( checked == 0 )); then
  echo "PASS: no project wrapper skills declaring ~/.agents global source"
elif (( fail == 0 )); then
  echo "PASS: all $checked project wrapper skills have thin-wrapper shape"
else
  echo "FAIL: wrapper shape check failed" >&2
  exit 1
fi
