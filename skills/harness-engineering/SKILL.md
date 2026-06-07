---
name: harness-engineering
description: "Use when initializing or refactoring a repository into an AI-agent harness: AGENTS.md navigation, repo-as-source-of-truth docs, feedback/improvement loops, mechanical consistency checks, git hook or CI guardrails, and project-specific harness onboarding."
metadata:
  short-description: Initialize repository harnesses
---

# Harness Engineering

Use this skill to turn a repository into a navigable, enforceable harness for coding agents.

Default to the current CTF harness shape in `/home/azhi/workspace/projects/ctf` as the local exploratory standard: root `AGENTS.md` as the entry map, `.harness/` for current-task scratch/state and local private reuse indexes, project `harness/` for reusable policies/templates/prompts/checks, `feedback/` for workflow lessons, and mechanical checks/hooks for drift control.

Keep `deusyu/harness-engineering` as an important upstream reference, especially for repo-as-source-of-truth, progressive navigation, feedback capture, mechanical enforcement, and agent readability. Use strict top-level reference directories only when the user explicitly asks to follow the upstream structure.

For brand-new project initialization, `harness-engineering` owns the harness subsystem itself. It should expose mechanical commands that a higher-level workflow or operator can call, but it does not own reusable frontend/backend code templates.

## Workflow

1. Read the target repo first: `AGENTS.md`, README, docs indexes, existing hooks, CI, scripts, plan/review/improvement folders, and current `git status`.
2. Classify existing harness assets:
   - navigation: root and nested `AGENTS.md`
   - source of truth: architecture, requirements, contracts, plans, reviews
   - feedback loop: improvements, incidents, review findings, prompts
   - enforcement: scripts, hooks, CI, tests, linters
3. Use the current CTF harness shape by default while preserving the upstream `deusyu/harness-engineering` principles.
4. Initialize or repair the harness with `~/.agents/harness/harness-initializer.py`, or for the normal harness bootstrap path use `bash ~/.agents/harness/init-project.sh "$PWD"`.
5. Ensure the repository root keeps `CLAUDE.md -> AGENTS.md`; create the symlink when missing, but do not overwrite an existing non-symlink file silently.
6. If the local workspace provides `~/workspace/projects/scripts/check-agent-entrypoints.sh`, run it against the target repo after initialization.
7. Ensure the generated scaffold includes `scripts/check-test-workflow.sh` and that `scripts/check-consistency.sh`, hooks, or CI actually invoke it instead of leaving test workflow rules as prompt text only.
8. Ensure the generated scaffold includes a minimal `scripts/check-architecture.sh` guard plus seed policy files, and that `scripts/check-consistency.sh`, hooks, or CI actually invoke it instead of leaving architecture ownership only in prompt text.
9. Ensure the generated scaffold includes `scripts/check-script-guard.sh` plus `harness/policies/script-guard.json`, and that `scripts/check-consistency.sh` actually invokes the script guard so large harness/operator scripts are forced to split before they drift.
10. Run the generated harness check and any affected existing hook/script checks.
11. Report changed files, validation evidence, and any residual gaps.
12. When the repository should adopt the shared non-trivial task workflow, install the common startup package with `bash ~/.agents/harness/workflow-installer.sh "$PWD" code-workflow`, or prefer the higher-level bootstrap wrapper `bash ~/.agents/harness/init-project.sh "$PWD"` during normal initialization.
13. Treat `code-workflow` as the owner of non-trivial task workflow semantics. `harness-engineering` should only install or repair that shared workflow entry, not redefine its rules here.

When the repo uses project todos, initialize a non-blocking reminder flow on the canonical path `docs/todo/`:

- add `scripts/check-open-todos.sh`
- wire root `AGENTS.md` to read it at task start
- surface its output from `scripts/check-consistency.sh`

When the repo has automated tests or an obvious test surface, initialize a mechanical test-workflow guard:

- add `scripts/check-test-workflow.sh`
- have it verify `AGENTS.md` documents the narrowest-relevant-test-first workflow and follow-up script checks
- have `scripts/check-consistency.sh` execute it
- rely on existing pre-commit or CI entry points to enforce it transitively

When the repo has architecture docs or any structural code surface, initialize a minimal architecture guard:

- add `scripts/check-architecture.sh`
- seed `harness/policies/architecture-guard-paths.txt`
- seed `harness/policies/architecture-guard-commands.txt`
- have `scripts/check-consistency.sh` execute it
- treat the command list as the project-local extension point for backend/frontend/module boundary checks

When the repo has harness/operator scripts, initialize a mechanical script-growth guard:

- add `scripts/check-script-guard.sh`
- seed `harness/policies/script-guard.json`
- have `scripts/check-consistency.sh` execute it
- keep the policy focused on harness/operator entrypoints, wrappers, and harness checks instead of unrelated domain build scripts

When the repo uses the local reuse index pattern, wire a non-blocking reminder into root `AGENTS.md`:

- when implementation first forms a stable reuse pattern in a module, remind the operator to add `.harness/reuse-index/<source-path>/README.md`
- when reuse structure stabilizes inside a module, remind the operator to add a deeper mirrored `README.md` for that subpath
- keep this as an operator reminder only; do not make local private indexes a pre-commit blocker

## Initialization Command

From any target repository root:

```bash
bash /home/azhi/.agents/harness/init-project.sh "$PWD"
```

To add the shared non-trivial task workflow after the harness exists:

```bash
bash ~/.agents/harness/workflow-installer.sh "$PWD" code-workflow
```

For strict upstream-reference mode:

```bash
bash /home/azhi/.agents/harness/init-project.sh "$PWD" --mode strict-reference
```

`init-project.sh` is the preferred high-level bootstrap wrapper. It runs `harness-initializer.py`, then installs the requested workflow package by default, then runs the repo-local consistency check when present. The lower-level Python initializer remains the repair/debugging entry for harness-only operations.

The initializer is idempotent. In both modes it also ensures the repo root keeps `CLAUDE.md -> AGENTS.md`, unless an existing conflicting `CLAUDE.md` requires manual resolution. In default CTF-current mode it creates `.harness/`, `.harness/reuse-decisions/`, optional local `.harness/reuse-index/`, `harness/policies/`, `harness/templates/`, `harness/prompts/`, `harness/checks/`, `feedback/`, `scripts/check-architecture.sh`, `scripts/check-test-workflow.sh`, and a consistency check. In strict reference mode it creates top-level `concepts/`, `thinking/`, `practice/`, `feedback/`, `works/`, `prompts/`, `references/`, `scripts/check-architecture.sh`, `scripts/check-test-workflow.sh`, and a consistency check.

## Harness Shape

Keep the harness as a map, not a manual. In the current local standard:

- `AGENTS.md`: repository navigation entry.
- `CLAUDE.md -> AGENTS.md`: Claude/Codex auto-discovery entrypoint alias; keep it as a symlink, not a divergent copy.
- `.harness/`: current-task state and short-lived execution evidence only.
- `harness/policies/`: project-local mechanical policy inputs.
- `harness/templates/`: project-local templates for repeated decisions.
- `harness/prompts/`: stable in-repo prompt entrypoints, local parameters, and prompts that are still truly project-local. Shared prompt bodies can live under `~/.agents/harness/prompts/`. Do not keep one-off initialization prompts, historical migration prompts, or rules already moved into a global skill.
- `harness/checks/`: deterministic guard scripts.
- `.harness/reuse-index/`: user-local, gitignored reuse index. Keep `index.yaml` as the top-level route map and mirrored `README.md` files as module/module-internal secondary indexes.
- `feedback/`: mistakes, corrections, workflow lessons, and reusable learning that has not yet been fully absorbed elsewhere.
- `scripts/check-consistency.sh`: deterministic guard against drift.
- `scripts/check-architecture.sh`: deterministic minimal architecture guard for docs/architecture routing and project-local architecture commands.
- `scripts/check-test-workflow.sh`: deterministic guard that checks whether test workflow instructions are documented and actually wired into enforcement paths.
- `scripts/check-script-guard.sh`: deterministic guard that limits harness/operator script growth and forces oversized scripts to split.
- `scripts/check-open-todos.sh`: non-blocking reminder for unchecked backlog items under `docs/todo/`, plus completed files that still need archiving.
- `scripts/check-skill-sync-reminder.sh`: non-blocking reminder that asks whether project harness changes should stay local or be synchronized into `~/.agents/skills/` or `~/.agents/harness/`.
- Shared non-trivial task workflow package: install and verify `~/.agents/harness/workflows/code-workflow/`, but keep its behavior definition in the `code-workflow` skill instead of duplicating it here.

When strict upstream reference mode is requested, use `concepts/`, `thinking/`, `practice/`, `feedback/`, `works/`, `prompts/`, and `references/` as demonstrated by `deusyu/harness-engineering`.

## Guardrails

- Do not duplicate long architecture content into harness docs; link to the owning source.
- Do not overwrite existing user text outside managed marker blocks.
- When the user says to strictly follow `deusyu/harness-engineering`, create the top-level reference directories even if the repo already has docs elsewhere.
- During the current exploration phase, treat the CTF harness as the preferred local standard, not a frozen universal law; preserve project-specific adaptation when the target repo has stronger existing conventions.
- Treat missing mechanical enforcement as a real harness gap, not just a documentation issue.
- Treat missing or drifted `CLAUDE.md -> AGENTS.md` as a harness gap; fix it during initialization or fail loudly if an existing file conflicts.
- If a repo has dirty worktree changes, avoid touching those files unless the task requires it.
- Feedback records should include a sedimentation status section that names whether the lesson is already absorbed, project-only, awaiting skill sync, mechanized, or obsolete. Once a lesson is fully captured by a global skill, global AGENTS rule, project policy, or mechanical check, remove the long feedback body and keep only an index note or rely on Git history.
- Add or preserve a non-blocking skill-sync reminder when feedback, reuse knowledge, prompts, policies, or templates change. The reminder should force a conscious decision: keep project-only knowledge local, or move cross-project methods and anti-patterns into the relevant global skill.
- Prefer the shared harness implementation at `~/.agents/harness/skill-sync/remind_skill_sync.py`; project repositories should usually keep only a thin wrapper script and local hook wiring.
- Reuse-first policies should cover both frontend and backend creation surfaces. Frontend surfaces usually include pages, components, hooks, stores, API wrappers, forms, tables, modals, layouts, and schemas. Backend surfaces usually include services, handlers, repositories, ports, jobs/workers, mappers, read models, runtime composition, schemas, and migrations.
- Reuse-index reminders should fire during active implementation, especially when a new module, feature slice, service cluster, or module-internal layer is becoming a reusable pattern for the first time.
- New harness/project initialization should include project documentation architecture by reusing `documentation-architecture` assets, normally `docs/documentation-rules.md` and `docs/README.md`. Project `AGENTS.md` should only route to those files, not duplicate the full documentation policy.

## References

Read `references/harness-adaptation.md` only when the user explicitly asks for an adapted, non-strict harness.
