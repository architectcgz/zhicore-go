# Route View Migration Boundaries

Use this reference when moving Vue, React, Nuxt, or Next route-level logic into feature slices.

## Target Shape

Route views should mostly compose feature/widget UI and pass events through. Move these out of route files when they are more than trivial display state:

- Route parameter parsing and query-tab synchronization.
- Navigation actions such as `router.push`, `router.replace`, and named-route builders.
- API calls, async workflows, submit/delete/export/download actions, and error policy.
- Business lifecycle effects in `onMounted`, `watch`, `useEffect`, loaders, or equivalent hooks.

Keep local-only presentation state in the route when it is genuinely view-specific and cheap to reason about, such as a local page number for an already-loaded list.

## Migration Order

1. Scan route files for router ownership, API ownership, and lifecycle ownership.
2. Extract one user-facing workflow at a time into `features/<slice>/model/useXxxPage.ts` or `useXxxRoutePage.ts`.
3. Export the new composable through the slice public API.
4. Keep the route component as composition plus event binding.
5. Add source boundary tests so the old dependency does not drift back into the route.
6. Add direct composable tests when the extracted composable owns security, sanitization, route normalization, or non-trivial branching.

## Scan Commands

Vue route ownership:

```bash
rg -n "useRoute|useRouter|router\.push|router\.replace|useRouteQueryTabs" src/views/**/*.vue
```

View API ownership, excluding DTO/type-only contracts:

```bash
rg -nP "from ['\"]@/api/(?!contracts)" src/views --glob '*.vue'
```

Broad API import scan, including type-only contracts:

```bash
rg -n "from ['\"]@/api/" src/views --glob '*.vue'
```

Lifecycle ownership triage:

```bash
rg -n "onMounted\(|watch\(|useEffect\(" src/views --glob '*.{vue,tsx,jsx}'
```

## Boundary Tests

For migrated route views, prefer lightweight source tests:

```ts
import pageSource from '../SomeRouteView.vue?raw'

expect(pageSource).toContain('useSomePage')
expect(pageSource).not.toContain('useRoute')
expect(pageSource).not.toContain('useRouter')
expect(pageSource).not.toContain("from '@/api/")
```

Do not rely only on a route-view test that mocks the new composable. If the composable contains behavior that can break correctness or security, test the composable directly. Examples:

- Redirect sanitization.
- Permission or role branching.
- Query normalization.
- Stale-response or duplicate-submit guards.
- Payload mapping before API calls.

## Public API Rule

After extraction, imports from route views should use the slice public API:

```ts
import { useSomePage } from '@/features/some-slice'
```

Avoid deep imports from route views into `features/some-slice/model/useSomePage`. Keep `index.ts` exports consistent and avoid multiple redundant re-export lines from the same target when one line is clearer.

## Review Checklist

- Route views no longer directly own router objects unless route ownership is explicitly the purpose of that file.
- Route views do not import non-contract API modules.
- Extracted composables with meaningful behavior have direct tests, not only mocked route-view tests.
- Scan commands are robust to both single-quoted and double-quoted imports.
- Project migration documents or TODOs are committed when they are part of the requested deliverable.
