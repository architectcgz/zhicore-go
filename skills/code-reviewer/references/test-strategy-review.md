# Test Strategy Review

Read this file when tests were added, removed, changed, or obviously should exist.

## Core test questions

- Do tests cover the behavior that can break, not just the easiest happy path
- Are boundary values, invalid inputs, empty states, and failure paths exercised
- Do assertions verify outcomes precisely instead of merely checking that code ran
- Are tests resilient, or are they tightly coupled to incidental implementation details

## Quality signals

- Good tests protect invariants, regression-prone paths, and bug-prone branches
- Mock only what must be isolated; excessive mocking can hide integration defects
- If concurrency, timing, retries, caching, or permissions are involved, test those behaviors directly when possible
- A single broad snapshot is rarely enough for risky logic changes

## Red flags

- No tests for a non-trivial bug fix or behavior change
- Tests that only mirror implementation structure without checking user or system outcomes
- Tests that pass even if the core branch is removed
- Mocked dependencies so heavily that the test proves almost nothing
- Changed behavior with stale tests that still encode the old contract
- Large component, service, or workflow decomposition with tests that only check file presence, import names, or shallow rendering while missing the behavior that could regress
- Tests updated only because the implementation stopped rendering hidden or inactive UI, without adding interaction steps that exercise the real user path

## Review stance

- Missing or weak tests are not automatically blockers, but they often become blockers when the production risk is high
- Explain what failure mode the missing test leaves exposed
- Prefer concrete test gaps over generic comments like "add more tests"
