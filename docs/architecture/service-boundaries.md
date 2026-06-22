# Service Boundaries And Data Contracts

This document defines how Go services in `zhicore-go` own data, expose queries, and share cross-service DTOs.

## Core Rule

The service that owns the data also owns the authoritative query.

Other services may call that query or expose a facade route, but they must not own the queried data model, persistence schema, or repository for another service's aggregate.

## Ownership Levels

### Service-Private Data

Service-private data lives under the owning service:

- Domain model: `services/<service>/internal/...`
- Application read/write model: `services/<service>/internal/...`
- Persistence model and repository: `services/<service>/internal/...`
- Schema migrations: `services/<service>/migrations/`
- Service-owned HTTP/API shape: `services/<service>/api/`

No other service may import `services/<service>/internal`.

### Synchronous Cross-Service Contracts

When one service calls another service synchronously, the provider-owned client contract lives under:

```text
libs/contracts/clients/<provider-service>/
```

Examples:

- `libs/contracts/clients/content/`: content service query DTOs and typed client contracts.
- `libs/contracts/clients/user/`: user service profile DTOs and typed client contracts.

The provider service owns the contract because it owns the API behavior and the data lifecycle. Consumers can depend on the contract, but they do not own it.

### Event Contracts

Cross-service event payloads live under:

```text
libs/contracts/events/<domain>/
```

Examples:

- `libs/contracts/events/content/`: post created, post published, post deleted events.
- `libs/contracts/events/user/`: user registered, user profile updated events.

Events should contain stable facts that consumers need, not provider-private persistence details.

## Example: Query A User's Published Posts

Question: for "query all posts published by a user", should the content service call the user service, and where should the data be defined?

Answer:

- `zhicore-content` owns posts, so it owns the authoritative query.
- The query endpoint belongs to content, for example `GET /api/v1/posts/authors/{authorId}`.
- The persistent `Post` model, post repository, pagination rules, post visibility rules, and post DTO mapping belong to `services/zhicore-content/internal` and `services/zhicore-content/api`.
- The cross-service DTO and client contract belong to `libs/contracts/clients/content`.
- `zhicore-user` may expose a facade route such as `GET /api/v1/users/{userId}/posts` only when the product API wants user-centered navigation. That route should call the content contract and return or lightly reshape the content-owned result.

So the direction is:

```text
user facade route -> content client contract -> content service query -> content-owned post store
```

Not:

```text
content service -> user service -> user-owned post query
```

because users do not own posts.

## When A Service May Call Another

A service may synchronously call another service when it needs data that belongs to the other service and the caller cannot maintain a correct local read model.

Acceptable examples:

- Content calls User to validate or snapshot author identity during post creation.
- User calls Content to expose a user-centered facade route for published posts.
- Search calls Content to fetch authoritative post details for indexing repair.

Avoid synchronous calls when the data can be maintained by events with acceptable eventual consistency.

## Facade Rules

A facade route is allowed only when all are true:

- It exists for product/API ergonomics, not data ownership.
- It does not duplicate another service's persistence logic.
- It delegates to the owning service through `libs/contracts/clients/<provider-service>`.
- Any reshaping is shallow and documented at the facade boundary.
- Errors from the owning service are translated consistently, without hiding ownership.

## Promotion Rules

Keep a DTO service-local until at least one external service needs it.

Promote to `libs/contracts/clients/<provider-service>` only when:

- It is part of a synchronous cross-service API.
- The provider service is willing to version and preserve it.
- Multiple consumers or a facade route need the same shape.

Do not promote internal domain models, database entities, or repository filters into `libs/contracts`.
