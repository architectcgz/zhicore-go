# Code Organization

Read this file when the task involves Vue SFC structure, naming, comments, or general code readability.

## `<script setup>` structure

- Keep a stable top-to-bottom order to avoid spaghetti code:
  1. Imports
  2. Types and interfaces
  3. `defineProps` and `defineEmits`
  4. State (`ref`, `reactive`)
  5. Derived state (`computed`)
  6. Side effects (`watch`, `watchEffect`)
  7. Methods and actions
  8. Lifecycle hooks
- Prefer one consistent SFC block order inside the repo.
- Follow the existing project convention instead of mixing file styles within the same codebase.

## Naming conventions

- Keep boolean names explicit with prefixes such as `is`, `has`, `should`, or `can`.
- Name local event handlers with a `handle` prefix.
- Composables should start with `use`.
- Composables should return an object rather than a positional array unless the repo already relies on tuple semantics for a good reason.
- Prefer multi-word component names to avoid collisions with native HTML tags.

## Comment rules

- Comments should explain business rules, edge cases, browser workarounds, or non-obvious tradeoffs.
- Do not write comments that merely restate what the code already says.
- If a workaround looks strange, explain why it exists and what would break if it is removed.
