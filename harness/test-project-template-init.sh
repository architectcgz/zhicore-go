#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
script_path="$script_dir/project-template-init.sh"

tmp_root="$(mktemp -d /tmp/project-template-init-test.XXXXXX)"
cleanup() {
  rm -rf "$tmp_root"
}
trap cleanup EXIT

dest_missing="$tmp_root/missing-identity"
if bash "$script_path" frontend-vue \
  --dest "$dest_missing" \
  --app-name demo-app >"$tmp_root/missing.out" 2>"$tmp_root/missing.err"; then
  echo "FAIL: expected new project init to require git identity" >&2
  exit 1
fi

if ! grep -q "git user" "$tmp_root/missing.err"; then
  echo "FAIL: missing-identity failure did not explain git identity requirement" >&2
  cat "$tmp_root/missing.err" >&2
  exit 1
fi

dest_success="$tmp_root/with-identity"
bash "$script_path" frontend-vue \
  --dest "$dest_success" \
  --app-name demo-app \
  --git-user-name "Example User" \
  --git-user-email "example@example.invalid" >"$tmp_root/success.out" 2>"$tmp_root/success.err"

git_root="$(git -C "$dest_success" rev-parse --show-toplevel)"
if [[ "$git_root" != "$dest_success" ]]; then
  echo "FAIL: expected template init to create a git repository at $dest_success" >&2
  exit 1
fi

actual_name="$(git -C "$dest_success" config --local user.name)"
actual_email="$(git -C "$dest_success" config --local user.email)"
if [[ "$actual_name" != "Example User" ]]; then
  echo "FAIL: git user.name = $actual_name" >&2
  exit 1
fi
if [[ "$actual_email" != "example@example.invalid" ]]; then
  echo "FAIL: git user.email = $actual_email" >&2
  exit 1
fi

echo "PASS: project template init enforces git identity and initializes local git config"
