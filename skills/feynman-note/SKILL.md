---
name: feynman-note
description: Use when user asks to create/record learning notes, mentions Feynman method, or after technical discussions that should be preserved for spaced repetition review
---

# Feynman Note Skill

Create structured learning notes with automated review scheduling.

## When to Use

- User says "创建费曼笔记" / "create Feynman note" / "记录知识点"
- User mentions Feynman method / 费曼学习法
- After technical discussion user wants to preserve for review

## Iron Rules - No Exceptions

### File Location
**MUST** save to `feynman/<filename>.md` directory.

**Never:**
- Save to repository root
- Create subdirectories under feynman/
- Use different directory names

### Review Schedule
**MUST** set `next_review: <tomorrow's date>`.

**Never:**
- Calculate "reasonable" intervals (7 days, 2 weeks, etc.)
- Use spaced repetition formulas
- Adjust based on topic difficulty

**Why:** Review scheduling is user's decision after first review. Always start with tomorrow.

### Confidence Level
**MUST** set `confidence: 2/5`.

**Never:**
- Evaluate topic difficulty yourself
- Adjust based on user's background
- Use different starting values

**Why:** Confidence is user's self-assessment after review, not AI's prediction.

### Content Filling
**Ask user** for explanation content when not provided in discussion.

**Never:**
- Generate complete explanations yourself
- "Help" by writing framework content
- Fill in technical details from your knowledge

**Why:** Feynman method requires user to articulate understanding in their own words.

## Workflow

### 1. Gather Information

Ask user (single question):
```
主题：<topic>
标签：<suggest tags based on topic>
核心问题：<suggest 1-2 core questions>

这样可以吗？
```

**If user provided explanation in discussion:** Use it directly.
**If not provided:** Ask "我的解释部分是现在写，还是留空等你复习时填写？"

### 2. Create Note

**MUST use template file:**
```bash
cp /home/azhi/workspace/projects/notes/.templates/feynman-note-template.md \
   /home/azhi/workspace/projects/notes/feynman/<kebab-case-filename>.md
```

**Never:**
- Create note from scratch
- Use your own format
- Skip the template

**Then fill these fields (no negotiation):**
```yaml
---
title: "<topic>"
tags: [tag1, tag2, tag3]
created: 2026-06-19        # Today
reviewed: 2026-06-19       # Today
next_review: 2026-06-20    # Tomorrow (always)
confidence: 2/5            # Default (always)
type: permanent            # Default
related: []                # Empty initially
---
```

**File location:**
```bash
/home/azhi/workspace/projects/notes/feynman/<kebab-case-filename>.md
```

### 3. Validate Format

Run validation:
```bash
# Check YAML
head -20 feynman/<filename>.md | grep -E "^(title|tags|created|reviewed|next_review|confidence|type):"

# All 7 fields must exist
# Date format: YYYY-MM-DD
# Tags format: [tag1, tag2]
# Confidence format: X/5
```

If validation fails, fix automatically before git operations.

### 4. Git Workflow

```bash
cd /home/azhi/workspace/projects/notes
git add feynman/<filename>.md
git commit -m "Add: <topic> 笔记

- 费曼学习法整理
- 记录知识缺口和反向问题"

git push
```

### 5. Confirm Completion

Report:
```
✅ 笔记已创建：feynman/<filename>.md
✅ 已提交并推送到 GitHub
📅 明天（<next_review date>）会收到复习提醒
📝 当前信心水平：2/5（首次记录）
```

Remind user:
- Tomorrow: GitHub Actions will create review reminder Issue
- After review: update `next_review` and `confidence` fields
- Fill knowledge gaps as you research them

## Common Mistakes - STOP

| Mistake | Why Wrong | Correct Action |
|---------|-----------|----------------|
| Save to root directory | Not in feynman/ folder | Always use feynman/ |
| Set next_review to 7 days | "Reasonable interval" | Always tomorrow |
| Set confidence to 4/5 | "Simple topic" | Always 2/5 |
| Generate full explanation | "Help the user" | Ask if they want to write it |
| Skip validation | "Looks correct" | Always run validation script |
| Create own format | "Simpler than template" | Always copy template file first |
| Missing YAML frontmatter | "Markdown is fine" | Must have `---` delimited YAML |

## Red Flags - You're Doing It Wrong

If you think any of these, STOP:
- "7 days is more reasonable for this topic"
- "They seem confident, let me set 4/5"
- "I'll write a framework to help them"
- "Root directory is fine for now"
- "This is a simple note, no need to validate"
- "The template is overkill, I'll create a simpler format"
- "I can skip YAML frontmatter for this quick note"

**All of these mean:** Re-read Iron Rules section above.
