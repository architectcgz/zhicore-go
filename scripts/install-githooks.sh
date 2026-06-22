#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"

cd "$ROOT_DIR"

git config core.hooksPath .githooks
chmod +x .githooks/commit-msg
chmod +x scripts/check-commit-message.sh

echo "Installed git hooks to .githooks (core.hooksPath=.githooks)"
