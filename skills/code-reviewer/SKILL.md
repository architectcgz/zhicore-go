---
name: code-reviewer
description: Use when reviewing code changes, pull requests, patches, or implementation plans where correctness, regressions, architecture impact, security, test quality, and review communication all matter more than writing new code.
---

# Code Reviewer

Review for risk reduction, not for style theater.

## Use When

- Reviewing a patch, pull request, commit range, or local diff
- Checking whether an implementation is safe to merge
- Looking for correctness bugs, regressions, security gaps, performance traps, or test blind spots
- The user asks for review feedback, review comments, or a merge-readiness assessment

## Do Not Use

- Pure implementation work with no review objective
- Broad brainstorming without a concrete diff, plan, or target to assess

## Core Guardrails

1. Start with correctness and regression risk before style or preference.
2. Distinguish blocker findings from suggestions and explain why.
3. Review the system impact, not just the changed lines in isolation.
4. Treat security, permission, data integrity, and concurrency issues as first-class review concerns.
5. Evaluate tests for coverage quality, not just test-file existence.
6. Prefer automation for repetitive nits instead of recurring human comments.
7. Separate verified findings from inference, and state assumptions when local evidence is incomplete.
8. For leader or pipeline gated work, review the classification instead of redefining the policy: agree, recommend upgrade to non-trivial, or ask leader to decide when evidence is incomplete.
9. For non-trivial review gates, return a concrete gate verdict and identify which findings are material before completion.
10. Treat oversized files, mixed ownership, and responsibility pileups as review risks, not style nits, when they make future changes hard to reason about or test.
11. Review from a senior engineer's implementation perspective: ask whether the same requirement could be implemented with clearer ownership, simpler control flow, stronger contracts, smaller blast radius, and better long-term maintainability.
12. Do not turn senior judgment into speculative rewrites. Recommend a more elegant implementation only when it reduces real risk, removes meaningful complexity, improves testability, or aligns better with existing project architecture.
13. If the diff touches a file, component, service, or module that is already tracked as structural debt, oversized ownership, or required decomposition, unresolved debt in that touched surface is a material finding, not residual risk.

## Workflow

1. Read the actual diff first, then load only the relevant reference files from `references/`.
2. Identify the dominant risk area: correctness, architecture, security, test strategy, engineering standards, or review communication.
3. Check changed code in local context, not line-by-line in isolation.
4. For frontend UI diffs that add or change visible copy, read `references/frontend-ui-copy-review.md`.
5. For frontend architecture reviews, read `frontend/architecture-review.md`.
6. Check whether the diff grows already oversized files, components, services, or functions in a way that increases ownership ambiguity, hides state flow, or makes tests weaker than the behavior.
7. If the diff touches a known oversized or owner-mixed surface at all, explicitly decide whether the change closes that debt. If not, block the review instead of recording it only as follow-up debt.
8. For frontend diffs, explicitly check route views, SFCs, composables, stores, forms, dialogs, async handlers, and visible workflow ownership.
9. For backend diffs, explicitly check handlers, services, repositories, transactions, background work, config, integrations, and DTO or API contracts.
10. Ask "how would a senior maintainer implement this after reading the surrounding code?" Compare against the submitted diff for ownership, simplicity, error handling, contracts, tests, and future extension cost.
11. Write findings in priority order with impact, fix direction, and whether the finding blocks completion.
12. For material findings, state the expected re-review or re-validation evidence.
13. Keep subjective preferences out of blocker comments unless they hide a real maintenance or correctness cost.
14. For independent reviews that gate non-trivial work, archive the review result using the Review Archive policy below.

## When Used As The code-workflow Gate Reviewer

When this skill is invoked as the final review gate for `code-workflow`:

1. Treat the review as independent gate review, not implementation help.
2. Assume the implementation context's own `completion-full` result is only self-check evidence.
3. Use the repository's architecture docs, contracts, AGENTS rules, and local review commands as the baseline.
4. Read the implementation plan and the executed validation evidence before judging merge-readiness.
5. If project-local architecture or workflow checks exist, decide whether the existing evidence is sufficient or whether the narrowest relevant subset must be rerun.
6. Return a clear gate verdict and identify material findings that must be fixed before completion.
7. Same-context review does not satisfy this gate; if you detect that the review is not independent, state that limitation explicitly.

## Review Archive

Default location for review evidence is the target repository:

```text
docs/reviews/{frontend|backend|architecture|security|general}/YYYY-MM-DD-{scope}-review-{short-topic}.md
```

Use the category that matches the dominant risk surface. If no category fits, use `general`. Create the category directory when needed.

Only use the global fallback for work that is not tied to a project repository, such as global agent, skill, hook, or personal tooling changes:

```text
/home/azhi/.codex/reviews/{frontend|backend|architecture|security|general}/YYYY-MM-DD-{scope}-review-{short-topic}.md
```

Review archive files must include:

- Review target: repository, branch or worktree, base/head or diff source, and files reviewed
- Classification check: agree with leader/pipeline, recommend upgrade, or needs leader decision
- Gate verdict: pass, pass with minor issues, or blocked
- Findings: ordered by severity, with file/line references when available
- Material findings: required fixes before completion
- Senior implementation assessment: current approach and lower-risk alternative when relevant
- Required re-validation: commands or behavior paths to re-check after fixes
- Residual risk: assumptions, missing evidence, or intentionally deferred issues
- Touched known-debt status: whether the diff touched any previously tracked structural-debt surface and, if yes, whether the debt was fully closed or blocked

## Reference Map

- `references/review-workflow.md`
  Read for review order, scope control, prioritization, and merge-readiness judgment.
- `references/technical-risk-checks.md`
  Read for correctness, performance, architecture, concurrency, and security review points.
- `references/engineering-standards.md`
  Read for maintainability, naming, complexity, logging, observability, and hard-coding checks.
- `references/operational-readiness.md`
  Read when the change touches config, rollout, migrations, retries, timeouts, feature flags, or rollback safety.
- `references/test-strategy-review.md`
  Read when tests changed, are missing, or look suspiciously shallow.
- `references/frontend-ui-copy-review.md`
  Read when reviewing frontend UI copy, helper prose, dashboard text, empty states, or page/workspace descriptions.
- `frontend/architecture-review.md`
  Read when the review target is frontend architecture, ownership, slice boundaries, state flow, or UI-domain decomposition.
- `references/review-communication.md`
  Read before writing review feedback so comments stay precise, constructive, and properly prioritized.
- `references/ctf-current-review-status-checks.md`
  Read when reviewing CTF repo changes or review documents that may reintroduce recently fixed backend/frontend review debt.
- `~/.agents/harness/workflows/code-workflow/independent-review-protocol.md`
  Read when the review is acting as the final `code-workflow` gate for a non-trivial task.

## Output Expectations

- Findings come first, ordered by severity.
- Each finding states location, risk, and reasoning.
- Suggestions are clearly separated from blockers.
- Classification check: agree with leader/pipeline, recommend upgrade, or needs leader decision.
- Gate verdict for non-trivial work: pass, pass with minor issues, or blocked.
- Material findings list with required fix and re-validation direction.
- Code quality risks include ownership ambiguity, oversized files, weak decomposition, hidden state flow, insufficient tests for the new shape, and hard-to-review complexity.
- Do not downgrade unresolved debt in a touched known oversized or owner-mixed surface into a suggestion or residual-risk note; treat it as a blocker until the touched surface is actually converged.
- Senior implementation assessment: whether the current approach is the simplest maintainable implementation for the requirement, and if not, the concrete lower-risk alternative.
- Archive path for independent non-trivial review gates, or an explicit reason no archive was created.
- If no material findings are discovered, say so explicitly and note residual risk, assumptions, or validation gaps.
- Do not block on subjective taste when the change is a net improvement and carries no meaningful risk.

## Source Basis

- Google Engineering Practices: code review should evaluate design, functionality, complexity, tests, naming, comments, style, and documentation.
- OWASP Secure Code Review guidance: manual review is valuable for business logic, data flow, trust boundaries, authorization, race conditions, configuration, and context-specific vulnerabilities that tools may miss.
