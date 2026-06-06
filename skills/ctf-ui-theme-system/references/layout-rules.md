# Layout Rules

Read this file when the task changes shell structure, panel organization, spacing, rails, lists, or action layout.

## Root workspace shell

- Use one dominant workspace surface per page.
- Preferred structure:
  - `workspace-shell`
  - `workspace-topbar`
  - `top-tabs` when the route contains multiple page-level views
  - `content-pane`
  - optional `context-rail`
- The page should fill the main area; avoid half-height shells that leave large dead space.

## Flattening principle

- Prefer separators, spacing, and hierarchy over stacked cards.
- Internal sections should feel embedded in the same workspace.
- Replace repeated mini-card stacks with section heads, dividers, flat rows, or compact rails.

## Module boundaries

- Use spacing first between modules.
- Keep only one explicit divider for each module boundary.
- Do not stack a previous block's `border-bottom` and the next block's `border-top` on the same boundary.

## Top tab rail pattern

- Route-level tabs sit directly below `workspace-topbar` and before the main content region.
- Route-level tabs synchronize with a stable `?panel=` query key.
- If a header or action group only applies to one panel, move it inside that `tabpanel`.
- Non-active panels must not repeat overview-only headers, copy, or actions.
- Do not override shared tab visibility by forcing every `.tab-panel` visible in local CSS.
- Do not use tabs to split "overview" metrics from the directory they describe.
  For backend-style management pages, prefer the SaaS workbench pattern in `references/saas-workbench-pattern.md`.

## Content pane and context rail

- `content-pane` holds the active task content.
- `context-rail` holds switching, navigation shortcuts, export entry points, or compact utility notes.
- The rail must not duplicate the main task body.

## Directory and list pattern

- Use one header row plus flat data rows.
- Prefer shared column tracks between header and rows.
- Avoid loose `auto` tracks on key columns such as category, status, or actions.
- Keep pagination attached to the list section, directly after the last row.
- Keep the section heading, filter bar, and list body inside the same `workspace-directory-section`.
- Prefer a light `list-heading` over a separate filter card header when the section purpose is already obvious from the list title.
- When the page also needs KPIs, place a metric strip directly above the directory section instead of moving the list into a sibling tab.

## Filter bars

- List and directory filter bars should auto-apply instead of relying on an explicit submit button.
- Text input filters should use a short debounce, around `250ms`, before refreshing results.
- Status, date, and toggle filters should join the same auto-apply flow rather than introducing a separate "应用筛选" action.
- Keep only actions that are not part of filtering itself, such as reset, refresh, export, or create.
- Do not render explanatory copy, “激活筛选 X 项”, or a dedicated filter card title when the surrounding section already makes the context clear.
- Order high-frequency filters first, then keyword search, then non-filter actions such as clear or export.
- When filter count grows, keep high-frequency filters visible and collapse low-frequency filters behind a compact “更多筛选 / 收起筛选” toggle.
- Expanded low-frequency filters should reuse the same control width as the primary select columns instead of switching to a different visual scale.
- A reset or clear action should live in the same action row as the collapse toggle rather than occupying a full extra row by itself.

## Action controls

- Primary row action gets accent-tinted styling.
- Secondary row action stays neutral-outline.
- Keep row actions compact and consistent:
  - min-height `34px` desktop
  - min-height `36px` mobile
  - radius around `10px`
  - gap `6px` to `8px`
- Use visible `:focus-visible` states and `role="group"` for grouped row actions.
- If a shortcut applies to one concrete row, it belongs in the row action group, not in a floating summary block above the list.
