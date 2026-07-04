# Lifecycle, Rendering, and Styling

Read this file when the task involves mount-time side effects, DOM-heavy rendering, third-party instances, or CSS boundaries.

## Lifecycle and cleanup

- Treat everything created in `onMounted` as something that must be cleaned in `onUnmounted`.
- Clean up `setInterval`, timers, `addEventListener`, observers, sockets, and third-party instances.
- For global events attached to `window` or `document`, prefer auto-cleanup hooks or wrappers such as VueUse `useEventListener` instead of manual listener bookkeeping.
- Dispose libraries such as ECharts explicitly.
- If the page can mount and unmount repeatedly, assume leaks will accumulate unless you prove otherwise.

## Large lists and rendering

- If a list can grow large, evaluate pagination, server slicing, or virtualization before shipping raw `v-for`.
- Do not render log, chat, rank, or history panels wholesale if they can realistically hit hundreds or thousands of rows.
- Keep list identity stable. Do not key by array index when item identity or local row state matters.

## Styling and CSS

- Prefer the project's existing utility-first or token-based styling approach over ad hoc component-local CSS.
- Avoid arbitrary magic-number sizing unless there is a clear, documented reason tied to the design system or runtime layout constraint.
- Use custom CSS when it improves clarity, but keep selectors shallow.
- Treat Vue deep selectors (`:deep()`, `::v-deep`, `/deep/`, `>>>`) as last-resort scoped-style escapes, not as a normal styling pattern.
- Before using a deep selector, check whether the same outcome can be owned through props, variants, slots, wrapper classes, design tokens, CSS variables, or a local component shell.
- Allow deep selectors only for narrow third-party or legacy component overrides where the component does not expose a stable styling contract. Keep the selector as short as possible and add a short comment naming the blocked API or component boundary.

## Spacing ownership and layout boundaries

- Default every reusable component to zero outer margin. Base components such as cards, buttons, inputs, and metric blocks should not push surrounding layout by themselves.
- Treat component padding as internal structure and page or section spacing as external layout. `padding` belongs to the component; `margin` belongs to the parent layout only when `gap` cannot express the relationship.
- Prefer `flex` or `grid` plus `gap` for sibling spacing. Do not rely on child `margin-right` or `margin-bottom` as the primary layout mechanism.
- Let the parent container own vertical rhythm. If a header and a metric group need spacing, put them in one parent column with a shared `gap` instead of stacking `margin-top` and `margin-bottom` across children.
- Use shared spacing tokens for `padding`, `margin`, and `gap`. Avoid one-off values such as `22px` unless there is a documented design-system exception.
- When debugging layout drift, check for stacked spacing first: parent `padding`, child `margin`, and wrapper spacing often accumulate into false misalignment.
- Inside a component, prefer internal `flex` or `grid` gaps over ad hoc child margins. A value block inside a card should usually be spaced by the card's internal layout, not by the value block pushing itself away.
- Treat slot boundaries as spacing boundaries. If a wrapper like `.header-slot` or `.body-slot` already defines spacing, require the slotted content to stay margin-less so spacing is owned in one place.
- When a component must expose layout control, prefer explicit props, variants, or documented wrapper patterns over hidden CSS margins baked into the component shell.

## Z-index and stacking contexts

- Global overlay surfaces such as dialogs, drawers, popovers, toasts, and message layers should mount with `<Teleport to="body">` so they are not trapped by a parent stacking context.
- Do not hardcode ad hoc values such as `z-index: 9999` in business components. Use the design system's z-index tokens or a shared overlay scale.
- Treat unexpected overlay clipping as a stacking-context bug first. Check transformed parents, positioned ancestors, `overflow`, and local `z-index` escalation before adding another layer value.

## User-visible content boundary

- Render only real end-user product content in the interface.
- Keep design rationale, structure notes, TODO markers, placeholder commentary, and implementation hints out of the rendered UI.
- Put that material in code comments, documentation, or assistant output instead unless the task explicitly requires explanatory text as part of the product experience.

## Best Practice: High-Quality Detail Modals

When implementing detail views or resource editors (e.g., Audit Executor Detail), follow these premium styling patterns:

- **Unified Background**: Avoid "card-within-a-card" visuals for simple detail lists. Set header, body, and footer to the same solid surface color (e.g., `--color-bg-surface`) to create a cohesive panel feel.
- **De-cardification**: Instead of wrapping every data point in a bordered box, use a clean grid with generous spacing (`gap`). This reduces visual noise and makes the information more scannable.
- **Frosted Glass (Backdrop Blur)**: Use `backdrop-filter: blur(12px)` on the modal overlay. This provides a sophisticated sense of depth, obscuring underlying page content while maintaining focus on the modal without needing pitch-black overlays.
- **Visual Cues**: Use small, consistent icons next to labels to aid rapid identification of data types (e.g., Fingerprint for IDs, User for names).
- **Typography Hierarchy**: Use bold, uppercase, letter-spaced small fonts for labels and larger, high-weight fonts for values. This creates a professional, dashboard-like aesthetic.
- **No Teleporting CSS Gaps**: Since modals are usually `<Teleport to="body">`, ensure their specific styles (including background overrides) are defined in a non-scoped `<style>` block or a global stylesheet to prevent them from being "trapped" by component scoping.

## Best Practice: Premium Dashboard Metric Panels

When building top-level dashboard metrics (e.g., Contest or Audit Log summaries), use the following patterns for a high-end terminal feel:

- **Adaptive Grid Columns**: Use CSS variables (e.g., `--metric-panel-columns`) within a global grid class. This allows components to switch between 2, 3, or 4 columns via simple utility classes without rewriting layout logic.
- **Floating Dividers**: Instead of full-height solid borders, use pseudo-elements (`::after`) to create vertical dividers that are vertically centered and only cover 60-80% of the height. Apply a subtle gradient fade to the ends for a "suspended" effect.
- **Micro-Contrast Borders (The Halo Ring)**: Use a 1px solid border combined with a 1px "halo" shadow ring (`box-shadow: 0 0 0 1px`). This creates a sharp, physical edge on frosted glass backgrounds.
- **Visual Cue Icons**: Every metric item MUST pair its label with a relevant Lucide icon (e.g., Trophy for totals, Activity for status). Icons should be placed within the label's flex container (usually `justify-content: space-between`) to anchor the visual weight and provide immediate context. Without icons, the large numeric values can feel unanchored and visually monotonous.
- **Theme-Aware Softening**: High-contrast white borders look great in light mode but are aggressive in dark mode. Define border and halo colors as variables and significantly reduce their opacity (e.g., from 0.8 to 0.05) in dark mode to maintain a soft, glow-like quality.
- **Typography Sizing**: For critical metrics, use large bold values (e.g., 30px+) paired with clearly legible, high-weight labels (e.g., 15px bold). Zero-pad small numbers (e.g., `01`, `08`) to maintain a steady visual rhythm and dashboard aesthetic.

