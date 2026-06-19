---
name: brainstorming
description: Use when starting creative work: creating features, building components, adding functionality, or modifying behavior. Stress-tests intent, requirements, constraints, and design choices before implementation.
---

# Brainstorming by Grilling the Design

Use this skill to turn a request into a clear, implementable direction without adding unnecessary ceremony.

The default style is `grill-me`: interview the problem relentlessly, walk the design tree one branch at a time, resolve dependencies between decisions, and give a recommended answer whenever you ask the user a question.

## Core Rule

Do not ask the user questions that can be answered by reading the codebase, configuration, docs, tests, logs, or existing patterns. Explore first; ask only when a real ambiguity remains and choosing wrong would change the outcome.

Before proposing a direction, establish an evidence base. A brainstorming pass that has not inspected the relevant local context is incomplete unless the task is explicitly greenfield or the required context is unavailable.

When asking is necessary:

- Ask one question at a time.
- Explain why the answer matters.
- Provide your recommended answer.
- Prefer concrete choices over broad open-ended prompts.
- Continue only after the ambiguity that blocks the next decision is resolved.

Ask the user instead of assuming when the answer would change one of these:

- user-visible behavior, workflow, copy, permissions, or compatibility
- data model, API contract, persistence, migration, rollout, or rollback
- ownership boundary, module split, architecture direction, or dependency choice
- destructive behavior, privacy/security posture, production operations, or cost
- acceptance criteria where multiple reasonable interpretations conflict

Do not ask about reversible implementation details, style choices already implied by the repository, or details that can be validated by reading existing code.

## Workflow

1. **Read context first**
   - Inspect relevant files, docs, configs, routes, tests, recent patterns, and existing behavior.
   - Identify what the project already implies about architecture, style, constraints, and likely implementation boundaries.
   - If the request is obviously too large, split it into the smallest coherent first task before discussing details.
   - Capture the evidence used: key files, docs, routes, tests, schemas, API surfaces, UI flows, logs, or commands inspected.
   - If context is missing, state what is missing and whether it blocks a decision.

2. **Map the decision tree**
   - List the decisions that materially affect the result.
   - Resolve prerequisite decisions before dependent ones.
   - Skip decisions that are reversible, local, or already implied by the codebase.

3. **Interrogate the uncertain parts**
   - Ask only the next most important unresolved question.
   - Include a recommendation in the same message.
   - If the user accepts the recommendation or gives enough signal, proceed without re-asking.

4. **State the working design**
   - Once the important branches are resolved, summarize the chosen direction.
   - Keep it proportional to the task: a few sentences for small changes, structured sections for larger work.
   - Cover only relevant areas: behavior, boundaries, data flow, UI states, errors, tests, rollout, or docs.
   - Classify the implementation surface before choosing a testing workflow:
     - Pure presentation work such as spacing, typography, color, static markup, or visual polish should not default to TDD.
     - Frontend logic such as state transitions, derived data, validation, permissions, async flows, reducers, stores, composables, or behavior-heavy interactions should default to TDD.
     - Mixed UI tasks should be split when practical so presentational changes stay lightweight while logic-bearing slices keep test-first discipline.

5. **Move to execution**
   - If the design is clear and no user approval gate is explicitly required, continue into the appropriate implementation/planning skill.
   - For high-risk, broad, or user-requested design work, write a short plan or spec before implementation.
   - Do not route pure UI polish directly into `test-driven-development` unless the user explicitly asks for test-first work or the change includes behavior that can be specified with failing tests.

## Question Format

Use this shape when a question is needed:

```text
这里卡在一个会影响实现方向的选择：<decision>。

我的推荐是 <recommended answer>，因为 <reason>。

你希望按这个方向走吗？
```

For multiple concrete choices:

```text
这里有两个可行方向：

1. <recommended option>：<tradeoff>
2. <other option>：<tradeoff>

我推荐 1，因为 <reason>。你确认按 1 走吗？
```

## Design Output

When enough is known, present a concise design:

- **依据**: the local evidence inspected and the assumptions still being carried.
- **目标**: what user-visible or developer-visible outcome changes.
- **边界**: what is in scope and what is intentionally left out.
- **方案**: the main implementation direction and affected modules.
- **风险**: important failure modes, compatibility concerns, or unclear assumptions.
- **验证**: the smallest sufficient checks.

For frontend work, make the testing stance explicit in one short line:

- `TDD`: required for logic or behavior changes.
- `No TDD`: acceptable for pure presentational UI changes; use focused visual or manual verification instead.

Do not write a formal spec, create a docs file, commit a design document, or run a spec review loop unless the user explicitly asks for that level of process or the task is large enough that skipping it would be irresponsible.

## Principles

- Prefer evidence from the repository over user interrogation.
- Prefer the smallest coherent change over broad redesign.
- Prefer project conventions over generic best practices.
- Be direct about weak requirements, hidden risks, and missing acceptance criteria.
- Keep momentum: once the design is good enough for the task, implement.
