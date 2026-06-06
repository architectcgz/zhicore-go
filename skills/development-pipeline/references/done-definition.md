# Done Definition

The work is done only when all applicable conditions below are true.

## Design and Plan
- the final implementation still matches the approved spec, or deviations are documented
- planned slices are completed or explicitly deferred
- if an implementation plan exists and the work is being reported as complete, `scripts/check_impl_plan_done.sh <impl-plan-path>` passes with no unchecked checklist items

## Validation
- slice-level validation is complete for the implemented work
- integration validation is complete for the critical paths
- unvalidated areas are explicitly called out

## Review
- required reviews have been completed
- unresolved comments are either addressed or explicitly recorded

## Operational Readiness
- rollout, migration, compatibility, or rollback notes exist when needed
- residual risk is summarized

## Delivery Readiness
- the branch is in a reviewable state
- the handoff summary exists
- the next consumer can understand what changed and what remains risky
