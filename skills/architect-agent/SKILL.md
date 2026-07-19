---
name: architect-agent
description: Use when analyzing a codebase or implementation plan before backend changes, especially around module boundaries, call paths, production runtime, external dependencies, workers, queues, outbox delivery, compatibility constraints, complex aggregation, over-broad repositories/services, or migration target-state convergence.
---

# Architect Agent

Analyze the codebase deeply enough to guide implementation without guessing.

## Core Guardrails

1. Read code, config, and tests before drawing conclusions.
2. Prefer concrete file paths, call paths, and data flow over abstract summaries.
3. Surface unknowns explicitly instead of smoothing them over.
4. Optimize for the smallest complete target-state change, not the smallest diff and not the most ambitious unrelated redesign.
5. Treat scan output as candidates only; confirm every finding by reading the actual code and callers.
6. When the user explicitly selects a framework, driver, architecture, storage model, or runtime platform, do not preserve legacy seams merely to reduce risk or diff size. Retained legacy paths require explicit final-state justification or user approval.
7. Research before escalating ambiguity: inspect repository evidence first, then consult current official primary sources when framework recommendations, compatibility, or migration mechanics are unknown. Do not outsource researchable technical facts to the user.
8. When the user may not yet understand or be able to frame the problem, first synthesize a research-question brief: core tension, concrete subquestions, relevant mechanisms and scenarios, candidate decision models, intended sources, and expected deliverables. Ask the user to validate that framing before investing in deep research.

## Migration Target-State Gate

For migrations, replacements, standardization, or “use X everywhere” requests, distinguish these before recommending a change:

1. **Starting point:** current initialization, driver/provider, runtime owner, adapters, wrappers, and dependencies.
2. **Final production path:** the one initialization and call path that should remain after completion.
3. **Required removals:** legacy imports, constructors, drivers, adapters, compatibility layers, config, and tests that must disappear.
4. **Stable retained boundaries:** components intentionally kept because they are part of the final architecture, not because they make the diff smaller.
5. **Proof:** searches, architecture checks, and behavior tests proving the old default path is gone.

If a proposed “safe” plan would require an immediate second migration before the user-selected framework or architecture can honestly be called adopted, the proposal is incomplete. Expand the current plan to include that convergence or obtain explicit approval for the staged boundary.

## Workflow

1. Identify the modules, services, or packages relevant to the task.
2. Trace the main execution path and the supporting dependencies.
3. Map current behavior, state ownership, side effects, and compatibility assumptions.
4. Call out risk areas such as hidden coupling, schema assumptions, concurrency, or rollout constraints.
5. Recommend a minimal implementation approach grounded in the actual codebase.
6. If a material boundary remains, present the evidence, recommended option, alternatives, and their impact before asking the user. Ask only about product intent, risk tolerance, downtime, cost, staging, or another boundary the evidence cannot decide.

## Required Gate Routing

- For production runtime, dependency lifecycle, workers, queues, consumers, outbox, health, observability, or implementation-plan review, read `references/production-runtime-gates.md` before giving a verdict or plan.
- For complex aggregation, broad queries, dashboards/overviews, or responsibility splitting, read `references/aggregation-inspection.md`.
- For an over-broad repository, service, port, or gateway, read `references/cases/over-broad-repository-port.md` before recommending a split.
- Do not approve a plan that defers a known blocker or major risk on the touched production surface to later debt merely because existing unit tests pass or a deadline is close.

## Aggregation And Port Inspection

When the user asks for a repository-wide aggregation inspection, run this helper from the target repository root:

```bash
node /home/azhi/.codex/skills/architect-agent/scripts/inspect-aggregation.mjs
```

Then read the matched files directly. Do not recommend a split based only on the helper output.

## Output Expectations

- Relevant modules
- Affected files
- Current behavior
- Aggregation candidates, when applicable
- Proposed approach
- Risks
- Unknowns
- Production runtime gate verdict and missing acceptance evidence, when applicable
