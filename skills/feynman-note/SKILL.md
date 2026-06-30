---
name: feynman-note
description: Use when user asks to create or record notes, says note/记录/记一下, mentions Feynman method, asks for 普通记录 or 费曼记录, or wants technical discussion preserved for review without losing the user's original wording.
---

# Feynman Note Skill

Create notes in the shared notes repo. Choose **普通记录** or **费曼记录** first, then apply only that mode's structure and constraints.

## Mode Selection

Default to **普通记录** when the user says:

- `记录到 note`
- `记一下`
- `记录下来`
- `保存一下`
- `记录知识点`

Use **费曼记录** only when the user explicitly says:

- `费曼笔记`
- `费曼记录`
- `整理成费曼`
- `复习卡片`
- `自测问题`
- `知识缺口`
- `闭卷复述`

If the user asks for `总结`、`整理`、`提炼`, summarize first only if requested, then record the summarized result. Do not silently convert 普通记录 into 费曼记录.

## Shared Rules

- Notes repo: `/home/azhi/workspace/projects/notes`.
- Use today's date for `created` and `reviewed`.
- If content to record is missing or ambiguous, ask `要记录的正文是哪一段？`.
- If content exists in the immediately preceding discussion, do not ask; infer a short title and tags.
- Do not invent technical details from general knowledge.
- After writing, validate frontmatter fields relevant to the selected mode.
- Commit and push the notes repo after creating the note unless the user asks not to.

## 普通记录

普通记录的目标是“忠实保存可直接阅读的内容”，不是学习卡片。

### Directory

Save to:

```bash
/home/azhi/workspace/projects/notes/records/<kebab-case-filename>.md
```

Create `records/` if it does not exist.

### Template

Prefer copying the template first:

```bash
cp /home/azhi/workspace/projects/notes/.templates/record-note-template.md \
   /home/azhi/workspace/projects/notes/records/<kebab-case-filename>.md
```

### Standard Framework

```markdown
---
title: "<topic>"
tags: [tag1, tag2]
created: YYYY-MM-DD
type: record
source: chat
related: []
---

# <topic>

<原文或用户指定内容>
```

### Rules

- Preserve the source wording as directly as possible.
- Do not summarize, reorganize, polish, expand, add examples, add conclusions, or add self-test questions.
- Only fix obvious Markdown breakage, such as broken list indentation or missing code fences.
- If the source is the assistant's previous answer, record that answer's content, not a new answer.
- Do not set `next_review` or `confidence`; those are 费曼记录特性.

## 费曼记录

费曼记录的目标是“帮助复习和自测”，可以结构化整理，但仍不能编造未提供的内容。

### Directory

Save to:

```bash
/home/azhi/workspace/projects/notes/feynman/<kebab-case-filename>.md
```

### Template

MUST copy the template first:

```bash
cp /home/azhi/workspace/projects/notes/.templates/feynman-note-template.md \
   /home/azhi/workspace/projects/notes/feynman/<kebab-case-filename>.md
```

Then fill:

```yaml
---
title: "<topic>"
tags: [tag1, tag2]
created: YYYY-MM-DD
reviewed: YYYY-MM-DD
next_review: YYYY-MM-DD   # tomorrow
confidence: 2/5
type: permanent
related: []
---
```

### Standard Framework

Use the template sections as intended:

- `# 核心问题`: one question the note answers.
- `# 我的解释（闭卷复述）`: user's explanation or explicitly requested summary.
- `# 知识缺口`: unclear points, only from user-provided content or explicit analysis request.
- `# 关键细节`: `什么`、`为什么`、`何时用`、`验证方法`.
- `# 反向问题（自测）`: self-test questions.
- `# 关联`: related notes.
- `# 一句话总结`: concise recall sentence.

### Iron Rules

- `next_review` MUST be tomorrow's date.
- `confidence` MUST start at `2/5`.
- Do not choose a longer interval.
- Do not set confidence from perceived difficulty.
- Do not add self-test questions or knowledge gaps unless the user asked for 费曼记录 or asked to summarize/extract them.

## Validation

For 普通记录:

```bash
head -20 records/<filename>.md | grep -E "^(title|tags|created|type|source|related):"
```

Required fields: `title`, `tags`, `created`, `type: record`, `source`, `related`.

For 费曼记录:

```bash
head -20 feynman/<filename>.md | grep -E "^(title|tags|created|reviewed|next_review|confidence|type):"
```

Required fields: `title`, `tags`, `created`, `reviewed`, `next_review`, `confidence`, `type`.

## Git Workflow

```bash
cd /home/azhi/workspace/projects/notes
git add records/<filename>.md  # 普通记录
git add feynman/<filename>.md  # 费曼记录
git commit -m "docs(note): 记录 <topic>"
git push
```

Use exact paths. Do not stage unrelated notes.

## Common Mistakes

| Mistake | Why Wrong | Correct Action |
| --- | --- | --- |
| Putting 普通记录 under `feynman/` | It will enter Feynman review reminders | Save to `records/` |
| Adding `next_review` to 普通记录 | Review scheduling is a Feynman feature | Omit review fields |
| Rewriting a record request | User asked to save readable content | Preserve wording |
| Creating self-test questions for 普通记录 | Adds unrequested Feynman traits | Only add in 费曼记录 |
| Treating all notes as Feynman notes | Mixes two workflows | Select mode first |

## Red Flags

Stop and switch to 普通记录 if you think:

- "They said record, so I should make it more complete."
- "All notes should use the Feynman template."
- "A record needs knowledge gaps and reverse questions."
- "I can put it in `feynman/` and leave sections blank."

Stop and ask if you cannot identify what content should be recorded.
