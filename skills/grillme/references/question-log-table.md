# Question Log Table Format

Use this format for `grillme` companion question logs unless the user asks for another shape.

```md
# {Subject} 提问记录

本文件记录 `{subject}` 设计压测过程中用户提出的澄清问题、术语问题、风险追问、例子请求和后续回看线索；它不是决策日志。

| # | User Question | Context | Answer Summary | Related Decision | Follow-up |
| --- | --- | --- | --- | --- | --- |
| 1 | {the user's exact question} | {what design question or topic triggered it} | {concise answer, not a full transcript} | {decision row/title, or "None"} | {remaining dependency, or "None"} |
```

Rules:

- Keep one user question per row.
- Append the row immediately after answering the user's question.
- Preserve the user's exact question text in `User Question` when possible.
- Use `Context` to make the question useful during later review without reading the whole chat.
- Use `Related Decision` only when the question is tied to a decision-log row or title; otherwise use `None`.
- Do not duplicate resolved design decisions here; those belong in `decision-log.md`.
- Preserve the existing row order; do not renumber historical rows unless explicitly asked.
- Use concise cells. If a cell needs multiple clauses, use semicolons instead of nested bullets.
- Escape literal `|` characters inside cells as `\|`.
- Use `None` for `Follow-up` when there is no remaining dependency.
