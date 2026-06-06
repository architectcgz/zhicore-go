#!/usr/bin/env bash
set -euo pipefail

usage() {
  printf 'Usage: %s TARGET_DIR [--ref GIT_REF]\n' "$(basename "$0")" >&2
  printf 'Creates a new HFUT thesis project from https://github.com/shinyypig/hfut-thesis.\n' >&2
}

target_dir=""
git_ref=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    --ref)
      [ "$#" -ge 2 ] || { usage; exit 2; }
      git_ref="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    -*)
      usage
      exit 2
      ;;
    *)
      if [ -n "$target_dir" ]; then
        usage
        exit 2
      fi
      target_dir="$1"
      shift
      ;;
  esac
done

[ -n "$target_dir" ] || { usage; exit 2; }

if [ -e "$target_dir" ]; then
  printf 'Error: target already exists: %s\n' "$target_dir" >&2
  exit 1
fi

command -v git >/dev/null 2>&1 || {
  printf 'Error: git is required.\n' >&2
  exit 1
}

repo_url="https://github.com/shinyypig/hfut-thesis.git"

if [ -n "$git_ref" ]; then
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' EXIT
  git clone --depth 1 "$repo_url" "$tmp_dir/hfut-thesis"
  git -C "$tmp_dir/hfut-thesis" fetch --depth 1 origin "$git_ref"
  git -C "$tmp_dir/hfut-thesis" checkout --detach FETCH_HEAD
  cp -R "$tmp_dir/hfut-thesis" "$target_dir"
else
  git clone --depth 1 "$repo_url" "$target_dir"
fi

printf 'Created HFUT thesis project: %s\n' "$target_dir"
printf 'Next: edit tex/info.tex and build with latexmk -pdfxe.\n'
