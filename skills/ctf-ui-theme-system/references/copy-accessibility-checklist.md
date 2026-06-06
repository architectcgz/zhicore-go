# Copy, Accessibility, and Ship Checklist

Read this file before closing any CTF UI refactor.

## Color and surface

- Use semantic tokens and shared variables; avoid hardcoded palette drift.
- Accent remains theme-driven.
- Keep backgrounds solid or subtly layered.
- Do not use glassmorphism or backdrop blur in primary workspaces.

## Copy rules

- Keep structural labels and functional hints.
- Remove design-presentation text that explains layout intent.
- Preserve established bilingual structure when the page family already uses it.
- Visible UI must contain only end-user product copy.
- Do not render mock notes, option labels, developer guidance, or process narration in the page.

## Interaction and accessibility

- Keyboard behavior must work by default.
- Tabs and collapses require proper `tablist` and `tabpanel` semantics plus `aria-selected`, `aria-controls`, and `aria-expanded` where appropriate.
- Inputs need explicit labels, not placeholder-only labeling.
- Empty datasets in select controls should show an explicit disabled placeholder option.
- Missing or invalid selections should show explicit empty-state feedback instead of a silent blank panel.
- Touch targets must remain usable on mobile.

## Visual anti-patterns

- Heavy card grids inside already-carded shells
- Random pill overuse for every control or badge
- Generic indigo gradient aesthetics
- Decorative explanatory paragraphs about UI organization
- Student dashboard task panels that stack shell intro, secondary directory header, and rationale footer around the same list

## Ship checklist

- The page still has one dominant workspace shell.
- Shared route-level headers stay above the active panel; panel-specific headers stay inside the relevant `tabpanel`.
- Inactive panels remain hidden.
- No redundant divider or empty spacer sits between the tab rail and the active panel.
- If the page uses a `context-rail`, it contains utility content rather than a duplicated second main page.
- For `/student/dashboard` recommendation, category, and difficulty panels, the body starts with primary task content rather than a repeated directory hero.
- Detached rationale sections are removed unless they carry new actionable information.
- Theme tokens are used consistently.
- Keyboard semantics are complete for tabs, collapses, and forms.
- If shared selectors changed, add or tighten a focused regression test that asserts the exact selector boundary.
