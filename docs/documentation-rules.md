# Documentation Rules

## Purpose

This file defines documentation ownership, placement, path registration, and validation rules for `zhicore-go`.

## Core Principles

- Documentation is durable project memory, not current-task scratch space.
- Each document should have one primary role: current fact, plan, review evidence, operations guide, external reference, or unresolved work.
- Entry-point documents route readers; they do not duplicate full source-of-truth content.
- Documentation changes must stay synchronized with code, contracts, scripts, tests, architecture boundaries, and migration status.

## No Circular References

- This file is the documentation rule source.
- `docs/README.md` is the documentation navigation index.
- Project `AGENTS.md` routes agents to both files but must not duplicate the full rules.

## Pre-Edit Reading

Before creating, moving, deleting, or editing documentation:

1. Read this file unless the current task is creating it.
2. Read `docs/README.md` or the nearest parent index.
3. Read the current source of truth for the fact being changed, such as Java source, Go code, contracts, configuration, tests, or operations docs.
4. Search references when adding, moving, renaming, or deleting paths.

## Registered Paths

Path: `docs/README.md`
Type: navigation index
Owner: repository documentation
Active entry: yes
Allowed: current documentation map, reading order, path routing
Forbidden: long-form implementation notes
Read before editing: this file
Validation: `bash scripts/check-structure.sh`

Path: `docs/migration/`
Type: migration plan and map
Owner: ZhiCore Java-to-Go migration
Active entry: yes
Allowed: service mapping, migration order, rollout notes, compatibility notes
Forbidden: unverified claims that a service has been migrated
Read before editing: Java source in `../zhicore-microservice`, Go module landing zone, this file
Validation: `bash scripts/check-structure.sh`

Path: `docs/architecture/`
Type: current architecture fact
Owner: ZhiCore Go service architecture
Active entry: yes
Allowed: service boundaries, data ownership, dependency direction, contract placement, long-lived technical constraints
Forbidden: temporary task notes, unreviewed implementation plans, review evidence
Read before editing: Java source in `../zhicore-microservice`, Go service modules, this file
Validation: `bash scripts/check-structure.sh`

Path: `docs/contracts/`
Type: current contract governance
Owner: ZhiCore Go cross-service contracts
Active entry: yes
Allowed: contract ownership, compatibility rules, versioning policy, change flow, rollout constraints
Forbidden: service-private DTO details, temporary migration notes, review evidence
Read before editing: `docs/architecture/service-boundaries.md`, affected `libs/contracts/...`, affected `services/<service>/api/`, this file
Validation: `bash scripts/check-structure.sh`

Path: `docs/reviews/`
Type: review evidence
Owner: implementation and architecture review process
Active entry: yes
Allowed: review findings, review rounds, validation notes
Forbidden: current architecture facts that are not promoted into source-of-truth docs
Read before editing: reviewed diff or commit, relevant code, this file
Validation: link and path checks by inspection; run `bash scripts/check-structure.sh` when paths change

Path: `docs/todos/debt/`
Type: unresolved debt tracking
Owner: migration debt management
Active entry: yes
Allowed: unresolved technical debt with owner, impact, and exit condition
Forbidden: generic task backlog or scratch notes
Read before editing: nearest debt index and source code or review that created the debt
Validation: `bash scripts/check-structure.sh`

## Validation

When documentation paths, indexes, or facts change:

- Run `bash scripts/check-structure.sh`.
- Search for stale references to renamed, moved, or deleted paths.
- Verify links from the nearest parent index.
