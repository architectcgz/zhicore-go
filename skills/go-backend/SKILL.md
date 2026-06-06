---
name: go-backend
description: Use when implementing, refactoring, or reviewing Go backend code, especially context propagation, handlers, services, repositories, jobs, workers, database access, concurrency, idempotency, cache, queues, external integrations, runtime operations, and Go-specific tests.
---

# Go Backend

Use this skill together with `backend-engineer` when the backend code is Go.

## Context Convention

Default every service, repository, handler, job, worker, checker, runner, and other operational boundary to an explicit `ctx context.Context` parameter.

- Prefer `Foo(ctx, ...)` as the canonical method name. Do not keep parallel `FooWithContext(...)` and `Foo(...)` APIs.
- Do not preserve no-context wrappers for compatibility unless the human explicitly asks for that compatibility window.
- Do not synthesize `context.Background()` inside normal application code to satisfy a missing context.
- `context.Background()` is only acceptable at true process roots, framework entrypoints, tests, or explicit lifecycle roots.
- If an internal helper is only called from context-aware code, pass `ctx` directly instead of introducing a second naming style.
- For async workers, cron jobs, report generators, cleanup workers, and background tasks, distinguish request context from application lifecycle context.
- Do not store request contexts in structs; pass `ctx` to each operation that needs cancellation, deadlines, or request-scoped values.
- When deriving a context with timeout, deadline, or cancel, call the returned cancel function, normally with `defer cancel()`.
- For HTTP handlers, propagate `r.Context()` into downstream database, cache, RPC, and worker calls unless a lifecycle context is intentionally required.

## Database and Transaction Rules

- Use context-aware database methods such as `QueryContext`, `ExecContext`, `QueryRowContext`, and `BeginTx` when using `database/sql`.
- Keep all transaction work on the returned `*sql.Tx`; do not mix transaction operations with direct `*sql.DB` calls for the same unit of work.
- Defer `tx.Rollback()` after a successful `BeginTx`; let rollback become a no-op after `Commit`.
- Close `Rows` and check `rows.Err()` after iteration.
- Do not build SQL with `fmt.Sprintf` or string concatenation for values. Pass values as query arguments so the driver can bind them safely.
- Tune connection pools only with evidence from the workload; avoid changing `SetMaxOpenConns`, `SetMaxIdleConns`, or lifetimes as a blind fix.

## Errors and API Contracts

- Always check returned errors unless the ignored error is intentionally harmless and documented at the call site.
- Wrap errors with `%w` only when callers should be able to inspect the underlying condition with `errors.Is` or `errors.As`.
- Do not expose internal sentinel errors, driver errors, SQL text, secrets, or implementation details through public API responses.
- Use package-local typed errors or sentinel errors only when callers have a real branch to take.
- Keep error strings lower-case and without trailing punctuation unless the project already uses another convention.

## Package, Interface, and API Shape

- Keep package names short, lower-case, and domain-specific; avoid `util`, `common`, `helper`, or catch-all packages.
- Put domain context in the import path, not by concatenating long package names. Prefer `internal/teaching/evidence` with `package evidence` over `internal/shared/teachingevidence` with `package teachingevidence`.
- Package names should describe the capability exported by the package, not every caller that currently uses it. If a package only serves a bounded context, keep that context in the directory path.
- Avoid package-name stutter in exported identifiers. Prefer `evidence.NewProxyRequestEvent(...)` over `evidence.NewEvidenceProxyRequestEvent(...)`; prefer `archive.Builder` over `archive.ArchiveBuilder` when the package already supplies the noun.
- Do not create `shared` packages unless the code is genuinely shared across bounded contexts. If reuse is between two submodules of the same domain, create a domain path such as `internal/teaching/evidence` instead of a global shared bucket.
- When extracting common code, name the package after the stable abstraction, not the first implementation detail. For example, event construction belongs in `evidence`; SQL row fetching belongs in a repository or reader, not in the event package.
- Accept interfaces at boundaries only when they reduce coupling for the caller or make tests meaningfully cleaner.
- Return concrete types by default; define interfaces at the consumer side when practical.
- Avoid Java-style service hierarchies, abstract factories, or one-method interfaces unless the existing codebase already owns that pattern.
- Prefer explicit domain structs and small functions over reflection-heavy or generic abstractions for ordinary backend paths.

## Go Implementation Guardrails

- Keep package boundaries boring and explicit; do not introduce generic utility packages to hide domain behavior.
- Preserve repository-local error wrapping, logging, tracing, and transaction conventions.
- Make goroutine ownership, cancellation, timeout, retry, and cleanup behavior visible.
- Keep database transactions scoped and deterministic; do not mix unrelated side effects into a transaction without a clear ordering reason.
- Wide provider-owned repository interfaces must be split into smaller capability repositories. Compose the final dependency in the consuming application package, not in `ports` or `contracts`.
- Use typed structs and existing mappers for API, DTO, persistence, and domain boundaries.
- For mapper migration/refactor, complete the migration to final shape in the same batch; do not leave temporary pass-through wrappers.
- After introducing generated mapper methods, remove redundant wrappers immediately (for example `mapped := ...; return &mapped`) unless they carry real business semantics.
- Generated mappers such as `goverter` are the default choice for pure field-copy `model -> dto` conversions. Keep repetitive structural copying out of hand-written code when the mapping is mostly mechanical.
- Keep business semantics out of generated mappers. Use a thin wrapper only for real domain shaping such as JSON/spec decode, derived counters, preview text, time normalization, owner-injected labels, or cross-record aggregation.
- Prefer field-scoped custom mapping over broad global extensions. Do not register a generic same-type helper such as `string -> string` globally if it would silently affect unrelated fields; map those fields explicitly or patch them in the wrapper.
- When `goverter` hits immutable types with unexported fields, such as `time.Time`, do not fall back to a fully hand-written mapper by default. Prefer a field-scoped `goverter:map Field | ConvertX` or narrow `goverter:extend ConvertX` helper that shallow-copies the value, for example `func ConvertTime(t time.Time) time.Time { return t }`. Avoid `ignoreUnexported` for this case.
- Shared mapper helpers are acceptable only for narrow, non-business shape normalization used by multiple modules, for example `NormalizeOptionalString`. Place them in a shared-kernel package rather than a single module when they are reused across bounded contexts.
- For DTO shape, prefer value fields by default. Use pointer fields only when the API must distinguish nullable, optional, omitted, or zero-versus-unset values.
- Returning `*dto.X` at service or handler boundaries is acceptable when `nil` has meaning, when the existing contract already uses pointers, or when avoiding large struct copies is useful.
- When using generated mappers such as goverter, expose both value and pointer conversion methods when callers need both forms, for example `ToX(source T) dto.X` and `ToXPtr(source *T) *dto.X`.
- If a caller returns a pointer DTO and no extra business semantics are added, call the generated pointer method directly. Do not write `mapped := mapper.ToX(*item); return &mapped`.
- Prefer table-driven tests for business branching and failure cases when the repo already uses them.
- Run the narrowest relevant Go verification before completion, such as `go test ./path/...` or the project-specific test command.
- Do not stop at `go test` on non-trivial changes. Follow the backend-engineer review loop: separate review pass, fix material findings, then re-run impacted verification before claiming completion.
- For code with goroutines, shared state, maps, caches, or background workers, consider `go test -race ./path/...` when the package can run under the race detector.
- For parsers, validators, decoders, protocol handlers, path handling, or security-sensitive input processing, consider a focused fuzz test or at least seed-corpus regression tests.
- Do not rely on `time.Sleep` as synchronization in tests; prefer channels, contexts, fake clocks, hooks, or observable state transitions.

## References

- Read `references/repository-interface-splitting.md` when a repository interface starts mixing unrelated query, command, profile, auth, report, or transaction methods, or when a tx closure currently receives a wide repo.
- In `backend-engineer`, read `references/concurrency-context-and-idempotency.md` for async workers, retries, duplicate execution, cancellation, timeout, and scheduled work.
- In `backend-engineer`, read `references/ctf-go-runtime-context-image-contracts.md` when working in the CTF repo on Go context propagation, runtime provisioning, image management, config safety, or review-driven backend debt.

## Source Basis

These rules are distilled from Go official docs, Go blog posts, package documentation, and Go wiki review guidance. Do not browse these sources during normal task execution. Only refresh from the web when the user explicitly asks for updated official references or when a rule appears stale.
