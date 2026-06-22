# Libraries

Shared Go modules used by services.

- `contracts`: cross-service DTO, event, and client contracts.
- `kit`: small shared technical primitives such as HTTP API envelopes, auth, config, observability, and infrastructure clients.

Keep business rules inside `services/<service>/internal`.
