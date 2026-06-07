#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF' >&2
Usage:
  bash ~/.agents/harness/check-agent-home.sh

Checks:
- ~/.agents/AGENTS.md exists
- ~/.agents/CLAUDE.md -> AGENTS.md
- ~/.claude/AGENTS.md -> ~/.agents/AGENTS.md
- ~/.claude/CLAUDE.md -> AGENTS.md
- ~/.claude/agents -> ~/.agents/claude-agents
- ~/.claude/skills -> ~/.agents/skills
- ~/.codex/AGENTS.md -> ~/.agents/AGENTS.md
- ~/.codex/CLAUDE.md -> AGENTS.md
- ~/.codex/agents -> ~/.agents/codex-agents
- ~/.codex/skills exists, and if it is a symlink it resolves to ~/.agents/skills
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

shared_root="${AGENTS_HOME:-$HOME/.agents}"
claude_root="${CLAUDE_HOME:-$HOME/.claude}"
codex_root="${CODEX_HOME:-$HOME/.codex}"

assert_exists() {
  local path="$1"
  local label="$2"
  if [[ ! -e "$path" ]]; then
    echo "FAIL: missing $label: $path" >&2
    exit 1
  fi
  echo "PASS: $label"
}

assert_symlink_resolves() {
  local path="$1"
  local target="$2"
  local label="$3"
  if [[ ! -L "$path" ]]; then
    echo "FAIL: $label must be a symlink: $path" >&2
    exit 1
  fi

  if [[ "$(readlink -f "$path")" != "$(readlink -f "$target")" ]]; then
    echo "FAIL: $label resolves to $(readlink -f "$path"), expected $(readlink -f "$target")" >&2
    exit 1
  fi

  echo "PASS: $label"
}

echo "[agent-home] shared root"
assert_exists "$shared_root/AGENTS.md" "~/.agents/AGENTS.md"
assert_exists "$shared_root/skills" "~/.agents/skills"
assert_exists "$shared_root/claude-agents" "~/.agents/claude-agents"
assert_exists "$shared_root/codex-agents" "~/.agents/codex-agents"
assert_symlink_resolves "$shared_root/CLAUDE.md" "$shared_root/AGENTS.md" "~/.agents/CLAUDE.md -> AGENTS.md"

echo "[agent-home] claude"
assert_symlink_resolves "$claude_root/AGENTS.md" "$shared_root/AGENTS.md" "~/.claude/AGENTS.md -> ~/.agents/AGENTS.md"
assert_symlink_resolves "$claude_root/CLAUDE.md" "$claude_root/AGENTS.md" "~/.claude/CLAUDE.md -> AGENTS.md"
assert_symlink_resolves "$claude_root/agents" "$shared_root/claude-agents" "~/.claude/agents -> ~/.agents/claude-agents"
assert_symlink_resolves "$claude_root/skills" "$shared_root/skills" "~/.claude/skills -> ~/.agents/skills"

echo "[agent-home] codex"
assert_symlink_resolves "$codex_root/AGENTS.md" "$shared_root/AGENTS.md" "~/.codex/AGENTS.md -> ~/.agents/AGENTS.md"
assert_symlink_resolves "$codex_root/CLAUDE.md" "$codex_root/AGENTS.md" "~/.codex/CLAUDE.md -> AGENTS.md"
assert_symlink_resolves "$codex_root/agents" "$shared_root/codex-agents" "~/.codex/agents -> ~/.agents/codex-agents"

if [[ ! -e "$codex_root/skills" ]]; then
  echo "FAIL: missing ~/.codex/skills" >&2
  exit 1
fi

if [[ -L "$codex_root/skills" ]]; then
  assert_symlink_resolves "$codex_root/skills" "$shared_root/skills" "~/.codex/skills -> ~/.agents/skills"
elif [[ -d "$codex_root/skills" ]]; then
  echo "PASS: ~/.codex/skills exists as a directory entrypoint"
else
  echo "FAIL: ~/.codex/skills must be a directory or symlink" >&2
  exit 1
fi

echo "PASS: agent home wiring is aligned"
