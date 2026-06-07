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

  if grep -qiE 'script check after the test command|relevant script check after the test command|check-test-workflow\.sh|check-consistency\.sh' AGENTS.md; then
    pass_msg "AGENTS requires a follow-up script check after tests"
  else
    fail_msg "AGENTS must require a follow-up script check after the test command"
  fi
fi

echo "[T3] test workflow guard is mechanically enforced"
if [[ -f scripts/check-consistency.sh ]] && grep -q 'check-test-workflow\.sh' scripts/check-consistency.sh; then
  pass_msg "scripts/check-consistency.sh runs scripts/check-test-workflow.sh"
else
  fail_msg "scripts/check-consistency.sh must run scripts/check-test-workflow.sh"
fi

if [[ -f .githooks/pre-commit ]] && grep -q 'check-consistency\.sh' .githooks/pre-commit; then
  pass_msg "pre-commit routes through scripts/check-consistency.sh"
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
