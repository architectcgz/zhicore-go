---
name: harness-workflow
description: Use when selecting, installing, upgrading, or checking shared workflow packages under ~/.agents/harness/workflows for a repository.
---

# Harness Workflow

Use this skill when the task is about shared workflow packages that live under `~/.agents/harness/workflows/`.

This skill does not define a workflow's semantics. It owns the harness-level packaging and mechanical operations around workflow packages:

- which workflow package should be used
- how to install it into a repository
- how to check whether a repository drifted from the shared package baseline
- how to keep workflow package entrypoints and references consistent

## Commands

Install a workflow package:

```bash
bash ~/.agents/harness/workflow-installer.sh <repo-root> <workflow-name>
```

Check whether a repository still matches the shared package baseline:

```bash
bash ~/.agents/harness/workflow-sync-check.sh <repo-root> <workflow-name>
```

Current workflow packages live under:

```text
~/.agents/harness/workflows/
```

For example:

```bash
bash ~/.agents/harness/workflow-installer.sh <repo-root> code-workflow
bash ~/.agents/harness/workflow-sync-check.sh <repo-root> code-workflow
```

## Boundaries

- `harness-workflow`: owns package installation, package sync checking, and harness-level workflow package layout.
- `code-workflow`: owns the non-trivial task workflow semantics for `code-workflow`.
- `harness-engineering`: owns broader repository harness setup and may call this skill when a repository should adopt a shared workflow package.
- `project-template`: does not own workflow package rules.

## Required Behavior

When this skill applies:

1. Confirm which workflow package the repository should adopt.
2. Install or check the package through the harness-level commands above, not through ad hoc file copying.
3. Keep repository-facing references pointed at the harness-level commands.
4. If a workflow package needs semantic changes, update its owning workflow skill rather than hiding semantics inside the installer.
