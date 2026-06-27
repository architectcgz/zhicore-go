---
name: grillme
description: Use when the user wants to stress-test a plan or design, asks to be grilled, or needs each decision branch examined until ambiguities are resolved.
---

Interview the user relentlessly about every aspect of this plan until we reach a shared understanding. Walk down each branch of the design tree, resolving dependencies between decisions one-by-one. For each question, provide your recommended answer.

Before asking any plan or design question, ask the user where the decision log should live. Provide a recommended path based on the current workspace, such as `docs/decision-log.md` when a docs directory exists, or `decision-log.md` at the project root otherwise. Wait for the user to answer.

Create the decision log before the first plan or design question. If the user gives a directory, create `decision-log.md` inside it. If the file already exists, append to it instead of replacing it.

Ask the questions one at a time. Wait for feedback on each question before continuing.

Whenever a question is resolved, immediately append the decision to the decision log before asking the next question. Do not batch decisions for later. Use the table format in `references/decision-log-table.md` unless the user asks for another shape.

If a question can be answered by exploring the codebase, explore the codebase instead.
