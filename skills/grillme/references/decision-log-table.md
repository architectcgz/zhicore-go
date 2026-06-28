# Decision Log Table Format

Use this format for `grillme` decision logs unless the user asks for another shape.

```md
# {Subject} 决策日志

本文件记录 `{subject}` 设计压测过程中已经确认的设计问题、结论、原因和后续依赖。

| # | 决策项 | Question | Decision | Rationale | Follow-up |
| --- | --- | --- | --- | --- | --- |
| 1 | {Short decision title} | {the question that was resolved} | {the agreed answer} | {why this answer was chosen; include prior decision rows considered and any supersedes/amends/scope-narrowing relationship} | {remaining dependency, unresolved conflict, or "None"} |
```

Rules:

- Keep one decision per row.
- Append the row immediately after the question is resolved.
- Before appending a row, re-read the whole existing decision log and identify active prior decisions that constrain the new row.
- The new row must be consistent with active prior rows. If it is not, pause the normal decision flow and ask a conflict-resolution question before writing.
- Use `Rationale` to record the consistency check when it matters: cite relevant row numbers or titles, and state whether the new decision follows, amends, supersedes, or narrows them by scope.
- If a prior row is amended or superseded, do not edit historical rows unless the user explicitly asks. Append the new row and make the relationship explicit.
- Use `Follow-up` for unresolved dependencies or conflict checks that still need review; use `None` only when no follow-up remains.
- Preserve the existing row order; do not renumber historical rows unless explicitly asked.
- Use concise cells. If a cell needs multiple clauses, use semicolons instead of nested bullets.
- Escape literal `|` characters inside cells as `\|`.
