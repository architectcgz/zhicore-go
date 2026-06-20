#!/usr/bin/env python3
"""
Skill Description Trigger Rate Tester

测试 skill description 的触发率，从 AGENTS.md 的 Quick Routing 表中提取任务类型，
生成用户可能的表达方式，测试对应 skill 的 description 是否能被触发。
"""

import argparse
import re
from pathlib import Path
from typing import Dict, List, Tuple


# 任务类型到用户表达方式的映射
TASK_VARIATIONS = {
    "Backend feature (API/Service/Repository)": [
        "加个 API",
        "实现后端接口",
        "新增一个服务",
        "写个 REST API",
        "添加数据库操作",
        "实现后端功能",
        "创建新的 API 端点",
        "需要一个后端接口",
    ],
    "Frontend feature (Page/Component)": [
        "做个页面",
        "添加前端组件",
        "实现前端功能",
        "创建一个 Vue 组件",
        "新增一个页面",
        "写个前端页面",
        "做个 UI 组件",
    ],
    "Code review": [
        "帮我 review 一下",
        "审查代码",
        "看看这个代码有没有问题",
        "做个 code review",
        "检查一下代码质量",
    ],
    "Bug fix (Backend)": [
        "修个后端 bug",
        "后端有个问题",
        "API 报错了",
        "修复后端错误",
        "后端功能不正常",
    ],
    "Bug fix (Frontend)": [
        "修个前端 bug",
        "页面有个问题",
        "前端报错了",
        "修复前端错误",
        "页面显示不正常",
    ],
    "Add/Edit test": [
        "写个测试",
        "添加单元测试",
        "补充测试用例",
        "修改测试",
        "写测试代码",
    ],
    "Architecture change": [
        "重构架构",
        "调整系统设计",
        "改变模块结构",
        "架构优化",
        "重新设计架构",
    ],
    "Documentation update": [
        "更新文档",
        "写文档",
        "补充说明",
        "修改 README",
        "完善文档",
    ],
    "New non-trivial task": [
        "开始新任务",
        "实现一个复杂功能",
        "做个大功能",
        "新需求",
    ],
}


def parse_quick_routing_table(agents_md_path: Path) -> List[Tuple[str, str, str]]:
    """
    从 AGENTS.md 中解析 Quick Routing 表。

    返回: [(task_type, required_reads, workflow_skill), ...]
    """
    content = agents_md_path.read_text(encoding="utf-8")

    # 查找 Quick Routing 表
    match = re.search(
        r"## Quick Routing.*?\n\n(.*?)\n\n## ",
        content,
        re.DOTALL
    )

    if not match:
        return []

    table_content = match.group(1)

    # 解析表格行
    rows = []
    for line in table_content.split("\n"):
        if line.startswith("|") and not line.startswith("|-"):
            # 跳过表头
            if "Task type" in line:
                continue

            parts = [p.strip() for p in line.split("|")[1:-1]]
            if len(parts) >= 3:
                rows.append((parts[0], parts[1], parts[2]))

    return rows


def extract_skill_names(workflow_skill: str) -> List[str]:
    """
    从 "Workflow / Skill" 列中提取 skill 名称。

    例如：
    - "`backend-engineer` skill + `code-workflow`" → ["backend-engineer"]
    - "`systematic-debugging` then `backend-engineer`" → ["systematic-debugging", "backend-engineer"]
    - 含 `<!-- FILL -->` 的占位行 → []（交给调用方豁免）

    注意：skill 名规范写在反引号里，优先整体提取反引号内容，绝不按连字符 `-` 拆分，
    否则会把 `code-reviewer` 拆成 `reviewer`。
    """
    # 占位行不含真实 skill
    if "FILL" in workflow_skill:
        return []

    # 优先提取所有反引号包裹的名称（连字符名也完整保留），排除编排层 code-workflow
    names = [n.strip() for n in re.findall(r"`([^`]+)`", workflow_skill)]
    skills = [n for n in names if n and not n.startswith("code-workflow")]
    if skills:
        return skills

    # 回退：无反引号时按编排分隔符切分（→ / then / + / , / /），不含连字符
    text = workflow_skill.replace(" skill", "")
    for part in re.split(r"→|\bthen\b|\+|,|/", text):
        part = part.strip()
        m = re.match(r"^([a-z][a-z0-9\-]+)$", part)
        if m and not part.startswith("code-workflow"):
            skills.append(m.group(1))

    return skills

    return skills


def find_skill_description(skill_name: str) -> str:
    """
    查找 skill 的 description。

    优先从 ~/.agents/skills/ 查找，其次从 ~/.codex/skills/ 查找。
    """
    skill_dirs = [
        Path.home() / ".agents" / "skills" / skill_name,
        Path.home() / ".codex" / "skills" / skill_name,
    ]

    for skill_dir in skill_dirs:
        skill_md = skill_dir / "SKILL.md"
        if skill_md.exists():
            content = skill_md.read_text(encoding="utf-8")
            # 提取 frontmatter 中的 description
            match = re.search(r"description:\s*(.+)", content)
            if match:
                return match.group(1).strip()

    return ""


def is_project_local_skill(skill_name: str, repo: Path) -> bool:
    """skill 是否是当前项目自有（位于 <repo>/.agents/skills/<name>/SKILL.md）。

    test-trigger 只应对项目自有 skill 的 description 质量把关；指向全局 skill 的标准行
    只报告、不作为退出码依据，避免刚 init 的项目因全局 skill 描述风格而必然失败。
    """
    return (repo / ".agents" / "skills" / skill_name / "SKILL.md").exists()


def test_trigger(user_input: str, skill_description: str) -> bool:
    """
    测试用户输入是否能触发 skill description。

    简化版：检查 description 中的关键词是否在用户输入中。
    实际应该使用更复杂的语义匹配。
    """
    if not skill_description:
        return False

    # 提取 description 中的关键词
    keywords = re.findall(r"\b[a-zA-Z一-龥]{2,}\b", skill_description.lower())
    user_lower = user_input.lower()

    # 如果用户输入包含任意关键词，视为匹配
    for keyword in keywords:
        if keyword in user_lower or keyword.replace("-", "") in user_lower:
            return True

    return False


def generate_report(results: Dict[str, Dict]) -> str:
    """生成触发率报告。"""
    report = []
    report.append("=" * 70)
    report.append("Skill Description Trigger Rate Report")
    report.append("=" * 70)
    report.append("")

    total_tasks = len(results)
    total_tests = sum(r["total"] for r in results.values())
    total_hits = sum(r["hits"] for r in results.values())

    for task_type, data in results.items():
        rate = (data["hits"] / data["total"] * 100) if data["total"] > 0 else 0
        status = "✓" if rate >= 80 else "✗"

        report.append(f"{status} {task_type}")
        report.append(f"   Skill: {', '.join(data['skills'])}")
        report.append(f"   Trigger Rate: {data['hits']}/{data['total']} ({rate:.1f}%)")

        if rate < 80:
            report.append(f"   ⚠ Low trigger rate! Recommendation:")
            report.append(f"      - Review skill description")
            report.append(f"      - Add more keywords")
            report.append(f"      - Consider user's natural language")

        report.append("")

    report.append("-" * 70)
    overall_rate = (total_hits / total_tests * 100) if total_tests > 0 else 0
    report.append(f"Overall Trigger Rate: {total_hits}/{total_tests} ({overall_rate:.1f}%)")
    report.append("=" * 70)

    return "\n".join(report)


def main():
    parser = argparse.ArgumentParser(
        description="Test skill description trigger rates"
    )
    parser.add_argument(
        "--agents-md",
        type=Path,
        default=Path.cwd() / "AGENTS.md",
        help="Path to AGENTS.md (default: ./AGENTS.md)"
    )
    parser.add_argument(
        "--verbose",
        action="store_true",
        help="Show detailed test results"
    )

    args = parser.parse_args()

    if not args.agents_md.exists():
        print(f"Error: {args.agents_md} not found")
        return 1

    # 解析 Quick Routing 表
    routing_table = parse_quick_routing_table(args.agents_md)

    if not routing_table:
        print("Error: Could not find Quick Routing table in AGENTS.md")
        return 1

    print(f"Found {len(routing_table)} task types in Quick Routing table\n")

    repo = args.agents_md.resolve().parent

    # 测试每个任务类型
    results = {}
    exempt = 0

    for task_type, required_reads, workflow_skill in routing_table:
        # 跳过 "Other" 类型
        if "Other" in task_type:
            continue

        # 占位/FILL 行豁免：项目尚未补充真实文件与 skill，不计入触发率
        if "FILL" in workflow_skill or "FILL" in required_reads:
            exempt += 1
            if args.verbose:
                print(f"– 豁免占位行: {task_type}")
            continue

        # 提取 skill 名称
        skills = extract_skill_names(workflow_skill)

        if not skills:
            if args.verbose:
                print(f"⚠ No skills found for: {task_type}")
            continue

        # 获取用户表达方式
        variations = TASK_VARIATIONS.get(task_type, [])

        if not variations:
            if args.verbose:
                print(f"⚠ No variations defined for: {task_type}")
            continue

        # 测试触发率
        hits = 0
        total = len(variations)

        for variation in variations:
            triggered = False
            for skill_name in skills:
                description = find_skill_description(skill_name)
                if test_trigger(variation, description):
                    triggered = True
                    break

            if triggered:
                hits += 1
            elif args.verbose:
                print(f"  ✗ Not triggered: \"{variation}\"")

        results[task_type] = {
            "skills": skills,
            "hits": hits,
            "total": total,
            "local": any(is_project_local_skill(s, repo) for s in skills),
        }

        if args.verbose:
            print(f"✓ Tested: {task_type} ({hits}/{total})\n")

    # 生成报告
    report = generate_report(results)
    print(report)

    # 返回状态码：只对项目自有 skill 把关；指向全局 skill 的标准行只报告、不计入退出码
    local_rows = {k: v for k, v in results.items() if v.get("local")}
    global_rows = [k for k, v in results.items() if not v.get("local")]
    gated_total = sum(v["total"] for v in local_rows.values())
    gated_hits = sum(v["hits"] for v in local_rows.values())
    gated_rate = (gated_hits / gated_total * 100) if gated_total > 0 else 0

    if exempt:
        print(f"\n（已豁免 {exempt} 个占位/FILL 行，未计入触发率；补充真实 skill 后再测）")
    if global_rows:
        print(f"（{len(global_rows)} 行指向全局 skill，只报告不作为退出码依据：{', '.join(global_rows)}）")

    # 没有任何项目自有 skill 可测（如刚 init 的占位薄壳）→ 不判失败
    if gated_total == 0:
        print("没有指向项目自有 skill 的已填充行，跳过触发率门禁判定。")
        return 0

    # 仅当项目自有 skill 的触发率 < 80% 才返回 1
    return 0 if gated_rate >= 80 else 1


if __name__ == "__main__":
    exit(main())
