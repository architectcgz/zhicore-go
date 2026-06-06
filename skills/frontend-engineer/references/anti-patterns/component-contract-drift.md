# Anti-Pattern: Component Contract Drift

Use this negative case when a component, composable, or route view has unclear ownership for props, emits, local draft state, remote mutations, or API-to-UI data shaping.

## Signals

- Props are copied into local state without an explicit draft or sync policy.
- Parent and child both mutate the same conceptual value.
- `v-model` emits, local form state, validation state, and remote save payloads do not have one clear owner.
- A child component performs page-level routing, fetching, retry policy, or cross-section coordination.
- Extracted components reduce line count but make state ownership harder to understand.

## Analysis

1. Identify the source of truth for each value.
2. Identify whether local state is a draft, derived state, cached remote data, or owned mutable state.
3. Map props, emits, `v-model`, store writes, and API payload shaping.
4. Check whether extraction changed page-level ownership, routing synchronization, or error policy.

## Recovery Direction

- Keep one source of truth per workflow.
- Name local draft state explicitly and sync it deliberately.
- Keep route/query synchronization, page-level data loading, cross-section coordination, and top-level business actions in the route owner unless the project has a stronger pattern.
- Extract components around stable contracts, not only around visual sections or line count.
