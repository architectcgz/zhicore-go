---
name: harness-router
description: Use when starting repository work to decide whether a request should enter the project harness workflow before execution. Trigger for most coding, refactor, testing, documentation, review, migration, architecture, prompt, skill, process, or multi-step tasks; skip only clearly simple local tasks such as answering a small question, reading one file, running a harmless command, or making a trivial typo-style edit.
metadata:
  short-description: Route tasks into harness workflows
---

# Harness Router

Use this skill as the default intake router for repository work. The rule is: **enter harness unless the task is clearly simple**.

This skill does not replace domain skills. It decides whether to load project harness context first, then routes to the right workflow.

## Default Decision

Default to `HARNESS` when the task may affect:

- code behavior, tests, config, scripts, hooks, CI, docs, prompts, skills, policies, architecture, or review evidence
- more than one file or one responsibility boundary
- implementation plans, validation strategy, feedback capture, or reusable project knowledge
- project conventions that future agents should reuse
- anything that could create drift between code, docs, tests, and agent instructions

Use `SIMPLE` only when all are true:

- local, obvious, and reversible
- no code behavior or project convention changes
- no need to read harness directories to avoid mistakes
- no durable knowledge should be recorded
- no validation beyond the direct command or answer is meaningful

Examples of `SIMPLE`:

- answer a small factual question from one visible file
- run `date`, `pwd`, `git status`, or a specific harmless command
- fix one typo in prose when no structure or rule changes
- explain a short code snippet without changing it

When uncertain, choose `HARNESS`.

## HARNESS Intake

For `HARNESS`, read the project entry points in this order, stopping once enough context is gathered:

1. root `AGENTS.md`
2. `concepts/AGENTS.md`
3. `practice/AGENTS.md`
4. `feedback/AGENTS.md`
5. `harness/prompts/AGENTS.md` or `prompts/AGENTS.md`, depending on the project harness shape
6. `references/AGENTS.md`
7. `works/AGENTS.md`
8. `scripts/check-consistency.sh`

Then classify the request:

- task type: question, implementation, bugfix, refactor, migration, docs, review, test, skill/prompt, process, cleanup
- complexity: simple, non-trivial, structural, high-risk
- touched harness areas: concepts, thinking, practice, feedback, prompts, references, works, scripts
- required specialist skills
- required validation

State the route briefly in commentary before substantial work.

## Routing Table

| Request shape | Route |
| --- | --- |
| Create, update, or initialize harness structure | `harness-engineering` |
| Non-trivial implementation, refactor, migration, or cross-module docs/code change | `development-pipeline` first, then domain skills |
| Backend code, API, config, DB, jobs, queues, concurrency | `backend-engineer` or language-specific backend skill |
| Frontend implementation, Vue state, components, routes, async behavior | `frontend-engineer`; if broad UI request, use `frontend-task-router` |
| Review a patch, plan, architecture document, or implementation | `reviewer` |
| Validate completed work | `test-engineer` |
| Create or update a skill | `skill-creator` |
| Create or update reusable prompts or workflows | read the project prompt directory AGENTS file; update `harness/prompts/` or `prompts/` only when the prompt remains project-local and reusable |
| Repeated mistake, missing project rule, process gap | update `feedback/`; use `improvement-tracker` if available and appropriate |
| External article, repo, research, or reference material | update `references/` |
| Practice run, workflow experiment, implementation history | update `practice/` |
| Presentable template/report/map | update `works/` |

## Harness Update Rules

Update harness artifacts when the task creates durable knowledge:

- New recurring rule or mistake -> `feedback/`
- New reusable project-local prompt -> `harness/prompts/` or the project prompt directory
- New external source or research index -> `references/`
- New workflow experiment or migration record -> `practice/`
- New explanation intended as a reusable output -> `works/`
- New concept or project-wide principle -> `concepts/`
- New mechanical invariant -> `scripts/check-consistency.sh`

Do not dump implementation details into harness directories. Keep harness files as maps, indexes, lessons, prompts, and checks.

Before completion, decide whether the task produced reusable experience:

- If yes, state where it was absorbed: `feedback/`, `harness/prompts/`, `AGENTS.md`, a global skill, a project policy, or a mechanical check.
- If no, state that no new reusable experience was found.
- If a lesson is already fully absorbed by a global skill, project policy, or mechanical check, do not leave a long duplicate feedback or prompt body in the project harness; keep only an index note when traceability is useful.

## Validation

For `HARNESS` tasks, finish with the smallest relevant verification. If the repo has `scripts/check-consistency.sh`, run it whenever harness files, AGENTS navigation, prompts, feedback, references, or checks changed.

If no verification is run, state why.

## Output Contract

At completion, include:

- route chosen: `SIMPLE` or `HARNESS`
- harness files changed, if any
- validation commands actually run
- residual risks or skipped harness updates
