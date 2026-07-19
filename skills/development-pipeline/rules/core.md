# Core Rules

Read this file when `development-pipeline` is active.

## Overview

Use `development-pipeline` as the stage controller for non-trivial engineering work. It defines when to enter each stage, what each stage must produce, when to loop back, and how to finish with evidence instead of vague progress claims.

## Rules

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
12. For migrations, replacements, standardization, and user-selected framework adoption, structural convergence includes the final initialization path, driver/provider, runtime owner, transactions, primary adapters, required legacy removals, and proof that the old default path is gone. Minimal diff is evaluated inside that target state.
13. If implementation reveals "this plan completes the feature but still leaves a second architecture redesign immediately afterward", stop and reopen the plan. Record the structural convergence task in the plan before continuing.
14. If the current slice touches a file, component, service, or module that is already tracked as structural debt, oversized ownership, or required decomposition, that debt becomes in-scope for the slice. Do not treat it as a follow-up note while still merging new behavior into the same surface.
15. When touched structural debt cannot be closed inside the current slice without expanding scope too far, loop back before coding and re-slice the work. Do not ship the feature first and leave the same touched debt behind.
16. Before final handoff, check whether the project harness declares a workflow completion script such as `scripts/check-workflow-complete.sh` or an equivalent completion gate. If it exists, run it and report the result; if it cannot run, state the blocker instead of relying on prompt-level memory.

## Model routing

When this workflow dispatches specialist work, use these defaults:

- plan writing and plan review: `gpt-5.5` with `medium`
- code review stages: `gpt-5.5` with `medium`
- implementation work: `gpt-5.4` with `medium`
- challenging implementation work with heavier reasoning, broad codebase context, or repeated blockage: `gpt-5.4` with `high`

If a stage needs a stronger model than its default, escalate explicitly and state why.

## When to use

Use this skill when the request involves one or more of the following:

- a feature, refactor, migration, or bugfix likely to touch multiple files or modules
- work that benefits from spec and plan review before coding
- a task that should be split into reviewable sub-tasks
- changes that need integration validation, handoff notes, or disciplined branch finishing
- changes that involve multiple supporting skills such as backend-engineer, frontend-engineer, test-master, reviewer, or runtime-ops-safety

Do not use this skill for trivial copy edits or one-line fixes, purely exploratory discussion with no implementation intent, standalone code review requests, or emergency hotfixes that should use a shorter incident or runtime-safety flow.

## Trivial vs non-trivial

Trivial work is local, obvious, reversible, and does not alter behavior boundaries. Examples: typo fixes, narrow docs edits, one styling-token adjustment, a small null guard, or a test mock update with no contract or state change.

Non-trivial work includes any API or DTO change, route or permission change, database/cache/queue/transaction/concurrency change, frontend async or state-flow change, multi-file or cross-module change, new feature, migration, deletion, split, rename, review-driven fix with regression risk, behavior-defining test update, or growth in already oversized code.

If the work touches code that is already known to be structurally oversized or owner-mixed, classify that debt payoff as part of the non-trivial slice instead of optional cleanup.

For non-trivial work, implementation-agent self-check is necessary but never sufficient: an independent review gate must pass before handoff. When the repository uses `code-workflow`, that skill owns the gate mechanics.

## Independent review gate

For non-trivial work, the implementing context's self-check never satisfies completion. The gate review must run in a separate subagent or an equivalently independent context; same-context review counts only as self-check.

- Entering `development-pipeline` or `code-workflow` counts as the user's explicit delegation authorization for the minimum necessary independent reviewer. Do not pause to ask again for delegation permission unless the user explicitly forbade it.
- If tool policy or an explicit user restriction blocks spawning the reviewer, stop and state that the gate is unmet. Never archive same-context review as if it met this gate.
- After the gate passes, fix material findings and re-run impacted verification.

When the repository uses `code-workflow`, this gate maps onto its order: `completion-full` is self-check, the independent review gate runs next, and only then can `workflow-governance` / final handoff be treated as completion-ready.

## Review evidence location

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

## Implementation plan requirement

For non-trivial work, an architecture or design document is not sufficient authorization to code. Before implementation starts, create or identify a concrete implementation plan under the target repository:

```text
docs/plan/impl-plan/YYYY-MM-DD-{scope}-implementation-plan.md
```

If the repository already has a more specific established plan directory, use it only when it serves the same purpose and cite the exact path.

The implementation plan must include objective and non-goals, source architecture or design docs, ordered task slices, expected file/module boundaries, data/API/state/migration/compatibility impacts, validation commands or behavior checks per slice, review focus per slice, and rollback or recovery notes when relevant.

Before coding starts, evaluate whether ownership boundaries are explicit, shared builders/readers/services/state owners have a landing zone, the plan delivers both behavior and required structure, any "implement now, redesign right after" risk remains hidden, and any touched structural debt will be closed in the same pipeline rather than deferred.

If the answer to any of these is unclear, revise the plan before implementation.

Plan review must confirm the plan is executable, sliced into reviewable units, and has credible validation before coding begins. If coding reveals the plan is wrong, return to planning instead of silently improvising.

## Skill composition

Use this skill to orchestrate, then pull in the appropriate specialist skill for each phase:

- frontend-heavy design or implementation: `frontend-engineer` and relevant framework-specific frontend skill
- service, API, cache, queue, consistency, or stateful backend work: `backend-engineer`
- test strategy and validation matrix design: `test-master`
- test execution and evidence capture: `test-engineer`
- design review, risk review, or final code review: `reviewer`
- UI or UX review: `ui-ux-pro-max`
- high-risk runtime, migration, or production-sensitive steps: `runtime-ops-safety`
- incident-style follow-up or postmortem capture: `incident-capture`

Do not import every skill at once. Only apply the ones relevant to the current stage.
