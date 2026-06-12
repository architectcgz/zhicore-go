#!/usr/bin/env bash
# Managed by code-workflow package (version: 2026-06-12.1)
set -euo pipefail

usage() {
  cat <<'EOF' >&2
Usage:
  bash scripts/check-epic-dependencies.sh [epic-slug]
  bash scripts/check-epic-dependencies.sh --list

Description:
  Check epic slice dependencies and overall progress.

Options:
  --list    List all epic index files
  --quiet   Only output on errors

Exit codes:
  0 - all dependencies satisfied, no blockers
  1 - dependency violations or missing index
EOF
}

REPO_ROOT="$(git rev-parse --show-toplevel)"
PLAN_DIR="$REPO_ROOT/docs/plan/impl-plan"
SESSION_GATES_DIR="$REPO_ROOT/.harness/session-gates"

list_mode=0
quiet=0
epic_slug=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --list)
      list_mode=1
      shift
      ;;
    --quiet)
      quiet=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --*)
      echo "FAIL: unknown argument: $1" >&2
      usage
      exit 1
      ;;
    *)
      if [[ -n "$epic_slug" ]]; then
        echo "FAIL: epic slug already set to $epic_slug" >&2
        usage
        exit 1
      fi
      epic_slug="$1"
      shift
      ;;
  esac
done

if [[ "$list_mode" -eq 1 ]]; then
  if [[ ! -d "$PLAN_DIR" ]]; then
    echo "FAIL: plan directory not found: $PLAN_DIR" >&2
    exit 1
  fi
  # List both old-style flat EPIC files and new-style subdirs with EPIC.md
  find "$PLAN_DIR" -maxdepth 1 -type f -name "*-EPIC.md" 2>/dev/null | sort
  find "$PLAN_DIR" -maxdepth 2 -type f -name "EPIC.md" 2>/dev/null | sort
  exit 0
fi

if [[ -z "$epic_slug" ]]; then
  usage
  exit 1
fi

# Support both new structure (epic-slug/EPIC.md) and old flat structure (epic-slug-EPIC.md)
if [[ -f "$PLAN_DIR/${epic_slug}/EPIC.md" ]]; then
  epic_index="$PLAN_DIR/${epic_slug}/EPIC.md"
  epic_dir="$PLAN_DIR/${epic_slug}"
elif [[ -f "$PLAN_DIR/${epic_slug}-EPIC.md" ]]; then
  epic_index="$PLAN_DIR/${epic_slug}-EPIC.md"
  epic_dir="$PLAN_DIR"
else
  echo "FAIL: epic index not found: tried $PLAN_DIR/${epic_slug}/EPIC.md and $PLAN_DIR/${epic_slug}-EPIC.md" >&2
  exit 1
fi

# Parse epic index and extract slices
# Format: "- Task Slug: `<slug>`"
# Status line: "- Status: `<status>`"
# Depends On line: "- Depends On: `<deps>`" or "- Depends On: 无"

declare -A slice_status
declare -A slice_depends
declare -a slice_order

current_slice=""
while IFS= read -r line; do
  # Extract task slug
  if [[ "$line" =~ ^-\ Task\ Slug: ]]; then
    current_slice=$(echo "$line" | sed -n 's/^- Task Slug: `\([^`]*\)`.*$/\1/p')
    if [[ -n "$current_slice" ]]; then
      slice_order+=("$current_slice")
    fi
  # Extract status
  elif [[ -n "$current_slice" && "$line" =~ ^-\ Status: ]]; then
    status=$(echo "$line" | sed -n 's/^- Status: `\([^`]*\)`.*$/\1/p')
    if [[ -n "$status" ]]; then
      slice_status["$current_slice"]="$status"
    fi
  # Extract depends on with backticks
  elif [[ -n "$current_slice" && "$line" =~ ^-\ Depends\ On:\ \` ]]; then
    deps=$(echo "$line" | sed -n 's/^- Depends On: `\([^`]*\)`.*$/\1/p')
    if [[ -n "$deps" ]]; then
      slice_depends["$current_slice"]="$deps"
    fi
  # Extract depends on without backticks (无)
  elif [[ -n "$current_slice" && "$line" =~ ^-\ Depends\ On:\ 无 ]]; then
    slice_depends["$current_slice"]="无"
  fi
done < "$epic_index"

if [[ "${#slice_order[@]}" -eq 0 ]]; then
  echo "FAIL: no slices found in epic index: $epic_index" >&2
  exit 1
fi

get_gate_status() {
  local task_slug="$1"
  local gate_file="$SESSION_GATES_DIR/${task_slug}.json"
  if [[ ! -f "$gate_file" ]]; then
    echo "not-started"
    return
  fi
  python3 -c "import json,sys; print(json.load(open(sys.argv[1])).get('status','unknown'))" "$gate_file" 2>/dev/null || echo "unknown"
}

has_errors=0
completed=0
in_progress=0
not_started=0

if [[ "$quiet" -eq 0 ]]; then
  echo "Epic: $epic_slug"
  echo "Index: $epic_index"
  echo ""
  echo "Slices:"
fi

for slice in "${slice_order[@]}"; do
  status="${slice_status[$slice]:-unknown}"
  depends="${slice_depends[$slice]:-无}"
  gate_status="$(get_gate_status "$slice")"

  case "$status" in
    completed) completed=$((completed + 1)) ;;
    in-progress) in_progress=$((in_progress + 1)) ;;
    not-started) not_started=$((not_started + 1)) ;;
  esac

  if [[ "$quiet" -eq 0 ]]; then
    printf "  - %s\n" "$slice"
    printf "    Status: %s (gate: %s)\n" "$status" "$gate_status"
    printf "    Depends On: %s\n" "$depends"
  fi

  # Check dependencies
  if [[ "$depends" != "无" ]]; then
    IFS=',' read -ra dep_array <<< "$depends"
    for dep in "${dep_array[@]}"; do
      dep="$(echo "$dep" | xargs)" # trim
      dep_status="${slice_status[$dep]:-unknown}"
      dep_gate_status="$(get_gate_status "$dep")"

      # If current slice is in-progress or completed, dependencies must be completed
      if [[ "$status" == "in-progress" || "$status" == "completed" ]]; then
        if [[ "$dep_status" != "completed" && "$dep_gate_status" != "ready_to_merge" ]]; then
          echo "  ERROR: $slice depends on $dep, but $dep status is $dep_status (gate: $dep_gate_status)" >&2
          has_errors=1
        fi
      fi

      # If current slice gate is active, dependencies should not be active
      if [[ "$gate_status" == "active" ]]; then
        if [[ "$dep_gate_status" == "active" ]]; then
          echo "  WARNING: $slice gate is active while dependency $dep is also active" >&2
        fi
      fi
    done
  fi

  if [[ "$quiet" -eq 0 ]]; then
    echo ""
  fi
done

total="${#slice_order[@]}"

if [[ "$quiet" -eq 0 ]]; then
  echo "Progress: $completed/$total completed, $in_progress in-progress, $not_started not-started"
  echo ""
fi

if [[ "$has_errors" -eq 1 ]]; then
  echo "FAIL: epic dependency violations detected" >&2
  exit 1
fi

if [[ "$quiet" -eq 0 ]]; then
  echo "PASS: epic dependencies satisfied"
fi

exit 0
