---
name: backend-engineer
description: Use when implementing, refactoring, or reviewing backend code that changes APIs, services, persistence, jobs, queues, caching, integrations, runtime operations, data ownership, or test strategy, especially when correctness and operational safety matter more than framework convenience.
---

# Backend Engineer

Use this skill for backend work before layering language-specific skills such as `go-backend`.

Language-specific backend skills should extend this skill rather than replace it. Put backend-wide ownership, persistence, migration, and operational rules here; keep language-specific skills focused on the extra constraints their language or framework introduces.

## Core Rules

- Keep ownership explicit across handler, service, repository, worker, and integration boundaries. Do not let convenience helpers blur who validates input, owns retries, owns transactions, or owns persistence-side effects.
- Prefer one durable owner for each backend concern: schema changes, contract normalization, permission checks, idempotency, and side-effect orchestration should not be silently duplicated across layers.
- Treat database schema evolution as an explicit deployment concern, not an incidental runtime effect.
- For backend behavior changes, load `test-driven-development` before production code and prove the behavior with a failing test first.
- When a backend branch enforces a non-obvious business rule, state transition, or failure path, keep the comment directly above that branch and explain the trigger, business intent, and side effect instead of paraphrasing the code.

## Backend TDD Boundaries

- TDD is required for handlers, services/use cases, repositories, jobs, queues, caching behavior, integrations, permission checks, idempotency, transactions, and persistence rules.
- Put the red test at the boundary that owns the behavior. Handler tests should prove request/response contracts, service tests should prove business decisions, repository tests should prove persistence contracts, and job/worker tests should prove retry and side-effect orchestration.
- Do not duplicate the same behavior across handler, service, repository, and end-to-end tests unless each level proves a distinct contract.
- Keep backend test growth under control during the refactor step: split test files by use case or behavior, extract builders/fixtures after duplication appears, and delete or merge tests with the same failure signal.
- Use table tests for multiple examples of one rule, not for unrelated behaviors.
- Prefer clear behavior assertions over broad mock-call verification. Mock external boundaries when needed, but do not test only the mock.
- Schema setup for integration tests belongs in migrations, shared test helpers, or ephemeral fixtures. Do not add ad hoc schema mutation to production runtime paths to satisfy tests.

## Schema Ownership

- Formal runtime paths such as app startup, container entrypoints, dev CLIs, import jobs, background workers, and long-lived services must not introduce a second schema owner through ORM schema mutators or framework auto-sync features.
- Production and shared development schema changes should live in versioned migrations that can be reviewed, replayed, rolled back, and audited.
- Auto-sync helpers such as GORM `AutoMigrate` are acceptable in test helpers, ephemeral integration fixtures, and temporary local test databases when those helpers are not the source of truth for the real schema.
- If a code path needs a missing table or column to exist before it can boot, the fix is usually to add or repair the formal migration path, not to patch the runtime path with schema creation logic.

## Operational Bias

- Prefer startup failures with actionable migration errors over silent boot-time schema drift.
- Keep rollback and disaster recovery credible: a reviewer should be able to tell which migration changed the schema and how to replay or revert it.
- When an ORM convenience feature conflicts with explicit ownership, choose explicit ownership.
