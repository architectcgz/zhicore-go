# Core Rules

Read this file whenever `code-workflow` is active.

## Core model

Use this layered timing model:

1. Installation / initialization
2. Task-intake analysis gate: relevant `superpowers` analysis pass first, then `grill-with-docs`
3. Coding-start gate
4. Pre-commit lightweight checks
5. Completion validation
6. Independent review gate
7. Review / doctor / CI governance audit

Do not collapse these into one script.

## Core rules

1. Distinguish `琐碎任务` and `非琐碎任务`.
2. `非琐碎任务` must not start directly in implementation.
3. Each non-trivial task slice must bind one isolated workspace context, one `task-slug`, one implementation plan, and one local startup gate record.
4. Implementation plan content is written in Chinese by default; keep code, commands, paths, error messages, protocol fields, enum values, external proper nouns, and machine-parsed keys unchanged.
5. The isolated workspace is normally a dedicated worktree. If the repository main worktree is clean, no other task is active there, and no parallel isolation is needed, the main worktree may serve as that isolated workspace.
6. Agent orchestration and mechanical enforcement must stay separate: agents decide and guide; scripts, hooks, and review checks enforce.
7. Reuse and owner reasoning should default into the implementation plan, not a mandatory standalone reuse document for every task.
8. Standalone reuse documents are only supplemental evidence for especially large, cross-module, or high-risk tasks.
9. When a non-trivial task arrives, do not jump straight to plan writing or implementation. First run the relevant `superpowers` analysis pass, then run `grill-with-docs` to challenge gaps, assumptions, owner boundaries, and missing facts. Only after that should the implementation plan be finalized and implementation begin.
10. Default analysis pass is usually `superpowers:brainstorming`; debugging or failure tasks usually use `superpowers:systematic-debugging`; if another `superpowers` analysis skill is a better fit, use that instead.
11. For non-trivial work, `completion-full` is still implementation-context self-check, not the final review gate.
12. The final review gate for non-trivial work must run in a separate agent or equivalently independent context.
13. Once work is classified as non-trivial under this workflow, treat entry into `code-workflow` itself as the user's explicit delegation authorization for the minimum necessary independent reviewer subagent unless the user explicitly forbids delegation.
14. If tool policy or an explicit user restriction still prevents spawning that reviewer, state clearly that the independent review gate remains unmet.

## Project adaptation boundary

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

## Interaction with other skills

- Use `development-pipeline` for multi-stage execution of a real engineering task.
- Use `superpowers:writing-plans` for the implementation plan itself; write the plan in Chinese by default unless the user explicitly requests another language.
- Use the relevant `superpowers` analysis skill at task intake, usually `superpowers:brainstorming`.
- Use `superpowers:systematic-debugging` instead when the task starts from a bug, failure, or unexpected behavior.
- Use `grill-with-docs` immediately after the `superpowers` analysis pass, before the plan is considered ready.
- Use `workflow-package-manager` when a repository needs the `code-workflow` package installed or checked.
- Use `harness-engineering` when a repository needs the scaffold installed or repaired.
- Use `reviewer` for the independent completion gate after `completion-full`.

## Required behavior

When this skill applies:

1. Check whether the repository already has a project-local startup workflow.
2. If it does, use it rather than inventing another parallel path.
3. If it does not, install the shared workflow package or state clearly why it cannot be installed.
4. Do not leave workflow governance only in prompt text when a shared mechanical scaffold can be added.
5. At task intake for a non-trivial task, run the analysis gate in order: relevant `superpowers` analysis skill, then `grill-with-docs`.
6. Use the analysis gate output to finish the implementation plan before implementation starts.
7. For non-trivial implementation, do not stop at `completion-full`; run the independent review gate before claiming completion.
8. For non-trivial work, prefer a separate `code-reviewer` agent for that gate instead of reusing the implementation context; entering `code-workflow` already supplies the explicit delegation authorization for that reviewer, so do not wait for a second permission prompt unless the user explicitly prohibited delegation.
9. If the shared `code-workflow` package itself was modified in the current task, run `bash ~/.agents/harness/workflow-sync.sh <repo-root> code-workflow` against each target repository before handoff.
10. When the repository uses this workflow, completed plan/task artifacts should be archived through the shared archive script instead of staying in the active plan/task directories indefinitely.
