# Card Usage Rules

Read this file when deciding whether a section should use a card surface or stay flat inside the page shell.

## Default rule

- Card is not the default container.
- If a block does not clearly qualify, keep it flat and solve hierarchy with spacing, headings, and separators.

## Allowed card cases

- Metric summary:
  - label + value + helper
  - dashboard snapshots, status totals, progress summaries
  - use the existing `metric-panel-*` family
- High-priority explanatory callout:
  - setup notice, security reminder, recovery guidance, one-time operational warning
  - this is the pattern used by the security settings guidance block
  - the card must justify stronger separation from surrounding content

## Disallowed card cases

- Filter bars
- Toolbars and action strips
- Form shells
- Directory or table containers
- Tab panels
- Ordinary content sections
- Repeated list items that already live inside the same page shell
- Any block that only becomes a card because "it looks cleaner"

## Boundary check

Before using a card, ask:

- Is this block primarily showing a metric?
- Is this block a high-priority explanation or warning that must stand apart from nearby content?

If both answers are no, do not use a card.

## Preferred alternatives

- Flat section with heading + helper copy
- Single divider between modules
- Embedded list rows
- Two-column layout with rail separation
- Compact note or inline callout without full card chrome

## Anti-patterns

- Wrapping filters in bordered cards above a flat directory
- Wrapping a form in a decorative card when the whole page is already a shell
- Stacking card after card inside one route, which lowers information density
- Using the same card treatment for metrics and ordinary prose
