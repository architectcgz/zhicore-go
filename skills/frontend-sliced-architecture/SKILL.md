---
name: frontend-sliced-architecture
description: Use when designing, scaffolding, reviewing, or refactoring frontend architecture for large Vue, React, Nuxt, Next, or TypeScript apps with feature boundaries, slices, widgets, entities, shared layers, complex page state, or frontend Clean Architecture concerns.
---

# Frontend Sliced Architecture

## Overview

Use this skill to structure frontend code around user-facing capabilities without turning the UI into backend-style ceremony. The default architecture is Feature-Sliced / Vertical Slice, with local Clean Architecture only for complex features.

## When To Use

Use for:

- New frontend project structure decisions.
- Refactoring route components that have accumulated API calls, state machines, derived data, and workflow logic.
- Designing feature/entity/shared boundaries in Vue, React, Nuxt, Next, or TypeScript apps.
- Reviewing whether UI, application state, domain rules, and API adapters are mixed together.
- Creating migration plans from page-centric or component-only frontend structure.

Do not use for:

- Pure visual polish with no structure change.
- Small components with obvious local state.
- Backend architecture work.

## Default Choice

Prefer Feature-Sliced Design for whole-app structure:

```text
src/
  app/
  pages/
  widgets/
  features/
  entities/
  shared/
```

Use frontend Clean Architecture inside a feature only when the feature has real business complexity.

## Dependency Rule

For sliced architecture:

```text
app -> pages -> widgets -> features -> entities -> shared
```

Lower layers must not import higher layers. Cross-slice imports should go through public APIs such as `index.ts`.

## Workflow

1. Identify user workflows and domain nouns before moving files.
2. Decide whether the app needs full sliced structure or only local feature extraction.
3. Keep route pages as composition surfaces.
4. Move user actions and async workflows into `features`.
5. Move reusable business objects and their local display/state into `entities`.
6. Keep generic UI, base API clients, config, and utilities in `shared`.
7. For complex features, split UI, application workflow, domain rules, and infrastructure adapters locally.
8. For route-view migrations, use the route boundary scan and test pattern in `references/route-view-migration-boundaries.md`.
9. Add import boundary checks or lint rules before large migrations.

## Boundary Checks

Ask these before approving a structure:

- Does a route page own too many API calls, watchers/effects, computed selectors, and user actions?
- Is `shared` holding business behavior that belongs to a feature or entity?
- Can a feature be tested without rendering the full page?
- Are API DTOs mapped before reaching large templates or view components?
- Are domain rules free of Vue refs/reactive/computed or React hooks?
- Are slices importing each other's internals instead of public APIs?

## References

Read `references/feature-sliced-design.md` for whole-app structure, migration, public APIs, and layer responsibilities.

Read `references/frontend-clean-architecture.md` when a single feature needs stricter presentation/application/domain/infrastructure boundaries.

Read `references/route-view-migration-boundaries.md` when route views still own router objects, query tabs, API calls, lifecycle workflows, or when you need migration scan commands and source boundary tests.

## Common Mistakes

- Applying four Clean Architecture layers to every small UI feature.
- Leaving a route component as the real application layer after files are moved.
- Creating `shared` as a second application layer.
- Letting API response shapes become the long-term view model.
- Importing deep internals across slices because the public API was not designed.
- Treating modal visibility, tab state, and minor display toggles as domain logic.
