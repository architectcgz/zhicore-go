# code-workflow package

This package owns the repo-local assets for the shared `code-workflow`.

Install into a repository:

```bash
bash ~/.agents/harness/workflow-installer.sh <repo-root> code-workflow
```

Check whether a repository still matches this package baseline:

```bash
bash ~/.agents/harness/workflow-sync-check.sh <repo-root> code-workflow
```

The package entrypoint is `workflow.sh`. Callers should prefer the harness-level commands above instead of invoking the package entrypoint directly.

Task-intake order for non-trivial task slices:

1. Run the relevant `superpowers` analysis pass, normally `superpowers:brainstorming`.
2. Then run `grill-with-docs` to look for gaps in scope, docs, assumptions, and owner boundaries.
3. Use that output to finish the implementation plan.
4. Only then start implementation.
