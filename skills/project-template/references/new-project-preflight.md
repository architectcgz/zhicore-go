# New Project Preflight

Use this checklist before scaffolding a new project or before writing the first feature in a greenfield repository. The goal is to establish the minimum durable project facts, boundaries, and mechanical checks so implementation does not invent the foundation ad hoc.

## 1. Project Positioning

Clarify:

- What the project builds and the first coherent delivery slice.
- Whether it is a personal project, team project, or production-critical system.
- Whether there is an old system, old frontend, old API, or old data source of truth.
- Whether compatibility is required for API, data, deployment, clients, or user workflows.
- What is explicitly out of scope for phase one.

Record decisions in `README.md`, `AGENTS.md`, `docs/architecture/`, or a project spec, depending on the repository's documentation shape.

## 2. Stack And Runtime Shape

Decide before scaffolding:

- Language, framework, package manager, database, cache, queue, object storage, and external services.
- Monolith, modular monolith, services, or microservices.
- Local development entry: direct process, Docker Compose, systemd, Nginx, or another runner.
- Early deployment shape.
- Edge entry: Nginx, Go Gateway, API Gateway, Kubernetes Ingress, or no gateway.

For personal or early-stage projects, prefer the simplest credible runtime. Do not introduce Kubernetes, service mesh, complex API gateway, gray release, or BFF aggregation unless the project has a real current need.

## 3. Repository Entry Points

Create or repair:

- `README.md` for project overview, structure, setup, and common commands.
- `AGENTS.md` for project-specific facts, commands, boundaries, docs routing, and verification.
- `CLAUDE.md -> AGENTS.md` when Claude/Codex auto-discovery is part of the workflow.
- `Makefile` or documented scripts for setup, test, and project checks.

`AGENTS.md` must route readers to the owning documents instead of duplicating full documentation taxonomies or global preferences.

## 4. Directory And Ownership Boundaries

Decide:

- Top-level directories and ownership.
- Runtime entrypoints such as `cmd/server`, `api/http`, `internal`, `configs`, `migrations`, or framework equivalents.
- Shared library rules: what can be shared and what must stay local.
- Contract locations.
- Generated file and vendor file rules.
- Prohibited roots such as accidental root modules, global `cmd/`, or cross-service `internal` imports.

Add mechanical checks when structure matters, for example `scripts/check-structure.sh`.

## 5. Documentation Architecture

At minimum, establish:

- `docs/README.md` as navigation index.
- `docs/documentation-rules.md` as documentation ownership and path registration source.
- `docs/architecture/` for current architecture facts.
- `docs/contracts/` for API and cross-boundary contracts.
- `docs/reviews/` for preserved review evidence when needed.
- `docs/todos/debt/` for unresolved technical debt with impact and exit conditions.

Every durable new path should be registered in the docs index, documentation rules, `AGENTS.md` when routing matters, and mechanical checks when stability matters.

## 6. Architecture Boundaries

Before feature work, define:

- Data ownership and authoritative query owner.
- Module or service responsibilities.
- Dependency direction.
- Whether cross-module or cross-service database joins are allowed.
- Where business rules, persistence, adapters, contracts, and runtime wiring live.
- Gateway or facade boundaries.
- Which technical debt is accepted for phase one and where it is tracked.

Prefer architecture-first scaffolding over "write features first and split later".

## 7. External Contracts

For API or integration projects, decide:

- HTTP path, method, headers, query/body/multipart conventions.
- Response envelope.
- Error model: business error code vs HTTP status.
- Time, ID, enum, nullability, number, boolean, and JSON field naming rules.
- Pagination, sorting, filtering, and cursor semantics.
- Event envelope, routing key, versioning, outbox, idempotency, and compatibility rules.
- Provider/consumer ownership of typed clients and events.

Contract decisions belong in `docs/contracts/` or the project-equivalent contract source.

## 8. Runtime Operations

Define before services become runnable:

- Configuration source priority, environment variable naming, defaults, required validation, and secret handling.
- Startup order and failure behavior.
- Health checks such as `live` and `ready`.
- Graceful shutdown for HTTP servers, workers, consumers, queues, and connections.
- Server, client, database, cache, queue, upload, and shutdown timeouts.
- Retry policy, retry eligibility, backoff, jitter, and max attempts.
- Circuit breaker or degradation extension points.
- Idempotency keys, unique constraints, outbox/inbox, and duplicate handling.

Runtime behavior is architecture, not deployment trivia. Record it in architecture or operations docs before first real service implementation.

## 9. Logging, Errors, And Reporting

Decide:

- Error layering across domain, application, adapters, infrastructure, and HTTP/API.
- Which errors are ignored, logged, reported, retried, or compensated.
- Log format and required fields such as service, env, requestId/traceId, operation, errorCode, duration, userId, and resourceId.
- Sensitive data rules: never log tokens, passwords, verification codes, secrets, full sensitive request bodies, or signed URLs.
- What "reporting" means in phase one: structured error logs, metrics, Sentry, OpenTelemetry, Prometheus, Alertmanager, or another system.

Do not let predictable business failures flood `ERROR` logs.

## 10. Authentication And Authorization

For multi-entry or multi-service systems, decide:

- Where authentication happens.
- How identity is passed downstream.
- Which headers are trusted and only from which network path.
- Where resource-level authorization happens.
- Token blacklist, JWT secret, session, role, and permission ownership.
- Public routes and admin routes.

A common pattern is gateway-level authentication plus service-level resource authorization.

## 11. Data And Migration

Define:

- Migration tool and file naming.
- Up/down or forward-only policy.
- Who executes migrations.
- Whether startup auto-migration is forbidden.
- Rollback and backup boundaries.
- Test database setup.
- Ownership of schema, seed data, and local fixtures.

Runtime paths should not become a second schema owner.

## 12. Test Strategy

Set test placement and required commands:

- Unit tests for pure domain or utility behavior.
- Handler/API contract tests.
- Application/use-case tests.
- Repository/integration tests.
- Runtime tests for startup, shutdown, health, and dependency wiring.
- System HTTP or black-box tests.
- Architecture guard tests.
- Shared testkit helpers.

Name the narrowest relevant test command and the full verification command. TDD tests are behavior specifications and regression guards, not disposable scaffolding.

## 13. Mechanical Checks

Do not rely only on prose when structure can be checked.

Common checks:

- Required files and directories.
- Symlink entrypoints.
- Root module or forbidden directory guards.
- Documentation path registration.
- Test command aggregation.
- Contract generation or synchronization.
- Formatting, linting, typechecking, and architecture import checks.

Expose one clear command, such as `make check`.

## 14. Definition Of Done For First Runnable Slice

A first runnable backend/service slice should usually include:

- Project entrypoints and docs routing.
- Runtime configuration and example local config.
- Health checks.
- Timeout and graceful shutdown.
- Logging and request/trace ID propagation.
- Error envelope and error code mapping.
- Authentication/authorization boundary if applicable.
- Migration or explicit no-persistence decision.
- Idempotency strategy for write paths.
- Narrow tests and full project check.
- README updates.

Anything intentionally deferred should be recorded as a "not yet" decision or debt item with impact and exit condition.

## 15. Explicit Non-Goals

Record what the project will not do yet:

- No Kubernetes.
- No gray release.
- No service mesh.
- No complex API Gateway.
- No BFF aggregation.
- No old data compatibility.
- No premature shared library extraction.
- No runtime auto-migration.

Clear non-goals prevent accidental complexity during early implementation.
