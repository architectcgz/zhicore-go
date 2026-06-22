# Documentation Index

This directory is the documentation entry point for `zhicore-go`.

## Reading Order

1. Read `docs/documentation-rules.md` for documentation ownership and placement rules.
2. Use this index to find the relevant current source of truth.
3. Verify facts against Java source, Go code, contracts, configuration, tests, or operations records before updating current documentation.

## Current Sources

- `docs/architecture/`: current service boundary and data ownership decisions.
- `docs/contracts/`: cross-service contract ownership, compatibility, versioning, and change process.
- `docs/migration/`: Java-to-Go service migration map, order, and rollout notes.

## Process And History

- `docs/reviews/`: review evidence and findings.
- `docs/todos/debt/`: unresolved technical debt that should not be lost during migration.

## Deployment Notes

Deployment assets live under `deploy/`:

- `deploy/docker/`
- `deploy/k8s/`
