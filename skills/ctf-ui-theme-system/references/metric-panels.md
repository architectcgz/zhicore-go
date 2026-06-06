# Metric Panels

Read this file when the task touches summary cards, `metric-panel-*`, `progress-card`, `journal-note`, or an exact summary-card selector/XPath.

## Ownership rules

- Treat `metric-panel-card` as a semantic summary-card variant, not as a decorative class.
- If the page should match the approved summary-card stack, wire the full class stack:
  - container: `metric-panel-grid` plus the shared surface class
  - item: `progress-card metric-panel-card`
  - text: `progress-card-label/value/hint` plus `metric-panel-label/value/helper`
- Summary-card helper copy must explain the metric. Do not leave unit-only placeholders.

## Exact-node rule

- When the user gives an exact selector or XPath, resolve it to the concrete template node before editing.
- After the patch, self-check that the exact target node, not only its parent shell, carries the required shared class stack.

## Shared selector guardrails

- Plain note variants such as `journal-notes-card` and `journal-notes-rail` must explicitly exclude `.metric-panel-card`.
- Do not let broad `.journal-note` selectors restyle summary cards by accident.
- Mixed nodes such as `journal-note metric-panel-card` still belong to the metric-panel surface family.

## Local override limits

- After a page adopts a shared metric-panel surface such as `metric-panel-default-surface`, page-local CSS must not take visual ownership back through ad hoc `--metric-panel-*` overrides.
- Allowed local overrides are narrow:
  - layout tokens such as responsive column count or spacing
  - surface bridge tokens when mapping the same card language onto another established shell family
- Local overrides must preserve the shared layered background semantics instead of flattening the card into a page-only look.
- Explicitly forbidden:
  - replacing the shared `radial-gradient + linear-gradient` metric panel background with a single flat `color-mix(...)`, solid color, or other page-local plain fill
  - downgrading a workspace-themed summary card from accent-tinted border/background tokens back to neutral page chrome only because the page feels "too colorful"
  - using a page-local `--metric-panel-background` override as a shortcut for visual simplification when the real issue is shell token selection
- If a workspace variant needs a calmer treatment, keep the layered structure and bridge through workspace tokens such as `--workspace-brand`, `--workspace-panel`, `--workspace-panel-soft`, or extract a dedicated shared surface variant.
- If multiple pages need the same bridged treatment, extract a shared surface variant instead of repeating local overrides.
