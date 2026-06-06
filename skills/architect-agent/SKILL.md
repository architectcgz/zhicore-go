---
name: architect-agent
description: Use when analyzing a codebase before implementation, especially to map module boundaries, call paths, data flow, dependencies, compatibility constraints, complex aggregated queries, over-broad repositories or services, and the smallest safe change surface for a planned task. Also use when manually asked to inspect broad aggregation, over-composed queries, over-broad repositories/services, dashboard/overview endpoints, or candidates for splitting into clearer subdomain ports and queries.
---

# Architect Agent

Analyze the codebase deeply enough to guide implementation without guessing.

## Use When

- The task needs structural understanding before code changes begin
- You need to map current behavior, dependencies, module boundaries, or risk points
- The codebase is large enough that implementation without prior analysis would be guesswork
- The user asks for aggregation inspection, query decomposition, subdomain query splitting, broad dashboard/overview query review, or repository/service responsibility splitting

## Do Not Use

- Straightforward changes where the owning files and behavior are already obvious
- Pure implementation after architecture-level questions have already been answered

## Core Guardrails

1. Read code, config, and tests before drawing conclusions.
2. Prefer concrete file paths, call paths, and data flow over abstract summaries.
3. Surface unknowns explicitly instead of smoothing them over.
4. Optimize for the smallest safe change, not the most ambitious redesign.
5. Treat scan output as candidates only; confirm every finding by reading the actual code and callers.

## Workflow

1. Identify the modules, services, or packages relevant to the task.
2. Trace the main execution path and the supporting dependencies.
3. Map current behavior, state ownership, side effects, and compatibility assumptions.
4. Call out risk areas such as hidden coupling, schema assumptions, concurrency, or rollout constraints.
5. Recommend a minimal implementation approach grounded in the actual codebase.

## Aggregation And Port Inspection

Use this path when manually asked to inspect complex aggregation, broad queries, dashboard data loaders, overview endpoints, report builders, or repositories/services that combine several use-case responsibilities.

Before concluding, look for:

- one function or method querying multiple business domains
- query functions with optional filters from unrelated features
- SQL or ORM joins that mix unrelated aggregates
- service methods returning mixed DTOs for several UI sections
- generic names such as `getDashboardData`, `getOverview`, `getSummary`, `getStats`, `getReport`, `queryAll`, or `searchEverything`
- result objects that contain several independently owned subdomain sections
- changes where a small feature requires editing a broad aggregated query
- repository or service interfaces that mix creation, mutation, member management, read models, user lookup, existence checks, uniqueness checks, or policy checks
- interfaces named around a whole aggregate but used by unrelated command and query flows

When the user asks for a repository-wide aggregation inspection, run this helper from the target repository root:

```bash
node /home/azhi/.codex/skills/architect-agent/scripts/inspect-aggregation.mjs
```

Then read the matched files directly. Do not recommend a split based only on the helper output.

If the candidate is an over-broad repository, service, port, or gateway, read `references/cases/over-broad-repository-port.md` for the detailed decomposition pattern.

For each confirmed aggregation candidate:

1. Identify the subdomains or use-case responsibilities currently coupled together.
2. Map each responsibility to its owning table, model, service, route, job, UI consumer, command flow, or query flow.
3. Explain why the current aggregation is risky or acceptable.
4. Propose smaller ports or queries with explicit names and result ownership.
5. Prefer use-case-oriented split names over table-oriented buckets when those are the actual consumers.
6. Preserve existing external behavior unless the user asked for API changes.
7. Prefer incremental extraction over broad rewrites.
8. Call out transaction, consistency, permission, ordering, pagination, caching, and performance risks.

Prefer splitting when subparts differ in lifecycle, command/query direction, permission model, cache strategy, pagination, ordering, consistency needs, UI consumer, transaction boundary, or release cadence.

## Output Expectations

- Relevant modules
- Affected files
- Current behavior
- Aggregation candidates, when applicable
- Proposed approach
- Risks
- Unknowns
