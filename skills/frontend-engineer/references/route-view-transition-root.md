# Route View Transition Root

## When to Read

Read this before changing a Vue route-view template when the parent layout renders routes through:

```vue
<RouterView v-slot="{ Component, route: resolvedRoute }">
  <Transition mode="out-in">
    <component
      :is="Component"
      :key="resolvedRoute.path"
      class="workspace-route-root"
    />
  </Transition>
</RouterView>
```

This also applies when the parent route renderer passes layout classes, attrs, or transition hooks to the route component.

## Rule

A route view rendered inside `Transition` must expose one concrete DOM root that can receive inherited attrs and classes.

Do not leave route views as fragments such as:

```vue
<template>
  <ClassManagementPage />
  <TeacherClassReportExportDialog />
</template>
```

Use a semantic route root or the established route shell:

```vue
<template>
  <section class="teacher-route-root">
    <ClassManagementPage />
    <TeacherClassReportExportDialog />
  </section>
</template>
```

Prefer `section`, `main`, or a shared route-shell component over an empty wrapper. The root exists because it is the route view's animation and layout target, not as decorative nesting.

## Why It Matters

Vue 3 supports fragment components in general, but a route component inside `Transition` has stricter runtime behavior. If the route view has multiple roots, Vue cannot reliably choose a single element for:

- transition enter/leave hooks
- `mode="out-in"` sequencing
- inherited `class` / attrs from `<component :is="Component" class="...">`
- route-level layout sizing and page-root selectors

Common warnings:

```text
Extraneous non-props attributes (class) were passed to component but could not be automatically inherited
Component inside <Transition> renders non-element root node that cannot be animated.
```

Symptoms may look unrelated to layout:

- sidebar navigation changes URL but the target page does not mount reliably
- `onMounted()` requests on the target route do not run
- global route animation breaks or skips
- parent layout classes such as `workspace-route-root` are missing

## Fix Pattern

1. Inspect the parent layout's `RouterView` and `Transition` contract.
2. Identify any route view that renders `PageBody + Dialog`, `PageBody + Drawer`, or sibling Teleport owners at the top level.
3. Wrap those siblings in one semantic route root.
4. Keep page-owned state, request loading, retry policy, and dialogs in the route view unless extracting them clarifies ownership.
5. Add or update a route-layout test that proves navigation from the affected route mounts the next route and triggers its expected load.

## CTF Reference Case

In the CTF frontend, `AppLayout.vue` wraps route views in a shared `Transition mode="out-in"` and passes `workspace-route-root` classes to the dynamic route component. Teacher route pages that rendered a main page component beside `TeacherClassReportExportDialog` as sibling roots broke this contract. Admin pages worked because their route views already had a single `workspace-shell` root.

The fix was to keep the teacher page and dialog under one semantic route root, then verify the full `AppLayout + Sidebar + RouterView` path rather than only the sidebar click handler.
