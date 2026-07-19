#!/usr/bin/env python3
"""Task phrase variations for trigger-rate checks."""

from __future__ import annotations

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
    "Review": [
        "帮我 review 一下",
        "审查代码",
        "看看这个代码有没有问题",
        "做个 code review",
        "检查一下代码质量",
        "评审实施计划",
        "审查架构设计文档",
        "检查迁移和 rollout 方案",
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
