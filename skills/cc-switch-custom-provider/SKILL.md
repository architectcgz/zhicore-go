---
name: cc-switch-custom-provider
description: Use when adding, migrating, or standardizing CC-Switch providers across Codex, Claude, or Gemini, especially when provider naming must stay consistent and new providers should use custom instead of codex-for-me or other ad hoc names
---

# CC-Switch Custom Provider

## Overview

CC-Switch provider naming must stay stable. For this environment, new providers should use `custom` as the provider configuration name.

## Rules

- Adding a new CC-Switch provider does not require a database backup by default.
- Before updating, replacing, deleting, migrating, or otherwise changing existing CC-Switch providers, back up `~/.cc-switch/cc-switch.db` with a timestamped copy.
- Prefer `/home/azhi/scripts/import-cc-switch-codex.py` for routine Codex provider imports so the generated config stays consistent.
- For Codex providers, always use:
  - `model_provider = "custom"`
  - `[model_providers.custom]`
  - `name = "custom"`
- Do not create new Codex providers with `codex-for-me` in the config block.
- For Claude and Gemini providers in CC-Switch, set `providers.provider_type` to `custom`.
- Unless the user explicitly asks for a different app-specific structure, keep Claude and Gemini `settings_config` shape unchanged apart from the provider classification that CC-Switch uses.
- When touching bootstrap scripts or current runtime config, keep them aligned with the same `custom` naming.

## Quick Checks

- Read current rows from `~/.cc-switch/cc-switch.db` before editing.
- After editing, verify:
  - target rows in `providers` have `provider_type = "custom"`
  - Codex `settings_config` contains no `model_provider = "codex-for-me"`
  - Codex `settings_config` contains no `[model_providers.codex-for-me]`
- If helper scripts generate Codex config, make sure they also emit `custom`.

## Common Mistakes

- Only changing the database but leaving bootstrap scripts on `codex-for-me`
- Only changing `model_provider` but forgetting the table name under `model_providers`
- Changing Claude or Gemini JSON structure without evidence that CC-Switch needs that change
- Adding a new provider first and trying to normalize naming later
