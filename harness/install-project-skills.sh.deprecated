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
shared_skills_check="$HOME/.agents/harness/check-project-shared-skills.sh"
project_entrypoints_check="$HOME/.agents/harness/check-project-agent-entrypoints.sh"

cd "$repo_root"

bash "$shared_skills_check" "$repo_root"

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

if [[ -x "$workspace_entrypoint_check" ]]; then
  bash "$workspace_entrypoint_check" "$repo_root"
else
  bash "$project_entrypoints_check" "$repo_root"
fi
echo "Installed shared skills for Claude and Codex."
