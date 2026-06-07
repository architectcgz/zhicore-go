#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF' >&2
Usage:
  bash ~/.agents/harness/bootstrap-agent-home.sh

Bootstraps the local agent home wiring:
- ~/.agents/CLAUDE.md -> AGENTS.md
- ~/.claude/AGENTS.md -> ~/.agents/AGENTS.md
- ~/.claude/CLAUDE.md -> AGENTS.md
- ~/.claude/agents -> ~/.agents/claude-agents
- ~/.claude/skills -> ~/.agents/skills
- ~/.codex/AGENTS.md -> ~/.agents/AGENTS.md
- ~/.codex/CLAUDE.md -> AGENTS.md
- ~/.codex/agents -> ~/.agents/codex-agents
- ~/.codex/skills -> ~/.agents/skills when ~/.codex/skills is missing

The script is idempotent. It does not overwrite conflicting non-symlink files or directories.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

shared_root="${AGENTS_HOME:-$HOME/.agents}"
claude_root="${CLAUDE_HOME:-$HOME/.claude}"
codex_root="${CODEX_HOME:-$HOME/.codex}"
checker="$shared_root/harness/check-agent-home.sh"

ensure_dir() {
  mkdir -p "$@"
}

ensure_link() {
  local path="$1"
  local target="$2"
  local label="$3"
  local expected=""

  if [[ "$target" = /* ]]; then
    expected="$(readlink -f "$target")"
  else
    expected="$(readlink -f "$(dirname "$path")/$target")"
  fi

  if [[ -L "$path" ]]; then
    if [[ "$(readlink -f "$path")" == "$expected" ]]; then
      echo "ok: $label"
      return 0
    fi

    echo "FAIL: conflicting symlink for $label: $path" >&2
    echo "  current:  $(readlink -f "$path")" >&2
    echo "  expected: $expected" >&2
    exit 1
  fi

  if [[ -e "$path" ]]; then
    echo "FAIL: existing non-symlink blocks $label: $path" >&2
    exit 1
  fi

  ln -s "$target" "$path"
  echo "linked: $label"
}

ensure_optional_codex_skills() {
  local path="$codex_root/skills"
  local target="$shared_root/skills"

  if [[ -L "$path" ]]; then
    if [[ "$(readlink -f "$path")" == "$(readlink -f "$target")" ]]; then
      echo "ok: ~/.codex/skills -> ~/.agents/skills"
      return 0
    fi

    echo "FAIL: conflicting symlink for ~/.codex/skills: $path" >&2
    echo "  current:  $(readlink -f "$path")" >&2
    echo "  expected: $(readlink -f "$target")" >&2
    exit 1
  fi

  if [[ -d "$path" ]]; then
    echo "keep: ~/.codex/skills already exists as a directory entrypoint"
    return 0
  fi

  if [[ -e "$path" ]]; then
    echo "FAIL: existing non-directory blocks ~/.codex/skills: $path" >&2
    exit 1
  fi

  ln -s "$target" "$path"
  echo "linked: ~/.codex/skills -> ~/.agents/skills"
}

if [[ ! -d "$shared_root" ]]; then
  echo "FAIL: shared root does not exist: $shared_root" >&2
  exit 1
fi

if [[ ! -f "$shared_root/AGENTS.md" ]]; then
  echo "FAIL: missing $shared_root/AGENTS.md" >&2
  exit 1
fi

ensure_dir "$claude_root" "$codex_root"

ensure_link "$shared_root/CLAUDE.md" "AGENTS.md" "~/.agents/CLAUDE.md -> AGENTS.md"
ensure_link "$claude_root/AGENTS.md" "$shared_root/AGENTS.md" "~/.claude/AGENTS.md -> ~/.agents/AGENTS.md"
ensure_link "$claude_root/CLAUDE.md" "AGENTS.md" "~/.claude/CLAUDE.md -> AGENTS.md"
ensure_link "$claude_root/agents" "$shared_root/claude-agents" "~/.claude/agents -> ~/.agents/claude-agents"
ensure_link "$claude_root/skills" "$shared_root/skills" "~/.claude/skills -> ~/.agents/skills"
ensure_link "$codex_root/AGENTS.md" "$shared_root/AGENTS.md" "~/.codex/AGENTS.md -> ~/.agents/AGENTS.md"
ensure_link "$codex_root/CLAUDE.md" "AGENTS.md" "~/.codex/CLAUDE.md -> AGENTS.md"
ensure_link "$codex_root/agents" "$shared_root/codex-agents" "~/.codex/agents -> ~/.agents/codex-agents"
ensure_optional_codex_skills

bash "$checker"
