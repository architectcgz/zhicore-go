---
name: development-pipeline
description: Use when engineering work is multi-step, cross-module, high-risk, or needs formal review gates, validation evidence, and disciplined branch finishing before handoff.
---

# Development Pipeline

## Overview

Use this skill as the top-level workflow controller for non-trivial engineering work. It defines when to enter each stage, what each stage must produce, when to loop back, and how to finish with evidence instead of vague progress claims.

This skill is an orchestrator. It does not replace domain implementation skills such as frontend, backend, testing, review, or runtime safety. It coordinates them.

## Core Rules

1. Do not skip directly from idea to implementation for medium or large tasks. Establish scope, review the spec, review the plan, then implement.
2. Every stage must have explicit inputs, outputs, and exit criteria.
3. If a review rejects the current artifact, revise it before moving forward.
4. If implementation or integration reveals a design mismatch, reopen the plan or the spec instead of silently patching around the problem.
5. Each task slice must be the smallest reviewable unit with a clear validation path.
6. Never claim validation that was not actually performed.
7. Finish with branch hygiene, release notes or handoff notes, and explicit residual risks.
8. Classify trivial versus non-trivial work during intake. Non-trivial work must complete implementation, verification, review, fix, and re-verification before handoff.
9. Architecture or design documents are inputs, not implementation plans. Do not start non-trivial code changes until a concrete implementation plan exists and passes plan review.
10. Before implementing any new feature or adding user/developer-visible capability, run `brainstorming` and record the chosen direction before design, planning, or code.
11. After an implementation plan is written for non-trivial work, do not move directly into coding. First run an explicit plan evaluation for architecture boundary clarity, reuse points, ownership, hidden redesign risk, and whether the plan is delivering behavior only or also the intended structural convergence.
12. If implementation reveals "this plan completes the feature but still leaves a second architecture redesign immediately afterward", stop and reopen the plan. Record the structural convergence task in the plan before continuing.
13. If the current slice touches a file, component, service, or module that is already tracked as structural debt, oversized ownership, or required decomposition, that debt becomes in-scope for the slice. Do not treat it as a follow-up note while still merging new behavior into the same surface.
14. When touched structural debt cannot be closed inside the current slice without expanding scope too far, loop back before coding and re-slice the work. Do not ship the feature first and leave the same touched debt behind.
15. Before final handoff, check whether the project harness declares a workflow completion script such as `scripts/check-workflow-complete.sh` or an equivalent completion gate. If it exists, run it and report the result; if it cannot run, state the blocker instead of relying on prompt-level memory.

## Model Routing

When this workflow dispatches specialist work, use these defaults:

- plan writing and plan review: `gpt-5.5` with `medium`
- code review stages: `gpt-5.5` with `medium`
- implementation work: `gpt-5.4` with `medium`
- challenging implementation work with heavier reasoning, broad codebase context, or repeated blockage: `gpt-5.4` with `high`

If a stage needs a stronger model than its default, escalate explicitly and state why.

## When To Use This Skill

Use this skill when the request involves one or more of the following:

- a feature, refactor, migration, or bugfix likely to touch multiple files or modules
- work that benefits from spec and plan review before coding
- a task that should be split into reviewable sub-tasks
- changes that need integration validation, handoff notes, or disciplined branch finishing
- changes that involve multiple supporting skills such as backend-engineer, frontend-engineer, test-master, code-reviewer, or runtime-ops-safety

Do not use this skill for:

- trivial copy edits or one-line fixes
- purely exploratory discussion with no implementation intent
- standalone code review requests
- emergency hotfixes that should use a shorter incident or runtime-safety flow

## Trivial vs Non-Trivial

Trivial work is local, obvious, reversible, and does not alter behavior boundaries. Examples: typo fixes, narrow docs edits, one styling-token adjustment, a small null guard, or a test mock update with no contract or state change.

Non-trivial work includes any API or DTO change, route or permission change, database/cache/queue/transaction/concurrency change, frontend async or state-flow change, multi-file or cross-module change, new feature, migration, deletion, split, rename, review-driven fix with regression risk, behavior-defining test update, or growth in already oversized code.

If the work touches code that is already known to be structurally oversized or owner-mixed, classify that debt payoff as part of the non-trivial slice instead of optional cleanup.

For non-trivial work, implementation-agent self-check is required but insufficient. The pipeline owns the final gate and must require independent review or an explicit review pass, then fix material findings and re-run impacted verification.
When the implementation was produced in the current session, do not perform that gate review in the same implementation context.
For non-trivial work, the default is also the requirement: the gate review must run in a separate subagent or an equivalently independent context. Same-context review may count as self-check only, never as the independent gate.
If the user explicitly asks to use this pipeline, a staged workflow, or an independent review gate, treat that request as explicit authorization to spawn the minimum necessary review subagent(s) required by this skill.
Do not stop to ask again for permission to spawn that reviewer. The pipeline request already authorizes the delegation needed to satisfy the review gate.
If tool policy or user instruction prevents spawning the independent reviewer, stop and state that the review gate is still unmet. Do not archive same-context review as if it satisfied the independent review requirement.
If the repository uses `code-workflow`, map this requirement onto that workflow explicitly: `completion-full` is self-check, then the independent review gate runs, and only after that can `workflow-governance` / final handoff be treated as completion-ready.

## Review Evidence Location

Independent reviews that gate non-trivial work must be archived in one default place inside the target repository:

```text
docs/reviews/{frontend|backend|architecture|security|general}/YYYY-MM-DD-{scope}-review-{short-topic}.md
```

Use the category that matches the dominant risk surface; use `general` when no category fits. Create the category directory if needed.

Only use this global fallback for non-project work such as global agent, skill, hook, or personal tooling changes:

```text
/home/azhi/.codex/reviews/{frontend|backend|architecture|security|general}/YYYY-MM-DD-{scope}-review-{short-topic}.md
```

Pipeline completion must cite the review archive path, or explicitly state why the change was classified as trivial and did not require archived independent review.

## Implementation Plan Requirement

For non-trivial work, an architecture or design document is not sufficient authorization to code. Before implementation starts, create or identify a concrete implementation plan under the target repository:

```text
docs/plan/impl-plan/YYYY-MM-DD-{scope}-implementation-plan.md
```

If the repository already has a more specific established plan directory, use it only when it serves the same purpose and cite the exact path.

The implementation plan must include:

- Objective and non-goals
- Source architecture or design docs used as inputs
- Ordered task slices with dependencies
- Files, modules, or boundaries expected to change
- Data, API, state, migration, or compatibility impacts
- Validation commands or behavior checks for each slice
- Review focus for each slice
- Rollback or recovery notes when relevant

Before coding starts, the implementation plan must also be evaluated for:

- whether target ownership boundaries are explicit instead of implied
- whether shared builders, readers, services, or state owners have a defined landing zone
- whether the plan is only aligning outputs while deferring structural convergence
- whether any "implement now, redesign right after" risk remains hidden
- whether the slice touches any already-known structural debt surface and, if so, exactly how that debt will be closed in the same pipeline rather than deferred

If the answer to any of these is unclear, revise the plan before implementation.

Plan review must confirm the plan is executable, sliced into reviewable units, and has credible validation before coding begins. If coding reveals the plan is wrong, return to planning instead of silently improvising.

## Workflow Overview

Follow this sequence:

1. intake or triage
2. brainstorming
3. design or spec
4. spec review
5. task planning
6. plan review
7. worktree setup
8. per-task implementation
9. per-task dual review
10. integration validation
11. final code review
12. release notes or handoff
13. finishing a development branch

When the repo uses `code-workflow`, the relevant completion order is:

1. `completion-full`
2. independent review gate
3. fix + impacted re-validation when needed
4. `workflow-governance`
5. handoff / archive / branch finishing

Read [references/stage-definitions.md](references/stage-definitions.md) for the full stage contract.
Read [references/review-gates.md](references/review-gates.md) for rejection and loopback rules.
Read [references/task-slicing-rules.md](references/task-slicing-rules.md) before planning or implementing.
Read [references/validation-evidence.md](references/validation-evidence.md) before claiming readiness.
Read [references/done-definition.md](references/done-definition.md) before final handoff.
Read [references/branch-finishing-checklist.md](references/branch-finishing-checklist.md) before closing the branch.
Read [references/spec-template.md](references/spec-template.md) when drafting the spec.
Read [references/plan-template.md](references/plan-template.md) when drafting the task plan.

## Skill Composition Map

Use this skill to orchestrate, then pull in the appropriate specialist skill for each phase:

- design or implementation in frontend-heavy work: use frontend-engineer and any framework-specific frontend skill
- design or implementation in service, api, cache, queue, consistency, or stateful backend work: use backend-engineer
- test strategy and validation matrix design: use test-master
- test execution and evidence capture: use test-engineer
- design review, risk review, or final code review: use code-reviewer
- ui or ux review: use ui-ux-pro-max
- high-risk runtime, migration, or production-sensitive steps: use runtime-ops-safety
- incident-style follow-up or postmortem capture: use incident-capture

Do not import every skill at once. Only apply the ones relevant to the current stage.

## Stage Execution Rules

### 1) Intake or triage

Classify the work before doing deep design.

You must determine:

- task type: feature, bugfix, refactor, migration, spike, hotfix, or mixed
- trivial or non-trivial classification, with the reason
- whether the task creates a new feature or capability and therefore requires `brainstorming`
- whether the full pipeline is necessary
- whether the work is high-risk or cross-module
- whether specialist skills are required immediately

Required output:

- a short classification
- scope summary
- initial risks
- recommended pipeline path

Exit only when the task is classified and the path is clear.

### 2) Brainstorming

Use this stage to expand options, constraints, and edge cases before fixing the design.

This stage is mandatory before implementing new features or adding user/developer-visible capability. Do not skip it because an architecture document already exists; architecture docs are inputs to brainstorming, not replacements for it.

Required output:

- candidate approaches
- tradeoffs
- rejected approaches when obvious
- open questions that materially affect the spec
- chosen direction to carry into design, spec, or implementation planning

Exit when one primary direction is chosen or when the decision space is narrowed enough to write a spec.

### 3) Design or spec

Draft the spec using the template. Cover scope, non-goals, architecture or flow, interfaces, risks, migrations, validation, and rollout assumptions.

Exit only when the spec is concrete enough for review.

### 4) Spec review

Review for missing boundaries, hidden complexity, migration risk, weak assumptions, and unverifiable claims.

If the spec fails review, revise it and repeat this stage.

Exit only when:

- major ambiguities are resolved
- validation and risk handling are plausible
- task planning can proceed without guessing core behavior

### 5) Task planning

Turn the approved spec into execution slices. Use the plan template.

Each task must include:

- goal
- touched modules or boundaries
- dependencies
- validation method
- review focus
- risk notes

For non-trivial implementation, this stage must produce or cite the implementation plan file. Architecture documents may be linked as inputs, but they do not satisfy this stage by themselves.

Exit only when the work is broken into small reviewable units.

### 6) Plan review

Review task ordering, missing dependencies, task size, risk concentration, and validation gaps.

For non-trivial work, this stage must also include an explicit architecture-fit evaluation:

- Is the target architecture boundary clear enough to code against?
- Are the owner, reuse point, and abstraction landing zone named?
- Is the plan solving both behavior and required structure, or only behavior?
- If structure is intentionally deferred, is that deferred convergence written as its own tracked task with completion criteria?
- If the slice touches an already-known debt surface, does the plan actually remove that debt from the touched surface instead of only documenting it?

If the plan fails review, revise it and repeat this stage.

Exit only when task sequencing and validation are credible, and the implementation plan path is known.

### 7) Worktree setup

Create or select the task's isolated implementation workspace when repository work warrants it.

Exit only when the correct working context is ready and the task-to-worktree mapping is clear.

### 8) Per-task implementation

Implement one task slice at a time. Do not collapse multiple major slices into one pass.

For each slice:

- cite the implementation plan item being executed
- restate the slice goal briefly

Implementation-stage self-check may catch obvious mistakes, but it does not satisfy the review gate for non-trivial work. Do not convert "I re-read my own patch" into a completed review stage.
- implement only the scoped change
- run the narrowest relevant validation
- update the corresponding implementation-plan checklist immediately after the slice or step passes its validation, before starting the next slice
- require the implementing specialist to state self-check results and review needs
- record anything left unverified

Checklist state is part of the implementation artifact, not a final reporting chore. If a slice is committed, include the matching checklist update in the same commit when practical; otherwise create a small follow-up docs commit before handoff. Never leave completed plan items as `- [ ]` after claiming the slice is done.

If implementation reveals spec or plan defects, return to the appropriate earlier stage.

### 9) Per-task dual review

Each completed slice gets two angles of review whenever applicable:

- implementation review: correctness, maintainability, boundary hygiene
- domain or validation review: product, API, data, testing, UX, or runtime safety as needed
- archived review evidence for non-trivial gates, following the Review Evidence Location policy

For non-trivial work, at least the gate review must be performed by a separate subagent or an equivalently independent context. The implementing agent may add self-review notes, but those notes do not satisfy this stage by themselves.
When the user asked to use this pipeline or equivalent staged workflow, trigger that reviewer automatically instead of reinterpreting the request as "pipeline without delegation."

If review rejects the slice, revise the slice and review it again.

### 10) Integration validation

After the slices are complete, validate the end-to-end path.

Check:

- cross-module behavior
- contracts and state transitions
- migrations or compatibility concerns
- user-visible behavior where relevant
- logs, metrics, or operational signals when relevant
- implementation-plan checklist completion, using `scripts/check_impl_plan_done.sh <impl-plan-path>` from this skill when an impl-plan exists

If integration fails, return to the affected task slice or reopen the plan or spec if the issue is architectural.

### 11) Final code review

Review the full change set as a coherent unit.

Focus on:

- hidden coupling
- design drift from the spec
- review debt left over from slice-by-slice work
- rollback, rollout, and residual risk
- review archive completeness and whether material findings have matching fixes and re-validation evidence

If the change touched a known structural-debt surface and that debt is still present, this stage must fail. Do not downgrade that condition into residual risk or future follow-up.

For non-trivial work, this final gate must cite the independent reviewer context explicitly: subagent id, review worktree if different, or another concrete independent review boundary. If that evidence is missing, the pipeline is not complete.
If the active tool environment allows subagents and the user asked for this pipeline, missing reviewer dispatch is a pipeline execution error. Fix it by dispatching the reviewer instead of downgrading to same-context review.

### 12) Release notes or handoff

Before closing the branch, summarize:

- what changed
- why it changed
- how it was validated
- what remains risky or intentionally deferred
- any rollout, migration, or operational notes

### 13) Finishing a development branch

Only finish the branch after the done definition is met. Clean up the branch state, ensure commit hygiene, and leave a merge-ready or handoff-ready result.

## Default Output Structure

When using this skill, structure your response around the current stage. Prefer this shape:

- current stage
- objective
- inputs used
- output produced
- validation or review result
- next stage
- loopback reason if not advancing

## Completion Standard

The work is only complete when the current stage artifacts exist, the relevant reviews have passed, validation evidence is explicit, and the done definition is satisfied.
