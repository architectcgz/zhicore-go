---
name: data-labeling-task-evaluator
description: Use when creating or grading data-labeling coding tasks for model evaluation, especially when the user needs repoId, annotator, submission date, Trae sessionId, user prompt, GitHub repo URL, commit id, task type, business domain, modification scope, difficulty, satisfaction, or dissatisfaction reason.
---

# Data Labeling Task Evaluator

## Overview

Use this skill to run one coding-task annotation end to end: design a realistic task, prepare or inspect a GitHub repository, write the prompt for the model, wait for the submitted result, and fill the annotation fields from evidence.

## Operating Mode

Work in two phases:

1. **Task setup**: create the repo/task/prompt and return the fields that are known before the model works.
2. **Result grading**: inspect the model's final repo/session output and fill type, domain, scope, difficulty, satisfaction, and dissatisfaction reason.

If the user only asks for one phase, perform only that phase and state which fields still need outside input.

## Required Inputs

For task setup, collect or infer:

- target stack or repo seed, if any
- evaluation goal, such as bugfix, feature, refactor, test, UI, backend, docs, or security
- expected difficulty and allowed time
- whether the task should be deterministic, open-ended, or adversarial
- GitHub repository destination, if the user expects a remote repo

For grading, collect:

- repoId
- annotator / 做题人
- submission date / 提交日期
- Trae sessionId
- GitHub repo URL
- GitHub commit id, branch, or submitted diff
- the exact user prompt given to the model
- any final answer, logs, tests, screenshots, or reviewer notes

Do not invent IDs, dates, session IDs, GitHub URLs, or commit IDs. Mark unknown fields as `待补充`.

## Task Setup Workflow

1. Inspect the candidate repo before writing the task. Read README, build files, tests, package manifests, and project rules.
2. Choose a task that is realistic for the repo and has an observable success condition. Prefer tasks with verifiable behavior over vague polish.
3. Prepare the repository only as much as needed:
   - If creating a new repo, initialize a minimal runnable project, include README instructions, and commit the baseline.
   - If a GitHub remote repo is needed, prefer the local `gh` CLI: run `gh auth status`, confirm or infer the repo owner/name and visibility, then create with `gh repo create ... --source . --remote origin --push` after the baseline commit exists.
   - If using an existing repo, record the baseline commit and avoid unrelated changes.
4. Write the user prompt as the exact instruction the model should receive. Include expected behavior, relevant files, constraints, and verification commands. Avoid giving implementation hints that would make the evaluation trivial.
5. Return a setup summary with the prompt and known annotation fields.

### GitHub Repository Creation

- Use `gh` when the user asks to create/publish a GitHub repo or when the annotation requires a GitHub repo URL.
- Do not create a remote repository until the local baseline is committed and the repo name plus visibility are known.
- If the user has not specified visibility, default to `private` for evaluation repos unless project rules require public.
- If `gh auth status` fails, stop and report that GitHub authentication is needed; do not fall back to embedding tokens or asking for secrets.
- After pushing, record `GitHub repo URL` from `gh repo view --json url -q .url` and record the baseline `GitHub commit id` with `git rev-parse HEAD`.

### Prompt Quality Rules

- Make the prompt self-contained enough that a model can work from it without hidden context.
- Include one clear acceptance criterion and one verification command when the repo supports it.
- Keep the request faithful to the intended difficulty; do not stack unrelated requirements.
- State constraints that matter, such as "do not change public API", "add tests", or "preserve existing layout".
- Avoid leaking the expected solution, exact file edits, or grading rubric unless the task type requires it.

## Grading Workflow

1. Confirm the submitted commit/diff is the result being graded. If the commit is missing, grade from the available workspace and mark commit id as `待补充`.
2. Compare the user prompt, repo baseline, and submitted changes. Use `git diff`, `git show`, tests, and relevant file reads.
3. Run the smallest meaningful verification when possible. If verification cannot run, record why.
4. Classify the task using the tables below.
5. Decide satisfaction from user intent and evidence, not from whether code merely changed.
6. If unsatisfied or partially satisfied, write a concrete reason tied to missing behavior, regression, failed verification, scope drift, low quality, or incomplete evidence.

## Classification Tables

Use one primary value. Add a secondary note only if it materially clarifies the task.

### Task Type

- `功能开发`: new user-visible or API behavior
- `Bug 修复`: incorrect behavior corrected
- `重构优化`: internal structure, performance, maintainability, or cleanup without intended behavior change
- `测试补充`: tests, fixtures, CI checks, or verification-only work
- `前端/UI`: visual layout, interaction, component, state, or responsive behavior
- `后端/API`: server routes, services, persistence, auth, jobs, queues, or integrations
- `文档/配置`: README, docs, config, scripts, dependency metadata, or deployment notes
- `安全加固`: vulnerability fix, permission boundary, secret handling, input validation, or supply-chain hardening
- `数据/算法`: data processing, model logic, scoring, search, ranking, analytics, or algorithmic behavior

### Business Domain

Choose the product domain, not the technical layer:

- `通用开发工具`
- `企业管理/SaaS`
- `电商/交易`
- `金融/支付`
- `教育/内容`
- `医疗/健康`
- `社交/社区`
- `游戏/互动娱乐`
- `数据分析/AI`
- `安全/合规`
- `基础设施/DevOps`
- `其他`

### Modification Scope

- `单文件`: one source/config/doc file
- `少量文件`: 2-4 related files in one module or layer
- `多文件`: 5+ files or multiple components within one subsystem
- `跨模块`: multiple subsystems, frontend-backend contract, shared library plus callers, or schema plus code
- `项目级`: build system, architecture, CI, dependency strategy, scaffolding, or broad repo organization

### Difficulty

- `简单`: local change, obvious success criteria, low risk, usually <30 minutes
- `中等`: requires repo understanding, tests, several files, or non-trivial state/data flow
- `困难`: cross-module behavior, migration, concurrency, security, complex UI state, integrations, or high regression risk
- `专家`: ambiguous architecture, distributed systems, deep domain constraints, major redesign, or hard-to-verify correctness

## Satisfaction Rubric

- `满意`: fulfills the prompt's core behavior, preserves existing behavior, passes relevant verification or has convincing evidence, and keeps scope appropriate.
- `基本满意`: core request is mostly done but has minor omissions, weak tests, small polish gaps, or unverified but plausible behavior.
- `不满意`: misses core behavior, introduces regression, fails relevant tests, ignores key constraints, has unusable quality, or cannot be judged from the submitted artifacts.

For `基本满意` or `不满意`, always provide `不满意原因`. For `满意`, set `不满意原因: 无`.

## Output Template

Use this table for each completed annotation:

| 字段 | 值 |
|---|---|
| repoId | 待补充 |
| 做题人 | 待补充 |
| 提交日期 | 待补充 |
| Trae sessionId | 待补充 |
| user prompt | 待补充 |
| GitHub repo URL | 待补充 |
| GitHub commit id | 待补充 |
| 任务类型 | 待补充 |
| 业务领域 | 待补充 |
| 修改范围 | 待补充 |
| 任务难度 | 待补充 |
| 任务是否满意 | 待补充 |
| 不满意原因 | 待补充 |

Then add a short `依据` section listing the files, diff, tests, or logs used for the classification.

## Guardrails

- Separate setup facts from grading conclusions.
- Prefer concrete evidence over impressions.
- Do not mark a task satisfied only because the implementation is large.
- Do not penalize missing work that was not requested unless it blocks the requested behavior.
- If the submitted artifacts are insufficient, mark unknown fields as `待补充` and explain the evidence gap.
