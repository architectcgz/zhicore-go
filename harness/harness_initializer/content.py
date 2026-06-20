#!/usr/bin/env python3
"""Harness initializer content templates."""

from __future__ import annotations

import json


def todo_reminder_script() -> str:
    return r"""#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cwd="$(cd "$script_dir/.." && pwd)"
python3 ~/.agents/harness/todo/remind_todos.py --cwd "$cwd" "$@"
"""


def todo_governance_check_script() -> str:
    return r"""#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cwd="$(cd "$script_dir/.." && pwd)"
python3 ~/.agents/harness/todo/check_todo_governance.py --cwd "$cwd"
"""


def skill_sync_reminder_script() -> str:
    return r"""#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cwd="$(cd "$script_dir/.." && pwd)"
python3 ~/.agents/harness/skill-sync/remind_skill_sync.py --cwd "$cwd" "$@"
"""


def agent_entrypoints_check_script() -> str:
    return r"""#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cwd="$(cd "$script_dir/.." && pwd)"
exec bash "$HOME/.agents/harness/check-project-agent-entrypoints.sh" "$cwd"
"""


def script_guard_policy_content() -> str:
    return """{
  "include": [
    "scripts/check-*.sh",
    "scripts/check-*.py",
    "scripts/start-*.sh",
    "scripts/start-*.py",
    "scripts/run-*.sh",
    "scripts/run-*.py",
    "scripts/install-*.sh",
    "scripts/install-*.py",
    "scripts/uninstall-*.sh",
    "scripts/uninstall-*.py",
    "scripts/doctor-*.sh",
    "scripts/doctor-*.py",
    "harness/checks/**/*.sh",
    "harness/checks/**/*.py",
    "tools/*.sh",
    "tools/*.py"
  ],
  "exclude": [
    "scripts/lib/**"
  ],
  "max_lines": 260,
  "max_lines_by_glob": {},
  "advice": "If an operator or harness script keeps growing, split the stable entrypoint from helpers, harness/checks modules, or workflow managed assets instead of extending one large file."
}"""


def script_guard_check_script() -> str:
    return r"""#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cwd="$(cd "$script_dir/.." && pwd)"
python3 ~/.agents/harness/checks/check_script_guard.py --cwd "$cwd" "$@"
"""


def architecture_guard_paths_policy() -> str:
    return """# One repo-relative path per line.
# These are the minimum architecture fact sources that must stay present.

docs/architecture/README.md
"""


def architecture_guard_commands_policy() -> str:
    return """# One shell command per line.
# Add project-specific architecture checks here when they exist.
# Examples:
# go test ./... -run 'Architecture|Boundar'
# npm run test:architecture
# bash scripts/check-backend-architecture.sh
"""


def test_workflow_check_script() -> str:
    return r"""#!/usr/bin/env bash
set -euo pipefail

fail=0

red() { printf '\033[31m%s\033[0m' "$1"; }
green() { printf '\033[32m%s\033[0m' "$1"; }

pass_msg() {
  echo "  $(green PASS) — $1"
}

fail_msg() {
  echo "  $(red FAIL) — $1"
  fail=1
}

has_test_surface=0
indicators=()

if [[ -f package.json ]] && grep -q '"test[^"]*"[[:space:]]*:' package.json; then
  has_test_surface=1
  indicators+=("package.json scripts")
fi

if [[ -f pyproject.toml ]] && grep -Eq 'pytest|tool\.pytest' pyproject.toml; then
  has_test_surface=1
  indicators+=("pyproject pytest")
fi

if [[ -f pytest.ini || -f tox.ini ]]; then
  has_test_surface=1
  indicators+=("pytest config")
fi

if [[ -f go.mod ]] || compgen -G '*_test.go' > /dev/null || find . -path '*/vendor' -prune -o -name '*_test.go' -print -quit | grep -q .; then
  has_test_surface=1
  indicators+=("go tests")
fi

if [[ -f Cargo.toml ]]; then
  has_test_surface=1
  indicators+=("cargo")
fi

if [[ -d tests || -d __tests__ ]]; then
  has_test_surface=1
  indicators+=("test directories")
fi

if find . -path '*/node_modules' -prune -o -path '*/dist' -prune -o -path '*/build' -prune -o \
  \( -name '*.test.*' -o -name '*.spec.*' \) -print -quit | grep -q .; then
  has_test_surface=1
  indicators+=("test files")
fi

echo "[T1] detect automated test surface"
if [[ "$has_test_surface" -eq 0 ]]; then
  pass_msg "no obvious automated test surface detected; skipping test workflow enforcement"
  exit 0
fi
pass_msg "detected test surface via: ${indicators[*]}"

echo "[T2] AGENTS documents the test workflow"
if [[ ! -f AGENTS.md ]]; then
  fail_msg "missing AGENTS.md for test workflow guidance"
else
  if grep -qiE 'write or update the narrowest relevant tests first|write or update the narrowest relevant test first' AGENTS.md; then
    pass_msg "AGENTS requires narrowest relevant tests first"
  else
    fail_msg "AGENTS must require writing or updating the narrowest relevant tests first"
  fi

  if grep -qiE 'smallest relevant test command|smallest relevant test' AGENTS.md; then
    pass_msg "AGENTS requires the smallest relevant test command"
  else
    fail_msg "AGENTS must require the smallest relevant test command after test changes"
  fi

  if grep -qiE 'script check after the test command|relevant script check after the test command|check-test-workflow\.sh|check-harness-consistency\.sh' AGENTS.md; then
    pass_msg "AGENTS requires a follow-up script check after tests"
  else
    fail_msg "AGENTS must require a follow-up script check after the test command"
  fi
fi

echo "[T3] test workflow guard is mechanically enforced"
if [[ -f scripts/check-harness-consistency.sh ]] && grep -q 'check-test-workflow\.sh' scripts/check-harness-consistency.sh; then
  pass_msg "scripts/check-harness-consistency.sh runs scripts/check-test-workflow.sh"
else
  fail_msg "scripts/check-harness-consistency.sh must run scripts/check-test-workflow.sh"
fi

if [[ -f .githooks/pre-commit ]] && grep -q 'check-harness-consistency\.sh' .githooks/pre-commit; then
  pass_msg "pre-commit routes through scripts/check-harness-consistency.sh"
elif find .github/workflows -type f 2>/dev/null | xargs -r grep -q 'check-test-workflow\.sh'; then
  pass_msg "CI directly runs scripts/check-test-workflow.sh"
else
  fail_msg "wire test workflow enforcement into pre-commit or CI"
fi

if [[ "$fail" -eq 0 ]]; then
  echo "$(green '✓ test workflow checks passed')"
else
  echo "$(red '✗ test workflow checks failed')"
fi

exit "$fail"
"""


def architecture_guard_script() -> str:
    return r"""#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$script_dir/.."

fail=0
ran_command=0

red() { printf '\033[31m%s\033[0m' "$1"; }
green() { printf '\033[32m%s\033[0m' "$1"; }

pass_msg() {
  echo "  $(green PASS) — $1"
}

fail_msg() {
  echo "  $(red FAIL) — $1"
  fail=1
}

check_contains() {
  local file="$1" pattern="$2" label="$3"
  if [[ ! -f "$file" ]]; then
    fail_msg "$label: missing $file"
  elif grep -qE "$pattern" "$file"; then
    pass_msg "$label"
  else
    fail_msg "$label"
  fi
}

echo "[A1] architecture fact source exists"
if [[ -d "docs/architecture" ]]; then
  pass_msg "docs/architecture directory exists"
else
  fail_msg "docs/architecture directory is missing"
fi

if [[ -f "docs/architecture/README.md" ]]; then
  pass_msg "docs/architecture/README.md exists"
else
  fail_msg "docs/architecture/README.md is missing"
fi

echo "[A2] architecture navigation is wired"
check_contains "AGENTS.md" 'docs/architecture/' "AGENTS references docs/architecture/"
check_contains "docs/README.md" 'docs/architecture/' "docs/README.md references docs/architecture/"

echo "[A3] architecture guard policies exist"
if [[ -f "harness/policies/architecture-guard-paths.txt" ]]; then
  pass_msg "harness/policies/architecture-guard-paths.txt exists"
else
  fail_msg "harness/policies/architecture-guard-paths.txt is missing"
fi

if [[ -f "harness/policies/architecture-guard-commands.txt" ]]; then
  pass_msg "harness/policies/architecture-guard-commands.txt exists"
else
  fail_msg "harness/policies/architecture-guard-commands.txt is missing"
fi

echo "[A4] required architecture paths stay present"
if [[ -f "harness/policies/architecture-guard-paths.txt" ]]; then
  while IFS= read -r path || [[ -n "$path" ]]; do
    path="${path#"${path%%[![:space:]]*}"}"
    path="${path%"${path##*[![:space:]]}"}"
    [[ -z "$path" || "$path" == \#* ]] && continue
    if [[ -e "$path" ]]; then
      pass_msg "$path exists"
    else
      fail_msg "$path is missing"
    fi
  done < "harness/policies/architecture-guard-paths.txt"
fi

echo "[A5] project-specific architecture commands"
if [[ -f "harness/policies/architecture-guard-commands.txt" ]]; then
  while IFS= read -r command || [[ -n "$command" ]]; do
    command="${command#"${command%%[![:space:]]*}"}"
    command="${command%"${command##*[![:space:]]}"}"
    [[ -z "$command" || "$command" == \#* ]] && continue
    ran_command=1
    echo "  [cmd] $command"
    if bash -lc "$command"; then
      pass_msg "$command"
    else
      fail_msg "$command"
    fi
  done < "harness/policies/architecture-guard-commands.txt"
fi

if [[ "$ran_command" -eq 0 ]]; then
  pass_msg "no project-specific architecture commands registered"
fi

if [[ "$fail" -eq 0 ]]; then
  echo "$(green '✓ architecture guard passed')"
else
  echo "$(red '✗ architecture guard failed')"
fi

exit "$fail"
"""


def commit_message_policy_content(profile: str) -> str:
    ctf = profile == "ctf-platform"
    examples = {
        "fix": "fix(frontend): 修正拓扑页导出按钮禁用态" if ctf else "fix(api): 修正登录接口限流判断",
        "refactor": "refactor(topology): 拆分画布工作区组件" if ctf else "refactor(harness): 拆分任务启动脚本",
        "docs": "docs: 补齐提交信息约束说明" if ctf else "docs(workflow): 补齐提交规范说明",
    }
    policy = {
        "allowed_types": [
            "feat",
            "fix",
            "refactor",
            "docs",
            "test",
            "chore",
            "build",
            "ci",
            "perf",
            "style",
            "revert",
        ],
        "require_chinese_description": True,
        "body": {
            "min_detail_lines": 2,
            "min_visible_chars": 20,
            "ignored_prefixes": ["Task:"],
        },
        "task": {
            "required_when_active": True,
            "line_prefix": "Task:",
        },
        "messages": {
            "invalid_subject": "\n".join(
                [
                    "[commit-msg] 提交信息格式不符合约束。",
                    "要求：英文类型 + 可选英文/模块 scope + 中文描述",
                    "示例：",
                    f"  {examples['fix']}",
                    f"  {examples['refactor']}",
                    f"  {examples['docs']}",
                ]
            ),
            "missing_chinese_description": "\n".join(
                [
                    "[commit-msg] 提交描述必须包含中文说明。",
                    "示例：",
                    f"  {examples['fix']}",
                ]
            ),
            "missing_detail_lines": "\n".join(
                [
                    "[commit-msg] 普通提交不能只有简短标题，必须补充详细正文。",
                    "要求：",
                    "  - 标题后保留空行",
                    "  - 正文至少两行有效内容",
                    "  - 建议直接使用多个 -m 组织提交信息",
                    "示例：",
                    '  git commit -m "docs(workflow): 收紧提交说明校验" \\',
                    '    -m "要求普通提交必须携带详细正文，不能只写单行标题。" \\',
                    '    -m "同步更新 hook 说明和仓库约定，减少后续提交漂移。"',
                ]
            ),
            "detail_too_short": "\n".join(
                [
                    "[commit-msg] 提交正文信息量不足，请补充更具体的变更说明。",
                    "要求：",
                    "  - 正文至少两行有效内容",
                    "  - 正文总信息量至少达到 20 个非空白字符",
                    "示例：",
                    '  git commit -m "docs(workflow): 收紧提交说明校验" \\',
                    '    -m "要求普通提交必须携带详细正文，不能只写单行标题。" \\',
                    '    -m "同步更新 hook 说明和仓库约定，减少后续提交漂移。"',
                ]
            ),
            "missing_task_binding": "\n".join(
                [
                    "[commit-msg] 当前 worktree 存在激活中的非琐碎任务 gate，提交正文必须显式带上 task slug。",
                    "要求：",
                    "  - 在正文单独写一行：{task_line_prefix} {task_slug}",
                    "示例：",
                    '  git commit -m "refactor(workflow): 拆分 shared skill 校验" \\',
                    '    -m "把 shared skill 完整性检查从 consistency 总脚本里拆成独立子脚本。" \\',
                    '    -m "同步让安装脚本与治理审计分别调用，减少职责混写。" \\',
                    '    -m "{task_line_prefix} {task_slug}"',
                ]
            ),
        },
    }
    return json.dumps(policy, ensure_ascii=False, indent=2)


def commit_message_check_script() -> str:
    return r"""#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "[commit-msg] 用法: bash scripts/check-commit-message.sh <commit-message-file>" >&2
  exit 1
fi

message_file="$1"
if [[ ! -f "$message_file" ]]; then
  echo "[commit-msg] 找不到提交信息文件: $message_file" >&2
  exit 1
fi

root_dir="$(git rev-parse --show-toplevel)"
agents_home="${AGENTS_HOME:-$HOME/.agents}"
checker="$agents_home/harness/commit-message/check_commit_message.py"
policy_file="$root_dir/harness/policies/commit-message.json"

if [[ ! -f "$checker" ]]; then
  echo "[commit-msg] 找不到共享提交信息检查器: $checker" >&2
  echo "[commit-msg] 请先同步 ~/.agents/harness，或检查 AGENTS_HOME 配置" >&2
  exit 1
fi

if [[ ! -f "$policy_file" ]]; then
  echo "[commit-msg] 找不到项目提交信息策略: $policy_file" >&2
  exit 1
fi

cmd=(python3 "$checker" --message-file "$message_file" --policy-file "$policy_file")
if [[ -x "$root_dir/scripts/check-startup-gate.sh" ]]; then
  active_task_slug="$(bash "$root_dir/scripts/check-startup-gate.sh" --print-active-slug 2>/dev/null || true)"
  if [[ -n "$active_task_slug" ]]; then
    cmd+=(--active-task-slug "$active_task_slug")
  fi
fi

exec "${cmd[@]}"
"""


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
  "harness/policies/reuse-first.yaml"
  "harness/policies/commit-message.json"
  "harness/policies/project-patterns.yaml"
  "harness/policies/script-layer-manifest.json"
  "docs/文档规范.md"
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
3. Document the change in feedback/ or commit message
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
                            "command": "bash harness/hooks/protect-core-files.sh",
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
                    "script": "bash harness/hooks/protect-core-files.sh"
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
  "是否踩到新坑？需要记录到 Known Gotchas 或 feedback/"
  "规则文件是否被意外修改？检查 AGENTS.md 和 harness/policies/"
  "测试是否都通过？运行相关测试命令"
  "是否有新的架构决策需要记录到 docs/architecture/？"
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
  if [[ -x scripts/check-startup-gate.sh ]]; then
    local active_slug
    active_slug="$(bash scripts/check-startup-gate.sh --print-active-slug 2>/dev/null || true)"
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
- [ ] harness/policies/
- [ ] docs/文档规范.md
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

<!-- 是否有新的架构决策？需要记录到 docs/architecture/ 吗？ -->

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

# 保存 AAR 到 feedback/
save_aar() {
  local task_slug="$1"
  local aar_content="$2"
  local timestamp
  timestamp=$(date +%Y%m%d-%H%M%S)

  local feedback_dir="feedback/aar"
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

  # 保存 AAR 到 feedback/
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
2. If there are new gotchas, consider adding them to harness/known-antipatterns/
3. If there are rule changes, document them
4. Archive the task artifacts with:
   bash harness/workflow-plugins/code-workflow/archive_task_artifacts.sh

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EOF_NOTIFY

  exit 0
}

main "$@"
"""


def aar_hook_readme() -> str:
    """Generate README for AAR hooks."""
    return """# After Action Review (AAR) Hook

## 作用

PostToolUse AAR Hook 在任务完成后自动触发，引导 Agent 完成 After Action Review。

## 触发条件

- 检测到任务完成信号：`completed`, `finished`, `done`, `merged`, `task complete`
- 当前 worktree 有激活的 task gate（非琐碎任务）

## AAR 检查清单

1. **新坑检测**：是否踩到新坑？需要记录到 Known Gotchas
2. **规则变化**：核心规则文件是否被修改？
3. **测试结果**：测试是否都通过？
4. **架构决策**：是否有新的架构决策需要记录？
5. **可复用模式**：是否有可以提取到 skill 或 policy 的模式？
6. **文档更新**：哪些文档需要更新？

## 工作流程

```
Task 完成
    ↓
PostToolUse Hook 检测到完成信号
    ↓
生成 AAR 模板 (feedback/aar/<timestamp>-<task-slug>.md)
    ↓
通知 Agent 完成 AAR 检查清单
    ↓
Agent 填写 AAR（手动）
    ↓
归档 task artifacts
```

## 生成的 AAR 文件

**位置**：`feedback/aar/<timestamp>-<task-slug>.md`

**格式**：
- 检查清单（checkboxes）
- 新发现的坑
- 规则变化
- 测试结果
- 架构决策
- 可复用模式
- 文档更新
- 经验教训
- 后续行动

## 配置

### 自定义完成信号

编辑 `harness/hooks/post-tooluse-aar.sh`：

```bash
COMPLETION_SIGNALS=(
  "completed"
  "finished"
  "done"
  "merged"
  "task complete"
  # 添加自定义信号
  "实现完成"
  "bugfix done"
)
```

### 自定义检查清单

编辑 `harness/hooks/post-tooluse-aar.sh`：

```bash
AAR_CHECKLIST=(
  "是否踩到新坑？"
  "规则文件是否被意外修改？"
  # 添加自定义检查项
  "是否更新了 API 文档？"
  "是否需要通知团队？"
)
```

## 与 completion-full gate 的集成

AAR Hook 是 completion-full gate 的一部分：

```
非琐碎任务完成
    ↓
1. 运行 completion validation（技术验证）
    ↓
2. PostToolUse AAR Hook（反思和记录）
    ↓
3. 归档 task artifacts
    ↓
4. 关闭 task gate
```

## 手动触发 AAR

```bash
# 为当前任务手动生成 AAR
bash harness/hooks/post-tooluse-aar.sh "task complete"
```

## 查看历史 AAR

```bash
# 列出所有 AAR
ls -lt feedback/aar/

# 查看最近的 AAR
cat feedback/aar/$(ls -t feedback/aar/ | head -1)
```

## 最佳实践

### 1. 及时填写 AAR
- 任务完成后立即填写，记忆最清晰
- 不要拖延，否则细节会遗忘

### 2. 具体而非抽象
- ❌ "遇到了一些问题"
- ✅ "Vue 组件在 SSR 时访问 window 对象导致报错"

### 3. 记录解决方案
- 不只记录问题，更要记录如何解决
- 包括尝试过的方案和最终方案

### 4. 提取可复用模式
- 如果同样的坑可能被其他人踩到，提取到 Known Gotchas
- 如果同样的解决方案可能被复用，提取到 skill 或 policy

### 5. 更新文档
- AAR 中发现的架构决策应该更新到 docs/architecture/
- 新的约束应该更新到 AGENTS.md 或相关 policy

## 示例 AAR

参见：`feedback/aar/example-aar.md`
"""


def aar_example() -> str:
    """Generate example AAR file."""
    return """# After Action Review — example-task

Date: 2026-06-20
Task: example-task

## 检查清单

- [x] 是否踩到新坑？需要记录到 Known Gotchas 或 feedback/
- [x] 规则文件是否被意外修改？检查 AGENTS.md 和 harness/policies/
- [x] 测试是否都通过？运行相关测试命令
- [x] 是否有新的架构决策需要记录到 docs/architecture/？
- [ ] 是否有可复用的模式需要提取到 skill 或 policy？
- [x] 是否有需要更新的文档？

## 新发现的坑（Known Gotchas）

### Vue 组件在 SSR 时访问 window 对象

- **现象**：开发环境正常，SSR 构建时报错 `ReferenceError: window is not defined`
- **原因**：Vue 组件在 `<script setup>` 顶层直接访问了 `window.innerWidth`
- **解决方案**：将访问 `window` 的代码移到 `onMounted()` 生命周期钩子中
- **预防措施**：
  - 在 AGENTS.md 中添加规则：前端组件不得在顶层访问浏览器全局对象
  - 添加 ESLint 规则检测这类问题

## 规则变化

- [x] AGENTS.md — 添加了 SSR 兼容性规则
- [ ] harness/policies/
- [ ] docs/文档规范.md
- [ ] .agents/skills/*/SKILL.md

## 测试结果

```bash
# 执行的测试命令
npm run test:unit
npm run build:ssr

# 测试结果
✓ 单元测试全部通过
✓ SSR 构建成功
```

## 架构决策

**决策**：前端组件统一使用 Composition API，避免在 `<script setup>` 顶层访问浏览器 API

**理由**：
- 保证 SSR 兼容性
- 更好的生命周期管理
- 更容易测试

**影响**：
- 需要更新现有组件（约 5 个）
- 需要在 AGENTS.md 中添加规则

**记录位置**：`docs/architecture/frontend-ssr-compatibility.md`

## 可复用模式

**模式**：SSR 安全的浏览器 API 访问

可以提取到 `frontend-engineer` skill 中：

```markdown
## SSR 兼容性

- 不要在 `<script setup>` 顶层访问 `window`、`document`、`navigator` 等浏览器全局对象
- 浏览器 API 访问必须在 `onMounted()` 或 `onBeforeMount()` 生命周期钩子中
- 使用条件判断：`if (typeof window !== 'undefined')`
```

## 文档更新

- [x] `docs/architecture/frontend-ssr-compatibility.md` — 新增
- [x] `AGENTS.md` — 添加 SSR 兼容性规则
- [ ] `code/frontend/README.md` — 补充 SSR 注意事项

## 经验教训

### 做得好的

- 及时发现问题，在 SSR 构建阶段拦截
- 快速定位根因（`window` 访问）
- 提取了可复用的架构规则

### 需要改进的

- 应该在开发初期就建立 SSR 兼容性检查
- 缺少 ESLint 规则自动检测这类问题
- 文档中没有提前说明 SSR 约束

### 下次注意

- 新增前端组件时，先检查是否有浏览器 API 访问
- 在 AGENTS.md 中明确 SSR 兼容性要求
- 添加自动化检查（ESLint 规则）

## 后续行动

- [ ] 更新现有的 5 个组件，修复 `window` 访问
- [ ] 添加 ESLint 规则：`no-restricted-globals` for `window`, `document`
- [ ] 在 CI 中添加 SSR 构建检查
- [ ] 更新 `frontend-engineer` skill，添加 SSR 兼容性规则
"""


def aar_directory_readme() -> str:
    """Generate README for feedback/aar/ directory."""
    return """# After Action Review (AAR) Archive

本目录存放任务完成后的 After Action Review (AAR) 记录。

## 目录结构

```
feedback/aar/
├── README.md                           # 本文件
├── example-aar.md                      # AAR 示例
├── 20260620-143000-task-slug.md       # 实际 AAR
└── ...
```

## AAR 文件命名

格式：`<timestamp>-<task-slug>.md`

示例：
- `20260620-143000-implement-pagination.md`
- `20260621-093000-fix-ssr-bug.md`

## AAR 触发

PostToolUse AAR Hook 在检测到任务完成信号时自动生成 AAR 模板：

```bash
# 自动触发（通过 Hook）
[Agent 完成任务] → PostToolUse Hook → 生成 AAR 模板

# 手动触发
bash harness/hooks/post-tooluse-aar.sh "task complete"
```

## AAR 内容结构

每个 AAR 包含：

1. **检查清单** — 标准化的反思项目
2. **新发现的坑** — Known Gotchas
3. **规则变化** — 核心规则文件的修改
4. **测试结果** — 验证结果
5. **架构决策** — 新的架构决策
6. **可复用模式** — 可以提取到 skill 或 policy 的模式
7. **文档更新** — 需要更新的文档
8. **经验教训** — 做得好的、需要改进的、下次注意的
9. **后续行动** — 待办事项

## 使用 AAR

### 查看最近的 AAR

```bash
cat feedback/aar/$(ls -t feedback/aar/*.md | head -1)
```

### 搜索特定主题的 AAR

```bash
grep -l "SSR" feedback/aar/*.md
grep -l "Vue" feedback/aar/*.md
```

### 提取 Known Gotchas

```bash
# 查看所有新发现的坑
grep -A 10 "## 新发现的坑" feedback/aar/*.md
```

### 提取可复用模式

```bash
# 查看所有可复用模式
grep -A 10 "## 可复用模式" feedback/aar/*.md
```

## AAR 生命周期

```
1. 任务完成
   ↓
2. PostToolUse Hook 生成 AAR 模板
   ↓
3. Agent 填写 AAR
   ↓
4. 保存到 feedback/aar/
   ↓
5. 定期回顾（每月/每季度）
   ↓
6. 提取到 Known Gotchas / Skills / Policies
   ↓
7. 归档（保留在 feedback/aar/ 或移到 archive/）
```

## 定期回顾

建议每月或每季度回顾 AAR：

```bash
# 查看本月的所有 AAR
ls feedback/aar/$(date +%Y%m)*.md

# 统计高频问题
grep "## 新发现的坑" feedback/aar/*.md | wc -l
```

## 提取到其他位置

### 提取到 Known Gotchas

如果同样的坑被多个 AAR 提到，提取到：
- `harness/known-antipatterns/EXAMPLES.md`
- 或项目特定的 gotchas 文档

### 提取到 Skills

如果发现可复用的模式，提取到：
- `.agents/skills/<skill-name>/`
- 或更新现有 skill

### 提取到 Policies

如果发现新的约束，提取到：
- `harness/policies/<policy-name>.yaml`
- 或更新 `AGENTS.md`

## 最佳实践

1. **及时填写**：任务完成后立即填写，记忆最清晰
2. **具体而非抽象**：记录具体现象和解决方案
3. **提取可复用模式**：主动思考是否可以防止其他人踩坑
4. **更新文档**：AAR 中的发现应该反映到文档中
5. **定期回顾**：每月回顾，提取高频问题

## 示例

参见：`example-aar.md`
"""


def known_antipatterns_examples() -> str:
    """Generate Known Antipatterns EXAMPLES.md template."""
    return """# Known Antipatterns — Real Examples

## 硬约束

**这张表只能从真实失败里抄，不能凭空想象。**

每个反例必须包含：
- 真实的 before/after 代码
- 明确的为什么错、如何改
- 可追溯的来源（commit hash、PR、issue）

---

## 反模式目录

### 代码质量
- [过度设计：用户只要修 bug，Agent 加了无关改动](#过度设计用户只要修-bug-agent-加了无关改动)
- [过早抽象：一个用例就抽象成通用组件](#过早抽象一个用例就抽象成通用组件)

### 测试
- [测试锁定实现细节而非行为](#测试锁定实现细节而非行为)
- [前端测试只断言 class 名](#前端测试只断言-class-名)

### 架构
- [跨层重复 normalize/default/validate](#跨层重复-normalizedefaultvalidate)
- [frontend entities 反向依赖 features](#frontend-entities-反向依赖-features)

---

## 代码质量反模式

### 过度设计：用户只要修 bug，Agent 加了无关改动

**来源**：[填写 commit hash 或 PR 链接]

**用户请求**：
```
修复登录按钮点击无响应的 bug
```

**❌ Before（违反 Surgical Changes）**：
```typescript
// 用户要求的修复 ✅
- 修复登录按钮事件绑定

// Agent 自己加的"改善" ❌
+ 添加 loading 状态显示
+ 添加按钮防抖
+ 添加错误重试逻辑
+ 添加日志记录
+ 重构按钮组件为通用组件
```

**为什么错**：
- 只有第一项是用户要求的
- loading、防抖、重试、日志、重构都是 Agent 自己加的
- 用户无法干净地"撤销最后一个功能"（因为混在一起了）

**✅ After（正确做法）**：
```typescript
// 只修复用户要求的
- 修复登录按钮事件绑定

// 其他"改善"应该：
// 1. 先问用户是否需要
// 2. 或者分成独立的 commit/PR
// 3. 或者完全不做（用户没要求）
```

**检验句**：
- 每一行改动都能追溯到用户的请求吗？ → **否**
- 如果用户说"撤销最后一个功能"，是否能干净删除？ → **否**

---

### 过早抽象：一个用例就抽象成通用组件

**来源**：[填写 commit hash 或 PR 链接]

**用户请求**：
```
添加用户列表页面
```

**❌ Before（违反 Avoid Premature Abstraction）**：
```typescript
// 创建了"通用" DataTable 组件
interface DataTableProps<T> {
  data: T[];
  columns: Column<T>[];
  onSort?: (key: keyof T) => void;
  onFilter?: (filters: Record<string, any>) => void;
  onPaginate?: (page: number) => void;
  loading?: boolean;
  error?: Error;
  // ... 30 行接口定义
}

// 创建了"通用" usePagination hook
// 创建了"通用" useFilter hook

// 只有一个用例：用户列表
```

**为什么错**：
- 只有一个用例（用户列表），但抽象了"通用"组件
- 接口比要解决的问题更复杂
- 通用接口有 30 行，但用户列表只用了 5 行

**✅ After（正确做法）**：
```typescript
// 第一个用例：直接实现，不抽象
<table>
  <thead>...</thead>
  <tbody>
    {users.map(user => <tr>...</tr>)}
  </tbody>
</table>

// 第二个用例：仍然直接实现，寻找共同点

// 第三个用例：现在有 3 个实例，可以抽象了
// 提取真正共同的部分（通常比第一次想象的简单得多）
```

**检验句**：
- 这个抽象是为几个用例设计的？ → **1 个**（过早）
- 这个接口是否比它要解决的问题更复杂？ → **是**

---

## 测试反模式

### 测试锁定实现细节而非行为

**来源**：[填写 commit hash 或 PR 链接]

**用户请求**：
```
测试用户登录功能
```

**❌ Before（锁定实现细节）**：
```typescript
test('login flow', () => {
  const spy = vi.spyOn(authService, 'login')

  render(<LoginForm />)
  fireEvent.input(emailInput, 'user@example.com')
  fireEvent.input(passwordInput, 'password123')
  fireEvent.click(submitButton)

  // 测试实现细节：调用了哪个函数
  expect(spy).toHaveBeenCalledWith({
    email: 'user@example.com',
    password: 'password123'
  })

  // 测试实现细节：内部状态
  expect(component.state.isLoading).toBe(true)
})
```

**为什么错**：
- 测试关心"调用了 authService.login"（实现）
- 测试关心"isLoading 状态"（实现）
- 重构内部实现时测试会失败，即使行为没变

**✅ After（测试行为）**：
```typescript
test('successful login redirects to dashboard', async () => {
  // Mock API 响应
  server.use(
    http.post('/api/login', () => {
      return HttpResponse.json({ token: 'abc123' })
    })
  )

  render(<LoginForm />)

  // 用户行为
  await userEvent.type(screen.getByLabelText('Email'), 'user@example.com')
  await userEvent.type(screen.getByLabelText('Password'), 'password123')
  await userEvent.click(screen.getByRole('button', { name: 'Login' }))

  // 验证结果：用户看到什么
  await waitFor(() => {
    expect(screen.getByText('Dashboard')).toBeInTheDocument()
  })
})
```

**检验句**：
- 这个测试是在验证行为还是在验证实现？ → **行为**
- 重构内部实现后，测试是否仍然通过？ → **是**

---

### 前端测试只断言 class 名

**来源**：[填写 commit hash 或 PR 链接]

**用户请求**：
```
测试按钮组件
```

**❌ Before（锁定 class 名）**：
```typescript
test('button has correct classes', () => {
  const { container } = render(<Button variant="primary">Click</Button>)

  expect(container.firstChild).toHaveClass('btn')
  expect(container.firstChild).toHaveClass('btn-primary')
  expect(container.firstChild).toHaveClass('px-4')
  expect(container.firstChild).toHaveClass('py-2')
  expect(container.firstChild).toHaveClass('rounded-md')
})
```

**为什么错**：
- 测试只证明"源码里包含这些 class"
- 改用不同的 CSS 方案（Tailwind → CSS Modules）时测试全炸
- 没有测试任何用户可见的行为

**✅ After（测试行为和可见状态）**：
```typescript
test('primary button is visually distinct and clickable', async () => {
  const handleClick = vi.fn()
  render(<Button variant="primary" onClick={handleClick}>Click</Button>)

  const button = screen.getByRole('button', { name: 'Click' })

  // 测试用户可见的行为
  await userEvent.click(button)
  expect(handleClick).toHaveBeenCalledOnce()

  // 如果需要测试样式，测试计算后的样式
  expect(button).toHaveStyle({
    backgroundColor: 'rgb(59, 130, 246)', // primary color
  })
})
```

**检验句**：
- 这个测试是否只锁定 class 名/markup 细节？ → **否**
- 改用不同的 CSS 方案后，测试是否仍然有效？ → **是**

---

## 架构反模式

### 跨层重复 normalize/default/validate

**来源**：[填写 commit hash 或 PR 链接]

**用户请求**：
```
实现分页查询 API
```

**❌ Before（跨层重复）**：
```go
// Handler 层
func (h *Handler) ListUsers(c *gin.Context) {
  page := c.DefaultQuery("page", "1")  // normalize + default
  pageNum, _ := strconv.Atoi(page)
  if pageNum < 1 { pageNum = 1 }       // validate + default

  users, _ := h.service.ListUsers(pageNum, 10)
  c.JSON(200, users)
}

// Service 层
func (s *Service) ListUsers(page, pageSize int) ([]User, error) {
  if page < 1 { page = 1 }            // 重复 validate + default
  if pageSize < 1 { pageSize = 10 }   // 重复 validate + default

  return s.repo.ListUsers(page, pageSize)
}

// Repository 层
func (r *Repo) ListUsers(page, pageSize int) ([]User, error) {
  if page < 1 { page = 1 }            // 重复 validate + default
  if pageSize < 1 { pageSize = 10 }   // 重复 validate + default

  offset := (page - 1) * pageSize
  // ...
}
```

**为什么错**：
- 同一个语义（"page 至少为 1"）在三层都重复
- 不是"安全兜底"，是"没有明确 owner"
- 修改默认值时需要改三个地方

**✅ After（单一 owner）**：
```go
// Handler 层：负责 normalize + default + validate
func (h *Handler) ListUsers(c *gin.Context) {
  req, err := parseListUsersRequest(c)  // 唯一 owner
  if err != nil {
    c.JSON(400, gin.H{"error": err.Error()})
    return
  }

  users, _ := h.service.ListUsers(req)
  c.JSON(200, users)
}

func parseListUsersRequest(c *gin.Context) (*ListUsersRequest, error) {
  page := c.DefaultQuery("page", "1")
  pageNum, err := strconv.Atoi(page)
  if err != nil || pageNum < 1 {
    return nil, errors.New("invalid page")
  }

  return &ListUsersRequest{Page: pageNum, PageSize: 10}, nil
}

// Service 层：只接收已验证的请求
func (s *Service) ListUsers(req *ListUsersRequest) ([]User, error) {
  // 不需要重复校验，req 已经是有效的
  return s.repo.ListUsers(req.Page, req.PageSize)
}

// Repository 层：只接收已验证的参数
func (r *Repo) ListUsers(page, pageSize int) ([]User, error) {
  // 不需要重复校验，参数已经是有效的
  offset := (page - 1) * pageSize
  // ...
}
```

**检验句**：
- 这个 normalize/default/validate 逻辑是在唯一 owner 层吗？ → **是**
- 改默认值时需要改几个地方？ → **1 个**

---

### frontend entities 反向依赖 features

**来源**：[填写 commit hash 或 PR 链接]

**用户请求**：
```
在用户列表中显示用户卡片
```

**❌ Before（反向依赖）**：
```typescript
// entities/user/ui/UserCard.vue
<script setup>
import { useRouter } from 'vue-router'
import { useUserActions } from '@/features/user-management/composables'

// entities 依赖了 features 的具体实现
const router = useRouter()
const { deleteUser, editUser } = useUserActions()

const handleEdit = () => {
  router.push(`/users/${props.user.id}/edit`)  // 知道具体路由
}
</script>
```

**为什么错**：
- `entities/user` 应该只表达"用户是什么"
- 现在它知道了"用户管理功能的路由"和"用户管理功能的操作"
- 反向依赖：entities → features（违反依赖方向）

**✅ After（正确的依赖方向）**：
```typescript
// entities/user/ui/UserCard.vue
<script setup>
// entities 不知道具体的 features
// 只暴露事件，由 features 决定如何处理
const emit = defineEmits<{
  edit: [userId: string]
  delete: [userId: string]
}>()
</script>

<template>
  <div class="user-card">
    <span>{{ user.name }}</span>
    <button @click="emit('edit', user.id)">Edit</button>
    <button @click="emit('delete', user.id)">Delete</button>
  </div>
</template>

// features/user-management/ui/UserListPage.vue
<script setup>
import UserCard from '@/entities/user/ui/UserCard.vue'
import { useRouter } from 'vue-router'
import { useDeleteUser } from '../api'

// features 决定如何处理 entities 的事件
const router = useRouter()
const { mutate: deleteUser } = useDeleteUser()

const handleEdit = (userId: string) => {
  router.push(`/users/${userId}/edit`)
}
</script>

<template>
  <UserCard
    v-for="user in users"
    :key="user.id"
    :user="user"
    @edit="handleEdit"
    @delete="deleteUser"
  />
</template>
```

**检验句**：
- `entities/*` 中的内容是否反向依赖了具体 feature？ → **否**
- 依赖方向是否是 features → entities？ → **是**

---

## 如何使用这个文件

### 1. 踩到坑时立即记录

```bash
# 创建新的反例条目
# 格式：### [简短标题]
# 必填：来源（commit hash/PR）、before/after、为什么错、检验句
```

### 2. 定期回顾

```bash
# 每月回顾 feedback/aar/，提取高频反模式
grep -h "## 新发现的坑" feedback/aar/*.md | sort | uniq -c | sort -rn
```

### 3. 集成到 skills

高频反模式应该：
- 更新对应 skill 的 SKILL.md
- 添加到 Red Flags 表
- 添加到检验句清单

### 4. 集成到 code review

code reviewer 可以引用这个文件：
```markdown
这个改动违反了 [Known Antipattern: 过度设计](#过度设计)
```

---

## 贡献指南

### 添加新反例的标准

1. **必须是真实失败**
   - 有 commit hash、PR 或 issue 可追溯
   - 不能是"理论上可能出错"

2. **必须有 before/after 代码**
   - Before：真实的错误代码
   - After：正确的修复代码
   - 不能只有文字描述

3. **必须说明为什么错**
   - 不是"不符合最佳实践"
   - 而是"导致了具体问题 X"

4. **必须有检验句**
   - 提供可验证的检查点
   - 让 Agent 在生成后能真的去问

### 反例质量检查清单

- [ ] 有可追溯的来源（commit hash/PR/issue）
- [ ] 有真实的 before/after 代码
- [ ] 说明了为什么错（具体问题，不是抽象原则）
- [ ] 提供了检验句（Agent 可以真的去检查）
- [ ] 归类到正确的章节

---

## 统计

- **总反例数**：[自动更新]
- **最近添加**：[自动更新]
- **最高频反模式**：[从 AAR 中统计]
"""


def known_antipatterns_readme() -> str:
    """Generate README for harness/known-antipatterns/ directory."""
    return """# Known Antipatterns

本目录存放从真实失败中提取的反模式案例。

## 硬约束

**这张表只能从真实失败里抄，不能凭空想象。**

## 文件结构

```
harness/known-antipatterns/
├── README.md          # 本文件
└── EXAMPLES.md        # 反例库（主文件）
```

## 反例来源

### 1. 从 AAR 中提取

```bash
# 查看所有"新发现的坑"
grep -A 10 "## 新发现的坑" feedback/aar/*.md

# 提取高频问题
grep -h "###" feedback/aar/*.md | sort | uniq -c | sort -rn
```

### 2. 从 Code Review 中提取

```bash
# 查看所有 review findings
ls docs/reviews/*/findings.md
```

### 3. 从 Git 历史中提取

```bash
# 查找修复类提交
git log --grep="fix:" --grep="bug:" --oneline

# 查看修复前后的 diff
git show <commit-hash>
```

## 如何添加反例

### 标准模板

```markdown
### [简短标题]

**来源**：[commit hash / PR链接 / issue链接]

**用户请求**：
```
[用户的原始请求]
```

**❌ Before（为什么错）**：
```[language]
[真实的错误代码]
```

**为什么错**：
- [具体问题1]
- [具体问题2]

**✅ After（正确做法）**：
```[language]
[修复后的代码]
```

**检验句**：
- [检验句1] → [结果]
- [检验句2] → [结果]
```

### 质量检查

添加前确认：
- [ ] 有可追溯的来源
- [ ] 有真实的代码（不是伪代码）
- [ ] 说明了具体问题（不是抽象原则）
- [ ] 提供了检验句

## 使用场景

### 1. Agent 自查

完成代码后，Agent 可以：
```bash
# 搜索相关反模式
grep -i "抽象" harness/known-antipatterns/EXAMPLES.md
```

### 2. Code Review 引用

Reviewer 可以引用：
```markdown
这个改动违反了 [Known Antipattern: 过度设计](harness/known-antipatterns/EXAMPLES.md#过度设计)
```

### 3. 更新 Skills

高频反模式应该：
- 更新到对应 skill
- 添加到 Red Flags
- 添加到检验句清单

## 定期维护

### 每月回顾

1. 回顾本月的 AAR
2. 提取高频反模式
3. 添加到 EXAMPLES.md
4. 更新相关 skills

### 统计

```bash
# 反例总数
grep -c "^###" harness/known-antipatterns/EXAMPLES.md

# 最近添加
git log --oneline harness/known-antipatterns/EXAMPLES.md | head -5
```

## 参考

- AAR 目录：`feedback/aar/`
- 检验句指南：`~/.agents/harness/docs/verification-questions-guide.md`
- Code Review：`docs/reviews/`
"""


def test_trigger_rate_script() -> str:
    """Generate test-trigger-rate.sh wrapper script."""
    return r"""#!/usr/bin/env bash
#
# Test skill description trigger rates
#
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cwd="$(cd "$script_dir/.." && pwd)"

agents_home="${AGENTS_HOME:-$HOME/.agents}"
python_script="$agents_home/harness/test-trigger-rate.py"

if [[ ! -f "$python_script" ]]; then
  echo "[test-trigger-rate] 找不到 Python 脚本: $python_script" >&2
  exit 1
fi

exec python3 "$python_script" --agents-md "$cwd/AGENTS.md" "$@"
"""


def test_trigger_rate_readme() -> str:
    """Generate README for trigger rate testing."""
    return """# Skill Description Trigger Rate Testing

## 作用

测试 skill description 的触发率，确保用户的自然语言能够正确触发对应的 skill。

## 原理

根据 [如何写一个好的 skill 让你的效率加倍](https://linux.do/t/topic/1923706)：

> test-trigger.sh 会从 Common Tasks 里生成真实用户可能说的提示词，用来测 description 的触发率——单独读一遍 SKILL.md 觉得没问题，跑 test-trigger.sh 才发现一半的触发短语命中不了。

## 使用方式

### 基本用法

```bash
# 测试当前项目的触发率
bash scripts/test-trigger-rate.sh

# 输出示例：
# ======================================================================
# Skill Description Trigger Rate Report
# ======================================================================
#
# ✓ Backend feature (API/Service/Repository)
#    Skill: backend-engineer
#    Trigger Rate: 7/8 (87.5%)
#
# ✗ Frontend feature (Page/Component)
#    Skill: frontend-engineer
#    Trigger Rate: 4/7 (57.1%)
#    ⚠ Low trigger rate! Recommendation:
#       - Review skill description
#       - Add more keywords
#       - Consider user's natural language
#
# ----------------------------------------------------------------------
# Overall Trigger Rate: 45/60 (75.0%)
# ======================================================================
```

### 详细模式

```bash
# 显示每个测试用例的结果
bash scripts/test-trigger-rate.sh --verbose
```

## 工作流程

```
1. 从 AGENTS.md 的 Quick Routing 表提取任务类型
   ↓
2. 为每种任务类型生成用户可能的表达方式
   ↓
3. 查找对应 skill 的 description
   ↓
4. 测试用户表达是否能触发 skill
   ↓
5. 生成触发率报告
```

## 触发率标准

- **✓ 良好**：触发率 ≥ 80%
- **✗ 需要改进**：触发率 < 80%

## 改进低触发率的方法

### 1. 扩展 skill description 的关键词

❌ Before：
```yaml
description: Use for backend development
```

✅ After：
```yaml
description: Use when implementing backend features, APIs, services, database operations, or backend bug fixes
```

### 2. 添加用户常用表达

在 `~/.agents/harness/test-trigger-rate.py` 的 `TASK_VARIATIONS` 中添加更多表达方式：

```python
"Backend feature": [
    "加个 API",
    "实现后端接口",
    # 添加更多用户可能说的话
    "写个接口",
    "做个服务",
]
```

### 3. 使用中英文关键词

```yaml
description: Use for backend/后端 feature/功能 implementation including API/接口, service/服务, database/数据库
```

## 集成到 CI

```yaml
# .github/workflows/test.yml
- name: Test skill trigger rates
  run: bash scripts/test-trigger-rate.sh
```

## 定期检查

建议：
- 每次添加新 skill 后运行
- 每月运行一次，确保触发率保持良好
- 更新 AGENTS.md 的 Quick Routing 表后运行

## 限制

当前实现是简化版，使用关键词匹配。更准确的实现应该：
- 使用语义相似度（embedding）
- 考虑上下文
- 支持多语言

## 自定义

### 添加新任务类型的表达方式

编辑 `~/.agents/harness/test-trigger-rate.py`：

```python
TASK_VARIATIONS = {
    "Your new task type": [
        "用户可能说的话1",
        "用户可能说的话2",
        # ...
    ],
}
```

### 调整触发率阈值

默认阈值是 80%，可以在脚本中修改：

```python
# 返回状态码（如果整体触发率 < 80%，返回 1）
return 0 if overall_rate >= 80 else 1
```

## 参考

- 原文：[如何写一个好的 skill 让你的效率加倍](https://linux.do/t/topic/1923706)
- Skill 编写指南：`~/.agents/docs/writing-skills.md`
"""
