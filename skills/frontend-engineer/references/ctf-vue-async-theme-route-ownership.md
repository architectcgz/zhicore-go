# CTF Review Debt Patterns

Use this when frontend work touches the CTF repo, especially Vue route views, composables, async flows, platform/teacher workspaces, theme-token work, or review-driven fixes.

## Stale Async Results

Bad:

```ts
const load = async () => {
  loading.value = true
  items.value = await fetchItems(filters.value)
  loading.value = false
}
```

Good:

```ts
let requestId = 0

const load = async () => {
  const current = ++requestId
  loading.value = true
  try {
    const result = await fetchItems(filters.value)
    if (current !== requestId) return
    items.value = result
  } catch (error) {
    if (current === requestId) errorState.value = normalizeError(error)
  } finally {
    if (current === requestId) loading.value = false
  }
}
```

Rules:

- Treat route params, query, tabs, filters, pagination, and polling as stale-response candidates.
- Use request tokens, abort signals, or one explicit "latest result" owner.
- Locally catch API failures at the page or composable owner; do not rely on Vue global error handling.
- Keep `request.ts` focused on transport concerns: error normalization, auth/session handling, redirects, and cancellation.
- Do not encode toast ownership or UI presentation policy in `api/*.ts`.
- Let the nearest page or composable owner decide whether a failure becomes toast, inline `loadError`, empty-state copy, or intentional silence.
- Do not show both inline failure state and toast for the same load path unless the UX explicitly needs two different messages.

## Duplicate Actions And Cleanup

Bad:

```vue
<form @submit.prevent="save">
  <button :disabled="saving" @click="save">Save</button>
</form>
```

Good:

```ts
const save = async () => {
  if (saving.value) return
  saving.value = true
  try {
    await submit()
  } finally {
    saving.value = false
  }
}
```

Rules:

- Every action reachable through multiple UI paths must guard in the handler.
- Disabled buttons are not enough when `@submit`, `@click`, and keyboard handlers can converge.
- Clean up timers, debounced functions, intervals, polling, event listeners, observers, and focus traps on unmount.

## Theme Token Drift

Bad:

```vue
<span class="text-slate-300 bg-white border-[var(--color-border)]">Ready</span>
```

Good:

```vue
<span class="instance-status instance-status--ready">Ready</span>
```

```css
.instance-status--ready {
  color: var(--color-success);
  background: color-mix(in srgb, var(--color-success) 12%, transparent);
}
```

Rules:

- Do not add raw hex colors, `rgba(...)` legacy shadows, `text-slate-*`, `bg-white`, or `bg/text/border-[var(--color...)]` in real product paths.
- Use semantic classes, CSS variables, `color-mix`, and existing workspace/surface primitives.
- Run `npm run check:theme-tail` when touching real product styling that could affect theme-token debt.

## Route View Ownership

Bad:

```ts
// Extracted child owns route query, page fetch, cross-panel state, and save action
```

Good:

```ts
// Parent route owns route/query sync, page fetch, error policy, and primary actions.
// Child owns a clear display block or local form.
```

Rules:

- Route views own route/query sync, page-level data loading, cross-section coordination, error policy, and primary business actions.
- Child components should own clear display blocks or local forms only.
- Do not extract code just to reduce line count if ownership becomes harder to reason about.
- For oversized CTF components, split one owner-safe slice at a time and add behavior or source-boundary tests.

## Dialogs And Drawers

Rules:

- Include ARIA semantics, title/description linkage, ESC/close behavior when appropriate, and focus return.
- Define max height and internal scrolling.
- Render loading, error, empty, success, and no-permission states intentionally when workflows can reach them.
- Do not store auth tokens in `localStorage`; keep auth aligned with server session and `HttpOnly` cookie behavior.
