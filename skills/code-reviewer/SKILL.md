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
14. Check for dead API surface: methods, wrappers, interface members, repository functions, ports, or compatibility paths with no clear owner and no production call path are review findings even if tests or string guards still reference them.

## Workflow

1. Read the actual diff first. Before loading references, establish the review context: read module boundaries, existing contracts, recent architecture decisions, test coverage baseline, and any project-specific review requirements (AGENTS.md, CLAUDE.md, docs/architecture/). A checklist without context yields shallow findings.
2. Load only the relevant reference files from `references/` based on what the diff touches.
3. Identify the dominant risk area: correctness, architecture, security, test strategy, engineering standards, or review communication.
4. Check changed code in local context, not line-by-line in isolation.
5. For frontend UI diffs that add or change visible copy, read `references/frontend-ui-copy-review.md`.
6. For frontend architecture reviews, read `frontend/architecture-review.md`.
7. Check whether the diff grows already oversized files, components, services, or functions in a way that increases ownership ambiguity, hides state flow, or makes tests weaker than the behavior.
8. If the diff touches a known oversized or owner-mixed surface at all, apply Guardrail 13: explicitly decide whether the change closes that debt, and block the review if it does not.
9. For frontend or backend diffs, load `references/technical-risk-checks.md` for the surface-specific scrutiny points (frontend: route views, SFCs, composables, stores, async handlers, lifecycle cleanup; backend: handlers, services, repositories, transactions, background work, config, integrations, DTO/API contracts).
10. Ask "how would a senior maintainer implement this after reading the surrounding code?" Compare against the submitted diff for ownership, simplicity, error handling, contracts, tests, and future extension cost.
11. Search for ownership and call-path drift. For changed or newly obsolete methods, use text search to distinguish production calls from tests, stubs, generated guards, and string-based architecture tests. If a method only has test/guard references and no production owner, flag it for removal.
12. Write findings in priority order with impact, fix direction, and whether the finding blocks completion.
13. For material findings, state the expected re-review or re-validation evidence.
14. Keep subjective preferences out of blocker comments unless they hide a real maintenance or correctness cost.
15. For independent reviews that gate non-trivial work, archive the review result using the Review Archive policy below.

## When Used As The code-workflow Gate Reviewer

When this skill is invoked as the final review gate for `code-workflow`:

1. Treat the review as independent gate review, not implementation help.
2. Assume the implementation context's own `completion-full` result is only self-check evidence.
3. Use the repository's architecture docs, contracts, AGENTS rules, and local review commands as the baseline.
4. Read the implementation plan and the executed validation evidence before judging merge-readiness.
5. If project-local architecture or workflow checks exist, decide whether the existing evidence is sufficient or whether the narrowest relevant subset must be rerun.
6. Return a clear gate verdict and identify material findings that must be fixed before completion.
7. Same-context review does not satisfy this gate; if you detect that the review is not independent, state that limitation explicitly.

## Checklist Mode

For complex or high-risk reviews, use the dimension-by-dimension checklist enforcer to prevent quality blind spots.

**Trigger conditions** (any of):
- Review involves concurrency, resource management, API compatibility, or initialization logic
- Diff size >300 lines or >5 files
- User explicitly requests comprehensive review
- Gate review for production-bound changes

**How to invoke**:
Use the Workflow tool with `workflows/checklist-review.js`:
```javascript
workflow({ scriptPath: '~/.agents/skills/code-reviewer/workflows/checklist-review.js', args: { diffSource: 'git diff HEAD~1' } })
```

**What it does**:
1. Loads `review-checklist.yaml` with 8 mandatory quality dimensions
2. For each dimension, spawns an agent that checks ALL items in that dimension against the diff
3. Forces a verdict (pass / findings / N/A) for every dimension—no skipping allowed
4. Consolidates findings by severity and produces a gate verdict

**Output**:
- `gate_verdict`: pass / pass_with_minor_issues / pass_with_major_issues / blocked
- `dimensions_checked`: count of dimensions reviewed
- `all_findings`: sorted by severity (Blocker → Major → Minor → Nit), each with dimension, location, issue, impact, suggestion

Use checklist mode when thoroughness matters more than speed, or when the review must satisfy compliance/audit requirements.

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
- `references/production-readiness-checks.md`
  Read when the change affects initialization order, observability, backward compatibility, configuration validation, or user-facing error handling.
- `references/test-strategy-review.md`
  Read when tests changed, are missing, or look suspiciously shallow.
- `references/security-review-checklist.md`
  Read when the change affects authentication, authorization, input handling, secrets management, session handling, or any trust boundary. Follows OWASP Top 10 Proactive Controls with detailed checks for injection, XSS, CSRF, encryption, access control, logging, and secure error handling.
- `references/frontend-ui-copy-review.md`
  Read when reviewing frontend UI copy, helper prose, dashboard text, empty states, or page/workspace descriptions.
- `frontend/architecture-review.md`
  Read when the review target is frontend architecture, ownership, slice boundaries, state flow, or UI-domain decomposition. This file is a standalone, paste-ready prompt with its own `P0/P1/P2` scale; when driving the review from this skill, map its levels back to the `Blocker/Major/Minor/Nit` scale below instead of mixing both.
- `references/review-communication.md`
  Read before writing review feedback so comments stay precise, constructive, and properly prioritized.
- `review-checklist.yaml`
  Structured checklist of 8 mandatory quality dimensions (resource lifecycle, boundary conditions, concurrency, initialization, observability, compatibility, configuration, error UX). Used by checklist mode to enforce dimension-by-dimension coverage.
- `workflows/checklist-review.js`
  Workflow script that enforces checklist-driven review: loads the checklist, checks each dimension against the diff, and produces a gate verdict with consolidated findings. Invoke via Workflow tool for comprehensive reviews.
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
- Dead code risks include no-owner methods, unused compatibility wrappers, interface members with no production callers, and old fallback paths left behind after a replacement path lands.
- Senior implementation assessment: whether the current approach is the simplest maintainable implementation for the requirement, and if not, the concrete lower-risk alternative.
- Archive path for independent non-trivial review gates, or an explicit reason no archive was created.
- If no material findings are discovered, say so explicitly and note residual risk, assumptions, or validation gaps.
- Do not block on subjective taste when the change is a net improvement and carries no meaningful risk.

## Source Basis

- Google Engineering Practices: code review should evaluate design, functionality, complexity, tests, naming, comments, style, and documentation.
- OWASP Secure Code Review guidance: manual review is valuable for business logic, data flow, trust boundaries, authorization, race conditions, configuration, and context-specific vulnerabilities that tools may miss.
