---
name: project-template
description: Use when coordinating repository initialization or project-level AGENTS.md setup, including routing to documentation, harness, and project-specific structure initialization without owning those domain templates directly.
---

# Project Template Orchestrator

Use this skill as the project initialization coordinator. It does not own documentation templates, harness templates, frontend templates, or backend templates.

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
4. Decide which initialization domains apply:
   - Documentation architecture: use `documentation-architecture`.
   - Harness scaffold: use `harness-engineering`.
   - Frontend architecture: use the relevant frontend architecture skill.
   - Backend architecture: use the relevant backend architecture skill.
5. Generate or update a project-level `AGENTS.md` in English unless the user explicitly requests another language.
6. Require the project root to keep `CLAUDE.md -> AGENTS.md` when Claude/Codex auto-discovery is part of the local workflow. Do not maintain two divergent entry files.
7. Start from `assets/project-agents-template.md`, then remove irrelevant placeholders and specialize the rules to the actual project.
8. Keep `AGENTS.md` as a repository navigation and routing file. Do not copy global preferences, skill indexes, full documentation taxonomies, or broad personal workflow rules into it.
9. Preserve minimal-diff behavior. If an `AGENTS.md` already exists, patch it instead of replacing it wholesale unless the user explicitly asks for a rewrite.
10. Add a project-specific testing prompt when the repository has tests or an obvious test surface. The prompt should tell future agents which test layers exist, to write or update the narrowest relevant tests first, and to run the smallest relevant test command after changing tests.
11. When the repository provides verification scripts such as `scripts/check-*.sh`, `scripts/check-*.py`, or another documented guard command, make the testing prompt require a follow-up script check after the test run. Prefer existing project scripts over generic advice.
12. Do not stop at prompt text alone when the repository already has mechanical enforcement entry points. If `scripts/check-consistency.sh`, git hooks, CI checks, or another repository guardrail exists, require the relevant test-related script check to be wired into at least one enforced path. A prompt-only rule is not sufficient once enforcement infrastructure exists.
13. If the repository has tests but no mechanical enforcement entry point yet, say so explicitly in `AGENTS.md` or the initialization report and route harness setup to `harness-engineering` instead of pretending the test workflow is enforced already.
14. Validate the final file by checking that it is readable, internally consistent, and does not reference missing tools, scripts, or skills.
15. If documentation scaffolding is needed, call or follow `documentation-architecture`; do not create documentation templates from this skill.
16. If the project uses harness scaffolding, ensure `AGENTS.md` points task start to `scripts/check-open-todos.sh --quiet-if-empty` so open backlog items are surfaced in future Codex sessions.
17. If the project uses the local reuse-index pattern, ensure `AGENTS.md` reminds operators to add `.harness/reuse-index/<source-path>/README.md` when module-level or module-internal reuse patterns first stabilize.
18. If a local entrypoint guard exists, require the initialization workflow to run it after creating the root files, for example `bash ~/workspace/projects/scripts/check-agent-entrypoints.sh <project-root>`.
## Template Use

Use the bundled template at:

```text
assets/project-agents-template.md
```

When creating a new project-level `AGENTS.md`, include only sections that are useful for the repository:

- Keep: project overview, setup commands, project-specific verification commands, architecture boundaries, documentation structure, and repository-specific delivery constraints.
- Customize: tech stack, package manager, test commands, build commands, lint/typecheck commands, service startup commands, database or migration rules, frontend/backend boundaries, generated-file rules, compatibility constraints, the testing prompt that tells agents what to run after writing tests, and the mechanical enforcement path that checks those commands.
- Ensure: the initial scaffold reflects the chosen architecture instead of leaving the first implementation to invent structure ad hoc.
- Remove: repeated global policy, generic communication rules, generic verification advice, generic git discipline, unused framework sections, placeholder examples, nonexistent scripts, nonexistent tools, and skill routing indexes.

## Domain Boundaries

- `project-template`: coordinates project initialization and creates/updates project-level `AGENTS.md`.
- `documentation-architecture`: owns documentation scaffold, documentation rules, docs indexes, and docs templates.
- `harness-engineering`: owns `.harness/`, `harness/`, `feedback/`, harness checks, and harness hooks.
- Root-level `CLAUDE.md` symlink creation and validation belong to the initialization workflow; do not leave Claude-specific entrypoints as a manual follow-up.
- Domain architecture skills own frontend/backend/module-specific structure.
- `brainstorming`: mandatory first pass for new-project intent, scope, and stack selection.
- `grill-with-docs`: mandatory second pass for stress-testing the chosen direction before scaffolding.

Do not use `project-template` to justify postponing architecture until after features land. The default stance is architecture first, then scaffold, then implementation, with later adjustments done as bounded corrections rather than rescue refactors.

Never move a domain template into `project-template` just because project initialization needs it. Add a routing step instead.

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
