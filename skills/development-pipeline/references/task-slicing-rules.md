# Task Slicing Rules

Use these rules before planning or implementing.

## Core Principle

A task slice should be the smallest reviewable and verifiable unit that still represents a meaningful step.

## Good Slice Properties

A good slice has:
- one clear objective
- a narrow touched surface
- a specific validation path
- a review focus that can be stated in one sentence
- limited rollback impact

## Avoid These Patterns

Do not create slices that:
- combine schema migration, API redesign, frontend adaptation, and operational rollout in one unit unless they are inseparable
- hide major refactors inside a feature slice
- are so small that they only create churn without reducing risk
- require reviewers to infer what changed across unrelated modules
- add new behavior inside a known oversized or owner-mixed debt surface while declaring the required decomposition as future work

## Preferred Decomposition Axes

Split by one of the following when possible:
- boundary or module
- state transition or user flow
- contract or interface layer
- migration step
- validation boundary

## Required Metadata Per Slice

Each slice must state:
- goal
- touched modules or boundaries
- dependency order
- validation method
- review focus
- risk notes
- whether any touched module is a known structural-debt surface, and if yes, the concrete closure criterion for that debt in this slice
