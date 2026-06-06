# Project Collaboration Rules

## 1. Scope and Inheritance

- This file defines repository-level instructions for agents working in this project.
- Follow this file after higher-priority system and user instructions.
- Keep the repository root `CLAUDE.md` as a symlink to `AGENTS.md` when Claude/Codex auto-discovery is part of the local workflow.
- Treat the global `AGENTS.md` as the default policy. Use this file only for repository-specific constraints, commands, architecture boundaries, documentation entry points, and routing rules.
- Prefer project-specific evidence over assumptions. Read the relevant source, configuration, and documentation before making implementation decisions.

## 2. Project Overview

- Project type: `[fill in: frontend / backend / full-stack / library / CLI / service / other]`
- Main language(s): `[fill in]`
- Framework(s): `[fill in]`
- Package manager / build tool: `[fill in]`
- Runtime requirements: `[fill in]`
- Primary entry points: `[fill in]`

## 3. Setup and Common Commands

Use the project's existing package manager and scripts. Do not invent commands that are not present in the repository.

```bash
# Install dependencies
[fill in]

# Start development server or local service
[fill in]

# Run tests
[fill in]

# Run lint
[fill in]

# Run typecheck
[fill in]

# Build
[fill in]
```

If a command is unavailable or dependencies are missing, report the exact blocked command and the reason.

## 4. Architecture and Change Boundaries

- Follow the existing module layout, naming conventions, layering, error handling, logging style, and test patterns.
- Describe the repository's actual code boundaries here, such as frontend/backend ownership, service layers, generated-file policy, migration flow, API compatibility rules, or directories that should normally not be edited.
- Do not introduce new abstractions, dependencies, frameworks, or architectural layers unless the current task requires them.
- Keep generated files, vendored code, build artifacts, and lockfiles consistent with the repository's existing policy.
- Do not render implementation notes, TODO markers, design explanations, or internal commentary into user-visible UI or API responses.

## 5. Project-Specific Verification

Use the global verification rules, then list the smallest project-specific command sequence that usually proves a change is correct.

```bash
[fill in project-specific verification sequence]
```

When the repository has tests, include an explicit testing prompt here:

- Write or update the narrowest relevant tests first.
- After writing tests, run the smallest relevant test command that covers the changed surface.
- If the repository has verification scripts such as `scripts/check-*.sh`, `scripts/check-*.py`, or another documented guard command, run the relevant script check after the test command before claiming completion.
- If the repository already has `scripts/check-consistency.sh`, git hooks, CI validation, or another mechanical guardrail, wire the relevant test-related script check into that enforcement path instead of leaving it as prompt text only.
- If no mechanical enforcement path exists yet, say that explicitly and treat harness setup or guardrail wiring as still pending.

Add any repository-specific constraints here, such as required test order, snapshot update policy, browser checks, fixture refresh steps, or commands that are known to be expensive.

## 6. Git and Delivery Notes

Use the global git and worktree rules, then record only repository-specific delivery constraints here.

- Branch or commit naming convention: `[fill in if different from the global default]`
- Generated files or lockfiles that must be included or excluded: `[fill in]`
- Release, migration, or deployment coupling that affects how changes should be grouped: `[fill in]`

## 7. Documentation

- Documentation rules are owned by `docs/documentation-rules.md`; documentation navigation is owned by `docs/README.md`.
- Before creating, moving, deleting, or editing documentation, read `docs/documentation-rules.md` when it exists, then the nearest existing index for the target area.
- When code behavior, APIs, configuration, database shape, setup flow, or user-visible behavior changes, plan the documentation owner and target path before implementation.
- Do not duplicate the full documentation taxonomy in this file. If a new durable documentation path changes agent routing, add only the route here and keep the detailed rule in the documentation owner file.
