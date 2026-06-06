---
name: onion-clean-architecture
description: Use when designing, scaffolding, reviewing, or refactoring backend projects toward Onion Architecture, Clean Architecture, modular monolith boundaries, ports/adapters, service ownership, or framework-independent domain/application layers.
---

# Onion Clean Architecture

## Overview

Use this skill to keep backend architecture clear without turning every project into ceremonial DDD. The goal is inward dependency direction, explicit ownership, small ports, and framework-independent business code.

## When To Use

Use for:

- New backend project structure decisions.
- Refactoring handler/service/repository code toward Clean or Onion Architecture.
- Reviewing whether domain/application layers are polluted by HTTP, ORM, cache, SDK, or framework types.
- Designing modular monoliths that may later split into services.
- Choosing where DTOs, domain models, persistence rows, ports, adapters, transactions, events, and read models belong.

Do not use for:

- Simple bug fixes where the owning file and boundary are obvious.
- Pure frontend work.
- Mechanical folder reshuffles without business boundary analysis.

## Core Rule

Dependencies point inward:

```text
api / transport -> application -> domain
                   application -> ports
infrastructure -> ports/domain mapping
composition/runtime -> wires concrete implementations
```

The inner layers must not know web frameworks, ORM clients, Redis clients, Docker SDKs, external SDKs, or deployment details.

## Error Contract Layering

Keep error contracts layered the same way as dependency direction:

- `ports` / `domain` / infrastructure adapters use sentinel or typed errors only for internal business branching, for example not-found, conflict, invalid state, or unavailable dependency semantics that application code must distinguish.
- application services decide whether an internal semantic becomes a normal return, an existing public app error, or a new public app error.
- handlers and API transport should consume only public-facing error contracts such as `errcode.AppError`, not ORM sentinels, cache-library errors, or internal module sentinel errors.

Do not let infrastructure sentinels leak outward just because they are convenient. A repository-level not-found may map to a zero-value read model, a domain-specific public not-found, or a different business result depending on the use case.

## Error Adapter Pattern

When the leak is "application code branches on `gorm.ErrRecordNotFound` / `redis.Nil` / SDK sentinel", prefer a narrow infrastructure adapter before widening or rewriting the raw repository.

Use this pattern:

1. Add one module-local sentinel or typed error in `ports` or `domain` for the semantic the application actually needs, for example `ErrUserScoreNotFound`, `ErrChallengeImageNotFound`, `ErrNotificationNotFound`.
2. Add a narrow infrastructure adapter that wraps the existing concrete dependency and translates the concrete sentinel into the module-local semantic.
3. Keep the application service branching only on the module-local semantic, then map that semantic to:
   - normal return such as `nil, nil` or zero-value fallback
   - an existing public app error
   - a new public app error when the external contract really needs one
4. Wire the adapter in `runtime` or composition, not inside the application package.

Prefer a narrow adapter over changing the raw repository when:

- the raw repository is shared by multiple use cases with different not-found meanings
- only one or two services need the semantic cleanup right now
- changing the raw repository would silently alter other callers' behavior

Prefer changing the main repository contract directly when:

- the repository is already use-case-specific
- all callers should share the same semantic
- keeping both raw and adapted paths would only preserve accidental complexity

Rules:

- Do not export ORM, cache, HTTP, or SDK sentinels from `ports`.
- Do not let handlers branch on module sentinels; handlers should only see public-facing app errors.
- Do not change a shared raw repository's global not-found behavior just to clean one application file.
- Keep adapters boring: translate semantics, pass through everything else.
- Name adapters by the use case or semantic they clean up, not by the concrete library, for example `image_query_repository`, `flag_repository`, `manual_review_repository`.

Test shape:

- application-service tests should assert "module sentinel -> expected business result / errcode"
- adapter tests should assert "concrete sentinel -> module sentinel"
- runtime wiring tests are usually not needed beyond existing compile-time or module tests unless the wiring is easy to regress

## Workflow

1. Identify business modules before choosing folders.
2. Name the owner for each write path, permission decision, state transition, retry policy, and side effect.
3. Define application use cases and their invariants.
4. Put pure rules and state machines in domain.
5. Define small consumer-side ports for persistence, cache, locks, clocks, events, and integrations.
6. Implement ports in infrastructure adapters.
7. Keep composition at the edge; do not let handlers or repositories secretly wire cross-module dependencies.
8. Add architecture tests or import rules before broad refactors.

## Boundary Checks

Ask these questions during design or review:

- Can the domain layer compile without HTTP, SQL, Redis, Docker, queue, or SDK packages?
- Does each mutation path have one owner?
- Are repository ports small enough for one use case, or are they becoming module-wide grab bags?
- Are API DTOs separate from domain models and persistence rows?
- Are cross-module reads handled by owner contracts or read models?
- Is the transaction boundary explicit and close to the use case?
- Can a module be extracted later without rewriting every caller?

## Port Granularity

- Split ports by capability and use-case boundary, not by module name and not by default into one-method interfaces.
- A `ports` interface should usually represent one stable capability such as lookup, write, status transition, list/query, stats, or a transaction-scoped use case.
- Do not keep provider-owned wide ports just because one command service currently needs `find + create + update`. Define those small capabilities in `ports`, then compose the final dependency locally inside the consuming application package.
- Do not mechanically split every read and write into the smallest possible pieces when the methods always travel together for one use case, one state machine, or one transaction boundary. Over-splitting increases ceremony without reducing coupling.
- Transaction-scoped ports may be grouped by transaction use case, for example `InstanceStartTxRepository` or `RoundReconcileTxRepository`, when the grouped methods belong to the same atomic workflow.
- A good rule of thumb:
  - split when different callers consume different subsets, or when one interface mixes unrelated query, write, auth, stats, and lifecycle concerns
  - keep together when the methods form one coherent lifecycle or one transaction-owned workflow

## Go Projects

For Go backends, read `references/go-ref.md` when the task involves project scaffolding, modular monolith layout, Go package boundaries, ports, GORM, Redis, Gin, or architecture tests.

Use `references/go-ref.md` as the generic Go reference. If a repository has stronger local architecture docs or a project-specific supplement skill, read that alongside this generic skill instead of baking repo-specific paths into the global version.

## Rust Projects

For Rust / Actix backends, read `references/rust-ref.md`.

Use Microsoft's `cookiecutter-rust-actix-clean-architecture` as the official Rust reference. Preserve its dependency direction and adapter pattern, not necessarily its exact files.

## Common Mistakes

- Moving files into Clean-looking folders while keeping the same wide service/repository dependencies.
- Defining provider-owned interfaces instead of consumer-side ports.
- Mechanically splitting ports into tiny read/write fragments even when one transaction or one lifecycle owns them as a single workflow.
- Letting `ports` expose ORM tags, cache clients, framework contexts, or transport DTOs.
- Treating every read query as domain logic; complex cross-module reads often belong in read models.
- Introducing global command buses, generic repositories, or excessive abstractions before the codebase needs them.
- Splitting deployable microservices before module ownership, failure boundaries, and data ownership are clear.

## Mapping Helper Convention

- For repeated DTO mapping glue (for example optional string normalization), prefer a shared-kernel helper package (for example `internal/shared/...`) when multiple modules need it.
- Use module-local helper packages only when the helper is truly module-specific and unlikely to be reused elsewhere.
- Prefer explicit helper names such as `NormalizeOptionalString`, `CopyTimeToPtr`, `SingleString`.
- Keep helper scope narrow: only mapping normalization and shape conversion, never business rules or policy decisions.
