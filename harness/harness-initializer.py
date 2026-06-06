#!/usr/bin/env python3
"""Initialize a Harness Engineering scaffold in a repo."""

from __future__ import annotations

import argparse
from pathlib import Path
import subprocess


START = "BEGIN HARNESS ENGINEERING"
END = "END HARNESS ENGINEERING"
WORKSPACE_AGENT_ENTRYPOINT_CHECK = Path.home() / "workspace" / "projects" / "scripts" / "check-agent-entrypoints.sh"
AGENTS_SKILLS_DIR = Path.home() / ".agents" / "skills"
CODEX_SKILLS_DIR = Path.home() / ".codex" / "skills"
STANDARD_DOC_DIRS = [
    "requirements",
    "contracts",
    "spec",
    "design",
    "todo",
    "architecture",
    "plan",
    "operations",
    "reviews",
    "reports",
    "improvements",
    "refs",
]
IMPROVEMENT_STATUS_DIRS = [
    "not-impl",
    "implemented",
    "agent-recorded",
    "rejected",
    "archived",
]


def resolve_skill_dir(skill_name: str) -> Path:
    for base in (AGENTS_SKILLS_DIR, CODEX_SKILLS_DIR):
        candidate = base / skill_name
        if candidate.exists():
            return candidate.resolve()
    raise RuntimeError(f"unable to resolve skill directory for {skill_name}")


DOCS_ASSETS = resolve_skill_dir("documentation-architecture") / "assets" / "docs"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", default=".", help="Target repository root")
    parser.add_argument("--project-name", default=None)
    parser.add_argument("--profile", default="generic")
    parser.add_argument("--mode", default="ctf-current", choices=["ctf-current", "strict-reference"])
    return parser.parse_args()


def write(path: Path, content: str, executable: bool = False) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content.rstrip() + "\n", encoding="utf-8")
    if executable:
        path.chmod(path.stat().st_mode | 0o111)


def read_asset(relative: str) -> str:
    return (DOCS_ASSETS / relative).read_text(encoding="utf-8")


def ensure_documentation_scaffold(repo: Path) -> None:
    for directory in STANDARD_DOC_DIRS:
        (repo / "docs" / directory).mkdir(parents=True, exist_ok=True)
    for directory in IMPROVEMENT_STATUS_DIRS:
        (repo / "docs" / "improvements" / directory).mkdir(parents=True, exist_ok=True)
    write(repo / "docs" / "documentation-rules.md", read_asset("documentation-rules.md"))
    write(repo / "docs" / "README.md", read_asset("README.md"))
    write(repo / "docs" / "improvements" / "README.md", read_asset("improvements/README.md"))


def ensure_claude_symlink(repo: Path) -> None:
    agents_path = repo / "AGENTS.md"
    claude_path = repo / "CLAUDE.md"

    if not agents_path.exists():
        raise RuntimeError(f"missing {agents_path} before creating CLAUDE.md symlink")

    if claude_path.is_symlink():
        if claude_path.resolve() == agents_path.resolve():
            return
        raise RuntimeError(
            f"{claude_path} already points to {claude_path.resolve()}, expected {agents_path.resolve()}"
        )

    if claude_path.exists():
        raise RuntimeError(
            f"{claude_path} already exists and is not a symlink; replace it manually with CLAUDE.md -> AGENTS.md"
        )

    claude_path.symlink_to("AGENTS.md")


def run_agent_entrypoint_check(repo: Path) -> None:
    if not WORKSPACE_AGENT_ENTRYPOINT_CHECK.is_file():
        return
    subprocess.run(
        ["bash", str(WORKSPACE_AGENT_ENTRYPOINT_CHECK), str(repo)],
        check=True,
    )


def managed_block(kind: str, body: str) -> str:
    return f"<!-- {START}: {kind} -->\n{body.rstrip()}\n<!-- {END}: {kind} -->"


def insert_or_replace(path: Path, kind: str, body: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    block = managed_block(kind, body)
    start = f"<!-- {START}: {kind} -->"
    end = f"<!-- {END}: {kind} -->"
    text = path.read_text(encoding="utf-8") if path.exists() else ""
    if start in text and end in text:
        before, rest = text.split(start, 1)
        _, after = rest.split(end, 1)
        path.write_text(before.rstrip() + "\n\n" + block + after, encoding="utf-8")
    else:
        sep = "\n\n" if text.rstrip() else ""
        path.write_text(text.rstrip() + sep + block + "\n", encoding="utf-8")


def insert_hook(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    text = path.read_text(encoding="utf-8") if path.exists() else "#!/usr/bin/env bash\nset -euo pipefail\n"
    start = f"# {START}: pre-commit"
    end = f"# {END}: pre-commit"
    body = f"""{start}
if [[ -x scripts/check-consistency.sh ]]; then
  bash scripts/check-consistency.sh
fi
# {END}: pre-commit"""
    if start in text and end in text:
        before, rest = text.split(start, 1)
        _, after = rest.split(end, 1)
        text = (before.rstrip() + "\n" + after.lstrip()).rstrip() + "\n"
    if 'if [[ "$needs_sync" -eq 0 ]]; then' in text:
        text = text.replace('if [[ "$needs_sync" -eq 0 ]]; then', body + '\n\nif [[ "$needs_sync" -eq 0 ]]; then', 1)
    else:
        text = text.rstrip() + "\n\n" + body + "\n"
    path.write_text(text, encoding="utf-8")
    path.chmod(path.stat().st_mode | 0o111)


def insert_commit_msg_hook(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    text = path.read_text(encoding="utf-8") if path.exists() else "#!/usr/bin/env bash\nset -euo pipefail\n"
    start = f"# {START}: commit-msg"
    end = f"# {END}: commit-msg"
    body = f"""{start}
cd "$(git rev-parse --show-toplevel)"

bash scripts/check-commit-message.sh "$1"
# {END}: commit-msg"""
    if start in text and end in text:
        before, rest = text.split(start, 1)
        _, after = rest.split(end, 1)
        text = (before.rstrip() + "\n" + after.lstrip()).rstrip() + "\n"
    text = text.rstrip() + "\n\n" + body + "\n"
    path.write_text(text, encoding="utf-8")
    path.chmod(path.stat().st_mode | 0o111)


def upsert_gitignore_block(text: str, kind: str, lines: list[str]) -> str:
    start = f"# {START}: {kind}"
    end = f"# {END}: {kind}"
    body = "\n".join([start, *lines, end])
    if start in text and end in text:
        before, rest = text.split(start, 1)
        _, after = rest.split(end, 1)
        return (before.rstrip() + "\n" + body + "\n" + after.lstrip()).rstrip() + "\n"

    if not text.rstrip():
        return body + "\n"
    return text.rstrip() + "\n\n" + body + "\n"


def add_gitignore_exceptions(repo: Path) -> None:
    path = repo / ".gitignore"
    text = path.read_text(encoding="utf-8") if path.exists() else ""
    local_defaults = [
        "/.claude/",
        "/.playwright-cli/",
        "/.tmp/",
        "/.harness/reuse-index/",
        "/.harness/runtime-runs/",
        "/backups/",
        "/tmp/",
        "/TODO/",
    ]
    additions = [
        "!.harness/",
        "!.harness/*.md",
        "!harness/",
        "!harness/**",
        "!concepts/",
        "!concepts/*.md",
        "!thinking/",
        "!thinking/*.md",
        "!practice/",
        "!practice/**",
        "!feedback/",
        "!feedback/*.md",
        "!works/",
        "!works/*.md",
        "!prompts/",
        "!prompts/*.md",
        "!references/",
        "!references/*.md",
        "!docs/reviews/general/",
        "!docs/reviews/general/*.md",
    ]
    text = upsert_gitignore_block(text, "local-artifacts", local_defaults)
    text = upsert_gitignore_block(text, "allowlist", additions)
    path.write_text(text, encoding="utf-8")


def todo_reminder_script() -> str:
    return r"""#!/usr/bin/env bash
set -euo pipefail

cwd="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
python3 ~/.agents/harness/todo/remind_todos.py --cwd "$cwd" "$@"
"""


def todo_governance_check_script() -> str:
    return r"""#!/usr/bin/env bash
set -euo pipefail

cwd="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
python3 ~/.agents/harness/todo/check_todo_governance.py --cwd "$cwd"
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

subject="$(sed -n '1p' "$message_file" | tr -d '\r')"

if [[ -z "$subject" ]]; then
  echo "[commit-msg] 提交信息不能为空" >&2
  exit 1
fi

if [[ "$subject" =~ ^Merge[[:space:]] ]] || [[ "$subject" =~ ^Revert[[:space:]] ]]; then
  exit 0
fi

pattern='^(feat|fix|refactor|docs|test|chore|build|ci|perf|style|revert)(\([^)]+\))?: .+$'
if [[ ! "$subject" =~ $pattern ]]; then
  cat >&2 <<'EOF'
[commit-msg] 提交信息格式不符合约束。
要求：英文类型 + 可选英文/模块 scope + 中文描述
示例：
  fix(frontend): 修正拓扑页导出按钮禁用态
  refactor(topology): 拆分画布工作区组件
  docs: 补齐提交信息约束说明
EOF
  exit 1
fi

description="${subject#*: }"
if ! python3 - "$description" <<'PY'
import re
import sys

description = sys.argv[1]
sys.exit(0 if re.search(r'[\u4e00-\u9fff]', description) else 1)
PY
then
  cat >&2 <<'EOF'
[commit-msg] 提交描述必须包含中文说明。
示例：
  fix(frontend): 修正拓扑页导出按钮禁用态
EOF
  exit 1
fi
"""


def strict_docs(project_name: str, profile: str) -> dict[str, str]:
    ctf = profile == "ctf-platform"
    project_line = (
        "学校教学场景 CTF 平台：学生刷题、教师分析、管理员治理，以及 Jeopardy / AWD / 靶机实例 / 题解复盘。"
        if ctf
        else f"{project_name} 项目。"
    )
    risk_line = (
        "重点关注 API 合同、后端 UTC/context 契约、前端路由命名空间、AWD 运行时、超大页面职责堆叠与验证闭环。"
        if ctf
        else "重点关注仓库事实源、规则漂移、验证闭环和 agent 可读性。"
    )
    return {
        "concepts/AGENTS.md": """# concepts/ — 概念笔记

严格参考 harness-engineering 仓库：每个核心概念一个文件，编号排序。

## 文件约定

- 文件名：`{编号}-{英文短名}.md`
- `00-overview.md` 是总览，先读这个。
- 每篇第一段必须说明概念是什么，随后写出它在本项目中的落点。

## 当前内容

- `00-overview.md`：Harness 总览
- `01-repo-as-source-of-truth.md`：仓库即记录系统
- `02-mechanical-enforcement.md`：机械化执行
- `03-feedback-loop.md`：反馈闭环
- `04-agent-readability.md`：智能体可读性
- `05-throughput-changes-merge.md`：吞吐量改变合并理念
- `06-harness-definition.md`：Harness 定义

## 下一步

读完概念后，去 `thinking/` 写项目化判断和质疑。
""",
        "concepts/00-overview.md": f"""# Harness Overview

Harness Engineering 在本仓库中的含义：人类维护约束、事实源、反馈与检查，AI agent 在这些边界内完成工程任务。

## 项目落点

{project_line}

本 harness 不替代业务架构，而是把 agent 需要读取和遵守的材料整理成可导航、可检查的结构。
""",
        "concepts/01-repo-as-source-of-truth.md": """# Repo As Source Of Truth

仓库即记录系统：不在仓库里的规则、决策、计划和反馈，对 agent 默认不存在。

## 本项目落点

- 长期架构事实进入 `docs/architecture/`。
- API 合同进入 `docs/contracts/`。
- 结构性实施进入 `docs/plan/impl-plan/`。
- Review 证据进入 `docs/reviews/`。
- 反复出现的问题进入 `feedback/` 或 `docs/improvements/`。
""",
        "concepts/02-mechanical-enforcement.md": """# Mechanical Enforcement

机械化执行：文档负责解释，脚本和 hook 负责阻止漂移。

## 本项目落点

- `scripts/check-consistency.sh` 检查 harness 目录、导航和计数声明。
- `.githooks/pre-commit` 在提交前执行一致性检查。
- 适合脚本化的规则应优先进入检查脚本，而不是只写进说明。
""",
        "concepts/03-feedback-loop.md": """# Feedback Loop

反馈闭环：失败、review finding 和重复问题必须回流为规则、prompt、计划或检查。

## 本项目落点

- `feedback/` 记录 harness 使用中的踩坑和修正。
- `docs/improvements/` 记录工程改进项。
- 当反馈已经固化为规则或脚本，回链到对应文件。
""",
        "concepts/04-agent-readability.md": """# Agent Readability

智能体可读性：目录、命名和入口要让 agent 知道下一步读什么，而不是要求它猜。

## 本项目落点

- 根 `AGENTS.md` 是入口地图。
- 每个 harness 子目录都有自己的 `AGENTS.md`。
- 长文档保留在事实源目录，harness 文件只做导航和约束摘要。
""",
        "concepts/05-throughput-changes-merge.md": """# Throughput Changes Merge

吞吐量改变合并理念：agent 交付速度高，等待和漂移成本上升，检查与回滚路径要更清楚。

## 本项目落点

- 每个结构性改动需要计划、验证和 review 记录。
- 小切片优先，避免把无关改动合并进一次交付。
- 检查脚本提供快速反馈，review 文档提供可追溯证据。
""",
        "concepts/06-harness-definition.md": """# Harness Definition

本仓库的 harness 是一组可版本化工件：导航、事实源、反馈记录、提示词、实践实验和机械化检查。

## 组件清单

- Guides：`AGENTS.md`、`concepts/`、`prompts/`
- Sensors：`scripts/check-consistency.sh`、hook、review 记录
- Memory：`feedback/`、`thinking/`、`references/`
- Practice：`practice/`
- Output：`works/`
""",
        "thinking/AGENTS.md": """# thinking/ — 独立思考

读完 `concepts/` 后，在这里写本项目对 Harness Engineering 的判断、质疑和取舍。

## 文件约定

- 文件名自由命名，建议用问题或论点。
- 结构：问题/论点 → 项目证据 → 判断 → 后续影响。

## 下一步

需要验证的判断进入 `practice/`；出现踩坑进入 `feedback/`。
""",
        "thinking/ctf-harness-boundary.md": f"""# CTF Harness Boundary

## 论点

本项目严格采用参考 harness 的顶层目录形态，但业务事实源仍保留在 CTF 原有代码和文档中。

## 项目证据

{risk_line}

## 判断

Harness 层负责让 agent 找到事实源和反馈，不把所有 CTF 架构内容复制进 harness 目录。
""",
        "practice/AGENTS.md": """# practice/ — 动手实践

严格参考 harness-engineering 仓库：每个实验一个子目录，包含 README 和必要的 AGENTS。

## 文件约定

- 每个实验一个子目录，如 `practice/01-ctf-harness-initialization/`。
- 实验说明写清楚目标、方法、验证命令和结果。

## 下一步

实践中的问题进入 `feedback/`。
""",
        "practice/01-ctf-harness-initialization/README.md": f"""# CTF Harness Initialization

## 目标

严格参考 `deusyu/harness-engineering`，为 `{project_name}` 建立顶层 harness 结构。

## 方法

- 创建 `concepts/ thinking/ practice/ feedback/ works/ prompts/ references/`。
- 为每个目录创建 `AGENTS.md`。
- 创建 `scripts/check-consistency.sh`。
- 接入 `.githooks/pre-commit`。

## 验证

```bash
bash scripts/check-consistency.sh
```
""",
        "practice/01-ctf-harness-initialization/AGENTS.md": """# practice/01-ctf-harness-initialization

本实验只记录 harness 初始化，不承载业务代码。

更新本实验时同步检查：

- 根 `AGENTS.md` 是否指向严格 harness 目录。
- `scripts/check-consistency.sh` 是否覆盖新增目录。
- `feedback/` 是否记录初始化过程中的偏差。
""",
        "feedback/AGENTS.md": """# feedback/ — 反馈记录

实践中的踩坑、修正、迭代心得。把失败变成可复用经验。

## 文件约定

- 文件名：`{日期}-{简述}.md`
- 结构：问题描述 → 原因分析 → 解决方案 → 收获
- 如果反馈导致 prompts、concepts、脚本或 AGENTS 更新，必须交叉链接。
""",
        "feedback/2026-05-05-strict-reference-harness.md": """# Strict Reference Harness

## 问题描述

第一版初始化偏向适配现有 CTF 文档体系，使用了 `docs/harness/` 作为项目内索引层。

## 原因分析

该做法符合“项目适配”，但不符合“严格参考 harness-engineering 仓库结构”的要求。

## 解决方案

改为创建参考仓库同构的顶层目录：`concepts/`、`thinking/`、`practice/`、`feedback/`、`works/`、`prompts/`、`references/`，并用 `scripts/check-consistency.sh` 检查这些目录和导航。

## 收获

当用户要求严格参考某个 harness 时，不能先把它折叠进现有 docs 体系；应优先保持参考项目的结构形态。
""",
        "works/AGENTS.md": """# works/ — 作品输出

可展示的成果：模板、教程、报告、可复用说明。

## 文件约定

- 每个作品一个文件或子目录。
- 作品应该可以独立理解，不依赖当前会话。
""",
        "works/ctf-harness-map.md": """# CTF Harness Map

这是 `ctf` 项目的 Harness Engineering 地图。

## 结构

- `concepts/`：概念与项目映射
- `thinking/`：判断与取舍
- `practice/`：实验和初始化记录
- `feedback/`：踩坑与修正
- `works/`：可展示输出
- `prompts/`：可复用提示词
- `references/`：外部资料索引
""",
        "prompts/AGENTS.md": """# prompts/ — 提示词积累

学习和实践中验证有效的提示词，按场景或工作流沉淀。只收录亲测有效的，不收录未验证的。

## 文件形态

- 单条 Prompt：用途、提示词正文、效果评价。
- Prompt 工作流：目标、步骤、链路、适用范围。
""",
        "prompts/ctf-harness-initialization.md": """# CTF Harness Initialization Prompt

## 用途

让 agent 严格参考 `deusyu/harness-engineering` 初始化项目级 harness。

## Prompt

请严格参考 `https://github.com/deusyu/harness-engineering` 的仓库结构，为当前项目创建顶层 `concepts/ thinking/ practice/ feedback/ works/ prompts/ references/`，每个目录都有 `AGENTS.md`，并创建 `scripts/check-consistency.sh` 和 hook 接入。不要把 harness 折叠进现有 `docs/` 目录。

## 效果评价

本次用于修正“过度适配现有项目结构”的偏差。
""",
        "references/AGENTS.md": """# references/ — 外部资源索引

相关文章、仓库、工具的统一索引。这里是指针，不是全文复制。

## 文件约定

- 按主题分文件。
- 每条记录包含链接、一句话说明、与 Harness Engineering 的关联。

## 当前入口

- `articles.md`
""",
        "references/articles.md": """# Articles

权威计数：3 篇。

## 脉络一：AI 时代的 Harness Engineering（2 篇）

### 1. OpenAI — Harness Engineering

- Link: https://openai.com/zh-Hans-CN/index/harness-engineering/
- 关联：原点，提出仓库事实源、机械化执行、agent 可读性等核心概念。

### 2. deusyu/harness-engineering

- Link: https://github.com/deusyu/harness-engineering
- 关联：本项目严格参考的仓库结构和一致性检查实践。

## 脉络二：反馈与标准编码（1 篇）

### 3. Fowler — Encoding Team Standards

- Link: https://martinfowler.com/articles/reduce-friction-ai/encoding-team-standards.html
- 关联：把团队标准显式化并写进 agent 可读取材料。
""",
    }


def ctf_current_docs(project_name: str, profile: str) -> dict[str, str]:
    project_line = (
        "学校教学场景 CTF 平台，后续项目可按自身业务替换本段。"
        if profile == "ctf-platform"
        else f"{project_name} 项目。"
    )
    return {
        ".harness/reuse-decisions/.gitkeep": "",
        ".harness/reuse-index/README.md": """# Local Reuse Index

This directory is user-local and gitignored.

- `index.yaml` is the top-level route map.
- Mirror source directories under this tree and place `README.md` files there as module-level and module-internal secondary indexes.
""",
        ".harness/reuse-index/index.yaml": """version: 1
entries: []
""",
        "harness/policies/reuse-first.yaml": """version: 1
protected_creation_types:
  - page
  - component
  - hook
  - service
  - handler
  - repository
  - port
  - job
  - mapper
  - readmodel
  - composition
  - store
  - api
  - form
  - table
  - modal
  - layout
  - schema
  - migration
required_before_creation:
  - search_existing_patterns
  - record_current_task_decision
  - prefer_extend_or_refactor_when_safe
""",
        "harness/policies/project-patterns.yaml": """version: 1
patterns:
  frontend_route_page:
    description: Reuse existing route page, component, composable, API wrapper, state, table, form, modal, and layout patterns before creating a parallel frontend implementation.
    search:
      - src/views
      - src/components
      - src/features
      - src/composables
      - src/api
      - src/stores
  backend_usecase:
    description: Reuse existing backend service/usecase, domain rule, contract, and port shape before creating a new application flow.
    search:
      - internal/module
      - internal/app
      - app
      - services
  backend_repository_port:
    description: Reuse or split existing consumer-side ports and infrastructure adapters before adding a parallel repository.
    search:
      - internal/module/*/ports
      - internal/module/*/infrastructure
      - repositories
      - ports
  backend_api_handler:
    description: Reuse existing handler/API, DTO/contract, auth/context extraction, validation, and error mapping patterns before adding transport glue.
    search:
      - internal/module/*/api
      - internal/module/*/contracts
      - handlers
      - dto
  backend_job_worker:
    description: Reuse existing context, lock, idempotency, timeout, retry, runtime wiring, and tests before adding a job, worker, runner, or scheduled flow.
    search:
      - internal/module/*/application
      - internal/pkg
      - workers
      - jobs
  backend_mapper_contract:
    description: Reuse generated mapper conventions, shared mapper helpers, and existing API/DTO contracts before adding hand-written structural copying.
    search:
      - internal/shared
      - internal/module/*/contracts
      - mappers
  backend_migration_schema:
    description: Reuse existing migration style, model conventions, timestamp policy, repository tests, and contract docs before adding schema changes.
    search:
      - migrations
      - internal/model
      - docs/contracts
""",
        "harness/templates/reuse-decision.md": """# Reuse Decision

## Task

TBD

## Search Scope

- Frontend: `src/`, `app/`, `components/`, `features/`, `views/`, `composables/`, `api/`, `stores/`
- Backend: `internal/`, `src/`, `app/`, `services/`, `handlers/`, `repositories/`, `ports/`, `jobs/`, `workers/`, `mappers/`, `readmodels/`, `migrations/`
- Local index: `.harness/reuse-index/index.yaml`, mirrored `README.md` files under `.harness/reuse-index/`
- Harness: `harness/policies/`

## Decision

- [ ] reuse_existing
- [ ] extend_existing
- [ ] refactor_existing
- [ ] create_new_with_reason

## Notes

This file stores current-task state only. Durable local reuse knowledge belongs in `.harness/reuse-index/`.
""",
        "harness/prompts/AGENTS.md": """# harness/prompts

Project-local validated prompt assets live here.

Do not store one-off current task notes here. Use `.harness/` for current-task state.
Do not keep one-off initialization prompts, historical migration prompts, or rules already moved into a global skill.
""",
        "harness/prompts/harness-router.md": f"""# Harness Router

Use this prompt before non-trivial work in `{project_name}`.

## Route

- `SIMPLE`: clearly local, reversible, and not worth recording.
- `HARNESS`: touches code, architecture, tests, documentation policy, reusable patterns, or repeated workflow.

## Project Context

{project_line}
""",
        "harness/checks/common.py": """from __future__ import annotations

from pathlib import Path


def repo_root() -> Path:
    return Path(__file__).resolve().parents[2]
""",
        "feedback/AGENTS.md": """# feedback

Record workflow mistakes, corrections, review lessons, and reusable process findings.

If a feedback item becomes cross-project guidance, sync it into the appropriate global skill instead of duplicating it here forever.
Each feedback record should include `## Sedimentation Status` or a project-local equivalent that names whether the lesson is already absorbed, project-only, awaiting skill sync, mechanized, or obsolete.
Once a lesson is fully captured by a global skill, project policy, or mechanical check, remove the long feedback body and rely on the index or Git history for traceability.
""",
    }


def ctf_current_check_script() -> str:
    return r"""#!/usr/bin/env bash
set -euo pipefail

fail=0

red() { printf '\033[31m%s\033[0m' "$1"; }
green() { printf '\033[32m%s\033[0m' "$1"; }

check_file() {
  if [[ -f "$1" ]]; then
    echo "  $(green PASS) — $1"
  else
    echo "  $(red FAIL) — missing $1"
    fail=1
  fi
}

check_dir() {
  if [[ -d "$1" ]]; then
    echo "  $(green PASS) — $1"
  else
    echo "  $(red FAIL) — missing $1"
    fail=1
  fi
}

check_contains() {
  local file="$1" pattern="$2" label="$3"
  if [[ ! -f "$file" ]]; then
    echo "  $(red FAIL) — $label: missing $file"
    fail=1
  elif grep -qE "$pattern" "$file"; then
    echo "  $(green PASS) — $label"
  else
    echo "  $(red FAIL) — $label"
    fail=1
  fi
}

echo "[C1] current-task and durable harness directories exist"
check_dir ".harness"
check_dir ".harness/reuse-decisions"
check_file ".harness/reuse-decisions/.gitkeep"
check_dir "harness"
check_dir "harness/policies"
check_dir "harness/templates"
check_dir "harness/prompts"
check_dir "harness/checks"
check_dir "feedback"

echo "[C2] local private reuse index is wired"
if grep -qx '/.harness/reuse-index/' ".gitignore"; then
  echo "  $(green PASS) — .gitignore reserves /.harness/reuse-index/"
else
  echo "  $(red FAIL) — .gitignore must ignore /.harness/reuse-index/"
  fail=1
fi
if [[ -d ".harness/reuse-index" ]]; then
  echo "  $(green PASS) — .harness/reuse-index exists"
else
  echo "  $(green PASS) — .harness/reuse-index is optional and currently absent"
fi

echo "[C3] project harness assets exist"
check_file "harness/policies/reuse-first.yaml"
check_file "harness/policies/project-patterns.yaml"
check_file "harness/templates/reuse-decision.md"
check_file "harness/prompts/AGENTS.md"
check_file "harness/prompts/harness-router.md"
check_file "harness/checks/common.py"
check_file "feedback/AGENTS.md"
check_file "docs/documentation-rules.md"
check_file "docs/README.md"
check_file "docs/improvements/README.md"
check_file "scripts/check-open-todos.sh"
check_file "scripts/check-todo-governance.sh"
for dir in requirements contracts spec design todo architecture plan operations reviews reports improvements refs; do
  check_dir "docs/$dir"
done
for dir in not-impl implemented agent-recorded rejected archived; do
  check_dir "docs/improvements/$dir"
done

echo "[C4] root navigation references current harness shape"
check_contains "AGENTS.md" '\.harness/' "AGENTS references current-task harness"
check_contains "AGENTS.md" '\.harness/reuse-index/' "AGENTS references local private reuse index"
check_contains "AGENTS.md" 'harness/policies/' "AGENTS references harness policies"
check_contains "AGENTS.md" 'harness/prompts/' "AGENTS references harness prompts"
check_contains "AGENTS.md" 'harness/checks/' "AGENTS references harness checks"
check_contains "AGENTS.md" 'feedback/' "AGENTS references feedback"
check_contains "AGENTS.md" 'docs/documentation-rules\.md' "AGENTS references documentation rules"
check_contains "AGENTS.md" 'docs/README\.md' "AGENTS references documentation index"
check_contains "AGENTS.md" 'scripts/check-open-todos\.sh' "AGENTS references todo reminder"
check_contains "docs/documentation-rules.md" 'Pre-Edit Reading Protocol' "documentation rules define pre-edit reading"
check_contains "docs/documentation-rules.md" 'New Path Registration' "documentation rules define new path registration"
check_contains "docs/documentation-rules.md" 'No Circular References' "documentation rules forbid circular references"

echo "[C4a] project agent entrypoints stay aligned"
if [[ -L "CLAUDE.md" ]]; then
  if [[ "$(readlink -f CLAUDE.md)" == "$(readlink -f AGENTS.md)" ]]; then
    echo "  $(green PASS) — CLAUDE.md points to AGENTS.md"
  else
    echo "  $(red FAIL) — CLAUDE.md does not resolve to AGENTS.md"
    fail=1
  fi
else
  echo "  $(red FAIL) — CLAUDE.md must be a symlink to AGENTS.md"
  fail=1
fi

echo "[C5] hooks and commit message guard are wired"
check_file "scripts/check-commit-message.sh"
check_file "scripts/check-test-workflow.sh"
if [[ -f ".githooks/pre-commit" ]]; then
  check_contains ".githooks/pre-commit" 'scripts/check-consistency\.sh' "pre-commit runs scripts/check-consistency.sh"
else
  echo "  $(red FAIL) — missing .githooks/pre-commit"
  fail=1
fi
if [[ -f ".githooks/commit-msg" ]]; then
  check_contains ".githooks/commit-msg" 'scripts/check-commit-message\.sh' "commit-msg runs scripts/check-commit-message.sh"
else
  echo "  $(red FAIL) — missing .githooks/commit-msg"
  fail=1
fi

echo "[C6] test workflow guard is surfaced to the operator"
if [[ -x "scripts/check-test-workflow.sh" ]]; then
  bash scripts/check-test-workflow.sh
else
  echo "  $(red FAIL) — scripts/check-test-workflow.sh is not executable"
  fail=1
fi

echo "[C7] open todos are surfaced to the operator"
if [[ -x "scripts/check-open-todos.sh" ]]; then
  bash scripts/check-open-todos.sh --quiet-if-empty
else
  echo "  $(red FAIL) — scripts/check-open-todos.sh is not executable"
  fail=1
fi

echo "[C8] todo governance stays consistent"
if [[ -x "scripts/check-todo-governance.sh" ]]; then
  bash scripts/check-todo-governance.sh
else
  echo "  $(red FAIL) — scripts/check-todo-governance.sh is not executable"
  fail=1
fi

if [[ "$fail" -eq 0 ]]; then
  echo "$(green '✓ all harness consistency checks passed')"
else
  echo "$(red '✗ harness consistency checks failed')"
fi

exit "$fail"
"""


def check_script() -> str:
    return r"""#!/usr/bin/env bash
set -euo pipefail

fail=0

red() { printf '\033[31m%s\033[0m' "$1"; }
green() { printf '\033[32m%s\033[0m' "$1"; }

check_file() {
  if [[ -f "$1" ]]; then
    echo "  $(green PASS) — $1"
  else
    echo "  $(red FAIL) — missing $1"
    fail=1
  fi
}

check_dir() {
  if [[ -d "$1" ]]; then
    echo "  $(green PASS) — $1"
  else
    echo "  $(red FAIL) — missing $1"
    fail=1
  fi
}

check_contains() {
  local file="$1" pattern="$2" label="$3"
  if [[ ! -f "$file" ]]; then
    echo "  $(red FAIL) — $label: missing $file"
    fail=1
  elif grep -qE "$pattern" "$file"; then
    echo "  $(green PASS) — $label"
  else
    echo "  $(red FAIL) — $label"
    fail=1
  fi
}

echo "[C1] strict harness directories exist"
for dir in concepts thinking practice feedback works prompts references; do
  check_dir "$dir"
  check_file "$dir/AGENTS.md"
done

echo "[C2] root navigation references strict harness"
check_contains "AGENTS.md" 'concepts/' "AGENTS references concepts"
check_contains "AGENTS.md" 'thinking/' "AGENTS references thinking"
check_contains "AGENTS.md" 'practice/' "AGENTS references practice"
check_contains "AGENTS.md" 'feedback/' "AGENTS references feedback"
check_contains "AGENTS.md" 'works/' "AGENTS references works"
check_contains "AGENTS.md" 'prompts/' "AGENTS references prompts"
check_contains "AGENTS.md" 'references/' "AGENTS references references"

echo "[C2a] project agent entrypoints stay aligned"
if [[ -L "CLAUDE.md" ]]; then
  if [[ "$(readlink -f CLAUDE.md)" == "$(readlink -f AGENTS.md)" ]]; then
    echo "  $(green PASS) — CLAUDE.md points to AGENTS.md"
  else
    echo "  $(red FAIL) — CLAUDE.md does not resolve to AGENTS.md"
    fail=1
  fi
else
  echo "  $(red FAIL) — CLAUDE.md must be a symlink to AGENTS.md"
  fail=1
fi

echo "[C3] articles.md numbering is contiguous 1..N"
nums=$(grep -nE '^### [0-9]+\.' references/articles.md | sed -E 's/^[0-9]+:### ([0-9]+)\..*/\1/' || true)
count=$(echo "$nums" | sed '/^$/d' | wc -l | tr -d ' ')
if [[ "$count" -eq 0 ]]; then
  echo "  $(red FAIL) — references/articles.md has no numbered entries"
  fail=1
else
  sorted=$(echo "$nums" | sort -n)
  expected=$(seq 1 "$count")
  if [[ "$sorted" = "$expected" ]]; then
    echo "  $(green PASS) — $count contiguous entries"
  else
    echo "  $(red FAIL) — article numbering is not contiguous"
    fail=1
  fi
fi

echo "[C4] article count claim matches numbered entries"
claim=$(grep -oE '权威计数：[0-9]+ 篇' references/articles.md | head -1 | grep -oE '[0-9]+' || true)
if [[ -z "$claim" || "$claim" != "$count" ]]; then
  echo "  $(red FAIL) — references/articles.md claims ${claim:-none}, actual $count"
  fail=1
else
  echo "  $(green PASS) — count claim $claim"
fi

echo "[C5] hooks and commit message guard are wired"
check_file "scripts/check-commit-message.sh"
check_file "scripts/check-test-workflow.sh"
if [[ -f ".githooks/pre-commit" ]]; then
  check_contains ".githooks/pre-commit" 'scripts/check-consistency\.sh' "pre-commit runs scripts/check-consistency.sh"
else
  echo "  $(red FAIL) — missing .githooks/pre-commit"
  fail=1
fi
if [[ -f ".githooks/commit-msg" ]]; then
  check_contains ".githooks/commit-msg" 'scripts/check-commit-message\.sh' "commit-msg runs scripts/check-commit-message.sh"
else
  echo "  $(red FAIL) — missing .githooks/commit-msg"
  fail=1
fi

echo "[C6] documentation architecture exists"
check_file "docs/documentation-rules.md"
check_file "docs/README.md"
check_file "scripts/check-open-todos.sh"
check_file "scripts/check-todo-governance.sh"
check_contains "docs/documentation-rules.md" 'No Circular References' "documentation rules forbid circular references"
check_contains "AGENTS.md" 'scripts/check-open-todos\.sh' "AGENTS references todo reminder"

echo "[C7] test workflow guard is surfaced to the operator"
if [[ -x "scripts/check-test-workflow.sh" ]]; then
  bash scripts/check-test-workflow.sh
else
  echo "  $(red FAIL) — scripts/check-test-workflow.sh is not executable"
  fail=1
fi

echo "[C8] open todos are surfaced to the operator"
if [[ -x "scripts/check-open-todos.sh" ]]; then
  bash scripts/check-open-todos.sh --quiet-if-empty
else
  echo "  $(red FAIL) — scripts/check-open-todos.sh is not executable"
  fail=1
fi

echo "[C9] todo governance stays consistent"
if [[ -x "scripts/check-todo-governance.sh" ]]; then
  bash scripts/check-todo-governance.sh
else
  echo "  $(red FAIL) — scripts/check-todo-governance.sh is not executable"
  fail=1
fi

if [[ "$fail" -eq 0 ]]; then
  echo "$(green '✓ all harness consistency checks passed')"
else
  echo "$(red '✗ harness consistency checks failed')"
fi

exit "$fail"
"""


def main() -> None:
    args = parse_args()
    repo = Path(args.repo).resolve()
    project_name = args.project_name or repo.name

    if args.mode == "strict-reference":
        ensure_documentation_scaffold(repo)
        for relative, content in strict_docs(project_name, args.profile).items():
            write(repo / relative, content)
        write(repo / "scripts/check-consistency.sh", check_script(), executable=True)
        write(repo / "scripts/check-test-workflow.sh", test_workflow_check_script(), executable=True)
        write(repo / "scripts/check-open-todos.sh", todo_reminder_script(), executable=True)
        write(repo / "scripts/check-todo-governance.sh", todo_governance_check_script(), executable=True)
        write(repo / "scripts/check-commit-message.sh", commit_message_check_script(), executable=True)

        insert_or_replace(
            repo / "AGENTS.md",
            "root-navigation",
            """## Harness Engineering 学习档案

严格参考 `deusyu/harness-engineering` 的顶层结构：

| 目录 | 内容 | 说明 |
|------|------|------|
| `concepts/` | 概念笔记 | Harness 核心概念与 CTF 项目映射 |
| `thinking/` | 独立思考 | 对项目 harness 边界和取舍的判断 |
| `practice/` | 动手实践 | 初始化和后续实验记录 |
| `feedback/` | 反馈记录 | 踩坑、修正和可复用经验 |
| `works/` | 作品输出 | 可展示模板、报告和说明 |
| `prompts/` | 提示词积累 | 已验证提示词和工作流 |
| `references/` | 外部资源 | 文章、仓库和工具索引 |

项目根保持 `CLAUDE.md -> AGENTS.md`，让 Claude / Codex 使用同一份入口规则。

机械化检查：`bash scripts/check-consistency.sh`。""",
        )
        insert_or_replace(
            repo / "AGENTS.md",
            "todo-reminder",
            """## Todo Reminder

开始新任务前，先运行 `bash scripts/check-open-todos.sh --quiet-if-empty`，先过一遍 `docs/todo/` 里的未完成事项；如果命中当前主题，首条回复先提醒。已完成但还没归档的 todo 也会在这里提示。""",
        )
        insert_or_replace(
            repo / "AGENTS.md",
            "test-workflow",
            """## Test Workflow

如果仓库存在自动化测试或明显测试面，先写或更新最小相关测试，再进入实现。

- Write or update the narrowest relevant tests first.
- After changing tests, run the smallest relevant test command that covers the touched surface.
- After the test command, run the relevant script check such as `bash scripts/check-test-workflow.sh` or `bash scripts/check-consistency.sh` before claiming completion.
- 如果当前仓库已经有 `scripts/check-consistency.sh`、git hooks 或 CI guardrail，测试相关脚本检查必须接入这些实际检查链路，不能只停留在提示词里。""",
        )
        insert_or_replace(
            repo / "README.md",
            "readme-harness",
            """## Harness Engineering

本项目按 `deusyu/harness-engineering` 建立顶层 harness 结构：

- `concepts/`：核心概念
- `thinking/`：独立思考
- `practice/`：实践记录
- `feedback/`：反馈闭环
- `works/`：作品输出
- `prompts/`：提示词积累
- `references/`：外部资料

一致性检查：

```bash
bash scripts/check-consistency.sh
```""",
        )
        hook_docs = """## Harness 检查

- `pre-commit`：运行 `scripts/check-consistency.sh`，其中会继续执行 `scripts/check-test-workflow.sh`，检查严格参考 harness 的顶层目录、导航、资料计数和测试工作流约束。
- `commit-msg`：运行 `scripts/check-commit-message.sh`，要求提交信息使用英文类型关键字与中文描述，例如 `fix(harness): 补齐提交信息校验`。
- 原有 API 合同同步逻辑继续保留。"""
        message = "Initialized strict-reference harness"
    else:
        ensure_documentation_scaffold(repo)
        for relative, content in ctf_current_docs(project_name, args.profile).items():
            write(repo / relative, content)
        write(repo / "scripts/check-consistency.sh", ctf_current_check_script(), executable=True)
        write(repo / "scripts/check-test-workflow.sh", test_workflow_check_script(), executable=True)
        write(repo / "scripts/check-open-todos.sh", todo_reminder_script(), executable=True)
        write(repo / "scripts/check-todo-governance.sh", todo_governance_check_script(), executable=True)
        write(repo / "scripts/check-commit-message.sh", commit_message_check_script(), executable=True)

        insert_or_replace(
            repo / "AGENTS.md",
            "root-navigation",
            """## Harness Engineering

当前默认采用 CTF 探索版 harness 形态，并保留 `deusyu/harness-engineering` 的核心原则作为重要参考。

| 路径 | 内容 | 说明 |
|------|------|------|
| `.harness/` | 当前任务状态 | 只保存短期执行证据和当前 reuse 决策 |
| `.harness/reuse-index/` | 本地私有索引 | 用户自用的长期复用线索，默认 gitignore，`index.yaml` + 镜像 `README.md` |
| `harness/policies/` | 项目策略 | 可被检查脚本读取的本地规则 |
| `harness/templates/` | 模板 | 当前项目重复使用的决策或记录模板 |
| `harness/prompts/` | Prompt 资产 | 已验证的项目本地 prompt 和工作流 |
| `harness/checks/` | 检查脚本 | 机械化一致性和规则检查 |
| `feedback/` | 反馈记录 | 踩坑、修正和可复用流程经验 |
| `docs/documentation-rules.md` | 文档规范 | 改文档前置读取与新增路径登记 |
| `docs/README.md` | 文档索引 | 当前事实源地图和文档阅读顺序 |

项目根保持 `CLAUDE.md -> AGENTS.md`，让 Claude / Codex 使用同一份入口规则。

机械化检查：`bash scripts/check-consistency.sh`。

开发过程中，如果某个模块第一次形成稳定复用模式，主动补 `.harness/reuse-index/<source-path>/README.md`；如果模块内部也已经分出稳定层次，再继续补该子路径下的镜像 `README.md`。这是本地提醒，不作为 pre-commit 阻塞项。

如果用户明确要求严格参考 `deusyu/harness-engineering` 的目录形态，再使用 strict reference 模式。""",
        )
        insert_or_replace(
            repo / "AGENTS.md",
            "todo-reminder",
            """## Todo Reminder

开始新任务前，先运行 `bash scripts/check-open-todos.sh --quiet-if-empty`，先过一遍 `docs/todo/` 里的未完成事项；如果命中当前主题，首条回复先提醒。已完成但还没归档的 todo 也会在这里提示。""",
        )
        insert_or_replace(
            repo / "AGENTS.md",
            "test-workflow",
            """## Test Workflow

如果仓库存在自动化测试或明显测试面，先写或更新最小相关测试，再进入实现。

- Write or update the narrowest relevant tests first.
- After changing tests, run the smallest relevant test command that covers the touched surface.
- After the test command, run the relevant script check such as `bash scripts/check-test-workflow.sh` or `bash scripts/check-consistency.sh` before claiming completion.
- 如果当前仓库已经有 `scripts/check-consistency.sh`、git hooks 或 CI guardrail，测试相关脚本检查必须接入这些实际检查链路，不能只停留在提示词里。""",
        )
        insert_or_replace(
            repo / "README.md",
            "readme-harness",
            """## Harness Engineering

本项目采用 CTF 探索版 harness 形态：

- `.harness/`：当前任务状态
- `.harness/reuse-index/`：用户本地私有的长期复用索引，默认 gitignore
- `harness/policies/`：项目策略
- `harness/templates/`：模板
- `harness/prompts/`：Prompt 资产
- `harness/checks/`：检查脚本
- `feedback/`：反馈记录
- `docs/documentation-rules.md`：文档修改前置读取与新增路径登记
- `docs/README.md`：文档索引和当前事实源地图

一致性检查：

```bash
bash scripts/check-consistency.sh
```""",
        )
        hook_docs = """## Harness 检查

- `pre-commit`：运行 `scripts/check-consistency.sh`，其中会继续执行 `scripts/check-test-workflow.sh`，检查 CTF 探索版 harness 的目录、导航、本地私有 reuse 索引归属和测试工作流约束。
- `commit-msg`：运行 `scripts/check-commit-message.sh`，要求提交信息使用英文类型关键字与中文描述，例如 `fix(harness): 补齐提交信息校验`。
- 原有项目 hook 逻辑继续保留。"""
        message = "Initialized CTF-current harness"

    insert_hook(repo / ".githooks/pre-commit")
    insert_commit_msg_hook(repo / ".githooks/commit-msg")
    insert_or_replace(
        repo / ".githooks/README.md",
        "hook-docs",
        hook_docs,
    )
    ensure_claude_symlink(repo)
    run_agent_entrypoint_check(repo)
    add_gitignore_exceptions(repo)
    print(f"{message} for {project_name} at {repo}")


if __name__ == "__main__":
    main()
