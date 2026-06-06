# Page Presets

Read this file when the target belongs to an established CTF page family.

## Challenge workspace

- Use one top-tab bar for question, solutions, submissions, and review.
- Show challenge base info in the question tab only.
- Hints belong under the question statement, not under the environment area.
- Non-question tabs should hide right-side flag tools.

## Admin platform list pages

- Pages such as `platform/challenges` and `platform/images` should use flat directory rows with operational actions.
- Treat `platform/challenges` as the canonical example page for this family.
- The default shape is one continuous workbench:
  top page actions -> compact metric strip -> seamless directory -> follow-up operational sections.
- Do not add a local top-tab rail when sections such as directory, import flow, and review queue are complementary parts of one operator workflow.
- Keep import and manage actions clear, concise, and dense.
- Prefer the admin light-mode audit language from `admin-design-system.md`.
- Keep a compact metric strip at the top only when it helps the operator scan system state; the directory and table region should remain seamless.
- Render audit-heavy fields such as IDs, UUIDs, and scores with monospace-friendly treatment instead of decorative display styles.
- Keep filter and sort controls as small floating islands; advanced filtering can live in a popover without turning the whole page into stacked cards.
- Reuse shared directory primitives from sibling pages:
  `WorkspaceDirectoryToolbar` for search, sort, filter islands;
  `WorkspaceDataTable` for row dividers and hover treatment;
  the approved row-action menu language for overflow actions.
- In dark mode, row dividers and toolbar surfaces should stay soft and low-contrast. Avoid hard black lines, fixed white filter popovers, or page-local bright panels.
- The outer workspace shell may drop its border for this family when the page already reads as one large canvas and the internal sections provide enough structure.

## Environment template workspace

- Use a single workspace surface.
- Use a flat tab rail plus flat template directory rows.
- Keep side notes and boundary status as inline or rail blocks, not nested cards.

## Teacher student analysis workspace

- Use one route-level workspace shell for the whole student detail page.
- Separate page-level concerns into top tabs such as overview, recommendations, writeups, manual review, and evidence or timeline.
- Keep class and student switching in a compact rail or context block, not as another full-height main page.
- Keep the full review archive as a separate route when it is materially denser than the tabbed overview workspace.

## Student dashboard task panels

- Treat `/student/dashboard` recommendation, category, and difficulty as task panels inside one shared workspace, not as separate landing pages.
- If the route shell already provides the title, summary cards, and main actions, the panel body should begin with the primary task area.
- Do not prepend a second directory-style hero such as `训练动作目录`, `Action Directory`, `分类行动列表`, or `强度推进列表` when the list header and toolbar already explain the task.
- Do not append detached rationale sections such as `为什么先做这些` or `为什么现在先推这一档` unless they add new actionable information.
- If orientation copy is still needed, compress it into one compact inline note tied to the current state.
- Before shipping, compare sibling panels side by side and remove repeated headers, CTA strips, and explanatory blocks that reduce single-screen density.
