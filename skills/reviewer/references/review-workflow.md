# Review Workflow

Read this file when starting a review or deciding how strong the review should be.

## Review order

1. Understand the claimed intent and approval question.
2. Identify the artifact type and read the primary review target.
3. Read the surrounding code, documents, contracts, decisions, or evidence that owns the claimed outcome.
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

- Do not review only touched lines or isolated paragraphs if the artifact clearly affects surrounding invariants.
- Do not expand into unrelated cleanup or redesign unless it directly affects the review risk.
- For code reviews, if the diff exceeds 400 lines or touches more than 8 files without a clear refactoring or migration boundary, ask whether it can be split into smaller, logically independent units before proceeding with deep review.
- For plan or design reviews, require a parent/child split when one artifact contains multiple independent delivery, owner, review, or rollback boundaries.
- If the artifact is too large to split, call that out and focus on the highest-risk surfaces first.

## Evidence discipline

- Distinguish what is directly verified in code from what is inferred from surrounding context.
- If you cannot verify a behavior, say what evidence is missing instead of presenting a guess as a fact.
- Prefer artifact-local, owner-local, and evidence-based reasoning over abstract review slogans.
- When the risk depends on runtime behavior or deployment topology, note the assumption explicitly.

## Approval judgment

- Ask whether the artifact is safe to merge, approve, or execute for its stated purpose.
- Do not let perfect become the enemy of a safe improvement.
- Do not use personal taste as a blocker when the artifact is correct, understandable, and maintainable enough.

## Default closing questions

- Does the artifact actually deliver the claimed outcome
- What can go wrong at boundaries, during execution, under failure, or under concurrency
- If this were approved today, what would be the most likely delivery or production problem
- Is the proposed evidence proving the right thing
- Is the review comment helping the author improve the artifact or just expressing reviewer preference
