# Documentation Rules

## Purpose

This file defines documentation ownership, placement, pre-edit reading, new path registration, and validation rules for this project.

## No Circular References

- This file is the documentation rule source.
- `docs/README.md` is the documentation navigation index.
- Project `AGENTS.md` may route agents to this file and `docs/README.md`, but must not duplicate the full rules.
- Do not make two documentation files require each other before either can be edited.
- When editing this file itself, read the current file, then inspect affected indexes and references as needed.
- When editing `docs/README.md` itself, read this file first, then inspect links and affected target documents; do not treat `docs/README.md` as a prerequisite for editing itself.

## Pre-Edit Reading Protocol

Before creating, moving, deleting, or editing documentation:

1. Read this file, unless this file does not exist yet and the current task is creating it.
2. Read the nearest existing index for the target area. Use `docs/README.md` for repository-wide navigation when it exists; otherwise inspect the nearest parent directory and project `AGENTS.md`.
3. Read the current source of truth for the fact being changed, such as code, contracts, configuration, operations docs, or review records.
4. Search references to any path being added, moved, renamed, or deleted.

Before writing, classify the change as current fact, draft/design, implementation plan, review evidence, operations guide, external reference, agent feedback, or harness asset. Then identify the owning source of truth, nearest index, stale references, and required mechanical checks.

## New Path Registration

When adding a durable documentation directory, active entry point, or new documentation category, register it in the same change:

- This file.
- `docs/README.md` or the nearest parent index.
- Project `AGENTS.md` when agent routing changes.
- Mechanical checks such as `scripts/check-consistency.sh`, `scripts/check-docs-consistency.py`, CI, or an equivalent guard when the path must remain stable.

Registration shape:

```md
Path: `...`
Type: current fact / draft / plan / review evidence / operations / external reference / feedback / harness asset
Owner: ...
Active entry: yes / no
Allowed:
Forbidden:
Read before editing:
Validation:
```

Do not create a new durable path for current-task scratch state. Use the project scratch or harness location if one exists, then either remove it or promote durable knowledge to the owning source of truth.

## Standard Paths

- `docs/README.md`: documentation entry point, reading order, current source-of-truth map, and stale-document rules.
- `docs/requirements/`: product requirements, scope definitions, acceptance criteria, user stories, constraints, and requirement gap analyses.
- `docs/contracts/`: API contracts, DTO/event schemas, protocol specs, payload examples, and compatibility notes.
- `docs/spec/`: implementation-ready feature specifications written before planning.
- `docs/design/`: product design, UI/UX design, design-system notes, interaction flows, and visual decisions that are not yet current architecture facts.
- `docs/todo/`: actionable task lists, backlog breakdowns, cleanup queues, and unresolved work items.
- `docs/architecture/`: current system design, module boundaries, data flow, dependency decisions, ADR-style notes, diagrams, and long-lived technical constraints.
- `docs/plan/`: implementation plans, migration plans, rollout plans, staged refactor plans, and temporary execution plans.
- `docs/operations/`: runbooks, local operations, deployment notes, incident procedures, maintenance commands, and operational verification records.
- `docs/reviews/`: code reviews, architecture reviews, UI/UX reviews, audit snapshots, and review findings.
- `docs/reports/`: status reports, gap reports, implementation summaries, investigation reports, and time-boxed analysis outputs.
- `docs/improvements/`: agent-discovered improvement items and promotion status, not general task backlog.
- `docs/refs/`: external references, research notes, source material, vendor docs summaries, papers, and copied context that should not be treated as project decisions by itself.

## Source-Of-Truth Rules

- Treat `docs/architecture/` and `docs/contracts/` as current technical facts only after they match code and tests.
- Treat `docs/spec/` as planning input.
- Treat `docs/plan/`, `docs/reviews/`, and `docs/reports/` as process history unless promoted into architecture, contracts, or requirements.
- Keep `docs/README.md` as the index that explains which documents are current and which are historical.
- If a draft design is adopted, move the stable conclusion into `docs/architecture/` or `docs/contracts/`.
- If a document is superseded, mark it as superseded or remove it from active indexes.

## Validation

When documentation paths, indexes, or facts change:

- Run the project documentation consistency check if it exists.
- Search for stale references to renamed, moved, or deleted paths.
- Verify new links from the nearest parent index.
- If no mechanical check exists for a new stable path, either add one or explicitly state why it is not needed.
