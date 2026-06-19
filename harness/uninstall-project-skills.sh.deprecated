#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF' >&2
Usage:
  bash ~/.agents/harness/uninstall-project-skills.sh <repo-root>

Removes Codex skill symlinks created for a project's .agents/skills sources.
The repository-owned .claude/skills bridge is kept in place.
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

if [[ ! -d "$shared_skills_dir" ]]; then
  echo "FAIL: missing shared skills dir: $shared_skills_dir" >&2
  exit 1
fi

while IFS= read -r source_dir; do
  skill="$(basename "$source_dir")"
  target_link="$codex_skills_dir/${project_prefix}-${skill}"

  if [[ -L "$target_link" && "$(readlink -f "$target_link")" == "$(readlink -f "$source_dir")" ]]; then
    rm "$target_link"
    echo "removed: $target_link"
  fi
done < <(find "$shared_skills_dir" -mindepth 1 -maxdepth 1 -type d | sort)

echo "Uninstalled shared Codex skill links for this project."
echo "Kept .claude/skills in place because it is part of the repository contract."
