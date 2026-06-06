# Async Execution

Read this file when the task touches data fetching, submissions, filter changes, uploads, retries, or staged async UI.

## Non-happy states

- Model `loading`, `error`, `empty`, and `success` explicitly.
- Include no-permission or forbidden state when the surface is role-sensitive.
- Do not let a blank area stand in for either loading failure or empty data.

## Re-entry and duplicate actions

- Disable submit or destructive actions while the same request is in flight.
- Treat repeated clicks as a duplicate-submission risk by default.
- Multi-step flows must not roll backward because an earlier request finishes late.

## Race conditions

- For fast tab, route, or filter switching, cancel stale requests with `AbortController` when possible.
- If cancellation is not available, use request ids or a latest-request guard before assigning state.
- For search, filter, and typeahead inputs, prefer debounce when users can fire rapid updates.

```ts
const requestId = ++latestRequestId
const controller = new AbortController()

const result = await fetchList(params, { signal: controller.signal })
if (requestId !== latestRequestId) return

rows.value = result
```

## Error handling

- Critical async operations should use `try/catch/finally`.
- Never swallow errors.
- A `catch` block must either surface feedback, report the error, or explicitly document why ignoring it is safe.
- Use `finally` to clear loading state so failures do not leave the UI stuck.

## Optimistic and staged updates

- If the UI updates optimistically, define rollback or reconciliation behavior before shipping it.
- Do not let a failed mutation leave the screen in a fake-success state.
- When a flow has staged local edits plus remote persistence, make the authoritative source explicit after save.
