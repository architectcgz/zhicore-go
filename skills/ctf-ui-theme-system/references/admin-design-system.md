# Admin Design System

Read this file when the task touches admin pages, contest operations, challenge management, audit lists, or other dense backend-style CTF surfaces.

Source notes:
- `ctf/code/frontend/docs/design-system.md`
- `references/examples/admin-example.tsx`
  This file now includes the canonical full directory-section example:
  `list-heading` -> lightweight toolbar islands -> optional advanced-filter popover -> flat list body -> attached pagination.

## Positioning

- Admin pages should feel audit-ready, dense, and controlled.
- Use a light workspace with restrained blue or slate accents, not neon security styling and not generic OA dashboard chrome.
- Prefer structure, spacing, typography, and separators over decorative containers.

## Core Principles

- Data first:
  Important data should read before decoration.
- Audit ready:
  IDs, UUIDs, IPs, scores, and similar operational fields should prefer `font-mono`, tighter tracking, and uppercase-friendly styling.
- Restrained palette:
  Keep the base in `slate` or blue-gray tones; reserve vivid color for status, active navigation, and primary actions.
- Defensive interaction:
  Dangerous actions must stay visually distinct and usually sit behind stronger confirmation.
- Decardified lists:
  Keep metric cards at the top only when they summarize the page; the main directory or table region should stay flat inside the workspace.

## Layout Pattern

- Page background uses a cool light base close to `slate-50`.
- Sidebar and topbar can use white surfaces with subtle dividers.
- Secondary tabs sit below the topbar and use a clear active underline.
- The main content area starts with a compact title and actions, then an optional metric band, then a seamless list or table section.
- For admin directories that mix macro stats and object rows, default to the SaaS workbench pattern from `references/saas-workbench-pattern.md`:
  page header -> KPI strip -> seamless directory.
  Do not split those layers into fake sibling tabs such as "总览" and "列表".

## Information Architecture

- Admin routes should usually follow this reading order:
  Sidebar navigation -> topbar context -> page title and primary actions -> secondary tabs if needed -> KPI strip -> directory or table.
- Use one clear operational page title, then place utility actions on the same horizontal line when space allows.
- Secondary tabs should switch peer workflows such as environment management, monitoring, announcements, or export; do not stack those workflows vertically on the same screen.
- Never use top tabs just to separate metrics from the list they summarize.

## Navigation and Shell Rules

- Sidebar stays clean and bright with a restrained SaaS shell feel:
  white surface, subtle right border, rounded active item, compact icon plus label rhythm.
- Active sidebar item should use a soft blue background plus stronger weight, not a heavy filled block.
- Topbar actions should be circular or near-circular icon buttons with quiet hover feedback.
- Account entry can stay compact and pill-like, but should not visually outrank the page title.
- Collapse affordances and floating shell controls should read as utilities, not as primary CTAs.

## Component Rules

- Metric cards:
  Allow for core KPIs only. Use concise labels, one strong number, and one small helper or trend.
- Seamless directory:
  Search, filter, sort, and pagination controls should behave like light floating islands instead of a full enclosing card.
- Tables:
  Prefer fixed or predictable column tracks, truncation for long text, and quiet row hover states.
- Dropdowns and menus:
  Use explicit layering, visible shadow separation, and smart upward placement near the bottom of the viewport.
- Status badges:
  Reuse shared semantic classes such as `.admin-badge-*`; do not handwrite per-page colors.

## Visual Language

- Headings prefer `slate-900` with strong weight and tight tracking.
- Helper labels, panel eyebrows, and metadata can use very small uppercase text with muted slate tone.
- Numbers, IDs, UUIDs, scores, and similar audit fields should visibly differ from prose through monospace treatment.
- Accent color should concentrate on active navigation, primary actions, focus rings, and successively important row actions.
- Hover states should stay controlled: subtle tint, subtle border lift, or subtle text emphasis rather than dramatic motion.

## Metric Card Rules

- Use a short uppercase label, one large value, and one compact helper line.
- Keep cards visually lightweight: white surface, fine border, soft shadow, slight accent reaction on hover.
- Metric cards summarize state; they should not become mini dashboards with multiple nested controls.
- A top metric strip is optional. If the page already starts with dense operational content, do not add cards just for symmetry.

## Directory and Table Rules

- The table section should sit directly on the page shell instead of inside a giant wrapper card.
- Search, sort, and high-frequency filters act as separate islands aligned around the list header.
- Use `table-fixed` or equivalent predictable tracks for audit-heavy lists.
- Long title and ID columns should truncate and preserve full value via native title or equivalent reveal.
- Rows should use a single horizontal divider and a restrained hover background.
- Operational columns such as status and actions should stay narrow and predictable across sibling pages.

## Row Action Rules

- Each row should expose one obvious primary action, then a compact overflow trigger for the rest.
- Reuse the approved row-action trigger and overflow menu treatment across sibling admin pages. If a page matches an existing workspace directory pattern such as `platform/challenges`, copy that language instead of renaming the menu header, repainting the trigger, or introducing a page-local white-panel variant.
- Overflow menus need strong layer separation and should open upward for bottom rows when needed.
- Neutral actions stay muted; dangerous actions escalate through orange or red semantic treatment.
- Do not present destructive actions with the same visual weight as ordinary edit or copy actions.
- Avoid floating shortcut cards such as "进入工作台". If an action applies to a specific record, render it inside that row's action column.

## Filter and Sort Rules

- The default state is lightweight keyword search plus a compact sort control.
- Advanced filters may live in a popover or panel when there are multiple dimensions to combine.
- If a popover groups several edits together, one apply button is acceptable inside that popover.
- Keep the visible toolbar short; do not turn every optional filter into a permanent full-width form.
- Result count, page size, and pagination should use the same quiet island language as the toolbar.

## Interaction Guidance

- High-frequency filters should stay visible and lightweight.
- Keyword search can debounce; direct selects and toggles should feel immediate.
- If advanced filtering is grouped in a popover or panel, it may expose one apply action after the user edits multiple fields.
- Primary actions use the blue accent; neutral utilities stay outline or white-surface.
- Destructive row actions should remain visually isolated from ordinary actions.

## Implementation Notes

- Prefer shared CSS classes or tokens over page-local hex values.
- Avoid wrapping the whole list region in another white card.
- Do not let status chips, IDs, and data numerals drift into decorative typography.
- Treat `references/examples/admin-example.tsx` as a React structure sample, not as production-ready component code.
