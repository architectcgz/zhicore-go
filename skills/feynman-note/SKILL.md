---
name: feynman-note
description: Use when creating, validating, or managing Feynman learning notes with proper format, git workflow, and review scheduling.
---

# Feynman Note Skill

Create and manage structured learning notes using the Feynman technique with automated review reminders.

## When to Use

- User asks to create a new learning note / 费曼笔记
- User asks to record knowledge / 记录知识点  
- User mentions Feynman method / 费曼学习法
- After a technical discussion that should be preserved for review

## Notes Repository Location

Default: `/home/azhi/workspace/projects/notes`

## Workflow

### 1. Gather Information
Ask user: topic, tags, core question

### 2. Create Note
- Auto-fill: created (today), reviewed (today), next_review (tomorrow), confidence (2/5)
- Use template at `/home/azhi/workspace/projects/notes/.templates/feynman-note-template.md`
- File naming: lowercase kebab-case, save to `feynman/<filename>.md`

### 3. Validate Format
Check YAML frontmatter, required fields, date format

### 4. Git Workflow
```bash
cd /home/azhi/workspace/projects/notes
git add feynman/<filename>.md
git commit -m "Add: <topic> 笔记"
git push
```

### 5. Confirm Completion
Report: file created, committed, pushed, next review date
