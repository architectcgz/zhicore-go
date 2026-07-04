---
name: development-pipeline
description: Use when engineering work is multi-step, cross-module, high-risk, or needs formal review gates, validation evidence, and disciplined branch finishing before handoff.
---

# Development Pipeline

Use this skill as the top-level workflow controller for non-trivial engineering work. It coordinates specialist implementation skills; it does not replace frontend, backend, testing, review, runtime safety, or branch-finishing skills.

When a repository uses `code-workflow`, that skill owns the mechanical enforcement layer: non-trivial task binding, isolated workspace or in-place context selection, implementation-plan startup gates, shared scaffold, and independent-review gate mechanics. This skill owns stage execution and artifact quality on top of that layer.

## Always Read

- `rules/core.md` for non-trivial classification, review gates, implementation-plan requirements, model routing, and skill composition.
- `workflows/stage-execution.md` for the staged pipeline, stage exit criteria, loopback behavior, output shape, and completion standard.

## Reference Map

| Task surface | Read |
|---|---|
| Full contract for each pipeline stage | `references/stage-definitions.md` |
| Spec, plan, per-task, integration, and final review rejection rules | `references/review-gates.md` |
| Planning or implementing slices | `references/task-slicing-rules.md` |
| Claiming readiness or handoff | `references/validation-evidence.md` |
| Final done criteria | `references/done-definition.md` |
| Closing or handing off a branch | `references/branch-finishing-checklist.md` |
| Drafting a spec | `references/spec-template.md` |
| Drafting or reviewing an implementation plan | `references/plan-template.md` |
| Checking an implementation plan checklist | `scripts/check_impl_plan_done.sh <impl-plan-path>` |
| Repository-level mechanical workflow | `../code-workflow/SKILL.md` |

## Known Gotchas

- Architecture or design documents are inputs, not implementation plans. Non-trivial code work needs a concrete implementation plan before coding.
- New user/developer-visible capability must run `brainstorming` before design, planning, or implementation.
- If implementation reveals a second architecture redesign is still needed immediately afterward, reopen the plan before continuing.
- If a slice touches known structural debt, the debt closure is in scope for that slice unless the work is re-sliced before coding.
- For non-trivial work, implementation-context self-check is not the independent review gate.

## Check

- Did you classify trivial vs non-trivial and state why?
- Did you read `rules/core.md` and `workflows/stage-execution.md`?
- Did you load only the stage/reference files relevant to the current step?
- Did the final handoff cite validation evidence, review status, and residual risk?
