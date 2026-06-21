#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOH' >&2
Usage:
  bash scripts/start-implementation.sh <topic-or-slug> [--title <plan-title>] [--base <git-ref>] [--dry-run]
EOH
}

ROOT="$(git rev-parse --show-toplevel)"
REPO_NAME="$(basename "$ROOT")"
WORKSPACE_ROOT="$(dirname "$ROOT")"
WORKTREE_PARENT="${WORKTREE_PARENT:-$WORKSPACE_ROOT/.worktrees/$REPO_NAME}"
PLAN_TEMPLATE="$ROOT/harness/templates/implementation-plan-skeleton.md"

topic_or_slug=""
plan_title=""
base_ref="HEAD"
dry_run=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --title)
      [[ $# -ge 2 ]] || { echo "FAIL: --title requires a value" >&2; exit 1; }
      plan_title="$2"
      shift 2
      ;;
    --base)
      [[ $# -ge 2 ]] || { echo "FAIL: --base requires a git ref" >&2; exit 1; }
      base_ref="$2"
      shift 2
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --*)
      echo "FAIL: unknown arg: $1" >&2
      usage
      exit 1
      ;;
    *)
      if [[ -n "$topic_or_slug" ]]; then
        echo "FAIL: topic-or-slug already set to '$topic_or_slug'" >&2
        usage
        exit 1
      fi
      topic_or_slug="$1"
      shift
      ;;
  esac
done

if [[ -z "$topic_or_slug" ]]; then
  usage
  exit 1
fi

normalize_slug() {
  local raw="$1"
  local cleaned
  cleaned="$(
    printf '%s' "$raw" \
      | tr '[:upper:]' '[:lower:]' \
      | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//; s/-{2,}/-/g'
  )"
  printf '%s' "$cleaned"
}

topic_is_slug=0
if [[ "$topic_or_slug" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9]+(-[a-z0-9]+)*$ ]]; then
  task_slug="$topic_or_slug"
  topic_is_slug=1
else
  normalized_topic="$(normalize_slug "$topic_or_slug")"
  if [[ -z "$normalized_topic" ]]; then
    echo "FAIL: topic '$topic_or_slug' cannot be normalized into a valid slug" >&2
    exit 1
  fi
  task_slug="$(date +%F)-$normalized_topic"
fi

if [[ ! "$task_slug" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9]+(-[a-z0-9]+)*$ ]]; then
  echo "FAIL: task slug must match YYYY-MM-DD-topic format: $task_slug" >&2
  exit 1
fi

if [[ -z "$plan_title" ]]; then
  if [[ "$topic_is_slug" -eq 1 ]]; then
    plan_title="$task_slug"
  else
    plan_title="$topic_or_slug"
  fi
fi

branch_name="task/$task_slug"
worktree_path="$WORKTREE_PARENT/$task_slug"
plan_file_name="$task_slug-implementation-plan.md"
gate_dir=".harness/session-gates"
gate_file_name="$task_slug.json"
started_at="$(date -u +%FT%TZ)"

created_worktree=0
created_plan=0
created_gate=0
plan_path=""
gate_path=""
initial_plan_sha=""

cleanup() {
  local status=$?
  if [[ "$status" -eq 0 ]]; then
    return 0
  fi

  if [[ "$created_gate" -eq 1 && -n "$gate_path" && -f "$gate_path" ]]; then
    rm -f "$gate_path"
  fi

  if [[ "$created_plan" -eq 1 && -n "$plan_path" && -f "$plan_path" ]]; then
    local current_sha
    current_sha="$(sha256sum "$plan_path" | awk '{print $1}')"
    if [[ "$current_sha" == "$initial_plan_sha" ]]; then
      rm -f "$plan_path"
    fi
  fi

  if [[ "$created_worktree" -eq 1 && -d "$worktree_path" ]]; then
    if [[ -z "$(git -C "$worktree_path" status --porcelain 2>/dev/null || true)" ]]; then
      git -C "$ROOT" worktree remove --force "$worktree_path" >/dev/null 2>&1 || true
    fi
  fi

  return "$status"
}

trap cleanup EXIT

if [[ ! -f "$PLAN_TEMPLATE" ]]; then
  echo "FAIL: missing implementation plan template: $PLAN_TEMPLATE" >&2
  exit 1
fi

if git -C "$ROOT" rev-parse --verify --quiet "$branch_name" >/dev/null; then
  echo "FAIL: branch already exists: $branch_name" >&2
  exit 1
fi

if [[ -e "$worktree_path" ]]; then
  echo "FAIL: worktree path already exists: $worktree_path" >&2
  exit 1
fi

mkdir -p "$WORKTREE_PARENT"

bash "$ROOT/scripts/check-task-intake.sh"

git -C "$ROOT" rev-parse --verify --quiet "$base_ref" >/dev/null || {
  echo "FAIL: base ref does not exist: $base_ref" >&2
  exit 1
}

if [[ "$dry_run" -eq 1 ]]; then
  printf '%s\n' "DRY RUN: implementation workspace would be initialized"
  printf '%s\n' "- task slug: $task_slug"
  printf '%s\n' "- worktree: $worktree_path"
  printf '%s\n' "- branch: $branch_name"
  printf '%s\n' "- plan: docs/plan/impl-plan/$plan_file_name"
  printf '%s\n' "- gate: $gate_dir/$gate_file_name"
  exit 0
fi

git -C "$ROOT" worktree add -b "$branch_name" "$worktree_path" "$base_ref" >/dev/null
created_worktree=1

plan_path="$worktree_path/docs/plan/impl-plan/$plan_file_name"
gate_path="$worktree_path/$gate_dir/$gate_file_name"

mkdir -p "$(dirname "$plan_path")" "$(dirname "$gate_path")"

sed \
  -e "s#__TASK_TITLE__#$(printf '%s' "$plan_title" | sed 's/[&/]/\\&/g')#g" \
  -e "s#__TASK_SLUG__#$(printf '%s' "$task_slug" | sed 's/[&/]/\\&/g')#g" \
  -e "s#__STARTED_AT__#$(printf '%s' "$started_at" | sed 's/[&/]/\\&/g')#g" \
  -e "s#__WORKTREE_PATH__#$(printf '%s' "$worktree_path" | sed 's/[&/]/\\&/g')#g" \
  -e "s#__BRANCH_NAME__#$(printf '%s' "$branch_name" | sed 's/[&/]/\\&/g')#g" \
  "$PLAN_TEMPLATE" > "$plan_path"
created_plan=1
initial_plan_sha="$(sha256sum "$plan_path" | awk '{print $1}')"

cat > "$gate_path" <<EOG
{
  "task_slug": "$task_slug",
  "status": "active",
  "started_at": "$started_at",
  "repo_root": "$ROOT",
  "worktree_path": "$worktree_path",
  "branch": "$branch_name",
  "plan_path": "docs/plan/impl-plan/$plan_file_name"
}
EOG
created_gate=1

printf '%s\n' "PASS: implementation workspace initialized"
printf '%s\n' "- task slug: $task_slug"
printf '%s\n' "- worktree: $worktree_path"
printf '%s\n' "- branch: $branch_name"
printf '%s\n' "- plan: docs/plan/impl-plan/$plan_file_name"
printf '%s\n' "- gate: $gate_dir/$gate_file_name"
printf '\n'
printf '%s\n' "Next steps:"
printf '%s\n' "1. cd $worktree_path"
printf '%s\n' "2. Run the intake analysis gate: relevant superpowers analysis pass first, then grill-with-docs"
printf '%s\n' "3. Complete the plan in Chinese by default via superpowers:writing-plans using that analysis output"
printf '%s\n' "4. Start implementation only after the plan is complete enough for the current slice"
