# Go Reference: Clean-ish Modular Monolith

Use this as the generic Go reference when starting or reshaping Go backend projects.

If the target repository already has stronger local architecture docs, modular-boundary docs, or a project supplement skill, treat those as the local fact source and use this file only for the stable cross-project shape.

Recommended Go shape:

```text
cmd/api/

internal/app/composition/
  root.go
  modules.go

internal/module/<name>/
  api/http/
  application/commands/
  application/queries/
  application/jobs/
  domain/
  ports/
  contracts/
  infrastructure/postgres/
  infrastructure/cache/
  runtime/
```

Layer responsibilities:

- `api/http`: Gin / net/http handlers, request validation, response mapping. No SQL, Redis, Docker SDK, or business state transitions.
- `application`: use case orchestration, transaction boundary, permission decision, idempotency, event publication.
- `domain`: entities, value objects, invariants, state machines, pure business rules.
- `ports`: consumer-side interfaces for repositories, gateways, clocks, locks, caches, event publishers, and external services.
- `contracts`: stable cross-module APIs and event payloads.
- `infrastructure`: GORM, Redis, Docker, filesystem, HTTP clients, external SDK adapters.
- `runtime`: module-local wiring. This is where concrete infrastructure is connected to application services and handlers.
- `internal/app/composition`: process-level root that connects modules together, not a place for every module's internal object graph.

Go-specific rules:

- Define interfaces at the consumer side.
- Keep ports small and use-case-specific; avoid wide repository interfaces.
- Do not put `*gorm.DB`, `*redis.Client`, Gin context, ORM tags, or HTTP DTOs in ports/domain.
- A single infrastructure repository may implement several small ports.
- Prefer explicit mappers between API DTO, application DTO, domain model, and persistence row.
- Pass `context.Context` through handlers, services, repositories, jobs, and external calls.
- Keep transactions in application or infrastructure through explicit port methods; do not let HTTP handlers own transaction flow.
- Use architecture tests to enforce import boundaries.

Architecture test examples:

```text
ports must not import gin/gorm/redis/internal/dto
domain must not import gin/gorm/redis/internal/model/internal/dto
application must not import infrastructure
api/http must not import gorm/redis
module A must not import module B infrastructure
```

When optimizing an existing Go project:

1. Keep the current deployment shape unless there is evidence that multiple deployable services are needed.
2. Identify business modules and data ownership first.
3. Move complex cross-module reads into readmodel modules.
4. Split wide repository ports before moving files around.
5. Add architecture tests before large refactors.
6. Migrate global DTO/model usage incrementally by use case.
