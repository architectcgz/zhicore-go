---
name: todo
description: Use when the user asks to 创建todo, create todo, 记录待办, 记个待办, 记一下后续要做的事, or otherwise create a todo note using the shared harness/todo path, naming, and archive rules.
---

# Todo

## Overview

Create a Markdown todo note using the shared harness todo rules. Inside a project, the note should land in the project's canonical todo directory. Outside a project, it should fall back to `~/workspace/projects/todo/`.

## Workflow

1. If the user specified a save location, use it first.
   - A user-specified file path overrides the default directories.
   - A user-specified directory overrides the default directories, and the todo file is created inside it.
2. Otherwise identify the current project by the nearest git root from the active working directory.
3. If inside a project, use the canonical project todo directory:
   - Always use `docs/todo/`.
   - Do not fall back to `docs/todos/` or `todo/`.
4. If not inside a project, use `~/workspace/projects/todo/`.
5. Create the directory if it does not exist.
6. Write a new Markdown file named `YYYY-MM-DD-HHMM-slug.md`.
7. Use the standard todo template below so future reminder scripts can detect open items.
8. When all open items are done, archive the file into the matching `archive/` directory instead of leaving it in the live todo directory.

## Standard Template

```md
# Todo Title

- Project: `/abs/path/to/repo`
- Created: `2026-05-24T12:34+08:00`
- Status: `Open`

## Context

Optional short background.

## Open Items

- [ ] First actionable item
- [ ] Second actionable item
```

## Content Rules

- Keep the todo short and actionable.
- Every actionable item must use `- [ ]` while open.
- Completed items must use `- [x]`.
- Do not invent custom open markers like `TODO:`, `pending`, or plain bullets for actionable work if you want reminder compatibility.
- If the user gives multiple next steps, write one checkbox per step.
- If there is only one next step, still write it as a checkbox item.
- If the user did not give enough context to infer a title, ask one short clarifying question before writing.
- Inside a project, do not write outside the project's canonical todo directory or its `archive/` subdirectory.
- Outside a project, do not write outside `~/workspace/projects/todo/` or its `archive/` subdirectory.
- If the user named a save location, honor that location before applying the default directory rule.
- Legacy paths such as `docs/todos/` and `todo/` are invalid and should be cleaned up instead of reused.
- Completed todo files should be archived instead of left in the live todo directory.

## Implementation Helper

Use the shared harness scripts under `~/.agents/harness/todo/`:

- `python3 ~/.agents/harness/todo/create_todo.py ...`
- `python3 ~/.agents/harness/todo/check_todo_governance.py --cwd <path>`
- `python3 ~/.agents/harness/todo/archive_completed_todos.py --cwd <path>`
