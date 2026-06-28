---
name: persisting-useful-findings
description: Use when user corrections, clarification questions, repeated mistakes, process gaps, reusable lessons, discovered risks, missing documentation, or improvement signals may need durable project or global recording.
---

# Persisting Useful Findings

## Overview

Preserve useful signals that would otherwise disappear in chat: user corrections, clarification questions, process gaps, repeated mistakes, reusable lessons, risks, and follow-up material.

Core rule: finish the user's immediate task, then persist only the parts that will improve future work.

## Baseline Failures To Avoid

This skill exists because agents often:

- answer a useful user question but do not record it;
- create a question log entry for a user instruction or rewrite request that was not actually a question;
- create a question log entry for a corrective semantic or architecture-boundary statement, such as "notification should not be responsible for sending email";
- write a question log before the answer is complete, causing the log to capture intent instead of the actual response;
- record only a brief summary when the full answer contains reusable detail worth indexing;
- write a project-specific rule into a global place, or bury a global rule in a project subdirectory;
- describe a recording process as dependent on another skill instead of making it standalone;
- save too much transcript noise instead of the reusable signal;
- mention a follow-up in the final response without creating a durable artifact.

## When To Persist

Consider recording when any of these occur:

- The user corrects your assumptions, priority, wording, workflow, or technical approach.
- The user says "record this", "以后这样", "this should be a skill", or equivalent.
- A genuine user question clarifies terminology, domain meaning, scope, risk, review concern, learning target, or interview emphasis.
- You discover a recurring bug pattern, process gap, missing project rule, absent prompt, or weak verification habit.
- The task produces a reusable checklist, SOP, prompt, template, decision rule, or learning note.

Do not persist:

- temporary command output, short-lived paths, or one-off exploration;
- ordinary Q&A with no future value;
- content already fully covered by existing rules;
- secrets, tokens, credentials, or sensitive raw data.

## Where To Persist

Read local instructions first: `AGENTS.md`, project docs, and existing logs.

| Signal | Preferred location |
| --- | --- |
| Genuine question or clarification | topic directory, e.g. `<topic>-提问记录.md` |
| Project workflow or repo convention | project `AGENTS.md`, `docs/`, `feedback/`, or repo-level log |
| Corrective semantic or architecture-boundary rule | architecture docs, ADR/decision log, ownership docs, `AGENTS.md`, or process-lesson log |
| Repeated implementation/review mistake | project `feedback/` or improvement tracker |
| Cross-project agent behavior rule | `~/.agents/memory/shared/`, `~/.agents/docs/`, or shared skill |
| Reusable prompt | `~/.agents/prompts/` or project prompt directory |
| Stable cross-project workflow | `~/.agents/skills/` after proper skill-creation validation |

Keep ownership clear: project-specific knowledge stays project-local; genuinely cross-project rules may go global.

## Record Format

For genuine user questions and clarification questions, answer first, then append one row after the response is complete:

```md
| # | User Question | Context | Answer Summary | Full Content Index | Related Material | Follow-up |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | {user's original question} | {triggering task/topic} | {concise conclusion after answering} | {anchor/path/section for full answer, or None} | {file/doc/rule} | {next action or None} |
```

`Answer Summary` is only an index summary. If the answer contains reusable explanation, examples, decision logic, or study material, preserve or point to the full content using `Full Content Index`.

Choose the full-content location by this priority. The question log should usually stay an index, not become a content archive:

1. Existing durable material with the complete answer, e.g. `docs/auth-review.md#token-refresh-risk`.
2. A generated or updated artifact that should own the content, e.g. `amazon-ops-trainee/亚马逊运营管培生-知识模块讲解.md#模块-3-listing-基础`.
3. A single sibling full-content directory next to the question log when no existing owner fits. Default to `question-log-full/` in that directory, even if the question log is named `<topic>-提问记录.md`; write files like `question-log-full/q-003-short-topic.md`.
4. Same-file appendix anchor only when the answer is short, self-contained, and no better owner exists, e.g. `#q-003-full-answer`.
5. `None` only when the concise summary is truly sufficient.

If no durable document contains the full answer and the content is short enough to live with the question log, add a short "Full Answer" appendix below the table:

```md
## q-003-full-answer

{complete answer or the reusable parts of the answer}
```

For process lessons, use:

```md
## YYYY-MM-DD Topic

- Trigger: what happened and how it was noticed.
- Lesson: future behavior required.
- Scope: where it applies.
- Updated: files changed.
- Follow-up: prompt / skill / script / None.
```

Do not put non-question instructions, rewrite requests, corrective semantic statements, architecture-boundary corrections, or ordinary task directives into a question log. If they expose a durable process or design lesson, update the relevant rule, architecture document, ADR/decision log, ownership note, prompt, skill, or process-lesson file instead.

## Workflow

1. Continue the user's current task; do not let recording become the main work unless requested.
2. Keep a short mental list of possible durable signals.
3. Before writing, check existing local rules and logs.
4. Choose the narrowest correct persistence location.
5. For question logs, wait until the answer is complete; then write the row from the final answer, not from your pre-answer plan.
6. Append or update succinctly; do not paste whole transcripts. Index full reusable content with `Full Content Index`; prefer existing content owners, then the sibling `question-log-full/` directory, and use same-file appendices only for short self-contained answers.
7. If the finding changes a guide, prompt, skill, resume, test plan, or checklist, update that artifact too.
8. In the final response, state where the finding was recorded and any remaining follow-up.

## Quality Bar

A good durable record lets a future agent understand the signal without reading the chat. It is specific enough to change behavior, short enough to stay useful, and independent enough that it does not require a separate skill to interpret.

## Red Flags

Stop and adjust if you are about to:

- record everything "just in case";
- save project-specific instructions globally;
- save global lessons only inside one project;
- reference an external skill as a dependency for understanding a local rule;
- treat every user instruction as a `User Question`;
- treat a corrective design statement as a `User Question`;
- write a question log before you have produced the answer;
- leave `Full Content Index` as `None` when the complete answer contains reusable substance;
- turn the question log into a dumping ground for full answers that belong in a guide, note, or generated artifact;
- claim you recorded a lesson without writing a file;
- skip recording after the user explicitly said the question or correction is useful.
