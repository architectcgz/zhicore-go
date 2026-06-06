# Rust Reference: microsoft/cookiecutter-rust-actix-clean-architecture

Use this as the official Rust / Actix reference.

Source:

- `https://github.com/microsoft/cookiecutter-rust-actix-clean-architecture`

Core ideas to preserve:

- Onion / Clean Architecture dependency direction.
- HTTP handlers are adapters, not business owners.
- Application services orchestrate use cases.
- Domain owns business rules and must not depend on Actix, SQLx, Redis, or framework details.
- Infrastructure implements repository / gateway traits.
- Traits define the dependency boundary between inner layers and outer adapters.
- Composition creates concrete implementations at the edge.

When adapting from Rust to another language, copy the dependency rules, not the exact folders.

Rust mapping:

```text
Actix handler        -> transport / api adapter
use case service    -> application layer
entity/value object  -> domain layer
trait repository    -> port
SQLx/Redis adapter   -> infrastructure
container / startup  -> composition root
```

Do not overfit to Rust-specific mechanics:

- Rust traits map to Go consumer-side interfaces.
- Rust modules map to Go packages, but Go package cycles and naming conventions are different.
- Rust error enums may map to Go typed errors, sentinel errors, or package-local error classifiers depending on the project.
- Rust extractors and Actix state should not leak into the application contract.
