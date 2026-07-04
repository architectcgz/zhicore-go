# Stage Execution Workflow

Read this file when executing `development-pipeline`.

## Workflow overview

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

Read `references/stage-definitions.md` for the full stage contract; `references/review-gates.md` for rejection and loopback rules; `references/task-slicing-rules.md` before planning or implementing; `references/validation-evidence.md` before claiming readiness; `references/done-definition.md` before final handoff; `references/branch-finishing-checklist.md` before closing the branch; `references/spec-template.md` when drafting the spec; and `references/plan-template.md` when drafting the task plan.

## Stage execution rules

### 1. Intake or triage

Classify the work before doing deep design. Determine task type, trivial/non-trivial classification with reason, whether the task creates a new feature or capability and therefore requires `brainstorming`, whether the full pipeline is necessary, whether the work is high-risk or cross-module, and whether specialist skills are required immediately.

Required output: short classification, scope summary, initial risks, and recommended pipeline path. Exit only when the task is classified and the path is clear.

### 2. Brainstorming

Use this stage to expand options, constraints, and edge cases before fixing the design. This stage is mandatory before implementing new features or adding user/developer-visible capability. Architecture docs are inputs, not replacements.

Required output: candidate approaches, tradeoffs, rejected approaches when obvious, open questions that materially affect the spec, and the chosen direction. Exit when one primary direction is chosen or the decision space is narrow enough to write a spec.

### 3. Design or spec

Draft the spec using the template. Cover scope, non-goals, architecture or flow, interfaces, risks, migrations, validation, and rollout assumptions. Exit only when the spec is concrete enough for review.

### 4. Spec review

Review for missing boundaries, hidden complexity, migration risk, weak assumptions, and unverifiable claims. If the spec fails review, revise it and repeat this stage. Exit only when major ambiguities are resolved and task planning can proceed without guessing core behavior.

### 5. Task planning

Turn the approved spec into execution slices using the plan template. Each task must include goal, touched modules or boundaries, dependencies, validation method, review focus, and risk notes.

For non-trivial implementation, this stage must produce or cite the implementation plan file. Architecture documents may be linked as inputs, but they do not satisfy this stage by themselves. Exit only when the work is broken into small reviewable units.

### 6. Plan review

Review task ordering, missing dependencies, task size, risk concentration, and validation gaps.

For non-trivial work, include architecture-fit evaluation: target boundary clarity, owner/reuse/abstraction landing zone, behavior plus structure delivery, deferred convergence criteria, and whether touched known debt surfaces are actually closed in this pipeline.

If the plan fails review, revise it and repeat this stage. Exit only when task sequencing and validation are credible and the implementation plan path is known.

### 7. Worktree setup

Create or select the task's isolated implementation workspace when repository work warrants it. Exit only when the correct working context is ready and the task-to-worktree mapping is clear.

### 8. Per-task implementation

Implement one task slice at a time. Do not collapse multiple major slices into one pass.

For each slice, cite the implementation plan item, restate the slice goal, implement only the scoped change, run the narrowest relevant validation, update the implementation-plan checklist immediately after the slice or step passes validation, require self-check results and review needs, and record anything left unverified.

Implementation-stage self-check may catch obvious mistakes, but it does not satisfy the review gate for non-trivial work. If implementation reveals spec or plan defects, return to the appropriate earlier stage.

Checklist state is part of the implementation artifact, not a final reporting chore. If a slice is committed, include the matching checklist update in the same commit when practical; otherwise create a small follow-up docs commit before handoff.

### 9. Per-task dual review

Each completed slice gets two angles of review whenever applicable: implementation review and domain/validation review. For non-trivial gates, archive review evidence following the Review Evidence Location policy in `rules/core.md`.

If review rejects the slice, revise the slice and review it again.

### 10. Integration validation

After the slices are complete, validate the end-to-end path: cross-module behavior, contracts, state transitions, migrations or compatibility, user-visible behavior, logs/metrics/operational signals, and implementation-plan checklist completion using `scripts/check_impl_plan_done.sh <impl-plan-path>` when an impl-plan exists.

If integration fails, return to the affected task slice or reopen the plan/spec if the issue is architectural.

### 11. Final code review

Review the full change set as a coherent unit: hidden coupling, design drift, review debt, rollback/rollout/residual risk, review archive completeness, and whether material findings have fixes plus re-validation evidence.

If the change touched a known structural-debt surface and that debt is still present, this stage fails. Do not downgrade that condition into residual risk or future follow-up.

For non-trivial work, cite the independent reviewer context explicitly. Missing reviewer evidence means the pipeline is not complete.

### 12. Release notes or handoff

Before closing the branch, summarize what changed, why it changed, how it was validated, what remains risky or intentionally deferred, and rollout/migration/operational notes.

### 13. Finishing a development branch

Only finish the branch after the done definition is met. Clean up branch state, ensure commit hygiene, and leave a merge-ready or handoff-ready result.

## Default output structure

When using this skill, structure the response around the current stage:

- current stage
- objective
- inputs used
- output produced
- validation or review result
- next stage
- loopback reason if not advancing

## Completion standard

The work is only complete when the current stage artifacts exist, relevant reviews have passed, validation evidence is explicit, and the done definition is satisfied.
