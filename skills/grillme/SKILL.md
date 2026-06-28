---
name: grillme
description: Use when the user wants to stress-test a plan or design, asks to be grilled, or needs each decision branch examined until ambiguities are resolved.
---

Interview the user relentlessly about every aspect of this plan until we reach a shared understanding. Walk down each branch of the design tree, resolving dependencies between decisions one-by-one. For each question, provide your recommended answer.

Before asking any plan or design question, ask the user where the decision log should live. Provide a recommended path based on the current workspace, such as `docs/decision-log.md` when a docs directory exists, or `decision-log.md` at the project root otherwise. Also state that a companion question log will be created next to it as `question-log.md` for user clarifications and review notes. Wait for the user to answer.

Create both logs before the first plan or design question. If the user gives a directory, create `decision-log.md` and `question-log.md` inside it. If the user gives a file path for the decision log, create the companion `question-log.md` in the same directory. If either file already exists, append to it instead of replacing it.

Ask the questions one at a time. Wait for feedback on each question before continuing.

Whenever a question is resolved, immediately append the decision to the decision log before asking the next question. Do not batch decisions for later. Use the table format in `references/decision-log-table.md` unless the user asks for another shape.

If the user asks a question while answering, treat it as valuable process context when it clarifies terminology, rationale, risk, examples, implications, scope, or later review concerns. Answer the question, then immediately append the user's exact question and a concise answer summary to the companion question log. Do not put these clarification questions in the decision log unless they also resolve or change a design decision. Use the table format in `references/question-log-table.md` unless the user asks for another shape.

If a question can be answered by exploring the codebase, explore the codebase instead.
