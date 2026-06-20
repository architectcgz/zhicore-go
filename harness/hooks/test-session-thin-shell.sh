#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
script="$script_dir/session-thin-shell.sh"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

extract_context() {
  HOOK_OUTPUT="$(cat)" python3 - <<'PY'
import json
import os
import sys

payload = json.loads(os.environ["HOOK_OUTPUT"])
print(payload["hookSpecificOutput"]["additionalContext"])
PY
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "$haystack" != *"$needle"* ]]; then
    printf 'expected output to contain: %s\n' "$needle" >&2
    exit 1
  fi
}

assert_not_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "$haystack" == *"$needle"* ]]; then
    printf 'expected output not to contain: %s\n' "$needle" >&2
    exit 1
  fi
}

project_with_markers="$tmpdir/with-markers"
mkdir -p "$project_with_markers"
cat >"$project_with_markers/AGENTS.md" <<'EOF'
# Project

outside before

<!-- codex-session-thin-shell:start -->
## Quick Routing

| Task | Must Read |
| --- | --- |
| Bug | docs/debug.md |

## Red Flags

- Do not skip project routing.
<!-- codex-session-thin-shell:end -->

outside after
EOF

marker_output="$(printf '{"cwd":"%s"}' "$project_with_markers" | "$script")"
marker_context="$(printf '%s' "$marker_output" | extract_context)"
assert_contains "$marker_context" "Quick Routing"
assert_contains "$marker_context" "Do not skip project routing."
assert_not_contains "$marker_context" "outside before"
assert_not_contains "$marker_context" "outside after"

project_with_headings="$tmpdir/with-headings"
mkdir -p "$project_with_headings"
cat >"$project_with_headings/AGENTS.md" <<'EOF'
# Project

intro text

## Quick Routing

| Task | Must Read |
| --- | --- |
| Feature | docs/feature.md |

## Other Notes

this should stay out

## Auto-Triggers

- New task -> reread AGENTS.md.

## Boundary

not part of the thin shell
EOF

heading_output="$(printf '{"cwd":"%s"}' "$project_with_headings" | "$script")"
heading_context="$(printf '%s' "$heading_output" | extract_context)"
assert_contains "$heading_context" "Feature | docs/feature.md"
assert_contains "$heading_context" "New task -> reread AGENTS.md."
assert_not_contains "$heading_context" "this should stay out"
assert_not_contains "$heading_context" "not part of the thin shell"

empty_project="$tmpdir/empty"
mkdir -p "$empty_project"
empty_output="$(printf '{"cwd":"%s"}' "$empty_project" | "$script")"
if [[ -n "$empty_output" ]]; then
  printf 'expected no output without AGENTS.md, got: %s\n' "$empty_output" >&2
  exit 1
fi

printf 'session-thin-shell tests passed\n'
