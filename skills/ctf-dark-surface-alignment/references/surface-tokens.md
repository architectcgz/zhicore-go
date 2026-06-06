# Surface Tokens

Read this file when the issue is mainly surface color, shell contrast, card tone, or dense-text legibility.

## Core rules

- Replace hardcoded light colors such as `#fff`, `#f8fafc`, `rgba(255,255,255,...)`, and `rgba(241,245,249,...)`.
- Prefer tokenized surfaces:
  - `--journal-ink: var(--color-text-primary)`
  - `--journal-muted: var(--color-text-secondary)`
  - `--journal-border: color-mix(in srgb, var(--color-border-default) 82%-84%, transparent)`
  - `--journal-surface: color-mix(in srgb, var(--color-bg-surface) 88%-92%, var(--color-bg-base))`
  - `--journal-surface-subtle: color-mix(in srgb, var(--color-bg-surface) 74%-78%, var(--color-bg-base))`
- Dark theme should stay lower contrast than raw black plus pure white.
- Dense table and list copy should usually be slightly softer than `var(--color-text-primary)`.

## Common surface patterns

- Hero or page shell:
  - bordered surface
  - subtle radial highlight
  - soft shadow
- Cards and rows:
  - use `var(--journal-surface)` or a shallow gradient between surface layers
  - keep borders real, not pure white
- Buttons:
  - ghost buttons use the journal border and surface
  - primary buttons use accent background plus soft accent shadow
- Dense text:
  - headline or key number uses `var(--journal-ink)` or a close mix
  - supporting copy uses `var(--journal-muted)` or a softened mix
