---
name: project-template
description: Use when creating, selecting, or evolving reusable project code templates and project-level AGENTS.md scaffolds, especially frontend, backend, or stack-specific starter structures that should live in one shared template catalog.
---

# Project Template Library

Use this skill as the shared owner for reusable project code templates.

It may provide:

- project-level `AGENTS.md` starter content
- frontend starter structures
- backend starter structures
- combined stack templates

It does not own harness initialization, workflow installation, or documentation scaffolding.

## Workflow

1. For new-project or greenfield initialization requests, first use `brainstorming` before touching templates or structure.
   - Ask what the user is trying to build, who it is for, what constraints matter, and what success looks like.
   - Analyze and recommend the technology stack instead of waiting for the user to fully specify it.
   - Establish an initial architecture before scaffolding: module boundaries, dependency direction, main runtime surfaces, core data/contracts, and the minimum directory shape needed for the first real features.
   - Prefer an architecture-first start over “write features first and split later”; only defer details that are not needed for the first coherent delivery slice.
   - Treat this as mandatory discovery, not optional polish.
2. After the initial direction and recommended stack are clear, use `grill-with-docs` to challenge the plan, terminology, boundaries, and likely documentation shape before scaffolding.
   - If the repo is truly greenfield and docs do not exist yet, grill the proposed structure and naming against the intended domain language and future documentation needs.
   - Stress-test the initial architecture as well: what should be a module now, what can stay simple for phase one, and which boundaries must not be crossed even in the first implementation.
   - Resolve major ambiguities here instead of baking them into the initial scaffold.
3. Inspect the project before writing rules. Read the repository structure and the most relevant existing files, such as `README.md`, package manifests, build configs, test configs, framework configs, verification scripts, and any existing agent instructions.
4. Decide which project code template family applies:
   - frontend-only
   - backend-only
   - combined stack
   - minimal AGENTS-only scaffold
5. Use the relevant architecture skills to constrain template design:
   - frontend starter structures should follow the relevant frontend architecture skill
   - backend starter structures should follow the relevant backend architecture skill
6. Generate or update a project-level `AGENTS.md` in English unless the user explicitly requests another language.
7. Require the project root to keep `CLAUDE.md -> AGENTS.md` when Claude/Codex auto-discovery is part of the local workflow. Do not maintain two divergent entry files.
8. Start from `assets/project-agents-template.md`, then remove irrelevant placeholders and specialize the rules to the actual project.
9. Keep `AGENTS.md` as a repository navigation and routing file. Do not copy global preferences, skill indexes, full documentation taxonomies, or broad personal workflow rules into it.
10. Preserve minimal-diff behavior. If an `AGENTS.md` already exists, patch it instead of replacing it wholesale unless the user explicitly asks for a rewrite.
11. Add a project-specific testing prompt when the repository has tests or an obvious test surface. The prompt should tell future agents which test layers exist, to write or update the narrowest relevant tests first, and to run the smallest relevant test command after changing tests.
12. When the repository provides verification scripts such as `scripts/check-*.sh`, `scripts/check-*.py`, or another documented guard command, make the testing prompt require a follow-up script check after the test run. Prefer existing project scripts over generic advice.
13. Do not stop at prompt text alone when the repository already has mechanical enforcement entry points. If `scripts/check-consistency.sh`, git hooks, CI checks, or another repository guardrail exists, require the relevant test-related script check to be wired into at least one enforced path. A prompt-only rule is not sufficient once enforcement infrastructure exists.
14. If the repository has tests but no mechanical enforcement entry point yet, say so explicitly in `AGENTS.md` or the initialization report; do not pretend the test workflow is enforced already.
15. Validate the final file by checking that it is readable, internally consistent, and does not reference missing tools, scripts, or skills.
16. If documentation scaffolding is needed, route to `documentation-architecture`; do not create documentation templates from this skill.
17. If a harness should be initialized, route to `harness-engineering` or `bash ~/.agents/harness/init-project.sh ...`; do not absorb harness ownership into this skill.
18. If a local entrypoint guard exists, require the initialization workflow to run it after creating the root files, for example `bash ~/workspace/projects/scripts/check-agent-entrypoints.sh <project-root>`.
## Template Use

Use the bundled template at:

```text
assets/project-agents-template.md
```

Current built-in asset:

- `assets/project-agents-template.md`
- `assets/backend/go-backend-onion-template/`
- `assets/frontend/vue-feature-sliced-template/`

Mechanical helper:

- `scripts/apply_project_template.py`
  - `--list` 列出模板
  - `--template ... --dest ... --var KEY=VALUE` 渲染 starter asset
- `bash ~/.agents/harness/project-template-init.sh`
  - 提供模板短名入口，如 `backend-go`、`frontend-vue`
  - 用更稳定的业务参数名包装模板变量，适合 agent / workflow / shell 复用

As this library grows, frontend/backend/stack starter assets should also live under `assets/`.

When creating a new project-level `AGENTS.md`, include only sections that are useful for the repository:

- Keep: project overview, setup commands, project-specific verification commands, architecture boundaries, documentation structure, and repository-specific delivery constraints.
- Customize: tech stack, package manager, test commands, build commands, lint/typecheck commands, service startup commands, database or migration rules, frontend/backend boundaries, generated-file rules, compatibility constraints, the testing prompt that tells agents what to run after writing tests, and the mechanical enforcement path that checks those commands.
- Ensure: the initial scaffold reflects the chosen architecture instead of leaving the first implementation to invent structure ad hoc.
- Remove: repeated global policy, generic communication rules, generic verification advice, generic git discipline, unused framework sections, placeholder examples, nonexistent scripts, nonexistent tools, and skill routing indexes.

## Domain Boundaries

- `project-template`: owns reusable project code template assets and project-level `AGENTS.md` starter content.
- `documentation-architecture`: owns documentation scaffold, documentation rules, docs indexes, and docs templates.
- `harness-engineering`: owns `.harness/`, `harness/`, `feedback/`, harness checks, and harness hooks.
- `workflow-package-manager`: owns workflow package installation and sync checking.
- `~/.agents/harness/init-project.sh`: mechanical harness bootstrap wrapper; useful when a repo should adopt harness + workflow, but not owned by this skill.
- Root-level `CLAUDE.md` symlink creation and validation belong to the initialization workflow; do not leave Claude-specific entrypoints as a manual follow-up.
- Domain architecture skills define the architectural constraints that frontend/backend starter templates should follow.
- `brainstorming`: mandatory first pass for new-project intent, scope, and stack selection.
- `grill-with-docs`: mandatory second pass for stress-testing the chosen direction before scaffolding.

Do not use `project-template` to justify postponing architecture until after features land. The default stance is architecture first, then scaffold, then implementation, with later adjustments done as bounded corrections rather than rescue refactors.

If you intentionally choose a centralized template library, keep the template assets here and keep harness/workflow installation decisions outside this skill.

## Required Checks

Before final response:

1. Confirm the file is named `AGENTS.md`.
2. Confirm the content is English.
3. Confirm section references and numbering are consistent.
4. Confirm `AGENTS.md` routes documentation work to `docs/documentation-rules.md` and `docs/README.md` when those files exist.
5. Confirm documentation scaffolding, if requested, was handled by `documentation-architecture` or an existing stronger project convention.
6. Confirm commands in the file either exist in the project or are clearly marked as project-specific placeholders only when the user requested a template rather than a concrete file.
7. Confirm the testing prompt names the narrowest relevant test command and the follow-up script check when such a script exists in the repository.
8. Confirm the test-related script check is wired into an actual enforcement path such as `scripts/check-consistency.sh`, git hooks, or CI when the repository already has such guardrails.
9. If no enforcement path exists yet, state that explicitly instead of implying the prompt text alone is enough.
10. State whether documentation impact exists. For creating or editing `AGENTS.md`, the file itself is the documentation update.
11. Confirm the project root keeps `CLAUDE.md -> AGENTS.md`, or explicitly state that the repository does not use Claude/Codex auto-discovery.
