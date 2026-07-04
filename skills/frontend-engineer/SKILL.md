---
name: frontend-engineer
description: Use when implementing or refactoring frontend code, especially Vue components, composables, route views, forms, tables, dialogs, async flows, or interaction behavior where correctness, state ownership, lifecycle cleanup, and maintainability matter more than visual polish alone.
---

# Frontend Engineer

Build frontend code that survives real interaction, not just the happy path.

## Scope

Act as a frontend engineering agent for frontend implementation, refactor, interaction behavior, state ownership, async workflow safety, component contracts, lifecycle cleanup, and maintainability analysis. Keep visual art direction and broad design strategy with dedicated design skills unless behavior or maintainability is affected.

- Implementing or refactoring Vue components, composables, or route-level behavior
- Fixing UI bugs involving stale data, duplicate actions, async races, or cleanup gaps
- Touching form state, props and emits contracts, `v-model`, or API-to-UI data shaping
- Editing interaction behavior, loading and error states, or lifecycle-driven effects

Do not use for pure visual art direction or page styling with no behavior or state changes. For audit-only work with no implementation or frontend design decision, prefer `audit` or `code-reviewer`.

## Always Read

- `rules/core-guardrails.md` for stable frontend engineering rules, TDD boundaries, decomposition routing, and inspection hooks.
- `workflows/implementation.md` when implementing or refactoring code.
- `workflows/completion-review.md` before finalizing implementation, refactor, interaction behavior, or maintainability work.

## Reference Map

| Task surface | Read |
|---|---|
| Vue SFC structure, names, comments, file responsibility | `references/code-organization.md` |
| Fetching, saving, retries, uploads, polling, tab switches, cancellation, stale responses, duplicate actions | `references/async-execution.md` |
| UI event/workflow lacks loading, error, duplicate-action, stale-response, cancellation, or cleanup owner | `references/anti-patterns/async-action-without-owner.md` |
| Props, emits, `v-model`, forms, user input, mock boundaries, API-to-UI data normalization | `references/component-contracts-and-inputs.md` |
| Branching by mode, status, provider, action type, tab, permission shape, lifecycle step, or workflow state | `references/design-pattern-selection.md` |
| Component/composable/route view contract drift for props, emits, draft state, remote mutations, API-to-UI shaping | `references/anti-patterns/component-contract-drift.md` |
| State ownership, Vue reactivity, component boundaries | `references/state-boundaries.md` |
| Mount effects, cleanup, large lists, third-party instances, styling boundaries | `references/lifecycle-rendering-and-styling.md` |
| Token drift, hardcoded local styling, global overrides, internal implementation copy | `references/anti-patterns/token-and-styling-drift.md` |
| Vue route-view template under `RouterView`, `Transition`, or parent-applied layout classes | `references/route-view-transition-root.md` |
| Visible UI copy, helper prose, dashboard descriptions, empty states, feature/navigation explanations | `references/ui-copy-boundaries.md` |
| CTF repo Vue async races, duplicate actions, theme-token drift, route extraction, dialogs, review-driven frontend debt | `references/ctf-vue-async-theme-route-ownership.md` |
| Route views migrating into feature slices, router/API/lifecycle scans, source boundary tests, composable tests | `../frontend-sliced-architecture/references/route-view-migration-boundaries.md` |
| Claiming frontend implementation is complete | `references/verification-checklist.md` |
| Architecture-level decomposition, Feature-Sliced Design, public API boundaries | `../frontend-sliced-architecture/SKILL.md` |

## Known Gotchas

- Mock data, page workflow, and UI rendering must stay separated; details in `references/component-contracts-and-inputs.md`.
- Vue deep selectors are last-resort scoped-style escapes, not normal styling; details in `references/lifecycle-rendering-and-styling.md`.
- Route views should not keep accumulating route sync, data loading, derived state, actions, and large templates; see `rules/core-guardrails.md` and `../frontend-sliced-architecture/SKILL.md`.
- Async entry points are rejection boundaries; direct submit/click/poll handlers must own failure, in-flight, stale-response, and cleanup behavior.

## Check

- Did you read `rules/core-guardrails.md` and the workflow file for this task?
- Did you load only the relevant references instead of the whole rulebook?
- Did you state verification actually run and any unverified risk?
