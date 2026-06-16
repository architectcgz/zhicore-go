---
name: code-agent
description: Use when implementing general code changes or review-driven fixes that do not need a specialized frontend or backend skill, especially when shared authoring, business-comment, and timestamp conventions must stay consistent.
---

# Code Agent

Apply these rules to general code implementation unless a stronger domain skill overrides them.

## Core Conventions

- Any newly added author metadata in file headers, generated comments, or similar code annotations must use `XX`.
- When code needs class-, method-, or block-level timestamps, format them as `yyyy-MM-dd HH:mm:ss` and treat the time as Beijing time (`UTC+8`).
- Translate business rules, state transitions, approval outcomes, fallback branches, and exception handling into comments placed directly above the owning code block.
- Keep comments close to the branch, loop, handler, or persistence step that actually enforces the business rule. Do not move the explanation to a distant class header or method header.
- Explain the business trigger, object, purpose, and effect. Do not rewrite the code in prose.
- Do not use empty comments such as "validate here", "process data", or other wording that hides what is being checked, why it matters, or what changes after the branch runs.
- Do not write mechanical phrases such as "according to requirements", "per design", or "based on the document". State the business fact directly.
- Do not dump one large business essay at the top of a class or method. Split comments so each one maps to the nearby implementation block it explains.
- When business logic changes, update or remove the nearby business comment in the same edit so comments never describe old behavior.

## Comment Pattern

Prefer comments like:

- "After approval passes, this branch creates the outbound order immediately so warehouse picking and finance reconciliation stay on the same record."
- "If the customer has already submitted a cancellation request, this transition rejects duplicate approval to avoid issuing two refunds."

Avoid comments like:

- "Handle data"
- "Validate here"
- "According to the design, process approval result"

## Scope Notes

- Add business comments only where the rule, branch, or side effect is not obvious from the surrounding code.
- Do not add decorative comment blocks to simple assignments or self-explanatory control flow just to satisfy the rule.
- When the repository already has a stronger file-header or annotation convention, keep that structure but normalize author and time values to the rules above.
