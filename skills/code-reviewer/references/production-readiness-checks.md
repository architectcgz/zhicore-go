# Production Readiness Checks

Read this file when the change affects initialization, observability, compatibility, configuration, or user-facing error handling.

## Initialization order and implicit dependencies

- Global variables, `init()` functions, and package-level initialization must not assume execution order or environment state
- Backend: check whether `var db = mustConnect(...)` or `init()` logic will fail in test environments, module import reordering, or when environment variables are not yet set
- Backend: verify that `init()` functions do not have hidden dependencies on other packages' `init()` execution order
- Frontend: check whether Pinia store `setup()` calls other stores or relies on app plugins that may not be initialized yet (SSR, test, or early router guard execution)
- Ask whether the initialization can fail silently or produce undefined behavior when preconditions are not met

## Observability gaps

- Critical decision paths must produce logs, metrics, or traces sufficient to diagnose failures in production
- Check whether permission checks, payment callbacks, async task enqueuing, cache fallback, retry decisions, timeout handling, and feature flag branches log enough context (user, resource, decision, reason)
- Check whether errors are logged at appropriate levels: `debug` for verbose detail, `info` for normal operation, `warn` for degraded behavior, `error` for actionable failures
- Flag `err != nil` branches that return early without logging the error (silent failures)
- Frontend: check whether critical user actions (form submit, payment confirm, data export, account deletion) have analytics or tracking events
- Ask what evidence would be available to diagnose a production incident involving this code path

## Backward compatibility and migration

- API contract changes (field renames, type changes, new required fields, new enum values, response structure changes) must not break older clients unless explicitly versioned
- Check whether old clients can still parse responses when new optional fields or enum values are added
- Check whether new required fields have server-side defaults or client-side fallback to avoid breaking old data or old request payloads
- Database schema changes (adding `NOT NULL` columns, changing column types, adding constraints) must be compatible with code currently running in production during gray deployment or rollback
- Check whether adding a `NOT NULL` column has a default value or a backfill migration
- Check whether changing column types preserves data compatibility for in-flight writes from the old version
- Ask whether the change requires ordered rollout (DB first, then code; or code first, then DB) and whether rollback is safe

## Configuration defaults and validation

- Configuration fields must have safe defaults or fail fast with clear validation errors at startup
- Check whether timeout, retry, concurrency, and rate-limit config defaults are safe (not 0/infinite/unbounded)
- Check whether security-sensitive config (CORS origins, allowed IPs, secret keys) defaults to deny-all or rejects placeholder values
- Check whether config values are validated at startup (type, range, format, required fields) rather than silently accepting invalid input and failing at runtime
- Ask what happens if a required config field is missing, empty, or malformed

## Error handling user experience

- User-facing error messages must be actionable and avoid exposing technical implementation details
- Frontend: check whether `catch (err) { message.error(err.message) }` exposes raw HTTP errors, stack traces, or internal error codes to users
- Frontend: check whether form validation errors indicate which fields are invalid and how to fix them
- Frontend: check whether critical workflow failures (submit, export, payment) provide fallback actions, retry guidance, or contact information
- Backend: check whether API error responses distinguish client errors (4xx with actionable fix) from server errors (5xx with "try again later")
- Ask whether a user encountering this error message would know what to do next
