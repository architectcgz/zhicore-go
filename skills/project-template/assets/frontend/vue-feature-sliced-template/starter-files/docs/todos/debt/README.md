# Technical Debt Index

This directory tracks unresolved technical debt that survives code review, post-module review, or follow-up analysis.

## Purpose

- Keep one debt file per review batch, module, or focused debt topic.
- Preserve the source review context without turning one `DEBT.md` into an unmaintainable ledger.
- Make it clear which debt is still open, who owns it, and what closes it.

## Directory Rules

- Active debt files stay here.
- Closed or obsolete debt files move to `archive/`.
- Use `_template.md` as the starting point for new debt entries.

Suggested naming:

- `YYYY-MM-DD-<scope>-debt.md`
- `YYYY-MM-DD-<module>-post-review-debt.md`
- `YYYY-MM-DD-<topic>-technical-debt.md`

## Debt File Format

Each debt file should follow this structure:

```md
# <Title>

> Status: Open | In Progress | Blocked | Closed
> Priority: Critical | High | Medium | Low
> Owner: <team / module / person>
> Source Review: [<review file>](../../reviews/...)
> Scope: <module / feature / route / workflow>
> Last Updated: YYYY-MM-DD

## Summary

One short paragraph describing why this debt exists and why it was not closed immediately.

## Debt Items

### D1. <Short debt title>

- Type: Architecture | Contract | Test | Performance | Security | DX | Operations | UI
- Surface: <files / modules / routes / runtime path>
- Trigger: <when this debt becomes visible>
- Current Impact: <today's cost or risk>
- Failure Mode: <what breaks, drifts, or slows down>
- Suggested Direction: <bounded fix direction, not a full implementation plan>
- Exit Condition: <what must be true to consider this debt closed>
- Related Evidence: <tests / docs / issue / review links>

### D2. <Short debt title>

- ...

## Prioritization Notes

- Why these items are ordered this way.
- Which items can be grouped into the same remediation slice.

## Next Review Trigger

- Example: "Re-check after the next route ownership refactor."
- Example: "Re-check before the next auth workflow release."
```

## Writing Rules

- Record only unresolved debt here. Fixed findings belong back in code, tests, or architecture docs.
- Keep each debt item actionable: trigger, impact, direction, exit condition.
- Do not copy the full review narrative here; link back to the review file instead.
- If a debt item becomes the scope of a formal implementation effort, link the plan and keep this file as the debt index.
