# Harness Todo

Shared todo governance scripts live here.

- `create_todo.py`
  - Create a todo in the canonical project todo directory.
  - If the current path is not inside a git project, fall back to `~/workspace/projects/todo/`.
- `remind_todos.py`
  - Remind about open items under the canonical live todo directory.
  - Also remind when completed todo files are still waiting to be archived.
- `check_todo_paths.py`
  - Check that todo files stay under the canonical live/archive directories.
  - Project canonical path is always `docs/todo/`.
- `check_todo_filenames.py`
  - Check that todo files use `YYYY-MM-DD-HHMM-slug.md`.
- `check_todo_archive_state.py`
  - Check that completed todos are archived and archived todos have no open items.
- `archive_completed_todos.py`
  - Move completed todos into the archive directory and mark them `Archived`.
- `check_todo_governance.py`
  - Run the three todo governance checks together.
