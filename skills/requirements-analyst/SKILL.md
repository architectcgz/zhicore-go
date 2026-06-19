---
name: requirements-analyst
description: Use when requirements are vague, risky, cross-module, or likely to hide edge cases and need explicit scope, assumptions, acceptance criteria, dependencies, and non-functional constraints before implementation.
---

# Requirements Analyst

Turn ambiguous requests into implementable requirements with explicit assumptions and risk boundaries.

## Use When

- Requirements are underspecified, contradictory, or likely to hide edge cases
- The task spans multiple modules, services, or user roles
- You need structured acceptance criteria before planning or implementation

## Do Not Use

- Straightforward tasks with already-clear scope and acceptance criteria
- Pure implementation after requirements are already stable

## Core Guardrails

1. Start from the actual user goal and business intent, not from a preferred technical solution.
2. Distinguish confirmed requirements from inferred assumptions.
3. Surface edge cases, failure cases, and dependencies early.
4. Keep analysis proportional to complexity; do not over-document trivial work.
5. Prefer concrete acceptance criteria over vague descriptions of success.

## Workflow

1. Read the requirement source and identify the core user outcome.
2. Clarify functional scope: must-have, should-have, and optional behavior.
3. Enumerate normal flows, edge cases, and failure scenarios.
4. Identify non-functional requirements such as performance, security, observability, compatibility, and rollout constraints.
5. Record dependencies, external systems, and technical constraints.
6. Summarize risks and unresolved questions explicitly.

## Output Expectations

- Requirement summary
- Assumptions
- Functional requirements
- Edge cases and failure cases
- Non-functional requirements
- Constraints and dependencies
- Risks
- Open questions
- Acceptance criteria
