# Plan Template

Use this structure when converting the spec into reviewable tasks.

If the target repository uses `code-workflow`, do not save this file's generic structure as the repository implementation plan. Use the repository's managed implementation plan skeleton, normally `harness/templates/implementation-plan-skeleton.md`, and treat this template as the task-slicing reference that fills its `Execution Slices`, `Validation Plan`, and `Review focus` sections.

## Plan Summary
- Objective
- Non-goals
- Source architecture or design docs
- Dependency order
- Expected specialist skills

## Task 1
- Goal
- Touched modules or boundaries
- Dependencies
- Validation
- Review focus
- Risk notes

## Task 2
- Goal
- Touched modules or boundaries
- Dependencies
- Validation
- Review focus
- Risk notes

Repeat as needed.

## Integration Checks
- Which paths must be validated after all tasks land
- Which contracts or state transitions are most likely to fail at integration time

## Rollback / Recovery Notes
- What can be reverted independently
- Any migration, data, config, or rollout recovery concerns

## Residual Risks
- Known risks not fully eliminated by the plan
