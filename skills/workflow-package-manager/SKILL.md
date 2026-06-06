---
name: workflow-package-manager
description: Use when selecting, installing, upgrading, or checking shared workflow packages under ~/.agents/harness/workflows for a repository.
---

# Workflow Package Manager

Use this skill when the task is about shared workflow packages that live under `~/.agents/harness/workflows/`.

This skill does not define a workflow's semantics. It owns the harness-level packaging and mechanical operations around workflow packages:

- which workflow package should be used
- how to install it into a repository
- how to resync a repository after the shared workflow package changes
- how to check whether a repository drifted from the shared package baseline
- how to keep workflow package entrypoints and references consistent

For normal greenfield project bootstrap, a higher-level harness wrapper may call these commands for you. That does not change ownership: this skill still owns workflow package installation and sync checking, while template selection and code starter ownership live elsewhere.

## Commands

Install a workflow package:

```bash
bash ~/.agents/harness/workflow-installer.sh <repo-root> <workflow-name>
```

Sync a repository to the latest shared workflow package baseline:

```bash
bash ~/.agents/harness/workflow-sync.sh <repo-root> <workflow-name>
```

Check whether a repository still matches the shared workflow package baseline:

```bash
bash ~/.agents/harness/workflow-sync-check.sh <repo-root> <workflow-name>
```

Current workflow packages live under:

```text
~/.agents/harness/workflows/
```

For example:

```bash
bash ~/.agents/harness/init-project.sh <repo-root>
bash ~/.agents/harness/workflow-installer.sh <repo-root> code-workflow
bash ~/.agents/harness/workflow-sync.sh <repo-root> code-workflow
bash ~/.agents/harness/workflow-sync-check.sh <repo-root> code-workflow
```

## Boundaries

- `workflow-package-manager`: owns package installation, package sync checking, and harness-level workflow package layout.
- `code-workflow`: owns the non-trivial task workflow semantics for `code-workflow`.
- `harness-engineering`: owns broader repository harness setup and may call this skill when a repository should adopt a shared workflow package.
- `project-template`: may own reusable code templates, but does not own workflow package rules.
- `~/.agents/harness/init-project.sh`: convenience bootstrap wrapper that may call `workflow-installer.sh`, but does not replace this skill's ownership boundary.

## Required Behavior

When this skill applies:

1. Confirm which workflow package the repository should adopt.
2. Install, sync, or check the package through the harness-level commands above, not through ad hoc file copying.
3. When the shared workflow package itself changes, finish by running `bash ~/.agents/harness/workflow-sync.sh <repo-root> <workflow-name>` for each target repository touched by the current task; do not assume automatic propagation.
4. Keep repository-facing references pointed at the harness-level commands.
5. If a workflow package needs semantic changes, update its owning workflow skill rather than hiding semantics inside the installer.
