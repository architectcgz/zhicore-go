---
name: grillme
description: Use when the user wants to stress-test a plan or design, asks to be grilled, or needs each decision branch examined until ambiguities are resolved.
---

Interview the user relentlessly about every aspect of this plan until we reach a shared understanding. Walk down each branch of the design tree, resolving dependencies between decisions one-by-one. For each question, provide your recommended answer.

Before asking any plan or design question, ask the user where the decision log should live. Provide a recommended path based on the current workspace, such as `docs/decision-log.md` when a docs directory exists, or `decision-log.md` at the project root otherwise. Also state that a companion question log will be created next to it as `question-log.md` for user clarifications and review notes. Wait for the user to answer.

Create both logs before the first plan or design question. If the user gives a directory, create `decision-log.md` and `question-log.md` inside it. If the user gives a file path for the decision log, create the companion `question-log.md` in the same directory. If either file already exists, append to it instead of replacing it.

If the decision log already contains rows, read it before the first plan or design question and summarize the active decisions that will constrain the interview. Use those active decisions to shape later questions and recommended answers. Do not present an option as ordinary if it would require amending or superseding an earlier decision; label it as a conflict path.

Ask the questions one at a time. Wait for feedback on each question before continuing.

Whenever a question is resolved, immediately append the decision to the decision log before asking the next question. Do not batch decisions for later. Use the table format in `references/decision-log-table.md` unless the user asks for another shape.

Before appending any new decision, re-read the existing decision log in full and extract the active prior decisions that constrain the new answer. Compare the candidate decision against those prior decisions before writing. Treat unresolved tension as a blocker: do not append a normal new decision if it contradicts, weakens, duplicates, or silently changes an earlier decision.

If the candidate decision conflicts with an earlier row, ask a conflict-resolution question before continuing. Name the prior row, the new candidate decision, the exact conflict, and your recommended resolution. The resolution must explicitly choose one of: keep the earlier decision and reject the candidate, amend the earlier decision, supersede the earlier decision, or narrow both decisions by scope. Only after the user resolves the conflict should you append a row, and the row must record the relationship to the prior decision in `Rationale` or `Follow-up`.

If the user asks a question while answering, treat it as valuable process context when it clarifies terminology, rationale, risk, examples, implications, scope, or later review concerns. Answer the question, then immediately append the user's exact question and a replayable answer to the companion question log. The log entry must preserve the substance needed to understand the answer later without reading the chat: definitions, concrete examples, implications, tradeoffs, and the resulting rule of thumb when relevant. Do not replace the answer with a teaser like "explained X" or a one-sentence summary when the actual answer contained important detail. Do not put these clarification questions in the decision log unless they also resolve or change a design decision. Use the table format in `references/question-log-table.md` unless the user asks for another shape.

When several user questions are different angles on the same underlying issue, keep one row per exact user question, but group them in the companion question log's index under a shared topic. Update that index whenever a new question changes the topic's reusable conclusion.

If a question can be answered by exploring the codebase, explore the codebase instead.
