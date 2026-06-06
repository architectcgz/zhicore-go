# Selector Ownership

Read this file when the user gives an XPath, the target mixes shared utility classes, or the bug may come from a shared selector instead of the local page.

## XPath workflow

1. Locate the route component.
2. Locate the real child component that renders the node.
3. Identify whether the bad node is:
   - page card
   - section shell
   - list item
   - Element Plus wrapper
   - nested utility container
4. Fix the actual style source instead of patching blindly around the XPath.

## Shared selector guardrails

- Before patching local CSS, check whether a shared selector in `journal-notes.css`, `teacher-surface.css`, or another shared stylesheet is overriding the node.
- When a shared selector styles plain notes inside variant shells such as `journal-notes-card` or `journal-notes-rail`, explicitly exclude summary-card variants like `.metric-panel-card`.
- If a summary card should match the established admin or teacher metric UI, verify the full shared class stack rather than adding a partial class subset.
- After a page adopts a shared metric-panel surface, do not re-style that card family with page-local `--metric-panel-*` overrides unless the override is a narrow token bridge that preserves the shared background semantics.
- If multiple pages need the same bridged treatment, promote it into a shared surface variant instead of repeating local overrides.
