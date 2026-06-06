# Feature-Sliced Frontend Reference

Use this reference when structuring medium or large frontend applications.

Primary references:

- Feature-Sliced Design: `https://fsd.how/`
- FSD overview: `https://feature-sliced.github.io/documentation/docs/get-started/overview`
- FSD vs Clean Architecture: `https://philrich.dev/fsd-vs-clean-architecture/`

Recommended shape:

```text
src/
  app/
  pages/
  widgets/
  features/
  entities/
  shared/
```

Layer intent:

- `app`: app startup, router, global providers, shell-level bootstrap.
- `pages`: route-level composition. Pages should coordinate blocks, not own large workflows.
- `widgets`: self-contained page sections that combine several entities/features.
- `features`: user actions and business capabilities such as submit flag, start instance, join team, review writeup.
- `entities`: business entities such as user, challenge, contest, team, instance, submission.
- `shared`: UI primitives, API client base, config, design tokens, generic utilities.

Dependency direction:

```text
app -> pages -> widgets -> features -> entities -> shared
```

Lower layers must not import higher layers.

Public API rule:

- A slice should expose imports through `index.ts`.
- Other slices should not import deep internals such as `features/foo/model/state`.
- Deep imports are acceptable only inside the same slice or during short migration windows.

Typical slice shape:

```text
features/start-instance/
  ui/
  model/
  api/
  lib/
  index.ts
```

Use `model` for state machines, async orchestration, query state, form state, permissions, derived data, and workflow transitions.

Use `ui` for Vue/React components that render that feature.

Use `api` for feature-specific request adapters or query functions. Keep raw transport details out of templates.

Use `lib` for feature-local pure helpers.

Avoid:

- Putting all routes, API calls, computed values, and actions into one large page component.
- Making `shared` a dumping ground for business logic.
- Importing a feature from an entity.
- Letting API DTOs leak straight into templates when view models or mappers would reduce coupling.
- Creating a new abstraction for every small component.

Migration rule:

Move one workflow at a time. Extract page logic into `features` or `widgets` only when it reduces page responsibility, test complexity, or repeated behavior.
