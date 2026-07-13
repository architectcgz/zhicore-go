# Review Workflow

Read this file when starting a review or deciding how strong the review should be.

## Review order

1. Understand the claimed intent of the change.
2. Read the diff.
3. Read the surrounding code that owns the changed behavior.
4. Identify the primary risk class before expanding into secondary concerns.
5. Write findings in severity order.
6. End with residual risk, missing validation, or a merge-readiness statement.

## Priority model

- `Blocker`
  The change is unsafe to merge because it can cause incorrect behavior, regression, data loss, security exposure, crash, or a serious maintainability trap.
- `Major`
  The change is mergeable only with caution; the issue is not instantly catastrophic but carries meaningful user or operational risk.
- `Minor`
  The issue should be fixed soon but does not materially change merge safety.
- `Nit`
  Preference or readability polish with low practical impact.

## Scope discipline

- Do not review only the touched lines if the change clearly affects surrounding invariants.
- Do not expand into unrelated code cleanup unless it directly affects the review risk.
- If the diff exceeds 400 lines or touches more than 8 files without a clear refactoring or migration boundary, ask whether the change can be split into smaller, logically independent units before proceeding with deep review. Large diffs invite shallow review and increase the risk of missing critical issues.
- If the diff is too large to split, call that out and focus on the highest-risk surfaces first.

## Evidence discipline

- Distinguish what is directly verified in code from what is inferred from surrounding context.
- If you cannot verify a behavior, say what evidence is missing instead of presenting a guess as a fact.
- Prefer file, function, and branch-local reasoning over abstract review slogans.
- When the risk depends on runtime behavior or deployment topology, note the assumption explicitly.

## Merge judgment

- Ask whether this change is strictly better than the current state.
- Do not let perfect become the enemy of a safe improvement.
- Do not use personal taste as a blocker when the code is already correct, understandable, and maintainable enough.

## Default closing questions

- Does the code really implement the intended behavior
- What can go wrong at boundaries, under failure, or under concurrency
- If this shipped today, what would be the most likely production problem
- Are the tests proving the right thing
- Is the review comment helping the author improve the code or just expressing reviewer preference
