# CTF Review Debt Checks

Use this when reviewing CTF repo changes or updating review documents after fixes.

## Current Fact Sources

- Backend current status: `docs/reviews/backend/README.md`
- Runtime/context review: `docs/reviews/ctf-backend-code-review-runtime-safety-round1-20260422.md`
- Image management review: `docs/reviews/backend/ctf-platform-code-review-image-management-round2-ae03fb5.md`
- Frontend current status: `docs/reviews/frontend/README.md`
- Frontend audit backlog: `docs/reviews/frontend/ctf-frontend-audit-20260422.md`

Old review files are historical snapshots unless a current fact source points to them. Do not turn old `未修复` grep hits into current findings without checking current code.

## Backend Review Checklist

- Public Go service, repository, port, job, and infrastructure methods use `Foo(ctx, ...)`, not `FooWithContext(...)` plus a no-ctx wrapper.
- DB, Redis, Docker, HTTP, and long-running operations receive the caller's `ctx`; GORM queries use `WithContext(ctx)`.
- `context.Background()` and `context.TODO()` appear only at process roots, framework entrypoints, explicit background task roots, or tests.
- Runtime engine absence, Docker failure, and cleanup failure return explicit errors instead of fake success.
- Port selection uses atomic reservation and releases on create failure or cleanup.
- Runtime infrastructure is owned by runtime; composition does not depend on another module's infrastructure for runtime event persistence.
- Production config rejects empty or placeholder secrets and credentialed CORS requires explicit origins.
- Patch DTOs preserve omitted vs empty semantics where that distinction matters.
- Image names, tags, IDs, statuses, pagination, and sort fields are validated before command services.
- Soft-delete filters, indexes, typed page results, and shared error-message constants remain explicit.

## Frontend Review Checklist

- Route param, query, tab, filter, pagination, and polling changes cannot apply stale results.
- User-triggered async actions catch failures at the nearest UI owner.
- Actions reachable through multiple UI paths own an in-flight guard in the handler.
- Timers, debounced functions, intervals, polling, event listeners, observers, and focus traps are cleaned up on unmount.
- Loading, error, empty, success, and no-permission states are intentional where the workflow can reach them.
- Real product styling does not reintroduce raw hex colors, old `rgba(...)`, `text-slate-*`, `bg-white`, or `bg/text/border-[var(--color...)]`.
- Route-view extraction keeps route/query sync, page fetch, cross-section coordination, error policy, and primary business actions in the parent unless a stronger existing pattern says otherwise.
- Dialogs and drawers include accessibility semantics, close behavior, focus handling, max height, and internal scrolling.
- Auth tokens are not reintroduced into `localStorage`.

## Review Document Rules

- Keep historical review text intact; add current status and validation evidence near the top or in the current index.
- When a fix changes current review status, record the validation command and result.
- Keep review-driven commits narrow: one debt class, module, or page slice per commit.
- Mark old findings as "needs current-code revalidation" when the path, module boundary, or product behavior has changed.
