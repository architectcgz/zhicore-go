# Core Guardrails

Read this file for every `frontend-engineer` task. These are stable implementation rules; load narrower references only when a task touches that area.

## Interaction and state

1. Code non-happy states on purpose: loading, error, empty, success, and no-permission when relevant.
2. Treat every rapid param or filter change as a stale-response and race-condition candidate.
3. Treat every repeated click or submit as a duplicate-action candidate.
4. Treat everything created on mount as something that must be cleaned up on unmount.
5. Keep state ownership narrow and explicit across component, composable, and store boundaries.
6. Keep component contracts understandable: one source of truth for props, emits, local draft state, and remote mutations.
7. Keep mock data, page logic, and UI separated. Mock data belongs in fixtures, mock services, stories, or demo-data adapters; page logic belongs in the route owner, composable, store, or workflow helper; UI components render props and emit intent. This separation still applies when no mock data exists.
8. Any user-triggered async action that can fail during normal operation must be caught at the nearest UI owner. Do not let submit/save/delete/export/download handlers or poll callbacks bubble API failures into Vue global error handling.
9. The request layer should stay focused on transport concerns such as error normalization, auth/session handling, redirects, and cancellation. The page or composable that owns the workflow decides whether a failure becomes toast, inline feedback, polling stop, or a silent abort.
10. Any async action reachable from more than one UI event path such as `@submit.prevent`, `@click`, or `@keyup.enter` must own an explicit in-flight guard in the handler. Do not rely on button disabled state alone to prevent duplicate requests.
11. Do not bind a raw async method directly to a user action unless that method already owns its failure handling and cleanup path.

## Component and route ownership

1. For route views and page-level workspaces, define ownership boundaries before extracting code. Parent pages should keep route/query synchronization, page-level data loading, cross-section coordination, error policy, and primary business actions. Do not extract code just to reduce line count if that makes state ownership harder to understand.
2. Do not let a route-level `.vue` file keep accumulating route state, API calls, derived data, interaction workflows, and large template branches in one place.
3. When a page owns more than two independent responsibilities, is already around 500 lines, or has a `<script setup>` that reads like a page controller, evaluate extraction before adding another feature flow.
4. Split composables by one page capability domain, such as `useXxxTabs`, `useXxxDetail`, `useXxxActions`, `useXxxMetrics`, or `useXxxPreview`; do not create a new catch-all utility.
5. Follow Vue 3 Composition API conventions and prefer `<script setup>` when the codebase already uses it.
6. Extract reusable logic into composables (`use*.ts`) when doing so clearly reduces component complexity.
7. Always provide stable `:key` values for list rendering.
8. Do not mix the Options API and the Composition API unless required by legacy code.
9. When a component, composable, store, or route workflow keeps branching on the same discriminator, read `references/design-pattern-selection.md` before extending the branch.

## Styling, tokens, and accessibility

1. Sanitize any HTML rendered from user input with DOMPurify.
2. Keep clickable elements visibly interactive and keep keyboard focus states visible.
3. Never hardcode font sizes in `px`; use design-system variables such as `var(--font-size-14)`.
4. Never hardcode spacing in `px` for margin, padding, or gap; use standard spacing variables such as `var(--space-4)`.
5. Prefer `color-mix` for transparency and subtle color adjustment so styling remains consistent across themes.
6. For form controls such as inputs, selects, textareas, search fields, and filters, drive background, border, placeholder, caret, focus ring, and inner highlight through theme tokens or semantic CSS variables, then check both light and dark themes.
7. Avoid broad generic local class names that are likely to collide with global styles. Prefer component- or page-scoped naming unless the project already provides a shared class.
8. Drive modal and drawer alignment, blur, spacing, and layout behavior through CSS variables instead of brute-force global overrides.
9. Do not use `!important`; solve style conflicts through specificity, ownership, or variable inheritance.
10. Do not use Vue deep selectors as the default way to style child components. Use `:deep()` only for a narrow third-party or legacy component override after checking props, slots, wrapper classes, tokens, and CSS variables.
11. Do not hardcode color values, font sizes, or spacing inline when the project already provides tokens or shared classes.
12. Use lazy loading for images and always provide `alt` text.

## Data, copy, and implementation hygiene

1. Do not use `any` for DTOs, form payloads, or critical business logic; define explicit interfaces.
2. Render usernames, slugs, and IDs in their raw business form without decorative prefixes such as `@`.
3. Prefer simpler interaction models over secondary highlight or focus flows; row-level state and side drawers are usually easier to reason about.
4. Every modal and drawer must define `max-height` and support internal scrolling.
5. For table/list row actions, do not hide one or two available actions behind a `More`/`更多` menu. Show them directly in the row action area; use overflow menus only when there are more than two actions or when secondary actions would otherwise crowd the row.
6. Reusable shell components such as modal, drawer, popover, panel, empty state, card, table, and form wrappers must not ship with visible scaffold or demo prose as runtime defaults. Keep examples in tests, docs, stories, or comments.
7. When a frontend handler, watcher, computed branch, or async flow enforces a non-obvious business rule or exception path, keep the comment adjacent to that code and describe the user/business trigger plus resulting behavior, not the syntax itself.
8. When a frontend project exposes alias paths such as `@/`, prefer alias imports for cross-directory shared modules, styles, and assets instead of deep relative paths. Keep relative paths for same-directory or clearly local companion files.
9. Do not use `replace_all` for short identifiers across the codebase.
10. Do not skip tests with `--no-verify`.

## TDD boundaries

- TDD is required for frontend state, validation, derived data, permissions, async flows, stores, composables, reducers, route workflow helpers, and user interaction rules.
- Pure visual styling, copy-only edits, static layout, and token-only polish do not require TDD unless behavior changes are mixed in.
- Prefer testing the smallest owner of the behavior. A composable/store/helper test is usually better than a route-view test when the route view only wires the behavior.
- Component tests should exercise real interaction and emitted contracts. Do not use snapshot-only tests as the failing red step for behavior changes.
- During the refactor step, keep frontend tests from becoming one oversized route or component spec. Split by behavior, use local builders for repeated setup, and remove duplicate tests that provide the same failure signal.

## Decomposition routing

Use `frontend-engineer` for local owner-safe extraction during implementation or refactor. Keep the split small when:

- a component has a stable visual region with clear props and emits
- a composable can own one async workflow, form workflow, or local state machine
- extraction reduces ownership ambiguity instead of only reducing line count
- the route view remains the owner of route/query synchronization, page-level loading, retry policy, cross-section coordination, and primary business actions

Use `frontend-sliced-architecture` when decomposition changes frontend architecture, such as route views owning multiple API calls, route/query synchronization, lifecycle workflows, cross-section state, feature/entity/widget/page/shared boundaries, Feature-Sliced Design, public API rules, source-boundary tests, or migration planning.

Use `extract` when the task is reusable UI pattern extraction, component library consolidation, or design token extraction rather than workflow or state ownership.

## Inspection hooks

For broad frontend boundary inspection from a target repository root:

```bash
node /home/azhi/.codex/skills/frontend-engineer/scripts/inspect-frontend-boundaries.mjs
```

For generic deep-relative import checks in projects that expose alias paths such as `@/`:

```bash
node /home/azhi/.codex/skills/frontend-engineer/scripts/check-alias-paths.mjs --cwd /path/to/repo
```

Add `--alias @ --root src` when the project alias cannot be detected from `tsconfig` / `jsconfig`. Use `--allow <pattern>` only for intentional local exceptions.
