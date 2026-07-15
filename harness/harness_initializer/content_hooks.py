#!/usr/bin/env python3
"""Harness initializer content hooks templates."""

from __future__ import annotations
import json


def protect_core_files_hook_script() -> str:
    """Generate PreToolUse hook script for protecting core rule files."""
    return r"""#!/usr/bin/env bash
#
# PreToolUse hook: 保护核心规则文件，防止意外修改
#
set -euo pipefail

# 受保护的文件列表（相对于项目根）
PROTECTED_PATHS=(
  "AGENTS.md"
  ".arccgz-harness/harness/policies/reuse-first.yaml"
  ".arccgz-harness/harness/policies/commit-message.json"
  ".arccgz-harness/harness/policies/project-patterns.yaml"
  ".arccgz-harness/harness/policies/script-layer-manifest.json"
  ".arccgz-harness/docs/文档规范.md"
  ".agents/skills/*/SKILL.md"
)

# 允许修改的例外情况（需要在提交信息或任务描述中明确说明）
ALLOW_PATTERNS=(
  "chore(harness): update policy"
  "docs(harness): fix policy typo"
  "feat(harness): extend policy"
)

# 从 TOOL_ARGS_JSON 中提取 file_path
extract_file_path() {
  local json="$1"
  echo "$json" | grep -oP '"file_path"\s*:\s*"\K[^"]+' || echo ""
}

# 检查路径是否匹配保护列表
is_protected() {
  local path="$1"
  local protected_pattern

  for protected_pattern in "${PROTECTED_PATHS[@]}"; do
    if [[ "$path" == $protected_pattern ]]; then
      return 0
    fi
    local rel_path="${path#$PWD/}"
    if [[ "$rel_path" == $protected_pattern ]]; then
      return 0
    fi
  done

  return 1
}

# 检查是否在允许的例外模式中
is_allowed_exception() {
  local reason="$1"
  local pattern

  for pattern in "${ALLOW_PATTERNS[@]}"; do
    if [[ "$reason" =~ $pattern ]]; then
      return 0
    fi
  done

  return 1
}

main() {
  local tool_args="${TOOL_ARGS_JSON:-}"

  if [[ -z "$tool_args" ]]; then
    if [[ $# -gt 0 ]]; then
      tool_args="$1"
    else
      exit 0
    fi
  fi

  local file_path
  file_path=$(extract_file_path "$tool_args")

  if [[ -z "$file_path" ]]; then
    exit 0
  fi

  if is_protected "$file_path"; then
    local task_context="${TASK_CONTEXT:-}"
    local commit_message="${COMMIT_MESSAGE:-}"

    if is_allowed_exception "$task_context" || is_allowed_exception "$commit_message"; then
      echo "[PreToolUse] Allowing protected file modification: $file_path (exception matched)" >&2
      exit 0
    fi

    cat >&2 <<EOF_MSG

[PreToolUse Hook] BLOCKED: Protected file modification attempt
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

File: $file_path

This file is protected by PreToolUse hook. Direct modification is not allowed.

Why this matters:
- Core policy files define project rules and should not be casually modified
- Accidental edits to these files can break the entire harness
- Changes to these files need explicit review and approval

If you really need to modify this file:
1. Confirm with the user that this change is intentional
2. Explain why the policy needs to change
3. Document the change in .arccgz-harness/feedback/ or commit message
4. Use an allowed commit pattern:
   - chore(harness): update policy
   - docs(harness): fix policy typo
   - feat(harness): extend policy

Protected files list:
$(printf '  - %s\n' "${PROTECTED_PATHS[@]}")

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EOF_MSG

    exit 1
  fi

  exit 0
}

main "$@"
"""


def codex_hooks_json() -> str:
    """Generate Codex hooks.json configuration."""
    config = {
        "hooks": {
            "PreToolUse": [
                {
                    "matcher": "Edit|Write",
                    "hooks": [
                        {
                            "type": "command",
                            "command": "bash .arccgz-harness/harness/hooks/protect-core-files.sh",
                            "statusMessage": "Checking protected files"
                        }
                    ]
                }
            ]
        }
    }
    return json.dumps(config, ensure_ascii=False, indent=2)


def claude_settings_hooks() -> str:
    """Generate Claude Code settings.local.json hooks configuration."""
    config = {
        "hooks": {
            "preToolUse": [
                {
                    "tools": ["edit", "write"],
                    "script": "bash .arccgz-harness/harness/hooks/protect-core-files.sh"
                }
            ]
        }
    }
    return json.dumps(config, ensure_ascii=False, indent=2)


def post_tooluse_aar_hook_script() -> str:
    """Generate PostToolUse hook script for After Action Review."""
    return r"""#!/usr/bin/env bash
#
# PostToolUse AAR Hook: 任务完成后自动触发 After Action Review
#
set -euo pipefail

# AAR 检查清单
AAR_CHECKLIST=(
  "是否踩到新坑？需要记录到 Known Gotchas 或 .arccgz-harness/feedback/"
  "规则文件是否被意外修改？检查 AGENTS.md 和 .arccgz-harness/harness/policies/"
  "测试是否都通过？运行相关测试命令"
  "是否有新的架构决策需要记录到 .arccgz-harness/docs/architecture/？"
  "是否有可复用的模式需要提取到 skill 或 policy？"
  "是否有需要更新的文档？"
)

# 任务完成信号（可配置）
COMPLETION_SIGNALS=(
  "completed"
  "finished"
  "done"
  "merged"
  "task complete"
)

# 从工具参数中检测任务完成信号
detect_completion_signal() {
  local tool_output="$1"
  local signal

  for signal in "${COMPLETION_SIGNALS[@]}"; do
    if echo "$tool_output" | grep -qi "$signal"; then
      return 0
    fi
  done

  return 1
}

# 检查当前是否有激活的 task gate
check_active_task_gate() {
  if [[ -x .arccgz-harness/scripts/check-startup-gate.sh ]]; then
    local active_slug
    active_slug="$(bash .arccgz-harness/scripts/check-startup-gate.sh --print-active-slug 2>/dev/null || true)"
    echo "$active_slug"
  fi
}

# 生成 AAR 模板
generate_aar_template() {
  local task_slug="$1"
  local timestamp
  timestamp=$(date +%Y%m%d-%H%M%S)

  cat <<EOF_AAR
# After Action Review — $task_slug

Date: $(date +%Y-%m-%d)
Task: $task_slug

## 检查清单

$(for item in "${AAR_CHECKLIST[@]}"; do
  echo "- [ ] $item"
done)

## 新发现的坑（Known Gotchas）

<!-- 如果踩到新坑，记录在这里，格式：
### 坑标题
- 现象：...
- 原因：...
- 解决方案：...
- 预防措施：...
-->

## 规则变化

<!-- 是否有规则文件被修改？如果是，说明原因和影响 -->

- [ ] AGENTS.md
- [ ] .arccgz-harness/harness/policies/
- [ ] .arccgz-harness/docs/文档规范.md
- [ ] .agents/skills/*/SKILL.md

## 测试结果

<!-- 记录测试执行情况 -->

```bash
# 执行的测试命令
[填写]

# 测试结果
[填写]
```

## 架构决策

<!-- 是否有新的架构决策？需要记录到 .arccgz-harness/docs/architecture/ 吗？ -->

## 可复用模式

<!-- 是否有可以提取到 skill 或 policy 的模式？ -->

## 文档更新

<!-- 哪些文档需要更新？ -->

## 经验教训

<!-- 这次任务的主要收获和教训 -->

### 做得好的

### 需要改进的

### 下次注意

## 后续行动

<!-- 有哪些后续任务或跟进事项？ -->

- [ ]
EOF_AAR
}

# 保存 AAR 到 .arccgz-harness/feedback/
save_aar() {
  local task_slug="$1"
  local aar_content="$2"
  local timestamp
  timestamp=$(date +%Y%m%d-%H%M%S)

  local feedback_dir=".arccgz-harness/feedback/aar"
  mkdir -p "$feedback_dir"

  local aar_file="$feedback_dir/${timestamp}-${task_slug}.md"
  echo "$aar_content" > "$aar_file"

  echo "$aar_file"
}

# 主函数
main() {
  local tool_name="${TOOL_NAME:-}"
  local tool_output="${TOOL_OUTPUT:-}"

  # 从命令行参数读取（兼容不同调用方式）
  if [[ -z "$tool_output" && $# -gt 0 ]]; then
    tool_output="$1"
  fi

  # 如果没有输出，不触发 AAR
  if [[ -z "$tool_output" ]]; then
    exit 0
  fi

  # 检测任务完成信号
  if ! detect_completion_signal "$tool_output"; then
    # 没有检测到完成信号，不触发 AAR
    exit 0
  fi

  # 检查是否有激活的 task gate
  local active_task_slug
  active_task_slug=$(check_active_task_gate)

  if [[ -z "$active_task_slug" ]]; then
    # 没有激活的任务，可能是琐碎任务，不强制 AAR
    echo "[PostToolUse AAR] No active task gate, skipping AAR" >&2
    exit 0
  fi

  # 生成 AAR 模板
  local aar_template
  aar_template=$(generate_aar_template "$active_task_slug")

  # 保存 AAR 到 .arccgz-harness/feedback/
  local aar_file
  aar_file=$(save_aar "$active_task_slug" "$aar_template")

  # 通知 Agent
  cat >&2 <<EOF_NOTIFY

[PostToolUse AAR Hook] Task completion detected
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Task: $active_task_slug

AAR template generated: $aar_file

Please review and complete the AAR checklist:

$(for i in "${!AAR_CHECKLIST[@]}"; do
  echo "  $((i+1)). ${AAR_CHECKLIST[$i]}"
done)

After completing the AAR:
1. Update the AAR file with findings
2. If there are new gotchas, consider adding them to .arccgz-harness/harness/known-antipatterns/
3. If there are rule changes, document them
4. Archive the task artifacts with:
   bash .arccgz-harness/harness/workflow-plugins/code-workflow/archive_task_artifacts.sh

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EOF_NOTIFY

  exit 0
}

main "$@"
"""
