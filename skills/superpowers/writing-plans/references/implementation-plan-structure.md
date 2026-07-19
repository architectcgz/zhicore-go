# Implementation Plan Structure Reference

Read this reference before writing a formal plan that spans multiple services, independently deliverable modules, a shared foundation plus adopters, a pilot migration, external preconditions, multiple review gates, or enough tasks that one executor would need unrelated context active at the same time.

## Core Principle

Optimize the active execution context, not the total amount of durable knowledge. Keep architecture decisions and complete acceptance evidence, but place each fact in the document that owns it.

Do not solve context drift by deleting hard requirements or merging several actions into vague steps. Reclassify a program-sized change into a parent program and executable child plans.

## Plan Package Ownership

Give every formal implementation plan its own directory:

```text
docs/plan/impl-plan/YYYY-MM-DD-<plan-slug>/
└── README.md
```

Use `README.md` as the stable plan entry. For a parent program, keep numbered child plans in the same package:

```text
docs/plan/impl-plan/YYYY-MM-DD-<program-slug>/
├── README.md
├── 01-preflight.md
├── 02-shared-foundation.md
└── 03-service-migration.md
```

Do not place a new family of prefixed sibling files directly in `docs/plan/impl-plan/`. Filename prefixes are not an ownership boundary: they make indexing, archiving, local references, plan-local assets, and plan-scoped validation harder to maintain.

Existing flat historical plans may remain until they are actively restructured. New formal plans and plans being structurally rewritten must use a package directory unless the target project explicitly defines a stronger convention.

## Choose The Document Shape

| Situation | Required shape |
| --- | --- |
| One cohesive owner, one independently testable output, one review boundary | Single executable implementation plan |
| Several services or deployable modules | Parent program plus one child plan per service or delivery boundary |
| Shared foundation followed by several adopters | Parent program, shared-foundation child, then adopter children |
| Pilot can change later APIs or file layout | Detail the pilot now; keep later service plans at acceptance level until pilot review |
| External branch, worktree, contract, or uncommitted baseline must land first | Named preflight child/gate with exact commit evidence |
| Migration guard must evolve from compatibility to strict mode | Early migration-mode guard child plus final strict-convergence gate |

Use size only as supporting evidence. A short document can still contain several programs; a long plan can remain executable when it has one owner and one review boundary.

## Parent Program Responsibilities

The parent is a router and status ledger. It owns:

- final target state and architectural direction;
- non-negotiable cross-cutting invariants;
- included and excluded services or subsystems;
- dependency graph and blocked preconditions;
- child-plan paths and current status;
- commit and review evidence for completed children;
- global rollback constraints and final convergence proof.

The parent does not own:

- per-file implementation steps;
- service-local test commands;
- hundreds of executable checkboxes;
- detailed code examples for every child;
- repeated copies of acceptance rules already owned by child plans or architecture documents.

### Parent Program Template

Save this content as the package `README.md`.

```markdown
# [Program Name] Implementation Program

**Goal:** [Complete target state]

**Architecture:** [Final initialization, owners, call paths, retained stable boundaries]

**Program status:** Blocked | Ready | In progress | Complete

## Preflight gates

| Gate | Owner | Evidence required | Status | Blocks |
| --- | --- | --- | --- | --- |
| [External baseline] | [child or team] | [exact commit, contract, review] | Blocked | Phase 1 |

## Cross-cutting invariants

| ID | Observable rule | Final proof |
| --- | --- | --- |
| INV-01 | [Specific positive or negative rule] | [Search/test/check] |

## Child plans and dependency order

| Phase | Child plan | Delivery owner | Depends on | Status | Commit | Review |
| --- | --- | --- | --- | --- | --- | --- |
| P0 | `[path]` | [owner] | - | Ready | - | - |

## Progressive planning rules

- [Which children must be refreshed after a pilot]
- [Which API/file-shape changes trigger replanning]

## Global rollback constraints

- [What may be reverted independently]
- [Unsafe legacy state that rollback must not recreate]

## Program completion definition

- [ ] Every child plan reached its exit gate.
- [ ] Final strict checks prove the old default path is gone.
- [ ] Final independent review has no material findings.
```

Keep parent checkboxes limited to program status and final gates. Child execution state belongs in child files.

## Executable Child Responsibilities

Each child owns one shared capability, one service, one independently testable subsystem, or one explicit governance/review boundary.

Each child must be executable without rereading every sibling plan.

### Child Plan Header

```markdown
# [Slice Name] Implementation Plan

> **For agentic workers:** [Required execution and testing skills]

**Goal:** [One independently testable result]

**Architecture:** [Local approach and interfaces]

**Parent program:** `[path]`

**Depends on:** `[prior child path]` at commit `[commit or pending evidence]`

**Base commit:** `[exact commit or BLOCKED]`

**Testing stance:** TDD | Mixed | No TDD

---
```

### Required Child Sections

```markdown
## Entry gate

| Requirement | Evidence | Failure action |
| --- | --- | --- |
| Prior child complete | Commit and review path | Stop; do not infer state |

## Context packet

**Must read:**
- `exact/path`

**Do not reload unless a replan trigger fires:**
- unrelated sibling plans

**Relevant invariant IDs:**
- `INV-01`: [repeat the exact observable rule needed by this child]

## Files

- Create: `exact/path`
- Modify: `exact/path`
- Delete: `exact/path`

## Acceptance checklist

- [ ] [Specific behavior, state, error, limit, owner, or negative-removal rule]

## Tasks

### Task 1: [Cohesive change]

- [ ] Write the focused failing test.
- [ ] Run it and confirm the expected failure.
- [ ] Implement the smallest behavior owned by this task.
- [ ] Run focused verification.
- [ ] Commit according to repository policy.

## Exit gate

| Evidence | Command or artifact | Expected result |
| --- | --- | --- |
| Focused tests | `exact command` | PASS |
| Negative search | `exact search` | No production matches |
| Review | `review path` | No material findings |

## Handoff record

- Commit:
- Validation:
- Review:
- Deviations:
- Remaining allowlist:
- Next child replan required: Yes | No

## Replan triggers

- Shared constructor/API differs from the parent assumption.
- The prior review changes owner or target state.
- Listed files moved or split.
- External contract or baseline commit changed.
```

## Decomposition Method

Build these maps before writing child steps.

### 1. Delivery Boundary Map

| Candidate slice | One owner | Independently testable | Independently reviewable | Safe rollback | Child plan? |
| --- | --- | --- | --- | --- | --- |

If a row has a different owner or review boundary, split it even if the code uses the same framework.

### 2. Artifact Ownership Map

| Artifact | Tasks that modify it | Single final owner | Required intermediate state | Action |
| --- | --- | --- | --- | --- |

Consolidate repeated edits unless the earlier state is necessary, observable, and explicitly removed later.

### 3. Constraint Activation Map

| Guard or policy | First risky phase | Enable phase | Strict phase | Action |
| --- | --- | --- | --- | --- |

Enable guards before the first adopter can violate them. For migrations, use a shrinking legacy allowlist and remove it at final convergence.

### 4. Acceptance Ownership Map

| Acceptance rule | Architecture fact | Child owner | Test/check owner | Final proof |
| --- | --- | --- | --- | --- |

Do not repeat the same rule as a parent checkbox, child checkbox, final checklist, and risk-table item. Keep the full rule in its owner and route to it from other documents.

## Progressive Elaboration

Freeze detail only as far as repository evidence supports.

Before a pilot:

- fully specify preflight, shared foundation, and pilot;
- preserve later services' target state, service-specific risks, and exit criteria;
- avoid freezing exact constructors and file lists that the pilot may change.

After pilot review:

- update shared architecture facts;
- enable migration-mode mechanical guards;
- refresh the next child plan against the actual base commit;
- execute one service or delivery boundary at a time.

Do not use progressive elaboration to hide known blockers. Known safety requirements remain explicit in the parent even when their file-level implementation is deferred to a child.

## Guardrail Timing

A final validation command is not automatically an effective migration guard.

Use two phases when legacy code cannot pass the final rule immediately:

| Phase | Behavior |
| --- | --- |
| Migration mode | Reject new violations; allow only named legacy paths; require the allowlist to shrink after every child |
| Strict mode | Remove the allowlist; reject every legacy path; prove the final target state |

Place migration mode immediately after the shared foundation or pilot that establishes the correct pattern.

## Context Budget Heuristics

Use these as review prompts, not universal blockers:

- A parent program should usually remain a compact router, often around 150-250 lines.
- A child plan should usually fit one service or one shared capability, often around 100-250 lines.
- A child should normally contain one review boundary and one to three focused commits.
- If one child requires unrelated runtime, repository, frontend, deployment, and governance contexts simultaneously, split by owner or delivery state.
- If late-child details depend on code not yet written, replace them with acceptance-level placeholders and a mandatory refresh gate.

## Good Structure Example

```text
docs/plan/impl-plan/2026-07-19-all-services-database-migration/
├── README.md
├── 01-preflight-and-architecture-baseline.md
├── 02-shared-database-foundation.md
├── 03-pilot-service-migration.md
├── 04-migration-mode-guardrails.md
├── 05-service-a-migration.md
├── 06-service-b-migration.md
├── 07-future-service-governance.md
└── 08-strict-convergence-and-final-review.md
```

The parent records status and dependencies. Each child records exact files, tests, commits, review evidence, and handoff state.

## Counterexample: Oversized All-Services ORM Plan

An approximately 1,000-line plan with more than 200 checkboxes combined:

- unresolved preflight work from another worktree;
- architecture documentation;
- a shared ORM foundation;
- a pilot service;
- three later service migrations;
- future service plan rewrites;
- mechanical checks introduced near the end;
- final repository-wide validation.

Specific structural failures:

1. The same future implementation-plan files were modified during both an early architecture task and a late governance task. No necessary intermediate state justified two owners.
2. Mechanical checks were added only after the risky service migrations. They could report drift but could not prevent it.
3. Detailed late-service file lists were frozen before the pilot established the final shared API.
4. Parent-program status, service execution steps, final review, and rollback checkboxes lived in one active context.
5. External uncommitted baseline changes appeared as ordinary checkboxes rather than a reproducible preflight gate and base commit.
6. Splitting the monolith into several long, prefixed files directly under `impl-plan/` still failed to create a plan ownership boundary; the parent and children needed one package directory with `README.md` as entry.

Do not fix this by deleting timeout, transaction, error, or runtime acceptance rules. Replace the monolith with a parent program, child plans, early migration-mode guards, consolidated document ownership, progressive plan refresh, and a final strict-convergence child.

## Final Structure Review

Before saving or handing off a formal plan, verify:

- [ ] The artifact is correctly classified as a single plan or parent program.
- [ ] The formal plan owns one package directory with `README.md` as its stable entry.
- [ ] Parent and child plans are co-located inside the package rather than grouped only by filename prefix.
- [ ] Every child has one delivery and review boundary.
- [ ] Every external dependency has a named gate and exact evidence.
- [ ] Mechanical constraints activate before the first risky adopter.
- [ ] Repeated file/document modifications have one owner or an explicit intermediate-state contract.
- [ ] Pilot-dependent later plans have refresh triggers instead of frozen guesses.
- [ ] Parent and child checkbox ownership is separate.
- [ ] Final positive and negative target-state proofs remain complete.
