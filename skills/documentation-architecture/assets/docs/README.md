# Documentation Index

This directory is the project documentation entry point. It points to current sources of truth and separates durable facts from process history.

## Reading Order

1. Read `docs/documentation-rules.md` for documentation ownership, placement, and path registration rules.
2. Use this index to find the relevant current source of truth.
3. Read the nearest parent index for the target documentation area when it exists.
4. Verify facts against code, contracts, configuration, tests, or operations records before updating current documentation.

## Current Source-Of-Truth Map

- `docs/requirements/`: product requirements, scope, acceptance criteria, and constraints.
- `docs/contracts/`: API, event, data, and compatibility contracts.
- `docs/spec/`: implementation-ready feature specifications before planning.
- `docs/design/`: product and UX design that is not yet current architecture fact.
- `docs/architecture/`: current system design and long-lived technical constraints.
- `docs/operations/`: runbooks, deployment notes, maintenance commands, and operational verification.

## Process And History

- `docs/plan/`: implementation, migration, rollout, and refactor plans.
- `docs/reviews/`: review evidence and findings.
- `docs/reports/`: time-boxed reports, investigation summaries, and status snapshots.
- `docs/todo/`: actionable backlog, cleanup queues, and unresolved work items.
- `docs/improvements/`: agent-discovered improvement items and promotion status.
- `docs/refs/`: external references and research notes.

## Stale Document Rules

- A document linked from the current source-of-truth map must either match the code or clearly state that it is draft, superseded, or historical.
- When a process document becomes a stable decision, move the stable conclusion into requirements, contracts, architecture, or operations.
- When adding a durable path, register it in `docs/documentation-rules.md` and this index or the nearest parent index.
