---
name: code-workflow
description: Use when establishing or running the shared non-trivial task workflow across repositories, especially task intake, worktree-per-task, implementation-plan startup gates, commit task binding, and the split between agent orchestration and mechanical enforcement.
---

# Code Workflow

Use this skill when the task is about the engineering workflow itself, or when non-trivial implementation work should enter a disciplined path instead of ad hoc coding.

This skill is global and cross-project. It defines the common workflow model. Project repositories still own their protected surfaces, local scripts, hooks, and review checks.

## Core Model

Use this layered timing model:

1. Installation / initialization
2. Task-intake analysis gate: relevant `superpowers` analysis pass first, then `grill-with-docs`
3. Coding-start gate
4. Pre-commit lightweight checks
5. Completion validation
6. Independent review gate
7. Review / doctor / CI governance audit

Do not collapse these into one script.

## Core Rules

1. Distinguish `琐碎任务` and `非琐碎任务`.
2. `非琐碎任务` must not start directly in implementation.
3. Each non-trivial task slice must bind:
   - one isolated workspace context
   - one `task-slug`
   - one implementation plan
   - one local startup gate record
   Normally this isolated workspace is a dedicated worktree. If the repository main worktree is currently clean, no other task is active there, and no parallel isolation is needed, the main worktree itself may serve as that isolated workspace.
4. Agent orchestration and mechanical enforcement must stay separate.
   - Agents decide and guide.
   - Scripts, hooks, and review checks enforce.
5. Reuse and owner reasoning should default into the implementation plan, not a mandatory standalone reuse document for every task.
6. Standalone reuse documents are only supplemental evidence for especially large, cross-module, or high-risk tasks.
7. When a non-trivial task arrives, do not jump straight to plan writing or implementation.
   - First run the relevant `superpowers` analysis pass.
   - Then run `grill-with-docs` to challenge gaps, assumptions, owner boundaries, and missing facts.
   - Only after that should the implementation plan be finalized and implementation begin.
8. Default `superpowers` analysis pass:
   - usually `superpowers:brainstorming`
   - debugging or failure tasks usually `superpowers:systematic-debugging`
   - if another `superpowers` analysis skill is a better fit, use that instead
9. For non-trivial work, `completion-full` is still implementation-context self-check, not the final review gate.
10. The final review gate for non-trivial work must run in a separate agent or equivalently independent context.
11. If the user explicitly asks to use `code-workflow` together with an independent reviewer / separate agent / 独立 review agent, that authorizes the minimum reviewer subagent required for the gate.
12. If tool policy or user instruction does not authorize spawning that reviewer, state clearly that the independent review gate remains unmet.

## Shared Entry

The preferred global entry is:

```bash
bash ~/workspace/projects/scripts/start-workflow.sh <topic-or-slug>
```

Behavior:

- If the current repository already has `scripts/start-implementation.sh`, delegate to it.
- If not, initialize the shared workflow scaffold first, then re-run the command.

## Shared Scaffold

The shared repo-local assets live under:

```text
~/.agents/harness/workflows/code-workflow/
```

To install the shared non-trivial task workflow into a repository:

```bash
bash ~/.agents/harness/workflow-installer.sh <repo-root> code-workflow
```

To verify that a repository still matches the shared workflow baseline:

```bash
bash ~/.agents/harness/workflow-sync-check.sh <repo-root> code-workflow
```

To resync a repository after the shared package changes:

```bash
bash ~/.agents/harness/workflow-sync.sh <repo-root> code-workflow
```

This scaffold provides the generic common pieces:

- `scripts/check-task-intake.sh`
- `scripts/start-implementation.sh`
- `scripts/check-startup-gate.sh`
- `harness/workflow-plugins/code-workflow/run_workflow_stage.sh`
- `harness/workflow-plugins/code-workflow/archive_task_artifacts.sh`
- `harness/checks/check_startup_gate.py`
- `harness/templates/implementation-plan-skeleton.md`
- `/.harness/session-gates/` ignore rule

The shared stage runner defines the common stage names:

- `pre-commit-quick`
- `completion-full`
- `workflow-governance`

Repositories still own which local plugins are registered under each `<stage>.d/` directory.

The independent review gate is intentionally not a shell stage owned by this package.
It is an orchestration step above the shell runner:

- `completion-full` proves implementation-context validation
- a separate `code-reviewer` agent performs the real gate review
- `workflow-governance` remains the post-review harness / docs / repo-governance audit

Generated managed files carry a `Managed by code-workflow package` version header so shared upgrades and drift checks have a stable mechanical target.

Use `bash harness/workflow-plugins/code-workflow/archive_task_artifacts.sh` when a task slice is complete, its conclusions have already been absorbed into the owning docs or code comments, and the active implementation plan should leave `docs/plan/impl-plan/`.

The archive script should:

- move the completed implementation plan into `docs/plan/archive/impl-plan/<YYYY-MM>/`
- archive matching `docs/tasks/*<task-slug>*.md` files into `docs/tasks/archive/<YYYY-MM>/` when that directory exists
- move the local startup gate from `active` to `ready_to_merge` when the current worktree owns the gate
- keep `archived` reserved as the terminal closed state after final cleanup or a later explicit close action

Project-specific protected-surface checks and repo-specific review audits remain local.

Read the shared handoff contract at:

```text
~/.agents/harness/workflows/code-workflow/independent-review-protocol.md
```

## Project Adaptation Boundary

Keep these global:

- the layered workflow model
- non-trivial task startup shape
- workspace / slug / plan / gate binding
- commit task binding convention
- shared scaffold installer
- the independent-review orchestration contract and handoff shape

Keep these project-local:

- protected file patterns
- frontend/backend reuse heuristics
- architecture-specific checks
- OpenAPI or contract sync audits
- project review categories and docs layout
- which project-local architecture / contract / review commands the reviewer should rerun

## Independent Review Gate

For non-trivial work, after `completion-full` passes:

1. Prepare a compact review handoff instead of reusing the whole implementation conversation.
2. Spawn a separate `code-reviewer` agent with that handoff.
3. Have the reviewer use:
   - the `code-reviewer` skill
   - the target repository's `AGENTS.md`
   - the relevant `docs/architecture/*`, contracts, and project-local review rules
4. Include the implementation plan path, changed files or diff basis, and executed validation evidence.
5. If the repository exposes project-local architecture or workflow review commands, include them as review inputs and rerun the narrowest relevant set when evidence is weak.
6. Treat same-context review as self-check only, never as the independent completion gate.

Recommended reviewer context:

- repo root
- task slug
- implementation plan path
- diff / commit range / files under review
- validation commands and results
- architecture / contract docs to use as the review basis
- known risk areas and expected review focus

Do not treat "I looked over my own changes after coding" as satisfying this gate.

## Interaction With Other Skills

- Use `development-pipeline` for multi-stage execution of a real engineering task.
- Use `superpowers:writing-plans` for the implementation plan itself.
- Use the relevant `superpowers` analysis skill at task intake, usually `superpowers:brainstorming`.
- Use `superpowers:systematic-debugging` instead when the task starts from a bug, failure, or unexpected behavior.
- Use `grill-with-docs` immediately after the `superpowers` analysis pass, before the plan is considered ready.
- Use `workflow-package-manager` when a repository needs the `code-workflow` package installed or checked.
- Use `harness-engineering` when a repository needs the scaffold installed or repaired.
- Use `code-reviewer` for the independent completion gate after `completion-full`.

## Required Behavior

When this skill applies:

1. Check whether the repository already has a project-local startup workflow.
2. If it does, use it rather than inventing another parallel path.
3. If it does not, install the shared workflow package or state clearly why it cannot be installed.
4. Do not leave workflow governance only in prompt text when a shared mechanical scaffold can be added.
5. At task intake for a non-trivial task, run the analysis gate in order:
   - relevant `superpowers` analysis skill
   - `grill-with-docs`
6. Use the analysis gate output to finish the implementation plan before implementation starts.
7. For non-trivial implementation, do not stop at `completion-full`; run the independent review gate before claiming completion.
8. When the user or tool policy permits delegation, prefer a separate `code-reviewer` agent for that gate instead of reusing the implementation context.
9. If the shared `code-workflow` package itself was modified in the current task, run `bash ~/.agents/harness/workflow-sync.sh <repo-root> code-workflow` against each target repository before handoff.
10. When the repository uses this workflow, completed plan/task artifacts should be archived through the shared archive script instead of staying in the active plan/task directories indefinitely.
