#!/usr/bin/env python3
"""Harness initializer content trigger rate templates."""

from __future__ import annotations


def test_trigger_rate_script() -> str:
    """Generate test-trigger-rate.sh wrapper script."""
    return r"""#!/usr/bin/env bash
#
# Test skill description trigger rates
#
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cwd="$(cd "$script_dir/../.." && pwd)"

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
