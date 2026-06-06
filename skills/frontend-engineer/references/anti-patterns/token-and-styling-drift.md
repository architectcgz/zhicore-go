# Anti-Pattern: Token And Styling Drift

Use this negative case when a frontend change bypasses the project token system, creates hardcoded local styling, or relies on global overrides that are likely to break theming and consistency.

## Signals

- New spacing, font sizes, colors, shadows, z-indexes, or radii are hardcoded where project tokens exist.
- Inline styles or `!important` are used to fight ownership or specificity problems.
- Modal, drawer, table, or route-view layouts rely on global overrides instead of local shell ownership and CSS variables.
- Text overflows, long IDs, translated strings, or small viewports are not considered.
- Clickable elements lack visible affordance or keyboard focus state.

## Analysis

1. Identify existing tokens, CSS variables, shared shells, and component conventions.
2. Check desktop, mobile, overflow, focus, and theme behavior.
3. Check whether the style belongs to the component, layout shell, design system, or third-party override boundary.
4. Confirm that visible copy is real product copy, not implementation notes or design explanation.

## Recovery Direction

- Use project tokens and CSS variables for spacing, typography, color, radius, and elevation.
- Prefer `color-mix` and token-based variation over hardcoded transparency or local color literals.
- Solve conflicts through ownership, wrapper classes, or variable inheritance rather than `!important`.
- Keep implementation explanations in comments or responses, not rendered UI.
