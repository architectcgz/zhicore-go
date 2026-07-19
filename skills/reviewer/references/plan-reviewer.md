# Implementation Plan Reviewer

Use this reference when the review target is an implementation plan, migration plan, rollout plan, refactor plan, or multi-stage technical program.

## Core Rule

Review the plan as an executable dependency graph, not as prose. A plan can contain correct technical decisions and still be unsafe to execute because constraints activate too late, the same artifact has multiple owners, external gates are unresolved, or future tasks are specified against a code shape that earlier tasks will replace.

## Review Sequence

1. Confirm the exact repository, worktree, branch, plan path, base commit, and referenced spec.
2. Confirm the formal plan owns a dedicated package directory with `README.md` as its stable entry, unless project rules explicitly define a stronger convention.
3. Classify the artifact as either:
   - a single executable plan with one cohesive delivery and review boundary; or
   - a parent program that must route to independently executable child plans.
4. Build the four matrices below before judging task wording.
5. Verify target-state completeness, acceptance evidence, rollback, and final negative checks.
6. Return findings by severity and state whether the artifact is executable as written.

## Matrix 1: Delivery Boundary

Create a compact map:

| Slice | Primary owner | Independently testable output | Commit boundary | Review boundary | Depends on |
| --- | --- | --- | --- | --- | --- |
| Example | One service or shared capability | Observable working state | One to three focused commits | Named review gate | Prior slice or external gate |

Treat the plan as a parent program when it contains multiple independently deployable services, multiple subsystem owners, a pilot followed by later migrations, or several mandatory independent reviews. Do not approve a program-sized artifact as one agent execution context merely because it has numbered tasks.

Treat a new group of prefixed sibling files placed directly in the shared formal-plan root as an ownership defect. Require one package directory containing `README.md`, child plans, and any plan-local references or assets. A flat prefix convention does not provide an archive, index, or lifecycle boundary.

Strong signals that child plans are required:

- more than one service or independently deployable module;
- shared foundation plus multiple adopters;
- a pilot whose result can change later implementation details;
- unresolved external baselines or cross-branch dependencies;
- multiple review/rollback boundaries;
- late tasks whose exact file lists depend on abstractions created by early tasks.

## Matrix 2: Constraint Activation Timing

For every architecture check, lint rule, migration guard, feature flag, compatibility check, or policy, record:

| Constraint | First phase that can violate it | Phase that enables the guard | Strict phase | Gap |
| --- | --- | --- | --- | --- |
| GORM import boundary | First service migration | After pilot | Final convergence | Guard is late if enabled after all migrations |

The guard must become effective before, or at the start of, the first phase it is meant to constrain. A check added only during final integration is evidence, not a migration guard.

For staged migrations, prefer two modes:

1. Migration mode: reject new violations and allow only an explicit shrinking legacy allowlist.
2. Strict mode: remove the allowlist and prove the legacy path is gone.

Raise a material finding when:

- implementation begins before the rule that prevents known drift exists;
- the plan relies on reviewers to remember a rule that could already be mechanical;
- a final check discovers violations only after several dependent slices were built on them;
- the allowlist has no owner, shrink point, or removal gate.

## Matrix 3: Artifact Ownership And Repeated Modification

Map every file, document, schema, plan, config owner, or shared API to the tasks that modify it:

| Artifact | Tasks | Intended owner | Required intermediate states | Final owner |
| --- | --- | --- | --- | --- |
| Future service plans | 1 and 11 | Governance slice | None stated | Ambiguous |

Repeated modification is not automatically wrong. It is a finding when the same artifact appears in separated tasks without an explicit staged state, dependency reason, or single final owner.

Ask:

- Why is this artifact changed twice?
- Is the first edit a required, testable intermediate state?
- Can both edits be owned by one slice?
- Will an executor have to rediscover which version is authoritative?
- Does the later task undo, supersede, or duplicate the earlier task?

Flag these patterns:

- the same active plans or architecture docs are updated during both initialization and final governance;
- the same config is renamed in multiple service tasks without one shared migration owner;
- a shared API is changed once for a pilot and again for later adopters without a compatibility contract;
- a final cleanup task repeats removals already required by every service slice.

## Matrix 4: Gate And Evidence Ownership

For each precondition and acceptance rule, record:

| Gate or acceptance | Owner | Evidence | Required before | Failure action |
| --- | --- | --- | --- | --- |
| Baseline runtime changes merged | Preflight slice | Exact commit and validation | Shared foundation | Block execution |

Reject unresolved external state hidden as ordinary checkboxes. A dependency on uncommitted work, another worktree, another repository, unpublished contract, or undecided rollout boundary needs a named preflight gate with exact evidence and a blocked status until satisfied.

## Progressive Elaboration Check

When a pilot or shared foundation can alter constructors, models, adapters, runtime wiring, or validation strategy, do not require late child plans to freeze exact file-level steps before the pilot is reviewed.

Require the parent plan to preserve late-stage service acceptance and target state, then require each child plan to be refreshed against the actual base commit before execution.

Treat detailed late-stage steps as stale-risk findings when:

- they depend on APIs not yet implemented;
- the plan itself says a pilot review gates later work;
- early tasks can rename or split the files listed by late tasks;
- later plans have no rebase/replan trigger.

## Context And Tracking Check

Count or estimate:

- independent owners and services;
- executable tasks and checkboxes;
- files touched per slice;
- repeated acceptance rules;
- external facts the executor must keep active;
- review and rollback boundaries.

Do not use a universal line-count blocker. Use size as evidence that the active context contains several delivery boundaries. Recommend a parent index plus child plans when an executor must repeatedly reload unrelated service details.

The parent should track phase status, dependency, commit, review evidence, and next gate. Child plans should own executable checkboxes, exact file deltas, tests, and handoff notes.

## Counterexample: Program Disguised As One Plan

An all-services ORM migration plan contained a shared database foundation, a pilot, three later service migrations, future-service plan updates, mechanical checks, and final integration in one file.

The review found:

- mechanical boundary checks were scheduled after all service migrations, so they could only detect drift at the end;
- the same future implementation-plan files were modified in both an early architecture task and a late governance task, with no necessary intermediate state;
- execution depended on uncommitted runtime baseline changes, but the dependency was represented only by checkboxes rather than a named preflight gate and base commit;
- exact file lists for late services were frozen before the pilot established the final shared API;
- hundreds of checkboxes mixed parent-program status with child execution state.

The correct response is not to delete acceptance detail. Reclassify the document as a parent program, move executable detail into child plans, activate migration-mode guards immediately after the pilot, consolidate artifact ownership, and keep a final strict-convergence gate.

## Severity Guidance

Treat as `Blocker` when:

- unresolved external state means the plan cannot start from a reproducible base;
- the artifact is presented as directly executable but requires mutually independent programs or incompatible intermediate states;
- task ordering can create an unsafe production state with no rollback or gate.

Treat as `Major` when:

- a new formal plan or parent/child plan family is stored as flat prefixed files in the shared plan root without a project-specific reason;
- a mechanical guard activates after the risky work it should constrain;
- the same artifact has multiple task owners without an explicit staged contract;
- late plans are likely to become stale after a pilot or shared API change;
- parent status and child execution checkboxes are mixed into one context large enough to hide dependencies.

Treat as `Minor` when:

- duplication or oversized sections increase reading cost but ownership and execution remain unambiguous;
- handoff fields, replan triggers, or traceability can be clearer without changing task boundaries.

## Required Review Output

For plan reviews, include:

- artifact classification: executable plan or parent program;
- gate verdict: pass, pass with minor issues, or blocked;
- constraint activation findings;
- repeated artifact ownership findings;
- unresolved preflight dependencies;
- progressive-elaboration and stale-detail risks;
- recommended parent/child split when applicable;
- re-review evidence required after restructuring.
