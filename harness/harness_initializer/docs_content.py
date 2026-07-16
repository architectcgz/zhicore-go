#!/usr/bin/env python3
"""Harness initializer documentation templates."""

from __future__ import annotations


def strict_docs(project_name: str, profile: str) -> dict[str, str]:
    project_line = f"{project_name} 项目。"
    risk_line = "重点关注仓库事实源、规则漂移、验证闭环和 agent 可读性。"
    return {
        "concepts/AGENTS.md": """# concepts/ — AGENTS 补充说明

本目录作为项目根 `AGENTS.md` 的补充层使用。根 `AGENTS.md` 负责项目说明、入口导航和索引；这里补充长期稳定的概念、原则和 harness 定义。

## 文件约定

- 文件名：`{编号}-{英文短名}.md`
- 先读根 `AGENTS.md`，再读 `00-overview.md`。
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

如果根 `AGENTS.md` 已经足够回答当前问题，就不要为了补目录而重复写一份；只有长期稳定、会反复影响 agent 判断的概念才进入这里。
""",
        "concepts/00-overview.md": f"""# Harness Overview

Harness Engineering 在本仓库中的含义：人类维护约束、事实源、反馈与检查，AI agent 在这些边界内完成工程任务。本目录作为项目根 `AGENTS.md` 的补充，只承接长期稳定的概念层说明。

## 项目落点

{project_line}

本 harness 不替代业务架构，而是把 agent 需要读取和遵守的材料整理成可导航、可检查的结构。
""",
        "concepts/01-repo-as-source-of-truth.md": """# Repo As Source Of Truth

仓库即记录系统：不在仓库里的规则、决策、计划和反馈，对 agent 默认不存在。

## 本项目落点

- 长期架构事实进入 `.arccgz-harness/docs/architecture/`。
- API 合同进入 `.arccgz-harness/docs/contracts/`。
- 结构性实施进入 `.arccgz-harness/docs/plan/`。
- Review 证据进入 `.arccgz-harness/docs/reviews/`。
- 反复出现的问题进入 `.arccgz-harness/feedback/` 或 `.arccgz-harness/docs/improvements/`。
""",
        "concepts/02-mechanical-enforcement.md": """# Mechanical Enforcement

机械化执行：文档负责解释，脚本和 hook 负责阻止漂移。

## 本项目落点

- `.arccgz-harness/scripts/check-harness-consistency.sh` 检查 harness 目录、导航和计数声明。
- `.githooks/pre-commit` 在提交前执行一致性检查。
- 适合脚本化的规则应优先进入检查脚本，而不是只写进说明。
""",
        "concepts/03-feedback-loop.md": """# Feedback Loop

反馈闭环：失败、review finding 和重复问题必须回流为规则、prompt、计划或检查。

## 本项目落点

- `.arccgz-harness/feedback/` 记录 harness 使用中的踩坑和修正。
- `.arccgz-harness/docs/improvements/` 记录工程改进项。
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
- Sensors：`.arccgz-harness/scripts/check-harness-consistency.sh`、hook、review 记录
- Memory：`.arccgz-harness/feedback/`、`thinking/`、`references/`
- Practice：`practice/`
- Output：`works/`
""",
        "thinking/AGENTS.md": """# thinking/ — 独立思考

读完根 `AGENTS.md` 和 `concepts/` 后，在这里写本项目对 Harness Engineering 的判断、质疑和取舍。

## 文件约定

- 文件名自由命名，建议用问题或论点。
- 结构：问题/论点 → 项目证据 → 判断 → 后续影响。

## 下一步

需要验证的判断进入 `practice/`；出现踩坑进入 `.arccgz-harness/feedback/`。
""",
        "thinking/harness-boundary.md": f"""# Harness Boundary

## 论点

本项目严格采用参考 harness 的顶层目录形态，但业务事实源仍保留在项目原有代码和文档中。

## 项目证据

{risk_line}

## 判断

Harness 层负责让 agent 找到事实源和反馈，不把所有业务架构内容复制进 harness 目录。
""",
        "practice/AGENTS.md": """# practice/ — 动手实践

严格参考 harness-engineering 仓库：每个实验一个子目录，包含 README 和必要的 AGENTS。

## 文件约定

- 每个实验一个子目录，如 `practice/01-harness-initialization/`。
- 实验说明写清楚目标、方法、验证命令和结果。

## 下一步

实践中的问题进入 `.arccgz-harness/feedback/`。
""",
        "practice/01-harness-initialization/README.md": f"""# Harness Initialization

## 目标

严格参考 `deusyu/harness-engineering`，为 `{project_name}` 建立顶层 harness 结构。

## 方法

- 创建 `concepts/ thinking/ practice/ .arccgz-harness/feedback/ works/ prompts/ references/`。
- 为每个目录创建 `AGENTS.md`。
- 创建 `.arccgz-harness/scripts/check-harness-consistency.sh`。
- 接入 `.githooks/pre-commit`。

## 验证

```bash
bash .arccgz-harness/scripts/check-harness-consistency.sh
```
""",
        "practice/01-harness-initialization/AGENTS.md": """# practice/01-harness-initialization

本实验只记录 harness 初始化，不承载业务代码。

更新本实验时同步检查：

- 根 `AGENTS.md` 是否指向严格 harness 目录。
- `.arccgz-harness/scripts/check-harness-consistency.sh` 是否覆盖新增目录。
- `.arccgz-harness/feedback/` 是否记录初始化过程中的偏差。
""",
        "feedback/AGENTS.md": """# .arccgz-harness/feedback/ — 反馈记录

实践中的踩坑、修正、迭代心得。把失败变成可复用经验。

## 文件约定

- 文件名：`{日期}-{简述}.md`
- 结构：问题描述 → 原因分析 → 解决方案 → 收获
- 如果反馈导致 prompts、concepts、脚本或 AGENTS 更新，必须交叉链接。
""",
        "feedback/2026-05-05-strict-reference-harness.md": """# Strict Reference Harness

## 问题描述

第一版初始化偏向适配现有项目文档体系，使用了 `docs/harness/` 作为项目内索引层。

## 原因分析

该做法符合“项目适配”，但不符合“严格参考 harness-engineering 仓库结构”的要求。

## 解决方案

改为创建参考仓库同构的顶层目录：`concepts/`、`thinking/`、`practice/`、`.arccgz-harness/feedback/`、`works/`、`prompts/`、`references/`，并用 `.arccgz-harness/scripts/check-harness-consistency.sh` 检查这些目录和导航。

## 收获

当用户要求严格参考某个 harness 时，不能先把它折叠进现有 docs 体系；应优先保持参考项目的结构形态。
""",
        "works/AGENTS.md": """# works/ — 作品输出

可展示的成果：模板、教程、报告、可复用说明。

## 文件约定

- 每个作品一个文件或子目录。
- 作品应该可以独立理解，不依赖当前会话。
""",
        "works/harness-map.md": """# Harness Map

这是项目的 Harness Engineering 地图。

## 结构

- `concepts/`：补充项目 `AGENTS.md` 的长期概念与原则
- `thinking/`：判断与取舍
- `practice/`：实验和初始化记录
- `.arccgz-harness/feedback/`：踩坑与修正
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
        "prompts/harness-initialization.md": """# Harness Initialization Prompt

## 用途

让 agent 严格参考 `deusyu/harness-engineering` 初始化项目级 harness。

## Prompt

请严格参考 `https://github.com/deusyu/harness-engineering` 的仓库结构，为当前项目创建顶层 `concepts/ thinking/ practice/ .arccgz-harness/feedback/ works/ prompts/ references/`，每个目录都有 `AGENTS.md`，并创建 `.arccgz-harness/scripts/check-harness-consistency.sh` 和 hook 接入。不要把 harness 折叠进现有 `docs/` 目录。

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


def current_docs(project_name: str, profile: str) -> dict[str, str]:
    project_line = f"{project_name} 项目。"
    return {
        "state/reuse-decisions/.gitkeep": "",
        "state/reuse-index/README.md": """# Local Reuse Index

This directory is user-local and gitignored.

- `index.yaml` is the top-level route map.
- Mirror source directories under this tree and place `README.md` files there as module-level and module-internal secondary indexes.
""",
        "state/reuse-index/index.yaml": """version: 1
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
- Local index: `.arccgz-harness/state/reuse-index/index.yaml`, mirrored `README.md` files under `.arccgz-harness/state/reuse-index/`
- Harness: `harness/policies/`

## Decision

- [ ] reuse_existing
- [ ] extend_existing
- [ ] refactor_existing
- [ ] create_new_with_reason

## Notes

This file stores current-task state only. Durable local reuse knowledge belongs in `.arccgz-harness/state/reuse-index/`.
""",
        "harness/prompts/AGENTS.md": """# harness/prompts

Project prompt entrypoints live here.

Shared reusable prompt bodies can live under `~/.agents/harness/prompts/`.
Keep this directory for stable in-repo entrypoints, local parameters, and project-specific prompt supplements.
Do not store one-off current task notes here. Use `.arccgz-harness/state/` for current-task state.
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
