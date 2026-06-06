# Improvements

This folder records improvement items that agents noticed but that are not yet fully handled by code, documentation, project rules, skills, or agent policies.

Use this folder for:

- recurring problems that should become project rules, skills, or agent policy updates
- architecture, code quality, workflow, testing, or documentation gaps that need follow-up
- improvements implemented in the project but not yet recorded in the relevant agent or skill
- rejected or archived improvement ideas that may otherwise be rediscovered repeatedly

Do not use this folder as a general task backlog. Actionable implementation work should move to `docs/todo/` or `docs/plan/` when it becomes ready for execution.

## Status Folders

```text
docs/improvements/
├── not-impl/
├── implemented/
├── agent-recorded/
├── rejected/
└── archived/
```

- `not-impl/`: observed improvement items that are not implemented and not yet recorded in the relevant agent, skill, or policy.
- `implemented/`: items already handled in code or documentation, but not yet recorded in the relevant agent, skill, or policy.
- `agent-recorded/`: items already recorded in the relevant agent, skill, project `AGENTS.md`, global policy, or equivalent durable rule location.
- `rejected/`: items reviewed and intentionally not pursued. Each file should explain the rejection reason.
- `archived/`: historical items that are obsolete, superseded, or no longer relevant.

## File Naming

Use a dated, descriptive filename:

```text
YYYY-MM-DD-short-topic.md
```

Example:

```text
2026-05-02-over-broad-repository-port.md
```

## Entry Template

```markdown
# Short Title

## Status

not-impl

## Context

What was observed.

## Problem

Why this matters.

## Suggested Direction

What should be improved.

## Target Owner

- skill:
- agent:
- docs:
- code area:

## Evidence

- file:
- command:
- behavior:

## Decision Log

- YYYY-MM-DD: Created.
```

## Promotion Rules

- Move an item to `implemented/` when the project has been changed but the corresponding agent or skill has not yet been updated.
- Move an item to `agent-recorded/` only after the durable rule exists in the relevant agent, skill, project `AGENTS.md`, global policy, or equivalent location.
- Move an item to `rejected/` only with a clear reason.
- Move an item to `archived/` only when it is obsolete, superseded, or no longer relevant.
