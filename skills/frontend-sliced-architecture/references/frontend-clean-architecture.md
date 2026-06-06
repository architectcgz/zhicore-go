# Frontend Clean Architecture Reference

Use this reference when a frontend feature has enough business complexity to justify stricter boundaries.

References:

- React Clean Architecture example: `https://github.com/bespoyasov/frontend-clean-architecture`
- CleanSlice: `https://cleanslice.org/`
- React App Clean Architecture example: `https://github.com/Abouelyatim/React-App-Clean-Architecture`

Core idea:

Frontend Clean Architecture is useful inside complex features, not necessarily as the entire app folder structure.

Strict feature shape:

```text
features/<feature>/
  presentation/
  application/
  domain/
  infrastructure/
  index.ts
```

Use it when the feature has:

- Non-trivial state transitions.
- Offline or cache consistency rules.
- Multiple API calls with rollback, retry, or idempotency behavior.
- Permission-sensitive workflows.
- Complex forms with domain validation.
- Multi-step user flows that need unit tests outside components.

Layer intent:

- `presentation`: components, composables/hooks bound to framework lifecycle, view models.
- `application`: use cases, workflow orchestration, commands, queries, async state ownership.
- `domain`: pure rules, entities, value objects, state machines, validation that does not need Vue/React.
- `infrastructure`: HTTP adapters, storage adapters, query client adapters, browser APIs.

Dependency rule:

```text
presentation -> application -> domain
application -> ports
infrastructure -> ports/domain mapping
```

Practical Vue mapping:

```text
presentation: .vue components, route views, UI-specific composables
application: composables or services that own use cases and async workflows
domain: pure TypeScript functions/types/state machines
infrastructure: API functions, localStorage/sessionStorage adapters, websocket adapters
```

Keep framework types out of domain:

- No Vue refs/reactive/computed in domain.
- No React hooks in domain.
- No router objects in domain.
- No HTTP response objects in domain.

Use mappers:

```text
API DTO -> application result -> view model
API DTO -> domain entity, only when domain rules need it
```

Avoid:

- Four-layer folders for simple display pages.
- Moving every small helper into domain.
- Treating UI state such as modal open/close as domain.
- Returning raw API DTOs directly into large templates when the template needs stable display data.
