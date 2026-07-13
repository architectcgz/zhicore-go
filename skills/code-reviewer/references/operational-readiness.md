# Operational Readiness

Read this file when a change can affect deployment safety, runtime configuration, or rollback behavior.

## Config and feature flags

- Check whether new configuration has safe defaults and explicit validation
- Review feature flags for default state, kill-switch behavior, and cleanup plan
- Watch for environment-specific assumptions hidden in code paths that look generic

## Timeouts, retries, and degradation

- Verify timeout and retry behavior at integration boundaries
- Check whether retries can duplicate side effects or amplify load
- Ask how the system behaves when dependencies are slow, unavailable, or partially failing
- Prefer explicit degradation paths over hanging forever or failing silently

## Migration and rollout risk

- Check whether schema, event, or protocol changes require ordered rollout
- Review backward and forward compatibility during mixed-version deployment windows
- Ask whether partial rollout, rollback, or replay can break invariants
- Watch for data migrations that are technically correct but operationally unsafe at production scale

## Observability for release safety

- New risky paths should usually have enough logs, metrics, or tracing to detect failure quickly
- If a review finding depends on runtime behavior, note what signal would confirm or falsify it in production
