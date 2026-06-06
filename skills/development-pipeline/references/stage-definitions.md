# Stage Definitions

Use this file when you need the full contract for each stage.

## 1. Intake or triage

Purpose: decide whether the work needs the full pipeline and how risky it is.

Inputs:
- user request
- repository or system context if available
- obvious constraints

Outputs:
- task classification
- recommended pipeline depth
- initial specialist skill mapping
- initial risk list
- explicit note on whether the task touches any already-known structural debt surface that must be closed in the same pipeline

## 2. Brainstorming

Purpose: widen and then narrow the solution space.

Mandatory before implementing any new feature or adding user/developer-visible capability. Existing architecture or design docs are inputs, not substitutes.

Inputs:
- triage result
- known constraints
- current architecture

Outputs:
- candidate approaches
- tradeoffs
- chosen direction
- unresolved questions that block the spec

## 3. Design or spec

Purpose: turn the chosen direction into a reviewable design artifact.

Outputs should cover:
- problem statement
- scope
- non-goals
- architecture or flow
- contracts or interfaces
- state or data impacts
- migration or rollout concerns
- validation approach
- risks and assumptions

## 4. Spec review

Purpose: reject weak design before coding starts.

Review for:
- missing boundaries
- unclear ownership
- unverifiable assumptions
- migration or compatibility gaps
- rollout blind spots
- incomplete validation thinking

## 5. Task planning

Purpose: split approved design into execution slices.

A good plan makes dependencies, validation, and review boundaries explicit.

For non-trivial implementation, architecture or design docs are only inputs. This stage must produce or cite a concrete implementation plan with ordered task slices, expected file or module boundaries, dependencies, validation, review focus, and risk notes before coding starts.
If the work touches a known oversized, owner-mixed, or otherwise tracked structural-debt surface, the plan must also say how that debt will be closed in the current pipeline and how the reviewer can verify closure.

## 6. Plan review

Purpose: reject bad slicing, hidden dependencies, and validation gaps.

The plan review must also reject attempts to treat an architecture document as an implementation plan when task slices, changed boundaries, validation commands, or review focus are missing.
The plan review must also reject any slice that adds behavior to a known debt surface while deferring the debt itself to a later task.

## 7. Worktree setup

Purpose: ensure implementation happens in the right isolated workspace when needed.

## 8. Per-task implementation

Purpose: deliver one reviewable slice at a time.

Each slice should stay narrow enough that validation and review are specific.

Each slice should cite the implementation plan item being executed. If the code no longer fits the plan, return to task planning or plan review.
If the slice touches a known debt surface and implementation reveals the debt cannot be closed safely within that slice, stop and return to planning instead of merging partial debt payoff plus new behavior.

## 9. Per-task dual review

Purpose: catch both code-level and domain-level defects before integration.

## 10. Integration validation

Purpose: prove the slices work together.

## 11. Final code review

Purpose: review the entire change as one coherent unit.

This stage must fail if the change touched a known structural-debt surface and still leaves that same debt behind as residual risk or follow-up work.

## 12. Release notes or handoff

Purpose: make downstream use, rollout, and review easier.

## 13. Finishing a development branch

Purpose: leave the branch clean, reviewable, and ready for merge or handoff.
