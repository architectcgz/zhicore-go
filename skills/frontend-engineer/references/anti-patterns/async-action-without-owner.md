# Anti-Pattern: Async Action Without Owner

Use this negative case when a user-triggered async action is reachable from UI events but has unclear ownership for loading, errors, duplicate execution, cancellation, or cleanup.

## Signals

- A click, submit, keypress, or emit handler calls an async function without local `try/catch` or equivalent error ownership.
- Button disabled state is the only duplicate-action guard.
- Multiple UI event paths can trigger the same request but the handler has no in-flight short-circuit.
- Route param, tab, filter, or search changes can leave stale responses racing with newer state.
- Polling, timers, subscriptions, uploads, or requests continue after unmount or route leave.

## Analysis

1. Identify the UI owner of the workflow and every event path that can trigger it.
2. Check loading, error, empty, success, cancellation, and stale-response behavior.
3. Check repeated click, enter key, form submit, retry, and route-change behavior.
4. Confirm cleanup for timers, listeners, subscriptions, controllers, and third-party instances.

## Recovery Direction

- Put the in-flight guard in the handler or workflow owner, not only in the template.
- Catch normal operational failures at the nearest UI owner and decide toast, inline error, retry, silent abort, or polling stop there.
- Use request tokens, abort controllers, sequence counters, or route guards when stale responses are possible.
- Keep transport-level error normalization in the request layer and workflow-specific error policy in the owning page or composable.
