# Verification

Read this file before claiming dark-surface alignment is finished.

## Required checks

- Run focused tests for the touched page if they exist.
- Run `npm run build` in `code/frontend`.
- If the task came from visual QA and credentials are available, verify in a real browser.
- When the user points to a specific leaking node, confirm that exact layer changed, not only the parent shell.

## Shared-style regressions

- If you changed a shared selector, add or tighten a regression test that asserts the exact selector boundary, including exclusions such as `:not(.metric-panel-card)`.
- Avoid loose string-presence checks that can falsely pass after a regression.
- If a page locally overrides `--metric-panel-*`, verify the exact overridden variables and compare them with the reference shared surface before closing the task.

## Avoid

- Creating a new unrelated palette
- Using pure white text for all dense content
- Fixing only the route shell when the actual issue is a nested component
- Adding duplicated one-off dark overrides when a shared style file is the correct home
- Treating a cross-page style regression as a single-page issue when the root cause is an over-broad shared selector
