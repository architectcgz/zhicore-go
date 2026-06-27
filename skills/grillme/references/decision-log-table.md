# Decision Log Table Format

Use this format for `grillme` decision logs unless the user asks for another shape.

```md
# {Subject} 决策日志

本文件记录 `{subject}` 设计压测过程中已经确认的设计问题、结论、原因和后续依赖。

| # | 决策项 | Question | Decision | Rationale | Follow-up |
| --- | --- | --- | --- | --- | --- |
| 1 | {Short decision title} | {the question that was resolved} | {the agreed answer} | {why this answer was chosen} | {remaining dependency, or "None"} |
```

Rules:

- Keep one decision per row.
- Append the row immediately after the question is resolved.
- Preserve the existing row order; do not renumber historical rows unless explicitly asked.
- Use concise cells. If a cell needs multiple clauses, use semicolons instead of nested bullets.
- Escape literal `|` characters inside cells as `\|`.
- Use `None` for `Follow-up` when there is no remaining dependency.
