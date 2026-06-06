#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <implementation-plan.md>" >&2
  exit 2
fi

plan_path="$1"

if [[ ! -f "$plan_path" ]]; then
  echo "FAIL: implementation plan not found: $plan_path" >&2
  exit 2
fi

unchecked_items="$(grep -nE '^[[:space:]]*-[[:space:]]+\[ \]' "$plan_path" || true)"
checkbox_count="$(grep -cE '^[[:space:]]*-[[:space:]]+\[[ xX]\]' "$plan_path" || true)"

if [[ "$checkbox_count" -eq 0 ]]; then
  echo "FAIL: no implementation-plan checklist items found in: $plan_path" >&2
  exit 1
fi

if [[ -n "$unchecked_items" ]]; then
  echo "FAIL: implementation plan has unchecked checklist items:" >&2
  echo "$unchecked_items" >&2
  exit 1
fi

echo "PASS: all $checkbox_count implementation-plan checklist items are marked complete."
