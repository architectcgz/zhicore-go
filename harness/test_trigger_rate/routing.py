#!/usr/bin/env python3
"""Quick Routing parser helpers."""

from __future__ import annotations

import re
from pathlib import Path
from typing import List, Tuple


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
    否则会把 `backend-engineer` 拆成 `engineer`。
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
