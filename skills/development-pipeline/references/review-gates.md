# Review Gates and Loopback Rules

Use this file to decide whether to advance or loop back.

## Spec Review Gate

Reject or reopen the spec when:
- core behavior is still ambiguous
- ownership or boundaries are unclear
- migration or rollout risk is ignored
- validation is too vague to test
- the spec quietly assumes implementation details that have not been proven

Loopback target: design or spec

## Plan Review Gate

Reject or reopen the plan when:
- tasks are too large to review cleanly
- dependencies are missing or out of order
- validation is absent or too broad
- multiple high-risk changes are collapsed into one slice
- the plan does not map cleanly to the spec
- the slice touches a known structural-debt surface but does not include explicit debt-closure work and completion criteria

Loopback target: task planning

## Per-task Review Gate

Reject or reopen the current slice when:
- the slice leaks outside its scope
- boundary hygiene is poor
- validation does not cover the claimed behavior
- code quality is weak enough to hide future defects
- the slice adds behavior to a touched known debt surface without closing that debt in the same change

Loopback target: per-task implementation

## Integration Validation Gate

Reject advancement when:
- slice-local validations pass but end-to-end behavior fails
- contracts disagree across modules
- state transitions or migrations break in realistic flows
- operational signals reveal unhandled failure paths

Loopback target:
- per-task implementation for local defects
- task planning for dependency defects
- design or spec for architectural defects

## Final Code Review Gate

Reject finishing when:
- the full change set has hidden coupling or design drift
- residual risk is not documented
- rollout, rollback, or migration notes are missing where required
- branch state is not review-ready
- the diff touched a known structural-debt surface and that debt still remains as a follow-up instead of being closed in this pipeline

Loopback target:
- the most local stage that can honestly fix the issue
