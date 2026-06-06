---
name: frontend-engineer
description: Use when implementing or refactoring frontend code, especially Vue components, composables, route views, forms, tables, dialogs, async flows, or interaction behavior where correctness, state ownership, lifecycle cleanup, and maintainability matter more than visual polish alone.
---

# Frontend Engineer

Build frontend code that survives real interaction, not just the happy path.

## Role

Act as a frontend engineering agent for frontend implementation, refactor, interaction behavior, state ownership, async workflow safety, component contracts, lifecycle cleanup, and maintainability analysis. Keep visual art direction and broad design strategy with dedicated design skills unless behavior or maintainability is affected.

## Use When

- Implementing or refactoring Vue components, composables, or route-level behavior
- Fixing UI bugs involving stale data, duplicate actions, async races, or cleanup gaps
- Touching form state, props and emits contracts, `v-model`, or API-to-UI data shaping
- Editing interaction behavior, loading and error states, or lifecycle-driven effects

## Do Not Use

- Pure visual art direction or page styling with no behavior or state changes
- Audit-only work where the task is to evaluate rather than implement; prefer `audit` or `code-reviewer` when no implementation or frontend design decision is needed

## Core Guardrails

1. Code non-happy states on purpose: loading, error, empty, success, and no-permission when relevant.
2. Treat every rapid param or filter change as a stale-response and race-condition candidate.
3. Treat every repeated click or submit as a duplicate-action candidate.
4. Treat everything created on mount as something that must be cleaned up on unmount.
5. Keep state ownership narrow and explicit across component, composable, and store boundaries.
6. Keep component contracts understandable: one source of truth for props, emits, local draft state, and remote mutations.
7. Sanitize any HTML rendered from user input with DOMPurify.
8. Keep clickable elements visibly interactive and keep keyboard focus states visible.
9. Never hardcode font sizes in `px`; use design-system variables such as `var(--font-size-14)`.
10. Never hardcode spacing in `px` for margin, padding, or gap; use standard spacing variables such as `var(--space-4)`.
11. Prefer `color-mix` for transparency and subtle color adjustment so styling remains consistent across themes.
12. Do not use `any` for DTOs, form payloads, or critical business logic; define explicit interfaces.
13. Render usernames, slugs, and IDs in their raw business form without decorative prefixes such as `@`.
14. Prefer simpler interaction models over secondary highlight or focus flows; row-level state and side drawers are usually easier to reason about.
15. Every modal and drawer must define `max-height` and support internal scrolling.
16. Any user-triggered async action that can fail during normal operation must be caught at the nearest UI owner. Do not let submit/save/delete/export/download handlers or poll callbacks bubble API failures into Vue global error handling.
17. The request layer should stay focused on transport concerns such as error normalization, auth/session handling, redirects, and cancellation. The page or composable that owns the workflow decides whether a failure becomes toast, inline feedback, polling stop, or a silent abort.
18. Any async action reachable from more than one UI event path such as `@submit.prevent`, `@click`, or `@keyup.enter` must own an explicit in-flight guard in the handler. Do not rely on button disabled state alone to prevent duplicate requests.
19. For route views and page-level workspaces, define ownership boundaries before extracting code. Parent pages should keep route/query synchronization, page-level data loading, cross-section coordination, error policy, and primary business actions. Do not extract code just to reduce line count if that makes state ownership harder to understand.
20. For table/list row actions, do not hide one or two available actions behind a `More`/`更多` menu. Show them directly in the row action area; use overflow menus only when there are more than two actions or when secondary actions would otherwise crowd the row.
21. Do not treat typecheck or a few happy-path tests as sufficient closure when the leader or pipeline has classified the work as non-trivial; completion requires a distinct review pass for interaction regressions, state-ownership drift, oversized component debt, contract mismatches, and test gaps.
22. Report frontend risk signals instead of redefining trivial/non-trivial policy locally: async flow, form, route sync, store, modal or drawer state, cross-component contract, user-visible workflow, extraction, or oversized component/service growth.
23. Do not let a route-level `.vue` file keep accumulating route state, API calls, derived data, interaction workflows, and large template branches in one place.
24. When a page owns more than two independent responsibilities, is already around 500 lines, or has a `<script setup>` that reads like a page controller, evaluate extraction before adding another feature flow.
25. Split composables by one page capability domain, such as `useXxxTabs`, `useXxxDetail`, `useXxxActions`, `useXxxMetrics`, or `useXxxPreview`; do not create a new catch-all utility.
26. For form controls such as inputs, selects, textareas, search fields, and filters, drive background, border, placeholder, caret, focus ring, and inner highlight through theme tokens or semantic CSS variables, then check both light and dark themes.
27. Avoid broad generic local class names that are likely to collide with global styles. Prefer component- or page-scoped naming unless the project already provides a shared class.
28. Reusable shell components such as modal, drawer, popover, panel, empty state, card, table, and form wrappers must not ship with visible scaffold or demo prose as runtime defaults. Keep examples in tests, docs, stories, or comments.

## Workflow

1. Read the route view, component, or composable that actually owns the behavior before editing.
2. Identify the dominant risk first: async execution, state ownership, component contract drift, lifecycle cleanup, or rendering pressure.
3. If the task touches styling, inspect the existing shared tokens, CSS variables, and component shells before introducing new local rules.
4. Load only the relevant reference files from `references/` instead of treating every task as the whole frontend rulebook.
5. Keep the implementation boundary small and explicit. One owner for one workflow is the default.
6. Before shrinking a large route view, write down what must remain page-owned versus what is safe to move into a child component or composable. Reduce ownership ambiguity first; line-count reduction is only a side effect.
7. For route-view template/root edits under `RouterView`, `Transition`, or parent-applied layout classes, read `references/route-view-transition-root.md`.
8. For visible UI copy changes, headings, helper text, empty states, or dashboard/workspace prose, read `references/ui-copy-boundaries.md`.
9. Validate loading, error, empty, and repeated-action behavior before closing the task.
10. Audit direct event-bound async entry points before closing the task: form submit handlers, click handlers, emit handlers, composable methods passed to components, and polling callbacks are all rejection boundaries.
11. Run the narrowest relevant tests available. If tests cannot be run, state that clearly and call out the highest-risk unverified paths.
12. After implementation and initial verification, perform a separate review pass. For leader/pipeline-classified non-trivial frontend work, use `requesting-code-review` or `code-reviewer`. For smaller changes, explicitly switch into review mode yourself instead of stopping at "typecheck passed".
13. Fix review findings that materially affect interaction correctness, state ownership, component boundaries, regressions, or test coverage, then re-run the impacted verification.
14. When a component mixes keyboard submit and pointer submit paths, inspect the template and handler together: check whether `@keyup.enter`, form submit, and action buttons can converge on the same async function, then verify the handler short-circuits while a request is already in flight.
15. If a reusable frontend rule gap or repeated miss is found, record it with `improvement-tracker` instead of only mentioning it in the response.

## Decomposition Routing

Use `frontend-engineer` for local owner-safe extraction during implementation or refactor. Keep the split small when:

- a component has a stable visual region with clear props and emits
- a composable can own one async workflow, form workflow, or local state machine
- extraction reduces ownership ambiguity instead of only reducing line count
- the route view remains the owner of route/query synchronization, page-level loading, retry policy, cross-section coordination, and primary business actions

Use `frontend-sliced-architecture` when decomposition changes frontend architecture, such as:

- route views owning multiple API calls, route/query synchronization, lifecycle workflows, or cross-section state
- feature, entity, widget, page, or shared boundaries are unclear
- the task needs Feature-Sliced Design, public API rules, source-boundary tests, or migration planning
- extraction changes project structure rather than one component or composable boundary

Use `extract` when the task is reusable UI pattern extraction, component library consolidation, or design token extraction rather than workflow or state ownership.

## Inspection Hook

For broad frontend boundary inspection from a target repository root, run:

```bash
node /home/azhi/.codex/skills/frontend-engineer/scripts/inspect-frontend-boundaries.mjs
```

Use the output only as candidate evidence. Confirm by reading the owning component, composable, route view, caller, styles, and tests before recommending changes.

For generic deep-relative import checks in projects that expose alias paths such as `@/`, run:

```bash
node /home/azhi/.codex/skills/frontend-engineer/scripts/check-alias-paths.mjs --cwd /path/to/repo
```

Add `--alias @ --root src` when the project alias cannot be detected from `tsconfig` / `jsconfig`. Use `--allow <pattern>` only for intentional local exceptions.

## Implementation Rules

- Follow Vue 3 Composition API conventions and prefer `<script setup>` when the codebase already uses it.
- Extract reusable logic into composables (`use*.ts`) when doing so clearly reduces component complexity.
- When extracting route-view code, keep the parent page as the owner of routing state, page-level fetch/retry policy, cross-panel coordination, and top-level business actions unless there is a stronger existing pattern in the codebase.
- Always provide stable `:key` values for list rendering.
- Keep form validation feedback close to the relevant field.
- Use lazy loading for images and always provide `alt` text.
- Drive modal and drawer alignment, blur, spacing, and layout behavior through CSS variables instead of brute-force global overrides.
- Do not use `!important`; solve style conflicts through specificity, ownership, or variable inheritance.
- Do not mix the Options API and the Composition API unless required by legacy code.
- Do not hardcode color values, font sizes, or spacing inline when the project already provides tokens or shared classes.
- When a frontend project exposes alias paths such as `@/`, prefer alias imports for cross-directory shared modules, styles, and assets instead of deep relative paths. Keep relative paths for same-directory or clearly local companion files.
- Do not use `replace_all` for short identifiers across the codebase.
- Do not skip tests with `--no-verify`.
- Do not bind a raw async method directly to a user action unless that method already owns its failure handling and cleanup path.

## Output Protocol

Use this structure for frontend implementation, refactor, interaction behavior, or maintainability work unless the user explicitly asks for another format.

Always include:

- Result
- Change Surface
- User-Facing Behavior
- State / Async Ownership
- Component Contract
- Verification
- Review / Completion Gate
- Risks / Unverified Points
- Improvement Records

Include only when relevant:

- Accessibility / Keyboard Behavior
- Responsive / Overflow Behavior
- Styling / Token Impact
- Lifecycle / Cleanup
- API / DTO Mapping
- Performance / Rendering Pressure
- Copy / User-Visible Content

Rules:

1. Start with the result.
2. Separate verified behavior from inference.
3. Name the owner of each async workflow, state source of truth, validation path, and remote mutation.
4. Keep route views, components, composables, and stores use-case-oriented; do not extract code only to reduce line count.
5. Do not include irrelevant conditional sections.
6. If a reusable frontend rule gap is discovered, use `improvement-tracker` and list the created file under `Improvement Records`.

Field guidance:

- `Change Surface`: list affected route views, components, composables, stores, API clients, styles, tests, and docs when applicable.
- `User-Facing Behavior`: state what users can now do, what changed, and what remains unchanged.
- `State / Async Ownership`: state where loading, error, empty, success, cancellation, stale-response, and duplicate-action handling live.
- `Component Contract`: state relevant props, emits, `v-model`, local draft state, and API-to-UI mapping boundaries.
- `Verification`: list only commands or checks actually run. If not run, state why and name the highest-risk unverified paths.
- `Review / Completion Gate`: state frontend risk signals, self-review result, independent review status when required, and whether leader or pipeline gating remains.
- `Improvement Records`: list created `docs/improvements/...` files, or state `None` if no reusable agent or policy gap was found.

## Improvement Recording

If a reusable frontend rule gap, repeated miss, missing checklist item, or frontend engineering optimization should be preserved for future runs, use `improvement-tracker` to record it under `docs/improvements/`.

Do not only mention durable frontend improvement gaps in the final response.

## Reference Map

- `../frontend-sliced-architecture/SKILL.md`
  Use `frontend-sliced-architecture` when frontend work becomes architecture-level decomposition, feature-slice design, route-view migration, layer boundary planning, or public API boundary work.
- `references/code-organization.md`
  Read when structuring Vue SFCs, naming things, placing comments, or deciding whether a file is carrying too many responsibilities.
- `references/async-execution.md`
  Read when the task involves fetching, saving, retries, uploads, polling, tab switches, or any workflow with loading, cancellation, stale responses, or duplicate actions.
- `references/anti-patterns/async-action-without-owner.md`
  Read when a UI event or workflow shows the negative pattern of unclear loading, error, duplicate-action, stale-response, cancellation, or cleanup ownership.
- `references/component-contracts-and-inputs.md`
  Read when editing props, emits, `v-model`, form state, validation flows, or API-to-UI data normalization.
- `references/anti-patterns/component-contract-drift.md`
  Read when a component, composable, or route view shows the negative pattern of unclear ownership for props, emits, draft state, remote mutations, or API-to-UI shaping.
- `references/state-boundaries.md`
  Read when deciding whether state belongs in a component, composable, or store, or when handling Vue reactivity and ownership boundaries.
- `references/lifecycle-rendering-and-styling.md`
  Read when the task touches mount-time effects, cleanup, large lists, DOM-heavy rendering, third-party instances, or styling boundaries.
- `references/anti-patterns/token-and-styling-drift.md`
  Read when a frontend change shows the negative pattern of bypassing the project token system, relying on hardcoded local styling, or rendering internal implementation copy.
- `references/route-view-transition-root.md`
  Read when editing a Vue route-view template that is rendered by `RouterView`, wrapped in `Transition`, or receives layout classes/attrs from its parent layout.
- `references/ui-copy-boundaries.md`
  Read before adding visible UI copy, helper prose, dashboard descriptions, empty-state descriptions, or feature/navigation explanations.
- `references/ctf-vue-async-theme-route-ownership.md`
  Read when working in the CTF repo on Vue async races, duplicate actions, theme-token drift, route-view extraction, dialogs, or review-driven frontend debt.
- `../frontend-sliced-architecture/references/route-view-migration-boundaries.md`
  Read when route views are being migrated into feature slices and you need router/API/lifecycle scans, source boundary tests, or composable test guidance.
- `references/verification-checklist.md`
  Read before claiming the frontend implementation is complete.

## Output Expectations

- The implemented surface handles real user interaction, not just the ideal path.
- Async ownership is clear enough that stale responses and duplicate actions are contained.
- Component contracts remain understandable and do not hide prop mutation, dual state ownership, or unclear `v-model` flow.
- Styling choices stay within the project’s token and variable system instead of drifting into hardcoded local exceptions.
- The answer states what was tested, what was inferred, and what remains unverified.
