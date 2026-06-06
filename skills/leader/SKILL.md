---
name: leader
description: Use when a task is large, cross-cutting, high-risk, or uncertain enough to need explicit stage control and delivery coordination instead of ad hoc execution.
---

# Leader

Coordinate delivery as a staged engineering workflow instead of a pile of disconnected actions.

## Use When

- The task spans multiple phases such as analysis, planning, implementation, review, validation, and documentation
- The work touches multiple modules or carries meaningful architectural, operational, or rollout risk
- The request would benefit from explicit sequencing, stage gates, and a clear fix loop

## Do Not Use

- Small, single-file changes with obvious scope
- Pure implementation tasks that already have a clear plan and do not need orchestration
- Work that needs formal spec review, plan review, gated multi-stage delivery, or branch-finishing discipline; use `development-pipeline` instead

## Core Guardrails

1. Start from the user goal, constraints, and acceptance standard before deciding how to execute.
2. Prefer the smallest delivery slice that still forms a coherent, reviewable unit.
3. Do not move into implementation until the current understanding is good enough to avoid blind thrashing.
4. Keep high-risk areas explicit: schema, caches, auth, concurrency, state transitions, public APIs, and rollout order.
5. Separate what is verified from what is inferred at every stage.
6. Do not mark work as complete while review, validation, or planned documentation status is still unknown.
7. Own the workflow gate for non-trivial work. Implementation agents may self-check, but they do not decide alone that a non-trivial task is complete.
8. Treat architecture and design documents as inputs, not implementation plans. For non-trivial work, require an executable implementation plan before code changes begin.
9. Before new feature implementation or adding user/developer-visible capability, require `brainstorming` and carry its chosen direction into the plan.
10. After the implementation plan exists, require a plan-evaluation gate before coding: architecture boundary clarity, owner and reuse point definition, hidden redesign risk, and whether the plan covers structural convergence rather than only output behavior.

## Trivial vs Non-Trivial

Treat a change as trivial only when it is local, obvious, reversible, and does not alter behavior boundaries: typo fixes, small docs edits, one styling token adjustment, a narrow null guard, or a test mock update with no contract or state change.

Treat a change as non-trivial when any of these apply:

- API, DTO, route, permission, data model, persistence, cache, queue, transaction, concurrency, or scheduled-work changes
- frontend async flows, forms, repeated actions, route synchronization, stores, modal or drawer state, or user-visible workflow changes
- multi-file or cross-module changes
- new features, migrations, deletions, splits, renames, or review-driven fixes that can create regressions
- tests must change to express the new behavior
- the touched code is already hard to review locally, such as a large route view, large service, oversized component, or long function gaining more responsibility

For non-trivial work, require the closed loop: implementation, initial verification, independent review or explicit review pass, material fixes, impacted re-verification, and only then completion.

## Workflow

1. **Intake**
   - Restate the objective, constraints, and acceptance criteria.
   - Mark obvious high-risk areas and unknowns.
   - Identify whether the task is a new feature or capability and therefore needs `brainstorming`.
2. **Codebase Analysis**
   - Identify relevant modules, key call paths, affected files, and current behavior.
   - Surface compatibility or rollout risks before planning changes.
3. **Brainstorming**
   - Use `brainstorming` for new features or new capability.
   - Record the chosen direction before implementation planning.
4. **Implementation Plan**
   - Convert the task into explicit, reviewable steps.
   - Keep stage boundaries clear enough that later verification can prove completion.
   - Cite the implementation plan path for non-trivial work; architecture docs alone are not enough.
   - Include Documentation Planning before coding: identify the documentation owner, source of truth, affected paths, and any new-path registration required by the project documentation rules.
   - Do not advance from this stage to coding until the plan has passed the architecture-fit evaluation gate.
5. **Implementation**
   - Execute the plan with minimal diff and clear ownership for each change, including planned documentation edits.
6. **Review**
   - Check correctness, regressions, architecture drift, security, and missing tests.
   - Use specialist self-checks as input, but keep the final review gate at the leader level for non-trivial work.
7. **Validation**
   - Run the smallest sufficient verification set and capture evidence.
8. **Documentation Check**
   - Verify that the documentation items planned before implementation were updated or explicitly ruled out by the same source-of-truth rules.
9. **Fix Loop**
   - If review or validation fails, fix the root cause and repeat the relevant gates, including any affected planned documentation check.

## Output Expectations

- The current phase is always clear.
- Findings, decisions, and next actions are explicit.
- Risks and blockers are stated concretely instead of implied.
- Final output distinguishes implemented, verified, documented, and still-unverified work.
