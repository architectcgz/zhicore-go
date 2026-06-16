# Reviews

Store post-delivery review evidence here.

Recommended use:

- Keep raw model-by-model review output in subdirectories when needed.
- Keep synthesized review conclusions in stable Markdown files.
- Link any unresolved technical debt to `../todos/debt/`.

Suggested naming:

- `YYYY-MM-DD-<scope>-review-round<N>.md`
- `YYYY-MM-DD-<scope>-cross-model-summary.md`

Keep review evidence separate from debt tracking:

- `docs/reviews/` answers "what was reviewed and what did the reviewers say".
- `docs/todos/debt/` answers "which debt is still open and needs follow-up".
