# Question Log Table Format

Use this format for `grillme` companion question logs unless the user asks for another shape.

```md
# {Subject} 提问记录

本文件记录 `{subject}` 设计压测过程中用户提出的澄清问题、术语问题、风险追问、例子请求和后续回看线索；它不是决策日志。

## 问题索引

| 主题 | 覆盖问题 | 关键结论 |
| --- | --- | --- |
| {shared topic} | #{n}-#{m} | {replayable conclusion that groups related angles; update this whenever a new question changes the reusable understanding} |

| # | User Question | Context | Answer Details | Related Decision | Follow-up |
| --- | --- | --- | --- | --- | --- |
| 1 | {the user's exact question} | {what design question or topic triggered it} | {self-contained explanation with the important definitions, examples, implications, tradeoffs, and final rule; concise is fine, but it must be enough to understand later without the chat transcript} | {decision row/title, or "None"} | {remaining dependency, or "None"} |
```

Rules:

- Keep one user question per row.
- Append the row immediately after answering the user's question.
- Preserve the user's exact question text in `User Question` when possible.
- Use `Context` to make the question useful during later review without reading the whole chat.
- Use `Answer Details` for a replayable explanation, not a terse summary. Include concrete examples and the practical consequence when they were part of the answer.
- If the answer contains code, config, TTLs, state transitions, examples, or a final rule of thumb, record those details in `Answer Details`.
- If a table cell becomes long, use `<br>` line breaks inside the cell rather than dropping details.
- Maintain `## 问题索引` above the table. Group related questions under one topic when they are different angles on the same concept, and update the topic's `覆盖问题` and `关键结论` as new angles are answered.
- Use `Related Decision` only when the question is tied to a decision-log row or title; otherwise use `None`.
- Do not duplicate resolved design decisions here; those belong in `decision-log.md`.
- Preserve the existing row order; do not renumber historical rows unless explicitly asked.
- Keep cells focused, but do not shorten them so much that the explanation cannot be reconstructed later.
- Escape literal `|` characters inside cells as `\|`.
- Use `None` for `Follow-up` when there is no remaining dependency.
