---
name: improvement-tracker
description: Use when an agent discovers an improvement gap, repeated mistake, missing skill or agent rule, unrecorded project convention, or optimization that should be captured under docs/improvements instead of only mentioned in a response. Records items with statuses such as not-impl, implemented, agent-recorded, rejected, and archived.
---

# Improvement Tracker

Record durable improvement items that agents discover while working.

## Use When

- An agent finds a recurring problem that should become a skill, agent policy, project rule, or global rule
- A project improvement is implemented but not yet recorded in the relevant agent, skill, or policy
- A review, architecture analysis, backend/frontend task, or debugging session reveals a gap that should not be lost after the response
- The user asks to record an optimization, missed consideration, or agent improvement item

## Do Not Use

- Ordinary task backlog items that belong directly in `docs/todo/`
- Execution plans that belong in `docs/plan/`
- Current architecture facts that belong in `docs/architecture/`
- API or schema contracts that belong in `docs/contracts/`

## Statuses

- `not-impl`: observed but not implemented and not recorded in the relevant agent, skill, or policy
- `implemented`: implemented in the project, but not recorded in the relevant agent, skill, or policy
- `agent-recorded`: recorded in the relevant agent, skill, project `AGENTS.md`, global policy, or equivalent durable rule location
- `rejected`: reviewed and intentionally not pursued
- `archived`: obsolete, superseded, or no longer relevant

## Recording Hook

From the target repository root, run:

```bash
node /home/azhi/.codex/skills/improvement-tracker/scripts/record-improvement.mjs --root . --status not-impl --title "Short title" --body "Observed issue and context"
```

The script creates `docs/improvements/`, the status folders, `README.md`, and a dated entry file when missing.

## Rules

1. Prefer `not-impl` for newly discovered gaps.
2. Use `implemented` only when the project change exists but the durable agent/skill/policy update is still missing.
3. Use `agent-recorded` only after the durable rule exists.
4. Use `rejected` only with a clear reason.
5. Use `archived` only for obsolete or superseded items.
6. Do not only mention durable improvement gaps in the final response; record them when the repository has or should have `docs/improvements/`.

## Output Expectations

- State the improvement file path created or updated.
- State the selected status and why it fits.
- If no record was created, state why the item belongs somewhere else.
