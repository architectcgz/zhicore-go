#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF' >&2
Usage:
  bash ~/.agents/harness/install-project-skills.sh <repo-root>

Installs project-local shared skills for:
- Claude via <repo-root>/.claude/skills -> ../.agents/skills
- Codex via ~/.codex/skills/<project>-<skill> symlinks
EOF
}

if [[ $# -ne 1 || "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  [[ $# -eq 1 ]] && [[ "${1:-}" =~ ^(-h|--help)$ ]] && exit 0
  exit 2
fi

repo_root="$(readlink -f "$1")"
if [[ ! -d "$repo_root" ]]; then
  echo "FAIL: repo root does not exist: $repo_root" >&2
  exit 1
fi

project_prefix="$(basename "$repo_root")"
codex_home="${CODEX_HOME:-$HOME/.codex}"
codex_skills_dir="$codex_home/skills"
shared_skills_dir="$repo_root/.agents/skills"
workspace_entrypoint_check="$HOME/workspace/projects/scripts/check-agent-entrypoints.sh"

check_shared_skills() {
  local root="$1"
  local skills_dir="$root/.agents/skills"
  local shared_readme="$skills_dir/README.md"

  if [[ -x "$root/scripts/check-shared-skills.sh" ]]; then
    bash "$root/scripts/check-shared-skills.sh" "$root"
    return
  fi

  echo "[shared-skills] check project: $root"

  if [[ ! -d "$skills_dir" ]]; then
    echo "FAIL: missing shared skills dir $skills_dir" >&2
    exit 1
  fi

  if [[ ! -f "$shared_readme" ]]; then
    echo "FAIL: missing shared skills README $shared_readme" >&2
    exit 1
  fi

  echo "PASS: $shared_readme"

  while IFS= read -r skill_dir; do
    skill_file="$skill_dir/SKILL.md"
    if [[ ! -f "$skill_file" ]]; then
      echo "FAIL: missing skill source $skill_file" >&2
      exit 1
    fi
    echo "PASS: $skill_file"
  done < <(find "$skills_dir" -mindepth 1 -maxdepth 1 -type d | sort)
}

check_agent_entrypoints() {
  local root="$1"
  local agents_file="$root/AGENTS.md"
  local claude_file="$root/CLAUDE.md"
  local resolved=""
  local expected=""

  if [[ -x "$workspace_entrypoint_check" ]]; then
    bash "$workspace_entrypoint_check" "$root"
    return
  fi

  echo "[agent-entrypoints] check project: $root"

  if [[ ! -f "$agents_file" ]]; then
    echo "FAIL: missing $agents_file" >&2
    exit 1
  fi

  if [[ ! -L "$claude_file" ]]; then
    echo "FAIL: $claude_file must be a symlink to AGENTS.md" >&2
    exit 1
  fi

  resolved="$(readlink -f "$claude_file")"
  expected="$(readlink -f "$agents_file")"
  if [[ "$resolved" != "$expected" ]]; then
    echo "FAIL: $claude_file resolves to $resolved, expected $expected" >&2
    exit 1
  fi

  echo "PASS: $claude_file -> AGENTS.md"
}

cd "$repo_root"

check_shared_skills "$repo_root"

mkdir -p "$codex_skills_dir" "$repo_root/.claude"
ln -sfn ../.agents/skills "$repo_root/.claude/skills"

while IFS= read -r source_dir; do
  skill="$(basename "$source_dir")"
  target_link="$codex_skills_dir/${project_prefix}-${skill}"

  if [[ -e "$target_link" && ! -L "$target_link" ]]; then
    echo "FAIL: existing non-symlink blocks install: $target_link" >&2
    exit 1
  fi

  ln -sfn "$source_dir" "$target_link"
  echo "linked: $target_link -> $source_dir"
done < <(find "$shared_skills_dir" -mindepth 1 -maxdepth 1 -type d | sort)

check_agent_entrypoints "$repo_root"
echo "Installed shared skills for Claude and Codex."
