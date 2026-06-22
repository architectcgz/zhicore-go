# Contract Change Policy

This document defines how to change cross-service contracts in `zhicore-go`.

## Scope

Contracts include:

- Synchronous client contracts under `libs/contracts/clients/<provider-service>/`.
- Event payload contracts under `libs/contracts/events/<domain>/`.
- API schema documents under `services/<service>/api/` when they describe externally visible behavior.

Service-private DTOs, domain models, database entities, repository filters, and internal command/query structs are not contracts.

## Ownership

The provider owns the contract.

Examples:

- Content-owned query DTOs and typed clients live under `libs/contracts/clients/content/`.
- User-owned profile DTOs and typed clients live under `libs/contracts/clients/user/`.
- Content domain events live under `libs/contracts/events/content/`.

Consumers may depend on a contract but must not redefine the provider's data model inside their own service.

## Change Classification

Before editing a contract, classify the change.

### Compatible Changes

Compatible changes may be made in place when they do not break existing providers or consumers.

Allowed examples:

- Add an optional response field.
- Add a nullable field with a safe zero-value interpretation.
- Add a new endpoint or client method without removing the old one.
- Add a new event type.
- Add an optional event field that old consumers can ignore.

### Breaking Changes

Breaking changes require versioning and staged rollout.

Breaking examples:

- Rename, remove, or change the meaning of a field.
- Change a required field into a different type.
- Change pagination, sorting, filtering, visibility, authorization, or idempotency semantics.
- Reuse an event name while changing its meaning.
- Remove an endpoint, client method, or event field that any consumer still uses.

## Required Change Flow

1. Identify the provider and all known consumers.
2. Read `docs/architecture/service-boundaries.md`.
3. Classify the change as compatible or breaking.
4. Update the provider-owned contract in `libs/contracts/...` or `services/<service>/api/`.
5. Add or update contract tests at the smallest owning boundary.
6. Update the provider service implementation.
7. Update consumers only after the provider-compatible path exists.
8. Update documentation when ownership, semantics, or rollout behavior changes.
9. Run the narrowest relevant service tests, then `make check`.

## Breaking Change Flow

Do not break consumers in place.

Use one of these patterns:

- Add `v2` DTOs, client methods, endpoints, or event types.
- Add a new field and keep the old field during the migration window.
- Add a new endpoint while keeping the old endpoint until all consumers move.
- Add a new event type while old consumers continue receiving the old event.

Then migrate in this order:

1. Add the new contract while preserving the old one.
2. Deploy or merge the provider-compatible implementation.
3. Move consumers to the new contract.
4. Prove old contract usage is gone.
5. Remove the old contract in a separate cleanup change.

## Event Contract Rules

- Never change the meaning of an existing event type in place.
- Prefer a new event type or explicit version when semantics change.
- Event payloads should contain stable facts, not provider-private persistence details.
- Consumers must tolerate unknown fields.
- Additive fields must be optional unless every consumer is updated in the same controlled slice.

## Facade Contract Rules

Facade routes may expose a consumer-friendly shape, but they do not own provider data.

For example, `GET /api/v1/users/{userId}/posts` may exist in the user service as a product-facing route. The data and authoritative query still belong to content, and the user route must delegate through the content contract.

If the facade shape differs from the provider shape, document the reshaping at the facade boundary and keep it shallow.

## Do Not

- Do not import another service's `internal` package.
- Do not copy provider-owned DTOs into a consumer service to avoid using `libs/contracts`.
- Do not promote a service-private model into `libs/contracts` before it is a real cross-service contract.
- Do not remove old contracts in the same change that introduces a replacement unless all consumers are proven in the same atomic change.
