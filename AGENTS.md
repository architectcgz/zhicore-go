# ZhiCore Go Agents

## Project Overview

- This repository is the Go migration workspace for the ZhiCore backend.
- The current Java implementation remains in `../zhicore-microservice` and is the contract source until a Go service fully replaces the matching Java service.
- Keep migration incremental. Do not rewrite multiple services in one slice unless the user explicitly asks for a broad migration batch.

## Commands

- `make check`: run scaffold checks and Go tests.
- `make test`: run `go test ./...` inside each Go workspace module.
- `bash scripts/check-structure.sh`: verify required service entrypoints, module directories, documentation entrypoints, and agent entrypoints.

There is no CI or git hook enforcement yet. Run `make check` manually before reporting scaffold or code changes as complete.

## Architecture Boundaries

- `services/<service>` is the deployable, testable, and buildable unit.
- Each service owns its own `go.mod`; do not add a root application module.
- `services/<service>/cmd/server` contains process entrypoints and runtime wiring only.
- `services/<service>/internal` is private to that service. Other services must not import it.
- `libs/kit` is for small, stable cross-service technical primitives only. Do not put service-specific business rules there.
- `libs/contracts/events` owns cross-service event payload contracts.
- `libs/contracts/clients` owns typed client contracts for service-to-service calls.
- Read `docs/architecture/service-boundaries.md` before changing cross-service data ownership, synchronous calls, facade routes, or contract placement.
- Read `docs/contracts/README.md` before changing synchronous client contracts, event payloads, or externally visible API schemas.
- Shared libraries must stay boring and explicit; prefer duplicating unstable service-local code over prematurely promoting it into `libs`.
- Database schema evolution must stay explicit and reviewable. Do not add runtime auto-migration in service startup paths.
- Preserve the existing Java API shape until the matching frontend, gateway, and callers are intentionally changed.

## Service Landing Zones

- `zhicore-gateway` -> `services/zhicore-gateway`
- `zhicore-user` -> `services/zhicore-user`
- `zhicore-content` -> `services/zhicore-content`
- `zhicore-comment` -> `services/zhicore-comment`
- `zhicore-message` -> `services/zhicore-message`
- `zhicore-notification` -> `services/zhicore-notification`
- `zhicore-search` -> `services/zhicore-search`
- `zhicore-ranking` -> `services/zhicore-ranking`
- `zhicore-admin` -> `services/zhicore-admin`
- `zhicore-upload` -> `services/zhicore-upload`
- `zhicore-id-generator` -> `services/zhicore-id-generator`
- `zhicore-ops` -> `services/zhicore-ops`
- Java `zhicore-common`, `zhicore-client`, and `zhicore-integration` map to `libs/kit` and `libs/contracts`; they are not deployable Go services by default.

## Documentation

- Read `docs/documentation-rules.md` before creating, moving, or editing durable docs.
- Use `docs/README.md` as the documentation index.
- Keep migration planning in `docs/migration/`.
- Keep formal review evidence under `docs/reviews/`.
- Keep unresolved technical debt under `docs/todos/debt/`.

## Testing Rules

- Backend behavior changes require TDD: write the boundary test first, confirm it fails for the expected reason, then implement.
- Package-local `*_test.go` files inside a service prove service-local behavior, handlers, services, repositories, workers, and adapters.
- Library tests under `libs/*` prove shared contracts or kit primitives only.
- `tests/architecture` is for source-level boundary checks.
- `tests/system/http` is for black-box HTTP scenarios.
- `tests/runtime` is for tests that need real services, containers, ports, or external dependencies.
- `tests/testkit` is for reusable black-box fixtures and assertions.
- TDD tests are maintained behavior specifications and regression guards. Remove or merge them only when the behavior signal is duplicated, obsolete, implementation-coupled, or intentionally moved to a better owner.

After changing code or tests, run the narrowest relevant `go test` command first, then run `make check` before handoff when the scaffold or shared boundaries changed.
