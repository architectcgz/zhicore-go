# Implementation Workflow

Read this file when implementing or refactoring frontend code with `frontend-engineer`.

1. Read the route view, component, or composable that actually owns the behavior before editing.
2. Identify the dominant risk first: async execution, state ownership, component contract drift, lifecycle cleanup, or rendering pressure.
3. If the task touches styling, inspect the existing shared tokens, CSS variables, and component shells before introducing new local rules.
4. Before adding or moving mock data, decide its boundary separately from page logic and UI. If the same file owns fixtures, workflow, and rendering, split at least the fixture/source boundary first.
5. For logic-bearing changes, load `test-driven-development` before writing production code and follow Red-Green-Refactor.
6. Load only the relevant reference files from `references/` instead of treating every task as the whole frontend rulebook.
7. Keep the implementation boundary small and explicit. One owner for one workflow is the default.
8. Before shrinking a large route view, write down what must remain page-owned versus what is safe to move into a child component or composable. Reduce ownership ambiguity first; line-count reduction is only a side effect.
9. For route-view template/root edits under `RouterView`, `Transition`, or parent-applied layout classes, read `references/route-view-transition-root.md`.
10. For visible UI copy changes, headings, helper text, empty states, or dashboard/workspace prose, read `references/ui-copy-boundaries.md`.
11. Validate loading, error, empty, and repeated-action behavior before closing the task.
12. Audit direct event-bound async entry points before closing the task: form submit handlers, click handlers, emit handlers, composable methods passed to components, and polling callbacks are all rejection boundaries.
13. Run the narrowest relevant tests available. If tests cannot be run, state that clearly and call out the highest-risk unverified paths.
14. After implementation and initial verification, perform a separate review pass. For leader/pipeline-classified non-trivial frontend work, use `requesting-code-review` or `reviewer`. For smaller changes, explicitly switch into review mode yourself instead of stopping at "typecheck passed".
15. Fix review findings that materially affect interaction correctness, state ownership, component boundaries, regressions, or test coverage, then re-run the impacted verification.
16. When a component mixes keyboard submit and pointer submit paths, inspect the template and handler together: check whether `@keyup.enter`, form submit, and action buttons can converge on the same async function, then verify the handler short-circuits while a request is already in flight.
17. If a reusable frontend rule gap or repeated miss is found, record it with `improvement-tracker` instead of only mentioning it in the response.
