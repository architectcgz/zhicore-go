---
name: drawio-business-diagrams
description: Use when generating, reviewing, or refining diagrams.net / draw.io XML diagrams for implementation plans, architecture designs, business workflows, migrations, runtime gates, owner boundaries, decision flows, or blocker paths.
---

# Draw.io Business Diagrams

## Overview

Use this skill to turn plans or architecture docs into reviewable draw.io XML that shows business relationships, owner boundaries, decision branches, data side effects, blockers, and non-goals. Keep `SKILL.md` as the index; load references only for the needed detail.

## Reference Map

| Need | Read |
| --- | --- |
| Generate a full reusable prompt for a plan / architecture / workflow diagram | `references/generation-prompt.md` |
| Apply draw.io node styles, edge styles, diamond decision-node rules, colors, and XML snippets | `references/drawio-styles.md` |

## Required Rules

- Always keep a reviewable draw.io XML source; PNG exports are only auxiliary.
- Mark target design / planned content clearly when code has not landed.
- Draw actual business relationships: ownership, cardinality, calls, reads, writes, states, blockers, side effects, and forbidden paths.
- Prefer service-level architecture over repository-level overview. When the source has service boundaries, create concrete diagrams for each relevant service first; cross-service overview pages are only indexes or dependency maps.
- For service detail diagrams, draw named business use-case workflows, not static module topology. Start from a concrete API / handler / job and continue through application orchestration, guards, repository or adapter calls, DB writes, outbox/events, async consumers, side effects, and visible results.
- A diagram that only shows boxes such as `handler -> application -> domain -> repository -> DB` without a named use case, concrete tables/events, branch conditions, and async outcomes is incomplete.
- Treat artifact placement as project-owned. Follow the repository's documentation rules; if none exist, suggest a reasonable default such as `docs/architecture/services/<service>/` for service diagrams and `docs/architecture/services/_overview/` for cross-service overviews.
- Use diamond / rhombus nodes for every decision, choice, condition, guard, or branch. Do not represent `if / switch / 是否 / 选择 / 分支` with ordinary rectangles.
- Label every diamond outgoing edge with a condition such as `yes / no`, `pass / fail`, `same target / different target`, or `allowed / blocked`.
- Do not invent queues, services, caches, middleware, events, or external systems that are not in the source material.

## Workflow

1. Read the project documentation rules for diagrams before producing or changing diagram artifacts.
2. Identify the concrete service scope and the named use case(s) to draw, including API / handler / job entrypoints, application use cases, guards, tables, events, consumers, side effects, and external dependencies.
3. Read the source plan / architecture / contract / code paths that define the business relationships.
4. Load `references/generation-prompt.md` when creating a prompt or complete diagram request.
5. Load `references/drawio-styles.md` before writing or editing draw.io XML styles.
6. Validate the XML by importing it or exporting through draw.io CLI when available.
7. If rendering images, place them according to project documentation rules; when the project has no rule, state the suggested default path and that images are not the fact source.

## Quick Check

- Does the diagram show who owns each business decision?
- Does it explain the selected service concretely, rather than only showing a whole-system overview?
- If it is a detail diagram, can you trace one named use case end to end from handler/job through transaction/outbox/consumer/result?
- Does artifact placement follow project rules, or clearly state a default suggestion when the project has no rule?
- Are facts providers, owners, consumers, and data tables visually distinct?
- Are all decisions diamonds with labeled outgoing edges?
- Are blockers, rollback limits, and deleted legacy paths visible?
- Is the diagram source reviewable XML rather than only an image?
