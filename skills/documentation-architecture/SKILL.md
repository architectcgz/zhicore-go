---
name: documentation-architecture
description: Use when initializing, repairing, or standardizing a project's docs/ architecture, documentation ownership rules, docs indexes, new-path registration rules, or documentation scaffolding for a repository.
---

# Documentation Architecture

Use this skill for documentation structure. Do not put documentation templates in `project-template`; project initialization should orchestrate this skill instead.

## Workflow

1. Inspect the repository first: existing `docs/`, README, AGENTS, scripts, CI, and any stronger project-specific documentation convention.
2. If the project already has a stronger convention, preserve it and only add missing ownership, indexing, and validation rules.
3. Initialize or repair the standard docs scaffold from `assets/docs/`.
4. Avoid documentation circular references:
   - `docs/documentation-rules.md` is the rule source.
   - `docs/README.md` is the navigation index.
   - `AGENTS.md` may route agents to both, but must not duplicate the full rules.
   - Do not make two documents require each other before either can be edited.
5. Register any durable new documentation path in `docs/documentation-rules.md`, `docs/README.md` or nearest parent index, project `AGENTS.md` when routing changes, and mechanical checks when stability matters.
6. Run the smallest available documentation or harness consistency check, then search for stale references when paths changed.

## Standard Scaffold

Use this structure unless the repository already has a stronger convention:

```text
docs/
├── documentation-rules.md
├── README.md
├── requirements/
├── contracts/
├── spec/
├── design/
├── todo/
├── architecture/
├── plan/
├── operations/
├── reviews/
├── reports/
├── improvements/
└── refs/
```

## Assets

- `assets/docs/documentation-rules.md`: documentation ownership and path registration rules.
- `assets/docs/README.md`: documentation index template.
- `assets/docs/improvements/README.md`: improvement tracker folder guide.

Keep assets here. `project-template` should only coordinate when to apply them.
