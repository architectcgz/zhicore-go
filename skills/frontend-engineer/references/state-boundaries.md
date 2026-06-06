# State Boundaries

Read this file when deciding where state lives, how a Vue surface reacts, or how to split a large component.

## State ownership

- Use component-local refs for local UI state such as open or closed, local tab, draft filter, or temporary selection.
- Move state to Pinia only when multiple distant consumers need the same source of truth.
- Do not let both the view and a child component own the same remote state mutation path.
- Prefer one composable to own one remote workflow.

## Vue 3 reactivity

- Do not directly destructure `props` into inert locals.
- Prefer `computed` for derived state.
- Use `watch` or `watchEffect` when bridging to side effects, external systems, or explicitly staged synchronization.

```ts
// bad
const { title } = props

// good
const { title } = toRefs(props)
// or
const title = toRef(props, 'title')
```

- Be careful when spreading reactive objects into plain objects for long-lived reuse.
- When exposing state from a composable, make ownership clear: mutable source vs derived computed state.
- Do not mutate props or use watchers to keep multiple equivalent sources in sync unless that duplication is explicitly part of the design.
- For large read-only structures such as long lists, chart option trees, or bulky history payloads, prefer `shallowRef` or `markRaw()` to avoid deep proxy conversion and unnecessary memory overhead.

## Component boundaries

- Route views should orchestrate, not carry every section inline.
- Extract side-effectful workflows into composables.
- Extract stable visual regions into child components.
- Keep smart and dumb components separate. Route views or container components should fetch data, compose state, and own business rules, while presentational components should stay side-effect free and communicate through `props` and `emit`.
- If data or callbacks are threaded through too many layers, evaluate `provide/inject` or a different boundary.
- In shared views, do not hardcode route names when a route resolver or route metadata layer already exists.
