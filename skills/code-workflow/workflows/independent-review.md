# Independent Review Workflow

Read this file for the independent completion gate after `completion-full`.

The independent review gate is intentionally not a shell stage owned by the shared package. It is an orchestration step above the shell runner:

- `completion-full` proves implementation-context validation
- a separate `code-reviewer` agent performs the real gate review
- `workflow-governance` remains the post-review harness / docs / repo-governance audit

Read the shared handoff contract at:

```text
~/.agents/harness/workflows/code-workflow/independent-review-protocol.md
```

## Gate steps

For non-trivial work, after `completion-full` passes:

1. Prepare a compact review handoff instead of reusing the whole implementation conversation.
2. Spawn a separate `code-reviewer` agent with that handoff.
3. Have the reviewer use:
   - the `reviewer` skill
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
