# Technical Risk Checks

Read this file when the review needs hard technical scrutiny.

## Language and runtime idioms

- Check whether the implementation follows the language and framework's idiomatic patterns rather than fighting them
- Look for misuse of concurrency primitives, collection APIs, resource ownership, or error handling conventions
- Ask whether the code will surprise an experienced maintainer in this language or runtime
- Treat non-idiomatic code as a real maintainability risk when it obscures guarantees or encourages misuse

## Correctness

- Does the code actually implement the requirement, not just compile
- Check null, empty, overflow, off-by-one, default branch, and invalid-state handling
- Verify assumptions around ordering, uniqueness, idempotency, and state transitions
- Look for hidden regressions introduced by changing shared utilities, contracts, or data shape
- Check exception handling boundaries, catch-all blocks, and graceful degradation paths
- Ask whether failures become actionable errors, silent corruption, or undefined partial success
- Boundary condition specifics:
  - Array/slice access: check length before indexing (e.g., `arr[0]` requires `len(arr) > 0`, `arr[len-1]` requires `len > 0`)
  - Collection queries: check `.find()`, `.filter()[0]`, map lookups, and optional unwrapping for missing-item cases before accessing properties
  - String operations: verify `.split()` result length matches expected parts before indexing (e.g., `path.split('/')[2]` when path may be shorter)
  - Pagination: verify `offset + limit` does not exceed collection size or produce invalid ranges

## Architecture and design

- Check whether the change respects existing module boundaries
- Look for SOLID violations, especially mixed responsibilities and extension-hostile designs
- Identify over-engineering, premature abstraction, or unnecessary pattern ceremony
- Ask whether the change increases coupling so much that simple future work will require edits across too many files
- Treat growth in already large route views, SFCs, services, repositories, handlers, or long functions as a material design risk when the added logic mixes state ownership, rendering, persistence, authorization, async orchestration, or business rules.
- Check whether decomposition follows ownership boundaries. Splitting only by line count is not enough; the parent should retain orchestration responsibilities while children own coherent local display, validation, or workflow slices.

## Performance

- Look for unnecessary loops, repeated allocations, wasteful serialization, and accidental quadratic work
- Check for N+1 query patterns, missing batching, repeated network calls, or poor cache usage
- Check whether query shape and filtering align with available indexes or likely access paths
- Watch for GC pressure, memory retention, heavy copies, and unbounded collection growth

## Concurrency and state safety

- Check shared mutable state, locking assumptions, transaction boundaries, and ordering guarantees
- Look for race conditions, duplicate execution paths, lost updates, or stale reads
- Verify retry behavior, timeout behavior, and idempotency when operations can run more than once
- In frontend reviews, treat async handlers, route watchers, tab switches, form submits, polling, dialogs, and store writes as state-safety surfaces. Check stale responses, duplicate actions, unhandled rejections, and lifecycle cleanup.
- Concurrency control specifics:
  - Backend: check for optimistic locking (version fields), compare-and-swap, or `WHERE` conditions that prevent lost updates (e.g., inventory decrement should use `WHERE stock >= quantity`, not unconditional `stock = stock - quantity`)
  - Backend: verify that SELECT-then-UPDATE sequences use transactions with appropriate isolation levels or row locks to prevent race conditions
  - Frontend: check for in-flight guards on user-triggered actions reachable through multiple UI paths (double-click, rapid navigation, concurrent tabs)
  - Frontend: verify that concurrent async operations (polling + manual refresh, multiple tabs updating the same store) handle response ordering and do not let stale data overwrite fresh data

## Resource lifecycle and cleanup

- Backend: DB connections, HTTP clients, file handles, goroutines, timers, and streams must be released in defer/finally or explicit cleanup paths
  - Check whether `resp.Body.Close()` is deferred after `http.Get/Post`
  - Check whether goroutines have context cancellation or wait group coordination before process exit
  - Check whether file handles are closed even when read/write returns an error
  - Check whether background timers or tickers are stopped when their owner context is cancelled
- Frontend: event listeners, intervals, observers, and store subscriptions must be cleaned up in `onUnmounted` or equivalent lifecycle hooks
  - Check whether `setInterval`, `setTimeout`, `addEventListener`, `IntersectionObserver`, `ResizeObserver`, `MutationObserver` are cleared or disconnected on component unmount
  - Check whether Pinia store subscriptions, route watchers, and custom event bus listeners are unsubscribed
  - Check whether WebSocket connections, SSE streams, or polling loops are closed when the component or view is destroyed
- Ask whether repeated mount/unmount cycles (e.g., dialogs, tabs, conditional routes) will accumulate leaked resources

## Security

- Check for SQL injection, XSS, CSRF, SSRF, path traversal, and command injection risks where relevant
- Review authorization boundaries, tenant isolation, and privilege escalation risk
- Check whether secrets, tokens, or sensitive identifiers are logged or stored unsafely
- Verify input validation and output encoding at trust boundaries

## Data integrity and compatibility

- Check whether schema, API, or event-contract changes preserve backward compatibility when required
- Look for silent data truncation, lossy transformations, or incompatible default values
- Review migration safety, rollback assumptions, and partial-failure behavior
